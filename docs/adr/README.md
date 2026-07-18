# Architecture Decision Records

These record the *why* behind the design. Format is MADR-lite: a single `Status:`
line, then `## Context`, `## Decision`, `## Consequences`. ADRs are immutable once
accepted — if a decision is reversed, a new ADR supersedes the old one.

Template:

```
# NNNN — Short title
Status: Accepted | Superseded by NNNN | Deprecated

## Context
Why we needed to decide.

## Decision
What we decided.

## Consequences
What follows — the wins and the costs.
```

## Index

- [0001 — Reflect only Traefik "policy" CRDs, not routers](0001-scope-policy-crds-not-routers.md)
- [0002 — Unstructured objects + a per-GVK config table](0002-unstructured-per-gvk-table.md)
- [0003 — Ownership markers + label-sweep GC, no finalizers](0003-ownership-label-sweep-no-finalizers.md)
- [0004 — Eager reflection with namespace exclusions](0004-eager-with-namespace-exclusions.md)
- [0005 — Namespace targeting: list, wildcard, label selector](0005-namespace-targeting.md)
- [0006 — Server-side apply and same-namespace-only owner-refs](0006-ssa-and-owner-refs.md)
- [0007 — Secrets are not reflected; detect and warn](0007-secrets-delegated-detect-warn.md)
- [0008 — Conflict guard: never clobber unmanaged resources](0008-conflict-guard.md)
- [0009 — Distribution: multi-arch image + OCI Helm chart](0009-distribution.md)
