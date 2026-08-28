package actioncontract

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestActivationVerificationReceiptSchemaIsEmbeddedAndOfflineValidatable(t *testing.T) {
	raw := []byte(`{"schema_id":"https://axym.dev/schemas/v1/action-contract-activation-verification-receipt.schema.json","schema_version":"1","artifact_id":"gact-1","raw_sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","producer_signature_digest":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","conformance_digest":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","signer_key_id":"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd","issued_at":"2026-08-01T12:00:00Z","signature":{"alg":"ed25519","key_id":"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd","sig":"AA==","signed_digest":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"}}`)
	var receipt ActivationVerificationReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	if err := validateSchema(raw, ActivationReceiptSchemaID); err != nil {
		t.Fatalf("embedded activation receipt schema rejected its receipt: %v", err)
	}
	unknown := append([]byte(nil), raw[:len(raw)-1]...)
	unknown = append(unknown, []byte(`,"unexpected":true}`)...)
	if _, err := decodeActivationVerificationReceipt(unknown); err == nil {
		t.Fatal("receipt decoder accepted an unknown field")
	}
}

func TestActivationReceiptRollbackRestoresExistingSidecars(t *testing.T) {
	path := filepath.Join(t.TempDir(), "activation.json")
	original := []byte("original")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := snapshotFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	restoreFileSnapshots([]fileSnapshot{snapshot})
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(original) {
		t.Fatalf("existing sidecar was not restored: %q %v", got, err)
	}

	newPath := filepath.Join(t.TempDir(), "new.json")
	empty, err := snapshotFile(newPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newPath, []byte("transient"), 0o600); err != nil {
		t.Fatal(err)
	}
	restoreFileSnapshots([]fileSnapshot{empty})
	if _, err := os.Stat(newPath); !os.IsNotExist(err) {
		t.Fatalf("new sidecar was not removed on rollback: %v", err)
	}
}

func TestNativeStorePreservesBytesAndEnvelope(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "action-contract-interop", "v1", "expected", "package-to-release", "pac-0d9384785d3b213a.json"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := ParseProposal(raw)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if _, err = PersistProposal(root, p); err != nil {
		t.Fatal(err)
	}
	items, err := LoadStored(root)
	if err != nil || len(items) != 1 {
		t.Fatalf("load stored: %d %v", len(items), err)
	}
	if !bytes.Equal(items[0].Raw, raw) || items[0].Envelope.RawSHA256 != RawDigest(raw) || !items[0].Envelope.NonBinding {
		t.Fatalf("native bytes/envelope changed: %+v", items[0].Envelope)
	}
	if _, err = PersistProposal(root, p); err != nil {
		t.Fatalf("idempotent persist: %v", err)
	}
	p.Raw[0] = 'x'
	if _, err = PersistProposal(root, p); err == nil {
		t.Fatal("changed bytes accepted for existing artifact identity")
	}
}

