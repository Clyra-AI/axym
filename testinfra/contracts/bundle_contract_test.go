package contracts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	verifysupport "github.com/Clyra-AI/axym/core/verifysupport"
	"github.com/Clyra-AI/proof"
)

func TestBundleAndVerifyBundleJSONEnvelopeContract(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	bundleDir := filepath.Join(t.TempDir(), "bundle")

	collectOut, collectExit := runAxymContract(t,
		"collect",
		"--fixture-dir", filepath.Join(testRepoRoot(t), "fixtures", "collectors"),
		"--store-dir", storeDir,
		"--json",
	)
	if collectExit != 0 {
		t.Fatalf("collect setup failed: exit=%d output=%s", collectExit, collectOut)
	}

	bundleOut, bundleExit := runAxymContract(t,
		"bundle",
		"--audit", "Q3-2026",
		"--frameworks", "eu-ai-act,soc2",
		"--store-dir", storeDir,
		"--output", bundleDir,
		"--json",
	)
	if bundleExit != 0 {
		t.Fatalf("bundle failed: exit=%d output=%s", bundleExit, bundleOut)
	}
	var bundlePayload map[string]any
	if err := json.Unmarshal([]byte(bundleOut), &bundlePayload); err != nil {
		t.Fatalf("decode bundle output: %v output=%s", err, bundleOut)
	}
	if bundlePayload["command"] != "bundle" {
		t.Fatalf("bundle command mismatch: %s", bundleOut)
	}
	if bundlePayload["ok"] != true {
		t.Fatalf("bundle expected ok=true: %s", bundleOut)
	}

	verifyOut, verifyExit := runAxymContract(t, "verify", "--bundle", bundleDir, "--json")
	if verifyExit != 0 {
		t.Fatalf("verify bundle failed: exit=%d output=%s", verifyExit, verifyOut)
	}
	var verifyPayload map[string]any
	if err := json.Unmarshal([]byte(verifyOut), &verifyPayload); err != nil {
		t.Fatalf("decode verify output: %v output=%s", err, verifyOut)
	}
	if verifyPayload["command"] != "verify" {
		t.Fatalf("verify command mismatch: %s", verifyOut)
	}
	data, _ := verifyPayload["data"].(map[string]any)
	verification, _ := data["verification"].(map[string]any)
	if verification["cryptographic"] != true {
		t.Fatalf("expected cryptographic=true: %s", verifyOut)
	}
	if verification["compliance_verified"] != true {
		t.Fatalf("expected compliance_verified=true: %s", verifyOut)
	}
	compliance, _ := verification["compliance"].(map[string]any)
	if _, ok := compliance["identity_governance"].(map[string]any); !ok {
		t.Fatalf("expected identity_governance envelope: %s", verifyOut)
	}
}

func TestVerifyBundleInvalidOSCALContractExit(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	bundleDir := filepath.Join(t.TempDir(), "bundle")
	if out, exit := runAxymContract(t,
		"collect",
		"--fixture-dir", filepath.Join(testRepoRoot(t), "fixtures", "collectors"),
		"--store-dir", storeDir,
		"--json",
	); exit != 0 {
		t.Fatalf("collect setup failed: exit=%d output=%s", exit, out)
	}
	if out, exit := runAxymContract(t,
		"bundle",
		"--audit", "Q3-2026",
		"--frameworks", "eu-ai-act,soc2",
		"--store-dir", storeDir,
		"--output", bundleDir,
		"--json",
	); exit != 0 {
		t.Fatalf("bundle setup failed: exit=%d output=%s", exit, out)
	}
	oscalPath := filepath.Join(bundleDir, "oscal-v1.1", "component-definition.json")
	if err := os.WriteFile(oscalPath, []byte(`{"bad":true}`), 0o600); err != nil {
		t.Fatalf("tamper oscal: %v", err)
	}
	if err := updateManifestHash(bundleDir, filepath.ToSlash(filepath.Join("oscal-v1.1", "component-definition.json"))); err != nil {
		t.Fatalf("update manifest hash: %v", err)
	}
	if err := resignManifest(storeDir, bundleDir); err != nil {
		t.Fatalf("resign manifest: %v", err)
	}
	out, exit := runAxymContract(t, "verify", "--bundle", bundleDir, "--json")
	if exit != 3 {
		t.Fatalf("exit mismatch: got=%d want=3 output=%s", exit, out)
	}
}

