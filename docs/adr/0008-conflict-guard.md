# 0008 — Conflict guard: never clobber unmanaged resources

Status: Accepted

## Context

A target namespace might already contain a resource with the same name as the mirror the
controller wants to create — one a user authored themselves. Blindly applying the mirror
would overwrite it.

## Decision

Before applying a mirror, do an uncached read of the target object. If it exists and does
**not** carry the private managed-by sentinel, skip that namespace and emit a warning
`Conflict` event on the source. The controller only ever writes objects it already owns or
that do not yet exist.

## Consequences

- **Pro**: A user's own same-named resource is never silently overwritten.
- **Pro**: The uncached read closes the window where a just-created object is not yet in the
  informer cache.
- **Con**: One extra API read per target per reconcile. Reflected fan-out is small, so the
  cost is negligible.
- **Con**: If a user wants the controller to take over a pre-existing object, they must
  first remove it (or add the sentinel) — deliberate, to keep "don't clobber" the default.