func TestNativeStoreRejectsUnsafeAndTamperedArtifacts(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "action-contract-interop", "v1", "expected", "package-to-release", "pac-0d9384785d3b213a.json"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := ParseProposal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = persist(t.TempDir(), ProposalDir, "../escape", raw, NativeArtifactEnvelope{}); err == nil {
		t.Fatal("path traversal accepted")
	}
	root := t.TempDir()
	if _, err = PersistProposal(root, p); err != nil {
		t.Fatal(err)
	}
	payload := filepath.Join(root, NativeArtifactDir, ProposalDir, p.ArtifactID+".json")
	if err = os.WriteFile(payload, append([]byte("x"), raw...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = LoadStored(root); err == nil {
		t.Fatal("tampered payload accepted")
	}
	_ = os.Remove(payload)
	if _, err = PersistProposal(root, p); err != nil {
		t.Fatal(err)
	}
	envelope := filepath.Join(root, NativeArtifactDir, ProposalDir, p.ArtifactID+".envelope.json")
	envRaw, _ := os.ReadFile(envelope)
	envRaw = bytes.Replace(envRaw, []byte(`"contract_id": "pac-0d9384785d3b213a"`), []byte(`"contract_id": "pac-wrong"`), 1)
	if err = os.WriteFile(envelope, envRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = LoadStored(root); err == nil {
		t.Fatal("metadata-mismatched envelope accepted")
	}
	_ = os.Remove(envelope)
	if err = os.WriteFile(payload, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = LoadStored(root); err == nil {
		t.Fatal("orphan payload accepted")
	}
	if err = os.Remove(payload); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink("/tmp/missing", payload); err != nil {
		t.Fatal(err)
	}
	if _, err = LoadStored(root); err == nil {
		t.Fatal("symlinked payload accepted")
	}
}

func TestNativeActivationEnvelopeRoundTripPreservesConformance(t *testing.T) {
	proposalRaw, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "action-contract-interop", "v1", "expected", "package-to-release", "pac-0d9384785d3b213a.json"))
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := ParseProposal(proposalRaw)
	if err != nil {
		t.Fatal(err)
	}
	activation, publicKey := makeTestActivation(t, proposal, "enforce_floor", nil)
	root := t.TempDir()
	verification := ActivationVerification{Result: ValidationResult{Valid: true, SignatureVerified: true}, PublicKey: publicKey}
	conformance := &ConformanceResult{ReasonCodes: []string{"context_only_non_binding"}}
	if _, err = PersistActivation(root, activation, conformance, verification); err != nil {
		t.Fatal(err)
	}
	items, err := LoadStored(root)
	if err != nil || len(items) != 1 {
		t.Fatalf("activation round trip: %d %v", len(items), err)
	}
	if got := items[0].Envelope.ConformanceReasonCodes; len(got) != 1 || got[0] != "context_only_non_binding" {
		t.Fatalf("conformance metadata lost: %v", got)
	}
	envelopePath := filepath.Join(root, NativeArtifactDir, ActivationDir, activation.ArtifactID+".envelope.json")
	envelopeRaw, err := os.ReadFile(envelopePath)
	if err != nil {
		t.Fatal(err)
	}
	envelopeRaw = bytes.Replace(envelopeRaw, []byte("context_only_non_binding"), []byte("tampered"), 1)
	if err = os.WriteFile(envelopePath, envelopeRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = LoadStored(root); err == nil {
		t.Fatal("activation conformance metadata tamper accepted")
	}
	if _, err = PersistActivation(root, activation, conformance, verification); err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(root, NativeArtifactDir, ActivationDir, activation.ArtifactID+".verification.json")
	if err = os.Remove(receiptPath); err != nil {
		t.Fatal(err)
	}
	if _, err = LoadStored(root); err == nil {
		t.Fatal("activation without trusted verification receipt accepted")
	}
}

func TestNativeActivationPersistenceRollsBackOnReceiptFailure(t *testing.T) {
	proposalRaw, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "action-contract-interop", "v1", "expected", "package-to-release", "pac-0d9384785d3b213a.json"))
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := ParseProposal(proposalRaw)
	if err != nil {
		t.Fatal(err)
	}
	activation, publicKey := makeTestActivation(t, proposal, "enforce_floor", nil)
	root := t.TempDir()
	receiptPath := filepath.Join(root, NativeArtifactDir, ActivationDir, activation.ArtifactID+".verification.json")
	if err := os.MkdirAll(receiptPath, 0o700); err != nil {
		t.Fatal(err)
	}
	_, err = PersistActivation(root, activation, &ConformanceResult{}, ActivationVerification{Result: ValidationResult{Valid: true, SignatureVerified: true}, PublicKey: publicKey})
	if err == nil {
		t.Fatal("activation persisted despite verification receipt write failure")
	}
	for _, path := range []string{
		filepath.Join(root, NativeArtifactDir, ActivationDir, activation.ArtifactID+".json"),
		filepath.Join(root, NativeArtifactDir, ActivationDir, activation.ArtifactID+".envelope.json"),
	} {
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			t.Fatalf("partial activation persisted at %s: %v", path, statErr)
		}
	}
}
