package pack

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/ingest/gait/evidence"
	"github.com/Clyra-AI/axym/core/ingest/gait/translate"
	"github.com/Clyra-AI/proof"
)

func TestReadDirectoryPack(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProofFile(t, filepath.Join(dir, "proof_records.jsonl"))
	if err := os.WriteFile(filepath.Join(dir, "native_records.jsonl"), []byte(`{"type":"trace","timestamp":"2026-02-28T21:00:00Z","event":{"tool_name":"planner"}}`+"\n"), 0o600); err != nil {
		t.Fatalf("write native file: %v", err)
	}

	result, err := Read(dir)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(result.ProofRecords) != 1 || len(result.NativeRecords) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestReadZipPack(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	zipPath := filepath.Join(root, "pack.zip")

	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	writer := zip.NewWriter(file)

	proofEntry, err := writer.Create("proof_records.jsonl")
	if err != nil {
		t.Fatalf("create proof entry: %v", err)
	}
	proofLine := buildProofLine(t)
	if _, err := proofEntry.Write([]byte(proofLine + "\n")); err != nil {
		t.Fatalf("write proof entry: %v", err)
	}

	nativeEntry, err := writer.Create("native_records.jsonl")
	if err != nil {
		t.Fatalf("create native entry: %v", err)
	}
	if _, err := nativeEntry.Write([]byte(`{"type":"approval_token","timestamp":"2026-02-28T21:01:00Z","event":{"decision":"allow"}}` + "\n")); err != nil {
		t.Fatalf("write native entry: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close zip file: %v", err)
	}

	result, err := Read(zipPath)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(result.ProofRecords) != 1 || len(result.NativeRecords) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestReadDirectoryDetectsAuthorizationBundle(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	authDir := filepath.Join(dir, "authorization")
	if err := os.MkdirAll(authDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	writeProofFile(t, filepath.Join(dir, "proof_records.jsonl"))
	writeAuthorizationArtifact(t, filepath.Join(authDir, "bundle.json"), map[string]any{
		"artifact_type":       "gate_authorization_bundle",
		"artifact_id":         "auth-artifact-1",
		"bundle_id":           "bundle-1",
		"timestamp":           "2026-05-12T17:00:00Z",
		"decision":            "allow",
		"trace_id":            "trace-1",
		"intent_digest":       "sha256:intent-1",
		"policy_digest":       "sha256:policy-1",
		"schema_version":      "v1",
		"approval_audit_refs": []string{"approval://chg-1"},
	})

	result, err := Read(dir)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(result.Artifacts) != 1 {
		t.Fatalf("expected 1 source artifact, got %+v", result.Artifacts)
	}
	if result.Artifacts[0].Kind() != translate.ArtifactTypeAuthorizationBundle {
		t.Fatalf("artifact kind mismatch: %+v", result.Artifacts[0])
	}
	if got := result.Artifacts[0].Base().SourceArtifactPath; got != "authorization/bundle.json" {
		t.Fatalf("source path mismatch: got %q", got)
	}
}

func TestReadZipOrdersAuthorizationBundlesDeterministically(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	zipPath := filepath.Join(root, "pack.zip")

	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	writer := zip.NewWriter(file)
	for _, item := range []struct {
		name     string
		bundleID string
	}{
		{name: "z/bundle-z.json", bundleID: "bundle-z"},
		{name: "a/bundle-a.json", bundleID: "bundle-a"},
	} {
		entry, err := writer.Create(item.name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		payload, err := json.Marshal(map[string]any{
			"artifact_type": "gate_authorization_bundle",
			"artifact_id":   item.bundleID + "-artifact",
			"bundle_id":     item.bundleID,
			"timestamp":     "2026-05-12T17:00:00Z",
			"decision":      "allow",
		})
		if err != nil {
			t.Fatalf("marshal auth payload: %v", err)
		}
		if _, err := entry.Write(payload); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close zip file: %v", err)
	}

	result, err := Read(zipPath)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	got := make([]string, 0, len(result.Artifacts))
	for _, artifact := range result.Artifacts {
		got = append(got, artifact.BundleID())
	}
	want := append([]string(nil), got...)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("expected deterministic sorted bundle ids, got=%v want=%v", got, want)
	}
}

func TestReadRejectsDuplicateAuthorizationBundleIDs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeAuthorizationArtifact(t, filepath.Join(dir, "bundle-a.json"), map[string]any{
		"artifact_type": "gate_authorization_bundle",
		"artifact_id":   "artifact-a",
		"bundle_id":     "duplicate-bundle",
		"timestamp":     "2026-05-12T17:00:00Z",
	})
	writeAuthorizationArtifact(t, filepath.Join(dir, "bundle-b.json"), map[string]any{
		"artifact_type": "gate_authorization_bundle",
		"artifact_id":   "artifact-b",
		"bundle_id":     "duplicate-bundle",
		"timestamp":     "2026-05-12T17:01:00Z",
	})

	_, err := Read(dir)
	if err == nil {
		t.Fatal("expected duplicate bundle id error")
	}
	if !strings.Contains(err.Error(), translate.ReasonDuplicateAuthorizationBundleID) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadDirectoryPackRejectsEmptyDirectory(t *testing.T) {
	t.Parallel()

	_, err := Read(t.TempDir())
	if err == nil {
		t.Fatal("expected error for empty gait pack directory")
	}
}

func TestReadLifecycleSchemaErrorsAreTyped(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, raw, reason string
	}{
		{"empty", `{"records":[]}`, evidence.ReasonEvidenceMissing},
		{"unknown", `{"unknown":true,"records":[]}`, evidence.ReasonUnknownField},
		{"duplicate", `{"records":[],"records":[]}`, evidence.ReasonMalformed},
		{"malformed", `{`, evidence.ReasonMalformed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "lifecycle.json")
			if err := os.WriteFile(path, []byte(tc.raw), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := Read(path)
			lifecycleErr, ok := err.(*LifecycleError)
			if !ok || lifecycleErr.ReasonCode != tc.reason {
				t.Fatalf("unexpected typed error: %T %v", err, err)
			}
		})
	}
}

func TestReadDirectoryDetectsLifecycleEvidence(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "..", "..", "testdata", "gait-action-contract-evidence", "v1")
	packDir := t.TempDir()
	raw, err := os.ReadFile(filepath.Join(root, "successful-execution-effect-containment", "lifecycle.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packDir, "lifecycle.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := Read(packDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.LifecyclePacks) != 1 || len(result.LifecyclePacks[0].Records) != 11 {
		t.Fatalf("lifecycle pack mismatch: %+v", result)
	}
}

func writeProofFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(buildProofLine(t)+"\n"), 0o600); err != nil {
		t.Fatalf("write proof file: %v", err)
	}
}

func buildProofLine(t *testing.T) string {
	t.Helper()
	record, err := proof.NewRecord(proof.RecordOpts{
		Timestamp:     time.Date(2026, 2, 28, 21, 0, 0, 0, time.UTC),
		Source:        "gait",
		SourceProduct: "gait",
		Type:          "approval",
		AgentID:       "agent://gait",
		Event: map[string]any{
			"decision": "allow",
		},
		Controls: proof.Controls{PermissionsEnforced: true},
	})
	if err != nil {
		t.Fatalf("NewRecord: %v", err)
	}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	return strings.TrimSpace(string(raw))
}

func writeAuthorizationArtifact(t *testing.T, path string, payload map[string]any) {
	t.Helper()
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal auth artifact: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write auth artifact: %v", err)
	}
}
