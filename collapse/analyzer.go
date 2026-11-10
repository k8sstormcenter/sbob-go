package collapse

import (
	"path"
	"sync"

	"github.com/k8sstormcenter/sbob-go/match"
)

// bufPool reuses byte-slice capacity across AnalyzePath calls. strings.Builder
// was tempting but its Reset() discards the buffer, defeating the pool; a raw
// []byte with len=0/cap-preserved survives reuse. Steady-state per-call cost
// is one string allocation (the final string conversion) and nothing else.
// Thread-safe by virtue of sync.Pool.
const defaultBuildBufCap = 128

var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, defaultBuildBufCap)
		return &b
	},
}

// NewPathAnalyzer builds an analyzer with a single global collapse threshold
// and no per-prefix overrides — equivalent behaviour to the pre-CollapseConfig
// world. Retained so existing callers don't need to change.
func NewPathAnalyzer(threshold int) *PathAnalyzer {
	return NewPathAnalyzerWithConfigs(threshold, nil)
}

// NewPathAnalyzerWithConfigs builds an analyzer whose collapse threshold can
// vary per path prefix. defaultThreshold applies when no CollapseConfig in
// configs matches; configs are checked longest-prefix-wins at walk time.
//
// configs is copied so the caller can reuse or mutate the slice without
// affecting the analyzer.
func NewPathAnalyzerWithConfigs(defaultThreshold int, configs []CollapseConfig) *PathAnalyzer {
	copied := make([]CollapseConfig, len(configs))
	copy(copied, configs)
	return &PathAnalyzer{
		RootNodes:  make(map[string]*SegmentNode),
		threshold:  defaultThreshold,
		configs:    copied,
		defaultCfg: CollapseConfig{Prefix: "/", Threshold: defaultThreshold},
	}
}

// effectiveThreshold returns the collapse threshold applicable to the given
// path prefix, picking the longest matching CollapseConfig or falling back
// to the analyzer's default. Loop is O(len(configs)) and configs is small
// (five entries in practice); no allocations.
//
// Tiebreak on equal-length prefixes: FIRST entry wins (strict `>`). This
// must mirror FindConfigForPath so callers using FindConfigForPath to
// introspect the active config see the same result the analyzer actually
// uses at walk time. Mismatched comparators (`>=` vs `>`) on duplicate
// prefixes are a silent footgun for anyone who doesn't dedupe configs.
func (ua *PathAnalyzer) effectiveThreshold(pathPrefix string) int {
	bestLen := -1
	best := ua.threshold
	for i := range ua.configs {
		c := &ua.configs[i]
		if len(c.Prefix) > bestLen && hasPrefixAtBoundary(pathPrefix, c.Prefix) {
			bestLen = len(c.Prefix)
			best = c.Threshold
		}
	}
	return best
}

// hasPrefixAtBoundary is like strings.HasPrefix but only matches if the
// prefix ends at a path boundary (either pathPrefix == prefix, or the next
// rune in pathPrefix is '/'). Prevents "/etc" matching "/etcd".
//
// Special case: prefix == "/" — the trailing '/' already implies a boundary,
// and any absolute path begins with '/'. Without this case, a user-supplied
// `{Prefix:"/", Threshold:X}` config would silently never match for any
// path past the root (e.g. "/foo" since pathPrefix[1] == 'f', not '/'),
// which means an explicit catch-all override could not actually override
// the analyzer's default threshold.
func hasPrefixAtBoundary(pathPrefix, prefix string) bool {
	// Empty-prefix guard: without this, hasPrefixAtBoundary("/foo", "")
	// falls through to
	// pathPrefix[0] == '/', which is true for any absolute path —
	// effectively treating `""` as a root-matching prefix. None of the
	// shipped configs use an empty prefix, but operators could supply
	// one via CollapseConfiguration CR, and an explicit guard makes
	// the invariant load-bearing rather than incidental.
	if prefix == "" {
		return true
	}
	// Normalise operator-supplied trailing slashes. A CollapseConfig with
	// `Prefix:"/etc/"` semantically means the same as `Prefix:"/etc"` —
	// the slash is the implicit segment boundary. Without this, an
	// operator authoring `/etc/` via CollapseConfiguration CR would
	// silently never match (the byte-equality check rejects `/etc/foo`
	// because `/etc/foo[:5]` = `/etc/` only when prefix is `/etc/`, which
	// then forces the next char to be at offset 5 — but offset 5 IS the
	// boundary slash itself; the boundary check expects another `/`
	// AFTER that, which is `f`, not `/` → false).
	for len(prefix) > 1 && prefix[len(prefix)-1] == '/' {
		prefix = prefix[:len(prefix)-1]
	}
	if len(pathPrefix) < len(prefix) {
		return false
	}
	if pathPrefix[:len(prefix)] != prefix {
		return false
	}
	if len(pathPrefix) == len(prefix) {
		return true
	}
	if prefix == "/" {
		return true
	}
	return pathPrefix[len(prefix)] == '/'
}

