package dynamicpathdetector

import (
	"fmt"
	"path"

	"github.com/kubescape/storage/pkg/registry/file/networkmatch"
)

type Entry struct {
	Path string
	Args []string
	Name string
}

const (
	SurfaceExec     = "exec"
	SurfaceExecPath = "exec_path"
	SurfaceOpen     = "open"
	SurfaceDomain   = "domain"
)

func ExecVectorMatches(entries []Entry, rt Entry) bool {
	for _, e := range entries {
		if CompareDynamic(e.Path, rt.Path) && CompareExecArgs(e.Args, rt.Args) {
			return true
		}
	}
	return false
}

func ExecPathMatches(entries []Entry, rt Entry) bool {
	for _, e := range entries {
		if CompareDynamic(e.Path, rt.Path) {
			return true
		}
	}
	return false
}

func ExecPathDenied(entries []Entry, rt Entry) bool {
	invoked := path.Base(rt.Path)
	if len(rt.Args) > 0 && rt.Args[0] != "" {
		invoked = rt.Args[0]
	}
	for _, e := range entries {
		if !CompareDynamic(e.Path, rt.Path) {
			continue
		}
		if len(e.Args) == 0 {
			return true
		}
		if len(e.Args) == 2 && e.Args[1] == ExecArgsWildcard && (e.Args[0] == "" || SameExecName(e.Args[0], invoked) || CompareDynamic(e.Args[0], invoked)) {
			return true
		}
	}
	return false
}

func OpenPathMatches(entries []Entry, rt Entry) bool {
	for _, e := range entries {
		if CompareDynamic(e.Path, rt.Path) {
			return true
		}
	}
	return false
}

func DomainMatches(entries []Entry, rt Entry) bool {
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name)
	}
	return networkmatch.MatchDNS(names, rt.Name)
}

func Allowed(surface string, allow, subtract []Entry, rt Entry) (bool, error) {
	var match, deny func([]Entry, Entry) bool
	switch surface {
	case SurfaceExec:
		match, deny = ExecVectorMatches, ExecVectorMatches
	case SurfaceExecPath:
		match, deny = ExecPathMatches, ExecPathDenied
	case SurfaceOpen:
		match, deny = OpenPathMatches, OpenPathMatches
	case SurfaceDomain:
		match, deny = DomainMatches, DomainMatches
	default:
		return false, fmt.Errorf("unknown subtract surface %q", surface)
	}
	return match(allow, rt) && !deny(subtract, rt), nil
}
