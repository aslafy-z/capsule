// Copyright 2020-2021 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package coredns

import (
	"testing"
)

func TestParseConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		options       map[string]string
		expectedMode  IsolationMode
		expectedNs    []string
		expectedLabel string
		expectError   bool
	}{
		{
			name:          "default config",
			options:       map[string]string{},
			expectedMode:  IsolationModeTenant,
			expectedNs:    []string{"default", "kube-system"},
			expectedLabel: "capsule.clastix.io/tenant",
			expectError:   false,
		},
		{
			name: "tenant isolation mode",
			options: map[string]string{
				"isolation_mode": "tenant",
			},
			expectedMode:  IsolationModeTenant,
			expectedNs:    []string{"default", "kube-system"},
			expectedLabel: "capsule.clastix.io/tenant",
			expectError:   false,
		},
		{
			name: "namespace isolation mode",
			options: map[string]string{
				"isolationMode": "namespace",
			},
			expectedMode:  IsolationModeNamespace,
			expectedNs:    []string{"default", "kube-system"},
			expectedLabel: "capsule.clastix.io/tenant",
			expectError:   false,
		},
		{
			name: "custom whitelist comma-separated",
			options: map[string]string{
				"whitelist": "default,kube-system,monitoring",
			},
			expectedMode:  IsolationModeTenant,
			expectedNs:    []string{"default", "kube-system", "monitoring"},
			expectedLabel: "capsule.clastix.io/tenant",
			expectError:   false,
		},
		{
			name: "custom whitelist space-separated",
			options: map[string]string{
				"whitelisted_namespaces": "ns1 ns2 ns3",
			},
			expectedMode:  IsolationModeTenant,
			expectedNs:    []string{"ns1", "ns2", "ns3"},
			expectedLabel: "capsule.clastix.io/tenant",
			expectError:   false,
		},
		{
			name: "custom tenant label",
			options: map[string]string{
				"tenant_label": "my.custom/tenant",
			},
			expectedMode:  IsolationModeTenant,
			expectedNs:    []string{"default", "kube-system"},
			expectedLabel: "my.custom/tenant",
			expectError:   false,
		},
		{
			name: "invalid isolation mode",
			options: map[string]string{
				"isolation_mode": "invalid",
			},
			expectError: true,
		},
		{
			name: "full config",
			options: map[string]string{
				"isolation_mode":   "namespace",
				"whitelist":        "default,kube-*",
				"tenant_label_key": "example.com/tenant",
				"cluster_domain":   "my.cluster.local",
			},
			expectedMode:  IsolationModeNamespace,
			expectedNs:    []string{"default", "kube-*"},
			expectedLabel: "example.com/tenant",
			expectError:   false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config, err := ParseConfig(tc.options)

			if tc.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if config.IsolationMode != tc.expectedMode {
				t.Errorf("expected isolation mode %q, got %q", tc.expectedMode, config.IsolationMode)
			}

			if len(config.WhitelistedNamespaces) != len(tc.expectedNs) {
				t.Errorf("expected %d whitelisted namespaces, got %d", len(tc.expectedNs), len(config.WhitelistedNamespaces))
			} else {
				for i, ns := range tc.expectedNs {
					if config.WhitelistedNamespaces[i] != ns {
						t.Errorf("expected whitelist[%d]=%q, got %q", i, ns, config.WhitelistedNamespaces[i])
					}
				}
			}

			if config.TenantLabelKey != tc.expectedLabel {
				t.Errorf("expected tenant label %q, got %q", tc.expectedLabel, config.TenantLabelKey)
			}
		})
	}
}

func TestConfigBuilder(t *testing.T) {
	t.Parallel()

	config := NewConfigBuilder().
		WithIsolationMode(IsolationModeNamespace).
		WithWhitelistedNamespaces("ns1", "ns2").
		AddWhitelistedNamespace("ns3").
		WithTenantLabelKey("custom/tenant").
		Build()

	if config.IsolationMode != IsolationModeNamespace {
		t.Errorf("expected isolation mode %q, got %q", IsolationModeNamespace, config.IsolationMode)
	}

	expectedNs := []string{"ns1", "ns2", "ns3"}
	if len(config.WhitelistedNamespaces) != len(expectedNs) {
		t.Errorf("expected %d whitelisted namespaces, got %d", len(expectedNs), len(config.WhitelistedNamespaces))
	}

	if config.TenantLabelKey != "custom/tenant" {
		t.Errorf("expected tenant label %q, got %q", "custom/tenant", config.TenantLabelKey)
	}
}

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name:        "valid tenant mode config",
			config:      DefaultConfig(),
			expectError: false,
		},
		{
			name: "valid namespace mode config",
			config: &Config{
				IsolationMode:         IsolationModeNamespace,
				WhitelistedNamespaces: []string{},
				TenantLabelKey:        "test/label",
				ClusterDomain:         "cluster.local",
			},
			expectError: false,
		},
		{
			name: "invalid isolation mode",
			config: &Config{
				IsolationMode:  "invalid",
				TenantLabelKey: "test/label",
				ClusterDomain:  "cluster.local",
			},
			expectError: true,
		},
		{
			name: "empty tenant label",
			config: &Config{
				IsolationMode:  IsolationModeTenant,
				TenantLabelKey: "",
				ClusterDomain:  "cluster.local",
			},
			expectError: true,
		},
		{
			name: "empty cluster domain",
			config: &Config{
				IsolationMode:  IsolationModeTenant,
				TenantLabelKey: "test/label",
				ClusterDomain:  "",
			},
			expectError: true,
		},
		{
			name: "invalid whitelist pattern",
			config: &Config{
				IsolationMode:         IsolationModeTenant,
				TenantLabelKey:        "test/label",
				ClusterDomain:         "cluster.local",
				WhitelistedNamespaces: []string{"[invalid"},
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.config.Validate()
			if tc.expectError && err == nil {
				t.Error("expected error, got nil")
			}

			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()

	if config.IsolationMode != IsolationModeTenant {
		t.Errorf("expected default isolation mode %q, got %q", IsolationModeTenant, config.IsolationMode)
	}

	if len(config.WhitelistedNamespaces) != 2 {
		t.Errorf("expected 2 default whitelisted namespaces, got %d", len(config.WhitelistedNamespaces))
	}

	if config.TenantLabelKey != "capsule.clastix.io/tenant" {
		t.Errorf("expected default tenant label key, got %q", config.TenantLabelKey)
	}

	// Validate should pass for default config
	if err := config.Validate(); err != nil {
		t.Errorf("default config validation failed: %v", err)
	}
}

func TestConfigString(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	str := config.String()

	if str == "" {
		t.Error("expected non-empty config string")
	}

	// Should contain key information
	if len(str) < 50 {
		t.Errorf("config string seems too short: %s", str)
	}
}

func TestParseList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{"a b c", []string{"a", "b", "c"}},
		{"a, b, c", []string{"a", "b", "c"}},
		{"  a  ,  b  ,  c  ", []string{"a", "b", "c"}},
		{"single", []string{"single"}},
		{"", []string(nil)},
		{"  ", []string(nil)},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			result := parseList(tc.input)

			if len(result) != len(tc.expected) {
				t.Errorf("expected %d items, got %d", len(tc.expected), len(result))
				return
			}

			for i, item := range tc.expected {
				if result[i] != item {
					t.Errorf("expected result[%d]=%q, got %q", i, item, result[i])
				}
			}
		})
	}
}
