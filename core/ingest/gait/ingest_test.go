package gait

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/ingest/gait/evidence"
	"github.com/Clyra-AI/axym/core/ingest/gait/translate"
	"github.com/Clyra-AI/axym/core/store"
	"github.com/Clyra-AI/proof"
)

func TestIngestNoInputReturnsNoInputReason(t *testing.T) {
	t.Parallel()

	st, err := store.New(store.Config{RootDir: filepath.Join(t.TempDir(), "store")})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	result, err := Ingest(context.Background(), Request{Store: st})
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if len(result.ReasonCodes) != 1 || result.ReasonCodes[0] != ReasonNoInput {
		t.Fatalf("reason mismatch: %+v", result)
	}
}

func TestIngestRejectsLifecycleWithoutExplicitVerification(t *testing.T) {
	t.Parallel()
	st, err := store.New(store.Config{RootDir: filepath.Join(t.TempDir(), "store")})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Ingest(context.Background(), Request{Store: st, InputPaths: []string{lifecycleFixturePath(t)}})
	if err == nil {
		t.Fatal("expected lifecycle verification requirement")
	}
	gaitErr, ok := err.(*Error)
	if !ok || gaitErr.ReasonCode != ReasonLifecycleVerificationRequired {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIngestVerifiesFixtureLifecycleWithoutAppendingAuthority(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "..", "testdata", "gait-action-contract-evidence", "v1")
	packRaw, err := os.ReadFile(lifecycleFixturePath(t))
	if err != nil {
		t.Fatal(err)
	}
	pack, err := evidence.ParseLifecyclePack(packRaw)
	if err != nil {
		t.Fatal(err)
	}
	keyRaw, err := os.ReadFile(filepath.Join(root, "fixture-signing-key.public.b64"))
	if err != nil {
		t.Fatal(err)
	}
	keyBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(keyRaw)))
	if err != nil {
		t.Fatal(err)
	}
	options := lifecycleFixtureVerificationOptions(pack, ed25519.PublicKey(keyBytes))
	st, err := store.New(store.Config{RootDir: filepath.Join(t.TempDir(), "store")})
	if err != nil {
		t.Fatal(err)
	}
	result, err := Ingest(context.Background(), Request{Store: st, InputPaths: []string{lifecycleFixturePath(t)}, LifecycleVerification: &options})
	if err != nil {
		t.Fatal(err)
	}
	if result.LifecycleParsed != 1 || result.LifecycleVerified != 1 || len(result.LifecycleEvidenceSets) != 1 {
		t.Fatalf("verification result mismatch: %+v", result)
	}
	if result.LifecycleAuthoritative != 0 || result.LifecycleTranslated != 0 || result.LifecycleEvidenceSets[0].Authoritative || !result.LifecycleEvidenceSets[0].FixtureOnly || result.Appended != 0 {
		t.Fatalf("fixture authority or append leak: %+v", result)
	}
	chain, err := st.LoadChain()
	if err != nil {
		t.Fatal(err)
	}
	if len(chain.Records) != 0 {
		t.Fatalf("lifecycle evidence became proof records: %d", len(chain.Records))
	}
}

func lifecycleFixturePath(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "..", "testdata", "gait-action-contract-evidence", "v1", "successful-execution-effect-containment", "lifecycle.json")
}

func lifecycleFixtureVerificationOptions(pack evidence.LifecyclePack, key ed25519.PublicKey) evidence.VerificationOptions {
	first := pack.Records[0]
	activation := evidence.Ref{}
	for _, record := range pack.Records {
		if record.ActivationRef != nil {
			activation = *record.ActivationRef
			break
		}
	}
	return evidence.VerificationOptions{
		TrustedPublicKey: key, EvaluationTime: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), AllowFixtureOnly: true,
		ExpectedContract: first.ContractRef, ExpectedFamily: first.ContractFamilyID, ExpectedRevision: first.Revision, ExpectedActivation: activation,
		ExpectedRuntimeDigest: "sha256:ffdb7187847ee43434cf0bc428d9defc9b407da4595be1bdfab4c16a47a801e1", ExpectedReadinessDigest: "sha256:5537a606ce771336b50c0f6f6ca978d8d310cb7e8d59eff47d7ac698264b4305",
		ExpectedPolicyDigest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", ExpectedTarget: "target:fixture", ExpectedEnvironment: "test",
		ExpectedProducerVersion: evidence.FixtureTag, ExpectedSourceCommit: evidence.FixtureCommit, ExpectedLifecycleDigest: pack.SourceArtifactDigest,
		ActivationNotBefore: time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC), ActivationNotAfter: time.Date(2027, 7, 19, 0, 0, 0, 0, time.UTC),
	}
}

