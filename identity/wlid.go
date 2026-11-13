package identity

import (
	"errors"
	"fmt"
	"strings"
)

const (
	wlidScheme      = "wlid://"
	sidScheme       = "sid://"
	clusterPrefix   = "cluster-"
	namespacePrefix = "namespace-"
)

var knownKinds = []string{"ComponentStatus", "ConfigMap", "ControllerRevision", "CronJob",
	"CustomResourceDefinition", "DaemonSet", "Deployment", "Endpoints", "Event", "HorizontalPodAutoscaler",
	"Ingress", "Job", "Lease", "LimitRange", "LocalSubjectAccessReview", "MutatingWebhookConfiguration",
	"Namespace", "NetworkPolicy", "Node", "PersistentVolume", "PersistentVolumeClaim", "Pod",
	"PodDisruptionBudget", "PodSecurityPolicy", "PodTemplate", "PriorityClass", "ReplicaSet",
	"ReplicationController", "ResourceQuota", "Role", "RoleBinding", "Secret", "SelfSubjectAccessReview",
	"SelfSubjectRulesReview", "Service", "ServiceAccount", "StatefulSet", "StorageClass",
	"SubjectAccessReview", "TokenReview", "ValidatingWebhookConfiguration", "VolumeAttachment"}

type WorkloadID struct {
	Cluster   string
	Namespace string
	Kind      string
	Name      string
}

func WLID(cluster, namespace, kind, name string) string {
	kind = strings.ReplaceAll(strings.ToLower(kind), "-", "")
	id := wlidScheme + clusterPrefix + cluster + "/" + namespacePrefix + namespace
	if kind == "" {
		return id
	}
	id += "/" + kind
	if name == "" {
		return id
	}
	return id + "-" + name
}

func ParseWLID(id string) WorkloadID {
	var w WorkloadID
	if id == "" || strings.Contains(id, " ") {
		return w
	}
	parts := splitInner(strings.Split(trimScheme(id), "/"))
	if len(parts) >= 1 {
		w.Cluster = parts[0]
	}
	if len(parts) >= 2 {
		w.Namespace = parts[1]
	}
	if len(parts) >= 3 {
		w.Kind = canonicalKind(parts[2])
	}
	if len(parts) >= 4 {
		w.Name = canonicalKind(parts[3])
	}
	return w
}

func ValidateWLID(id string) error {
	if id == "" {
		return errors.New("in RestoreMicroserviceIDsFromSpiffe, expecting valid wlid recieved empty string")
	}
	if strings.Contains(id, " ") {
		return fmt.Errorf("wlid %s invalid. whitespace found", id)
	}
	parts := strings.Split(trimScheme(id), "/")
	if len(parts) <= 3 {
		parts = splitInner(parts)
	}
	if len(parts) != 4 {
		return fmt.Errorf("invalid WLID format. format received: %v", parts)
	}
	for _, p := range parts {
		if p == "" {
			return fmt.Errorf("one or more entities are empty, spiffeSlices: %v", parts)
		}
	}
	return nil
}

func trimScheme(id string) string {
	if rest, ok := strings.CutPrefix(id, wlidScheme); ok {
		return rest
	}
	rest, _ := strings.CutPrefix(id, sidScheme)
	return rest
}

func splitInner(parts []string) []string {
	if len(parts) >= 1 {
		parts[0] = strings.TrimPrefix(parts[0], clusterPrefix)
	}
	if len(parts) >= 2 {
		parts[1] = strings.TrimPrefix(parts[1], namespacePrefix)
	}
	if len(parts) >= 3 {
		if kind, name, ok := strings.Cut(parts[2], "-"); ok {
			parts[2] = kind
			parts = append(parts, name)
		}
	}
	return parts
}

func canonicalKind(kind string) string {
	lower := strings.ToLower(kind)
	for _, k := range knownKinds {
		if lower == strings.ToLower(k) {
			return k
		}
	}
	return kind
}
