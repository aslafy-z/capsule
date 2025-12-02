// Copyright 2020-2021 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package coredns

import (
	"path/filepath"
	"strings"
)

// AccessController determines whether DNS requests should be allowed or denied
// based on tenant/namespace isolation rules.
type AccessController struct {
	config         *Config
	tenantResolver TenantResolver
}

// NewAccessController creates a new AccessController with the given configuration.
func NewAccessController(config *Config, resolver TenantResolver) *AccessController {
	return &AccessController{
		config:         config,
		tenantResolver: resolver,
	}
}

// IsAccessAllowed determines if a DNS request from the source context
// should be allowed to resolve the target namespace.
func (ac *AccessController) IsAccessAllowed(ctx *RequestContext) (bool, error) {
	// Check whitelisted namespaces first
	if ac.isNamespaceWhitelisted(ctx.TargetNamespace) {
		return true, nil
	}

	// Apply isolation mode rules
	switch ac.config.IsolationMode {
	case IsolationModeNamespace:
		return ac.checkNamespaceIsolation(ctx)
	case IsolationModeTenant:
		return ac.checkTenantIsolation(ctx)
	default:
		// Unknown mode - deny by default
		return false, nil
	}
}

// checkNamespaceIsolation verifies that source and target are in the same namespace.
func (ac *AccessController) checkNamespaceIsolation(ctx *RequestContext) (bool, error) {
	return ctx.SourceNamespace == ctx.TargetNamespace, nil
}

// checkTenantIsolation verifies that source and target are in the same tenant.
func (ac *AccessController) checkTenantIsolation(ctx *RequestContext) (bool, error) {
	// If source tenant is already known, use it
	sourceTenant := ctx.SourceTenant
	if sourceTenant == "" && ac.tenantResolver != nil {
		// Try to resolve tenant for source namespace
		var err error
		sourceTenant, err = ac.tenantResolver.GetTenantForNamespace(ctx.SourceNamespace)
		if err != nil {
			return false, err
		}
	}

	// If source has no tenant, it can only resolve within its own namespace
	if sourceTenant == "" {
		return ctx.SourceNamespace == ctx.TargetNamespace, nil
	}

	// Get tenant for target namespace
	var targetTenant string
	if ac.tenantResolver != nil {
		var err error
		targetTenant, err = ac.tenantResolver.GetTenantForNamespace(ctx.TargetNamespace)
		if err != nil {
			return false, err
		}
	}

	// Allow if both are in the same tenant
	return sourceTenant == targetTenant && sourceTenant != "", nil
}

// isNamespaceWhitelisted checks if the namespace matches any whitelist pattern.
func (ac *AccessController) isNamespaceWhitelisted(namespace string) bool {
	for _, pattern := range ac.config.WhitelistedNamespaces {
		if matchPattern(pattern, namespace) {
			return true
		}
	}

	return false
}

// matchPattern checks if a string matches a glob-like pattern.
// Supports * as a wildcard for any characters.
func matchPattern(pattern, value string) bool {
	// Use filepath.Match for glob pattern matching
	matched, err := filepath.Match(pattern, value)
	if err != nil {
		// If pattern is invalid, try exact match
		return pattern == value
	}

	return matched
}

// ParseDNSQuery extracts namespace and service information from a Kubernetes DNS query.
// Kubernetes DNS format: <service>.<namespace>.svc.<cluster-domain>
func ParseDNSQuery(qname, clusterDomain string) (service, namespace string, ok bool) {
	// Remove trailing dot and cluster domain
	qname = strings.TrimSuffix(qname, ".")
	clusterDomain = strings.TrimSuffix(clusterDomain, ".")

	// Check if this is a service query
	suffix := ".svc." + clusterDomain
	if !strings.HasSuffix(qname, suffix) {
		// Also try pod format: <pod-ip>.<namespace>.pod.<cluster-domain>
		podSuffix := ".pod." + clusterDomain
		if strings.HasSuffix(qname, podSuffix) {
			// Extract namespace from pod query
			parts := strings.Split(strings.TrimSuffix(qname, podSuffix), ".")
			if len(parts) >= 2 {
				return "", parts[len(parts)-1], true
			}
		}

		return "", "", false
	}

	// Remove the suffix to get service.namespace
	nameWithNs := strings.TrimSuffix(qname, suffix)
	parts := strings.Split(nameWithNs, ".")

	// Handle different formats:
	// - service.namespace (basic)
	// - service.namespace (with port/protocol prefix not common in basic queries)
	if len(parts) < 2 {
		return "", "", false
	}

	// Last part is namespace, everything before is service-related
	namespace = parts[len(parts)-1]
	service = strings.Join(parts[:len(parts)-1], ".")

	return service, namespace, true
}
