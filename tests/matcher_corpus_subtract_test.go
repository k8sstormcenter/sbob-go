package dynamicpathdetectortests

import (
	"testing"

	dp "github.com/kubescape/storage/pkg/registry/file/dynamicpathdetector"
	"github.com/kubescape/storage/pkg/registry/file/dynamicpathdetector/matchercorpus"
)

func toEntry(e matchercorpus.Entry) dp.Entry {
	return dp.Entry{Path: e.Path, Args: e.Args, Name: e.Name}
}

func toEntries(entries []matchercorpus.Entry) []dp.Entry {
	out := make([]dp.Entry, 0, len(entries))
	for _, e := range entries {
		out = append(out, toEntry(e))
	}
	return out
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
			got, err := dp.Allowed(c.Surface, toEntries(c.Allow), toEntries(c.Subtract), toEntry(c.Runtime))
			if err != nil {
				t.Fatalf("subtract[%d]: %v", i, err)
			}
			if got != c.Want {
				t.Fatalf("subtract[%d] %s: allowed %v, want %v", i, c.Surface, got, c.Want)
			}
		})
	}
}