func TestIngestMixesPassthroughAndNativeRecords(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	st, err := store.New(store.Config{RootDir: filepath.Join(root, "store")})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	packDir := filepath.Join(root, "pack")
	if err := os.MkdirAll(packDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	writeProofJSONL(t, filepath.Join(packDir, "proof_records.jsonl"))
	if err := os.WriteFile(filepath.Join(packDir, "native_records.jsonl"), []byte(`{"type":"trace","timestamp":"2026-02-28T21:30:00Z","event":{"tool_name":"planner"}}`+"\n"), 0o600); err != nil {
		t.Fatalf("write native: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packDir, "authorization-bundle.json"), []byte(`{
  "artifact_type": "gate_authorization_bundle",
  "artifact_id": "artifact-1",
  "bundle_id": "bundle-1",
  "timestamp": "2026-02-28T21:31:00Z",
  "decision": "allow",
  "trace_id": "trace-1",
  "intent_digest": "sha256:intent-1",
  "policy_digest": "sha256:policy-1",
  "schema_version": "v1",
  "credential_posture": {
    "standing_credential_blocked": true,
    "jit_required": true,
    "broker_source": "gait-broker",
    "issuer": "issuer-a",
    "ttl_seconds": 300,
    "scope": "payments.write",
    "binding_proof_ref": "binding://proof-1"
  }
}`), 0o600); err != nil {
		t.Fatalf("write auth artifact: %v", err)
	}

	result, err := Ingest(context.Background(), Request{
		Store:      st,
		InputPaths: []string{packDir},
	})
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if result.Appended != 4 || result.Passthrough != 1 || result.Translated != 1 || result.SourceArtifactsTranslated != 2 {
		t.Fatalf("result mismatch: %+v", result)
	}
	if result.AuthorizationBundles != 1 || result.ControlArtifacts != 1 {
		t.Fatalf("expected source artifact counts in result: %+v", result)
	}
}

func TestIngestRejectsUnverifiableLinkedSourceArtifactRefs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	st, err := store.New(store.Config{RootDir: filepath.Join(root, "store")})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	packDir := filepath.Join(root, "pack")
	if err := os.MkdirAll(packDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	writeProofJSONL(t, filepath.Join(packDir, "proof_records.jsonl"))
	if err := os.WriteFile(filepath.Join(packDir, "authorization-bundle.json"), []byte(`{
  "artifact_type": "gate_authorization_bundle",
  "artifact_id": "artifact-1",
  "bundle_id": "bundle-1",
  "timestamp": "2026-02-28T21:31:00Z",
  "linked_record_ids": ["missing-record-id"]
}`), 0o600); err != nil {
		t.Fatalf("write auth artifact: %v", err)
	}

	_, err = Ingest(context.Background(), Request{
		Store:      st,
		InputPaths: []string{packDir},
	})
	if err == nil {
		t.Fatal("expected unverifiable link error")
	}
	gaitErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if gaitErr.ReasonCode != translate.ReasonUnverifiableLinkedRefs {
		t.Fatalf("reason mismatch: got %s", gaitErr.ReasonCode)
	}
}

func writeProofJSONL(t *testing.T, path string) {
	t.Helper()
	record, err := proof.NewRecord(proof.RecordOpts{
		Timestamp:     time.Date(2026, 2, 28, 21, 29, 0, 0, time.UTC),
		Source:        "wrkr",
		SourceProduct: "wrkr",
		Type:          "approval",
		Event: map[string]any{
			"decision": "allow",
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
		t.Fatalf("write: %v", err)
	}
}
