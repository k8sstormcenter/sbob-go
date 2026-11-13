package compose

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/k8sstormcenter/sbob-go/profile"
)

var (
	ErrNoFragments = errors.New("no fragments to merge")
	ErrNilFragment = errors.New("nil fragment")
	ErrClassCount  = errors.New("one class per fragment is required")
)

type Option func(*options)

type options struct {
	classes    []string
	precedence []string
	byClass    bool
	canonical  bool
}

func StackByClass(classes []string, precedence ...string) Option {
	return func(o *options) {
		o.classes = classes
		o.precedence = precedence
		o.byClass = true
	}
}

func CanonicalOrder() Option {
	return func(o *options) {
		o.canonical = true
	}
}

func Merge(fragments []*profile.RuntimeProfile, opts ...Option) (*profile.RuntimeProfile, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	ordered, err := stack(fragments, o)
	if err != nil {
		return nil, err
	}

	subject := profile.Subject("")
	top := newSection()
	archs := map[string]bool{}
	matchLabels := map[string]string{}
	var matchExpressions []profile.LabelSelectorRequirement
	groups := [3]*sectionGroup{newSectionGroup(), newSectionGroup(), newSectionGroup()}

	for _, f := range ordered {
		if f.Subject != "" {
			subject = f.Subject
		}
		s := &f.Spec
		for _, a := range s.Architectures {
			archs[a] = true
		}
		for k, v := range s.MatchLabels {
			matchLabels[k] = v
		}
		matchExpressions = append(matchExpressions, cloneRequirements(s.MatchExpressions)...)
		if err := top.add(specBody(s)); err != nil {
			return nil, err
		}
		for i, list := range [3][]profile.Container{s.Containers, s.InitContainers, s.EphemeralContainers} {
			for j := range list {
				if err := groups[i].of(list[j].Name).add(containerBody(&list[j])); err != nil {
					return nil, err
				}
			}
		}
	}

	out := &profile.RuntimeProfile{
		APIVersion: profile.APIVersion,
		Kind:       profile.Kind,
		Subject:    subject,
	}
	b, err := top.body()
	if err != nil {
		return nil, err
	}
	out.Spec = profile.Spec{
		Architectures:        sortedKeys(archs),
		Capabilities:         b.capabilities,
		Execs:                b.execs,
		Opens:                b.opens,
		Syscalls:             b.syscalls,
		Endpoints:            b.endpoints,
		Ingress:              b.ingress,
		Egress:               b.egress,
		SeccompProfile:       b.seccomp,
		ImageID:              b.imageID,
		ImageTag:             b.imageTag,
		RulePolicies:         b.policies,
		IdentifiedCallStacks: b.stacks,
		Subtract:             b.subtract,
		MatchExpressions:     matchExpressions,
		Extra:                b.extra,
	}
	if len(matchLabels) > 0 {
		out.Spec.MatchLabels = matchLabels
	}
	if out.Spec.Containers, err = groups[0].containers(); err != nil {
		return nil, err
	}
	if out.Spec.InitContainers, err = groups[1].containers(); err != nil {
		return nil, err
	}
	if out.Spec.EphemeralContainers, err = groups[2].containers(); err != nil {
		return nil, err
	}
	return out, nil
}

func stack(fragments []*profile.RuntimeProfile, o options) ([]*profile.RuntimeProfile, error) {
	if len(fragments) == 0 {
		return nil, ErrNoFragments
	}
	if o.byClass && len(o.classes) != len(fragments) {
		return nil, fmt.Errorf("%w: %d classes for %d fragments", ErrClassCount, len(o.classes), len(fragments))
	}
	type item struct {
		doc    *profile.RuntimeProfile
		rank   int
		digest string
	}
	items := make([]item, len(fragments))
	for i, f := range fragments {
		if f == nil {
			return nil, fmt.Errorf("%w at index %d", ErrNilFragment, i)
		}
		items[i].doc = f
		if o.byClass {
			items[i].rank = rank(o.precedence, o.classes[i])
		}
		if o.canonical {
			d, err := digest(f)
			if err != nil {
				return nil, err
			}
			items[i].digest = d
		}
	}
	if o.canonical {
		sort.SliceStable(items, func(i, j int) bool { return items[i].digest < items[j].digest })
	}
	if o.byClass {
		sort.SliceStable(items, func(i, j int) bool { return items[i].rank < items[j].rank })
	}
	out := make([]*profile.RuntimeProfile, len(items))
	for i := range items {
		out[i] = items[i].doc
	}
	return out, nil
}

