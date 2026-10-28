package dynamicpathdetectortests

import (
	"testing"

	dp "github.com/kubescape/storage/pkg/registry/file/dynamicpathdetector"
	"github.com/kubescape/storage/pkg/registry/file/dynamicpathdetector/matchercorpus"
)

func TestMatcherCorpus(t *testing.T) {
	corpus, err := matchercorpus.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(corpus.Paths) == 0 || len(corpus.Args) == 0 {
		t.Fatalf("corpus %s is empty: %d path cases, %d arg cases", matchercorpus.Version(), len(corpus.Paths), len(corpus.Args))
	}
	t.Logf("corpus %s: %d path cases, %d arg cases", matchercorpus.Version(), len(corpus.Paths), len(corpus.Args))
	for i, c := range corpus.Paths {
		name := c.Name
		if name == "" {
			name = c.Pattern + " ~ " + c.Path
		}
		t.Run("path/"+name, func(t *testing.T) {
			if got := dp.CompareDynamic(c.Pattern, c.Path); got != c.Want {
				t.Fatalf("paths[%d]: CompareDynamic(%q, %q) = %v, want %v", i, c.Pattern, c.Path, got, c.Want)
			}
		})
	}
	for i, c := range corpus.Args {
		name := c.Name
		if name == "" {
			name = "case"
		}
		t.Run("args/"+name, func(t *testing.T) {
			if got := dp.CompareExecArgs(c.Profile, c.Runtime); got != c.Want {
				t.Fatalf("args[%d]: CompareExecArgs(%q, %q) = %v, want %v", i, c.Profile, c.Runtime, got, c.Want)
			}
		})
	}
}
