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
		Opens    []string `json:"opens"`
		Subtract struct {
			Execs []struct {
				Path string   `json:"path"`
				Args []string `json:"args"`
			} `json:"execs"`
			Opens []string `json:"opens"`
		} `json:"subtract"`
	} `json:"data"`
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

	opens, execs, deadOpen, deadArgs := 0, 0, 0, 0
	denies, deadDeny := 0, 0
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
			if UnmatchablePath(p) {
				deadOpen++
				note("dead open", cp.Namespace, cp.Name, "pattern does not match its own instantiation")
			}
		}
		for _, e := range cp.Execs {
			execs++
			if len(e.Args) == 0 {
				continue
			}
			if UnmatchableExec(e.Args) {
				deadArgs++
				note("unmatchable exec entry", cp.Namespace, cp.Name, "the vector cannot match any real argv, so the entry grants nothing")
			}
		}
		for _, p := range cp.Subtract.Opens {
			denies++
			if UnmatchablePath(p) {
				deadDeny++
				note("dead deny open", cp.Namespace, cp.Name, "a deny that cannot match is a deny that does not apply")
			}
		}
		for _, e := range cp.Subtract.Execs {
			denies++
			dead := false
			for _, a := range e.Args {
				if UnmatchableArg(a) {
					dead = true
				}
			}
			if UnmatchableExec(e.Args) {
				dead = true
			}
			if dead {
				deadDeny++
				note("dead deny exec", cp.Namespace, cp.Name, "a deny vector that cannot match is a deny that does not apply")
			}
		}
	}

	profiles := make([]string, 0, len(byProfile))
	for k, v := range byProfile {
		profiles = append(profiles, k+" ("+strconv.Itoa(v)+")")
	}
	sort.Strings(profiles)
	t.Logf("profiles %d, opens %d, exec entries %d", len(ex.Data), opens, execs)
	t.Logf("dead opens %d, unmatchable exec entries %d", deadOpen, deadArgs)
	t.Logf("deny entries %d, dead denies %d", denies, deadDeny)
	for _, e := range examples {
		t.Logf("  %s", e)
	}
	if len(profiles) > 0 {
		t.Logf("affected profiles: %s", strings.Join(profiles, ", "))
	}
	if deadOpen+deadArgs+deadDeny > 0 {
		t.Fatalf("live patterns break the contract: %d dead opens, %d unmatchable exec entries, %d dead denies", deadOpen, deadArgs, deadDeny)
	}
}
