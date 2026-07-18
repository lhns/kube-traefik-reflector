# 0002 — Unstructured objects + a per-GVK config table

Status: Accepted

## Context

The controller mirrors five third-party Traefik CRDs. It could vendor Traefik's Go
API types and use typed clients, or operate on `unstructured.Unstructured` driven by
a small table of GVKs. The controller never interprets a resource's `spec` beyond
copying it and locating Secret-name fields.

## Decision

Operate entirely on `unstructured.Unstructured`, with an authoritative
`ReflectedTypes` table (`gvks.go`) that lists each GVK and the JSON paths within it
that reference Secret names. One generic reconciler is instantiated per GVK.

## Consequences

- **Pro**: No dependency on Traefik's Go module and no coupling to a specific Traefik
  version; the controller keeps working across CRD schema changes as long as the
  copied fields exist.
- **Pro**: Adding a CRD (or a new secret-bearing field) is a one-line table change,
  not new types + regenerated deepcopy.
- **Pro**: The generic reconciler is written once and reused for all five kinds.
- **Con**: No compile-time schema validation of the fields we read; the Secret-path
  table must be kept correct by unit tests and reviewed when the supported Traefik CRD
  version changes.
