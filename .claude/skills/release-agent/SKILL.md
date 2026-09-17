---
name: release-agent
description: Cut a service-generic release, and keep the manifest version, the git tag, and the published archive names in the one shape Core's downloader resolves against. Use when bumping the version in agent.codefly.yaml, repinning codefly-dev/core, tagging a release, editing .goreleaser.yaml or .github/workflows/releaser.yml, or when an agent install fails to find its download URL or a release job fails to build.
---

# Releasing service-generic

Three names must agree, or the binary builds and publishes successfully and is
still unreachable:

| Where | Value for the 0.0.35 release |
| --- | --- |
| `agent.codefly.yaml` → `version:` | `0.0.35` |
| git tag | `v0.0.35` |
| published archives | `service-generic_0.0.35_{os}_{arch}.tar.gz` |

Core resolves an agent download URL from the manifest version
(`agents/manager/downloader_url.go`) as
`service-{name}_{version}_{os}_{arch}.tar.gz`, with the binary inside named
`service-{name}`. GoReleaser derives `{{ .Version }}` from **the tag**, not from
the manifest. So a manifest that says `0.0.36` under a tag `v0.0.35` publishes a
complete, green, correctly-signed release that Core asks for at a URL which does
not exist. Nothing in CI cross-checks the two — that is what this file is for.

## The walk

1. Bump `version:` in `agent.codefly.yaml`. If the release also moves the Core
   pin, bump `go.mod` and run `go mod tidy` in the same commit — the two have
   historically travelled together (`release: service-generic 0.0.35 on core
   0.3.29`).
2. Open a PR, land it on `main` through CI (`go build`, `go vet`, `go test`).
3. Tag the merge commit `v<version>`, exactly matching the manifest, and push
   the tag.

Step 3 is what publishes. `.github/workflows/releaser.yml` triggers only on a
pushed `v*` tag; **nothing in this repository creates that tag**, so it comes
from outside CI. Tag the commit that carries the bumped manifest — past releases
tag the merge commit on `main`, so the released tree is the reviewed one.

The release job re-runs `go test -v ./...` before building. A red suite stops the
release; do not tag around it.

## Why the build is not a plain `go build`

Release builds set `CGO_ENABLED=1` and cross-compile inside
`ghcr.io/goreleaser/goreleaser-cross`, which supplies the Linux and macOS
compilers for all four targets (darwin/linux × amd64/arm64). Core's
authoritative source inspection uses real tree-sitter grammars, so disabling CGO
would silently ship a binary whose production parser cannot work.

A local `go build ./...` uses your own toolchain defaults and proves the source
compiles. It does not rehearse the release.

## When a release fails

- **Job never ran.** The tag did not match `v*`, or it was created without being
  pushed. Check `git ls-remote --tags origin`.
- **Job failed at the release step.** It needs `secrets.GH_PAT`; the built-in
  token is not what this workflow uses.
- **Published, but installs fail.** Compare the three names in the table above
  first — `gh release view v<version> --json assets` against the manifest's
  `version:`. This is the failure mode with no CI signal.
- **Archive names look wrong.** `name_template` in `.goreleaser.yaml` is pinned
  to Core's downloader contract. It is not a formatting preference; changing it
  breaks every consumer's resolution. If a new shape is genuinely needed, the
  change belongs in Core's downloader and this repo follows.
