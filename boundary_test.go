package main

import (
	"io/fs"
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

type transportViolation struct {
	path  string
	token string
}

// scanTransportTokens walks the production Go source under root and reports any
// transportToken it finds. It scans the whole tree (not just root's package) so
// integration hidden in a subpackage is still caught. Test files, vendored
// dependencies, and hidden directories are excluded: the boundary is about the
// plugin's own runtime source, not third-party code or test fixtures that name
// the forbidden responsibilities deliberately.
func scanTransportTokens(root string) ([]transportViolation, error) {
	var violations []transportViolation
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lowered := strings.ToLower(string(content))
		for _, token := range transportTokens {
			if strings.Contains(lowered, token) {
				violations = append(violations, transportViolation{path: path, token: token})
			}
		}
		return nil
	})
	return violations, err
}

// TestSourceContainsNoTransportIntegration keeps the boundary honest over time:
// the plugin's own source must not grow repository or reconciler integration.
func TestSourceContainsNoTransportIntegration(t *testing.T) {
	violations, err := scanTransportTokens(".")
	if err != nil {
		t.Fatalf("scan source: %v", err)
	}
	for _, violation := range violations {
		t.Errorf("%s references transport/reconciler responsibility %q; the generic agent must stay manifest- and transport-neutral", violation.path, violation.token)
	}
}

// TestScanTransportTokens_FlagsSubpackageSource proves the scan reaches beyond
// the root package: a token in a nested production file must be reported.
func TestScanTransportTokens_FlagsSubpackageSource(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "deploy", "internal")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	writeSource(t, filepath.Join(nested, "reconciler.go"), "package internal\n\nconst source = \"argocd repoURL\"\n")

	violations, err := scanTransportTokens(root)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !containsToken(violations, "argocd") || !containsToken(violations, "repourl") {
		t.Fatalf("nested production source not flagged: %+v", violations)
	}
}

// TestScanTransportTokens_ExcludesTestAndVendorSource proves the scan is scoped
// to the plugin's own runtime source: tokens in _test.go files and under
// vendor/ (and hidden dirs) are not the plugin's responsibility and must not
// trip the guard.
func TestScanTransportTokens_ExcludesTestAndVendorSource(t *testing.T) {
	root := t.TempDir()
	writeSource(t, filepath.Join(root, "fixture_test.go"), "package p\n\nconst x = \"argocd\"\n")
	vendored := filepath.Join(root, "vendor", "example.com", "dep")
	if err := os.MkdirAll(vendored, 0o755); err != nil {
		t.Fatal(err)
	}
	writeSource(t, filepath.Join(vendored, "client.go"), "package dep\n\nconst x = \"kubeconfig\"\n")

	violations, err := scanTransportTokens(root)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("test and vendored source must be excluded, got %+v", violations)
	}
}

func writeSource(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func containsToken(violations []transportViolation, token string) bool {
	for _, violation := range violations {
		if violation.token == token {
			return true
		}
	}
	return false
}
