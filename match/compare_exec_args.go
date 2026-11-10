package match

import (
	"slices"
	"strings"
)

// MatchExecArgs reports whether runtimeArgs satisfies a profile entry's argv
// contract. argsRequired carries the entry's ExecCalls.ArgsRequired flag:
//
//	false → no constraint; matches any runtimeArgs.
//	true  → strict anchored match against profileArgs (empty profileArgs
//	        matches only an empty runtimeArgs).
//
// The flag exists because v1beta1.ExecCalls.Args is `json:",omitempty"`: an
// explicit `args: []` round-trips back as nil, so the stored vector alone
// cannot distinguish "no constraint" from "must have no args".
func MatchExecArgs(profileArgs []string, argsRequired bool, runtimeArgs []string) bool {
	if !argsRequired {
		return true
	}
	return matchExecArgsStrict(profileArgs, runtimeArgs)
}

// CompareExecArgs reports whether runtimeArgs matches profileArgs anchored at
// both ends by matchExecArgsStrict: an empty profileArgs matches only an empty
// runtimeArgs, so a path-only exec entry admits exactly an argument-less exec.
//
// Use MatchExecArgs with argsRequired=false to express "no constraint".
func CompareExecArgs(profileArgs, runtimeArgs []string) bool {
	return matchExecArgsStrict(profileArgs, runtimeArgs)
}

// matchExecArgsStrict matches profileArgs against runtimeArgs position by
// position, anchored at both ends. An empty profileArgs matches only an empty
// runtimeArgs. Tokens are matched by argMatches, except a standalone
// ExecArgsWildcard ("⋯⋯"), which absorbs zero or more consecutive runtime
// args.
//
// Index-based backtracking memoised on (profileIndex, runtimeIndex): without
// the memo, patterns like [*, *, …, x] against a long literal vector backtrack
// exponentially; the memo bounds it at O(len(profile)*len(runtime)).
func matchExecArgsStrict(profileArgs, runtimeArgs []string) bool {
	if len(profileArgs) == 0 {
		return len(runtimeArgs) == 0
	}

	if !slices.Contains(profileArgs, ExecArgsWildcard) {
		if len(profileArgs) != len(runtimeArgs) {
			return false
		}
		for i := range profileArgs {
			if !ArgMatches(profileArgs[i], runtimeArgs[i]) {
				return false
			}
		}
		return true
	}

	np, nr := len(profileArgs), len(runtimeArgs)
	memo := make([]uint8, (np+1)*(nr+1))

	var match func(pi, ri int) bool
	match = func(pi, ri int) bool {
		idx := pi*(nr+1) + ri
		if m := memo[idx]; m != 0 {
			return m == 2
		}
		store := func(v bool) bool {
			if v {
				memo[idx] = 2
			} else {
				memo[idx] = 1
			}
			return v
		}

		if pi == np {
			return store(ri == nr)
		}

		head := profileArgs[pi]

		if head == ExecArgsWildcard {
			// Absorb 0..remaining runtime args; first successful split wins.
			for k := ri; k <= nr; k++ {
				if match(pi+1, k) {
					return store(true)
				}
			}
			return store(false)
		}

		if ri == nr {
			return store(false)
		}

		if ArgMatches(head, runtimeArgs[ri]) {
			return store(match(pi+1, ri+1))
		}

		return store(false)
	}

	return match(0, 0)
}

// argMatches matches one profile token against one runtime arg:
//
//	bare "⋯"              — any single arg.
//	token containing "⋯"  — a path-shaped arg matched segment-wise, where "⋯"
//	                        matches exactly one segment and every other segment
//	                        (including any "*") is literal.
//	anything else         — literal equality.
//
// Exec args use dedicated, collision-free wildcards: "⋯" (one arg / one
// segment) and "⋯⋯" (zero-or-more whole args, consumed positionally by
// matchExecArgsStrict). A "*" is NEVER a wildcard here — a process is routinely
// invoked with a literal "*" (an unexpanded shell glob), so a "*" is matched as
// data. Treating it literally is what prevents the R0040 over-broadening (a
// recorded "/plugins/*" arg must not match "/plugins/evil.so"), and because the
// wildcards are sentinels that cannot occur in real argv, no escaping is needed.
func ArgMatches(profileArg, runtimeArg string) bool {
	if profileArg == DynamicIdentifier {
		return true
	}
	if strings.Contains(profileArg, DynamicIdentifier) {
		return matchExecArgSegments(profileArg, runtimeArg)
	}
	return profileArg == runtimeArg
}

// matchExecArgSegments matches a "/"-segmented profile token against a runtime
// arg. "⋯" matches exactly one segment; every other segment — "*" included — is
// matched literally. "⋯" never spans more than one segment. A segment that is
// exactly "⋯⋯" spans one or more segments; at most one per token, anywhere in
// it, with the literal segments around it anchoring the match. A standalone
// "⋯⋯" argument keeps its positional meaning (zero or more whole args).
func matchExecArgSegments(profileArg, runtimeArg string) bool {
	p := strings.Split(profileArg, "/")
	r := strings.Split(runtimeArg, "/")
	span := -1
	for i := range p {
		if p[i] == ExecArgsWildcard {
			if span >= 0 {
				return false
			}
			span = i
		}
	}
	if span < 0 {
		if len(p) != len(r) {
			return false
		}
		return segmentsMatch(p, r)
	}
	head, tail := p[:span], p[span+1:]
	if len(r) < len(head)+len(tail)+1 {
		return false
	}
	return segmentsMatch(head, r[:len(head)]) && segmentsMatch(tail, r[len(r)-len(tail):])
}

func segmentsMatch(p, r []string) bool {
	for i := range p {
		if p[i] != DynamicIdentifier && p[i] != r[i] {
			return false
		}
	}
	return true
}
