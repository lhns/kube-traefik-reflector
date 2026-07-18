# 0005 — Namespace targeting: list, wildcard, label selector

Status: Accepted

## Context

A source needs to express which namespaces may use it. Options include an explicit list,
a wildcard for "all", glob patterns on namespace names, and a label selector over
namespace labels.

## Decision

Support three, via two annotations on the source:

- `traefik-reflector/namespaces`: a comma-separated list of namespace names, or `*` for
  all. Absent means the object is not reflected (default-deny).
- `traefik-reflector/namespace-selector`: a standard Kubernetes label selector matched
  against **namespace labels**.

The effective target set is the union of the list and the selector matches, minus the
exclusions from ADR 0004. Name globs are intentionally **not** supported.

## Consequences

- **Pro**: Label selectors are the idiomatic Kubernetes way to target namespaces (as in
  `NetworkPolicy.namespaceSelector`), and compose with existing namespace-labelling
  conventions.
- **Pro**: An explicit list covers the common small cases with zero ceremony; `*` covers
  "everywhere".
- **Con**: Teams that identify namespaces by a name pattern (`team-*`) must add a shared
  label instead. This keeps matching consistent with the rest of Kubernetes and avoids a
  bespoke glob dialect.
