package profile

import "encoding/json"

type Subject string

const (
	SubjectContainer Subject = "container"
	SubjectHost      Subject = "host"
)

type RuntimeProfile struct {
	APIVersion string                     `json:"apiVersion,omitempty"`
	Kind       string                     `json:"kind,omitempty"`
	Metadata   Metadata                   `json:"metadata,omitempty"`
	Subject    Subject                    `json:"subject,omitempty"`
	Spec       Spec                       `json:"spec,omitempty"`
	Status     json.RawMessage            `json:"status,omitempty"`
	Extra      map[string]json.RawMessage `json:"-"`
	seen       map[string]bool
}

func (v *RuntimeProfile) UnmarshalJSON(data []byte) error {
	r, err := readObject(data)
	if err != nil {
		return err
	}
	n := RuntimeProfile{}
	readField(r, "apiVersion", &n.APIVersion)
	readField(r, "kind", &n.Kind)
	readField(r, "metadata", &n.Metadata)
	readField(r, "subject", &n.Subject)
	readField(r, "spec", &n.Spec)
	readRaw(r, "status", &n.Status)
	n.seen, n.Extra, err = r.finish()
	if err != nil {
		return err
	}
	*v = n
	return nil
}

func (v RuntimeProfile) MarshalJSON() ([]byte, error) {
	w := writeObject(v.seen, v.Extra)
	writeField(w, "apiVersion", v.APIVersion, v.APIVersion == "")
	writeField(w, "kind", v.Kind, v.Kind == "")
	writeField(w, "metadata", v.Metadata, v.Metadata.isZero())
	writeField(w, "subject", v.Subject, v.Subject == "")
	writeField(w, "spec", v.Spec, v.Spec.isZero())
	writeRaw(w, "status", v.Status, "{}")
	return w.finish()
}

type Metadata struct {
	Name        string                     `json:"name,omitempty"`
	Namespace   string                     `json:"namespace,omitempty"`
	Labels      map[string]string          `json:"labels,omitempty"`
	Annotations map[string]string          `json:"annotations,omitempty"`
	Extra       map[string]json.RawMessage `json:"-"`
	seen        map[string]bool
}

func (v *Metadata) UnmarshalJSON(data []byte) error {
	r, err := readObject(data)
	if err != nil {
		return err
	}
	n := Metadata{}
	readField(r, "name", &n.Name)
	readField(r, "namespace", &n.Namespace)
	readField(r, "labels", &n.Labels)
	readField(r, "annotations", &n.Annotations)
	n.seen, n.Extra, err = r.finish()
	if err != nil {
		return err
	}
	*v = n
	return nil
}

func (v Metadata) MarshalJSON() ([]byte, error) {
	w := writeObject(v.seen, v.Extra)
	writeField(w, "name", v.Name, v.Name == "")
	writeField(w, "namespace", v.Namespace, v.Namespace == "")
	writeField(w, "labels", v.Labels, v.Labels == nil)
	writeField(w, "annotations", v.Annotations, v.Annotations == nil)
	return w.finish()
}

type Spec struct {
	Architectures        []string                   `json:"architectures,omitempty"`
	Capabilities         []string                   `json:"capabilities,omitempty"`
	Execs                []ExecCall                 `json:"execs,omitempty"`
	Opens                []OpenCall                 `json:"opens,omitempty"`
	Syscalls             []string                   `json:"syscalls,omitempty"`
	Endpoints            []Endpoint                 `json:"endpoints,omitempty"`
	Ingress              []NetworkNeighbor          `json:"ingress,omitempty"`
	Egress               []NetworkNeighbor          `json:"egress,omitempty"`
	SeccompProfile       json.RawMessage            `json:"seccompProfile,omitempty"`
	ImageID              string                     `json:"imageID,omitempty"`
	ImageTag             string                     `json:"imageTag,omitempty"`
	RulePolicies         map[string]RulePolicy      `json:"rulePolicies,omitempty"`
	IdentifiedCallStacks json.RawMessage            `json:"identifiedCallStacks,omitempty"`
	Subtract             *Subtract                  `json:"subtract,omitempty"`
	MatchLabels          map[string]string          `json:"matchLabels,omitempty"`
	MatchExpressions     []LabelSelectorRequirement `json:"matchExpressions,omitempty"`
	Containers           []Container                `json:"containers,omitempty"`
	InitContainers       []Container                `json:"initContainers,omitempty"`
	EphemeralContainers  []Container                `json:"ephemeralContainers,omitempty"`
	Extra                map[string]json.RawMessage `json:"-"`
	seen                 map[string]bool
}

