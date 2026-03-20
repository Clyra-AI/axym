package bundle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	coreoverride "github.com/Clyra-AI/axym/core/override"
)

func TestBuildRequiresAuditName(t *testing.T) {
	t.Parallel()

	_, err := Build(BuildRequest{})
	if err == nil {
		t.Fatal("expected audit validation error")
	}
	bErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if bErr.ExitCode != 6 {
		t.Fatalf("exit mismatch: got %d want 6", bErr.ExitCode)
	}
}

func TestBuildRejectsUnmanagedOutputPath(t *testing.T) {
	t.Parallel()

	outDir := filepath.Join(t.TempDir(), "bundle")
	if err := os.MkdirAll(outDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "foreign.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := Build(BuildRequest{
		AuditName: "Q3-2026",
		OutputDir: outDir,
		StoreDir:  filepath.Join(t.TempDir(), "store"),
	})
	if err == nil {
		t.Fatal("expected unmanaged output rejection")
	}
	bErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if bErr.ExitCode != 8 {
		t.Fatalf("exit mismatch: got %d want 8", bErr.ExitCode)
	}
}

func TestBuildIncludesOverrideArtifactWhenPresent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	outDir := filepath.Join(root, "bundle")

	if _, err := coreoverride.Create(coreoverride.Request{
		Bundle:    "Q3-2026",
		Reason:    "fixture",
		Signer:    "ops-key",
		StoreDir:  storeDir,
		ExpiresAt: time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC),
		Now:       func() time.Time { return time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC) },
	}); err != nil {
		t.Fatalf("override.Create: %v", err)
	}

	result, err := Build(BuildRequest{
		AuditName:    "Q3-2026",
		FrameworkIDs: []string{"eu-ai-act", "soc2"},
		StoreDir:     storeDir,
		OutputDir:    outDir,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if result.Files == 0 {
		t.Fatalf("expected bundle files, got %+v", result)
	}

	overridePath := filepath.Join(outDir, "overrides", "overrides.jsonl")
	rawOverride, err := os.ReadFile(overridePath)
	if err != nil {
		t.Fatalf("ReadFile override artifact: %v", err)
	}
	if len(rawOverride) == 0 {
		t.Fatal("expected non-empty override artifact")
	}

	rawManifest, err := os.ReadFile(filepath.Join(outDir, "manifest.json"))
	if err != nil {
		t.Fatalf("ReadFile manifest: %v", err)
	}
	var manifest struct {
		Files []struct {
			Path string `json:"path"`
		} `json:"files"`
	}
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		t.Fatalf("Unmarshal manifest: %v", err)
	}
	found := false
	for _, entry := range manifest.Files {
		if entry.Path == "overrides/overrides.jsonl" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("manifest missing override artifact: %s", rawManifest)
	}
}

func TestBuildIncludesIdentityGovernanceArtifacts(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	outDir := filepath.Join(root, "bundle")

	result, err := Build(BuildRequest{
		AuditName:    "Q3-2026",
		FrameworkIDs: []string{"eu-ai-act", "soc2"},
		StoreDir:     storeDir,
		OutputDir:    outDir,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if result.Files == 0 {
		t.Fatalf("expected bundle files, got %+v", result)
	}

	required := []string{
		"identity-chain-summary.json",
		"ownership-register.json",
		"privilege-drift-report.json",
		"delegated-chain-exceptions.json",
		"record-signing-key.json",
	}
	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(outDir, rel)); err != nil {
			t.Fatalf("missing identity artifact %s: %v", rel, err)
		}
	}
}
