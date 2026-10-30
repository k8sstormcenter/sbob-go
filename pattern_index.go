package dynamicpathdetector

import (
	"strings"

	types "github.com/kubescape/storage/pkg/apis/softwarecomposition"
)

type execPatternNode struct {
	literal  map[string]*execPatternNode
	any      *execPatternNode
	segments *execSegmentNode
	spans    []execSpanChild
	terminal []int
}

type execSpanChild struct {
	token string
	next  *execPatternNode
}

type execSegmentNode struct {
	literal map[string]*execSegmentNode
	any     *execSegmentNode
	end     *execPatternNode
}

type execPatternIndex struct {
	execs      []types.ExecCalls
	roots      map[int]*execPatternNode
	anyByFirst map[string][]int
	anyOther   []int
	states     []*execPatternNode
	next       []*execPatternNode
	segStates  []*execSegmentNode
	segNext    []*execSegmentNode
}

func newExecPatternIndex(execs []types.ExecCalls, patterns []int) *execPatternIndex {
	idx := &execPatternIndex{execs: execs, roots: make(map[int]*execPatternNode), anyByFirst: make(map[string][]int)}
	for _, j := range patterns {
		idx.insert(j, execs[j].Args)
	}
	return idx
}

func (idx *execPatternIndex) insert(j int, args []string) {
	if len(args) == 0 {
		return
	}
	for _, arg := range args {
		if arg == ExecArgsWildcard {
			first := args[0]
			if first == ExecArgsWildcard || strings.Contains(first, DynamicIdentifier) {
				idx.anyOther = append(idx.anyOther, j)
			} else {
				idx.anyByFirst[first] = append(idx.anyByFirst[first], j)
			}
			return
		}
	}
	node, ok := idx.roots[len(args)]
	if !ok {
		node = &execPatternNode{}
		idx.roots[len(args)] = node
	}
	for _, arg := range args {
		node = node.child(arg)
	}
	node.terminal = append(node.terminal, j)
}

func (n *execPatternNode) child(arg string) *execPatternNode {
	if arg == DynamicIdentifier {
		if n.any == nil {
			n.any = &execPatternNode{}
		}
		return n.any
	}
	if !strings.Contains(arg, DynamicIdentifier) {
		if n.literal == nil {
			n.literal = make(map[string]*execPatternNode)
		}
		child, ok := n.literal[arg]
		if !ok {
			child = &execPatternNode{}
			n.literal[arg] = child
		}
		return child
	}
	hasSpan := false
	forEachSegment(arg, func(seg string) {
		if seg == ExecArgsWildcard {
			hasSpan = true
		}
	})
	if hasSpan {
		for i := range n.spans {
			if n.spans[i].token == arg {
				return n.spans[i].next
			}
		}
		child := &execPatternNode{}
		n.spans = append(n.spans, execSpanChild{token: arg, next: child})
		return child
	}
	if n.segments == nil {
		n.segments = &execSegmentNode{}
	}
	seg := n.segments
	forEachSegment(arg, func(s string) {
		seg = seg.child(s)
	})
	if seg.end == nil {
		seg.end = &execPatternNode{}
	}
	return seg.end
}

func (s *execSegmentNode) child(seg string) *execSegmentNode {
	if seg == DynamicIdentifier {
		if s.any == nil {
			s.any = &execSegmentNode{}
		}
		return s.any
	}
	if s.literal == nil {
		s.literal = make(map[string]*execSegmentNode)
	}
	child, ok := s.literal[seg]
	if !ok {
		child = &execSegmentNode{}
		s.literal[seg] = child
	}
	return child
}

func forEachSegment(s string, fn func(seg string)) {
	start := 0
	for {
		i := strings.IndexByte(s[start:], '/')
		if i < 0 {
			fn(s[start:])
			return
		}
		fn(s[start : start+i])
		start += i + 1
	}
}

func (idx *execPatternIndex) coverers(args []string, self int) []int {
	var out []int
	if root, ok := idx.roots[len(args)]; ok {
		idx.states = append(idx.states[:0], root)
		for _, arg := range args {
			idx.next = idx.next[:0]
			for _, n := range idx.states {
				if n.literal != nil {
					if c, ok := n.literal[arg]; ok {
						idx.next = append(idx.next, c)
					}
				}
				if n.any != nil {
					idx.next = append(idx.next, n.any)
				}
				if n.segments != nil {
					idx.matchSegments(n.segments, arg)
				}
				for i := range n.spans {
					if argMatches(n.spans[i].token, arg) {
						idx.next = append(idx.next, n.spans[i].next)
					}
				}
			}
			idx.states, idx.next = idx.next, idx.states
			if len(idx.states) == 0 {
				break
			}
		}
		for _, n := range idx.states {
			for _, j := range n.terminal {
				if j != self {
					out = append(out, j)
				}
			}
		}
	}
	for _, j := range idx.anyByFirst[args[0]] {
		if j != self && CompareExecArgs(idx.execs[j].Args, args) {
			out = append(out, j)
		}
	}
	for _, j := range idx.anyOther {
		if j != self && CompareExecArgs(idx.execs[j].Args, args) {
			out = append(out, j)
		}
	}
	return out
}

