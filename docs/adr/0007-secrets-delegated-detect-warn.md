# 0007 — Secrets are not reflected; detect and warn

Status: Accepted

## Context

Some of the reflected CRDs reference Kubernetes Secrets by name — for example a
`Middleware` `basicAuth.secret` or `forwardAuth.tls.certSecret`, a `TLSOption`
`clientAuth.secretNames`, or a `ServersTransport` `certificatesSecrets`. When such a
resource is copied to another namespace, those Secret references dangle unless the Secret
also exists there. Copying Secrets is a distinct, well-solved problem
(e.g. [emberstack/reflector](https://github.com/emberstack/kubernetes-reflector)), and
handling Secret data would greatly widen this controller's blast radius.

## Decision

Do **not** reflect Secrets. Instead, per the `ReflectedTypes` table, extract the Secret
names a source references and, for each target namespace, check whether each Secret
exists there. If one is missing, emit a warning `Event` on the source and increment
`traefik_reflector_missing_secret_total`, but still create the mirror (so it starts
working the moment the Secret appears). Existence is checked with a **metadata-only** read
(`PartialObjectMetadata`), so the controller never reads Secret data.

## Consequences

- **Pro**: Clean separation of concerns; pair with any secret reflector for the Secret
  half. The controller's Secret access is read-metadata-only in code.
- **Pro**: A missing Secret is surfaced loudly instead of failing silently at request time
  inside Traefik.
- **Con**: The RBAC grant still lists `secrets: get,list,watch` (RBAC cannot express
  "metadata only"), so the ServiceAccount is technically capable of reading Secret data
  even though the code does not. This is documented and much narrower than a Secret-copying
  controller's cluster-wide read+write.
- **Con**: The Secret-path table must be kept in step with the supported Traefik CRD
  version (guarded by unit tests).
