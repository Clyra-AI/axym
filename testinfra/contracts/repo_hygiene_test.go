package contracts

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestRequiredPlanningArtifactsExist(t *testing.T) {
	t.Parallel()

	required := []string{
		"CONTRIBUTING.md",
		"SECURITY.md",
		"product/PLAN_v1.0.md",
	}

	for _, path := range required {
		if _, err := os.Stat(filepath.Join(repoRoot(t), filepath.FromSlash(path))); err != nil {
			t.Fatalf("required planning artifact missing: %s: %v", path, err)
		}
	}
}

func TestRequiredLaunchAssetsExist(t *testing.T) {
	t.Parallel()

	required := []string{
		"LICENSE",
		"CHANGELOG.md",
		"CODE_OF_CONDUCT.md",
		".github/ISSUE_TEMPLATE/bug_report.yml",
		".github/ISSUE_TEMPLATE/feature_request.yml",
		".github/pull_request_template.md",
	}

	for _, path := range required {
		if _, err := os.Stat(filepath.Join(repoRoot(t), filepath.FromSlash(path))); err != nil {
			t.Fatalf("required launch asset missing: %s: %v", path, err)
		}
	}
}

func TestLaunchFacingDocsReferenceCurrentOSSBoundary(t *testing.T) {
	t.Parallel()

	readme, err := os.ReadFile(filepath.Join(repoRoot(t), "README.md"))
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	commandGuide, err := os.ReadFile(filepath.Join(repoRoot(t), "docs", "commands", "axym.md"))
	if err != nil {
		t.Fatalf("read command guide: %v", err)
	}
	for _, raw := range [][]byte{readme, commandGuide} {
		content := string(raw)
		for _, snippet := range []string{
			"Smoke test",
			"Sample proof path",
			"Real integration path",
			"./axym init --sample-pack ./axym-sample --json",
		} {
			if !strings.Contains(content, snippet) {
				t.Fatalf("launch-facing docs missing snippet %q", snippet)
			}
		}
	}
}

func TestTrackedSecretArtifactsAreAbsent(t *testing.T) {
	t.Parallel()

	prohibited := []string{
		".env",
		"id_rsa",
		"secrets.txt",
	}

	for _, path := range prohibited {
		if _, err := os.Stat(filepath.Join(repoRoot(t), filepath.FromSlash(path))); err == nil {
			t.Fatalf("prohibited artifact exists: %s", path)
		}
	}
}
