// Copyright 2020-2021 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

/*
Package coredns provides a CoreDNS plugin for Capsule that enforces
tenant and namespace DNS isolation in multi-tenant Kubernetes clusters.

# Overview

The plugin can be configured to run in two isolation modes:

  - Tenant mode: pods can only resolve services within their own tenant
  - Namespace mode: pods can only resolve services within their own namespace

Whitelisted namespaces (e.g., default, kube-system) are always accessible
regardless of the isolation mode.

# Configuration

The plugin configuration can be created programmatically:

	config := coredns.NewConfigBuilder().
		WithIsolationMode(coredns.IsolationModeTenant).
		WithWhitelistedNamespaces("default", "kube-system").
		WithTenantLabelKey("capsule.clastix.io/tenant").
		Build()

Or parsed from a map of options:

	options := map[string]string{
		"isolation_mode": "tenant",
		"whitelist":      "default,kube-system",
	}
	config, err := coredns.ParseConfig(options)

# Usage

Create a plugin instance with a tenant resolver:

	plugin := coredns.NewPlugin(config, tenantResolver, nextHandler)

The plugin implements the standard DNS handler pattern and can be chained
with other handlers:

	response, err := plugin.ServeDNS(ctx, request)

# Tenant Resolution

To integrate with a tenant management system, implement the TenantResolver
interface:

	type TenantResolver interface {
		GetTenantForNamespace(namespace string) (string, error)
		GetNamespacesForTenant(tenant string) ([]string, error)
	}

# Security

  - Denied queries return NXDOMAIN to prevent information leakage
  - Resolver errors result in access denial by default
  - External (non-cluster) DNS queries pass through without restriction
*/
package coredns
