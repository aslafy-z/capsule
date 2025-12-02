// Copyright 2020-2021 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package coredns

// IsolationMode defines how DNS resolution is restricted.
type IsolationMode string

const (
	// IsolationModeTenant restricts DNS resolution to services within the same tenant.
	IsolationModeTenant IsolationMode = "tenant"
	// IsolationModeNamespace restricts DNS resolution to services within the same namespace.
	IsolationModeNamespace IsolationMode = "namespace"
)

// Config holds the configuration for the Capsule CoreDNS plugin.
type Config struct {
	// IsolationMode determines whether to use tenant-level or namespace-level isolation.
	IsolationMode IsolationMode

	// WhitelistedNamespaces is a list of namespace patterns that are always accessible.
	// Supports exact matches and glob patterns (e.g., "kube-*").
	WhitelistedNamespaces []string

	// TenantLabelKey is the label key used to identify tenant membership.
	// Defaults to "capsule.clastix.io/tenant".
	TenantLabelKey string

	// ClusterDomain is the DNS domain for the Kubernetes cluster.
	// Defaults to "cluster.local".
	ClusterDomain string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		IsolationMode:         IsolationModeTenant,
		WhitelistedNamespaces: []string{"default", "kube-system"},
		TenantLabelKey:        "capsule.clastix.io/tenant",
		ClusterDomain:         "cluster.local",
	}
}

// RequestContext contains metadata extracted from a DNS request.
type RequestContext struct {
	// SourceNamespace is the namespace of the pod making the DNS request.
	SourceNamespace string
	// SourceTenant is the tenant of the pod making the DNS request.
	SourceTenant string
	// TargetNamespace is the namespace being resolved.
	TargetNamespace string
	// TargetService is the service name being resolved.
	TargetService string
}

// TenantResolver is an interface for looking up tenant membership.
type TenantResolver interface {
	// GetTenantForNamespace returns the tenant name for a given namespace.
	// Returns empty string if the namespace doesn't belong to any tenant.
	GetTenantForNamespace(namespace string) (string, error)

	// GetNamespacesForTenant returns all namespaces belonging to a tenant.
	GetNamespacesForTenant(tenant string) ([]string, error)
}
