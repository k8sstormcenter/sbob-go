package match

import "strings"

// CompareDynamic checks whether `regularPath` is matched by `dynamicPath`.
// The dynamic path may contain DynamicIdentifier (⋯, exactly-one-segment
// wildcard) or WildcardIdentifier (*, zero-or-more-segment mid-path /
// one-or-more-segment trailing wildcard). The node-agent R0002 rule
// (Files Access Anomalies) uses this at every file-open to decide whether
// the access is in-profile.
//
// Anchoring contract:
//   - Anchored patterns (start with `/`): `/etc/*` matches files UNDER
//     /etc but NOT the bare `/etc` directory itself, mirroring shell
//     glob semantics. This avoids R0002 silently allowing access to a
//     profiled directory's parent.
//   - Unanchored `*` (no leading slash): explicit catch-all that also
//     matches the root path `/`. The only way to allowlist `/` itself
//     is an explicit unanchored `*`.
//
// Trailing-slash insensitivity: `/etc/` is treated as `/etc`, and
// `/etc/passwd/` as `/etc/passwd`. Trailing empty path components from
// `strings.Split` are trimmed so `len(regular) > 0` correctly reflects
// the presence of a real path tail when matching trailing `*`.
//
// The empty regular path (`""`) is treated as "no path" and matches
// nothing — distinct from the root path `/`, which matches unanchored
// `*` per the contract above.
func CompareDynamic(dynamicPath, regularPath string) bool {
	// Empty inputs match nothing. Note that SplitPath("") and SplitPath("/")
	// both yield [""] after trim, so without this guard an empty profile
	// entry would silently match the root path.
	if dynamicPath == "" || regularPath == "" {
		return false
	}
	// Dispatch by `*` count:
	//
	//   0 or 1 `*`  → zero-alloc index-based segment walk (compareSegmentsIndex).
	//                 This is the hot R0002 / file-open path; on `main` it
	//                 measured ~16 ns/op, 0 allocs/op. The previous splitPath +
	//                 slice-based descent moved it to ~85 ns/op, 128 B/op,
	//                 2 allocs/op — reverted here.
	//   2+ `*`      → splitPath + DP-memoised core. The memo absorbs the
	//                 exponential re-entry of multi-`*` patterns, which the
	//                 index-based walk would still hit. Allocation cost is
	//                 acceptable because multi-`*` patterns are rare and
	//                 author-supplied, not on the per-event hot path.
	if CountStarSegments(dynamicPath) >= 2 {
		// Multiple `*` segments: try collapsing consecutive runs first.
		// Spec §5.1 makes adjacent `*`s redundant (mid `*` is 0+
		// idempotent; trailing `*` is preserved by the run's last
		// element). After collapse, a 2+-`*` pattern like `/a/*/*/b`
		// reduces to a single-`*` `/a/*/b` and falls into the
		// zero-allocation linear path. Only TRULY non-adjacent
		// multi-`*` patterns (e.g. `/a/*/b/*/c`) still take the memo
		// path. The collapse is only invoked on the slow path; the
		// 0/1-`*` hot path pays no overhead.
		dynamicPath = CollapseConsecutiveStars(dynamicPath)
		if CountStarSegments(dynamicPath) >= 2 {
			dynamic := SplitPath(dynamicPath)
			regular := SplitPath(regularPath)
			memo := make(map[[2]int]bool, len(dynamic)*len(regular))
			return compareSegmentsMemo(dynamic, regular, 0, 0, memo)
		}
	}
	return compareSegmentsIndex(dynamicPath, 0, regularPath, 0)
}

