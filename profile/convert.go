package profile

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/k8sstormcenter/sbob-go/match"
)

const (
	Kind       = "RuntimeProfile"
	Group      = "profiles.sbob"
	Version    = "v1beta1"
	APIVersion = Group + "/" + Version

	LegacyKind       = "ContainerProfile"
	LegacyGroup      = "spdx.softwarecomposition.kubescape.io"
	LegacyAPIVersion = LegacyGroup + "/" + Version

	SubjectTypeLabel       = "sbob/subject-type"
	LegacySubjectTypeLabel = "kubescape.io/sbom-type"
	LegacySubjectTypeNode  = "node"
)

type Invalid struct {
	Field  string
	Reason string
}

func (e *Invalid) Error() string {
	return e.Field + ": " + e.Reason
}

type DeadEntry struct {
	Field string
	Value string
}

func (e *DeadEntry) Error() string {
	return fmt.Sprintf("%s: entry %q can never match", e.Field, e.Value)
}

func FromLegacy(doc []byte) (*RuntimeProfile, error) {
	p := &RuntimeProfile{}
	if err := json.Unmarshal(doc, p); err != nil {
		return nil, err
	}
	if p.Kind != LegacyKind {
		return nil, fmt.Errorf("kind %q is not %s", p.Kind, LegacyKind)
	}
	version, ok := strings.CutPrefix(p.APIVersion, LegacyGroup+"/")
	if !ok || version == "" {
		return nil, fmt.Errorf("apiVersion %q is not in group %s", p.APIVersion, LegacyGroup)
	}
	p.Kind = Kind
	p.APIVersion = Group + "/" + version
	p.Subject = subjectFromLabels(p.Metadata.Labels)
	return p, nil
}

func subjectFromLabels(labels map[string]string) Subject {
	if labels[LegacySubjectTypeLabel] == LegacySubjectTypeNode {
		return SubjectHost
	}
	switch labels[SubjectTypeLabel] {
	case LegacySubjectTypeNode, string(SubjectHost):
		return SubjectHost
	}
	return SubjectContainer
}

func ToLegacy(p *RuntimeProfile) ([]byte, error) {
	if p == nil {
		return nil, fmt.Errorf("nil profile")
	}
	if p.Kind != Kind {
		return nil, fmt.Errorf("kind %q is not %s", p.Kind, Kind)
	}
	version, ok := strings.CutPrefix(p.APIVersion, Group+"/")
	if !ok || version == "" {
		return nil, fmt.Errorf("apiVersion %q is not in group %s", p.APIVersion, Group)
	}
	out := *p
	out.Kind = LegacyKind
	out.APIVersion = LegacyGroup + "/" + version
	switch p.Subject {
	case SubjectContainer:
	case SubjectHost:
		if out.Metadata.Labels[LegacySubjectTypeLabel] != LegacySubjectTypeNode {
			labels := make(map[string]string, len(out.Metadata.Labels)+1)
			for k, v := range out.Metadata.Labels {
				labels[k] = v
			}
			labels[LegacySubjectTypeLabel] = LegacySubjectTypeNode
			out.Metadata.Labels = labels
		}
	default:
		return nil, fmt.Errorf("unknown subject %q", p.Subject)
	}
	out.Subject = ""
	if _, had := p.seen["subject"]; had {
		seen := make(map[string]bool, len(p.seen))
		for k, v := range p.seen {
			if k != "subject" {
				seen[k] = v
			}
		}
		out.seen = seen
	}
	return json.Marshal(out)
}

func Validate(p *RuntimeProfile) []error {
	if p == nil {
		return []error{&Invalid{Field: "document", Reason: "nil profile"}}
	}
	var errs []error
	if p.Kind != Kind {
		errs = append(errs, &Invalid{Field: "kind", Reason: fmt.Sprintf("%q is not %s", p.Kind, Kind)})
	}
	if p.Subject != SubjectContainer && p.Subject != SubjectHost {
		errs = append(errs, &Invalid{Field: "subject", Reason: fmt.Sprintf("unknown subject %q", p.Subject)})
	}
	if p.Metadata.Name == "" {
		errs = append(errs, &Invalid{Field: "metadata.name", Reason: "empty name"})
	}
	s := &p.Spec
	errs = validateSections(errs, "spec", s.Execs, s.Opens, s.Ingress, s.Egress, s.Subtract)
	errs = validateContainers(errs, "spec.containers", s.Containers)
	errs = validateContainers(errs, "spec.initContainers", s.InitContainers)
	errs = validateContainers(errs, "spec.ephemeralContainers", s.EphemeralContainers)
	return errs
}

func validateContainers(errs []error, field string, containers []Container) []error {
	for i := range containers {
		c := &containers[i]
		at := fmt.Sprintf("%s[%d]", field, i)
		if c.Name == "" {
			errs = append(errs, &Invalid{Field: at + ".name", Reason: "empty name"})
		}
		errs = validateSections(errs, at, c.Execs, c.Opens, c.Ingress, c.Egress, c.Subtract)
	}
	return errs
}

func validateSections(errs []error, field string, execs []ExecCall, opens []OpenCall, ingress, egress []NetworkNeighbor, subtract *Subtract) []error {
	for i := range execs {
		errs = validatePath(errs, fmt.Sprintf("%s.execs[%d].path", field, i), execs[i].Path)
	}
	for i := range opens {
		errs = validatePath(errs, fmt.Sprintf("%s.opens[%d].path", field, i), opens[i].Path)
	}
	errs = validateNeighbors(errs, field+".ingress", ingress)
	errs = validateNeighbors(errs, field+".egress", egress)
	if subtract != nil {
		errs = validateSections(errs, field+".subtract", subtract.Execs, subtract.Opens, subtract.Ingress, subtract.Egress, nil)
	}
	return errs
}

func validatePath(errs []error, field, path string) []error {
	if path == "" {
		return append(errs, &Invalid{Field: field, Reason: "empty path"})
	}
	if match.UnmatchablePath(path) {
		return append(errs, &DeadEntry{Field: field, Value: path})
	}
	return errs
}

func validateNeighbors(errs []error, field string, neighbors []NetworkNeighbor) []error {
	for i := range neighbors {
		n := &neighbors[i]
		at := fmt.Sprintf("%s[%d]", field, i)
		if n.IPAddress != "" {
			if err := match.ValidateIPEntry(n.IPAddress); err != nil {
				errs = append(errs, &Invalid{Field: at + ".ipAddress", Reason: err.Error()})
			}
		}
		for j, ip := range n.IPAddresses {
			if err := match.ValidateIPEntry(ip); err != nil {
				errs = append(errs, &Invalid{Field: fmt.Sprintf("%s.ipAddresses[%d]", at, j), Reason: err.Error()})
			}
		}
		if n.DNS != "" {
			if err := match.ValidateDNSEntry(n.DNS); err != nil {
				errs = append(errs, &Invalid{Field: at + ".dns", Reason: err.Error()})
			}
		}
		for j, name := range n.DNSNames {
			if err := match.ValidateDNSEntry(name); err != nil {
				errs = append(errs, &Invalid{Field: fmt.Sprintf("%s.dnsNames[%d]", at, j), Reason: err.Error()})
			}
		}
	}
	return errs
}
