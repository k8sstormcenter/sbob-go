package profile

import "encoding/json"

func cloneValues[T any](s []T) []T {
	if s == nil {
		return nil
	}
	out := make([]T, len(s))
	copy(out, s)
	return out
}

func cloneEach[T interface{ Clone() T }](s []T) []T {
	if s == nil {
		return nil
	}
	out := make([]T, len(s))
	for i := range s {
		out[i] = s[i].Clone()
	}
	return out
}

func cloneMap[V any](m map[string]V) map[string]V {
	if m == nil {
		return nil
	}
	out := make(map[string]V, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func cloneRaw(r json.RawMessage) json.RawMessage {
	if r == nil {
		return nil
	}
	out := make(json.RawMessage, len(r))
	copy(out, r)
	return out
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

func (v *RuntimeProfile) Clone() *RuntimeProfile {
	if v == nil {
		return nil
	}
	out := *v
	out.Metadata = v.Metadata.Clone()
	out.Spec = v.Spec.Clone()
	out.Status = cloneRaw(v.Status)
	out.Extra = cloneExtra(v.Extra)
	return &out
}

func (v Metadata) Clone() Metadata {
	v.Labels = cloneMap(v.Labels)
	v.Annotations = cloneMap(v.Annotations)
	v.Extra = cloneExtra(v.Extra)
	return v
}

func (v Spec) Clone() Spec {
	v.Architectures = cloneValues(v.Architectures)
	v.Capabilities = cloneValues(v.Capabilities)
	v.Execs = cloneEach(v.Execs)
	v.Opens = cloneEach(v.Opens)
	v.Syscalls = cloneValues(v.Syscalls)
	v.Endpoints = cloneEach(v.Endpoints)
	v.Ingress = cloneEach(v.Ingress)
	v.Egress = cloneEach(v.Egress)
	v.SeccompProfile = cloneRaw(v.SeccompProfile)
	v.RulePolicies = clonePolicies(v.RulePolicies)
	v.IdentifiedCallStacks = cloneRaw(v.IdentifiedCallStacks)
	v.Subtract = v.Subtract.Clone()
	v.MatchLabels = cloneMap(v.MatchLabels)
	v.MatchExpressions = cloneEach(v.MatchExpressions)
	v.Containers = cloneEach(v.Containers)
	v.InitContainers = cloneEach(v.InitContainers)
	v.EphemeralContainers = cloneEach(v.EphemeralContainers)
	v.Extra = cloneExtra(v.Extra)
	return v
}

func (v Container) Clone() Container {
	v.Capabilities = cloneValues(v.Capabilities)
	v.Execs = cloneEach(v.Execs)
	v.Opens = cloneEach(v.Opens)
	v.Syscalls = cloneValues(v.Syscalls)
	v.Endpoints = cloneEach(v.Endpoints)
	v.Ingress = cloneEach(v.Ingress)
	v.Egress = cloneEach(v.Egress)
	v.SeccompProfile = cloneRaw(v.SeccompProfile)
	v.RulePolicies = clonePolicies(v.RulePolicies)
	v.IdentifiedCallStacks = cloneRaw(v.IdentifiedCallStacks)
	v.Subtract = v.Subtract.Clone()
	v.Extra = cloneExtra(v.Extra)
	return v
}

func (v *Subtract) Clone() *Subtract {
	if v == nil {
		return nil
	}
	out := *v
	out.Capabilities = cloneValues(v.Capabilities)
	out.Execs = cloneEach(v.Execs)
	out.Opens = cloneEach(v.Opens)
	out.Syscalls = cloneValues(v.Syscalls)
	out.Endpoints = cloneEach(v.Endpoints)
	out.Ingress = cloneEach(v.Ingress)
	out.Egress = cloneEach(v.Egress)
	out.Extra = cloneExtra(v.Extra)
	return &out
}

func (v ExecCall) Clone() ExecCall {
	v.Args = cloneValues(v.Args)
	v.Envs = cloneValues(v.Envs)
	v.Extra = cloneExtra(v.Extra)
	return v
}

func (v OpenCall) Clone() OpenCall {
	v.Flags = cloneValues(v.Flags)
	v.Extra = cloneExtra(v.Extra)
	return v
}

func (v Endpoint) Clone() Endpoint {
	v.Methods = cloneValues(v.Methods)
	v.Headers = cloneRaw(v.Headers)
	v.Extra = cloneExtra(v.Extra)
	return v
}

func (v RulePolicy) Clone() RulePolicy {
	v.ProcessAllowed = cloneValues(v.ProcessAllowed)
	v.Extra = cloneExtra(v.Extra)
	return v
}

func clonePolicies(m map[string]RulePolicy) map[string]RulePolicy {
	if m == nil {
		return nil
	}
	out := make(map[string]RulePolicy, len(m))
	for k, v := range m {
		out[k] = v.Clone()
	}
	return out
}

func (v NetworkNeighbor) Clone() NetworkNeighbor {
	v.DNSNames = cloneValues(v.DNSNames)
	v.Ports = cloneEach(v.Ports)
	v.PodSelector = v.PodSelector.Clone()
	v.NamespaceSelector = v.NamespaceSelector.Clone()
	v.IPAddresses = cloneValues(v.IPAddresses)
	v.ServiceSelector = v.ServiceSelector.Clone()
	v.Extra = cloneExtra(v.Extra)
	return v
}

func (v NetworkPort) Clone() NetworkPort {
	if v.Port != nil {
		port := *v.Port
		v.Port = &port
	}
	v.Extra = cloneExtra(v.Extra)
	return v
}

func (v *LabelSelector) Clone() *LabelSelector {
	if v == nil {
		return nil
	}
	out := *v
	out.MatchLabels = cloneMap(v.MatchLabels)
	out.MatchExpressions = cloneEach(v.MatchExpressions)
	out.Extra = cloneExtra(v.Extra)
	return &out
}

func (v LabelSelectorRequirement) Clone() LabelSelectorRequirement {
	v.Values = cloneValues(v.Values)
	v.Extra = cloneExtra(v.Extra)
	return v
}
