# 0009 — Distribution: multi-arch image + OCI Helm chart

Status: Accepted

## Context

The controller needs a distribution path consistent with the sibling projects
(`kube-vnet`, `kube-pv-reaper`): a container image and a Helm chart (the real install
path), released reproducibly.

## Decision

- Build a **multi-arch** (`linux/amd64,linux/arm64`) image and push it to **GHCR**
  (`ghcr.io/lhns/kube-traefik-reflector`), cross-compiled on the native toolchain from a
  distroless nonroot base.
- Publish the Helm chart as an **OCI artifact** to `oci://ghcr.io/lhns/charts`.
- Version from git tags: `vX.Y.Z` → image `:X.Y.Z` + `:latest` and chart version `X.Y.Z`;
  a push to a branch → image `:sha-<short>` and a dev chart version. Releases are
  **CHANGELOG-driven** — the GitHub Release body is the matching `## [X.Y.Z]` section.

## Consequences

- **Pro**: One tag produces a coherent, reproducible image + chart + release notes.
- **Pro**: Matches the operational conventions of the sibling repos, so users install it the
  same way.
- **Con**: The `CHANGELOG.md` must carry an entry per released version or the release notes
  fall back to auto-generated text.
