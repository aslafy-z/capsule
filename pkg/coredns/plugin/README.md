# Capsule CoreDNS Plugin

A CoreDNS plugin that enforces DNS resolution boundaries based on Capsule tenant/namespace isolation policies.

## Description

This plugin allows you to control DNS resolution within a Kubernetes cluster based on Capsule multi-tenancy policies. It supports:

- **Tenant Isolation**: DNS resolution is allowed only within the same Capsule tenant
- **Namespace Isolation**: DNS resolution is allowed only within the same namespace
- **Whitelisting**: Specific namespaces (e.g., `default`, `kube-system`) can be whitelisted to allow cross-tenant/namespace resolution

## Syntax

```
capsule [ISOLATION_MODE] {
    isolation ISOLATION_MODE
    whitelist NAMESPACE [NAMESPACE ...]
    cluster_domain DOMAIN
    kubernetes [tenant_label LABEL] [cache_ttl TTL]
}
```

- **ISOLATION_MODE**: Either `tenant` (default) or `namespace`
- **whitelist**: List of namespaces that are always resolvable from any namespace
- **cluster_domain**: The cluster's DNS domain (default: `cluster.local`)
- **kubernetes**: Configure Kubernetes-based tenant resolution
  - **tenant_label**: Label used to identify tenant namespaces (default: `capsule.clastix.io/tenant`)
  - **cache_ttl**: Cache TTL in seconds (default: `30`)

## Examples

### Basic Configuration (Tenant Isolation)

```
.:53 {
    errors
    health
    ready
    capsule {
        isolation tenant
        whitelist default kube-system
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

### Namespace Isolation

```
.:53 {
    errors
    health
    ready
    capsule namespace {
        whitelist default kube-system monitoring
        cluster_domain my.cluster.local
    }
    kubernetes my.cluster.local in-addr.arpa ip6.arpa
    forward . /etc/resolv.conf
}
```

### Full Configuration with Kubernetes Resolver

```
.:53 {
    errors
    health
    ready
    capsule {
        isolation tenant
        whitelist default kube-system kube-public
        cluster_domain cluster.local
        kubernetes tenant_label capsule.clastix.io/tenant cache_ttl 60
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

## How It Works

1. The plugin intercepts DNS queries before they reach the Kubernetes plugin
2. It extracts the source namespace and tenant from request metadata (provided by the metadata plugin)
3. It parses the target namespace from the DNS query (e.g., `my-service.target-namespace.svc.cluster.local`)
4. It checks if the resolution is allowed based on the configured isolation mode:
   - If the target namespace is whitelisted, allow
   - If source and target are the same namespace, allow
   - In `namespace` mode, block cross-namespace resolution
   - In `tenant` mode, allow only if both namespaces belong to the same tenant
5. If blocked, return NXDOMAIN; otherwise, pass to the next plugin

## Requirements

- This plugin should be chained before the `kubernetes` plugin in your Corefile
- The `metadata` plugin should be configured to provide source namespace/tenant information
- For tenant isolation mode, namespaces should be labeled with the tenant label (default: `capsule.clastix.io/tenant`)

## Building

To include this plugin in your CoreDNS build, add it to the `plugin.cfg` file:

```
capsule:github.com/clastix/capsule/pkg/coredns/plugin
```

Then rebuild CoreDNS:

```bash
go generate
go build
```

## Metrics

The plugin does not currently expose any metrics.

## See Also

- [Capsule](https://capsule.clastix.io/) - Multi-tenancy and Policy-Based framework for Kubernetes
- [CoreDNS](https://coredns.io/) - DNS and Service Discovery
- [CoreDNS Kubernetes Plugin](https://coredns.io/plugins/kubernetes/)
