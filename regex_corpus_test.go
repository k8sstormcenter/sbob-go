package dynamicpathdetector

import (
	"regexp"
	"strings"
	"testing"

	"github.com/kubescape/storage/pkg/registry/file/dynamicpathdetector/matchercorpus"
)

func TestRegexRendering_AgreesWithCompareDynamicOnTheCorpus(t *testing.T) {
	for i, c := range matchercorpus.Paths() {
		re, err := regexp.Compile(PathRegexp(c.Pattern))
		if err != nil {
			t.Fatalf("paths[%d] %q: %v", i, c.Pattern, err)
		}
		want := CompareDynamic(c.Pattern, c.Path)
		if want != c.Want {
			t.Fatalf("paths[%d] %q vs %q: matcher %v, corpus %v", i, c.Pattern, c.Path, want, c.Want)
		}
		if got := re.MatchString(c.Path); got != want {
			t.Errorf("paths[%d] %q vs %q: regex %s gives %v, matcher %v", i, c.Pattern, c.Path, re, got, want)
		}
	}
}

func TestRegexRendering_AgreesWithCompareExecArgsOnTheCorpus(t *testing.T) {
	const sep = "\x00"
	for i, c := range matchercorpus.Args() {
		re, err := regexp.Compile(ArgvRegexp(c.Profile, sep))
		if err != nil {
			t.Fatalf("args[%d] %v: %v", i, c.Profile, err)
		}
		want := CompareExecArgs(c.Profile, c.Runtime)
		if want != c.Want {
			t.Fatalf("args[%d] %v vs %v: matcher %v, corpus %v", i, c.Profile, c.Runtime, want, c.Want)
		}
		subject := strings.Join(c.Runtime, sep)
		if len(c.Runtime) == 0 {
			subject = ""
		}
		if got := re.MatchString(subject); got != want {
			t.Errorf("args[%d] %v vs %v: regex %s gives %v, matcher %v", i, c.Profile, c.Runtime, re, got, want)
		}
	}
}

func TestRegexRendering_Forms(t *testing.T) {
	cases := []struct {
		pattern, path string
		want          bool
	}{
		{"/etc/*", "/etc/passwd", true},
		{"/etc/*", "/etc", false},
		{"/etc/*", "/etc/", false},
		{"/a/*/b", "/a/b", true},
		{"/a/*/b", "/a/x/y/b", true},
		{"/a/*/b", "/a/x/c", false},
		{"/a/⋯/b", "/a/x/b", true},
		{"/a/⋯/b", "/a/b", false},
		{"*", "/", true},
		{"/*", "/", false},
		{"/opt/x.y", "/opt/xzy", false},
	}
	for _, c := range cases {
		re := regexp.MustCompile(PathRegexp(c.pattern))
		if got := re.MatchString(c.path); got != c.want {
			t.Errorf("%q vs %q: regex %s gives %v, want %v", c.pattern, c.path, re, got, c.want)
		}
		if got := CompareDynamic(c.pattern, c.path); got != c.want {
			t.Errorf("%q vs %q: matcher gives %v, want %v", c.pattern, c.path, got, c.want)
		}
	}
}