func (v *Spec) UnmarshalJSON(data []byte) error {
	r, err := readObject(data)
	if err != nil {
		return err
	}
	n := Spec{}
	readField(r, "architectures", &n.Architectures)
	readField(r, "capabilities", &n.Capabilities)
	readField(r, "execs", &n.Execs)
	readField(r, "opens", &n.Opens)
	readField(r, "syscalls", &n.Syscalls)
	readField(r, "endpoints", &n.Endpoints)
	readField(r, "ingress", &n.Ingress)
	readField(r, "egress", &n.Egress)
	readRaw(r, "seccompProfile", &n.SeccompProfile)
	readField(r, "imageID", &n.ImageID)
	readField(r, "imageTag", &n.ImageTag)
	readField(r, "rulePolicies", &n.RulePolicies)
	readRaw(r, "identifiedCallStacks", &n.IdentifiedCallStacks)
	readField(r, "subtract", &n.Subtract)
	readField(r, "matchLabels", &n.MatchLabels)
	readField(r, "matchExpressions", &n.MatchExpressions)
	readField(r, "containers", &n.Containers)
	readField(r, "initContainers", &n.InitContainers)
	readField(r, "ephemeralContainers", &n.EphemeralContainers)
	n.seen, n.Extra, err = r.finish()
	if err != nil {
		return err
	}
	*v = n
	return nil
}

func (v Spec) MarshalJSON() ([]byte, error) {
	w := writeObject(v.seen, v.Extra)
	writeField(w, "architectures", v.Architectures, v.Architectures == nil)
	writeField(w, "capabilities", v.Capabilities, v.Capabilities == nil)
	writeField(w, "execs", v.Execs, v.Execs == nil)
	writeField(w, "opens", v.Opens, v.Opens == nil)
	writeField(w, "syscalls", v.Syscalls, v.Syscalls == nil)
	writeField(w, "endpoints", v.Endpoints, v.Endpoints == nil)
	writeField(w, "ingress", v.Ingress, v.Ingress == nil)
	writeField(w, "egress", v.Egress, v.Egress == nil)
	writeRaw(w, "seccompProfile", v.SeccompProfile, "{}")
	writeField(w, "imageID", v.ImageID, v.ImageID == "")
	writeField(w, "imageTag", v.ImageTag, v.ImageTag == "")
	writeField(w, "rulePolicies", v.RulePolicies, v.RulePolicies == nil)
	writeRaw(w, "identifiedCallStacks", v.IdentifiedCallStacks, "[]")
	writeField(w, "subtract", v.Subtract, v.Subtract == nil)
	writeField(w, "matchLabels", v.MatchLabels, v.MatchLabels == nil)
	writeField(w, "matchExpressions", v.MatchExpressions, v.MatchExpressions == nil)
	writeField(w, "containers", v.Containers, v.Containers == nil)
	writeField(w, "initContainers", v.InitContainers, v.InitContainers == nil)
	writeField(w, "ephemeralContainers", v.EphemeralContainers, v.EphemeralContainers == nil)
	return w.finish()
}

