package dynamicpathdetectortests

import (
	"testing"

	dp "github.com/kubescape/storage/pkg/registry/file/dynamicpathdetector"
	"github.com/kubescape/storage/pkg/registry/file/dynamicpathdetector/matchercorpus"
	"github.com/kubescape/storage/pkg/registry/file/networkmatch"
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

func TestMatcherCorpus_ExecNames(t *testing.T) {
	cases := matchercorpus.ExecNames()
	if len(cases) == 0 {
		t.Fatal("corpus carries no exec name cases")
	}
	for i, c := range cases {
		if got := dp.SameExecName(c.Recorded, c.Invoked); got != c.Want {
			t.Errorf("execNames[%d] %s: SameExecName(%q, %q) = %v, want %v", i, c.Name, c.Recorded, c.Invoked, got, c.Want)
		}
	}
}

func TestMatcherCorpus_Sentinels(t *testing.T) {
	cases := matchercorpus.Sentinels()
	if len(cases) == 0 {
		t.Fatal("corpus carries no sentinel cases")
	}
	for i, c := range cases {
		var got bool
		switch c.Surface {
		case "capabilities", "syscalls":
			got = dp.MatchAny(c.Profile, c.Observed)
		case "dnsNames":
			got = networkmatch.MatchDNS(c.Profile, c.Observed)
		default:
			t.Fatalf("sentinels[%d]: unknown surface %q", i, c.Surface)
		}
		if got != c.Want {
			t.Errorf("sentinels[%d] %s: %v vs %q = %v, want %v", i, c.Name, c.Profile, c.Observed, got, c.Want)
		}
	}
}
