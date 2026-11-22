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

	allowRegexps := regexpsOf(bound.Execs)
	var subtractRegexps []*match.ExecRegexp
	if bound.Subtract != nil {
		subtractRegexps = regexpsOf(bound.Subtract.Execs)
	}
	for i := range shadow.Execs {
		e := &shadow.Execs[i]
		if execCovered(bound.Execs, allowRegexps, e) || execDenied(bound.Subtract, subtractRegexps, e) {
			continue
		}
		out.Execs = append(out.Execs, e.Clone())
	}
	for i := range shadow.Opens {
		if r, ok := openResidual(bound, &shadow.Opens[i]); ok {
			out.Opens = append(out.Opens, r)
		}
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

func regexpsOf(execs []profile.ExecCall) []*match.ExecRegexp {
	var out []*match.ExecRegexp
	for i := range execs {
		if execs[i].ArgsRegex == "" {
			continue
		}
		if out == nil {
			out = make([]*match.ExecRegexp, len(execs))
		}
		out[i] = match.ExecRegexpOf(execs[i].ArgsRegex)
	}
	return out
}

func execCovered(bound []profile.ExecCall, regexps []*match.ExecRegexp, e *profile.ExecCall) bool {
	for i := range bound {
		if !match.CompareDynamic(bound[i].Path, e.Path) {
			continue
		}
		if regexps != nil && regexps[i] != nil {
			if regexps[i].MatchArgs(e.Args) {
				return true
			}
			continue
		}
		if match.CompareExecArgs(bound[i].Args, e.Args) {
			return true
		}
	}
	return false
}

func openResidual(bound *profile.Spec, o *profile.OpenCall) (profile.OpenCall, bool) {
	var sub []profile.OpenCall
	if bound.Subtract != nil {
		sub = bound.Subtract.Opens
	}
	if len(o.Processes) == 0 {
		if openAdmits(bound.Opens, o.Path, "") || openAdmits(sub, o.Path, "") {
			return profile.OpenCall{}, false
		}
		return o.Clone(), true
	}
	var left []string
	for _, p := range o.Processes {
		if !openAdmits(bound.Opens, o.Path, p) && !openAdmits(sub, o.Path, p) {
			left = append(left, p)
		}
	}
	if len(left) == 0 {
		return profile.OpenCall{}, false
	}
	r := o.Clone()
	r.Processes = left
	return r, true
}

func openAdmits(entries []profile.OpenCall, path, exe string) bool {
	for i := range entries {
		if match.CompareDynamic(entries[i].Path, path) && match.ProcessAdmitted(entries[i].Processes, exe) {
			return true
		}
	}
	return false
}

func execDenied(sub *profile.Subtract, regexps []*match.ExecRegexp, e *profile.ExecCall) bool {
	if sub == nil {
		return false
	}
	for i := range sub.Execs {
		d := &sub.Execs[i]
		if !match.CompareDynamic(d.Path, e.Path) {
			continue
		}
		if regexps != nil && regexps[i] != nil {
			if !regexps[i].Valid() || regexps[i].MatchArgs(e.Args) {
				return true
			}
			continue
		}
		if len(d.Args) == 0 || match.CompareExecArgs(d.Args, e.Args) {
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