type Container struct {
	Name                 string                     `json:"name,omitempty"`
	Capabilities         []string                   `json:"capabilities,omitempty"`
	Execs                []ExecCall                 `json:"execs,omitempty"`
	Opens                []OpenCall                 `json:"opens,omitempty"`
	Syscalls             []string                   `json:"syscalls,omitempty"`
	Endpoints            []Endpoint                 `json:"endpoints,omitempty"`
	Ingress              []NetworkNeighbor          `json:"ingress,omitempty"`
	Egress               []NetworkNeighbor          `json:"egress,omitempty"`
	SeccompProfile       json.RawMessage            `json:"seccompProfile,omitempty"`
	ImageID              string                     `json:"imageID,omitempty"`
	ImageTag             string                     `json:"imageTag,omitempty"`
	RulePolicies         map[string]RulePolicy      `json:"rulePolicies,omitempty"`
	IdentifiedCallStacks json.RawMessage            `json:"identifiedCallStacks,omitempty"`
	Subtract             *Subtract                  `json:"subtract,omitempty"`
	Extra                map[string]json.RawMessage `json:"-"`
	seen                 map[string]bool
}

func (v *Container) UnmarshalJSON(data []byte) error {
	r, err := readObject(data)
	if err != nil {
		return err
	}
	n := Container{}
	readField(r, "name", &n.Name)
	readField(r, "capabilities", &n.Capabilities)
	readField(r, "execs", &n.Execs)
	readField(r, "opens", &n.Opens)
	readField(r, "syscalls", &n.Syscalls)
	readField(r, "endpoints", &n.Endpoints)
	readField(r, "ingress", &n.Ingress)
	readField(r, "egress", &n.Egress)
	readRaw(r, "seccompProfile", &n.SeccompProfile)
	readField(r, "imageID", &n.ImageID)
	readField(r, "imageTag", &n.ImageTag)
	readField(r, "rulePolicies", &n.RulePolicies)
	readRaw(r, "identifiedCallStacks", &n.IdentifiedCallStacks)
	readField(r, "subtract", &n.Subtract)
	n.seen, n.Extra, err = r.finish()
	if err != nil {
		return err
	}
	*v = n
	return nil
}

func (v Container) MarshalJSON() ([]byte, error) {
	w := writeObject(v.seen, v.Extra)
	writeField(w, "name", v.Name, v.Name == "")
	writeField(w, "capabilities", v.Capabilities, v.Capabilities == nil)
	writeField(w, "execs", v.Execs, v.Execs == nil)
	writeField(w, "opens", v.Opens, v.Opens == nil)
	writeField(w, "syscalls", v.Syscalls, v.Syscalls == nil)
	writeField(w, "endpoints", v.Endpoints, v.Endpoints == nil)
	writeField(w, "ingress", v.Ingress, v.Ingress == nil)
	writeField(w, "egress", v.Egress, v.Egress == nil)
	writeRaw(w, "seccompProfile", v.SeccompProfile, "{}")
	writeField(w, "imageID", v.ImageID, v.ImageID == "")
	writeField(w, "imageTag", v.ImageTag, v.ImageTag == "")
	writeField(w, "rulePolicies", v.RulePolicies, v.RulePolicies == nil)
	writeRaw(w, "identifiedCallStacks", v.IdentifiedCallStacks, "[]")
	writeField(w, "subtract", v.Subtract, v.Subtract == nil)
	return w.finish()
}

type Subtract struct {
	Capabilities []string                   `json:"capabilities,omitempty"`
	Execs        []ExecCall                 `json:"execs,omitempty"`
	Opens        []OpenCall                 `json:"opens,omitempty"`
	Syscalls     []string                   `json:"syscalls,omitempty"`
	Endpoints    []Endpoint                 `json:"endpoints,omitempty"`
	Ingress      []NetworkNeighbor          `json:"ingress,omitempty"`
	Egress       []NetworkNeighbor          `json:"egress,omitempty"`
	Extra        map[string]json.RawMessage `json:"-"`
	seen         map[string]bool
}

func (v *Subtract) UnmarshalJSON(data []byte) error {
	r, err := readObject(data)
	if err != nil {
		return err
	}
	n := Subtract{}
	readField(r, "capabilities", &n.Capabilities)
	readField(r, "execs", &n.Execs)
	readField(r, "opens", &n.Opens)
	readField(r, "syscalls", &n.Syscalls)
	readField(r, "endpoints", &n.Endpoints)
	readField(r, "ingress", &n.Ingress)
	readField(r, "egress", &n.Egress)
	n.seen, n.Extra, err = r.finish()
	if err != nil {
		return err
	}
	*v = n
	return nil
}

