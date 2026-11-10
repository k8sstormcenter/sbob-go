package match

import "strings"

func WildcardSegmentCount(path string) int {
	n := 0
	for _, s := range strings.Split(path, "/") {
		if s == WildcardIdentifier {
			n++
		}
	}
	return n
}

func instantiateSegments(s string, starIsWildcard bool) string {
	segs := strings.Split(s, "/")
	for i, seg := range segs {
		if seg == DynamicIdentifier || seg == ExecArgsWildcard || (starIsWildcard && seg == WildcardIdentifier) {
			segs[i] = instantiationSegment
			continue
		}
		seg = strings.ReplaceAll(seg, ExecArgsWildcard, instantiationSegment)
		segs[i] = strings.ReplaceAll(seg, DynamicIdentifier, instantiationSegment)
	}
	return strings.Join(segs, "/")
}

const instantiationSegment = "x"

func UnmatchablePath(path string) bool {
	if path == "" {
		return true
	}
	if !strings.Contains(path, DynamicIdentifier) && !strings.Contains(path, WildcardIdentifier) {
		return false
	}
	return !CompareDynamic(path, instantiateSegments(path, true))
}

func UnmatchableArg(token string) bool {
	if token == DynamicIdentifier || token == ExecArgsWildcard {
		return false
	}
	if !strings.Contains(token, DynamicIdentifier) {
		return false
	}
	return !ArgMatches(token, instantiateSegments(token, false))
}

func InstantiateExecArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		out = append(out, instantiateSegments(a, false))
	}
	return out
}

func UnmatchableExec(args []string) bool {
	if len(args) == 0 {
		return false
	}
	for _, a := range args {
		if UnmatchableArg(a) {
			return true
		}
	}
	return !CompareExecArgs(args, InstantiateExecArgs(args))
}
