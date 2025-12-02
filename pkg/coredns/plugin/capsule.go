// Copyright 2020-2021 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"strings"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

// Capsule is a CoreDNS plugin that enforces DNS resolution boundaries
// based on Capsule tenant/namespace isolation policies.
type Capsule struct {
	Next plugin.Handler

	// IsolationMode specifies how DNS isolation is enforced
	// Possible values: "tenant", "namespace"
	IsolationMode string

	// WhitelistedNamespaces contains namespaces that are always resolvable
	WhitelistedNamespaces []string

	// ClusterDomain is the cluster's DNS domain (e.g., "cluster.local")
	ClusterDomain string

	// TenantResolver resolves tenant information from request metadata
	TenantResolver TenantResolverInterface
}

// ServeDNS implements the plugin.Handler interface.
func (c Capsule) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}

	// Extract the source namespace and tenant from the request metadata
	sourceNamespace := extractMetadata(ctx, "namespace")
	sourceTenant := extractMetadata(ctx, "tenant")

	// If no source metadata is available, pass to the next plugin
	if sourceNamespace == "" {
		return plugin.NextOrFailure(c.Name(), c.Next, ctx, w, r)
	}

	// Extract the target namespace from the DNS query
	targetNamespace := c.extractTargetNamespace(state.QName())

	// If target namespace couldn't be determined, pass to next plugin
	if targetNamespace == "" {
		return plugin.NextOrFailure(c.Name(), c.Next, ctx, w, r)
	}

	// Check if resolution is allowed
	allowed, err := c.isResolutionAllowed(ctx, sourceNamespace, sourceTenant, targetNamespace)
	if err != nil {
		return dns.RcodeServerFailure, err
	}

	if !allowed {
		// Return NXDOMAIN for blocked queries
		msg := new(dns.Msg)
		msg.SetRcode(r, dns.RcodeNameError)

		if err := w.WriteMsg(msg); err != nil {
			return dns.RcodeServerFailure, err
		}

		return dns.RcodeNameError, nil
	}

	return plugin.NextOrFailure(c.Name(), c.Next, ctx, w, r)
}

// Name returns the plugin name.
func (c Capsule) Name() string {
	return "capsule"
}

// extractTargetNamespace extracts the namespace from a Kubernetes DNS query name.
// Kubernetes DNS format: <service>.<namespace>.svc.<cluster-domain>
// or <pod-ip>.<namespace>.pod.<cluster-domain>.
func (c Capsule) extractTargetNamespace(qname string) string {
	// Remove trailing dot if present
	qname = strings.TrimSuffix(qname, ".")

	// Check if the query is for a Kubernetes service or pod
	if !strings.HasSuffix(qname, "."+c.ClusterDomain) {
		return ""
	}

	// Remove the cluster domain
	qname = strings.TrimSuffix(qname, "."+c.ClusterDomain)

	// Split the remaining parts
	parts := strings.Split(qname, ".")

	// For services: <service>.<namespace>.svc
	// For pods: <pod-ip>.<namespace>.pod
	// For headless services: <hostname>.<subdomain>.<namespace>.svc
	for i, part := range parts {
		if (part == "svc" || part == "pod") && i > 0 {
			return parts[i-1]
		}
	}

	return ""
}

// isResolutionAllowed checks if DNS resolution from source to target namespace is allowed.
func (c Capsule) isResolutionAllowed(ctx context.Context, sourceNamespace, sourceTenant, targetNamespace string) (bool, error) {
	// Check if target namespace is whitelisted
	if c.isNamespaceWhitelisted(targetNamespace) {
		return true, nil
	}

	// Same namespace is always allowed
	if sourceNamespace == targetNamespace {
		return true, nil
	}

	// Apply isolation based on mode
	switch c.IsolationMode {
	case IsolationModeNamespace:
		// In namespace mode, only same namespace is allowed (already checked above)
		return false, nil
	case IsolationModeTenant:
		// In tenant mode, check if both namespaces belong to the same tenant
		if c.TenantResolver == nil {
			// If no resolver is configured, default to namespace isolation
			return false, nil
		}

		targetTenant, err := c.TenantResolver.GetTenantForNamespace(ctx, targetNamespace)
		if err != nil {
			return false, err
		}

		// Allow if both namespaces belong to the same tenant
		return sourceTenant != "" && sourceTenant == targetTenant, nil
	default:
		// Default to allowing resolution if mode is not recognized
		return true, nil
	}
}

// isNamespaceWhitelisted checks if a namespace is in the whitelist.
func (c Capsule) isNamespaceWhitelisted(namespace string) bool {
	for _, ns := range c.WhitelistedNamespaces {
		if ns == namespace {
			return true
		}
	}

	return false
}