func (v Subtract) MarshalJSON() ([]byte, error) {
	w := writeObject(v.seen, v.Extra)
	writeField(w, "capabilities", v.Capabilities, v.Capabilities == nil)
	writeField(w, "execs", v.Execs, v.Execs == nil)
	writeField(w, "opens", v.Opens, v.Opens == nil)
	writeField(w, "syscalls", v.Syscalls, v.Syscalls == nil)
	writeField(w, "endpoints", v.Endpoints, v.Endpoints == nil)
	writeField(w, "ingress", v.Ingress, v.Ingress == nil)
	writeField(w, "egress", v.Egress, v.Egress == nil)
	return w.finish()
}

type ExecCall struct {
	Path         string                     `json:"path,omitempty"`
	Args         []string                   `json:"args,omitempty"`
	Envs         []string                   `json:"envs,omitempty"`
	ArgsRequired bool                       `json:"argsRequired,omitempty"`
	Extra        map[string]json.RawMessage `json:"-"`
	seen         map[string]bool
}

func (v *ExecCall) UnmarshalJSON(data []byte) error {
	r, err := readObject(data)
	if err != nil {
		return err
	}
	n := ExecCall{}
	readField(r, "path", &n.Path)
	readField(r, "args", &n.Args)
	readField(r, "envs", &n.Envs)
	readField(r, "argsRequired", &n.ArgsRequired)
	n.seen, n.Extra, err = r.finish()
	if err != nil {
		return err
	}
	*v = n
	return nil
}

func (v ExecCall) MarshalJSON() ([]byte, error) {
	w := writeObject(v.seen, v.Extra)
	writeField(w, "path", v.Path, v.Path == "")
	writeField(w, "args", v.Args, v.Args == nil)
	writeField(w, "envs", v.Envs, v.Envs == nil)
	writeField(w, "argsRequired", v.ArgsRequired, !v.ArgsRequired)
	return w.finish()
}

type OpenCall struct {
	Path  string                     `json:"path,omitempty"`
	Flags []string                   `json:"flags,omitempty"`
	Extra map[string]json.RawMessage `json:"-"`
	seen  map[string]bool
}

func (v *OpenCall) UnmarshalJSON(data []byte) error {
	r, err := readObject(data)
	if err != nil {
		return err
	}
	n := OpenCall{}
	readField(r, "path", &n.Path)
	readField(r, "flags", &n.Flags)
	n.seen, n.Extra, err = r.finish()
	if err != nil {
		return err
	}
	*v = n
	return nil
}

func (v OpenCall) MarshalJSON() ([]byte, error) {
	w := writeObject(v.seen, v.Extra)
	writeField(w, "path", v.Path, v.Path == "")
	writeField(w, "flags", v.Flags, v.Flags == nil)
	return w.finish()
}

type Endpoint struct {
	Endpoint  string                     `json:"endpoint,omitempty"`
	Methods   []string                   `json:"methods,omitempty"`
	Internal  bool                       `json:"internal,omitempty"`
	Direction string                     `json:"direction,omitempty"`
	Headers   json.RawMessage            `json:"headers,omitempty"`
	Extra     map[string]json.RawMessage `json:"-"`
	seen      map[string]bool
}

func (v *Endpoint) UnmarshalJSON(data []byte) error {
	r, err := readObject(data)
	if err != nil {
		return err
	}
	n := Endpoint{}
	readField(r, "endpoint", &n.Endpoint)
	readField(r, "methods", &n.Methods)
	readField(r, "internal", &n.Internal)
	readField(r, "direction", &n.Direction)
	readRaw(r, "headers", &n.Headers)
	n.seen, n.Extra, err = r.finish()
	if err != nil {
		return err
	}
	*v = n
	return nil
}

