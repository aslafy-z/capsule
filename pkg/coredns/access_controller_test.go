// Copyright 2020-2021 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package coredns

import (
	"context"
	"errors"
	"testing"
)

// mockTenantResolver is a mock implementation of TenantResolver for testing.
type mockTenantResolver struct {
	namespaceToTenant map[string]string
	tenantNamespaces  map[string][]string
	err               error
}

func (m *mockTenantResolver) GetTenantForNamespace(namespace string) (string, error) {
	if m.err != nil {
		return "", m.err
	}

	return m.namespaceToTenant[namespace], nil
}

func (m *mockTenantResolver) GetNamespacesForTenant(tenant string) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.tenantNamespaces[tenant], nil
}

func TestAccessController_IsAccessAllowed_NamespaceIsolation(t *testing.T) {
	t.Parallel()

	config := &Config{
		IsolationMode:         IsolationModeNamespace,
		WhitelistedNamespaces: []string{"default", "kube-system"},
		TenantLabelKey:        "capsule.clastix.io/tenant",
	}

	ac := NewAccessController(config, nil)

	tests := []struct {
		name           string
		sourceNs       string
		targetNs       string
		expectedResult bool
	}{
		{
			name:           "same namespace access allowed",
			sourceNs:       "tenant-a-ns1",
			targetNs:       "tenant-a-ns1",
			expectedResult: true,
		},
		{
			name:           "different namespace access denied",
			sourceNs:       "tenant-a-ns1",
			targetNs:       "tenant-a-ns2",
			expectedResult: false,
		},
		{
			name:           "access to whitelisted namespace allowed",
			sourceNs:       "tenant-a-ns1",
			targetNs:       "default",
			expectedResult: true,
		},
		{
			name:           "access to kube-system allowed",
			sourceNs:       "tenant-a-ns1",
			targetNs:       "kube-system",
			expectedResult: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := &RequestContext{
				SourceNamespace: tc.sourceNs,
				TargetNamespace: tc.targetNs,
			}

			result, err := ac.IsAccessAllowed(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tc.expectedResult {
				t.Errorf("expected %v, got %v", tc.expectedResult, result)
			}
		})
	}
}

func TestAccessController_IsAccessAllowed_TenantIsolation(t *testing.T) {
	t.Parallel()

	resolver := &mockTenantResolver{
		namespaceToTenant: map[string]string{
			"tenant-a-ns1": "tenant-a",
			"tenant-a-ns2": "tenant-a",
			"tenant-b-ns1": "tenant-b",
		},
	}

	config := &Config{
		IsolationMode:         IsolationModeTenant,
		WhitelistedNamespaces: []string{"default", "kube-*"},
		TenantLabelKey:        "capsule.clastix.io/tenant",
	}

	ac := NewAccessController(config, resolver)

	tests := []struct {
		name           string
		sourceNs       string
		sourceTenant   string
		targetNs       string
		expectedResult bool
	}{
		{
			name:           "same tenant access allowed",
			sourceNs:       "tenant-a-ns1",
			targetNs:       "tenant-a-ns2",
			expectedResult: true,
		},
		{
			name:           "different tenant access denied",
			sourceNs:       "tenant-a-ns1",
			targetNs:       "tenant-b-ns1",
			expectedResult: false,
		},
		{
			name:           "access to whitelisted namespace allowed",
			sourceNs:       "tenant-a-ns1",
			targetNs:       "default",
			expectedResult: true,
		},
		{
			name:           "access to kube-system with glob pattern allowed",
			sourceNs:       "tenant-a-ns1",
			targetNs:       "kube-system",
			expectedResult: true,
		},
		{
			name:           "access to kube-public with glob pattern allowed",
			sourceNs:       "tenant-a-ns1",
			targetNs:       "kube-public",
			expectedResult: true,
		},
		{
			name:           "pre-resolved tenant context",
			sourceNs:       "tenant-a-ns1",
			sourceTenant:   "tenant-a",
			targetNs:       "tenant-a-ns2",
			expectedResult: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := &RequestContext{
				SourceNamespace: tc.sourceNs,
				SourceTenant:    tc.sourceTenant,
				TargetNamespace: tc.targetNs,
			}

			result, err := ac.IsAccessAllowed(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tc.expectedResult {
				t.Errorf("expected %v, got %v", tc.expectedResult, result)
			}
		})
	}
}

