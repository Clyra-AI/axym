package evidence

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestReleasedFixtureScenarioMatrix(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "testdata", "gait-action-contract-evidence", "v1")
	keyRaw, err := os.ReadFile(filepath.Join(root, "fixture-signing-key.public.b64"))
	if err != nil {
		t.Fatal(err)
	}
	keyBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(keyRaw)))
	if err != nil || len(keyBytes) != ed25519.PublicKeySize {
		t.Fatalf("key: %v", err)
	}
	manifestRaw, err := os.ReadFile(filepath.Join(root, "fixture-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Scenarios []struct {
			ScenarioID     string `json:"scenario_id"`
			Path           string `json:"path"`
			ExpectedReason string `json:"expected_reason"`
			ExpectedValid  bool   `json:"expected_valid"`
			EvaluationTime string `json:"evaluation_time"`
		} `json:"scenarios"`
	}
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Scenarios) != 9 {
		t.Fatalf("fixture scenario count=%d", len(manifest.Scenarios))
	}
	for _, scenario := range manifest.Scenarios {
		t.Run(scenario.ScenarioID, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(scenario.Path)))
			if err != nil {
				t.Fatal(err)
			}
			pack, err := ParseLifecyclePack(raw)
			if err != nil {
				t.Fatal(err)
			}
			first := pack.Records[0]
			activation := first.ActivationRef
			if activation == nil {
				for _, record := range pack.Records {
					if record.ActivationRef != nil {
						activation = record.ActivationRef
						break
					}
				}
			}
			if activation == nil {
				t.Fatal("missing activation ref")
			}
			now := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
			if scenario.EvaluationTime != "" {
				now, err = time.Parse(time.RFC3339, scenario.EvaluationTime)
				if err != nil {
					t.Fatal(err)
				}
			}
			options := fixtureOptions(pack, ed25519.PublicKey(keyBytes), now, true)
			options.ExpectedContract, options.ExpectedFamily, options.ExpectedRevision, options.ExpectedActivation = first.ContractRef, first.ContractFamilyID, first.Revision, *activation
			result := VerifyLifecyclePack(pack, options)
			if result.Valid != scenario.ExpectedValid {
				for i, record := range pack.Records {
					if err := verifyLifecycleRecord(record, options.TrustedPublicKey); err != nil {
						t.Logf("record[%d] kind=%s verification=%v", i, record.Kind, err)
					}
				}
				t.Fatalf("valid=%v want=%v reasons=%v", result.Valid, scenario.ExpectedValid, result.ReasonCodes)
			}
			if !scenario.ExpectedValid && scenario.ExpectedReason != "" && !containsReason(result.ReasonCodes, axymReasonForFixture(scenario.ExpectedReason)) {
				t.Fatalf("reasons=%v missing semantic equivalent of %q", result.ReasonCodes, scenario.ExpectedReason)
			}
			if scenario.ExpectedValid && !result.FixtureOnly || scenario.ExpectedValid && result.Authoritative {
				t.Fatalf("fixture authority leak: %+v", result)
			}
		})
	}
}

func TestAxymSourceManifestPinsReleasedFixtureBytes(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "testdata", "gait-action-contract-evidence", "v1")
	manifestRaw, err := os.ReadFile(filepath.Join(root, "SOURCE-MANIFEST.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		SourceTag             string            `json:"source_tag"`
		SourceCommit          string            `json:"source_commit"`
		FixtureManifestSHA256 string            `json:"fixture_manifest_sha256"`
		PublicKeySHA256       string            `json:"public_key_sha256"`
		Files                 map[string]string `json:"files"`
	}
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SourceTag != FixtureTag || manifest.SourceCommit != FixtureCommit || len(manifest.Files) != 11 {
		t.Fatalf("source manifest drift: %+v", manifest)
	}
	fixtureManifest, err := os.ReadFile(filepath.Join(root, "fixture-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if RawDigest(fixtureManifest) != manifest.FixtureManifestSHA256 {
		t.Fatal("fixture manifest digest drift")
	}
	publicKey, err := os.ReadFile(filepath.Join(root, "fixture-signing-key.public.b64"))
	if err != nil {
		t.Fatal(err)
	}
	if RawDigest(publicKey) != manifest.PublicKeySHA256 {
		t.Fatal("public key digest drift")
	}
	for path, digest := range manifest.Files {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		if RawDigest(raw) != digest {
			t.Fatalf("fixture digest drift: %s", path)
		}
	}
}

func TestReleasedFixtureRejectsWrongKeyAndTamper(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "testdata", "gait-action-contract-evidence", "v1")
	raw, err := os.ReadFile(filepath.Join(root, "successful-execution-effect-containment", "lifecycle.json"))
	if err != nil {
		t.Fatal(err)
	}
	pack, err := ParseLifecyclePack(raw)
	if err != nil {
		t.Fatal(err)
	}
	wrong := make(ed25519.PublicKey, ed25519.PublicKeySize)
	wrongOptions := fixtureOptions(pack, wrong, time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), true)
	if result := VerifyLifecyclePack(pack, wrongOptions); result.Valid || !containsReason(result.ReasonCodes, ReasonSignatureInvalid) {
		t.Fatalf("wrong key result=%+v", result)
	}
	fixtureKey := keyForTest(t, root)
	productionOptions := fixtureOptions(pack, fixtureKey, time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), false)
	if result := VerifyLifecyclePack(pack, productionOptions); result.Valid || result.Authoritative || !containsReason(result.ReasonCodes, ReasonFixtureNonAuthoritative) {
		t.Fatalf("fixture key became production authority: %+v", result)
	}
	pack.Records[5].Execution.Outcome = "failed"
	if result := VerifyLifecyclePack(pack, fixtureOptions(pack, fixtureKey, time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), true)); result.Valid {
		t.Fatalf("tampered evidence accepted: %+v", result)
	}
}