func (v Endpoint) MarshalJSON() ([]byte, error) {
	w := writeObject(v.seen, v.Extra)
	writeField(w, "endpoint", v.Endpoint, v.Endpoint == "")
	writeField(w, "methods", v.Methods, v.Methods == nil)
	writeField(w, "internal", v.Internal, !v.Internal)
	writeField(w, "direction", v.Direction, v.Direction == "")
	writeRaw(w, "headers", v.Headers, "{}")
	return w.finish()
}

type RulePolicy struct {
	ProcessAllowed   []string                   `json:"processAllowed,omitempty"`
	ContainerAllowed bool                       `json:"containerAllowed,omitempty"`
	Extra            map[string]json.RawMessage `json:"-"`
	seen             map[string]bool
}

func (v *RulePolicy) UnmarshalJSON(data []byte) error {
	r, err := readObject(data)
	if err != nil {
		return err
	}
	n := RulePolicy{}
	readField(r, "processAllowed", &n.ProcessAllowed)
	readField(r, "containerAllowed", &n.ContainerAllowed)
	n.seen, n.Extra, err = r.finish()
	if err != nil {
		return err
	}
	*v = n
	return nil
}

func (v RulePolicy) MarshalJSON() ([]byte, error) {
	w := writeObject(v.seen, v.Extra)
	writeField(w, "processAllowed", v.ProcessAllowed, v.ProcessAllowed == nil)
	writeField(w, "containerAllowed", v.ContainerAllowed, !v.ContainerAllowed)
	return w.finish()
}

type NetworkNeighbor struct {
	Identifier          string                     `json:"identifier,omitempty"`
	Type                string                     `json:"type,omitempty"`
	DNS                 string                     `json:"dns,omitempty"`
	DNSNames            []string                   `json:"dnsNames,omitempty"`
	Ports               []NetworkPort              `json:"ports,omitempty"`
	PodSelector         *LabelSelector             `json:"podSelector,omitempty"`
	NamespaceSelector   *LabelSelector             `json:"namespaceSelector,omitempty"`
	IPAddress           string                     `json:"ipAddress,omitempty"`
	IPAddresses         []string                   `json:"ipAddresses,omitempty"`
	ServiceRefNamespace string                     `json:"serviceRefNamespace,omitempty"`
	ServiceRefName      string                     `json:"serviceRefName,omitempty"`
	ServiceSelector     *LabelSelector             `json:"serviceSelector,omitempty"`
	Entity              string                     `json:"entity,omitempty"`
	Extra               map[string]json.RawMessage `json:"-"`
	seen                map[string]bool
}

func (v *NetworkNeighbor) UnmarshalJSON(data []byte) error {
	r, err := readObject(data)
	if err != nil {
		return err
	}
	n := NetworkNeighbor{}
	readField(r, "identifier", &n.Identifier)
	readField(r, "type", &n.Type)
	readField(r, "dns", &n.DNS)
	readField(r, "dnsNames", &n.DNSNames)
	readField(r, "ports", &n.Ports)
	readField(r, "podSelector", &n.PodSelector)
	readField(r, "namespaceSelector", &n.NamespaceSelector)
	readField(r, "ipAddress", &n.IPAddress)
	readField(r, "ipAddresses", &n.IPAddresses)
	readField(r, "serviceRefNamespace", &n.ServiceRefNamespace)
	readField(r, "serviceRefName", &n.ServiceRefName)
	readField(r, "serviceSelector", &n.ServiceSelector)
	readField(r, "entity", &n.Entity)
	n.seen, n.Extra, err = r.finish()
	if err != nil {
		return err
	}
	*v = n
	return nil
}

