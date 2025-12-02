# Capsule CoreDNS Plugin

The Capsule CoreDNS plugin provides DNS security and isolation for multi-tenant Kubernetes clusters. It enforces tenant/namespace-level DNS isolation and supports namespace whitelisting.

## Features

- **Tenant Isolation**: Pods can only resolve DNS names for services within their own tenant
- **Namespace Isolation**: Pods can only resolve DNS names for services within their own namespace
- **Namespace Whitelisting**: Configure specific namespaces that are always accessible regardless of isolation rules (e.g., `default`, `kube-system`)
- **Glob Pattern Support**: Whitelist patterns support glob matching (e.g., `kube-*`)

## Configuration

The plugin supports the following configuration options:

| Option | Description | Default |
|--------|-------------|---------|
| `isolation_mode` | Mode of isolation: `tenant` or `namespace` | `tenant` |
| `whitelist` | Comma or space-separated list of whitelisted namespace patterns | `default,kube-system` |
| `tenant_label` | Label key used to identify tenant membership | `capsule.clastix.io/tenant` |

## Isolation Modes

### Tenant Mode (`isolation_mode tenant`)

In tenant mode, pods can resolve DNS names for:
- Services/pods within any namespace belonging to the same tenant
- Services/pods in whitelisted namespaces

### Namespace Mode (`isolation_mode namespace`)

In namespace mode, pods can resolve DNS names for:
- Services/pods within their own namespace only
- Services/pods in whitelisted namespaces

## Usage

### Example Corefile Configuration

```
.:53 {
    errors
    health
    ready
    capsule {
        isolation_mode tenant
        whitelist default kube-system kube-public
        tenant_label capsule.clastix.io/tenant
    }
    kubernetes cluster.local in-addr.arpa ip6.arpa {
        pods insecure
        fallthrough in-addr.arpa ip6.arpa
    }
    forward . /etc/resolv.conf
    cache 30
    loop
    reload
    loadbalance
}
```

### Programmatic Usage

```go
package main

import (
    "github.com/clastix/capsule/pkg/coredns"
)

func main() {
    // Create configuration
    config := coredns.NewConfigBuilder().
        WithIsolationMode(coredns.IsolationModeTenant).
        WithWhitelistedNamespaces("default", "kube-system", "kube-*").
        WithTenantLabelKey("capsule.clastix.io/tenant").
        Build()

    // Create plugin with a tenant resolver
    plugin := coredns.NewPlugin(config, tenantResolver, nextHandler)

    // Use plugin to serve DNS
    // plugin.ServeDNS(ctx, request)
}
```

### Configuration from Map

```go
options := map[string]string{
    "isolation_mode": "tenant",
    "whitelist":      "default,kube-system,monitoring",
    "tenant_label":   "my.custom/tenant",
}

config, err := coredns.ParseConfig(options)
if err != nil {
    // handle error
}
```

## Integration

The plugin is designed to be chained with the standard `kubernetes` CoreDNS plugin. It intercepts DNS requests, checks the source pod's tenant/namespace membership, and either:
- Allows the request to pass through to the next handler
- Returns NXDOMAIN (Name Error) if access is denied

### Request Context

The plugin expects the following metadata to be available in DNS requests:
- `namespace`: The namespace of the requesting pod
- `<tenant_label>`: The tenant label value of the requesting pod

This metadata is typically provided by CoreDNS's metadata plugin or through integration with the Kubernetes API.

## Tenant Resolution

The plugin uses the `TenantResolver` interface to determine tenant membership:

```go
type TenantResolver interface {
    // GetTenantForNamespace returns the tenant name for a given namespace.
    GetTenantForNamespace(namespace string) (string, error)

    // GetNamespacesForTenant returns all namespaces belonging to a tenant.
    GetNamespacesForTenant(tenant string) ([]string, error)
}
```

Implement this interface to integrate with your tenant management system (e.g., Capsule's Tenant CRD).

## Security Considerations

- Denied queries return NXDOMAIN to avoid information leakage
- On resolver errors, access is denied by default for security
- External DNS queries (non-cluster queries) pass through without restriction
