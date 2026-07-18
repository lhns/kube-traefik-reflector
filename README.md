# kube-traefik-reflector

A small Kubernetes controller that **mirrors Traefik "policy" CRDs across namespaces**, driven by an
annotation on the source object. It lets a router in namespace `X` reference a `Middleware` (or
`TLSOption` / `ServersTransport`) that is *defined* in namespace `Y` — **without** turning on Traefik's
cluster-wide `providers.kubernetesCRD.allowCrossNamespace` or maintaining a global
`providers.kubernetesIngress.crossProviderNamespaces` list.

Instead, the **owner of the resource** declares which namespaces may use it (default-deny), and the
controller copies it there and keeps the copies in sync — so every router can reference the resource
**locally** and the global cross-namespace switches can be turned off.

## What it reflects

Only the Traefik CRDs that are *referenced by routers from other namespaces* (routers themselves are
never reflected — nothing references them):

| CRD (`traefik.io/v1alpha1`) | Reflected |
|---|---|
| `Middleware`, `MiddlewareTCP` | yes |
| `TLSOption` | yes |
| `ServersTransport`, `ServersTransportTCP` | yes |

## Usage

Annotate the source object with the namespaces allowed to use it:

```yaml
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: authelia
  namespace: authelia
  annotations:
    # explicit list, or "*" for all namespaces
    traefik-reflector/namespaces: "duckdb,dashy,navidrome"
    # and/or a label selector over namespace labels
    # traefik-reflector/namespace-selector: "team=web"
spec:
  forwardAuth:
    address: http://10.1.2.10:9091/api/authz/forward-auth
```

A copy named `authelia` appears in each selected namespace; routers there reference it locally
(`authelia@kubernetescrd`). Change the annotation and the copies follow; delete the source and they are
removed.

**Exclusions.** `*` / selector expansion skips a configurable system set (`kube-system`, `kube-public`,
`kube-node-lease`), the source's own namespace, terminating namespaces, and any namespace labelled
`traefik-reflector/exclude: "true"`. An **explicitly named** namespace is always honored.

**Secrets are not copied.** Some Traefik resources reference Secrets by name (e.g.
`basicAuth.secret`, `serversTransport.certificatesSecrets`). This controller does **not** reflect
Secrets — pair it with a secret reflector (e.g. [emberstack/reflector](https://github.com/emberstack/kubernetes-reflector))
for those. If a referenced Secret is missing in a target namespace, the controller emits a warning
Event on the source and a `traefik_reflector_missing_secret_total` metric, but still creates the mirror.

## How copies are tracked

Every mirror is stamped with a private `traefik-reflector.system/managed-by=traefik-reflector` label
(the only key the controller ever garbage-collects on) plus `source-namespace` / `source-name` labels
back to its origin. Cleanup is by label sweep (no finalizers) with a periodic orphan sweep. See
[`docs/adr`](docs/adr) for the design decisions.

## License

[Apache-2.0](LICENSE).
