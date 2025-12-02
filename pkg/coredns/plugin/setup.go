// Copyright 2020-2021 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"strconv"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

const (
	// PluginName is the name of the plugin as registered with CoreDNS.
	PluginName = "capsule"

	// IsolationModeTenant allows resolution within the same tenant.
	IsolationModeTenant = "tenant"

	// IsolationModeNamespace allows resolution only within the same namespace.
	IsolationModeNamespace = "namespace"

	// DefaultClusterDomain is the default Kubernetes cluster domain.
	DefaultClusterDomain = "cluster.local"
)

func init() {
	plugin.Register(PluginName, setup)
}

// setup is the setup function for the Capsule plugin.
func setup(c *caddy.Controller) error {
	capsule, err := parseConfig(c)
	if err != nil {
		return plugin.Error(PluginName, err)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		capsule.Next = next

		return capsule
	})

	return nil
}

// parseConfig parses the Capsule plugin configuration from the Corefile.
func parseConfig(c *caddy.Controller) (*Capsule, error) {
	capsule := &Capsule{
		IsolationMode:         IsolationModeTenant,
		WhitelistedNamespaces: []string{"default", "kube-system"},
		ClusterDomain:         DefaultClusterDomain,
		TenantResolver:        nil,
	}

	for c.Next() {
		// Parse optional arguments on the same line
		args := c.RemainingArgs()
		if len(args) > 0 {
			capsule.IsolationMode = args[0]
		}

		// Parse block configuration
		for c.NextBlock() {
			switch c.Val() {
			case "isolation":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				capsule.IsolationMode = c.Val()
			case "whitelist":
				capsule.WhitelistedNamespaces = c.RemainingArgs()
			case "cluster_domain":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				capsule.ClusterDomain = c.Val()
			case "kubernetes":
				// Parse kubernetes configuration for tenant resolution
				resolver, err := parseKubernetesConfig(c)
				if err != nil {
					return nil, err
				}
				capsule.TenantResolver = resolver
			default:
				return nil, c.Errf("unknown property '%s'", c.Val())
			}
		}
	}

	return capsule, nil
}

// parseKubernetesConfig parses the kubernetes resolver configuration.
func parseKubernetesConfig(c *caddy.Controller) (*KubernetesTenantResolver, error) {
	resolver := &KubernetesTenantResolver{
		TenantLabel: DefaultTenantLabel,
		TTL:         DefaultCacheTTL,
	}

	args := c.RemainingArgs()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "tenant_label":
			if i+1 >= len(args) {
				return nil, c.Errf("tenant_label requires a value")
			}
			i++
			resolver.TenantLabel = args[i]
		case "cache_ttl":
			if i+1 >= len(args) {
				return nil, c.Errf("cache_ttl requires a value")
			}
			i++
			ttl, err := strconv.Atoi(args[i])
			if err != nil {
				return nil, c.Errf("invalid cache_ttl value: %s", args[i])
			}
			resolver.TTL = ttl
		}
	}

	return resolver, nil
}