func (ua *PathAnalyzer) AnalyzePath(p, identifier string) (string, error) {
	// path.Clean("") returns "." — an empty input must stay empty, never mint
	// a relative segment into a consolidated profile.
	if p == "" {
		return "", nil
	}
	p = path.Clean(p)
	node, exists := ua.RootNodes[identifier]
	if !exists {
		node = &SegmentNode{
			SegmentName: identifier,
			Count:       0,
			Children:    make(map[string]*SegmentNode),
		}
		ua.RootNodes[identifier] = node
	}
	return ua.processSegments(node, p), nil
}

func (ua *PathAnalyzer) processSegments(node *SegmentNode, p string) string {
	// Acquire a pooled byte-slice. len=0, cap preserved from previous reuse.
	bufPtr := bufPool.Get().(*[]byte)
	buf := (*bufPtr)[:0]
	if cap(buf) < len(p) {
		// Pooled capacity is too small for this input; grow once.
		buf = make([]byte, 0, len(p)+16)
	}

	currentNode := node
	i := 0
	for {
		start := i
		for i < len(p) && p[i] != '/' {
			i++
		}
		segment := p[start:i]
		// Two thresholds at two scopes — necessary because processSegment
		// and updateNodeStats ask different questions about different nodes:
		//
		// insertThreshold is for the PARENT node's config (path prefix up
		// to, but not including, the current segment). It answers: "if
		// we need to add `segment` under the parent, should we wildcard
		// the parent's children instead (threshold == 1)?". Using p[:i]
		// here would incorrectly apply the current segment's own config,
		// causing a {Prefix: "/instant", Threshold: 1} rule to wildcard
		// the "instant" segment itself and produce "/*/*/*" rather than
		// "/instant/*".
		//
		// collapseThreshold is for the CURRENT node's config (path prefix
		// INCLUDING the current segment, i.e. the node we just descended
		// to). It answers: "do this node's direct children exceed the
		// collapse threshold configured for this node's path?". Here we
		// do want p[:i] — updateNodeStats then collapses the current
		// node's children to ⋯ when Count > threshold.
		insertThreshold := ua.effectiveThreshold(p[:start])
		collapseThreshold := ua.effectiveThreshold(p[:i])
		// Collapsed on sight rather than on cardinality: see
		// procIdentifierSegment and generatedCounterSegment.
		if procIdentifierSegment(p[:start], segment) || generatedCounterSegment(segment) {
			segment = procIdentifierKey
		}
		currentNode = ua.processSegment(currentNode, segment, insertThreshold)
		ua.updateNodeStats(currentNode, collapseThreshold)
		buf = append(buf, currentNode.SegmentName...)
		// Wildcard absorbs the rest of the path: once a segment has been
		// emitted as `*`, walking deeper would just append more "/*"
		// suffixes, producing "/a/*/*/*" where the correct output is
		// "/a/*". Terminate emission here.
		if currentNode.SegmentName == match.WildcardIdentifier {
			break
		}
		i++
		if len(p) < i {
			break
		}
		buf = append(buf, '/')
	}

	// Post-process: collapse runs of adjacent match.DynamicIdentifier segments
	// (e.g. "/a/⋯/⋯/b") into a single match.WildcardIdentifier ("/a/*/b"). Done
	// in place by shrinking buf — zero allocation because the output is
	// always shorter than the input.
	buf = collapseAdjacentDynamic(buf)

	// string(buf) always copies, so it is safe to return the pool capacity
	// immediately afterwards — the returned string does not alias buf.
	out := string(buf)
	*bufPtr = buf
	bufPool.Put(bufPtr)
	return out
}

