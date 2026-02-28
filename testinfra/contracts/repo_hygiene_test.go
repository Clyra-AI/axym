package contracts

import (
	"os"
	"path/filepath"
	"runtime"
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
		"product/PLAN_v1.0.md",
	}

	for _, path := range required {
		if _, err := os.Stat(filepath.Join(repoRoot(t), filepath.FromSlash(path))); err != nil {
			t.Fatalf("required planning artifact missing: %s: %v", path, err)
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
