package actioncontract

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestWrkrFixtureManifestAndCanonicalDigests(t *testing.T) {
	rawManifest, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "action-contract-interop", "v1", "expected", "fixture-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		FixtureVersion string `json:"fixture_version"`
		Producer       struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"producer"`
		Schemas struct {
			Artifact string `json:"artifact"`
			Contract string `json:"contract"`
		} `json:"schemas"`
		Scenarios []struct {
			ScenarioID             string `json:"scenario_id"`
			ArtifactPath           string `json:"artifact_path"`
			ArtifactSHA256         string `json:"artifact_sha256"`
			ArtifactID             string `json:"artifact_id"`
			CanonicalContentDigest string `json:"canonical_content_digest"`
			ContractID             string `json:"contract_id"`
			ContractFamilyID       string `json:"contract_family_id"`
			Revision               int    `json:"revision"`
		} `json:"scenarios"`
	}
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.FixtureVersion != "1" || manifest.Producer.Name != "wrkr" || manifest.Producer.Version != "v1.14.0" || manifest.Schemas.Artifact != "1" || manifest.Schemas.Contract != "3" || len(manifest.Scenarios) != 9 {
		t.Fatalf("unexpected fixture provenance: %+v", manifest)
	}
	root := filepath.Join("..", "..", "..", "testdata", "action-contract-interop", "v1", "expected")
	for _, scenario := range manifest.Scenarios {
		path := filepath.Join(root, scenario.ScenarioID, filepath.Base(scenario.ArtifactPath))
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", scenario.ArtifactPath, err)
		}
		sum := sha256.Sum256(raw)
		if got := "sha256:" + hex.EncodeToString(sum[:]); got != scenario.ArtifactSHA256 {
			t.Fatalf("raw digest mismatch for %s: got %s want %s", scenario.ScenarioID, got, scenario.ArtifactSHA256)
		}
		proposal, err := ParseProposal(raw)
		if err != nil {
			t.Fatalf("parse %s: %v", scenario.ScenarioID, err)
		}
		if proposal.ArtifactID != scenario.ArtifactID || proposal.ContractID != scenario.ContractID || proposal.ContractFamilyID != scenario.ContractFamilyID || proposal.Revision != scenario.Revision || proposal.CanonicalContentDigest != scenario.CanonicalContentDigest {
			t.Fatalf("identity mismatch for %s: %+v", scenario.ScenarioID, proposal)
		}
	}
}

func TestWrkrProposalTamperFailsClosed(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "action-contract-interop", "v1", "expected", "customer-data-to-egress", "pac-6dcee5a6d9a65e8c.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := ParseProposal(raw)
	if err != nil {
		t.Fatal(err)
	}
	raw[len(raw)-2] ^= 1
	tampered, err := ParseProposal(raw)
	if err == nil {
		tamperedResult := ValidateProposal(tampered, ValidationOptions{})
		if tamperedResult.Valid {
			t.Fatal("tampered proposal validated")
		}
	}
	if proposal.RawSHA256 == rawDigest(raw) {
		t.Fatal("tampered bytes retained original digest")
	}
}

func TestAxymPublicAndEmbeddedSchemasMatch(t *testing.T) {
	for _, name := range []string{"proposed-action-contract-artifact.schema.json", "proposed-action-contract-v3.schema.json", "activated-action-contract-artifact.schema.json", "consumer-receipt.schema.json"} {
		public, err := os.ReadFile(filepath.Join("..", "..", "..", "schemas", "v1", "action_contract", name))
		if err != nil {
			t.Fatal(err)
		}
		embedded, err := os.ReadFile(filepath.Join("schemaassets", name))
		if err != nil {
			t.Fatal(err)
		}
		if string(public) != string(embedded) {
			t.Fatalf("public and embedded schema drifted: %s", name)
		}
	}
}

func TestAxymReceiptSchemaAcceptsDeterministicConsumerReceipt(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "action-contract-interop", "v1", "expected", "customer-data-to-egress", "pac-6dcee5a6d9a65e8c.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := ParseProposal(raw)
	if err != nil {
		t.Fatal(err)
	}
	receipt := Receipt{
		Consumer:           ConsumerName,
		Version:            ConsumerVersion,
		ScenarioID:         "customer-data-to-egress",
		ArtifactSHA256:     proposal.RawSHA256,
		Status:             StatusPass,
		SelfAttestation:    false,
		ProposalArtifactID: proposal.ArtifactID,
		ContractID:         proposal.ContractID,
		ContractFamilyID:   proposal.ContractFamilyID,
		Revision:           proposal.Revision,
		ResolutionKey:      proposal.ResolutionKey,
		CorrelationRefs:    SortedStrings(append(append(append([]string{}, proposal.SourceScanRefs...), proposal.CompositionRefs...), proposal.CreationEvidence...)),
		SchemaVersions:     map[string]string{"artifact": proposal.SchemaVersion, "contract": proposal.Producer.ContractSchemaVersion},
		SemanticResult:     SemanticResult{ProposalValid: true, Classification: ClassIncomplete, ExecutionClaim: false, EffectClaim: false},
	}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateSchema(encoded, ReceiptSchemaID); err != nil {
		t.Fatalf("receipt does not satisfy Axym-owned schema: %v", err)
	}
}

func TestActivationBindingAndConformanceAreDeterministic(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "action-contract-interop", "v1", "expected", "customer-data-to-egress", "pac-6dcee5a6d9a65e8c.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := ParseProposal(raw)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	data := map[string]any{
		"schema_id": ActivationSchemaID, "schema_version": ActivationSchemaVersion,
		"artifact_id": "", "contract_id": proposal.ContractID, "contract_family_id": proposal.ContractFamilyID, "revision": proposal.Revision,
		"producer":      map[string]any{"name": "gait", "artifact_schema_version": "1", "contract_schema_version": "1"},
		"proposal":      map[string]any{"artifact_id": proposal.ArtifactID, "canonical_content_digest": proposal.CanonicalContentDigest, "contract_id": proposal.ContractID, "contract_family_id": proposal.ContractFamilyID, "revision": proposal.Revision, "schema_id": ProposalSchemaID, "schema_version": "1", "contract_schema_version": "3"},
		"policy_digest": "sha256:" + hex.EncodeToString(make([]byte, 32)), "activating_principal": "principal:axym", "authority_refs": []any{"authority:gait"}, "target": "prod", "environment": "production", "activation_mode": "context_only", "validity": map[string]any{"not_before": "2026-01-01T00:00:00Z"}, "explicit_exceptions": []any{}, "report_only": false, "development_signing": false,
		"signature": map[string]any{"alg": "ed25519", "key_id": "", "sig": "", "signed_digest": ""},
	}
	activatedData, err := signActivationForTest(data, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	activationRaw, err := json.Marshal(activatedData)
	if err != nil {
		t.Fatal(err)
	}
	activation, err := ParseActivation(activationRaw)
	if err != nil {
		t.Fatal(err)
	}
	selection := testSelection(proposal)
	validation := ValidateActivation(activation, ValidationOptions{Now: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), Proposal: &proposal, Selection: selection, PublicKey: publicKey})
	if !validation.Valid {
		t.Fatalf("activation validation failed: %+v", validation)
	}
	contextual := CompareProposalActivation(proposal, activation, ValidateProposal(proposal, ValidationOptions{}), validation)
	if contextual.Classification != ClassContextual || !contextual.Valid || !contextual.NonBinding {
		t.Fatalf("context-only classification mismatch: %+v", contextual)
	}
	activation2, err := ParseActivation(activationRaw)
	if err != nil {
		t.Fatal(err)
	}
	if got := CompareProposalActivation(proposal, activation2, ValidateProposal(proposal, ValidationOptions{}), ValidateActivation(activation2, ValidationOptions{Now: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), Proposal: &proposal, Selection: selection, PublicKey: publicKey})); got.Classification != contextual.Classification || got.Valid != contextual.Valid {
		t.Fatalf("non-deterministic conformance: got %+v want %+v", got, contextual)
	}
}

func TestActivationTamperFailsSignatureVerification(t *testing.T) {
	proposalPath := filepath.Join("..", "..", "..", "testdata", "action-contract-interop", "v1", "expected", "customer-data-to-egress", "pac-6dcee5a6d9a65e8c.json")
	raw, err := os.ReadFile(proposalPath)
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := ParseProposal(raw)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	data := map[string]any{"schema_id": ActivationSchemaID, "schema_version": ActivationSchemaVersion, "artifact_id": "", "contract_id": proposal.ContractID, "contract_family_id": proposal.ContractFamilyID, "revision": proposal.Revision, "producer": map[string]any{"name": "gait", "artifact_schema_version": "1", "contract_schema_version": "1"}, "proposal": map[string]any{"artifact_id": proposal.ArtifactID, "canonical_content_digest": proposal.CanonicalContentDigest, "contract_id": proposal.ContractID, "contract_family_id": proposal.ContractFamilyID, "revision": proposal.Revision, "schema_id": ProposalSchemaID, "schema_version": "1", "contract_schema_version": "3"}, "policy_digest": "sha256:" + hex.EncodeToString(make([]byte, 32)), "activating_principal": "principal:axym", "authority_refs": []any{"authority:gait"}, "target": "prod", "environment": "production", "activation_mode": "enforce_floor", "validity": map[string]any{"not_before": "2026-01-01T00:00:00Z"}, "explicit_exceptions": []any{}, "report_only": false, "development_signing": false, "signature": map[string]any{"alg": "ed25519", "key_id": "", "sig": "", "signed_digest": ""}}
	data, err = signActivationForTest(data, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	data["target"] = "tampered"
	mutated, _ := json.Marshal(data)
	activation, err := ParseActivation(mutated)
	if err != nil {
		t.Fatal(err)
	}
	validation := ValidateActivation(activation, ValidationOptions{Now: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), Proposal: &proposal, Selection: testSelection(proposal), PublicKey: publicKey})
	if validation.Valid || !containsReason(validation.ReasonCodes, ReasonSignature) {
		t.Fatalf("tampered activation unexpectedly verified: %+v", validation)
	}
}

func TestActivationRevisionModeAndExceptionClassification(t *testing.T) {
	proposalPath := filepath.Join("..", "..", "..", "testdata", "action-contract-interop", "v1", "expected", "customer-data-to-egress", "pac-6dcee5a6d9a65e8c.json")
	raw, err := os.ReadFile(proposalPath)
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := ParseProposal(raw)
	if err != nil {
		t.Fatal(err)
	}
	activation, publicKey := makeTestActivation(t, proposal, "enforce_floor", nil)
	options := ValidationOptions{Now: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), Proposal: &proposal, Selection: testSelection(proposal), PublicKey: publicKey}
	if result := ValidateActivation(activation, options); !result.Valid {
		t.Fatalf("enforce_floor activation should verify: %+v", result)
	}

	wrongRevision := activation
	wrongRevision.Revision++
	wrongRevisionResult := ValidateActivation(wrongRevision, ValidationOptions{Now: options.Now, Proposal: &proposal, Selection: options.Selection, PublicKey: publicKey, ExpectedRevision: proposal.Revision})
	if wrongRevisionResult.Valid || !containsReason(wrongRevisionResult.ReasonCodes, ReasonBinding) {
		t.Fatalf("revision drift must fail closed: %+v", wrongRevisionResult)
	}

	wrongMode := activation
	wrongMode.ActivationMode = "not-a-mode"
	wrongModeResult := ValidateActivation(wrongMode, options)
	if wrongModeResult.Valid || !containsReason(wrongModeResult.ReasonCodes, ReasonIdentity) {
		t.Fatalf("unsupported activation mode must fail closed: %+v", wrongModeResult)
	}

	wrongID := activation
	wrongID.ArtifactID = "gact-0000000000000000"
	wrongIDResult := ValidateActivation(wrongID, options)
	if wrongIDResult.Valid || !containsReason(wrongIDResult.ReasonCodes, ReasonIdentity) {
		t.Fatalf("activation identity drift must fail closed: %+v", wrongIDResult)
	}

	activation.ExplicitExceptions = []string{"authority_requirements"}
	conformance := CompareProposalActivation(proposal, activation, ValidateProposal(proposal, ValidationOptions{}), ValidateActivation(activation, options))
	if conformance.Classification != ClassExplicitlyExcepted || !conformance.Valid {
		t.Fatalf("explicit exception classification mismatch: %+v", conformance)
	}
}

func TestActivationRequiresSelectionAndExplicitEvaluationTime(t *testing.T) {
	proposalPath := filepath.Join("..", "..", "..", "testdata", "action-contract-interop", "v1", "expected", "customer-data-to-egress", "pac-6dcee5a6d9a65e8c.json")
	raw, err := os.ReadFile(proposalPath)
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := ParseProposal(raw)
	if err != nil {
		t.Fatal(err)
	}
	activation, publicKey := makeTestActivation(t, proposal, "context_only", nil)
	withoutSelection := ValidateActivation(activation, ValidationOptions{Now: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), Proposal: &proposal, PublicKey: publicKey})
	if withoutSelection.Valid || !containsReason(withoutSelection.ReasonCodes, ReasonSelectionRequired) {
		t.Fatalf("activation without current selection must fail closed: %+v", withoutSelection)
	}
	withoutTime := ValidateActivation(activation, ValidationOptions{Proposal: &proposal, Selection: testSelection(proposal), PublicKey: publicKey})
	if withoutTime.Valid || withoutTime.Status != StatusUnverifiable || !containsReason(withoutTime.ReasonCodes, ReasonEvaluationTime) {
		t.Fatalf("activation without explicit evaluation time must remain unverifiable: %+v", withoutTime)
	}
	contextual := CompareProposalActivation(proposal, activation, ValidateProposal(proposal, ValidationOptions{}), withoutTime)
	if contextual.Classification != ClassContextual || contextual.Valid || !contextual.NonBinding {
		t.Fatalf("unverifiable context-only activation must remain non-binding: %+v", contextual)
	}
	stale := testSelection(proposal)
	stale.Current = false
	staleResult := ValidateActivation(activation, ValidationOptions{Now: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), Proposal: &proposal, Selection: stale, PublicKey: publicKey})
	if staleResult.Valid || !containsReason(staleResult.ReasonCodes, ReasonSelectionNotCurrent) {
		t.Fatalf("stale selection must fail closed: %+v", staleResult)
	}
}

func TestDevelopmentSigningIsNeverVerified(t *testing.T) {
	proposalPath := filepath.Join("..", "..", "..", "testdata", "action-contract-interop", "v1", "expected", "customer-data-to-egress", "pac-6dcee5a6d9a65e8c.json")
	raw, err := os.ReadFile(proposalPath)
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := ParseProposal(raw)
	if err != nil {
		t.Fatal(err)
	}
	activation, publicKey := makeTestActivation(t, proposal, "context_only", nil)
	activation.DevelopmentSigning = true
	options := ValidationOptions{Now: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), Proposal: &proposal, Selection: testSelection(proposal), PublicKey: publicKey}
	defaultResult := ValidateActivation(activation, options)
	if defaultResult.Valid || !containsReason(defaultResult.ReasonCodes, ReasonDevelopmentSigning) {
		t.Fatalf("development signing must be rejected by default: %+v", defaultResult)
	}
	options.AllowDevelopmentSigning = true
	testResult := ValidateActivation(activation, options)
	if testResult.Valid || testResult.Status != StatusUnverifiable || !containsReason(testResult.ReasonCodes, ReasonDevelopmentUnverified) {
		t.Fatalf("explicit development allowance must remain unverifiable: %+v", testResult)
	}
}

func TestDeepProjectionClassifiesExactTightenedWeakenedAndExactExceptions(t *testing.T) {
	proposalPath := filepath.Join("..", "..", "..", "testdata", "action-contract-interop", "v1", "expected", "customer-data-to-egress", "pac-6dcee5a6d9a65e8c.json")
	raw, err := os.ReadFile(proposalPath)
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := ParseProposal(raw)
	if err != nil {
		t.Fatal(err)
	}
	activation, publicKey := makeTestActivation(t, proposal, "enforce_floor", nil)
	options := ValidationOptions{Now: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), Proposal: &proposal, Selection: testSelection(proposal), PublicKey: publicKey}
	validation := ValidateActivation(activation, options)
	values := proposalControlValues(proposal)
	compensation := boolField(proposal.Contract, "compensation_required")
	projection := &ActivationProjection{
		AuthorityRefs: strings.Split(values["authority"], "|"), Preconditions: strings.Split(values["preconditions"], "|"),
		ConfirmationRequirement: values["confirmation"], ApprovalRequirement: values["approval"], CredentialMode: values["credential_mode"],
		DelegationDepth: intValue(values["delegation_depth"]), ExpectedOutcomeClass: values["expected_outcome"], ForbiddenEffects: strings.Split(values["forbidden_effects"], "|"),
		CompensationRequired: &compensation, ValidityNotAfter: values["validity"],
	}
	exact := CompareProposalActivationWithProjection(proposal, activation, ValidateProposal(proposal, ValidationOptions{}), validation, projection)
	if exact.Classification != ClassExact || !exact.Valid {
		t.Fatalf("equal control projection must be exact: %+v", exact)
	}
	projection.AuthorityRefs = append(projection.AuthorityRefs, "additional:authority")
	tightened := CompareProposalActivationWithProjection(proposal, activation, ValidateProposal(proposal, ValidationOptions{}), validation, projection)
	if tightened.Classification != ClassTightened || !tightened.Valid {
		t.Fatalf("actual stricter authority projection must be tightened: %+v", tightened)
	}
	projection.DelegationDepth++
	weakened := CompareProposalActivationWithProjection(proposal, activation, ValidateProposal(proposal, ValidationOptions{}), validation, projection)
	if weakened.Classification != ClassWeakened || weakened.Valid {
		t.Fatalf("weaker delegation projection must fail closed: %+v", weakened)
	}
	projection.DelegationDepth = intValue(values["delegation_depth"])
	projection.AuthorityRefs = nil
	activation.ExplicitExceptions = []string{"authority"}
	excepted := CompareProposalActivationWithProjection(proposal, activation, ValidateProposal(proposal, ValidationOptions{}), validation, projection)
	if excepted.Classification != ClassExplicitlyExcepted || !excepted.Valid {
		t.Fatalf("exact authority exception must classify explicitly_excepted: %+v", excepted)
	}
	activation.ExplicitExceptions = []string{"author"}
	unknown := CompareProposalActivationWithProjection(proposal, activation, ValidateProposal(proposal, ValidationOptions{}), validation, projection)
	if unknown.Classification != ClassMismatched || unknown.Valid {
		t.Fatalf("near-match exception must not authorize omission: %+v", unknown)
	}
}

func TestSelectionManifestAndStableReadSafety(t *testing.T) {
	proposalPath := filepath.Join("..", "..", "..", "testdata", "action-contract-interop", "v1", "expected", "customer-data-to-egress", "pac-6dcee5a6d9a65e8c.json")
	raw, err := os.ReadFile(proposalPath)
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := ParseProposal(raw)
	if err != nil {
		t.Fatal(err)
	}
	manifest := []byte(`{"fixture_version":"1","producer":{"name":"wrkr","version":"v1.14.0"},"schemas":{"artifact":"1","contract":"3"},"scenarios":[{"scenario_id":"customer-data-to-egress","artifact_path":"customer-data-to-egress/pac-6dcee5a6d9a65e8c.json","artifact_sha256":"` + rawDigest(raw) + `","artifact_id":"` + proposal.ArtifactID + `","canonical_content_digest":"` + proposal.CanonicalContentDigest + `","contract_id":"` + proposal.ContractID + `","contract_family_id":"` + proposal.ContractFamilyID + `","revision":` + fmt.Sprint(proposal.Revision) + `,"current":true}]}`)
	selection, err := ParseSelectionManifest(manifest, proposal)
	if err != nil || !selection.Current || selection.Revision != proposal.Revision {
		t.Fatalf("current selection manifest should validate: %v %+v", err, selection)
	}
	if _, err := ParseSelectionManifest(bytes.Replace(manifest, []byte(`"current":true`), []byte(`"current":false`), 1), proposal); err == nil || !containsReason(err.(*ValidationError).Reasons, ReasonSelectionNotCurrent) {
		t.Fatalf("stale selection manifest must fail: %v", err)
	}
	if runtime.GOOS == "windows" {
		t.Skip("symlink safety test requires portable symlink support")
	}
	root := t.TempDir()
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	regular := filepath.Join(root, "proposal.json")
	if err := os.WriteFile(regular, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link.json")
	if err := os.Symlink(regular, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadProposal(link); err == nil || !containsReason(err.(*ValidationError).Reasons, ReasonMalformed) {
		t.Fatalf("final symlink must be rejected: %v", err)
	}
	ancestor := filepath.Join(root, "ancestor", "proposal.json")
	if err := os.Symlink(root, filepath.Join(root, "ancestor")); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadProposal(ancestor); err == nil || !containsReason(err.(*ValidationError).Reasons, ReasonMalformed) {
		t.Fatalf("symlinked ancestor must be rejected: %v", err)
	}
	overflow := filepath.Join(root, "overflow.json")
	file, err := os.Create(overflow)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(maxArtifactBytes + 1); err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	if _, err := ReadProposal(overflow); err == nil || !containsReason(err.(*ValidationError).Reasons, ReasonMalformed) {
		t.Fatalf("oversized artifact must be rejected: %v", err)
	}
}

func TestStableReasonCodesHideFilesystemDetails(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, err = ReadProposal(filepath.Join(root, "missing.json"))
	if err == nil {
		t.Fatal("missing artifact unexpectedly read")
	}
	codes := StableReasonCodes(err)
	if len(codes) != 1 || codes[0] != ReasonInputUnreadable {
		t.Fatalf("unstable filesystem error leaked or was misclassified: err=%v codes=%v", err, codes)
	}
}

func makeTestActivation(t *testing.T, proposal Proposal, mode string, exceptions []string) (Activation, ed25519.PublicKey) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	encodedExceptions := make([]any, 0, len(exceptions))
	for _, exception := range exceptions {
		encodedExceptions = append(encodedExceptions, exception)
	}
	data := map[string]any{
		"schema_id": ActivationSchemaID, "schema_version": ActivationSchemaVersion,
		"artifact_id": "", "contract_id": proposal.ContractID, "contract_family_id": proposal.ContractFamilyID, "revision": proposal.Revision,
		"producer":      map[string]any{"name": "gait", "artifact_schema_version": "1", "contract_schema_version": "1"},
		"proposal":      map[string]any{"artifact_id": proposal.ArtifactID, "canonical_content_digest": proposal.CanonicalContentDigest, "contract_id": proposal.ContractID, "contract_family_id": proposal.ContractFamilyID, "revision": proposal.Revision, "schema_id": ProposalSchemaID, "schema_version": "1", "contract_schema_version": "3"},
		"policy_digest": "sha256:" + hex.EncodeToString(make([]byte, 32)), "activating_principal": "principal:axym", "authority_refs": []any{"authority:gait"}, "target": "prod", "environment": "production", "activation_mode": mode, "validity": map[string]any{"not_before": "2026-01-01T00:00:00Z"}, "explicit_exceptions": encodedExceptions, "report_only": false, "development_signing": false,
		"signature": map[string]any{"alg": "ed25519", "key_id": "", "sig": "", "signed_digest": ""},
	}
	encoded, err := signActivationForTest(data, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(encoded)
	if err != nil {
		t.Fatal(err)
	}
	activation, err := ParseActivation(raw)
	if err != nil {
		t.Fatal(err)
	}
	return activation, publicKey
}

func testSelection(proposal Proposal) *SelectionEvidence {
	return &SelectionEvidence{ArtifactID: proposal.ArtifactID, ArtifactSHA256: rawDigest(proposal.Raw), CanonicalContentDigest: proposal.CanonicalContentDigest, ContractID: proposal.ContractID, ContractFamilyID: proposal.ContractFamilyID, Revision: proposal.Revision, Current: true}
}

func containsReason(reasons []string, wanted string) bool {
	for _, reason := range reasons {
		if reason == wanted {
			return true
		}
	}
	return false
}
