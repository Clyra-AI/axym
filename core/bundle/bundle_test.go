package bundle

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	ingestgait "github.com/Clyra-AI/axym/core/ingest/gait"
	coreoverride "github.com/Clyra-AI/axym/core/override"
	"github.com/Clyra-AI/axym/core/store"
	"github.com/Clyra-AI/proof"
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

func TestBuildEmitsDerivedArtifactsFromGaitBundles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	outDir := filepath.Join(root, "bundle")

	ingestGaitControlFixture(t, storeDir)

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
		"authorization-register.json",
		"insurance-evidence-profile.json",
		"credential-posture-register.json",
		"freeze-window-coverage.json",
		"kill-switch-coverage.json",
		"enforcement-explain-register.json",
		"sandbox-coverage.json",
		"control-maturity.json",
	}
	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(outDir, rel)); err != nil {
			t.Fatalf("missing derived artifact %s: %v", rel, err)
		}
	}
}

func ingestGaitControlFixture(t *testing.T, storeDir string) {
	t.Helper()

	packDir := filepath.Join(t.TempDir(), "gait-pack")
	if err := os.MkdirAll(packDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	linkedRecordID := writeProofFixture(t, filepath.Join(packDir, "proof_records.jsonl"))
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

	st, err := store.New(store.Config{RootDir: storeDir})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	if _, err := ingestgait.Ingest(context.Background(), ingestgait.Request{
		Store:      st,
		InputPaths: []string{packDir},
	}); err != nil {
		t.Fatalf("gait.Ingest: %v", err)
	}
}

func writeProofFixture(t *testing.T, path string) string {
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
		t.Fatalf("marshal record: %v", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		t.Fatalf("write proof fixture: %v", err)
	}
	return record.RecordID
}
