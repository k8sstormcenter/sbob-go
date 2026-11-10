package collapse

import (
	"maps"
	"slices"
	"strings"

	"github.com/k8sstormcenter/sbob-go/match"
)

func AnalyzeOpens(opens []Open, analyzer *PathAnalyzer, pinned func(string) bool) ([]Open, error) {
	if opens == nil {
		return nil, nil
	}

	dynamicOpens := make(map[string]Open)
	for _, open := range opens {
		_, _ = AnalyzeOpen(open.Path, analyzer)
	}

	for i := range opens {
		// sbomSet files have to be always present in the dynamicOpens
		if pinned != nil && pinned(opens[i].Path) {
			dynamicOpens[opens[i].Path] = opens[i]
			continue
		}

		result, err := AnalyzeOpen(opens[i].Path, analyzer)
		if err != nil {
			continue
		}

		if result != opens[i].Path {
			if existing, ok := dynamicOpens[result]; ok {
				existing.Flags = newStringSet(slices.Concat(existing.Flags, opens[i].Flags)...).sorted()
				dynamicOpens[result] = existing
			} else {
				dynamicOpen := Open{Path: result, Flags: opens[i].Flags}
				dynamicOpens[result] = dynamicOpen
			}
		} else {
			dynamicOpens[opens[i].Path] = opens[i]
		}
	}

	result := slices.SortedFunc(maps.Values(dynamicOpens), func(a, b Open) int {
		return strings.Compare(a.Path, b.Path)
	})

	return ConsolidateOpens(result, pinned), nil
}

// consolidateOpens drops any literal Open whose path is already covered
// by a wildcard / dynamic-identifier sibling in the same result set, and
// merges the dropped entry's Flags into that sibling. This is a
// post-trie cleanup pass: AnalyzeOpens may emit both a collapsed pattern
// (e.g. /etc/⋯) AND the original literals (/etc/passwd) when only some
// children at that node hit threshold. Without this pass, downstream
// matchers see both forms and the literal acts as redundant noise.
//
// Two invariants:
//
//  1. Patterns are always preserved — if a path contains either
//     match.WildcardIdentifier or match.DynamicIdentifier it counts as a pattern
//     and is never absorbed.
//  2. SBOM-listed paths are always preserved — they are part of the
//     image's manifest and must remain identifiable on their own,
//     even if a wildcard pattern would otherwise cover them.
//
// Subsumption check uses the same CompareDynamic the runtime matcher
// (CEL `ap.was_opened`) uses, so both sides agree on what "covered"
// means at every depth.
func ConsolidateOpens(opens []Open, pinned func(string) bool) []Open {
	if len(opens) <= 1 {
		return opens
	}

	isPattern := make([]bool, len(opens))
	// Sorted slice of pattern indices, longest-path-first. Order matters:
	// when two patterns both cover the same literal, the more specific
	// (longer) one wins so folded Flags land deterministically. With ties,
	// the input's existing sort (alphabetical by Path) breaks them.
	patternOrder := make([]int, 0, len(opens))
	for i, o := range opens {
		if strings.Contains(o.Path, match.WildcardIdentifier) || strings.Contains(o.Path, match.DynamicIdentifier) {
			isPattern[i] = true
			patternOrder = append(patternOrder, i)
		}
	}
	if len(patternOrder) == 0 {
		return opens
	}
	slices.SortFunc(patternOrder, func(a, b int) int {
		if d := len(opens[b].Path) - len(opens[a].Path); d != 0 {
			return d
		}
		return strings.Compare(opens[a].Path, opens[b].Path)
	})

	rank := make(map[int]int, len(patternOrder))
	for r, pi := range patternOrder {
		rank[pi] = r
	}
	index := newOpenPatternIndex(opens, patternOrder)

	keep := make([]bool, len(opens))
	for i := range opens {
		keep[i] = true
	}

	for i, o := range opens {
		if isPattern[i] {
			continue
		}
		if pinned != nil && pinned(o.Path) {
			continue
		}
		best := -1
		for _, pi := range index.coverers(o.Path) {
			if best < 0 || rank[pi] < rank[best] {
				best = pi
			}
		}
		if best >= 0 {
			opens[best].Flags = newStringSet(slices.Concat(opens[best].Flags, o.Flags)...).sorted()
			keep[i] = false
		}
	}

	for _, narrow := range patternOrder {
		if !keep[narrow] {
			continue
		}
		for _, broad := range patternOrder {
			if broad == narrow || !keep[broad] || rank[broad] <= rank[narrow] {
				continue
			}
			if patternCovers(opens[broad].Path, opens[narrow].Path) {
				opens[broad].Flags = newStringSet(slices.Concat(opens[broad].Flags, opens[narrow].Flags)...).sorted()
				keep[narrow] = false
				break
			}
		}
	}

	out := make([]Open, 0, len(opens))
	for i, o := range opens {
		if keep[i] {
			out = append(out, o)
		}
	}
	return out
}

func AnalyzeOpen(path string, analyzer *PathAnalyzer) (string, error) {
	return analyzer.AnalyzePath(path, "opens")
}

func patternCovers(broad, narrow string) bool {
	if broad == narrow {
		return true
	}
	return segmentsCover(match.SplitPath(broad), match.SplitPath(narrow))
}

func segmentsCover(broad, narrow []string) bool {
	if len(broad) == 0 {
		return len(narrow) == 0
	}
	switch broad[0] {
	case match.WildcardIdentifier:
		if len(broad) == 1 {
			return len(narrow) >= 1
		}
		for k := 0; k <= len(narrow); k++ {
			if segmentsCover(broad[1:], narrow[k:]) {
				return true
			}
		}
		return false
	case match.DynamicIdentifier:
		if len(narrow) == 0 || narrow[0] == match.WildcardIdentifier {
			return false
		}
		return segmentsCover(broad[1:], narrow[1:])
	default:
		if len(narrow) == 0 || narrow[0] != broad[0] {
			return false
		}
		return segmentsCover(broad[1:], narrow[1:])
	}
}
