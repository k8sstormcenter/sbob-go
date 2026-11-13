package compose

import "github.com/k8sstormcenter/sbob-go/profile"

type ContainerKind string

const (
	AnyContainer       ContainerKind = ""
	Container          ContainerKind = "container"
	InitContainer      ContainerKind = "initContainer"
	EphemeralContainer ContainerKind = "ephemeralContainer"
)

func Grouped(doc *profile.RuntimeProfile) bool {
	if doc == nil {
		return false
	}
	s := &doc.Spec
	return len(s.Containers) > 0 || len(s.InitContainers) > 0 || len(s.EphemeralContainers) > 0
}

func Section(doc *profile.RuntimeProfile, containerName string, kind ContainerKind) (*profile.Spec, bool) {
	if doc == nil {
		return nil, false
	}
	if !Grouped(doc) {
		flat := doc.Spec
		return &flat, true
	}
	s := &doc.Spec
	var groups [][]profile.Container
	switch kind {
	case AnyContainer:
		groups = [][]profile.Container{s.Containers, s.InitContainers, s.EphemeralContainers}
	case Container:
		groups = [][]profile.Container{s.Containers}
	case InitContainer:
		groups = [][]profile.Container{s.InitContainers}
	case EphemeralContainer:
		groups = [][]profile.Container{s.EphemeralContainers}
	default:
		return nil, false
	}
	for _, group := range groups {
		for i := range group {
			c := &group[i]
			if c.Name != containerName {
				continue
			}
			return &profile.Spec{
				Architectures:        s.Architectures,
				MatchLabels:          s.MatchLabels,
				MatchExpressions:     s.MatchExpressions,
				Capabilities:         c.Capabilities,
				Execs:                c.Execs,
				Opens:                c.Opens,
				Syscalls:             c.Syscalls,
				Endpoints:            c.Endpoints,
				Ingress:              c.Ingress,
				Egress:               c.Egress,
				SeccompProfile:       c.SeccompProfile,
				ImageID:              c.ImageID,
				ImageTag:             c.ImageTag,
				RulePolicies:         c.RulePolicies,
				IdentifiedCallStacks: c.IdentifiedCallStacks,
				Subtract:             c.Subtract,
				Extra:                c.Extra,
			}, true
		}
	}
	return nil, false
}
