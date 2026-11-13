package compose

import (
	"slices"
	"strconv"
	"strings"

	"github.com/k8sstormcenter/sbob-go/match"
	"github.com/k8sstormcenter/sbob-go/profile"
)

const (
	HostPeer         = "host-entity"
	HostPeerResolved = "host-entity-resolved"
	HostPeerClaimed  = "host-entity-claimed"
)

type Field struct {
	Values   map[string]struct{}
	Patterns []string
}

type Peer struct {
	PodSelector       *profile.LabelSelector
	NamespaceSelector *profile.LabelSelector
	Ports             map[string]struct{}
}

type AddrPorts struct {
	Addrs []string
	Ports map[string]struct{}
}

type DomainPorts struct {
	Domains []string
	Addrs   []string
	Ports   map[string]struct{}
}

type Sets struct {
	Opens            Field
	Execs            Field
	Endpoints        Field
	Capabilities     Field
	Syscalls         Field
	EgressDomains    Field
	EgressAddresses  Field
	IngressDomains   Field
	IngressAddresses Field

	ExecArgs  map[string][][]string
	OpenFlags map[string][]string

	EgressPeers  []Peer
	IngressPeers []Peer

	EgressAddrPorts   []AddrPorts
	IngressAddrPorts  []AddrPorts
	EgressDomainPorts []DomainPorts

	EgressHost        []AddrPorts
	IngressHost       []AddrPorts
	EgressHostClaimed []AddrPorts
}

type Projection struct {
	Sets
	RulePolicies map[string]profile.RulePolicy
	Subtract     *Sets
}

func PortKey(protocol string, port int32) string {
	return strings.ToUpper(protocol) + "-" + strconv.Itoa(int(port))
}

func Project(spec *profile.Spec) *Projection {
	p := &Projection{}
	if spec == nil {
		return p
	}
	if len(spec.RulePolicies) > 0 {
		p.RulePolicies = make(map[string]profile.RulePolicy, len(spec.RulePolicies))
		for id, policy := range spec.RulePolicies {
			p.RulePolicies[id] = clonePolicy(policy)
		}
	}
	p.Sets = projectSets(spec.Capabilities, spec.Execs, spec.Opens, spec.Syscalls, spec.Endpoints, spec.Ingress, spec.Egress)
	_, _, claimed := splitHostPeers(spec.Egress)
	p.EgressHostClaimed = addrPorts(claimed)
	if sub := spec.Subtract; sub != nil {
		s := projectSets(sub.Capabilities, sub.Execs, sub.Opens, sub.Syscalls, sub.Endpoints, sub.Ingress, sub.Egress)
		p.Subtract = &s
	}
	return p
}

func projectSets(capabilities []string, execs []profile.ExecCall, opens []profile.OpenCall, syscalls []string, endpoints []profile.Endpoint, ingressAll, egressAll []profile.NetworkNeighbor) Sets {
	var s Sets

	openPaths := make([]string, len(opens))
	for i := range opens {
		openPaths[i] = opens[i].Path
	}
	s.Opens = field(openPaths, true)
	s.OpenFlags = openFlags(opens)

	execPaths := make([]string, len(execs))
	for i := range execs {
		execPaths[i] = execs[i].Path
	}
	s.Execs = field(execPaths, true)
	s.ExecArgs = execArgs(execs)

	endpointPaths := make([]string, len(endpoints))
	for i := range endpoints {
		endpointPaths[i] = endpoints[i].Endpoint
	}
	s.Endpoints = field(endpointPaths, true)

	s.Capabilities = field(capabilities, false)
	s.Syscalls = field(syscalls, false)

	egress, egressHost, _ := splitHostPeers(egressAll)
	ingress, ingressHost, _ := splitHostPeers(ingressAll)

	s.EgressDomains = field(domains(egress), false)
	s.EgressAddresses = field(addresses(egress), false)
	s.IngressDomains = field(domains(ingress), false)
	s.IngressAddresses = field(addresses(ingress), false)

	s.EgressPeers = peers(egressAll)
	s.IngressPeers = peers(ingressAll)

	s.EgressAddrPorts = addrPorts(egress)
	s.IngressAddrPorts = addrPorts(ingress)
	s.EgressDomainPorts = domainPorts(egress)

	s.EgressHost = addrPorts(egressHost)
	s.IngressHost = addrPorts(ingressHost)
	return s
}