func TestAccessController_NoTenantNamespace(t *testing.T) {
	t.Parallel()

	resolver := &mockTenantResolver{
		namespaceToTenant: map[string]string{
			"tenant-a-ns1": "tenant-a",
			// "standalone-ns" has no tenant
		},
	}

	config := &Config{
		IsolationMode:         IsolationModeTenant,
		WhitelistedNamespaces: []string{"default"},
		TenantLabelKey:        "capsule.clastix.io/tenant",
	}

	ac := NewAccessController(config, resolver)

	tests := []struct {
		name           string
		sourceNs       string
		targetNs       string
		expectedResult bool
	}{
		{
			name:           "no-tenant namespace can access itself",
			sourceNs:       "standalone-ns",
			targetNs:       "standalone-ns",
			expectedResult: true,
		},
		{
			name:           "no-tenant namespace cannot access other",
			sourceNs:       "standalone-ns",
			targetNs:       "tenant-a-ns1",
			expectedResult: false,
		},
		{
			name:           "no-tenant namespace can access whitelisted",
			sourceNs:       "standalone-ns",
			targetNs:       "default",
			expectedResult: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := &RequestContext{
				SourceNamespace: tc.sourceNs,
				TargetNamespace: tc.targetNs,
			}

			result, err := ac.IsAccessAllowed(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tc.expectedResult {
				t.Errorf("expected %v, got %v", tc.expectedResult, result)
			}
		})
	}
}

func TestAccessController_ResolverError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("resolver error")
	resolver := &mockTenantResolver{
		err: expectedErr,
	}

	config := &Config{
		IsolationMode:         IsolationModeTenant,
		WhitelistedNamespaces: []string{},
		TenantLabelKey:        "capsule.clastix.io/tenant",
	}

	ac := NewAccessController(config, resolver)

	ctx := &RequestContext{
		SourceNamespace: "some-ns",
		TargetNamespace: "other-ns",
	}

	_, err := ac.IsAccessAllowed(ctx)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestParseDNSQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		qname      string
		clusterDom string
		expService string
		expNs      string
		expectedOk bool
	}{
		{
			name:       "basic service query",
			qname:      "my-service.my-namespace.svc.cluster.local.",
			clusterDom: "cluster.local",
			expService: "my-service",
			expNs:      "my-namespace",
			expectedOk: true,
		},
		{
			name:       "service query without trailing dot",
			qname:      "my-service.my-namespace.svc.cluster.local",
			clusterDom: "cluster.local",
			expService: "my-service",
			expNs:      "my-namespace",
			expectedOk: true,
		},
		{
			name:       "kubernetes default service",
			qname:      "kubernetes.default.svc.cluster.local.",
			clusterDom: "cluster.local",
			expService: "kubernetes",
			expNs:      "default",
			expectedOk: true,
		},
		{
			name:       "external query",
			qname:      "www.google.com.",
			clusterDom: "cluster.local",
			expService: "",
			expNs:      "",
			expectedOk: false,
		},
		{
			name:       "pod query",
			qname:      "10-244-0-1.my-namespace.pod.cluster.local.",
			clusterDom: "cluster.local",
			expService: "",
			expNs:      "my-namespace",
			expectedOk: true,
		},
		{
			name:       "headless service pod query",
			qname:      "pod-0.my-service.my-namespace.svc.cluster.local.",
			clusterDom: "cluster.local",
			expService: "pod-0.my-service",
			expNs:      "my-namespace",
			expectedOk: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service, ns, ok := ParseDNSQuery(tc.qname, tc.clusterDom)

			if ok != tc.expectedOk {
				t.Errorf("expected ok=%v, got ok=%v", tc.expectedOk, ok)
			}

			if ok {
				if service != tc.expService {
					t.Errorf("expected service=%q, got service=%q", tc.expService, service)
				}

				if ns != tc.expNs {
					t.Errorf("expected namespace=%q, got namespace=%q", tc.expNs, ns)
				}
			}
		})
	}
}

func TestMatchPattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pattern  string
		value    string
		expected bool
	}{
		{"kube-system", "kube-system", true},
		{"kube-*", "kube-system", true},
		{"kube-*", "kube-public", true},
		{"kube-*", "default", false},
		{"*-system", "kube-system", true},
		{"*", "anything", true},
		{"exact", "notexact", false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.pattern+"_"+tc.value, func(t *testing.T) {
			t.Parallel()

			result := matchPattern(tc.pattern, tc.value)
			if result != tc.expected {
				t.Errorf("matchPattern(%q, %q) = %v, expected %v", tc.pattern, tc.value, result, tc.expected)
			}
		})
	}
}

// mockHandler is a mock implementation of Handler for testing.
type mockHandler struct {
	response *DNSResponse
	err      error
	called   bool
}

func (m *mockHandler) ServeDNS(_ context.Context, _ *DNSRequest) (*DNSResponse, error) {
	m.called = true

	return m.response, m.err
}

func TestPlugin_ServeDNS(t *testing.T) {
	t.Parallel()

	resolver := &mockTenantResolver{
		namespaceToTenant: map[string]string{
			"tenant-a-ns1": "tenant-a",
			"tenant-b-ns1": "tenant-b",
		},
	}

	config := &Config{
		IsolationMode:         IsolationModeTenant,
		WhitelistedNamespaces: []string{"default"},
		TenantLabelKey:        "capsule.clastix.io/tenant",
	}

	tests := []struct {
		name           string
		request        *DNSRequest
		expectedRcode  int
		expectNextCall bool
	}{
		{
			name: "allowed access passes to next handler",
			request: &DNSRequest{
				Question: DNSQuestion{Name: "my-svc.tenant-a-ns1.svc.cluster.local."},
				Metadata: map[string]string{
					"namespace":                 "tenant-a-ns1",
					"capsule.clastix.io/tenant": "tenant-a",
				},
			},
			expectedRcode:  RcodeSuccess,
			expectNextCall: true,
		},
		{
			name: "denied access returns NXDOMAIN",
			request: &DNSRequest{
				Question: DNSQuestion{Name: "my-svc.tenant-b-ns1.svc.cluster.local."},
				Metadata: map[string]string{
					"namespace":                 "tenant-a-ns1",
					"capsule.clastix.io/tenant": "tenant-a",
				},
			},
			expectedRcode:  RcodeNameError,
			expectNextCall: false,
		},
		{
			name: "whitelisted namespace access allowed",
			request: &DNSRequest{
				Question: DNSQuestion{Name: "kubernetes.default.svc.cluster.local."},
				Metadata: map[string]string{
					"namespace":                 "tenant-a-ns1",
					"capsule.clastix.io/tenant": "tenant-a",
				},
			},
			expectedRcode:  RcodeSuccess,
			expectNextCall: true,
		},
		{
			name: "external query passes through",
			request: &DNSRequest{
				Question: DNSQuestion{Name: "www.google.com."},
				Metadata: map[string]string{
					"namespace": "tenant-a-ns1",
				},
			},
			expectedRcode:  RcodeSuccess,
			expectNextCall: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			nextHandler := &mockHandler{
				response: &DNSResponse{Rcode: RcodeSuccess},
			}

			plugin := NewPlugin(config, resolver, nextHandler)

			resp, _ := plugin.ServeDNS(context.Background(), tc.request)

			if resp.Rcode != tc.expectedRcode {
				t.Errorf("expected rcode %d, got %d", tc.expectedRcode, resp.Rcode)
			}

			if nextHandler.called != tc.expectNextCall {
				t.Errorf("expected next handler called=%v, got called=%v", tc.expectNextCall, nextHandler.called)
			}
		})
	}
}

func TestPlugin_Name(t *testing.T) {
	t.Parallel()

	plugin := NewPlugin(nil, nil, nil)
	if plugin.Name() != PluginName {
		t.Errorf("expected plugin name %q, got %q", PluginName, plugin.Name())
	}
}
