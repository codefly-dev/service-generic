---
name: boundary-triage
description: Diagnose a failing boundary or conformance test in service-generic and tell a real finding from the escape hatch that would only make it green. Use when go test reports "must register no Builder", "references transport/reconciler responsibility", "agent README does not document the no-image SBOM status", "shared contract accepted a false image-coverage claim", or a capability that should be unsupported is answering; or before adding a Builder, an SBOM response, a new dependency, or any manifest, GitOps, or cluster-credential code to this agent.
---

# Triaging a boundary failure

Every guard in this repo protects a capability that must stay **absent**. Each
one can be silenced in a way that keeps CI green and destroys the property, so
the first question is never "how do I make this pass" — it is "which of the two
things that could be true is true".

For every failure below: **A** is the real finding, **B** is the escape hatch.
Taking B is a hack under the repo's own rules and must be labelled as one in the
PR body — but the honest move is nearly always to stop and fix what owns the
behaviour, usually in `codefly-dev/core` or a language-specific agent.

## `generic agent must register no Builder`

`TestManifestCapabilityUnsupported`, `boundary_test.go`.

- **A.** Something asked this agent to produce Kubernetes manifests or deploy
  output. That responsibility belongs to a language-specific agent or to Core —
  not here. The manifest-bundle capability is reported unsupported by design.
- **B.** Registering the Builder anyway, or deleting the assertion. This pulls
  manifest and transport responsibility into a plugin that must never carry it,
  and it silently invalidates the SBOM status below.

## `references transport/reconciler responsibility`

`TestSourceContainsNoTransportIntegration` scans production `.go` source for the
tokens in `transportTokens` (argocd, argoproj, fluxcd, gitops, reconcil, repourl,
targetrevision, appproject, kubeconfig, go-github, pullrequest).

- **A.** Source you added — or a Core surface you started importing — carries
  GitOps, repository-binding, or cluster-credential responsibility. Remove it;
  if the capability is genuinely needed, it belongs to whoever owns deployment.
- **B.** Deleting a token from `transportTokens`, or relocating the code to a
  `_test.go` file, `vendor/`, or a dotted directory. The scan deliberately skips
  all three, so this "passes" while the responsibility is still in the binary.

Local git reads (`GitLog`, `GitDiff`, `GitBlame`) are **not** in the list and are
not violations: inspecting a working tree is source tooling, not transport.

## `agent README does not document the no-image SBOM status`

`TestAgentReadMeDocumentsNoImageStatus` reads the template through
`GetAgentInformation` and requires it to name `no runtime image` and `NO_IMAGE`.

- **A.** `templates/agent/README.md.tmpl` was edited and the advertised
  documentation drifted from the enforced status. The status is authoritative;
  restore the documented boundary.
- **B.** Loosening the assertion to match whatever the template now says. The
  test exists precisely because a consumer reads that README to learn the
  boundary, and a README that no longer states it is the failure.

## `shared contract accepted a false image-coverage claim`

`TestFalseImageCoverageClaimsAreRejected` measures four dishonest answers against
Core's shared `sbom.ValidateCoverage`. A failure here means the **shared
contract** stopped rejecting something it used to reject.

- **A.** A Core change weakened the fleet-wide coverage check. That is a finding
  about `codefly-dev/core`, not about this repo — file it there. Do not adjust
  this repo's expectations to match a loosened contract.
- **B.** Editing the `want` strings or dropping a case until it passes. This
  repo's `NO_IMAGE` claim is only honest while the alternatives are rejected;
  weakening the test removes the thing that makes the claim meaningful.

## Adding a Builder later

`TestAnyRegisteredBuilderServesConformantImageEvidence` currently skips —
there is no image-producing path to gate. Registering a Builder activates it,
and it will fail until that Builder's image-scope answer is conformant.

That failure is the gate working. The answer is to derive real subjects from the
build you declare and assert coverage against them. Inheriting the toolbox's
`NO_IMAGE` status from a Builder that actually builds an image is a fabricated
coverage claim, and `TestGenericAgentClaimsNoImageProducingCapability` will
contradict it.

## A capability answers instead of refusing

`Build`, `Test`, `Lint` on both `Runtime` and `Tooling`, plus `AddDependency` and
`RemoveDependency` on `Tooling`, return a typed
`FAILURE_CODE_UNSUPPORTED_OPERATION`. Two rules hold:

- Never return gRPC `Unimplemented` instead — that misclassifies a deliberate
  capability boundary as a transport failure.
- Never invent a command. This agent has no language knowledge; guessing
  `make test` for an unknown project is the hardcoding rule violated directly.

`TestGenericAgentProvidesRealBaselineCodeAndProjectInfo` pins the other half:
declarative inspection still returns *real* data (preserved file hashes) while
reporting `language: "unknown"` and the unsupported code. Real inspection and an
honest capability answer are not in tension — do not relax one to get the other.

## Before you believe a pass

- `go vet ./...` runs in CI ahead of the suite; a vet failure blocks everything.
- The transport scan walks the working tree from `.`, not the git index, so it
  reads whatever is on disk — staged or not. Run `go test ./...` against the
  tree you are actually about to push.
