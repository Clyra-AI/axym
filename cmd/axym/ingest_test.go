package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/ingest/gait"
	gaitevidence "github.com/Clyra-AI/axym/core/ingest/gait/evidence"
	"github.com/Clyra-AI/proof"
)

func TestIngestRequiresSource(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"ingest", "--json"}, &stdout, &stderr)
	if exit != exitInvalidInput {
		t.Fatalf("exit mismatch: got %d want %d stdout=%s", exit, exitInvalidInput, stdout.String())
	}
}

func TestIngestWrkrNoInputContract(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"ingest", "--source", "wrkr", "--store-dir", storeDir, "--state-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	data, _ := payload["data"].(map[string]any)
	result, _ := data["result"].(map[string]any)
	reasons, _ := result["reason_codes"].([]any)
	if len(reasons) != 1 || reasons[0] != "NO_INPUT" {
		t.Fatalf("reason mismatch: %s", stdout.String())
	}
}

func TestIngestWrkrAppendsRecords(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	inputPath := filepath.Join(root, "wrkr.jsonl")
	writeWrkrInput(t, inputPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{
		"ingest",
		"--source", "wrkr",
		"--input", inputPath,
		"--store-dir", storeDir,
		"--state-dir", root,
		"--json",
	}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	data, _ := payload["data"].(map[string]any)
	result, _ := data["result"].(map[string]any)
	if appended, _ := result["appended"].(float64); appended != 1 {
		t.Fatalf("appended mismatch: %s", stdout.String())
	}
}

func TestIngestWrkrUsesStoreDirForStateWhenStateDirOmitted(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	inputPath := filepath.Join(root, "wrkr.jsonl")
	writeWrkrInput(t, inputPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{
		"ingest",
		"--source", "wrkr",
		"--input", inputPath,
		"--store-dir", storeDir,
		"--json",
	}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}

	if _, err := os.Stat(filepath.Join(storeDir, "wrkr-last-ingest.json")); err != nil {
		t.Fatalf("expected wrkr state in store dir: %v", err)
	}
}

func TestIngestGaitNoInputContract(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"ingest", "--source", "gait", "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	data, _ := payload["data"].(map[string]any)
	result, _ := data["result"].(map[string]any)
	reasons, _ := result["reason_codes"].([]any)
	if len(reasons) != 1 || reasons[0] != "NO_INPUT" {
		t.Fatalf("reason mismatch: %s", stdout.String())
	}
}

func TestIngestGaitLifecycleUsesExplicitCallerVerificationConfig(t *testing.T) {
	t.Parallel()
	fixtureRoot := filepath.Join("..", "..", "testdata", "gait-action-contract-evidence", "v1")
	lifecyclePath := filepath.Join(fixtureRoot, "successful-execution-effect-containment", "lifecycle.json")
	lifecycleRaw, err := os.ReadFile(lifecyclePath)
	if err != nil {
		t.Fatal(err)
	}
	pack, err := gaitevidence.ParseLifecyclePack(lifecycleRaw)
	if err != nil {
		t.Fatal(err)
	}
	keyRaw, err := os.ReadFile(filepath.Join(fixtureRoot, "fixture-signing-key.public.b64"))
	if err != nil {
		t.Fatal(err)
	}
	first := pack.Records[0]
	activation := gaitevidence.Ref{}
	for _, record := range pack.Records {
		if record.ActivationRef != nil {
			activation = *record.ActivationRef
			break
		}
	}
	config := map[string]any{
		"trusted_public_key": strings.TrimSpace(string(keyRaw)), "allow_fixture_only": true,
		"expected_contract": first.ContractRef, "expected_family": first.ContractFamilyID, "expected_revision": first.Revision, "expected_activation": activation,
		"expected_runtime_digest": "sha256:ffdb7187847ee43434cf0bc428d9defc9b407da4595be1bdfab4c16a47a801e1", "expected_readiness_digest": "sha256:5537a606ce771336b50c0f6f6ca978d8d310cb7e8d59eff47d7ac698264b4305",
		"expected_policy_digest": "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "expected_target": "target:fixture", "expected_environment": "test",
		"expected_producer_version": gaitevidence.FixtureTag, "expected_source_commit": gaitevidence.FixtureCommit, "expected_lifecycle_digest": pack.SourceArtifactDigest,
		"evaluation_time": "2026-07-20T00:00:00Z", "activation_not_before": "2026-07-19T00:00:00Z", "activation_not_after": "2027-07-19T00:00:00Z",
	}
	configRaw, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	configPath := filepath.Join(root, "verification.json")
	if err := os.WriteFile(configPath, configRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"ingest", "--source", "gait", "--input", lifecyclePath, "--gait-lifecycle-verification", configPath, "--store-dir", filepath.Join(root, "store"), "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	data, _ := payload["data"].(map[string]any)
	result, _ := data["result"].(map[string]any)
	if result["lifecycle_verified"] != float64(1) || result["lifecycle_authoritative"] != float64(0) || result["lifecycle_translated"] != float64(0) || result["appended"] != float64(0) {
		t.Fatalf("fixture lifecycle authority leaked: %s", stdout.String())
	}
}

func TestIngestRejectsLifecycleVerificationConfigForWrkr(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"ingest", "--source", "wrkr", "--gait-lifecycle-verification", filepath.Join(t.TempDir(), "unused.json"), "--json"}, &stdout, &stderr)
	if exit != exitInvalidInput {
		t.Fatalf("exit mismatch: got %d stdout=%s stderr=%s", exit, stdout.String(), stderr.String())
	}
}

