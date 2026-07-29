package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestManifestCapabilityUnsupported locks in the generic agent's boundary: it
// is a manifest producer that intentionally emits no Kubernetes output, so it
// registers no Builder and the manifest-bundle/deploy capability is reported as
// unsupported. Wiring a Builder here would pull manifest and transport
// responsibility into a plugin that must never carry it.
func TestManifestCapabilityUnsupported(t *testing.T) {
	if builder := pluginRegistration().Builder; builder != nil {
		t.Fatalf("generic agent must register no Builder (manifest capability is unsupported), got %T", builder)
	}
}

// transportTokens name responsibilities the generic agent must never own:
// GitOps/reconciler control planes, repository source bindings, and cluster
// credentials. Plain local-git read operations (GitLog/GitDiff/GitBlame) are
// deliberately not listed — inspecting a working tree is source tooling, not
// manifest transport.
var transportTokens = []string{
	"argocd",
	"argoproj",
	"fluxcd",
	"gitops",
	"reconcil",
	"repourl",
	"targetrevision",
	"appproject",
	"kubeconfig",
	"go-github",
	"pullrequest",
}

// TestSourceContainsNoTransportIntegration keeps the boundary honest over time:
// the plugin's own source must not grow repository or reconciler integration.
func TestSourceContainsNoTransportIntegration(t *testing.T) {
	entries, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob source: %v", err)
	}
	for _, path := range entries {
		if path == "boundary_test.go" {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		lowered := strings.ToLower(string(content))
		for _, token := range transportTokens {
			if strings.Contains(lowered, token) {
				t.Errorf("%s references transport/reconciler responsibility %q; the generic agent must stay manifest- and transport-neutral", path, token)
			}
		}
	}
}
