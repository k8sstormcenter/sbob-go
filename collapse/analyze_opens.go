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
	keys, groups := groupByProcesses(opens)
	if len(keys) == 1 {
		return ConsolidateOpens(analyzeOpenGroup(opens, analyzer, pinned), pinned), nil
	}
	byPath := make(map[string]Open)
	for _, k := range keys {
		for _, o := range analyzeOpenGroup(groups[k], NewPathAnalyzerWithConfigs(analyzer.threshold, analyzer.configs), pinned) {
			if existing, ok := byPath[o.Path]; ok {
				byPath[o.Path] = mergeOpen(existing, o)
			} else {
				byPath[o.Path] = o
			}
		}
	}
	result := slices.SortedFunc(maps.Values(byPath), func(a, b Open) int {
		return strings.Compare(a.Path, b.Path)
	})
	return ConsolidateOpens(result, pinned), nil
}

func groupByProcesses(opens []Open) ([]string, map[string][]Open) {
	groups := make(map[string][]Open)
	var keys []string
	for _, o := range opens {
		k := match.ProcessSetKey(o.Processes)
		if _, ok := groups[k]; !ok {
			keys = append(keys, k)
		}
		groups[k] = append(groups[k], o)
	}
	slices.Sort(keys)
	return keys, groups
}

func analyzeOpenGroup(opens []Open, analyzer *PathAnalyzer, pinned func(string) bool) []Open {
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
				dynamicOpens[result] = mergeOpen(existing, opens[i])
			} else {
				dynamicOpens[result] = Open{Path: result, Flags: opens[i].Flags, Processes: opens[i].Processes}
			}
		} else {
			dynamicOpens[opens[i].Path] = opens[i]
		}
	}

	return slices.SortedFunc(maps.Values(dynamicOpens), func(a, b Open) int {
		return strings.Compare(a.Path, b.Path)
	})
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
			if !processesCover(opens[pi].Processes, o.Processes) {
				continue
			}
			if best < 0 || rank[pi] < rank[best] {
				best = pi
			}
		}
		if best >= 0 {
			opens[best] = mergeOpen(opens[best], o)
			keep[i] = false
		}
	}

	patternSegs := make([][]string, len(patternOrder))
	byPrefix := make(map[segPrefix][]int, len(patternOrder))
	for r, pi := range patternOrder {
		segs := match.SplitPath(opens[pi].Path)
		patternSegs[r] = segs
		k := literalPrefixLen(segs)
		key := segPrefix{k, strings.Join(segs[:k], "/")}
		byPrefix[key] = append(byPrefix[key], r)
	}
	var candidates []int
	for r, narrow := range patternOrder {
		if !keep[narrow] {
			continue
		}
		segs := patternSegs[r]
		candidates = candidates[:0]
		for j := 0; j <= literalPrefixLen(segs); j++ {
			bucket := byPrefix[segPrefix{j, strings.Join(segs[:j], "/")}]
			from, _ := slices.BinarySearch(bucket, r+1)
			candidates = append(candidates, bucket[from:]...)
		}
		slices.Sort(candidates)
		for _, b := range candidates {
			broad := patternOrder[b]
			if !keep[broad] {
				continue
			}
			if !processesCover(opens[broad].Processes, opens[narrow].Processes) {
				continue
			}
			if opens[broad].Path == opens[narrow].Path || segmentsCover(patternSegs[b], segs) {
				opens[broad] = mergeOpen(opens[broad], opens[narrow])
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

func mergeOpen(into, from Open) Open {
	into.Flags = newStringSet(slices.Concat(into.Flags, from.Flags)...).sorted()
	into.Processes = match.UnionProcesses(into.Processes, from.Processes)
	return into
}

func processesCover(broad, narrow []string) bool {
	if len(broad) == 0 {
		return true
	}
	if len(narrow) == 0 {
		return false
	}
	for _, n := range narrow {
		covered := false
		for _, b := range broad {
			if b == n || segmentsCover(match.SplitPath(b), match.SplitPath(n)) {
				covered = true
				break
			}
		}
		if !covered {
			return false
		}
	}
	return true
}

func AnalyzeOpen(path string, analyzer *PathAnalyzer) (string, error) {
	return analyzer.AnalyzePath(path, "opens")
}

type segPrefix struct {
	n int
	s string
}

func literalPrefixLen(segs []string) int {
	for i, s := range segs {
		if s == match.WildcardIdentifier || s == match.DynamicIdentifier {
			return i
		}
	}
	return len(segs)
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
