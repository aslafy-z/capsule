// Copyright 2020-2021 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"sync"
	"time"
)

const (
	// DefaultTenantLabel is the default label used to identify tenant namespaces.
	DefaultTenantLabel = "capsule.clastix.io/tenant"

	// DefaultCacheTTL is the default cache TTL in seconds.
	DefaultCacheTTL = 30
)

// TenantResolverInterface defines the interface for resolving tenant information.
type TenantResolverInterface interface {
	// GetTenantForNamespace returns the tenant name for a given namespace.
	// Returns empty string if the namespace doesn't belong to a tenant.
	GetTenantForNamespace(ctx context.Context, namespace string) (string, error)
}

// KubernetesTenantResolver implements TenantResolverInterface using Kubernetes API.
// Note: The current implementation is a stub that returns empty tenant for all namespaces.
// In production, this should be extended to query the Kubernetes API to retrieve
// namespace labels and extract the tenant information based on TenantLabel.
// Users can either provide their own implementation or integrate with the Capsule operator.
type KubernetesTenantResolver struct {
	// TenantLabel is the label used to identify tenant namespaces.
	TenantLabel string

	// TTL is the cache TTL in seconds.
	TTL int

	// cache stores namespace-to-tenant mappings.
	cache sync.Map
}

// cacheEntry represents a cached tenant lookup result.
type cacheEntry struct {
	tenant string
	expiry time.Time
}

// GetTenantForNamespace returns the tenant name for a given namespace.
func (r *KubernetesTenantResolver) GetTenantForNamespace(ctx context.Context, namespace string) (string, error) {
	// Check cache first
	if entry, ok := r.cache.Load(namespace); ok {
		ce := entry.(*cacheEntry)
		if time.Now().Before(ce.expiry) {
			return ce.tenant, nil
		}
		// Cache expired, remove entry
		r.cache.Delete(namespace)
	}

	// TODO: In production, implement Kubernetes API call to get namespace labels.
	// This stub returns empty tenant. Users should extend this or use StaticTenantResolver.
	tenant := ""

	// Store in cache
	r.cache.Store(namespace, &cacheEntry{
		tenant: tenant,
		expiry: time.Now().Add(time.Duration(r.TTL) * time.Second),
	})

	return tenant, nil
}

// StaticTenantResolver implements TenantResolverInterface using a static mapping.
// This is useful for testing and simple configurations.
type StaticTenantResolver struct {
	// NamespaceToTenant maps namespaces to their tenant.
	NamespaceToTenant map[string]string
}

// GetTenantForNamespace returns the tenant for a namespace from the static mapping.
func (r *StaticTenantResolver) GetTenantForNamespace(ctx context.Context, namespace string) (string, error) {
	if r.NamespaceToTenant == nil {
		return "", nil
	}

	return r.NamespaceToTenant[namespace], nil
}

// NewStaticTenantResolver creates a new StaticTenantResolver with the given mapping.
func NewStaticTenantResolver(mapping map[string]string) *StaticTenantResolver {
	return &StaticTenantResolver{
		NamespaceToTenant: mapping,
	}
}
