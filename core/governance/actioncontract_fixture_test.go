package governance

import (
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/ingest/actioncontract"
	"github.com/Clyra-AI/axym/core/ingest/gait/evidence"
	"github.com/Clyra-AI/axym/core/ingest/gait/translate"
)

func TestContractProjectionPreservesCompensationRequirementObject(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "action-contract-interop", "v1", "expected", "package-to-release", "pac-0d9384785d3b213a.json"))
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := actioncontract.ParseProposal(raw)
	if err != nil {
		t.Fatal(err)
	}
	contract, err := contractFromProposal(proposal)
	if err != nil {
		t.Fatal(err)
	}
	if len(contract.Compensation) != 1 || contract.Compensation[0]["required"] != true || contract.Compensation[0]["kind"] != "documented_recovery" {
		t.Fatalf("producer compensation requirement was omitted: %+v", contract.Compensation)
	}
}

func TestLifecycleFixturePreservesEachEffectTransitionRef(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "gait-action-contract-evidence", "v1")
	raw, err := os.ReadFile(filepath.Join(root, "compensation-required-started-completed", "lifecycle.json"))
	if err != nil {
		t.Fatal(err)
	}
	pack, err := evidence.ParseLifecyclePack(raw)
	if err != nil {
		t.Fatal(err)
	}
	keyRaw, err := os.ReadFile(filepath.Join(root, "fixture-signing-key.public.b64"))
	if err != nil {
		t.Fatal(err)
	}
	keyBytes, err := base64.StdEncoding.DecodeString(string(keyRaw))
	if err != nil {
		t.Fatal(err)
	}
	first := pack.Records[0]
	activation := *pack.Records[4].ActivationRef
	pack.SourceArtifactDigest = evidence.RawDigest(raw)
	result := evidence.VerifyLifecyclePack(pack, evidence.VerificationOptions{
		TrustedPublicKey: ed25519.PublicKey(keyBytes), EvaluationTime: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), AllowFixtureOnly: true,
		ExpectedContract: first.ContractRef, ExpectedFamily: first.ContractFamilyID, ExpectedRevision: first.Revision, ExpectedActivation: activation,
		ExpectedRuntimeDigest: "sha256:ffdb7187847ee43434cf0bc428d9defc9b407da4595be1bdfab4c16a47a801e1", ExpectedReadinessDigest: "sha256:5537a606ce771336b50c0f6f6ca978d8d310cb7e8d59eff47d7ac698264b4305",
		ExpectedPolicyDigest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", ExpectedTarget: "target:fixture", ExpectedEnvironment: "test",
		ExpectedProducerVersion: evidence.FixtureTag, ExpectedSourceCommit: evidence.FixtureCommit, ExpectedLifecycleDigest: pack.SourceArtifactDigest,
		ActivationNotBefore: time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC), ActivationNotAfter: time.Date(2027, 7, 19, 0, 0, 0, 0, time.UTC),
	})
	if !result.Valid {
		t.Fatalf("fixture verification failed: %+v", result)
	}
	result.Authoritative, result.FixtureOnly = true, false
	result.EvidenceSet.Authoritative, result.EvidenceSet.FixtureOnly = true, false
	record, err := translate.TranslateLifecycle(result, pack)
	if err != nil {
		t.Fatal(err)
	}
	contractRef := Ref{ID: first.ContractRef.ID, Kind: first.ContractRef.Kind, Digest: first.ContractRef.Digest, Source: "wrkr", SourceProduct: first.ContractRef.SourceProduct, SchemaID: first.ContractRef.SchemaID, SchemaVersion: first.ContractRef.SchemaVersion}
	evidenceItems := verifiedLifecycleEvidence(*record, contractRef)
	var effects []Evidence
	for _, item := range evidenceItems {
		if item.Kind == "effect" {
			effects = append(effects, item)
		}
	}
	if len(effects) != 2 || effects[0].Ref.ID == effects[1].Ref.ID || effects[0].Ref.Digest == effects[1].Ref.Digest {
		t.Fatalf("effect transition refs were collapsed: %+v", effects)
	}
}