// collapseAdjacentDynamic compacts buf in place: any run of
// "⋯/⋯[/⋯…]" becomes a single "*". Returns a buf[:n] slice where n is
// the compacted length. Does not allocate; suitable for the hot path.
func collapseAdjacentDynamic(buf []byte) []byte {
	// match.DynamicIdentifier is U+22EF, three UTF-8 bytes: 0xE2 0x8B 0xAF.
	const d0, d1, d2 = 0xE2, 0x8B, 0xAF
	const dynLen = 3
	isDyn := func(i int) bool {
		return i+dynLen <= len(buf) && buf[i] == d0 && buf[i+1] == d1 && buf[i+2] == d2
	}

	out := 0
	i := 0
	for i < len(buf) {
		// Need at least "⋯/⋯" (7 bytes) to trigger a collapse.
		if isDyn(i) && i+dynLen+1+dynLen <= len(buf) && buf[i+dynLen] == '/' && isDyn(i+dynLen+1) {
			buf[out] = '*'
			out++
			// Consume "⋯/⋯" plus any further "/⋯" in the run.
			i += dynLen + 1 + dynLen
			for i+1+dynLen <= len(buf) && buf[i] == '/' && isDyn(i+1) {
				i += 1 + dynLen
			}
			continue
		}
		buf[out] = buf[i]
		out++
		i++
	}
	return buf[:out]
}

// procIdentifierKey is the trie key under which every pid and every tid at the
// same level shares one subtree. It is deliberately NOT match.DynamicIdentifier: a ⋯
// child makes the whole level dynamic (IsNextDynamic), which would route
// /proc/self, /proc/cpuinfo and /proc/sys through it as well and cost the
// profile the distinction between reading a named procfs file and walking a
// task directory. The key can never collide with a real segment because
// AnalyzePath splits on '/', so no segment can contain one.
//
// A node reached through this key still emits match.DynamicIdentifier — see
// handleNewSegment — so the key is an internal routing detail and never
// appears in a profile.
const procIdentifierKey = "/pid"

// procIdentifierSegment reports whether segment is a kernel-assigned number in
// a procfs path. prefix is the path up to and including the '/' that precedes
// segment.
//
// The rule is deliberately blunt: ANY all-digit segment anywhere under /proc is
// an identifier. A pid, a tid, a file descriptor, an fdinfo entry, an irq
// number — none of them carries meaning past the run that produced it, and a
// profile that records one records a path that can never match again.
//
// An earlier version anchored this to /proc/<pid> and /proc/<pid>/task/<tid>
// only. That is the case that hurts most, because the collapse threshold fires
// on child COUNT and a container with a few threads never produces enough
// distinct tids to reach it — but the anchoring bought nothing. There is no
// all-digit segment under /proc whose exact value is worth keeping, so the
// narrow rule was extra machinery guarding a distinction that does not exist.
//
// The rule stops at /proc for a reason that does not apply below it. Outside
// /proc an integer in a path is routinely meaningful — a StatefulSet ordinal,
// a container restart index in a log path, a rotation suffix — and collapsing
// those would merge paths an operator means to tell apart. There the threshold
// is the right instrument, because it acts on observed cardinality rather than
// on a guess about what the number means.
//
// An already-collapsed identifier counts too: the node agent rewrites
// /proc/<pid> at report time, and storage re-analyses its own emitted patterns
// on every save, so both arrive here carrying the dynamic identifier and must
// take the same route as a literal number.
//
// Names are untouched. /proc/self, /proc/thread-self, /proc/net, /proc/sys and
// every other procfs name has no digits-only segment and stays literal.
func procIdentifierSegment(prefix, segment string) bool {
	const procRoot = "/proc/"
	if len(prefix) < len(procRoot) || prefix[:len(procRoot)] != procRoot {
		return false
	}
	return isNumericSegment(segment) || segment == match.DynamicIdentifier
}

