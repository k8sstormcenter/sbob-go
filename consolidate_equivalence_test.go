package dynamicpathdetector

import (
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"testing"

	mapset "github.com/deckarep/golang-set/v2"
	types "github.com/kubescape/storage/pkg/apis/softwarecomposition"
	"github.com/stretchr/testify/require"
)

func referenceConsolidateExecs(execs []types.ExecCalls) []types.ExecCalls {
	sorted := slices.Clone(execs)
	slices.SortFunc(sorted, compareExecCalls)
	var patterns []int
	anyArgs := make([]bool, len(sorted))
	for i := range sorted {
		anyArgs[i] = slices.Contains(sorted[i].Args, ExecArgsWildcard)
		if hasDynamicToken(sorted[i].Args) {
			patterns = append(patterns, i)
		}
	}
	if len(patterns) == 0 {
		return sorted
	}
	covers := func(j, i int) bool {
		if i == j || anyArgs[i] || (!anyArgs[j] && len(sorted[j].Args) != len(sorted[i].Args)) {
			return false
		}
		return CompareExecArgs(sorted[j].Args, sorted[i].Args)
	}
	covered := make([]bool, len(sorted))
	for i := range sorted {
		for _, j := range patterns {
			if covers(j, i) {
				covered[i] = true
				break
			}
		}
	}
	var kept, keptPatterns []int
	for i := range sorted {
		if !covered[i] {
			kept = append(kept, i)
			if hasDynamicToken(sorted[i].Args) {
				keptPatterns = append(keptPatterns, i)
			}
		}
	}
	merged := map[int]mapset.Set[string]{}
	for i := range sorted {
		if !covered[i] {
			continue
		}
		into := -1
		for _, j := range keptPatterns {
			if covers(j, i) {
				into = j
				break
			}
		}
		if into < 0 {
			kept = append(kept, i)
			if hasDynamicToken(sorted[i].Args) {
				keptPatterns = append(keptPatterns, i)
			}
			continue
		}
		set, ok := merged[into]
		if !ok {
			set = mapset.NewThreadUnsafeSet(sorted[into].Envs...)
			merged[into] = set
		}
		set.Append(sorted[i].Envs...)
	}
	slices.Sort(kept)
	out := make([]types.ExecCalls, 0, len(kept))
	for _, i := range kept {
		e := sorted[i]
		if set, ok := merged[i]; ok {
			e.Envs = mapsetSortedOrNil(set)
		} else {
			e.Envs = normalizedEnvs(e.Envs)
		}
		out = append(out, e)
	}
	return out
}

func referenceConsolidateOpens(opens []types.OpenCalls, sbomSet mapset.Set[string]) []types.OpenCalls {
	if len(opens) <= 1 {
		return opens
	}
	isPattern := make([]bool, len(opens))
	patternOrder := make([]int, 0, len(opens))
	for i, o := range opens {
		if strings.Contains(o.Path, WildcardIdentifier) || strings.Contains(o.Path, DynamicIdentifier) {
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
	keep := make([]bool, len(opens))
	for i := range opens {
		keep[i] = true
	}
	for i, o := range opens {
		if isPattern[i] {
			continue
		}
		if sbomSet != nil && sbomSet.ContainsOne(o.Path) {
			continue
		}
		for _, pi := range patternOrder {
			if CompareDynamic(opens[pi].Path, o.Path) {
				opens[pi].Flags = mapset.Sorted(mapset.NewThreadUnsafeSet(slices.Concat(opens[pi].Flags, o.Flags)...))
				keep[i] = false
				break
			}
		}
	}
	out := make([]types.OpenCalls, 0, len(opens))
	for i, o := range opens {
		if keep[i] {
			out = append(out, o)
		}
	}
	return out
}

var equivalenceTokens = []string{
	"a", "b", "c", "--flag", "x/y", "x/⋯", "⋯/y", "x/y/z", "x/⋯/z", "⋯/⋯", "x/⋯⋯/z", "⋯⋯/z", "x/⋯⋯",
	DynamicIdentifier, ExecArgsWildcard, "*", "x*", "", "/", "a/", "/a", "a//b", "⋯a", "x/⋯y", "x⋯⋯y",
}

func randomExecs(r *rand.Rand, n int) []types.ExecCalls {
	seen := map[string]bool{}
	var out []types.ExecCalls
	for len(out) < n {
		argc := 1 + r.Intn(4)
		args := make([]string, argc)
		for i := range args {
			args[i] = equivalenceTokens[r.Intn(len(equivalenceTokens))]
		}
		key := tokensKey(args)
		if seen[key] {
			continue
		}
		seen[key] = true
		var envs []string
		if r.Intn(2) == 0 {
			envs = []string{fmt.Sprintf("E=%d", r.Intn(3))}
		}
		out = append(out, types.ExecCalls{Path: "/bin/p", Args: args, Envs: envs})
	}
	return out
}

func TestConsolidateExecsMatchesReference(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	for round := 0; round < 400; round++ {
		execs := randomExecs(r, 1+r.Intn(40))
		want := referenceConsolidateExecs(slices.Clone(execs))
		got := consolidateExecs(slices.Clone(execs))
		require.Equal(t, want, got, "round %d input %v", round, execs)
	}
}

var equivalenceSegments = []string{"a", "b", "c", "⋯", "*", "", "a*", "⋯b", "d"}

func randomOpens(r *rand.Rand, n int) []types.OpenCalls {
	seen := map[string]bool{}
	var out []types.OpenCalls
	for len(out) < n {
		depth := 1 + r.Intn(5)
		var b strings.Builder
		if r.Intn(4) != 0 {
			b.WriteByte('/')
		}
		for i := 0; i < depth; i++ {
			if i > 0 {
				b.WriteByte('/')
			}
			b.WriteString(equivalenceSegments[r.Intn(len(equivalenceSegments))])
		}
		if r.Intn(5) == 0 {
			b.WriteByte('/')
		}
		p := b.String()
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, types.OpenCalls{Path: p, Flags: []string{fmt.Sprintf("F%d", r.Intn(3))}})
	}
	slices.SortFunc(out, func(a, b types.OpenCalls) int { return strings.Compare(a.Path, b.Path) })
	return out
}

func cloneOpens(opens []types.OpenCalls) []types.OpenCalls {
	out := make([]types.OpenCalls, len(opens))
	for i, o := range opens {
		out[i] = types.OpenCalls{Path: o.Path, Flags: slices.Clone(o.Flags)}
	}
	return out
}

func TestConsolidateOpensMatchesReference(t *testing.T) {
	r := rand.New(rand.NewSource(11))
	for round := 0; round < 600; round++ {
		opens := randomOpens(r, 1+r.Intn(40))
		var sbom mapset.Set[string]
		if r.Intn(3) == 0 && len(opens) > 0 {
			sbom = mapset.NewThreadUnsafeSet(opens[r.Intn(len(opens))].Path)
		}
		want := referenceConsolidateOpens(cloneOpens(opens), sbom)
		got := consolidateOpens(cloneOpens(opens), pinnedFrom(sbom))
		require.Equal(t, want, got, "round %d input %v", round, opens)
	}
}

func pinnedFrom(set mapset.Set[string]) func(string) bool {
	if set == nil {
		return nil
	}
	return set.ContainsOne
}

func mapsetSortedOrNil(set mapset.Set[string]) []string {
	if set.Cardinality() == 0 {
		return nil
	}
	return mapset.Sorted(set)
}