func TestVerificationRequiresCallerOwnedBindingsAndLifecycleOrder(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "testdata", "gait-action-contract-evidence", "v1")
	pack := loadFixturePack(t, root, "successful-execution-effect-containment")
	key := keyForTest(t, root)
	now := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)

	missing := fixtureOptions(pack, key, now, true)
	missing.ExpectedRuntimeDigest = ""
	if result := VerifyLifecyclePack(pack, missing); result.Valid || !containsReason(result.ReasonCodes, ReasonLineageMissing) {
		t.Fatalf("missing caller-owned runtime binding accepted: %+v", result)
	}

	missingWindow := fixtureOptions(pack, key, now, true)
	missingWindow.ActivationNotAfter = time.Time{}
	if result := VerifyLifecyclePack(pack, missingWindow); result.Valid || !containsReason(result.ReasonCodes, ReasonReadinessInvalid) {
		t.Fatalf("missing activation validity window accepted: %+v", result)
	}

	unreleased := pack
	unreleased.SourceArtifactDigest = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	unreleasedOptions := fixtureOptions(unreleased, key, now, true)
	if result := VerifyLifecyclePack(unreleased, unreleasedOptions); result.Valid || !containsReason(result.ReasonCodes, ReasonSourceProvenanceInvalid) {
		t.Fatalf("unreleased fixture digest accepted: %+v", result)
	}

	wrongCommit := fixtureOptions(pack, key, now, true)
	wrongCommit.ExpectedSourceCommit = "ffffffffffffffffffffffffffffffffffffffff"
	if result := VerifyLifecyclePack(pack, wrongCommit); result.Valid || !containsReason(result.ReasonCodes, ReasonSourceProvenanceInvalid) {
		t.Fatalf("wrong fixture source commit accepted: %+v", result)
	}

	mismatched := fixtureOptions(pack, key, now, true)
	mismatched.ExpectedContract.ID = "pac-same-digest-different-id"
	if result := VerifyLifecyclePack(pack, mismatched); result.Valid || !containsReason(result.ReasonCodes, ReasonLineageMismatch) {
		t.Fatalf("same-digest/different-identity contract accepted: %+v", result)
	}

	withoutActivation := pack
	withoutActivation.Records = append([]LifecycleRecord(nil), pack.Records...)
	for i, record := range withoutActivation.Records {
		if record.Kind == "activated" {
			withoutActivation.Records = append(withoutActivation.Records[:i], withoutActivation.Records[i+1:]...)
			break
		}
	}
	if result := VerifyLifecyclePack(withoutActivation, fixtureOptions(pack, key, now, true)); result.Valid || !containsReason(result.ReasonCodes, ReasonEvidenceOrder) {
		t.Fatalf("execution without activation accepted: %+v", result)
	}
}

func TestPreconditionRequiresActivationRequest(t *testing.T) {
	if lifecyclePreconditionReady(true, false) {
		t.Fatal("precondition accepted before activation request")
	}
	if !lifecyclePreconditionReady(true, true) {
		t.Fatal("precondition rejected after proposal and activation request")
	}
}

func TestParseLifecyclePackRejectsUnknownAndDuplicateFields(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "testdata", "gait-action-contract-evidence", "v1")
	raw, err := os.ReadFile(filepath.Join(root, "successful-execution-effect-containment", "lifecycle.json"))
	if err != nil {
		t.Fatal(err)
	}
	unknown := append([]byte(`{"unexpected":true,`), raw[1:]...)
	if _, err := ParseLifecyclePack(unknown); err == nil || !strings.Contains(err.Error(), ReasonUnknownField) {
		t.Fatalf("unknown field accepted: %v", err)
	}
	duplicate := append([]byte(`{"records":[],`), raw[1:]...)
	if _, err := ParseLifecyclePack(duplicate); err == nil || !strings.Contains(err.Error(), "duplicate key") {
		t.Fatalf("duplicate field accepted: %v", err)
	}
}

