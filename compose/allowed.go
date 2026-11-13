package compose

import (
	"sort"

	"github.com/k8sstormcenter/sbob-go/match"
)

func Allowed(p *Projection, surface string, rt match.Entry) (bool, error) {
	if p == nil {
		p = &Projection{}
	}
	var subtract []match.Entry
	if p.Subtract != nil {
		subtract = entries(p.Subtract, surface)
	}
	return match.Allowed(surface, entries(&p.Sets, surface), subtract, rt)
}

func entries(s *Sets, surface string) []match.Entry {
	var out []match.Entry
	switch surface {
	case match.SurfaceExec, match.SurfaceExecPath:
		paths := make([]string, 0, len(s.ExecArgs))
		for path := range s.ExecArgs {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			for _, args := range s.ExecArgs[path] {
				out = append(out, match.Entry{Path: path, Args: args})
			}
		}
	case match.SurfaceOpen:
		for _, path := range fieldEntries(s.Opens) {
			out = append(out, match.Entry{Path: path})
		}
	case match.SurfaceDomain:
		for _, name := range fieldEntries(s.EgressDomains) {
			out = append(out, match.Entry{Name: name})
		}
	}
	return out
}

func fieldEntries(f Field) []string {
	out := make([]string, 0, len(f.Values)+len(f.Patterns))
	for v := range f.Values {
		out = append(out, v)
	}
	sort.Strings(out)
	return append(out, f.Patterns...)
}
