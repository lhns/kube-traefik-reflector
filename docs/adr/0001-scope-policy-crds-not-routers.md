# 0001 — Reflect only Traefik "policy" CRDs, not routers

Status: Accepted

## Context

Traefik CRDs split into two roles. **Routers** (`IngressRoute`, `IngressRouteTCP`,
`IngressRouteUDP`) declare routes and *reference* other resources. **Policy objects**
(`Middleware`, `MiddlewareTCP`, `TLSOption`, `ServersTransport`, `ServersTransportTCP`)
are reusable configuration that routers point at. `TraefikService` and `TLSStore` are
special cases (a routing target usually co-located with its app; a mostly-singleton
default store).

The cross-namespace problem only exists for a resource that is *defined once and
referenced by routers in other namespaces*. A router is never referenced by anything,
so it has the reverse problem: it reaches *out* to policy objects elsewhere.

## Decision

Reflect only the policy CRDs: `Middleware`, `MiddlewareTCP`, `TLSOption`,
`ServersTransport`, `ServersTransportTCP`. Do not reflect routers, `TraefikService`,
or `TLSStore`.

## Consequences

- **Pro**: Reflection localizes the objects a router references, so the router's own
  namespace can reference them directly — which is exactly what removes the need for
  Traefik's global `allowCrossNamespace` / `crossProviderNamespaces`.
- **Pro**: Routers stay where their app is; there is nothing to mirror for them.
- **Con**: A user who wants a shared `TraefikService` or a cross-namespace `TLSStore`
  is not covered. Both are rare and out of scope; they can be added as table entries
  later if a real need appears.