func TestBundleAndVerifyDerivedArtifactContract(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	bundleDir := filepath.Join(t.TempDir(), "bundle")
	packDir := createGaitContractPack(t, t.TempDir())

	ingestOut, ingestExit := runAxymContract(t,
		"ingest",
		"--gait-pack", packDir,
		"--store-dir", storeDir,
		"--json",
	)
	if ingestExit != 0 {
		t.Fatalf("ingest failed: exit=%d output=%s", ingestExit, ingestOut)
	}
	var ingestPayload map[string]any
	if err := json.Unmarshal([]byte(ingestOut), &ingestPayload); err != nil {
		t.Fatalf("decode ingest output: %v output=%s", err, ingestOut)
	}
	ingestData, _ := ingestPayload["data"].(map[string]any)
	ingestResult, _ := ingestData["result"].(map[string]any)
	if ingestResult["authorization_bundles"] != float64(1) {
		t.Fatalf("expected authorization bundle count output=%s", ingestOut)
	}

	bundleOut, bundleExit := runAxymContract(t,
		"bundle",
		"--audit", "Q3-2026",
		"--frameworks", "eu-ai-act,soc2",
		"--store-dir", storeDir,
		"--output", bundleDir,
		"--json",
	)
	if bundleExit != 0 {
		t.Fatalf("bundle failed: exit=%d output=%s", bundleExit, bundleOut)
	}
	for _, rel := range []string{
		"authorization-register.json",
		"insurance-evidence-profile.json",
		"credential-posture-register.json",
		"freeze-window-coverage.json",
		"kill-switch-coverage.json",
		"enforcement-explain-register.json",
		"sandbox-coverage.json",
		"control-maturity.json",
	} {
		if _, err := os.Stat(filepath.Join(bundleDir, rel)); err != nil {
			t.Fatalf("expected derived artifact %s: %v", rel, err)
		}
	}

	verifyOut, verifyExit := runAxymContract(t, "verify", "--bundle", bundleDir, "--json")
	if verifyExit != 0 {
		t.Fatalf("verify bundle failed: exit=%d output=%s", verifyExit, verifyOut)
	}
	var verifyPayload map[string]any
	if err := json.Unmarshal([]byte(verifyOut), &verifyPayload); err != nil {
		t.Fatalf("decode verify output: %v output=%s", err, verifyOut)
	}
	verifyData, _ := verifyPayload["data"].(map[string]any)
	verification, _ := verifyData["verification"].(map[string]any)
	derivedArtifacts, _ := verification["derived_artifacts"].([]any)
	if len(derivedArtifacts) == 0 {
		t.Fatalf("expected derived_artifacts in verify output: %s", verifyOut)
	}
}

func TestVerifyDerivedArtifactSchemaViolationContractExit(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	bundleDir := filepath.Join(t.TempDir(), "bundle")
	packDir := createGaitContractPack(t, t.TempDir())

	if out, exit := runAxymContract(t, "ingest", "--gait-pack", packDir, "--store-dir", storeDir, "--json"); exit != 0 {
		t.Fatalf("ingest setup failed: exit=%d output=%s", exit, out)
	}
	if out, exit := runAxymContract(t,
		"bundle",
		"--audit", "Q3-2026",
		"--frameworks", "eu-ai-act,soc2",
		"--store-dir", storeDir,
		"--output", bundleDir,
		"--json",
	); exit != 0 {
		t.Fatalf("bundle setup failed: exit=%d output=%s", exit, out)
	}
	invalidPath := filepath.Join(bundleDir, "authorization-register.json")
	if err := os.WriteFile(invalidPath, []byte(`{"version":"v1","entries":{}}`), 0o600); err != nil {
		t.Fatalf("write invalid authorization register: %v", err)
	}
	if err := updateManifestHash(bundleDir, "authorization-register.json"); err != nil {
		t.Fatalf("update manifest hash: %v", err)
	}
	if err := resignManifest(storeDir, bundleDir); err != nil {
		t.Fatalf("resign manifest: %v", err)
	}
	out, exit := runAxymContract(t, "verify", "--bundle", bundleDir, "--json")
	if exit != 3 {
		t.Fatalf("exit mismatch: got=%d want=3 output=%s", exit, out)
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
	var manifest struct {
		Files []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"files"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return err
	}
	for i := range manifest.Files {
		if manifest.Files[i].Path == relPath {
			manifest.Files[i].SHA256 = want
		}
	}
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
	var manifest map[string]any
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return err
	}
	delete(manifest, "signatures")
	cleaned, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(manifestPath, cleaned, 0o600); err != nil {
		return err
	}
	_, err = proof.SignBundleFile(bundleDir, signingKey)
	return err
}

func createGaitContractPack(t *testing.T, root string) string {
	t.Helper()

	packDir := filepath.Join(root, "gait-pack")
	if err := os.MkdirAll(packDir, 0o700); err != nil {
		t.Fatalf("mkdir gait pack: %v", err)
	}
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
	rawProof, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal proof fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packDir, "proof_records.jsonl"), append(rawProof, '\n'), 0o600); err != nil {
		t.Fatalf("write proof_records.jsonl: %v", err)
	}
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
}`, record.RecordID)
	if err := os.WriteFile(filepath.Join(packDir, "authorization-bundle.json"), []byte(authPayload), 0o600); err != nil {
		t.Fatalf("write authorization bundle: %v", err)
	}
	return packDir
}
