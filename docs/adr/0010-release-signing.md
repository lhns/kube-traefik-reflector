# 0010 — Release signing: cosign keyless, tagged releases only

Status: Accepted

## Context

Releases already signed the image keyless (GitHub OIDC) and attested its SBOM, but no ADR recorded
it, the chart was unsigned, and `id-token: write` was granted workflow-wide. The repository, image
and chart are public, so keyless signing leaks nothing, and the sibling operators sign the same
way. Dev builds now publish from every branch, and nothing verifies them.

## Decision

- A `vX.Y.Z` tag signs every image digest and the chart digest with cosign keyless and attests an
  SPDX image SBOM. The signing identity is the release workflow at that tag; there is no key.
- `id-token: write` is granted to the `release` job only; the workflow default is `contents: read`.
- Dev builds are published unsigned. Their SBOM is still generated, so that step stays exercised.

## Consequences

- **Pro**: Each release is verifiable with `cosign verify` against the workflow identity, with no
  key to store or rotate.
- **Pro**: Dev pushes add no public Rekor entries or `.sig`/`.att` tags.
- **Con**: A dev build cannot be verified; install a release where provenance matters.
