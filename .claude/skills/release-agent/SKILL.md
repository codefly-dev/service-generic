---
name: release-agent
description: Cut a service-generic release, and keep the manifest version, the git tag, and the published archive names in the one shape Core's downloader resolves against. Use when releasing (`codefly publish`), repinning codefly-dev/core, editing .goreleaser.yaml or .github/workflows/releaser.yml, or when an agent install fails to find its download URL or a release job fails to build.
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

**`codefly publish patch` is the release** (or `minor` / `major`). It owns every
step — the version bump in `agent.codefly.yaml`, the `release: vX` commit on
`main`, the `vX` tag on that commit, and handing off to the workflow the
manifest names (`release: {owner: workflow, workflow: releaser.yml}`) — and the
steps are not separable by hand. Run it from a clean, up-to-date clone of `main`:

```bash
codefly publish patch --dry-run   # shows the bump and tag, changes nothing
codefly publish patch
```

**Never bump `agent.codefly.yaml` by hand, and never push the tag yourself.**
The version and the tag are one fact, and `publish` keeps them one; a hand bump
tagged separately is exactly the drift the table above warns about. v0.0.36
through v0.0.40 were cut this way (`release: v0.0.40` is b8c8931, tagged
`v0.0.40`).

A Core re-pin is its own PR, landed through CI (`go build`, `go vet`,
`go test`) before the release — e.g. #76 (core v0.19.4) and #77 (Go 1.27.2), then `release: v0.0.40`.

`publish` runs its pre-release gates (including govulncheck on the agent it
builds) and the release workflow re-runs `go test -v ./...` before building. A
red gate or suite stops the release; do not tag around it. If `publish` stopped
after the release commit but before the tag, re-run it — it finishes the
untagged release instead of bumping again.

## Why the build is not a plain `go build`

Release builds set `CGO_ENABLED=1` and cross-compile inside
`ghcr.io/goreleaser/goreleaser-cross`, which supplies the Linux and macOS
compilers for all four targets (darwin/linux × amd64/arm64). Core's
authoritative source inspection uses real tree-sitter grammars, so disabling CGO
would silently ship a binary whose production parser cannot work.

A local `go build ./...` uses your own toolchain defaults and proves the source
compiles. It does not rehearse the release.

## When a release fails

- **Job never ran.** `codefly publish` stopped before the tag (re-run it), or the
  tag did not match `v*`. Check `git ls-remote --tags origin`.
- **Job failed at the release step.** It needs `secrets.GH_PAT`; the built-in
  token is not what this workflow uses.
- **Published, but installs fail.** Compare the three names in the table above
  first — `gh release view v<version> --json assets` against the manifest's
  `version:`. This is the failure mode with no CI signal.
- **Archive names look wrong.** `name_template` in `.goreleaser.yaml` is pinned
  to Core's downloader contract. It is not a formatting preference; changing it
  breaks every consumer's resolution. If a new shape is genuinely needed, the
  change belongs in Core's downloader and this repo follows.