func splitHostPeers(neighbors []profile.NetworkNeighbor) (declared, host, claimed []profile.NetworkNeighbor) {
	for i := range neighbors {
		switch neighbors[i].Identifier {
		case HostPeerClaimed:
			claimed = append(claimed, neighbors[i])
		case HostPeer, HostPeerResolved:
			host = append(host, neighbors[i])
		default:
			declared = append(declared, neighbors[i])
		}
	}
	return declared, host, claimed
}

func field(entries []string, pathSurface bool) Field {
	var f Field
	var seen map[string]struct{}
	for _, e := range entries {
		if pathSurface && match.HasWildcard(e) {
			if _, dup := seen[e]; !dup {
				if seen == nil {
					seen = map[string]struct{}{}
				}
				seen[e] = struct{}{}
				f.Patterns = append(f.Patterns, e)
			}
			continue
		}
		if f.Values == nil {
			f.Values = map[string]struct{}{}
		}
		f.Values[e] = struct{}{}
	}
	slices.Sort(f.Patterns)
	return f
}

func openFlags(opens []profile.OpenCall) map[string][]string {
	if len(opens) == 0 {
		return nil
	}
	m := make(map[string][]string)
	for i := range opens {
		o := &opens[i]
		cur, present := m[o.Path]
		if !present {
			cur = []string{}
		}
		for _, f := range o.Flags {
			if !slices.Contains(cur, f) {
				cur = append(cur, f)
			}
		}
		slices.Sort(cur)
		m[o.Path] = cur
	}
	return m
}

func execArgs(execs []profile.ExecCall) map[string][][]string {
	if len(execs) == 0 {
		return nil
	}
	m := make(map[string][][]string, len(execs))
	for i := range execs {
		e := &execs[i]
		args := make([]string, len(e.Args))
		copy(args, e.Args)
		m[e.Path] = append(m[e.Path], args)
	}
	return m
}

func domains(neighbors []profile.NetworkNeighbor) []string {
	var out []string
	for i := range neighbors {
		out = append(out, neighborDomains(&neighbors[i])...)
	}
	return out
}

func addresses(neighbors []profile.NetworkNeighbor) []string {
	var out []string
	for i := range neighbors {
		out = append(out, neighborAddrs(&neighbors[i])...)
	}
	return out
}

func neighborDomains(n *profile.NetworkNeighbor) []string {
	var out []string
	if n.DNS != "" {
		out = append(out, n.DNS)
	}
	return append(out, n.DNSNames...)
}

func neighborAddrs(n *profile.NetworkNeighbor) []string {
	var out []string
	if n.IPAddress != "" {
		out = append(out, n.IPAddress)
	}
	return append(out, n.IPAddresses...)
}

func ports(list []profile.NetworkPort) map[string]struct{} {
	if len(list) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(list))
	for i := range list {
		if list[i].Port == nil {
			continue
		}
		out[PortKey(list[i].Protocol, *list[i].Port)] = struct{}{}
	}
	return out
}

func peers(neighbors []profile.NetworkNeighbor) []Peer {
	var out []Peer
	for i := range neighbors {
		n := &neighbors[i]
		if n.PodSelector == nil {
			continue
		}
		out = append(out, Peer{
			PodSelector:       cloneSelector(n.PodSelector),
			NamespaceSelector: cloneSelector(n.NamespaceSelector),
			Ports:             ports(n.Ports),
		})
	}
	return out
}

func addrPorts(neighbors []profile.NetworkNeighbor) []AddrPorts {
	var out []AddrPorts
	for i := range neighbors {
		n := &neighbors[i]
		addrs := neighborAddrs(n)
		if len(addrs) == 0 {
			continue
		}
		out = append(out, AddrPorts{Addrs: addrs, Ports: ports(n.Ports)})
	}
	return out
}

func domainPorts(neighbors []profile.NetworkNeighbor) []DomainPorts {
	var out []DomainPorts
	for i := range neighbors {
		n := &neighbors[i]
		names := neighborDomains(n)
		if len(names) == 0 {
			continue
		}
		out = append(out, DomainPorts{Domains: names, Addrs: neighborAddrs(n), Ports: ports(n.Ports)})
	}
	return out
}
