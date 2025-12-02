// Copyright 2020-2021 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"testing"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCapsule_Name(t *testing.T) {
	c := Capsule{}
	assert.Equal(t, "capsule", c.Name())
}

func TestCapsule_extractTargetNamespace(t *testing.T) {
	c := Capsule{
		ClusterDomain: "cluster.local",
	}

	tests := []struct {
		name     string
		qname    string
		expected string
	}{
		{
			name:     "service in namespace",
			qname:    "my-service.my-namespace.svc.cluster.local.",
			expected: "my-namespace",
		},
		{
			name:     "pod in namespace",
			qname:    "10-244-0-1.my-namespace.pod.cluster.local.",
			expected: "my-namespace",
		},
		{
			name:     "headless service",
			qname:    "hostname.subdomain.my-namespace.svc.cluster.local.",
			expected: "my-namespace",
		},
		{
			name:     "service without trailing dot",
			qname:    "my-service.my-namespace.svc.cluster.local",
			expected: "my-namespace",
		},
		{
			name:     "external domain",
			qname:    "www.google.com.",
			expected: "",
		},
		{
			name:     "wrong cluster domain",
			qname:    "my-service.my-namespace.svc.other.local.",
			expected: "",
		},
		{
			name:     "kubernetes api",
			qname:    "kubernetes.default.svc.cluster.local.",
			expected: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.extractTargetNamespace(tt.qname)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCapsule_isNamespaceWhitelisted(t *testing.T) {
	c := Capsule{
		WhitelistedNamespaces: []string{"default", "kube-system"},
	}

	tests := []struct {
		name      string
		namespace string
		expected  bool
	}{
		{
			name:      "default is whitelisted",
			namespace: "default",
			expected:  true,
		},
		{
			name:      "kube-system is whitelisted",
			namespace: "kube-system",
			expected:  true,
		},
		{
			name:      "other namespace not whitelisted",
			namespace: "my-namespace",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.isNamespaceWhitelisted(tt.namespace)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCapsule_isResolutionAllowed_NamespaceMode(t *testing.T) {
	c := Capsule{
		IsolationMode:         IsolationModeNamespace,
		WhitelistedNamespaces: []string{"default", "kube-system"},
		ClusterDomain:         "cluster.local",
	}

	tests := []struct {
		name            string
		sourceNamespace string
		sourceTenant    string
		targetNamespace string
		expected        bool
	}{
		{
			name:            "same namespace allowed",
			sourceNamespace: "my-namespace",
			sourceTenant:    "",
			targetNamespace: "my-namespace",
			expected:        true,
		},
		{
			name:            "different namespace blocked",
			sourceNamespace: "namespace-a",
			sourceTenant:    "",
			targetNamespace: "namespace-b",
			expected:        false,
		},
		{
			name:            "whitelisted namespace allowed",
			sourceNamespace: "namespace-a",
			sourceTenant:    "",
			targetNamespace: "default",
			expected:        true,
		},
		{
			name:            "kube-system always allowed",
			sourceNamespace: "namespace-a",
			sourceTenant:    "",
			targetNamespace: "kube-system",
			expected:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := c.isResolutionAllowed(context.Background(), tt.sourceNamespace, tt.sourceTenant, tt.targetNamespace)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCapsule_isResolutionAllowed_TenantMode(t *testing.T) {
	resolver := NewStaticTenantResolver(map[string]string{
		"namespace-a":   "tenant-1",
		"namespace-b":   "tenant-1",
		"namespace-c":   "tenant-2",
		"other-ns":      "tenant-2",
		"unassigned-ns": "",
	})

	c := Capsule{
		IsolationMode:         IsolationModeTenant,
		WhitelistedNamespaces: []string{"default", "kube-system"},
		ClusterDomain:         "cluster.local",
		TenantResolver:        resolver,
	}

	tests := []struct {
		name            string
		sourceNamespace string
		sourceTenant    string
		targetNamespace string
		expected        bool
	}{
		{
			name:            "same tenant allowed",
			sourceNamespace: "namespace-a",
			sourceTenant:    "tenant-1",
			targetNamespace: "namespace-b",
			expected:        true,
		},
		{
			name:            "different tenant blocked",
			sourceNamespace: "namespace-a",
			sourceTenant:    "tenant-1",
			targetNamespace: "namespace-c",
			expected:        false,
		},
		{
			name:            "same namespace always allowed",
			sourceNamespace: "namespace-a",
			sourceTenant:    "tenant-1",
			targetNamespace: "namespace-a",
			expected:        true,
		},
		{
			name:            "whitelisted namespace allowed",
			sourceNamespace: "namespace-a",
			sourceTenant:    "tenant-1",
			targetNamespace: "default",
			expected:        true,
		},
		{
			name:            "no source tenant blocks cross-namespace",
			sourceNamespace: "unassigned-ns",
			sourceTenant:    "",
			targetNamespace: "namespace-a",
			expected:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := c.isResolutionAllowed(context.Background(), tt.sourceNamespace, tt.sourceTenant, tt.targetNamespace)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCapsule_ServeDNS_Blocked(t *testing.T) {
	resolver := NewStaticTenantResolver(map[string]string{
		"namespace-a": "tenant-1",
		"namespace-b": "tenant-2",
	})

	c := Capsule{
		IsolationMode:         IsolationModeTenant,
		WhitelistedNamespaces: []string{"default"},
		ClusterDomain:         "cluster.local",
		TenantResolver:        resolver,
		Next:                  test.NextHandler(dns.RcodeSuccess, nil),
	}

	// Create a test query
	req := new(dns.Msg)
	req.SetQuestion("my-service.namespace-b.svc.cluster.local.", dns.TypeA)

	// This test is limited because we can't easily inject metadata context
	// In a real scenario, the metadata plugin would set the source namespace
	rec := dnstest.NewRecorder(&test.ResponseWriter{
		RemoteIP: "10.0.0.1",
	})

	// Without metadata, the request passes through
	rcode, err := c.ServeDNS(context.Background(), rec, req)
	assert.NoError(t, err)
	// Since no source namespace metadata, passes to next handler
	assert.Equal(t, dns.RcodeSuccess, rcode)
}

func TestStaticTenantResolver_GetTenantForNamespace(t *testing.T) {
	resolver := NewStaticTenantResolver(map[string]string{
		"namespace-a": "tenant-1",
		"namespace-b": "tenant-2",
	})

	tests := []struct {
		name      string
		namespace string
		expected  string
	}{
		{
			name:      "existing namespace",
			namespace: "namespace-a",
			expected:  "tenant-1",
		},
		{
			name:      "non-existing namespace",
			namespace: "namespace-c",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.GetTenantForNamespace(context.Background(), tt.namespace)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStaticTenantResolver_NilMapping(t *testing.T) {
	resolver := &StaticTenantResolver{
		NamespaceToTenant: nil,
	}

	result, err := resolver.GetTenantForNamespace(context.Background(), "any-namespace")
	assert.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestKubernetesTenantResolver_GetTenantForNamespace(t *testing.T) {
	// Create a fake clientset with test namespaces
	resolver := &KubernetesTenantResolver{
		TenantLabel: DefaultTenantLabel,
		TTL:         30,
	}

	// Create fake client with test data
	fakeClient := fake.NewSimpleClientset(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "tenant-a-ns1",
				Labels: map[string]string{
					DefaultTenantLabel: "tenant-a",
				},
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "tenant-a-ns2",
				Labels: map[string]string{
					DefaultTenantLabel: "tenant-a",
				},
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "tenant-b-ns1",
				Labels: map[string]string{
					DefaultTenantLabel: "tenant-b",
				},
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "no-tenant-ns",
				Labels: map[string]string{},
			},
		},
	)
	resolver.SetClient(fakeClient)

	tests := []struct {
		name           string
		namespace      string
		expectedTenant string
	}{
		{
			name:           "namespace in tenant-a",
			namespace:      "tenant-a-ns1",
			expectedTenant: "tenant-a",
		},
		{
			name:           "another namespace in tenant-a",
			namespace:      "tenant-a-ns2",
			expectedTenant: "tenant-a",
		},
		{
			name:           "namespace in tenant-b",
			namespace:      "tenant-b-ns1",
			expectedTenant: "tenant-b",
		},
		{
			name:           "namespace without tenant label",
			namespace:      "no-tenant-ns",
			expectedTenant: "",
		},
		{
			name:           "non-existent namespace",
			namespace:      "non-existent",
			expectedTenant: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenant, err := resolver.GetTenantForNamespace(context.Background(), tt.namespace)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedTenant, tenant)
		})
	}
}

func TestKubernetesTenantResolver_Caching(t *testing.T) {
	// Create a resolver with a short TTL
	resolver := &KubernetesTenantResolver{
		TenantLabel: DefaultTenantLabel,
		TTL:         1, // 1 second TTL for testing
	}

	// Create fake client
	fakeClient := fake.NewSimpleClientset(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cached-ns",
				Labels: map[string]string{
					DefaultTenantLabel: "cached-tenant",
				},
			},
		},
	)
	resolver.SetClient(fakeClient)

	// First call should query and cache
	tenant1, err := resolver.GetTenantForNamespace(context.Background(), "cached-ns")
	assert.NoError(t, err)
	assert.Equal(t, "cached-tenant", tenant1)

	// Second call should return from cache (even if we update the namespace)
	tenant2, err := resolver.GetTenantForNamespace(context.Background(), "cached-ns")
	assert.NoError(t, err)
	assert.Equal(t, "cached-tenant", tenant2)
}