func rank(precedence []string, class string) int {
	for i, c := range precedence {
		if c == class {
			return i
		}
	}
	return len(precedence)
}

func digest(p *profile.RuntimeProfile) (string, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

type body struct {
	capabilities []string
	execs        []profile.ExecCall
	opens        []profile.OpenCall
	syscalls     []string
	endpoints    []profile.Endpoint
	ingress      []profile.NetworkNeighbor
	egress       []profile.NetworkNeighbor
	seccomp      json.RawMessage
	imageID      string
	imageTag     string
	policies     map[string]profile.RulePolicy
	stacks       json.RawMessage
	subtract     *profile.Subtract
	extra        map[string]json.RawMessage
}

func specBody(s *profile.Spec) body {
	return body{
		capabilities: s.Capabilities,
		execs:        s.Execs,
		opens:        s.Opens,
		syscalls:     s.Syscalls,
		endpoints:    s.Endpoints,
		ingress:      s.Ingress,
		egress:       s.Egress,
		seccomp:      s.SeccompProfile,
		imageID:      s.ImageID,
		imageTag:     s.ImageTag,
		policies:     s.RulePolicies,
		stacks:       s.IdentifiedCallStacks,
		subtract:     s.Subtract,
		extra:        s.Extra,
	}
}

func containerBody(c *profile.Container) body {
	return body{
		capabilities: c.Capabilities,
		execs:        c.Execs,
		opens:        c.Opens,
		syscalls:     c.Syscalls,
		endpoints:    c.Endpoints,
		ingress:      c.Ingress,
		egress:       c.Egress,
		seccomp:      c.SeccompProfile,
		imageID:      c.ImageID,
		imageTag:     c.ImageTag,
		policies:     c.RulePolicies,
		stacks:       c.IdentifiedCallStacks,
		subtract:     c.Subtract,
		extra:        c.Extra,
	}
}

type neighbors struct {
	order []string
	byID  map[string]profile.NetworkNeighbor
}

func (n *neighbors) add(list []profile.NetworkNeighbor) {
	for _, e := range list {
		if _, ok := n.byID[e.Identifier]; !ok {
			if n.byID == nil {
				n.byID = map[string]profile.NetworkNeighbor{}
			}
			n.order = append(n.order, e.Identifier)
		}
		n.byID[e.Identifier] = cloneNeighbor(e)
	}
}

func (n *neighbors) list() []profile.NetworkNeighbor {
	var out []profile.NetworkNeighbor
	for _, id := range n.order {
		out = append(out, n.byID[id])
	}
	return out
}

type lists struct {
	capabilities map[string]bool
	syscalls     map[string]bool
	execs        []profile.ExecCall
	opens        []profile.OpenCall
	endpoints    []profile.Endpoint
	seen         map[string]bool
	ingress      neighbors
	egress       neighbors
	extra        map[string]json.RawMessage
}

func newLists() *lists {
	return &lists{capabilities: map[string]bool{}, syscalls: map[string]bool{}, seen: map[string]bool{}}
}

func (l *lists) once(kind string, key any) (bool, error) {
	b, err := json.Marshal(key)
	if err != nil {
		return false, err
	}
	k := kind + string(b)
	if l.seen[k] {
		return false, nil
	}
	l.seen[k] = true
	return true, nil
}

func (l *lists) add(capabilities []string, execs []profile.ExecCall, opens []profile.OpenCall, syscalls []string, endpoints []profile.Endpoint, ingress, egress []profile.NetworkNeighbor, extra map[string]json.RawMessage) error {
	for _, c := range capabilities {
		l.capabilities[c] = true
	}
	for _, s := range syscalls {
		l.syscalls[s] = true
	}
	for _, e := range execs {
		n := profile.ExecCall{Path: e.Path, ArgsRequired: e.ArgsRequired, Extra: cloneExtra(e.Extra)}
		if len(e.Args) > 0 {
			n.Args = cloneStrings(e.Args)
		}
		if len(e.Envs) > 0 {
			n.Envs = cloneStrings(e.Envs)
		}
		fresh, err := l.once("x", execKey{n.Path, n.Args, n.Envs, n.ArgsRequired, n.Extra})
		if err != nil {
			return err
		}
		if fresh {
			l.execs = append(l.execs, n)
		}
	}
	for _, e := range opens {
		n := profile.OpenCall{Path: e.Path, Flags: cloneStrings(e.Flags), Extra: cloneExtra(e.Extra)}
		fresh, err := l.once("o", openKey{n.Path, n.Flags, n.Extra})
		if err != nil {
			return err
		}
		if fresh {
			l.opens = append(l.opens, n)
		}
	}
	for _, e := range endpoints {
		n := profile.Endpoint{Endpoint: e.Endpoint, Internal: e.Internal, Direction: e.Direction, Headers: cloneRaw(e.Headers), Extra: cloneExtra(e.Extra)}
		if len(e.Methods) > 0 {
			n.Methods = cloneStrings(e.Methods)
		}
		fresh, err := l.once("e", endpointKey{n.Endpoint, n.Methods, n.Internal, n.Direction, n.Headers, n.Extra})
		if err != nil {
			return err
		}
		if fresh {
			l.endpoints = append(l.endpoints, n)
		}
	}
	l.ingress.add(ingress)
	l.egress.add(egress)
	for k, v := range extra {
		if l.extra == nil {
			l.extra = map[string]json.RawMessage{}
		}
		l.extra[k] = cloneRaw(v)
	}
	return nil
}

type execKey struct {
	Path         string                     `json:"path,omitempty"`
	Args         []string                   `json:"args,omitempty"`
	Envs         []string                   `json:"envs,omitempty"`
	ArgsRequired bool                       `json:"argsRequired,omitempty"`
	Extra        map[string]json.RawMessage `json:"extra,omitempty"`
}

type openKey struct {
	Path  string                     `json:"path"`
	Flags []string                   `json:"flags"`
	Extra map[string]json.RawMessage `json:"extra,omitempty"`
}

type endpointKey struct {
	Endpoint  string                     `json:"endpoint,omitempty"`
	Methods   []string                   `json:"methods,omitempty"`
	Internal  bool                       `json:"internal"`
	Direction string                     `json:"direction,omitempty"`
	Headers   json.RawMessage            `json:"headers,omitempty"`
	Extra     map[string]json.RawMessage `json:"extra,omitempty"`
}

type section struct {
	lists
	policies map[string]profile.RulePolicy
	imageID  string
	imageTag string
	seccomp  json.RawMessage
	stacks   []json.RawMessage
	subtract *lists
}

func newSection() *section {
	return &section{lists: *newLists(), policies: map[string]profile.RulePolicy{}}
}

func (s *section) add(b body) error {
	if err := s.lists.add(b.capabilities, b.execs, b.opens, b.syscalls, b.endpoints, b.ingress, b.egress, b.extra); err != nil {
		return err
	}
	if len(b.stacks) > 0 {
		var stacks []json.RawMessage
		if err := json.Unmarshal(b.stacks, &stacks); err != nil {
			return fmt.Errorf("identifiedCallStacks: %w", err)
		}
		for _, raw := range stacks {
			var v any
			dec := json.NewDecoder(bytes.NewReader(raw))
			dec.UseNumber()
			if err := dec.Decode(&v); err != nil {
				return fmt.Errorf("identifiedCallStacks: %w", err)
			}
			canonical, err := json.Marshal(v)
			if err != nil {
				return err
			}
			if key := "s" + string(canonical); !s.seen[key] {
				s.seen[key] = true
				s.stacks = append(s.stacks, canonical)
			}
		}
	}
	for id, p := range b.policies {
		s.policies[id] = clonePolicy(p)
	}
	if b.imageID != "" {
		s.imageID = b.imageID
	}
	if b.imageTag != "" {
		s.imageTag = b.imageTag
	}
	if !emptyJSON(b.seccomp) {
		s.seccomp = cloneRaw(b.seccomp)
	}
	if sub := b.subtract; sub != nil {
		if s.subtract == nil {
			s.subtract = newLists()
		}
		if err := s.subtract.add(sub.Capabilities, sub.Execs, sub.Opens, sub.Syscalls, sub.Endpoints, sub.Ingress, sub.Egress, sub.Extra); err != nil {
			return err
		}
	}
	return nil
}

func (s *section) body() (body, error) {
	b := body{
		capabilities: sortedKeys(s.capabilities),
		execs:        s.execs,
		opens:        s.opens,
		syscalls:     sortedKeys(s.syscalls),
		endpoints:    s.endpoints,
		ingress:      s.ingress.list(),
		egress:       s.egress.list(),
		seccomp:      s.seccomp,
		imageID:      s.imageID,
		imageTag:     s.imageTag,
		extra:        s.extra,
	}
	if len(s.policies) > 0 {
		b.policies = s.policies
	}
	if len(s.stacks) > 0 {
		raw, err := json.Marshal(s.stacks)
		if err != nil {
			return body{}, err
		}
		b.stacks = raw
	}
	if sub := s.subtract; sub != nil {
		b.subtract = &profile.Subtract{
			Capabilities: sortedKeys(sub.capabilities),
			Execs:        sub.execs,
			Opens:        sub.opens,
			Syscalls:     sortedKeys(sub.syscalls),
			Endpoints:    sub.endpoints,
			Ingress:      sub.ingress.list(),
			Egress:       sub.egress.list(),
			Extra:        sub.extra,
		}
	}
	return b, nil
}

type sectionGroup struct {
	order  []string
	byName map[string]*section
}

func newSectionGroup() *sectionGroup {
	return &sectionGroup{byName: map[string]*section{}}
}

func (g *sectionGroup) of(name string) *section {
	s, ok := g.byName[name]
	if !ok {
		s = newSection()
		g.byName[name] = s
		g.order = append(g.order, name)
	}
	return s
}

func (g *sectionGroup) containers() ([]profile.Container, error) {
	var out []profile.Container
	for _, name := range g.order {
		b, err := g.byName[name].body()
		if err != nil {
			return nil, err
		}
		out = append(out, profile.Container{
			Name:                 name,
			Capabilities:         b.capabilities,
			Execs:                b.execs,
			Opens:                b.opens,
			Syscalls:             b.syscalls,
			Endpoints:            b.endpoints,
			Ingress:              b.ingress,
			Egress:               b.egress,
			SeccompProfile:       b.seccomp,
			ImageID:              b.imageID,
			ImageTag:             b.imageTag,
			RulePolicies:         b.policies,
			IdentifiedCallStacks: b.stacks,
			Subtract:             b.subtract,
			Extra:                b.extra,
		})
	}
	return out, nil
}

func sortedKeys(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func emptyJSON(raw json.RawMessage) bool {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return true
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return false
	}
	return emptyValue(v)
}

func emptyValue(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return t == ""
	case bool:
		return !t
	case float64:
		return t == 0
	case map[string]any:
		for _, e := range t {
			if !emptyValue(e) {
				return false
			}
		}
		return true
	}
	return false
}