func (idx *execPatternIndex) matchSegments(root *execSegmentNode, arg string) {
	idx.segStates = append(idx.segStates[:0], root)
	forEachSegment(arg, func(seg string) {
		if len(idx.segStates) == 0 {
			return
		}
		idx.segNext = idx.segNext[:0]
		for _, s := range idx.segStates {
			if s.literal != nil {
				if c, ok := s.literal[seg]; ok {
					idx.segNext = append(idx.segNext, c)
				}
			}
			if s.any != nil {
				idx.segNext = append(idx.segNext, s.any)
			}
		}
		idx.segStates, idx.segNext = idx.segNext, idx.segStates
	})
	for _, s := range idx.segStates {
		if s.end != nil {
			idx.next = append(idx.next, s.end)
		}
	}
}

type openPatternNode struct {
	literal  map[string]*openPatternNode
	dynamic  *openPatternNode
	star     *openPatternNode
	isStar   bool
	terminal []int
}

type openPatternIndex struct {
	opens    []types.OpenCalls
	root     *openPatternNode
	fallback []int
	states   map[*openPatternNode]bool
	next     map[*openPatternNode]bool
	segments []string
}

func newOpenPatternIndex(opens []types.OpenCalls, patterns []int) *openPatternIndex {
	idx := &openPatternIndex{opens: opens, root: &openPatternNode{}, states: make(map[*openPatternNode]bool), next: make(map[*openPatternNode]bool)}
	for _, pi := range patterns {
		p := opens[pi].Path
		if countStarSegments(p) >= 2 {
			idx.fallback = append(idx.fallback, pi)
			continue
		}
		node := idx.root
		for _, seg := range pathSegments(p, nil) {
			node = node.child(seg)
		}
		node.terminal = append(node.terminal, pi)
	}
	return idx
}

func (n *openPatternNode) child(seg string) *openPatternNode {
	switch seg {
	case WildcardIdentifier:
		if n.star == nil {
			n.star = &openPatternNode{isStar: true}
		}
		return n.star
	case DynamicIdentifier:
		if n.dynamic == nil {
			n.dynamic = &openPatternNode{}
		}
		return n.dynamic
	}
	if n.literal == nil {
		n.literal = make(map[string]*openPatternNode)
	}
	child, ok := n.literal[seg]
	if !ok {
		child = &openPatternNode{}
		n.literal[seg] = child
	}
	return child
}

func pathSegments(p string, buf []string) []string {
	buf = buf[:0]
	for i := 0; i < len(p); {
		seg, next := segAt(p, i)
		buf = append(buf, seg)
		i = next
	}
	return buf
}

func (idx *openPatternIndex) add(set map[*openPatternNode]bool, n *openPatternNode, absorbed bool) {
	if prev, ok := set[n]; ok {
		set[n] = prev || absorbed
	} else {
		set[n] = absorbed
	}
	for s := n.star; s != nil; s = s.star {
		if _, ok := set[s]; !ok {
			set[s] = false
		}
	}
}

func (idx *openPatternIndex) coverers(path string) []int {
	var out []int
	if path == "" {
		return out
	}
	idx.segments = pathSegments(path, idx.segments)
	clear(idx.states)
	idx.add(idx.states, idx.root, false)
	for _, seg := range idx.segments {
		clear(idx.next)
		for n := range idx.states {
			if n.isStar {
				idx.add(idx.next, n, true)
			}
			if n.literal != nil {
				if c, ok := n.literal[seg]; ok {
					idx.add(idx.next, c, false)
				}
			}
			if n.dynamic != nil {
				idx.add(idx.next, n.dynamic, false)
			}
		}
		idx.states, idx.next = idx.next, idx.states
		if len(idx.states) == 0 {
			break
		}
	}
	for n, absorbed := range idx.states {
		if n.isStar && !absorbed {
			continue
		}
		out = append(out, n.terminal...)
	}
	for _, pi := range idx.fallback {
		if CompareDynamic(idx.opens[pi].Path, path) {
			out = append(out, pi)
		}
	}
	return out
}
