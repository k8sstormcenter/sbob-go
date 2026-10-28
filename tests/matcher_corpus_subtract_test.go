package dynamicpathdetectortests

import (
	"testing"

	dp "github.com/kubescape/storage/pkg/registry/file/dynamicpathdetector"
	"github.com/kubescape/storage/pkg/registry/file/dynamicpathdetector/matchercorpus"
	"github.com/kubescape/storage/pkg/registry/file/networkmatch"
)

func execEntryMatches(entries []matchercorpus.Entry, rt matchercorpus.Entry) bool {
	for _, e := range entries {
		if dp.CompareDynamic(e.Path, rt.Path) && dp.CompareExecArgs(e.Args, rt.Args) {
			return true
		}
	}
	return false
}

func openEntryMatches(entries []matchercorpus.Entry, rt matchercorpus.Entry) bool {
	for _, e := range entries {
		if dp.CompareDynamic(e.Path, rt.Path) {
			return true
		}
	}
	return false
}

func domainEntryMatches(entries []matchercorpus.Entry, rt matchercorpus.Entry) bool {
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name)
	}
	return networkmatch.MatchDNS(names, rt.Name)
}

func TestMatcherCorpus_Subtract(t *testing.T) {
	cases := matchercorpus.Subtracts()
	if len(cases) == 0 {
		t.Fatal("corpus carries no subtract cases")
	}
	for i, c := range cases {
		name := c.Name
		if name == "" {
			name = "case"
		}
		t.Run(name, func(t *testing.T) {
			var match func([]matchercorpus.Entry, matchercorpus.Entry) bool
			switch c.Surface {
			case "exec":
				match = execEntryMatches
			case "open":
				match = openEntryMatches
			case "domain":
				match = domainEntryMatches
			default:
				t.Fatalf("subtract[%d]: unknown surface %q", i, c.Surface)
			}
			allowed := match(c.Allow, c.Runtime)
			denied := match(c.Subtract, c.Runtime)
			if got := allowed && !denied; got != c.Want {
				t.Fatalf("subtract[%d] %s: allowed=%v denied=%v -> %v, want %v", i, c.Surface, allowed, denied, got, c.Want)
			}
		})
	}
}