func TestIngestRejectsLifecycleVerificationBeforeWrkrMutation(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	inputPath := filepath.Join(root, "wrkr.jsonl")
	storeDir := filepath.Join(root, "store")
	stateDir := filepath.Join(root, "state")
	writeWrkrInput(t, inputPath)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"ingest", "--source", "wrkr", "--input", inputPath, "--store-dir", storeDir, "--state-dir", stateDir, "--gait-lifecycle-verification", filepath.Join(root, "unused.json"), "--json"}, &stdout, &stderr)
	if exit != exitInvalidInput {
		t.Fatalf("exit mismatch: got %d want %d stdout=%s stderr=%s", exit, exitInvalidInput, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(storeDir); !os.IsNotExist(err) {
		t.Fatalf("store was initialized or mutated: err=%v", err)
	}
	if _, err := os.Stat(stateDir); !os.IsNotExist(err) {
		t.Fatalf("Wrkr state was initialized or mutated: err=%v", err)
	}
}

func TestIngestLifecycleVerificationFailureReturnsExitTwoAndStableReasons(t *testing.T) {
	t.Parallel()
	fixtureRoot := filepath.Join("..", "..", "testdata", "gait-action-contract-evidence", "v1")
	lifecyclePath := filepath.Join(fixtureRoot, "successful-execution-effect-containment", "lifecycle.json")
	lifecycleRaw, err := os.ReadFile(lifecyclePath)
	if err != nil {
		t.Fatal(err)
	}
	pack, err := gaitevidence.ParseLifecyclePack(lifecycleRaw)
	if err != nil {
		t.Fatal(err)
	}
	keyRaw, err := os.ReadFile(filepath.Join(fixtureRoot, "fixture-signing-key.public.b64"))
	if err != nil {
		t.Fatal(err)
	}
	first := pack.Records[0]
	activation := gaitevidence.Ref{}
	for _, record := range pack.Records {
		if record.ActivationRef != nil {
			activation = *record.ActivationRef
			break
		}
	}
	config := map[string]any{
		"trusted_public_key": strings.TrimSpace(string(keyRaw)), "allow_fixture_only": true,
		"expected_contract": first.ContractRef, "expected_family": first.ContractFamilyID, "expected_revision": first.Revision, "expected_activation": activation,
		"expected_runtime_digest": "sha256:ffdb7187847ee43434cf0bc428d9defc9b407da4595be1bdfab4c16a47a801e1", "expected_readiness_digest": "sha256:5537a606ce771336b50c0f6f6ca978d8d310cb7e8d59eff47d7ac698264b4305",
		"expected_policy_digest": "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "expected_target": "target:fixture", "expected_environment": "test",
		"expected_producer_version": gaitevidence.FixtureTag, "expected_source_commit": gaitevidence.FixtureCommit, "expected_lifecycle_digest": pack.SourceArtifactDigest,
		"evaluation_time": "2026-07-20T00:00:00Z", "activation_not_before": "2026-07-19T00:00:00Z", "activation_not_after": "2027-07-19T00:00:00Z",
	}
	configRaw, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	tamperedDir := filepath.Join(root, "tampered-pack")
	if err := os.MkdirAll(tamperedDir, 0o700); err != nil {
		t.Fatal(err)
	}
	tamperedPath := filepath.Join(tamperedDir, "lifecycle.json")
	tamperedRaw := bytes.Replace(lifecycleRaw, []byte("gait-lr-56341bd700fb0d35"), []byte("gait-lr-56341bd700fb0d36"), 1)
	if err := os.WriteFile(tamperedPath, tamperedRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "verification.json")
	if err := os.WriteFile(configPath, configRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"ingest", "--source", "gait", "--input", tamperedPath, "--gait-lifecycle-verification", configPath, "--store-dir", filepath.Join(root, "store"), "--json"}, &stdout, &stderr)
	if exit != exitVerificationFailed {
		t.Fatalf("exit mismatch: got %d want %d stdout=%s stderr=%s", exit, exitVerificationFailed, stdout.String(), stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	errObj, _ := payload["error"].(map[string]any)
	if errObj["reason"] != gait.ReasonLifecycleVerificationFailed {
		t.Fatalf("reason mismatch: %s", stdout.String())
	}
	reasons, _ := errObj["reason_codes"].([]any)
	found := false
	for _, reason := range reasons {
		if reason == gaitevidence.ReasonSourceProvenanceInvalid {
			found = true
		}
	}
	if !found {
		t.Fatalf("stable verifier reason missing: %s", stdout.String())
	}
}

func TestIngestGaitPackAliasReportsAuthorizationBundleCounts(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	inputDir := filepath.Join(root, "gait")
	if err := os.MkdirAll(inputDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(inputDir, "authorization-bundle.json"), []byte(`{
  "artifact_type": "gate_authorization_bundle",
  "artifact_id": "artifact-1",
  "bundle_id": "bundle-1",
  "timestamp": "2026-05-12T17:00:00Z",
  "decision": "allow",
  "credential_posture": {
    "standing_credential_blocked": true,
    "jit_required": true,
    "broker_source": "gait-broker",
    "issuer": "issuer-a",
    "ttl_seconds": 600,
    "scope": "payments.write",
    "binding_proof_ref": "binding://proof-1"
  }
}`), 0o600); err != nil {
		t.Fatalf("write auth artifact: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"ingest", "--gait-pack", inputDir, "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	data, _ := payload["data"].(map[string]any)
	result, _ := data["result"].(map[string]any)
	if result["authorization_bundles"] != float64(1) {
		t.Fatalf("expected authorization_bundles=1 output=%s", stdout.String())
	}
	if result["control_artifacts"] != float64(1) {
		t.Fatalf("expected control_artifacts=1 output=%s", stdout.String())
	}
}

func TestIngestGaitPackRejectsUnsupportedStandaloneJSONAsInvalidInput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	inputPath := filepath.Join(root, "unsupported.json")
	if err := os.WriteFile(inputPath, []byte(`{"hello":"world"}`), 0o600); err != nil {
		t.Fatalf("write unsupported input: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"ingest", "--gait-pack", inputPath, "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitInvalidInput {
		t.Fatalf("exit mismatch: got %d want %d stderr=%s stdout=%s", exit, exitInvalidInput, stderr.String(), stdout.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v output=%s", err, stdout.String())
	}
	errObj, _ := payload["error"].(map[string]any)
	if errObj["reason"] != "GAIT_INVALID_AUTHORIZATION_ARTIFACT" {
		t.Fatalf("reason mismatch: got %v output=%s", errObj["reason"], stdout.String())
	}
}

func TestIngestUsesCommandContextCancellation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	inputDir := filepath.Join(root, "gait")
	if err := os.MkdirAll(inputDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(inputDir, "native_records.jsonl"), []byte("{\"type\":\"trace\",\"timestamp\":\"2026-02-28T23:05:00Z\",\"agent_id\":\"agent://executor\",\"event\":{\"tool_name\":\"planner\"}}\n"), 0o600); err != nil {
		t.Fatalf("write native records: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := executeContext(ctx, []string{"ingest", "--source", "gait", "--input", inputDir, "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitRuntimeFailure {
		t.Fatalf("exit mismatch: got %d want %d stdout=%s stderr=%s", exit, exitRuntimeFailure, stdout.String(), stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v output=%s", err, stdout.String())
	}
	errObj, _ := payload["error"].(map[string]any)
	if errObj["reason"] != "GAIT_CONTEXT_CANCELED" {
		t.Fatalf("reason mismatch: got %v output=%s", errObj["reason"], stdout.String())
	}
}

func writeWrkrInput(t *testing.T, path string) {
	t.Helper()
	record, err := proof.NewRecord(proof.RecordOpts{
		Timestamp:     time.Date(2026, 2, 28, 20, 0, 0, 0, time.UTC),
		Source:        "wrkr",
		SourceProduct: "wrkr",
		Type:          "scan_finding",
		AgentID:       "agent-a",
		Event: map[string]any{
			"finding_id":   "finding-1",
			"principal_id": "agent-a",
			"privilege":    "read",
			"approved":     true,
		},
		Metadata: map[string]any{
			"principal_id": "agent-a",
			"scope":        "read",
		},
		Controls: proof.Controls{PermissionsEnforced: true},
	})
	if err != nil {
		t.Fatalf("proof.NewRecord: %v", err)
	}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(string(raw))+"\n"), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}
}
