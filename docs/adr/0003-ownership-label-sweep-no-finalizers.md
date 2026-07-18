# 0003 — Ownership markers + label-sweep GC, no finalizers

Status: Accepted

## Context

The controller creates copies (mirrors) of a source object in other namespaces and
must, later, find exactly those copies to update or delete them — including copies in
namespaces the source no longer targets, and copies left behind when the source is
deleted. Kubernetes owner-references cannot cross namespaces, so `.Owns()`-style
garbage collection does not work here. A finalizer on the source would guarantee
cleanup but blocks source deletion whenever the controller is down and needs an
uninstall hook to avoid wedging objects in `Terminating`.

## Decision

Mark every mirror and garbage-collect by label sweep; use no finalizers.

- Private sentinel label `traefik-reflector.system/managed-by=traefik-reflector` — the
  **only** key the controller ever selects, sweeps, or deletes on.
- `traefik-reflector.system/source-namespace` / `source-name` back-reference labels
  for reverse-mapping and selection (with a truncate+hash fallback for names over 63
  chars), plus a `reflected-from` annotation holding the exact `<ns>/<name>` origin and
  a `source-uid` annotation.
- The public `app.kubernetes.io/managed-by` label is set for humans/tooling but is
  **never** selected or deleted on (users and Helm write it too).
- Cleanup: on each source reconcile, list its mirrors by the sentinel + source labels
  and delete any not in the desired namespace set; on source deletion, delete them all.
- A periodic **orphan sweep** (leader-only) deletes any mirror whose source no longer
  exists — the backstop for a deletion missed while the controller was down.

## Consequences

- **Pro**: No finalizers means a source can always be deleted, and uninstalling the
  controller never strands objects in `Terminating`.
- **Pro**: Cross-namespace GC works uniformly (it is label-driven, not owner-ref-driven).
- **Pro**: Gating deletion on a private, admission-free-of-user-writes sentinel avoids
  ever deleting something a user or Helm merely labelled `managed-by`.
- **Con**: A source deleted while the controller is down leaves orphans until the next
  orphan sweep, rather than being cleaned synchronously.
