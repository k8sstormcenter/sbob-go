package dynamicpathdetector

import (
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type contractLivePatternExport struct {
	ContainerProfiles int `json:"containerprofiles"`
	Data              []struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
		ManagedBy string `json:"managed_by"`
		Execs     []struct {
			Path string   `json:"path"`
			Args []string `json:"args"`
		} `json:"execs"`
		Opens []string `json:"opens"`
	} `json:"data"`
}

func contractStarSegments(p string) int {
	n := 0
	for _, s := range strings.Split(p, "/") {
		if s == WildcardIdentifier {
			n++
		}
	}
	return n
}

func contractInstantiatePath(p string) string {
	segs := strings.Split(p, "/")
	for i, s := range segs {
		if s == DynamicIdentifier || s == WildcardIdentifier {
			segs[i] = "x"
		}
	}
	return strings.Join(segs, "/")
}

func contractFusedSentinel(tok string) bool {
	if tok == DynamicIdentifier || tok == ExecArgsWildcard {
		return false
	}
	if !strings.Contains(tok, DynamicIdentifier) {
		return false
	}
	if !strings.Contains(tok, "/") {
		return true
	}
	for _, s := range strings.Split(tok, "/") {
		if s == DynamicIdentifier || s == ExecArgsWildcard {
			continue
		}
		if strings.Contains(s, DynamicIdentifier) {
			return true
		}
	}
	return false
}

func contractInstantiateArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		switch {
		case a == ExecArgsWildcard:
			out = append(out, "x")
		case a == DynamicIdentifier:
			out = append(out, "x")
		case strings.Contains(a, DynamicIdentifier):
			segs := strings.Split(a, "/")
			for i, s := range segs {
				if s == DynamicIdentifier || s == ExecArgsWildcard {
					segs[i] = "x"
				}
			}
			out = append(out, strings.Join(segs, "/"))
		default:
			out = append(out, a)
		}
	}
	return out
}

func TestContract_LivePatternsObeyTheContract(t *testing.T) {
	path := os.Getenv("LIVE_PATTERNS")
	if path == "" {
		t.Skip("set LIVE_PATTERNS to a cluster export of ContainerProfile match patterns")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var ex contractLivePatternExport
	if err := json.Unmarshal(b, &ex); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	opens, execs, multiStar, deadOpen, fused, deadArgs := 0, 0, 0, 0, 0, 0
	byProfile := map[string]int{}
	var examples []string
	note := func(kind, ns, name, detail string) {
		byProfile[ns+"/"+name]++
		if len(examples) < 12 {
			examples = append(examples, kind+" "+ns+"/"+name+": "+detail)
		}
	}

	for _, cp := range ex.Data {
		for _, p := range cp.Opens {
			opens++
			if contractStarSegments(p) > 1 {
				multiStar++
				note("multi-star open", cp.Namespace, cp.Name, strconv.Itoa(contractStarSegments(p))+" star segments")
			}
			if strings.ContainsAny(p, DynamicIdentifier+WildcardIdentifier) && !CompareDynamic(p, contractInstantiatePath(p)) {
				deadOpen++
				note("dead open", cp.Namespace, cp.Name, "pattern does not match its own instantiation")
			}
		}
		for _, e := range cp.Execs {
			execs++
			if len(e.Args) == 0 {
				continue
			}
			for _, a := range e.Args {
				if contractFusedSentinel(a) {
					fused++
					note("fused sentinel arg", cp.Namespace, cp.Name, "a sentinel inside a token that is not slash-segmented")
					break
				}
			}
			if !CompareExecArgs(e.Args, contractInstantiateArgs(e.Args)) {
				deadArgs++
				note("dead argv", cp.Namespace, cp.Name, "vector does not match its own instantiation")
			}
		}
	}

	profiles := make([]string, 0, len(byProfile))
	for k, v := range byProfile {
		profiles = append(profiles, k+" ("+strconv.Itoa(v)+")")
	}
	sort.Strings(profiles)
	t.Logf("profiles %d, opens %d, exec entries %d", len(ex.Data), opens, execs)
	t.Logf("multi-star opens %d, dead opens %d, fused-sentinel exec entries %d, dead argv vectors %d", multiStar, deadOpen, fused, deadArgs)
	for _, e := range examples {
		t.Logf("  %s", e)
	}
	if len(profiles) > 0 {
		t.Logf("affected profiles: %s", strings.Join(profiles, ", "))
	}
	if multiStar+deadOpen+fused+deadArgs > 0 {
		t.Fatalf("live patterns break the contract: %d multi-star opens, %d dead opens, %d fused-sentinel exec entries, %d dead argv vectors", multiStar, deadOpen, fused, deadArgs)
	}
}
