// Copyright 2020-2021 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package coredns

import (
	"fmt"
	"strings"
)

// ParseConfig parses plugin configuration from a key-value map.
// This is used to parse configuration from CoreDNS Corefile format.
func ParseConfig(options map[string]string) (*Config, error) {
	config := DefaultConfig()

	for key, value := range options {
		switch strings.ToLower(key) {
		case "isolation_mode", "isolationmode":
			mode := IsolationMode(strings.ToLower(value))
			if mode != IsolationModeTenant && mode != IsolationModeNamespace {
				return nil, fmt.Errorf("invalid isolation mode: %s (expected 'tenant' or 'namespace')", value)
			}

			config.IsolationMode = mode

		case "whitelist", "whitelisted_namespaces", "whitelistednamespaces":
			// Parse comma-separated list of namespaces
			namespaces := parseList(value)
			if len(namespaces) > 0 {
				config.WhitelistedNamespaces = namespaces
			}

		case "tenant_label", "tenantlabel", "tenant_label_key":
			if value != "" {
				config.TenantLabelKey = value
			}

		default:
			// Unknown option - ignore or warn
			continue
		}
	}

	return config, nil
}

// parseList parses a comma or space separated list of values.
func parseList(value string) []string {
	var result []string

	// Replace commas with spaces for uniform splitting
	value = strings.ReplaceAll(value, ",", " ")

	for _, item := range strings.Fields(value) {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}

	return result
}

// ConfigBuilder provides a fluent interface for building plugin configuration.
type ConfigBuilder struct {
	config *Config
}

// NewConfigBuilder creates a new ConfigBuilder with default configuration.
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: DefaultConfig(),
	}
}

// WithIsolationMode sets the isolation mode.
func (b *ConfigBuilder) WithIsolationMode(mode IsolationMode) *ConfigBuilder {
	b.config.IsolationMode = mode

	return b
}

// WithWhitelistedNamespaces sets the whitelisted namespaces.
func (b *ConfigBuilder) WithWhitelistedNamespaces(namespaces ...string) *ConfigBuilder {
	b.config.WhitelistedNamespaces = namespaces

	return b
}

// AddWhitelistedNamespace adds a namespace to the whitelist.
func (b *ConfigBuilder) AddWhitelistedNamespace(namespace string) *ConfigBuilder {
	b.config.WhitelistedNamespaces = append(b.config.WhitelistedNamespaces, namespace)

	return b
}

// WithTenantLabelKey sets the tenant label key.
func (b *ConfigBuilder) WithTenantLabelKey(key string) *ConfigBuilder {
	b.config.TenantLabelKey = key

	return b
}

// Build returns the configured Config.
func (b *ConfigBuilder) Build() *Config {
	return b.config
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.IsolationMode != IsolationModeTenant && c.IsolationMode != IsolationModeNamespace {
		return fmt.Errorf("invalid isolation mode: %s", c.IsolationMode)
	}

	if c.TenantLabelKey == "" {
		return fmt.Errorf("tenant label key cannot be empty")
	}

	return nil
}

// String returns a human-readable representation of the config.
func (c *Config) String() string {
	return fmt.Sprintf("Config{IsolationMode: %s, WhitelistedNamespaces: %v, TenantLabelKey: %s}",
		c.IsolationMode, c.WhitelistedNamespaces, c.TenantLabelKey)
}
