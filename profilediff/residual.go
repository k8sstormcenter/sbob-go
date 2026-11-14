package profilediff

import (
	"github.com/k8sstormcenter/sbob-go/match"
	"github.com/k8sstormcenter/sbob-go/profile"
)

func Residual(shadow, bound *profile.Spec) *profile.Spec {
	out := &profile.Spec{}
	if shadow == nil {
		return out
	}
	if bound == nil {
		bound = &profile.Spec{}
	}

	for i := range shadow.Execs {
		e := &shadow.Execs[i]
		if execCovered(bound.Execs, e) || execDenied(bound.Subtract, e) {
			continue
		}
		out.Execs = append(out.Execs, e.Clone())
	}
	for i := range shadow.Opens {
		o := &shadow.Opens[i]
		if openCovered(bound.Opens, o) || openDenied(bound.Subtract, o) {
			continue
		}
		out.Opens = append(out.Opens, o.Clone())
	}
	for _, s := range shadow.Syscalls {
		if contains(bound.Syscalls, s) || (bound.Subtract != nil && contains(bound.Subtract.Syscalls, s)) {
			continue
		}
		out.Syscalls = append(out.Syscalls, s)
	}
	for _, c := range shadow.Capabilities {
		if contains(bound.Capabilities, c) || (bound.Subtract != nil && contains(bound.Subtract.Capabilities, c)) {
			continue
		}
		out.Capabilities = append(out.Capabilities, c)
	}
	return out
}

func execCovered(bound []profile.ExecCall, e *profile.ExecCall) bool {
	for i := range bound {
		if match.CompareDynamic(bound[i].Path, e.Path) && match.CompareExecArgs(bound[i].Args, e.Args) {
			return true
		}
	}
	return false
}

func openCovered(bound []profile.OpenCall, o *profile.OpenCall) bool {
	for i := range bound {
		if match.CompareDynamic(bound[i].Path, o.Path) {
			return true
		}
	}
	return false
}

func execDenied(sub *profile.Subtract, e *profile.ExecCall) bool {
	if sub == nil {
		return false
	}
	for i := range sub.Execs {
		d := &sub.Execs[i]
		if !match.CompareDynamic(d.Path, e.Path) {
			continue
		}
		if len(d.Args) == 0 || match.CompareExecArgs(d.Args, e.Args) {
			return true
		}
	}
	return false
}

func openDenied(sub *profile.Subtract, o *profile.OpenCall) bool {
	if sub == nil {
		return false
	}
	for i := range sub.Opens {
		if match.CompareDynamic(sub.Opens[i].Path, o.Path) {
			return true
		}
	}
	return false
}

func contains(set []string, v string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}