// collapseConsecutiveStars replaces runs of consecutive `*` segments
// with a single `*`. `/a/*/*/b` → `/a/*/b`, `/*/*/*/x` → `/*/x`, etc.
//
// Semantic equivalence under v0.0.1 spec §5.1:
//
//   - Mid `*` matches 0+ segments; collapsing N consecutive mid `*`s
//     into one preserves the 0+ arity.
//   - Trailing `*` matches 1+ segments. A run of consecutive `*`s
//     ending in a trailing position still requires 1+ from the run's
//     last element; the upstream mid `*`s each contribute 0+. Net: 1+.
//   - Therefore `/x/*/*/* ` (trailing run) ≡ `/x/*` (single trailing `*`)
//     and `/x/*/*/y` (mid run) ≡ `/x/*/y` (single mid `*`).
//
// Two-pass implementation:
//
//  1. Zero-allocation scan for `/*/*` with a segment-boundary check on
//     the trailing `*`. If absent (the common case), return p as-is.
//  2. Build the collapsed string via strings.Builder (one allocation).
//
// The hot path — patterns with no adjacent `*` — pays only Pass 1's
// scan cost (~few ns) and no allocation.
//
// Producers' linter SHOULD flag adjacent `*` and have authors collapse
// in source. This matcher-side collapse is the safety net for legacy
// or hand-authored profiles.
func CollapseConsecutiveStars(p string) string {
	// Pass 1: detect any "/*/*" with the second `*` actually a `*` segment.
	needsCollapse := false
	for i := 0; i+3 < len(p); i++ {
		if p[i] != '/' || p[i+1] != '*' || p[i+2] != '/' || p[i+3] != '*' {
			continue
		}
		// p[i+3] is `*`. It's a `*` SEGMENT iff p[i+4] is `/` or string-end.
		if i+4 == len(p) || p[i+4] == '/' {
			needsCollapse = true
			break
		}
	}
	if !needsCollapse {
		return p
	}

	// Pass 2: build collapsed string. We walk segments and drop any `*`
	// segment whose immediate predecessor was also `*`.
	var b strings.Builder
	b.Grow(len(p))
	// Track whether we just emitted a `*` segment, so we know to drop
	// subsequent `*` segments AND their preceding separator slash.
	prevSegWasStar := false
	// Track the position we're about to write a `/` from (so we can
	// drop it if the next segment turns out to be a collapsed `*`).
	pendingSlash := false
	i := 0
	for i < len(p) {
		// Emit any pending separator if the next byte starts a segment.
		// We DON'T emit a `/` if we're about to drop a `*` segment.
		if p[i] == '/' {
			pendingSlash = true
			i++
			continue
		}
		segStart := i
		for i < len(p) && p[i] != '/' {
			i++
		}
		seg := p[segStart:i]
		isStar := seg == "*"
		if isStar && prevSegWasStar {
			// Skip this `*` AND its leading slash (pendingSlash) —
			// they're absorbed by the previous `*`.
			pendingSlash = false
			continue
		}
		if pendingSlash {
			b.WriteByte('/')
			pendingSlash = false
		}
		b.WriteString(seg)
		prevSegWasStar = isStar
	}
	// Trailing slash, if any (the path ended in `/`).
	if pendingSlash {
		b.WriteByte('/')
	}
	return b.String()
}

// countStarSegments counts the number of standalone `*` segments in a
// path. A `*` segment is a single `*` byte bounded by `/` or string-edge
// — distinct from literal `*` characters embedded inside other tokens,
// which are not path wildcards. (Opens/paths only; exec args never treat
// `*` as a wildcard.)
//
// Zero-allocation: scans the string in place.
func CountStarSegments(p string) int {
	count := 0
	for i := 0; i < len(p); i++ {
		if p[i] != '*' {
			continue
		}
		leftOK := i == 0 || p[i-1] == '/'
		rightOK := i+1 == len(p) || p[i+1] == '/'
		if leftOK && rightOK {
			count++
		}
	}
	return count
}

// segAt returns the segment of `s` starting at byte offset `pos`,
// together with the byte index immediately after the segment's trailing
// `/` (or len(s) if the segment is the last one).
//
// Zero-allocation: returns a slice into the source string.
func SegAt(s string, pos int) (seg string, nextPos int) {
	start := pos
	for pos < len(s) && s[pos] != '/' {
		pos++
	}
	seg = s[start:pos]
	if pos < len(s) {
		// skip trailing `/`
		return seg, pos + 1
	}
	return seg, pos
}

