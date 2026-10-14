# Working in codefly-dev/service-generic

`github.com/codefly-dev/service-generic` is the **generic codefly agent**: one
`package main` binary that serves the Agent, Code, Runtime, and Tooling gRPC
contracts for *any* repository, in any language. It is the fallback every
language-specific agent falls back to, and it inherits its baseline filesystem,
git, search, and read-only inspection surface from Core rather than
reimplementing it.

It does **not** own: the contracts themselves (`codefly-dev/core`), the
`codefly` CLI, any language-specific agent, or Kubernetes manifests and their
transport. It also does not own build, test, or lint *of the attached project* —
it has no language knowledge, and says so through a typed failure instead of
guessing a command.

## How to behave

Fleet standard — [handbook#68](https://github.com/obin-ai/handbook/issues/68).
These bite hard here: almost every capability this repo has is supplied by Core,
so the standing temptation is to reimplement locally what is missing upstream.

- **A gap in the tooling is a bug in the tooling — never a reason to reach
  around it.** When Core's `DefaultCodeServer`, semantic analyzer, or resource
  loader cannot do what you need, the fix is a capability landed in
  `codefly-dev/core` and named in the PR. It is never a local re-implementation —
  not as a "workaround", not "just this once", not "until the capability lands".
  `Code` embeds `DefaultCodeServer` with no overrides on purpose.
- **Never hack. Provide the best fix, even when it spans repos.** The fix living
  in `codefly-dev/core` or the CLI is not a reason to work around it here. Open
  the PR there and consume the reviewed result. When it genuinely cannot be fixed
  now, the deliverable is a precise issue against that owner plus an explicitly
  labelled stopgap — never an unlabelled one.
- **Classify every change that makes something work**, in the PR body: a *fix* at
  the place that owns the behaviour, or a *hack*. A hack does not become a fix by
  working, by being small, by being local, or by the real fix belonging elsewhere.
- **Never hardcode what the system resolves** — injected environment, derived
  ports, service addresses, credentials copied out of another component.
  `CODEFLY_AGENT_WORKDIR` is injected; the generated service declaration is the
  single authority for `source-dir`, which is why `sourceLocationForCode` reads it
  through Core's `LoadServiceFromDir` instead of parsing or copying the contract.
  Typing one of these encodes something true only on one machine for ten minutes,
  and it fails quietly — a runtime missing a credential can skip registration
  *silently*, so the service boots, serves, and is simply absent.
- **Diagnose, do not pattern-match.** "It started working when I set X" is not a
  diagnosis — set X back and confirm it breaks. Do not trust an error message
  before checking its claim: `resolveAttachedSource` exists because a perfectly
  valid attachment reported *empty* after a successful `Runtime.Load`, and the
  cause was `filepath.WalkDir` refusing to descend a symlink passed as its root —
  nothing to do with the attachment.
- **Say what you did not verify.** Unverified is not the same as working. The
  suite here needs no network, no cluster, and runs in well under a second, so
  "I did not run it" is a choice; if you could not exercise something, the PR
  says so.

## Build and test

Derived from `.github/workflows/ci.yml` (job `boundary`), which pins Go from
`go.mod` (1.27.0) and runs with no network, no kubeconfig, and no cloud
credentials:

```bash
go build ./...
go vet ./...
go test ./...      # boundary and conformance tests; ~0.05s, no external deps
```

That is the entire gate — no linter, no coverage threshold, no build tags, no
integration suite. `.github/workflows/releaser.yml` re-runs `go test -v ./...`
before it builds artifacts.

Releases cross-compile with `CGO_ENABLED=1` in `goreleaser/goreleaser-cross`,
because Core's authoritative source inspection uses real tree-sitter grammars.
A local `go build` uses your own default and is not a release rehearsal.

## The boundary is the product

Most of the test suite exists to keep capabilities **absent**. Each guard below
protects a property that a small, plausible edit would destroy:

- **No Builder, no `BUILDER` capability.** That absence is what makes an
  image-coverage claim structurally impossible, which is what entitles this
  agent to the `NO_IMAGE` SBOM status — a service that legitimately ships nothing
  to scan, distinct from one whose SBOM support is merely unimplemented.
- **No repository or reconciler integration.** `transportTokens` in
  `boundary_test.go` scans production `.go` source for GitOps, reconciler, and
  cluster-credential responsibilities this agent must never carry.
- **Build/Test/Lint answer with a typed
  `FAILURE_CODE_UNSUPPORTED_OPERATION`**, never gRPC `Unimplemented` (which
  would misreport a deliberate capability boundary as transport failure) and
  never an invented command for an unknown language.
- **The advertised README and the enforced status may not drift.** A test reads
  `templates/agent/README.md.tmpl` through `GetAgentInformation` and asserts the
  boundary it documents.

Every one of these has an obvious way to turn a red test green while destroying
what it protects. `.claude/skills/boundary-triage/` walks that distinction.

## Where things live

| Path | Owns |
| --- | --- |
| `service.go` | `Service`, `Settings` (`source-dir` only), source resolution, `pluginRegistration` |
| `code.go` | `Code` — Core's `DefaultCodeServer` bound to the resolved source |
| `runtime.go` | `Runtime` — Load/Init/Start/Stop; typed unsupported build/test/lint |
| `tooling.go` | `Tooling` — project info, semantic index, edits; delegates to `Code` |
| `templates/agent/README.md.tmpl` | the capability guide served by `GetAgentInformation` |
| `agent.codefly.yaml` | publisher, kind, name, and the **released version** |

## Procedures

Step-by-step procedures live in `.claude/skills/`, loaded on demand rather than
carried here:

- `boundary-triage` — a guard in this repo is red, and you need to tell a real
  finding from the escape hatch that would only make it green.
- `release-agent` — cutting a version, and the manifest/tag/archive-name
  contract that Core's downloader resolves against.

## Workflow

- Branch and PR; never commit to `main`. Conventional Commits for the title.
- `agent_context_test.go` holds this file's length budget and each skill's
  frontmatter contract. It runs under `go test ./...`, so CI enforces it with no
  workflow change.
- Keep this file under ~150 lines (hard cap 200). Push depth into a nested
  `AGENTS.md` beside what it describes, or into `.claude/skills/`.
- If a `CLAUDE.md` is ever added, it is the single line `@AGENTS.md`. One
  canonical source.
- Treat this file as code: the PR that changes a process updates it.
