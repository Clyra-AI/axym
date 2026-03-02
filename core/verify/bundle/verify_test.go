package bundle_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	corebundle "github.com/Clyra-AI/axym/core/bundle"
	corecollect "github.com/Clyra-AI/axym/core/collect"
	"github.com/Clyra-AI/axym/core/collector"
	"github.com/Clyra-AI/axym/core/policy/sink"
	"github.com/Clyra-AI/axym/core/store"
	verifybundle "github.com/Clyra-AI/axym/core/verify/bundle"
)

func TestVerifyBundleWithCompliance(t *testing.T) {
	t.Parallel()

	storeDir, outDir := setupBundleFixture(t)
	result, err := verifybundle.Verify(outDir, []string{"eu-ai-act", "soc2"})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !result.Cryptographic {
		t.Fatalf("expected cryptographic=true: %+v", result)
	}
	if !result.ComplianceVerified {
		t.Fatalf("expected compliance_verified=true: %+v", result)
	}
	if !result.OSCALValid {
		t.Fatalf("expected oscal_valid=true: %+v", result)
	}
	if result.Path == "" || result.Files == 0 || result.Algo == "" {
		t.Fatalf("unexpected result fields: %+v", result)
	}
	_ = storeDir
}

func TestVerifyBundleDetectsComplianceMismatch(t *testing.T) {
	t.Parallel()

	_, outDir := setupBundleFixture(t)
	summaryPath := filepath.Join(outDir, "executive-summary.json")
	raw, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("ReadFile summary: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("Unmarshal summary: %v", err)
	}
	compliance := payload["compliance"].(map[string]any)
	compliance["complete"] = true
	tampered, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("Marshal tampered summary: %v", err)
	}
	if err := os.WriteFile(summaryPath, tampered, 0o600); err != nil {
		t.Fatalf("WriteFile summary: %v", err)
	}
	if err := updateManifestHash(outDir, "executive-summary.json"); err != nil {
		t.Fatalf("update manifest hash: %v", err)
	}

	_, err = verifybundle.Verify(outDir, []string{"eu-ai-act", "soc2"})
	if err == nil {
		t.Fatal("expected compliance mismatch error")
	}
	vErr, ok := err.(*verifybundle.Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if vErr.ReasonCode != verifybundle.ReasonBundleCompleteness {
		t.Fatalf("reason mismatch: got %s", vErr.ReasonCode)
	}
	if vErr.ExitCode != 2 {
		t.Fatalf("exit mismatch: got %d", vErr.ExitCode)
	}
}

func updateManifestHash(bundleDir string, relPath string) error {
	payload, err := os.ReadFile(filepath.Join(bundleDir, filepath.FromSlash(relPath)))
	if err != nil {
		return err
	}
	sum := sha256.Sum256(payload)
	want := "sha256:" + hex.EncodeToString(sum[:])

	manifestPath := filepath.Join(bundleDir, "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var manifest map[string]any
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return err
	}
	files, ok := manifest["files"].([]any)
	if !ok {
		return nil
	}
	for i := range files {
		entry, ok := files[i].(map[string]any)
		if !ok {
			continue
		}
		if entry["path"] == relPath {
			entry["sha256"] = want
		}
	}
	out, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(manifestPath, out, 0o600)
}

func setupBundleFixture(t *testing.T) (string, string) {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	fixtureDir := filepath.Join(repoRoot, "fixtures", "collectors")

	req := collector.Request{
		Now:        time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
		FixtureDir: fixtureDir,
	}
	registry, err := corecollect.BuildRegistry(req)
	if err != nil {
		t.Fatalf("BuildRegistry: %v", err)
	}
	storeDir := filepath.Join(t.TempDir(), "store")
	st, err := store.New(store.Config{RootDir: storeDir, ComplianceMode: true})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	runner := corecollect.Runner{Registry: registry, Store: st, SinkMode: sink.ModeFailClosed}
	if _, err := runner.Run(context.Background(), req, false); err != nil {
		t.Fatalf("collect runner: %v", err)
	}
	outDir := filepath.Join(t.TempDir(), "bundle")
	if _, err := corebundle.Build(corebundle.BuildRequest{
		AuditName:    "Q3-2026",
		FrameworkIDs: []string{"eu-ai-act", "soc2"},
		StoreDir:     storeDir,
		OutputDir:    outDir,
	}); err != nil {
		t.Fatalf("bundle build: %v", err)
	}
	return storeDir, outDir
}
