# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/), and this project adheres to
[Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.1] - 2026-10-22
### Security
- Dependencies: `golang.org/x/net` v0.49.0 to v0.60.0 (CVE-2026-78669) and
  `golang.org/x/text` v0.33.0 to v0.42.0 (CVE-2026-56851). `go mod tidy` also
  moved `x/sync`, `x/sys` and `x/term` (#1).
- Rebuilt on current base images. The image uses
  `gcr.io/distroless/static:nonroot` (Debian 13), whose CA bundle drops the
  emSign roots compromised in the October 2026 eMudhra intermediate CA key
  compromise, and a Go 1.26 toolchain of 1.26.10 or later, which also rejects
  the eight compromised keys in `crypto/x509`.

There are no functional changes.

## [0.1.0] - 2026-07-19
### Added
- Initial implementation. A controller that mirrors Traefik "policy" CRDs
  (`Middleware`, `MiddlewareTCP`, `TLSOption`, `ServersTransport`,
  `ServersTransportTCP`) into other namespaces based on an annotation on the
  source object, so routers can reference them **locally** and Traefik's global
  `allowCrossNamespace` / `crossProviderNamespaces` can be turned off.
- Per-resource, default-deny target selection: `traefik-reflector/namespaces`
  (a list or `*`) and `traefik-reflector/namespace-selector` (a namespace label
  selector), with configurable system-namespace exclusions and a per-namespace
  `traefik-reflector/exclude` opt-out.
- Label-sweep garbage collection (no finalizers) plus a periodic orphan sweep.
- Secret-reference detection: secrets are **not** copied (delegate to a secret
  reflector); a warning Event + metric is emitted if a referenced Secret is
  missing in a target namespace.