// generatedCounterSegment reports whether segment is a zero-padded counter —
// a machine-generated sequence number, never a name a person chose.
//
// These are volatile in the way a tid is: the value advances every time and
// never comes back, so a profile that records one records a path that can
// never match again. Unlike a tid the cardinality threshold does eventually
// reach them, but not before learning ends, and learning ends first for
// exactly the workloads that produce them slowly. Observed in production:
// nginx proxy_temp writes /tmp/nginx/proxy_temp/<n>/<nn>/0000000001, one file
// landed inside the learning window, and the completed profile froze that
// single literal while the running container went on to write 0000000002 and
// beyond — paths its own allowlist can never match.
//
// The padding is what makes this safe to apply outside /proc, where a bare
// integer is routinely meaningful. A StatefulSet ordinal is 0, 1, 2; a
// container restart index is 0; a log rotation suffix is 1 or 2. None of them
// is written 0000000001. Requiring a leading zero AND at least four digits
// takes the generated sequence numbers and leaves every meaningful integer
// alone — "0" and "007" stay literal.
func generatedCounterSegment(segment string) bool {
	const minCounterDigits = 4
	return len(segment) >= minCounterDigits && segment[0] == '0' && isNumericSegment(segment)
}

func isNumericSegment(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func (ua *PathAnalyzer) processSegment(node *SegmentNode, segment string, threshold int) *SegmentNode {
	if segment == match.DynamicIdentifier {
		return ua.handleDynamicSegment(node)
	}
	// An explicitly supplied * subsumes a ⋯ already at this node: ⋯ is exactly
	// one segment and * is zero-or-more, so routing the * through the existing
	// ⋯ child silently NARROWS the profile and the deeper paths the author
	// meant to admit start alerting.
	if segment == match.WildcardIdentifier {
		if wildcardChild, exists := node.Children[match.WildcardIdentifier]; exists {
			return wildcardChild
		}
		return ua.createWildcardNode(node)
	}
	// Wildcard short-circuit: once a node has a * child, all paths through
	// it go there. This is the glob-style "collapse everything below here"
	// behaviour; set up either by threshold=1 (see below) or by a caller
	// explicitly feeding a match.WildcardIdentifier segment.
	if wildcardChild, exists := node.Children[match.WildcardIdentifier]; exists {
		return wildcardChild
	}
	if node.IsNextDynamic() {
		if len(node.Children) > 1 {
			temp := node.Children[match.DynamicIdentifier]
			node.Children = map[string]*SegmentNode{}
			node.Children[match.DynamicIdentifier] = temp
		}
		return node.Children[match.DynamicIdentifier]
	}
	if child, exists := node.Children[segment]; exists {
		return child
	}
	// Threshold-1 short-circuit: a prefix explicitly configured to accept
	// one unique child (CollapseConfig Threshold == 1) collapses to * on
	// the first *new* segment rather than going through the ⋯ path. This
	// matches the caller's intent of "anything under /app is noise".
	if threshold == 1 {
		return ua.createWildcardNode(node)
	}
	return ua.handleNewSegment(node, segment)
}

func (ua *PathAnalyzer) handleNewSegment(node *SegmentNode, segment string) *SegmentNode {
	node.Count++
	name := segment
	if segment == procIdentifierKey {
		name = match.DynamicIdentifier
	}
	newNode := &SegmentNode{
		SegmentName: name,
		Count:       0,
		Children:    make(map[string]*SegmentNode),
	}
	node.Children[segment] = newNode
	return newNode
}

func (ua *PathAnalyzer) handleDynamicSegment(node *SegmentNode) *SegmentNode {
	if dynamicChild, exists := node.Children[match.DynamicIdentifier]; exists {
		return dynamicChild
	} else {
		return ua.createDynamicNode(node)
	}
}

// createWildcardNode replaces all of node's existing children with a single
// match.WildcardIdentifier (*) child, absorbing any existing subtree counts into it.
// Used for the threshold-1 short-circuit: a CollapseConfig with Threshold == 1
// means "any new child segment under this prefix is noise", so the FIRST new
// segment immediately wildcards (there are typically no children to absorb on
// the first call; if the analyzer has previously seen children there, they
// get folded into the wildcard subtree at this point). The semantics are
// pinned by TestAnalyzeOpensThreshold1ImmediateWildcard /
// "single path - no collapse yet" which expects /instant/only-child/data
// to collapse to /instant/* after a single insert.
func (ua *PathAnalyzer) createWildcardNode(node *SegmentNode) *SegmentNode {
	wildcard := &SegmentNode{
		SegmentName: match.WildcardIdentifier,
		Count:       0,
		Children:    make(map[string]*SegmentNode),
	}
	// Absorb any previously-accumulated children. Mirrors createDynamicNode.
	for _, child := range node.Children {
		shallowChildrenCopy(child, wildcard)
	}
	node.Children = map[string]*SegmentNode{
		match.WildcardIdentifier: wildcard,
	}
	return wildcard
}

func (ua *PathAnalyzer) createDynamicNode(node *SegmentNode) *SegmentNode {
	dynamicNode := &SegmentNode{
		SegmentName: match.DynamicIdentifier,
		Count:       0,
		Children:    make(map[string]*SegmentNode),
	}

	// Copy all existing children to the new dynamic node
	for _, child := range node.Children {
		shallowChildrenCopy(child, dynamicNode)
	}

	// Replace all children with the new dynamic node
	node.Children = map[string]*SegmentNode{
		match.DynamicIdentifier: dynamicNode,
	}

	return dynamicNode
}

// updateNodeStats collapses node's children into a single ⋯ (match.DynamicIdentifier)
// child once the number of distinct children exceeds the provided threshold.
// Threshold is passed in by the caller so per-prefix overrides (via
// CollapseConfig) can take effect without this function knowing about them.
func (ua *PathAnalyzer) updateNodeStats(node *SegmentNode, threshold int) {
	if node.Count > threshold && !node.IsNextDynamic() {
		dynamicChild := &SegmentNode{
			SegmentName: match.DynamicIdentifier,
			Count:       0,
			Children:    make(map[string]*SegmentNode),
		}

		// Copy all descendants
		for _, child := range node.Children {
			shallowChildrenCopy(child, dynamicChild)
		}

		// The absorbed children become dynamicChild's own children —
		// update dynamicChild.Count so subsequent updateNodeStats calls
		// on this node can correctly detect that the grandchild level
		// also exceeds its threshold and trigger the next collapse.
		// Without this, multi-level grids like /a/{many}/{many}/leaf
		// only collapse the first level and leave the grandchild
		// literals intact in the output.
		dynamicChild.Count = len(dynamicChild.Children)

		node.Children = map[string]*SegmentNode{
			match.DynamicIdentifier: dynamicChild,
		}
	}
}

func shallowChildrenCopy(src, dst *SegmentNode) {
	for segmentName := range src.Children {
		if _, ok := dst.Children[segmentName]; !ok {
			dst.Children[segmentName] = src.Children[segmentName]
		} else {
			dst.Children[segmentName].Count += src.Children[segmentName].Count
			shallowChildrenCopy(src.Children[segmentName], dst.Children[segmentName])
		}
	}
}

// FindConfigForPath returns a value copy of the CollapseConfig whose
// Prefix matches `path` with the longest match. Falls back to the
// analyzer's default config (Prefix:"/") when no per-prefix override
// applies, so the result is always meaningful — there is no "no match"
// signal.
//
// Returning by value keeps the analyzer's internal state immutable
// from callers. NewPathAnalyzerWithConfigs already makes a defensive
// inbound copy of `configs`; this is its outbound twin. Without it,
// `cfg := analyzer.FindConfigForPath(p); cfg.Threshold = 1` would
// silently mutate the analyzer's threshold map for every future call.
func (ua *PathAnalyzer) FindConfigForPath(path string) CollapseConfig {
	bestIdx := -1
	bestLen := -1
	for i := range ua.configs {
		cfg := &ua.configs[i]
		if hasPrefixAtBoundary(path, cfg.Prefix) && len(cfg.Prefix) > bestLen {
			bestIdx = i
			bestLen = len(cfg.Prefix)
		}
	}
	if bestIdx == -1 {
		return ua.defaultCfg
	}
	return ua.configs[bestIdx]
}

// CollapseAdjacentDynamicIdentifiers replaces runs of adjacent
// match.DynamicIdentifier segments (e.g. "/a/⋯/⋯/b") with a single
// match.WildcardIdentifier ("/a/*/b"). Static segments between dynamic
// identifiers prevent collapsing. String wrapper over the internal
// byte-level collapseAdjacentDynamic, intended for test coverage.
func CollapseAdjacentDynamicIdentifiers(p string) string {
	return string(collapseAdjacentDynamic([]byte(p)))
}
