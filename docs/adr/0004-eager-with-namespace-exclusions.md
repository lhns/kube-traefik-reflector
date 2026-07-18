# 0004 — Eager reflection with namespace exclusions

Status: Accepted

## Context

When a source selects target namespaces, the controller can either copy into **all**
selected namespaces (eager) or only into namespaces where a router actually references
the resource (lazy). Lazy would need the controller to watch and parse every router
type (`IngressRoute`, `IngressRouteTCP`, native `Ingress` annotations) for references,
and it opens a race window: a router that references the resource reconciles before the
copy exists, so Traefik reports "middleware does not exist" until the controller
catches up. Eager avoids the race but could create copies in namespaces where they make
no sense (e.g. `kube-system`).

## Decision

Reflect **eagerly** into every selected namespace, with a sanity guard on expansion:

- Skip a configurable default-excluded set (`kube-system`, `kube-public`,
  `kube-node-lease`), the source's own namespace, terminating namespaces, and any
  namespace labelled `traefik-reflector/exclude: "true"`.
- These exclusions apply to `*` / selector expansion. An **explicitly named** namespace
  is honored anyway (explicit intent wins) — except the source's own namespace and
  terminating namespaces, which are always skipped.

## Consequences

- **Pro**: No ordering race; a router always finds its middleware already present.
- **Pro**: The owner controls scope explicitly (default-deny), and system namespaces are
  protected from accidental `*` blast radius.
- **Pro**: No need to watch or understand router resources at all.
- **Con**: A namespace listed by a source but where nothing references the resource still
  gets an unused copy — an accepted cost of the owner's explicit choice.