func (v NetworkNeighbor) MarshalJSON() ([]byte, error) {
	w := writeObject(v.seen, v.Extra)
	writeField(w, "identifier", v.Identifier, v.Identifier == "")
	writeField(w, "type", v.Type, v.Type == "")
	writeField(w, "dns", v.DNS, v.DNS == "")
	writeField(w, "dnsNames", v.DNSNames, v.DNSNames == nil)
	writeField(w, "ports", v.Ports, v.Ports == nil)
	writeField(w, "podSelector", v.PodSelector, v.PodSelector == nil)
	writeField(w, "namespaceSelector", v.NamespaceSelector, v.NamespaceSelector == nil)
	writeField(w, "ipAddress", v.IPAddress, v.IPAddress == "")
	writeField(w, "ipAddresses", v.IPAddresses, v.IPAddresses == nil)
	writeField(w, "serviceRefNamespace", v.ServiceRefNamespace, v.ServiceRefNamespace == "")
	writeField(w, "serviceRefName", v.ServiceRefName, v.ServiceRefName == "")
	writeField(w, "serviceSelector", v.ServiceSelector, v.ServiceSelector == nil)
	writeField(w, "entity", v.Entity, v.Entity == "")
	return w.finish()
}

type NetworkPort struct {
	Name     string                     `json:"name,omitempty"`
	Protocol string                     `json:"protocol,omitempty"`
	Port     *int32                     `json:"port,omitempty"`
	Extra    map[string]json.RawMessage `json:"-"`
	seen     map[string]bool
}

func (v *NetworkPort) UnmarshalJSON(data []byte) error {
	r, err := readObject(data)
	if err != nil {
		return err
	}
	n := NetworkPort{}
	readField(r, "name", &n.Name)
	readField(r, "protocol", &n.Protocol)
	readField(r, "port", &n.Port)
	n.seen, n.Extra, err = r.finish()
	if err != nil {
		return err
	}
	*v = n
	return nil
}

func (v NetworkPort) MarshalJSON() ([]byte, error) {
	w := writeObject(v.seen, v.Extra)
	writeField(w, "name", v.Name, v.Name == "")
	writeField(w, "protocol", v.Protocol, v.Protocol == "")
	writeField(w, "port", v.Port, v.Port == nil)
	return w.finish()
}

type LabelSelector struct {
	MatchLabels      map[string]string          `json:"matchLabels,omitempty"`
	MatchExpressions []LabelSelectorRequirement `json:"matchExpressions,omitempty"`
	Extra            map[string]json.RawMessage `json:"-"`
	seen             map[string]bool
}

func (v *LabelSelector) UnmarshalJSON(data []byte) error {
	r, err := readObject(data)
	if err != nil {
		return err
	}
	n := LabelSelector{}
	readField(r, "matchLabels", &n.MatchLabels)
	readField(r, "matchExpressions", &n.MatchExpressions)
	n.seen, n.Extra, err = r.finish()
	if err != nil {
		return err
	}
	*v = n
	return nil
}

func (v LabelSelector) MarshalJSON() ([]byte, error) {
	w := writeObject(v.seen, v.Extra)
	writeField(w, "matchLabels", v.MatchLabels, v.MatchLabels == nil)
	writeField(w, "matchExpressions", v.MatchExpressions, v.MatchExpressions == nil)
	return w.finish()
}

type LabelSelectorRequirement struct {
	Key      string                     `json:"key,omitempty"`
	Operator string                     `json:"operator,omitempty"`
	Values   []string                   `json:"values,omitempty"`
	Extra    map[string]json.RawMessage `json:"-"`
	seen     map[string]bool
}

func (v *LabelSelectorRequirement) UnmarshalJSON(data []byte) error {
	r, err := readObject(data)
	if err != nil {
		return err
	}
	n := LabelSelectorRequirement{}
	readField(r, "key", &n.Key)
	readField(r, "operator", &n.Operator)
	readField(r, "values", &n.Values)
	n.seen, n.Extra, err = r.finish()
	if err != nil {
		return err
	}
	*v = n
	return nil
}

func (v LabelSelectorRequirement) MarshalJSON() ([]byte, error) {
	w := writeObject(v.seen, v.Extra)
	writeField(w, "key", v.Key, v.Key == "")
	writeField(w, "operator", v.Operator, v.Operator == "")
	writeField(w, "values", v.Values, v.Values == nil)
	return w.finish()
}

func (v Metadata) isZero() bool {
	b, err := v.MarshalJSON()
	return err == nil && string(b) == "{}"
}

func (v Spec) isZero() bool {
	b, err := v.MarshalJSON()
	return err == nil && string(b) == "{}"
}
