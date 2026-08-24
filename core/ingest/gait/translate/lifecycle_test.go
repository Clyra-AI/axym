package translate

import (
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/ingest/gait/evidence"
)

func TestTranslateLifecycleProducesDistinctConformanceTestRecord(t *testing.T) {
	result, pack := authoritativeFixtureResult(t)
	first, err := TranslateLifecycle(result, pack)
	if err != nil {
		t.Fatal(err)
	}
	second, err := TranslateLifecycle(result, pack)
	if err != nil {
		t.Fatal(err)
	}
	if first.RecordType != "test_result" || first.RecordID != second.RecordID || LifecycleTranslationDigest(first) != LifecycleTranslationDigest(second) {
		t.Fatalf("translation not deterministic or distinct: first=%+v second=%+v", first, second)
	}
	if first.Event["evidence_set_id"] != result.EvidenceSet.EvidenceSetID || first.Metadata["evidence_kind"] != "gait_lifecycle" || first.Event["authoritative_success"] != true {
		t.Fatalf("lifecycle semantics missing: event=%+v metadata=%+v", first.Event, first.Metadata)
	}
	if first.Relationship == nil || len(first.Relationship.EntityRefs) < 3 {
		t.Fatalf("exact lifecycle references missing: %+v", first.Relationship)
	}
}

func TestTranslateLifecycleRejectsFixtureAndPackMismatch(t *testing.T) {
	result, pack := authoritativeFixtureResult(t)
	fixture := result
	fixture.Authoritative = false
	fixture.EvidenceSet.Authoritative = false
	fixture.EvidenceSet.FixtureOnly = true
	fixture.FixtureOnly = true
	if _, err := TranslateLifecycle(fixture, pack); err == nil || !strings.Contains(err.Error(), ReasonLifecycleAuthorityRequired) {
		t.Fatalf("fixture accepted: %v", err)
	}
	pack.SourceArtifactDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err := TranslateLifecycle(result, pack); err == nil || !strings.Contains(err.Error(), ReasonLifecyclePackMismatch) {
		t.Fatalf("tampered pack accepted: %v", err)
	}
}

func authoritativeFixtureResult(t *testing.T) (evidence.VerificationResult, evidence.LifecyclePack) {
	t.Helper()
	root := filepath.Join("..", "..", "..", "..", "testdata", "gait-action-contract-evidence", "v1")
	path := filepath.Join(root, "successful-execution-effect-containment", "lifecycle.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	pack, err := evidence.ParseLifecyclePack(raw)
	if err != nil {
		t.Fatal(err)
	}
	pack.SourceArtifactDigest = evidence.RawDigest(raw)
	keyRaw, err := os.ReadFile(filepath.Join(root, "fixture-signing-key.public.b64"))
	if err != nil {
		t.Fatal(err)
	}
	keyBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(keyRaw)))
	if err != nil {
		t.Fatal(err)
	}
	first := pack.Records[0]
	activation := evidence.Ref{}
	for _, record := range pack.Records {
		if record.ActivationRef != nil {
			activation = *record.ActivationRef
			break
		}
	}
	verified := evidence.VerifyLifecyclePack(pack, evidence.VerificationOptions{
		TrustedPublicKey: ed25519.PublicKey(keyBytes), EvaluationTime: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), AllowFixtureOnly: true,
		ExpectedContract: first.ContractRef, ExpectedFamily: first.ContractFamilyID, ExpectedRevision: first.Revision, ExpectedActivation: activation,
		ExpectedRuntimeDigest: "sha256:ffdb7187847ee43434cf0bc428d9defc9b407da4595be1bdfab4c16a47a801e1", ExpectedReadinessDigest: "sha256:5537a606ce771336b50c0f6f6ca978d8d310cb7e8d59eff47d7ac698264b4305",
		ExpectedPolicyDigest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", ExpectedTarget: "target:fixture", ExpectedEnvironment: "test",
		ExpectedProducerVersion: evidence.FixtureTag, ExpectedSourceCommit: evidence.FixtureCommit, ExpectedLifecycleDigest: pack.SourceArtifactDigest,
		ActivationNotBefore: time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC), ActivationNotAfter: time.Date(2027, 7, 19, 0, 0, 0, 0, time.UTC),
	})
	if !verified.Valid {
		t.Fatalf("fixture verification failed: %+v", verified)
	}
	verified.Authoritative = true
	verified.FixtureOnly = false
	verified.EvidenceSet.Authoritative = true
	verified.EvidenceSet.FixtureOnly = false
	verified.EvidenceSet.SourceCommit = evidence.FixtureCommit
	verified.EvidenceSet.SourceArtifactDigests = []string{pack.SourceArtifactDigest}
	verified.EvidenceSet.DerivedEvidenceDigests = []string{"sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}
	verified.ReasonCodes = nil
	verified.EvidenceSet.ReasonCodes = nil
	return verified, pack
}
