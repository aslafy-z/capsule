// Copyright 2020-2021 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package api

// ResourceEnforcementSpec defines labels and annotations that should be enforced on tenant resources.
// +kubebuilder:object:generate=true
type ResourceEnforcementSpec struct {
	// Labels that must be present on resources within the tenant.
	// +optional
	RequiredLabels map[string]string `json:"requiredLabels,omitempty"`
	// Annotations that must be present on resources within the tenant.
	// +optional
	RequiredAnnotations map[string]string `json:"requiredAnnotations,omitempty"`
	// DefaultLabels are labels that will be applied to resources if not already present.
	// +optional
	DefaultLabels map[string]string `json:"defaultLabels,omitempty"`
	// DefaultAnnotations are annotations that will be applied to resources if not already present.
	// +optional
	DefaultAnnotations map[string]string `json:"defaultAnnotations,omitempty"`
}
