package compose

import (
	"encoding/json"

	"github.com/k8sstormcenter/sbob-go/profile"
)

func cloneStrings(s []string) []string {
	if s == nil {
		return nil
	}
	return append([]string{}, s...)
}

func cloneRaw(r json.RawMessage) json.RawMessage {
	if r == nil {
		return nil
	}
	return append(json.RawMessage{}, r...)
}

func cloneExtra(m map[string]json.RawMessage) map[string]json.RawMessage {
	if m == nil {
		return nil
	}
	out := make(map[string]json.RawMessage, len(m))
	for k, v := range m {
		out[k] = cloneRaw(v)
	}
	return out
}

func cloneLabels(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func cloneRequirements(list []profile.LabelSelectorRequirement) []profile.LabelSelectorRequirement {
	if list == nil {
		return nil
	}
	out := make([]profile.LabelSelectorRequirement, len(list))
	for i, r := range list {
		r.Values = cloneStrings(r.Values)
		r.Extra = cloneExtra(r.Extra)
		out[i] = r
	}
	return out
}

func cloneSelector(s *profile.LabelSelector) *profile.LabelSelector {
	if s == nil {
		return nil
	}
	out := *s
	out.MatchLabels = cloneLabels(s.MatchLabels)
	out.MatchExpressions = cloneRequirements(s.MatchExpressions)
	out.Extra = cloneExtra(s.Extra)
	return &out
}

func cloneNeighbor(n profile.NetworkNeighbor) profile.NetworkNeighbor {
	n.DNSNames = cloneStrings(n.DNSNames)
	n.IPAddresses = cloneStrings(n.IPAddresses)
	if n.Ports != nil {
		ports := make([]profile.NetworkPort, len(n.Ports))
		for i, p := range n.Ports {
			if p.Port != nil {
				v := *p.Port
				p.Port = &v
			}
			p.Extra = cloneExtra(p.Extra)
			ports[i] = p
		}
		n.Ports = ports
	}
	n.PodSelector = cloneSelector(n.PodSelector)
	n.NamespaceSelector = cloneSelector(n.NamespaceSelector)
	n.ServiceSelector = cloneSelector(n.ServiceSelector)
	n.Extra = cloneExtra(n.Extra)
	return n
}

func clonePolicy(p profile.RulePolicy) profile.RulePolicy {
	p.ProcessAllowed = cloneStrings(p.ProcessAllowed)
	p.Extra = cloneExtra(p.Extra)
	return p
}
