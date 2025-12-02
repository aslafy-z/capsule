// Copyright 2020-2021 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"os"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
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
// It queries namespace labels to extract tenant information based on the configured TenantLabel.
type KubernetesTenantResolver struct {
	// TenantLabel is the label used to identify tenant namespaces.
	TenantLabel string

	// TTL is the cache TTL in seconds.
	TTL int

	// cache stores namespace-to-tenant mappings.
	cache sync.Map

	// mu protects client initialization.
	mu sync.Mutex

	// client is the Kubernetes client used for API calls.
	client kubernetes.Interface

	// clientInitialized indicates if the client has been initialized.
	clientInitialized bool

	// clientErr stores any error from client initialization.
	clientErr error
}

// cacheEntry represents a cached tenant lookup result.
type cacheEntry struct {
	tenant string
	expiry time.Time
}

// initClient initializes the Kubernetes client.
func (r *KubernetesTenantResolver) initClient() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if already initialized
	if r.clientInitialized {
		return r.clientErr
	}

	var config *rest.Config

	// Try in-cluster config first (when running inside a pod)
	config, r.clientErr = rest.InClusterConfig()
	if r.clientErr != nil {
		// Fall back to kubeconfig file
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = os.Getenv("HOME") + "/.kube/config"
		}
		config, r.clientErr = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if r.clientErr != nil {
			return r.clientErr
		}
	}

	r.client, r.clientErr = kubernetes.NewForConfig(config)
	r.clientInitialized = true

	return r.clientErr
}

// GetTenantForNamespace returns the tenant name for a given namespace.
// It queries the Kubernetes API to get the namespace labels and extracts
// the tenant information based on the configured TenantLabel.
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

	// Initialize client if needed
	if err := r.initClient(); err != nil {
		return "", err
	}

	// Query Kubernetes API for namespace
	ns, err := r.client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		// Only cache empty result for NotFound errors
		// Other errors (network, permissions) should not be cached
		if k8serrors.IsNotFound(err) {
			r.cache.Store(namespace, &cacheEntry{
				tenant: "",
				expiry: time.Now().Add(time.Duration(r.TTL) * time.Second),
			})

			return "", nil
		}

		// Return error for temporary failures
		return "", err
	}

	// Extract tenant from labels
	tenant := r.getTenantFromNamespace(ns)

	// Store in cache
	r.cache.Store(namespace, &cacheEntry{
		tenant: tenant,
		expiry: time.Now().Add(time.Duration(r.TTL) * time.Second),
	})

	return tenant, nil
}

// getTenantFromNamespace extracts the tenant name from namespace labels.
func (r *KubernetesTenantResolver) getTenantFromNamespace(ns *corev1.Namespace) string {
	if ns.Labels == nil {
		return ""
	}

	return ns.Labels[r.TenantLabel]
}

// SetClient sets a custom Kubernetes client (useful for testing).
// This method should only be called before any GetTenantForNamespace calls.
func (r *KubernetesTenantResolver) SetClient(client kubernetes.Interface) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.client = client
	r.clientInitialized = true
	r.clientErr = nil
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
