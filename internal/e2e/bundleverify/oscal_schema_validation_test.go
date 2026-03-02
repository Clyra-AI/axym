package bundleverify

import (
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBundleVerifyIncludesOSCALValidation(t *testing.T) {
	t.Parallel()

	repoRoot := testRepoRoot(t)
	storeDir := filepath.Join(t.TempDir(), "store")
	outDir := filepath.Join(t.TempDir(), "bundle")

	collectOut, collectExit := runAxym(t, repoRoot,
		"collect",
		"--fixture-dir", filepath.Join(repoRoot, "fixtures", "collectors"),
		"--store-dir", storeDir,
		"--json",
	)
	if collectExit != 0 {
		t.Fatalf("collect failed: exit=%d output=%s", collectExit, collectOut)
	}
	bundleOut, bundleExit := runAxym(t, repoRoot,
		"bundle",
		"--audit", "Q3-2026",
		"--frameworks", "eu-ai-act,soc2",
		"--store-dir", storeDir,
		"--output", outDir,
		"--json",
	)
	if bundleExit != 0 {
		t.Fatalf("bundle failed: exit=%d output=%s", bundleExit, bundleOut)
	}
	verifyOut, verifyExit := runAxym(t, repoRoot, "verify", "--bundle", outDir, "--json")
	if verifyExit != 0 {
		t.Fatalf("verify failed: exit=%d output=%s", verifyExit, verifyOut)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(verifyOut), &payload); err != nil {
		t.Fatalf("decode verify json: %v output=%s", err, verifyOut)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("missing verify data envelope: %s", verifyOut)
	}
	verification, ok := data["verification"].(map[string]any)
	if !ok {
		t.Fatalf("missing verification payload: %s", verifyOut)
	}
	if verification["compliance_verified"] != true {
		t.Fatalf("expected compliance verification true: %s", verifyOut)
	}
	if verification["oscal_valid"] != true {
		t.Fatalf("expected oscal_valid=true: %s", verifyOut)
	}
}

func testRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func runAxym(t *testing.T, repoRoot string, args ...string) (string, int) {
	t.Helper()
	binaryName := "axym"
	if runtime.GOOS == "windows" {
		binaryName = "axym.exe"
	}
	binaryPath := filepath.Join(t.TempDir(), binaryName)

	build := exec.Command("go", "build", "-o", binaryPath, "./cmd/axym")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build axym: %v output=%s", err, string(out))
	}

	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err == nil {
		return string(out), 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return string(out), exitErr.ExitCode()
	}
	t.Fatalf("run axym: %v", err)
	return "", 1
}
