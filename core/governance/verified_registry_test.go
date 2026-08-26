package governance

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/ingest/actioncontract"
	"github.com/Clyra-AI/axym/core/store"
	"github.com/Clyra-AI/proof"
)

func TestVerifiedRegistryIsRequiredForSignedLifecycleProjection(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "action-contract-interop", "v1", "expected", "compensation", "pac-4b7f1402784256ce.json"))
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := actioncontract.ParseProposal(raw)
	if err != nil {
		t.Fatal(err)
	}
	register, _, err := RegisterAndPackets([]actioncontract.Proposal{proposal}, nil)
	if err != nil {
		t.Fatal(err)
	}
	contract := register.Contracts[0]
	key, err := proof.GenerateSigningKey()
	if err != nil {
		t.Fatal(err)
	}
	ref := map[string]any{"id": contract.ID, "kind": "action_contract", "digest": contract.CausalRef.Digest, "schema_id": contract.CausalRef.SchemaID, "schema_version": contract.CausalRef.SchemaVersion, "source_product": "wrkr"}
	record, err := proof.NewRecord(proof.RecordOpts{Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Source: "gait", SourceProduct: "gait", Type: "test_result", Event: map[string]any{"contract_ref": ref, "evidence_refs": []map[string]any{{"id": "e", "kind": "execution", "digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "schema_id": "https://gait.dev/schemas/v1/execution-evidence.schema.json", "schema_version": "1", "source_product": "gait"}}, "gait_execution": "succeeded"}, Metadata: map[string]any{"evidence_kind": "gait_lifecycle", "gait_evidence_set_id": "set-1", "gait_producer_version": "v1", "gait_source_commit": "0123456789012345678901234567890123456789", "gait_verification_state": "verified", "gait_authoritative": true, "gait_fixture_only": false, "gait_source_artifact_digest": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "gait_source_artifact_digests": []string{"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}, "gait_derived_evidence_digests": []string{"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err = store.SignLifecycleReceipt(record, key, "2026-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	record.Integrity.RecordHash = ""
	if _, err = proof.Sign(record, key); err != nil {
		t.Fatal(err)
	}
	if got, hashErr := proof.ComputeRecordHash(record); hashErr != nil || got != record.Integrity.RecordHash {
		t.Fatalf("signed record is not self-consistent: compute=%s stored=%s err=%v metadata=%#v", got, record.Integrity.RecordHash, hashErr, record.Metadata)
	}
	if _, _, err = RegisterAndPacketsVerifiedWithRegistry([]actioncontract.Proposal{proposal}, []proof.Record{*record}, proof.PublicKey{KeyID: key.KeyID, Public: key.Public}, t.TempDir()); err == nil {
		t.Fatal("signed caller-authored lifecycle promoted without registry")
	}
	root := t.TempDir()
	if err = RegisterVerifiedLifecycle(root, *record, key); err != nil {
		t.Fatal(err)
	}
	if _, _, err = RegisterAndPacketsVerifiedWithRegistry([]actioncontract.Proposal{proposal}, []proof.Record{*record}, proof.PublicKey{KeyID: key.KeyID, Public: key.Public}, root); err != nil {
		t.Fatal(err)
	}
	if err = VerifyLifecycleRecordsWithRegistry(root, []proof.Record{*record}, proof.PublicKey{KeyID: key.KeyID, Public: key.Public}); err != nil {
		t.Fatalf("standalone lifecycle registry verification failed: %v", err)
	}
	tampered := *record
	tampered.Event = map[string]any{"contract_ref": ref, "evidence_refs": []map[string]any{{"id": "e", "kind": "execution", "digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "schema_id": "https://gait.dev/schemas/v1/execution-evidence.schema.json", "schema_version": "1", "source_product": "gait"}}, "gait_execution": "failed"}
	if _, _, err = RegisterAndPacketsVerifiedWithRegistry([]actioncontract.Proposal{proposal}, []proof.Record{tampered}, proof.PublicKey{KeyID: key.KeyID, Public: key.Public}, root); err == nil {
		t.Fatal("tampered lifecycle receipt accepted")
	}
	wrong, _, _ := ed25519.GenerateKey(rand.Reader)
	if _, _, err = RegisterAndPacketsVerifiedWithRegistry([]actioncontract.Proposal{proposal}, []proof.Record{*record}, proof.PublicKey{Public: wrong}, root); err == nil {
		t.Fatal("wrong-key lifecycle accepted")
	}
}

func TestLifecyclePacketRetainsProposalRoot(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "action-contract-interop", "v1", "expected", "compensation", "pac-4b7f1402784256ce.json"))
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := actioncontract.ParseProposal(raw)
	if err != nil {
		t.Fatal(err)
	}
	register, packets, err := RegisterAndPackets([]actioncontract.Proposal{proposal}, nil)
	if err != nil {
		t.Fatal(err)
	}
	evidence := packets[register.Contracts[0].ID].Evidence
	if len(evidence) == 0 || evidence[0].Kind != "proposal" {
		t.Fatalf("proposal root was omitted from packet: %+v", evidence)
	}
}