// compareSegmentsIndex is the zero-allocation core. It implements the
// same recursive-descent contract as the slice-based compareSegments
// but walks the source strings via byte indices, never splitting.
//
// di / ri are byte offsets into dynamicPath / regularPath respectively.
//
// Per the precondition in CompareDynamic, the dynamic path contains
// AT MOST one `*` segment. The function therefore never re-enters with
// a stale `*` position — backtracking depth is bounded by the number
// of segments in the regular path on the unique mid-`*` shape.
func compareSegmentsIndex(dynamicPath string, di int, regularPath string, ri int) bool {
	dl, rl := len(dynamicPath), len(regularPath)
	if di >= dl {
		return ri >= rl
	}
	dSeg, dNext := SegAt(dynamicPath, di)
	if dSeg == WildcardIdentifier {
		// Trailing `*` matches one OR MORE remaining segments — never
		// zero. This is what makes `/etc/*` not match the bare `/etc`
		// directory, while still matching `/etc/passwd` and deeper.
		if dNext >= dl {
			return ri < rl
		}
		// Mid-path `*`: zero-or-more semantics. Try every offset
		// including ri itself (wildcard consumed zero segments).
		for rTry := ri; rTry <= rl; {
			if compareSegmentsIndex(dynamicPath, dNext, regularPath, rTry) {
				return true
			}
			// Advance rTry past one segment + its trailing `/`.
			for rTry < rl && regularPath[rTry] != '/' {
				rTry++
			}
			if rTry >= rl {
				return false
			}
			rTry++ // skip `/`
		}
		return false
	}
	if ri >= rl {
		return false
	}
	rSeg, rNext := SegAt(regularPath, ri)
	// "⋯" matches any single segment; otherwise a literal segment match
	// (a bare "*" was already handled as a zero-or-more wildcard above).
	if dSeg == DynamicIdentifier || dSeg == rSeg {
		return compareSegmentsIndex(dynamicPath, dNext, regularPath, rNext)
	}
	return false
}

// compareSegmentsMemo is the DP-memoised core, reached only when the
// dynamic pattern has two or more `*` segments. It walks (di, ri) cursor
// pairs over the dynamic and regular slices. The semantics are identical
// to the index-based compareSegmentsIndex: only the redundant re-entry
// is eliminated.
//
// Per-state outcomes are cached in memo. On a cache hit the prior
// boolean is returned directly; on a miss the recursive expansion runs
// and the result is stored before return.
func compareSegmentsMemo(dynamic, regular []string, di, ri int, memo map[[2]int]bool) bool {
	if di == len(dynamic) {
		return ri == len(regular)
	}
	key := [2]int{di, ri}
	if v, ok := memo[key]; ok {
		return v
	}

	var result bool
	if dynamic[di] == WildcardIdentifier {
		// Trailing `*` matches one OR MORE remaining segments — never
		// zero. Identical semantics to the un-memoised form.
		if di == len(dynamic)-1 {
			result = ri < len(regular)
		} else {
			// Mid-path `*`: zero-or-more semantics. Try every offset
			// including i == 0 (wildcard consumed zero segments). The
			// memoisation cache absorbs the repeated work this loop
			// would otherwise inflict on re-entrant states.
			for i := ri; i <= len(regular); i++ {
				if compareSegmentsMemo(dynamic, regular, di+1, i, memo) {
					result = true
					break
				}
			}
		}
	} else if ri == len(regular) {
		result = false
	} else if dynamic[di] == DynamicIdentifier || dynamic[di] == regular[ri] {
		// "⋯" matches any single segment; otherwise a literal segment match.
		result = compareSegmentsMemo(dynamic, regular, di+1, ri+1, memo)
	}

	memo[key] = result
	return result
}

// splitPath splits a path on `/` and trims trailing empty segments
// produced by trailing slashes (e.g. `/etc/` -> ["", "etc"] not
// ["", "etc", ""]). The leading empty segment from a leading slash is
// preserved as the anchor marker. Single-element results are not
// trimmed so the root path `/` retains its `[""]` shape.
func SplitPath(p string) []string {
	s := strings.Split(p, "/")
	for len(s) > 1 && s[len(s)-1] == "" {
		s = s[:len(s)-1]
	}
	return s
}