func TestExactReferenceIdentityAndDeterministicEvidenceSet(t *testing.T) {
	base := Ref{Kind: "execution", ID: "gait-exec-1", Digest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", SchemaID: ExecutionSchemaID, SchemaVersion: EvidenceSchemaVersion, SourceProduct: GaitProducer}
	if !sameRef(base, base) {
		t.Fatal("identical reference did not match")
	}
	mutations := []Ref{base, base, base, base, base}
	mutations[0].ID = "gait-exec-2"
	mutations[1].SchemaID = EffectSchemaID
	mutations[2].SchemaVersion = "2"
	mutations[3].SourceProduct = "other"
	mutations[4].Kind = "effect_event"
	for _, mutation := range mutations {
		if sameRef(base, mutation) {
			t.Fatalf("partial reference identity matched: %+v", mutation)
		}
	}

	root := filepath.Join("..", "..", "..", "..", "testdata", "gait-action-contract-evidence", "v1")
	key := keyForTest(t, root)
	now := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	success := loadFixturePack(t, root, "successful-execution-effect-containment")
	partial := loadFixturePack(t, root, "partial-containment")
	first := VerifyLifecyclePack(success, fixtureOptions(success, key, now, true))
	repeated := VerifyLifecyclePack(success, fixtureOptions(success, key, now, true))
	other := VerifyLifecyclePack(partial, fixtureOptions(partial, key, now, true))
	if !first.Valid || !repeated.Valid || !other.Valid {
		t.Fatalf("released fixtures failed verification: first=%+v repeated=%+v other=%+v", first, repeated, other)
	}
	if first.EvidenceSet.EvidenceSetID != repeated.EvidenceSet.EvidenceSetID || !reflect.DeepEqual(first.EvidenceSet.SourceArtifactDigests, repeated.EvidenceSet.SourceArtifactDigests) || !reflect.DeepEqual(first.EvidenceSet.DerivedEvidenceDigests, repeated.EvidenceSet.DerivedEvidenceDigests) {
		t.Fatalf("evidence-set projection is not deterministic: first=%+v repeated=%+v", first.EvidenceSet, repeated.EvidenceSet)
	}
	if first.EvidenceSet.EvidenceSetID == other.EvidenceSet.EvidenceSetID {
		t.Fatalf("distinct lifecycle packs share evidence-set ID %q", first.EvidenceSet.EvidenceSetID)
	}
	if len(first.EvidenceSet.SourceArtifactDigests) == 0 || !sort.StringsAreSorted(first.EvidenceSet.SourceArtifactDigests) {
		t.Fatalf("source artifact digests are empty or unstable: %v", first.EvidenceSet.SourceArtifactDigests)
	}
	if len(first.EvidenceSet.DerivedEvidenceDigests) == 0 || !sort.StringsAreSorted(first.EvidenceSet.DerivedEvidenceDigests) {
		t.Fatalf("derived evidence digests are empty or unstable: %v", first.EvidenceSet.DerivedEvidenceDigests)
	}
	for _, digest := range first.EvidenceSet.SourceArtifactDigests {
		if !strings.HasPrefix(digest, "sha256:") || !validDigest(digest) {
			t.Fatalf("invalid source artifact digest %q", digest)
		}
	}
}

func loadFixturePack(t *testing.T, root, scenario string) LifecyclePack {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, scenario, "lifecycle.json"))
	if err != nil {
		t.Fatal(err)
	}
	pack, err := ParseLifecyclePack(raw)
	if err != nil {
		t.Fatal(err)
	}
	return pack
}

func keyForTest(t *testing.T, root string) ed25519.PublicKey {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "fixture-signing-key.public.b64"))
	if err != nil {
		t.Fatal(err)
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	return ed25519.PublicKey(key)
}

func fixtureOptions(pack LifecyclePack, key ed25519.PublicKey, now time.Time, allowFixture bool) VerificationOptions {
	first := pack.Records[0]
	activation := Ref{}
	for _, record := range pack.Records {
		if record.ActivationRef != nil {
			activation = *record.ActivationRef
			break
		}
	}
	return VerificationOptions{
		TrustedPublicKey: key, EvaluationTime: now, AllowFixtureOnly: allowFixture,
		ExpectedContract: first.ContractRef, ExpectedFamily: first.ContractFamilyID, ExpectedRevision: first.Revision, ExpectedActivation: activation,
		ExpectedRuntimeDigest: "sha256:ffdb7187847ee43434cf0bc428d9defc9b407da4595be1bdfab4c16a47a801e1", ExpectedReadinessDigest: "sha256:5537a606ce771336b50c0f6f6ca978d8d310cb7e8d59eff47d7ac698264b4305",
		ExpectedPolicyDigest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", ExpectedTarget: "target:fixture", ExpectedEnvironment: "test",
		ExpectedProducerVersion: FixtureTag, ExpectedSourceCommit: FixtureCommit, ExpectedLifecycleDigest: pack.SourceArtifactDigest,
		ActivationNotBefore: time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC), ActivationNotAfter: time.Date(2027, 7, 19, 0, 0, 0, 0, time.UTC),
	}
}

func containsReason(reasons []string, want string) bool {
	for _, reason := range reasons {
		if reason == want || strings.Contains(reason, want) {
			return true
		}
	}
	return false
}

func axymReasonForFixture(reason string) string {
	switch reason {
	case "conformance_replay":
		return ReasonReplay
	case "conformance_identifier_only":
		return ReasonCorrelationInvalid
	default:
		return reason
	}
}
