/*
Copyright 2024 The Kubescape Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dynamicpathdetector

// clampInt32 clamps a runtime int into the int32 wire range used by the
// CollapseConfiguration CRD. Thresholds are physically small (single- or
// double-digit counts of trie children); clamping defends only against
// the autotune path being handed a pathological value.
type CollapseSettings struct {
	OpenDynamicThreshold     int
	EndpointDynamicThreshold int
	CollapseConfigs          []CollapseConfig
	NetworkIPGroupThreshold  int
	NetworkCIDRFloorBits     int
	ExecDynamicThreshold     int
}

// DefaultCollapseSettings returns the built-in baseline. The returned
// value is a fresh copy on every call — callers may freely mutate the
// CollapseConfigs slice without affecting the package state. This
// mirrors the defensive-copy contract the bare DefaultCollapseConfigs()
// accessor already enforces.
func DefaultCollapseSettings() CollapseSettings {
	return CollapseSettings{
		OpenDynamicThreshold:     OpenDynamicThreshold,
		EndpointDynamicThreshold: EndpointDynamicThreshold,
		CollapseConfigs:          DefaultCollapseConfigs(),
		NetworkIPGroupThreshold:  NetworkIPGroupThreshold,
		NetworkCIDRFloorBits:     NetworkCIDRFloorBits,
		ExecDynamicThreshold:     ExecDynamicThreshold,
	}
}

// CollapseSettingsFromCRD projects a CollapseConfiguration custom resource
// into the runtime form. The per-prefix override slice is converted
// entry-by-entry; per-prefix Threshold values are validated >= 1 at
// admission (see validateCollapseConfigurationSpec) so they are copied
// verbatim. Returns a value that does not alias the CRD's internal slice.
//
// Zero-guard on the global thresholds: a partial CR may omit a global
// threshold (or set only collapseConfigs), and JSON/proto decode leaves the
// omitted field at 0. A literal 0 here would mean "collapse any node with
// >= 1 child" — updateNodeStats collapses on Count > threshold — silently
// flattening every open/endpoint in every profile to a single ⋯. Treat a
// non-positive global threshold as "use the compiled-in default" instead,
// matching the absent-CR fallback the provider already performs. The same
// non-positive-means-default guard applies to NetworkIPGroupThreshold and
// NetworkCIDRFloorBits: a literal 0 group threshold would CIDR-collapse
// every NetworkNeighbor group, and a literal 0 floor would forbid any
// aggregated CIDR block.
type CollapseSettingsProvider func() CollapseSettings
