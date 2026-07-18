# 0006 — Server-side apply and same-namespace-only owner-refs

Status: Accepted

## Context

The controller writes a mirror repeatedly and must converge it to the source's spec
without fighting other writers or leaving stale fields. It also wants Kubernetes garbage
collection to help where it can.

## Decision

Write mirrors with **server-side apply** using a fixed field manager
(`FieldOwner("traefik-reflector")`, force ownership). Set an owner-reference on a mirror
**only** when the mirror is in the same namespace as its source (which, in practice, does
not happen because the source's own namespace is always skipped) — cross-namespace
owner-references are illegal, which is precisely why cross-namespace cleanup is done by
the label sweep of ADR 0003 instead.

## Consequences

- **Pro**: Apply converges the mirror to the desired spec and removes fields the
  controller no longer sets, without read-modify-write races.
- **Pro**: A fixed field manager gives clear ownership of the fields the controller
  manages, coexisting with any other managers.
- **Con**: Server-side apply requires reconstructing the intended object each time (which
  the controller does in `buildMirror`); there is no partial patch.
