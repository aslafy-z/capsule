// Copyright 2020-2021 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"testing"

	"github.com/coredns/caddy"
	"github.com/stretchr/testify/assert"
)

func TestParseConfig_Defaults(t *testing.T) {
	// Test default configuration
	c := caddy.NewTestController("dns", "capsule")
	capsule, err := parseConfig(c)

	assert.NoError(t, err)
	assert.NotNil(t, capsule)
	assert.Equal(t, IsolationModeTenant, capsule.IsolationMode)
	assert.Equal(t, DefaultClusterDomain, capsule.ClusterDomain)
	assert.Contains(t, capsule.WhitelistedNamespaces, "default")
	assert.Contains(t, capsule.WhitelistedNamespaces, "kube-system")
}

func TestParseConfig_IsolationModeArg(t *testing.T) {
	// Test isolation mode specified as argument
	c := caddy.NewTestController("dns", "capsule namespace")
	capsule, err := parseConfig(c)

	assert.NoError(t, err)
	assert.NotNil(t, capsule)
	assert.Equal(t, "namespace", capsule.IsolationMode)
}

func TestParseConfig_Block(t *testing.T) {
	// Test block configuration
	config := `capsule {
		isolation namespace
		whitelist default kube-system monitoring
		cluster_domain my.cluster.local
	}`
	c := caddy.NewTestController("dns", config)
	capsule, err := parseConfig(c)

	assert.NoError(t, err)
	assert.NotNil(t, capsule)
	assert.Equal(t, IsolationModeNamespace, capsule.IsolationMode)
	assert.Equal(t, "my.cluster.local", capsule.ClusterDomain)
	assert.Len(t, capsule.WhitelistedNamespaces, 3)
	assert.Contains(t, capsule.WhitelistedNamespaces, "default")
	assert.Contains(t, capsule.WhitelistedNamespaces, "kube-system")
	assert.Contains(t, capsule.WhitelistedNamespaces, "monitoring")
}

func TestParseConfig_KubernetesResolver(t *testing.T) {
	// Test kubernetes resolver configuration
	config := `capsule {
		isolation tenant
		kubernetes tenant_label my.label/tenant cache_ttl 60
	}`
	c := caddy.NewTestController("dns", config)
	capsule, err := parseConfig(c)

	assert.NoError(t, err)
	assert.NotNil(t, capsule)
	assert.NotNil(t, capsule.TenantResolver)

	resolver, ok := capsule.TenantResolver.(*KubernetesTenantResolver)
	assert.True(t, ok)
	assert.Equal(t, "my.label/tenant", resolver.TenantLabel)
	assert.Equal(t, 60, resolver.TTL)
}

func TestParseConfig_InvalidProperty(t *testing.T) {
	// Test invalid property
	config := `capsule {
		unknown_property value
	}`
	c := caddy.NewTestController("dns", config)
	_, err := parseConfig(c)

	assert.Error(t, err)
}

func TestParseConfig_MissingValue(t *testing.T) {
	// Test missing value for isolation
	config := `capsule {
		isolation
	}`
	c := caddy.NewTestController("dns", config)
	_, err := parseConfig(c)

	assert.Error(t, err)
}

func TestSetup(t *testing.T) {
	// Test setup function
	c := caddy.NewTestController("dns", "capsule")
	err := setup(c)
	assert.NoError(t, err)
}
