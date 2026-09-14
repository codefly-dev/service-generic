package main

import (
	"strings"
	"testing"

	"github.com/codefly-dev/core/agents/services"
	"github.com/codefly-dev/core/agents/services/sbom"
	agentv0 "github.com/codefly-dev/core/generated/go/codefly/services/agent/v0"
	builderv0 "github.com/codefly-dev/core/generated/go/codefly/services/builder/v0"
)

// declaredImageSBOM is the image-scope answer the generic agent's status maps to
// under the shared contract. A passive toolbox ships nothing to scan, so
// NO_IMAGE is the only honest declaration it can make; it is built through
// Core's wrapper rather than hand-assembled so this repository is measured
// against the fleet contract instead of a local copy of it.
func declaredImageSBOM(t *testing.T) *builderv0.SBOMResponse {
	t.Helper()
	response, err := (&services.BuilderWrapper{}).SBOMNoImage(
		builderv0.NoImageReason_NO_IMAGE_REASON_NO_IMAGE,
		"generic agent is a passive language-agnostic toolbox and emits no image",
	)
	if err != nil {
		t.Fatalf("build no-image declaration: %v", err)
	}
	return response
}

// fabricatedImageEvidence is the shape a false "everything is covered" answer
// takes: inventories that look complete for an image this agent never builds.
func fabricatedImageEvidence() []*builderv0.ImageSBOM {
	return []*builderv0.ImageSBOM{{
		Digest:   "sha256:" + strings.Repeat("a", 64),
		Platform: "linux/amd64",
		Subjects: []*builderv0.ImageSubject{{
			Reference: "ghcr.io/codefly-dev/generic:0.0.35",
			Role:      "runtime",
			Service:   "generic",
		}},
		Bom:    &agentv0.Bom{Components: []*agentv0.Component{{}}},
		Sha256: "0f1e2d3c4b5a",
	}}
}

// TestGenericAgentClaimsNoImageProducingCapability locks the antecedent of the
// no-image status: image evidence reaches a consumer only through Builder.SBOM,
// and Core gates that on the advertised BUILDER capability. Registering neither
// is what makes an image-coverage claim structurally impossible here, so both
// must stay absent for as long as the agent emits no image.
func TestGenericAgentClaimsNoImageProducingCapability(t *testing.T) {
	info, err := NewService().GetAgentInformation(t.Context(), &agentv0.AgentInformationRequest{})
	if err != nil {
		t.Fatalf("get agent information: %v", err)
	}
	for _, capability := range info.GetCapabilities() {
		if capability.GetType() == agentv0.Capability_BUILDER {
			t.Fatal("generic agent advertises BUILDER: a builder-capable agent owes digest-bound image SBOM evidence")
		}
	}
	if builder := pluginRegistration().Builder; builder != nil {
		t.Fatalf("generic agent registers a Builder (%T) but serves no image SBOM evidence", builder)
	}
}

// TestAgentReadMeDocumentsNoImageStatus keeps the advertised documentation and
// the enforced status from drifting apart: a consumer reading the agent's README
// must find the same no-image boundary the tests below hold it to.
func TestAgentReadMeDocumentsNoImageStatus(t *testing.T) {
	info, err := NewService().GetAgentInformation(t.Context(), &agentv0.AgentInformationRequest{})
	if err != nil {
		t.Fatalf("get agent information: %v", err)
	}
	readme := info.GetReadMe()
	if !strings.Contains(readme, "no runtime image") || !strings.Contains(readme, "NO_IMAGE") {
		t.Fatalf("agent README does not document the no-image SBOM status: %q", readme)
	}
}

// TestDeclaredNoImageStatusSatisfiesSharedCoverageContract measures this
// repository's declared status with ValidateCoverage, the single check every
// agent in the fleet is measured against.
func TestDeclaredNoImageStatusSatisfiesSharedCoverageContract(t *testing.T) {
	response := declaredImageSBOM(t)
	if response.GetScope() != builderv0.SBOMScope_SBOM_SCOPE_IMAGE {
		t.Fatalf("scope = %v, want image scope", response.GetScope())
	}
	if response.GetNoImageReason() != builderv0.NoImageReason_NO_IMAGE_REASON_NO_IMAGE {
		t.Fatalf("no-image reason = %v, want NO_IMAGE", response.GetNoImageReason())
	}
	if err := sbom.ValidateCoverage(nil, response); err != nil {
		t.Fatalf("declared no-image status fails shared coverage conformance: %v", err)
	}
}

// TestFalseImageCoverageClaimsAreRejected is the release gate the no-image
// status rests on. Declaring NO_IMAGE is only honest while the dishonest
// alternatives are rejected, so each way this agent could overstate its
// coverage is checked against the shared contract.
func TestFalseImageCoverageClaimsAreRejected(t *testing.T) {
	wrapper := &services.BuilderWrapper{}

	unspecified, err := wrapper.SBOMNoImage(builderv0.NoImageReason_NO_IMAGE_REASON_UNSPECIFIED, "")
	if err != nil {
		t.Fatalf("build unspecified declaration: %v", err)
	}
	source, err := wrapper.SBOMResponse(&agentv0.Bom{Components: []*agentv0.Component{{}}}, "syft", "GO", "0f1e2d")
	if err != nil {
		t.Fatalf("build source inventory: %v", err)
	}
	fabricated, err := wrapper.SBOMImageResponse(fabricatedImageEvidence())
	if err != nil {
		t.Fatalf("build fabricated image response: %v", err)
	}
	contradictory := declaredImageSBOM(t)
	contradictory.Images = fabricatedImageEvidence()

	tests := []struct {
		name     string
		response *builderv0.SBOMResponse
		want     string
	}{
		{name: "no reason", response: unspecified, want: "explicit reason"},
		{name: "source inventory as image coverage", response: source, want: "not image coverage"},
		{name: "images without a declared expectation", response: fabricated, want: "must declare a no-image reason"},
		{name: "images under a no-image declaration", response: contradictory, want: "carries 1 inventories"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := sbom.ValidateCoverage(nil, test.response)
			if err == nil {
				t.Fatal("shared contract accepted a false image-coverage claim")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("rejection = %q, want it to name %q", err, test.want)
			}
		})
	}
}

// TestAnyRegisteredBuilderServesConformantImageEvidence is the forward gate: a
// Builder is the only path by which this agent could ever serve image evidence,
// so introducing one must not inherit the toolbox's no-image status by default.
// The skip is the current, honest state; the assertions activate with the first
// Builder and fail release until its image-scope answer is conformant.
func TestAnyRegisteredBuilderServesConformantImageEvidence(t *testing.T) {
	builder := pluginRegistration().Builder
	if builder == nil {
		t.Skip("generic agent registers no Builder: there is no image-producing path to gate")
	}
	response, err := builder.SBOM(t.Context(), &builderv0.SBOMRequest{Scope: builderv0.SBOMScope_SBOM_SCOPE_IMAGE})
	if err != nil {
		t.Fatalf("image-scope SBOM: %v", err)
	}
	// Measured against no expected subjects, ValidateCoverage admits only a
	// declared no-image response. A Builder that emits images must derive its
	// subjects from the build it declares and assert coverage against those.
	if err := sbom.ValidateCoverage(nil, response); err != nil {
		t.Fatalf("registered Builder does not satisfy image coverage conformance: %v", err)
	}
}
