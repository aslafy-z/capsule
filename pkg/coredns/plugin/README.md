# Capsule CoreDNS Plugin

A CoreDNS plugin that enforces DNS resolution boundaries based on Capsule tenant/namespace isolation policies.

## Description

This plugin allows you to control DNS resolution within a Kubernetes cluster based on Capsule multi-tenancy policies. It supports:

- **Tenant Isolation**: DNS resolution is allowed only within the same Capsule tenant
- **Namespace Isolation**: DNS resolution is allowed only within the same namespace
- **Whitelisting**: Specific namespaces (e.g., `default`, `kube-system`) can be whitelisted to allow cross-tenant/namespace resolution

## How It Works

The plugin integrates directly with Capsule by:

1. Querying the Kubernetes API to retrieve namespace labels
2. Extracting the tenant name from the `capsule.clastix.io/tenant` label (configurable)
3. Caching tenant lookups to minimize API calls
4. Enforcing DNS isolation based on tenant/namespace boundaries

When a DNS query is received:
1. The plugin extracts the source namespace and tenant from request metadata
2. It parses the target namespace from the DNS query (e.g., `my-service.target-namespace.svc.cluster.local`)
3. It checks if resolution is allowed:
   - If the target namespace is whitelisted → allow
   - If source and target are the same namespace → allow
   - In `namespace` mode → block cross-namespace resolution
   - In `tenant` mode → allow only if both namespaces belong to the same tenant (via API lookup)
4. If blocked, return NXDOMAIN; otherwise, pass to the next plugin

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
        kubernetes
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

### Full Configuration with Custom Tenant Label

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

## Capsule Integration

This plugin is designed to work seamlessly with Capsule. When Capsule creates a namespace for a tenant, it automatically labels the namespace with `capsule.clastix.io/tenant=<tenant-name>`. The plugin queries these labels via the Kubernetes API to determine which tenant a namespace belongs to.

### Prerequisites

1. Capsule must be installed and configured in your cluster
2. Namespaces should be created through Capsule (so they get the tenant label)
3. CoreDNS must have RBAC permissions to read namespaces:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: coredns-capsule
rules:
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: coredns-capsule
subjects:
- kind: ServiceAccount
  name: coredns
  namespace: kube-system
roleRef:
  kind: ClusterRole
  name: coredns-capsule
  apiGroup: rbac.authorization.k8s.io
```

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
