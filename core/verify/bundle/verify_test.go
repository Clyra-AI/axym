package bundle_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	corebundle "github.com/Clyra-AI/axym/core/bundle"
	corecollect "github.com/Clyra-AI/axym/core/collect"
	"github.com/Clyra-AI/axym/core/collector"
	ingestgait "github.com/Clyra-AI/axym/core/ingest/gait"
	"github.com/Clyra-AI/axym/core/policy/sink"
	"github.com/Clyra-AI/axym/core/store"
	verifybundle "github.com/Clyra-AI/axym/core/verify/bundle"
	verifysupport "github.com/Clyra-AI/axym/core/verifysupport"
	"github.com/Clyra-AI/proof"
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

	storeDir, outDir := setupBundleFixture(t)
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
	if err := resignManifest(storeDir, outDir); err != nil {
		t.Fatalf("resign manifest: %v", err)
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

func TestVerifyBundleSchemaValidationReturnsPolicyViolationExit(t *testing.T) {
	t.Parallel()

	storeDir, outDir := setupBundleFixture(t)
	summaryPath := filepath.Join(outDir, "executive-summary.json")
	raw, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("ReadFile summary: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("Unmarshal summary: %v", err)
	}
	delete(payload, "compliance")
	invalid, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("Marshal invalid summary: %v", err)
	}
	if err := os.WriteFile(summaryPath, invalid, 0o600); err != nil {
		t.Fatalf("WriteFile summary: %v", err)
	}
	if err := updateManifestHash(outDir, "executive-summary.json"); err != nil {
		t.Fatalf("update manifest hash: %v", err)
	}
	if err := resignManifest(storeDir, outDir); err != nil {
		t.Fatalf("resign manifest: %v", err)
	}

	_, err = verifybundle.Verify(outDir, []string{"eu-ai-act", "soc2"})
	if err == nil {
		t.Fatal("expected schema validation error")
	}
	vErr, ok := err.(*verifybundle.Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if vErr.ReasonCode != verifybundle.ReasonSchemaViolation {
		t.Fatalf("reason mismatch: got %s", vErr.ReasonCode)
	}
	if vErr.ExitCode != 3 {
		t.Fatalf("exit mismatch: got %d want 3", vErr.ExitCode)
	}
}

func TestVerifyBundleChecksChainIntegrityBeforeCompliance(t *testing.T) {
	t.Parallel()

	storeDir, outDir := setupBundleFixture(t)
	chainPath := filepath.Join(outDir, "chain.json")
	raw, err := os.ReadFile(chainPath)
	if err != nil {
		t.Fatalf("ReadFile chain: %v", err)
	}
	var chain proof.Chain
	if err := json.Unmarshal(raw, &chain); err != nil {
		t.Fatalf("Unmarshal chain: %v", err)
	}
	if len(chain.Records) == 0 {
		t.Fatal("expected non-empty chain fixture")
	}
	chain.Records[0].Event["tamper"] = true
	tampered, err := json.MarshalIndent(chain, "", "  ")
	if err != nil {
		t.Fatalf("Marshal tampered chain: %v", err)
	}
	if err := os.WriteFile(chainPath, tampered, 0o600); err != nil {
		t.Fatalf("WriteFile chain: %v", err)
	}
	if err := updateManifestHash(outDir, "chain.json"); err != nil {
		t.Fatalf("update manifest hash: %v", err)
	}
	if err := resignManifest(storeDir, outDir); err != nil {
		t.Fatalf("resign manifest: %v", err)
	}

	_, err = verifybundle.Verify(outDir, []string{"eu-ai-act", "soc2"})
	if err == nil {
		t.Fatal("expected bundle verify failure for tampered chain")
	}
	vErr, ok := err.(*verifybundle.Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if vErr.ReasonCode != verifybundle.ReasonBundleVerify {
		t.Fatalf("reason mismatch: got %s", vErr.ReasonCode)
	}
	if vErr.ExitCode != 2 {
		t.Fatalf("exit mismatch: got %d want 2", vErr.ExitCode)
	}
}

func TestVerifyBundleFailsOnInvalidRecordSignature(t *testing.T) {
	t.Parallel()

	storeDir, outDir := setupBundleFixture(t)
	chainPath := filepath.Join(outDir, "chain.json")
	raw, err := os.ReadFile(chainPath)
	if err != nil {
		t.Fatalf("ReadFile chain: %v", err)
	}
	var chain proof.Chain
	if err := json.Unmarshal(raw, &chain); err != nil {
		t.Fatalf("Unmarshal chain: %v", err)
	}
	if len(chain.Records) == 0 {
		t.Fatal("expected non-empty chain fixture")
	}
	chain.Records[0].Integrity.Signature = "base64:AAAA"
	tampered, err := json.MarshalIndent(chain, "", "  ")
	if err != nil {
		t.Fatalf("Marshal tampered chain: %v", err)
	}
	if err := os.WriteFile(chainPath, tampered, 0o600); err != nil {
		t.Fatalf("WriteFile chain: %v", err)
	}
	if err := updateManifestHash(outDir, "chain.json"); err != nil {
		t.Fatalf("update manifest hash: %v", err)
	}
	if err := resignManifest(storeDir, outDir); err != nil {
		t.Fatalf("resign manifest: %v", err)
	}

	_, err = verifybundle.Verify(outDir, []string{"eu-ai-act", "soc2"})
	if err == nil {
		t.Fatal("expected bundle signature verification failure")
	}
	vErr, ok := err.(*verifybundle.Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if vErr.ReasonCode != verifybundle.ReasonBundleSignature {
		t.Fatalf("reason mismatch: got %s", vErr.ReasonCode)
	}
	if vErr.ExitCode != 2 {
		t.Fatalf("exit mismatch: got %d want 2", vErr.ExitCode)
	}
}

func TestVerifyFailsDerivedArtifactDrift(t *testing.T) {
	t.Parallel()

	storeDir, outDir := setupDerivedBundleFixture(t)
	authPath := filepath.Join(outDir, "authorization-register.json")
	raw, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatalf("read authorization register: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode authorization register: %v", err)
	}
	entries := payload["entries"].([]any)
	entry := entries[0].(map[string]any)
	entry["status"] = "partial"
	tampered, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal authorization register: %v", err)
	}
	if err := os.WriteFile(authPath, tampered, 0o600); err != nil {
		t.Fatalf("write authorization register: %v", err)
	}
	if err := updateManifestHash(outDir, "authorization-register.json"); err != nil {
		t.Fatalf("update manifest hash: %v", err)
	}
	if err := resignManifest(storeDir, outDir); err != nil {
		t.Fatalf("resign manifest: %v", err)
	}

	_, err = verifybundle.Verify(outDir, []string{"eu-ai-act", "soc2"})
	if err == nil {
		t.Fatal("expected derived artifact drift error")
	}
	vErr, ok := err.(*verifybundle.Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if vErr.ReasonCode != verifybundle.ReasonBundleCompleteness {
		t.Fatalf("reason mismatch: got %s", vErr.ReasonCode)
	}
	if vErr.ExitCode != 2 {
		t.Fatalf("exit mismatch: got %d want 2", vErr.ExitCode)
	}
}

func TestVerifySkipsUndeclaredDerivedArtifacts(t *testing.T) {
	t.Parallel()

	storeDir, outDir := setupDerivedBundleFixture(t)
	if err := os.Remove(filepath.Join(outDir, "control-maturity.json")); err != nil {
		t.Fatalf("remove control-maturity artifact: %v", err)
	}
	if err := removeManifestEntry(outDir, "control-maturity.json"); err != nil {
		t.Fatalf("remove manifest entry: %v", err)
	}
	if err := resignManifest(storeDir, outDir); err != nil {
		t.Fatalf("resign manifest: %v", err)
	}

	if _, err := verifybundle.Verify(outDir, []string{"eu-ai-act", "soc2"}); err != nil {
		t.Fatalf("Verify should ignore undeclared derived artifact: %v", err)
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

func removeManifestEntry(bundleDir string, relPath string) error {
	manifestPath := filepath.Join(bundleDir, "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var manifest struct {
		Files []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"files"`
		Signatures []any  `json:"signatures,omitempty"`
		AlgoID     string `json:"algo_id,omitempty"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return err
	}
	filtered := manifest.Files[:0]
	for _, file := range manifest.Files {
		if file.Path == relPath {
			continue
		}
		filtered = append(filtered, file)
	}
	manifest.Files = filtered
	out, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(manifestPath, out, 0o600)
}

func resignManifest(storeDir string, bundleDir string) error {
	signingKey, err := verifysupport.LoadStoreSigningKey(storeDir)
	if err != nil {
		return err
	}
	manifestPath := filepath.Join(bundleDir, "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var manifestPayload map[string]any
	if err := json.Unmarshal(raw, &manifestPayload); err != nil {
		return err
	}
	delete(manifestPayload, "signatures")
	cleaned, err := json.MarshalIndent(manifestPayload, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(manifestPath, cleaned, 0o600); err != nil {
		return err
	}
	signedManifest, err := proof.SignBundleFile(bundleDir, signingKey)
	if err != nil {
		return err
	}
	if len(signedManifest.Signatures) == 0 {
		return errors.New("manifest signatures missing after resign")
	}
	return nil
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

func setupDerivedBundleFixture(t *testing.T) (string, string) {
	t.Helper()

	storeDir := filepath.Join(t.TempDir(), "store")
	packDir := filepath.Join(t.TempDir(), "gait-pack")
	if err := os.MkdirAll(packDir, 0o700); err != nil {
		t.Fatalf("mkdir gait pack: %v", err)
	}
	linkedRecordID := writeDerivedProofFixture(t, filepath.Join(packDir, "proof_records.jsonl"))
	authPayload := fmt.Sprintf(`{
  "artifact_type": "gate_authorization_bundle",
  "artifact_id": "artifact-1",
  "bundle_id": "bundle-1",
  "timestamp": "2026-05-12T17:00:00Z",
  "decision": "allow",
  "trace_id": "trace-1",
  "intent_digest": "sha256:intent-1",
  "policy_digest": "sha256:policy-1",
  "environment": "prod",
  "target_kind": "workflow",
  "target_id": "deploy.pipeline",
  "verification": {
    "status": "verified",
    "schema_valid": true,
    "signature_valid": true
  },
  "approval_audit_refs": ["approval://chg-1"],
  "linked_record_ids": [%q],
  "credential_posture": {
    "standing_credential_blocked": true,
    "jit_required": true,
    "broker_source": "gait-broker",
    "issuer": "issuer-a",
    "ttl_seconds": 600,
    "scope": "deploy.write",
    "binding_proof_ref": "binding://proof-1"
  },
  "freeze_window": {
    "state": "enforced",
    "reason": "approved_change_window",
    "explain": "freeze window enforced"
  },
  "kill_switch": {
    "state": "active",
    "blocked_dispatch_proof": "proof://kill-1",
    "actor": "owner://platform"
  },
  "enforcement_explain": {
    "missing_fields": [],
    "credential_posture": "jit_verified",
    "freeze_state": "enforced",
    "kill_switch_state": "active",
    "sandbox_state": "confined"
  },
  "sandbox": {
    "path": "proc.exec",
    "network_mode": "deny",
    "writable_paths": ["/tmp"],
    "env_exposure": ["PATH"],
    "timeout_seconds": 30,
    "filesystem_isolation": "readonly",
    "policy_result": "allow"
  },
  "trust_graduation": {
    "path": "proc.exec",
    "current_stage": "brokered_write",
    "prior_stage": "approval_gated_write",
    "posture_state": "healthy",
    "approval_fatigue_signals": ["low_queue_depth"]
  }
}`, linkedRecordID)
	if err := os.WriteFile(filepath.Join(packDir, "authorization-bundle.json"), []byte(authPayload), 0o600); err != nil {
		t.Fatalf("write auth artifact: %v", err)
	}
	st, err := store.New(store.Config{RootDir: storeDir, ComplianceMode: true})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	if _, err := ingestgait.Ingest(context.Background(), ingestgait.Request{
		Store:      st,
		InputPaths: []string{packDir},
	}); err != nil {
		t.Fatalf("gait.Ingest: %v", err)
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

func writeDerivedProofFixture(t *testing.T, path string) string {
	t.Helper()

	record, err := proof.NewRecord(proof.RecordOpts{
		Timestamp:     time.Date(2026, 5, 12, 16, 59, 0, 0, time.UTC),
		Source:        "gait",
		SourceProduct: "gait",
		Type:          "approval",
		Event: map[string]any{
			"approval_token_ref": "approval://chg-1",
			"trace_id":           "trace-1",
		},
		Controls: proof.Controls{PermissionsEnforced: true},
	})
	if err != nil {
		t.Fatalf("proof.NewRecord: %v", err)
	}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal proof fixture: %v", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		t.Fatalf("write proof fixture: %v", err)
	}
	return record.RecordID
}
