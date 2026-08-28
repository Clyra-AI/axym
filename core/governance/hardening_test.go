package governance

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/ingest/actioncontract"
	"github.com/Clyra-AI/proof"
)

func TestEvidenceDerivedCompletenessAndAxes(t *testing.T) {
	partial := []Event{{ID: "p", ContractRef: verifiedRef("c"), Kind: "proposed", OccurredAt: "2026-01-01T00:00:00Z", SourceDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, {ID: "a", ContractRef: verifiedRef("c"), Kind: "approved", OccurredAt: "2026-01-01T00:00:01Z", SourceDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}}
	got := DeriveCompletenessFromEvents(partial)
	if got.Status != Partial || len(got.ReasonCodes) == 0 {
		t.Fatalf("partial derivation: %+v", got)
	}
	unknown := []Event{{ID: "p", ContractRef: verifiedRef("c"), Kind: "proposed", OccurredAt: "2026-01-01T00:00:00Z", SourceDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, {ID: "a", ContractRef: verifiedRef("c"), Kind: "approved", OccurredAt: "2026-01-01T00:00:01Z", SourceDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}, {ID: "x", ContractRef: verifiedRef("c"), Kind: "contained", Status: "unresolved", OccurredAt: "2026-01-01T00:00:02Z", SourceDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}}
	if got := DeriveCompletenessFromEvents(unknown); got.Status != Gap {
		t.Fatalf("unresolved effect should be gap: %+v", got)
	}
	if got := DeriveCompletenessFromEvents(nil); got.Status != Partial {
		t.Fatalf("absent evidence should be partial: %+v", got)
	}
	if got := DeriveCompleteness("c", []Event{{ID: "foreign", ContractRef: verifiedRef("other"), Kind: "approved", OccurredAt: "2026-01-01T00:00:00Z", SourceDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}); got.Status != OutOfScope {
		t.Fatalf("foreign evidence should be out of scope: %+v", got)
	}
}

func TestReduceVerifiedRejectsMissingDigestEvidence(t *testing.T) {
	ref := verifiedRef("c")
	if _, err := ReduceVerified("c", []Event{{ID: "p", ContractRef: ref, Kind: "proposed", OccurredAt: "2026-01-01T00:00:00Z"}}); err == nil {
		t.Fatal("missing source digest accepted")
	}
	ref.Digest = ""
	if _, err := ReduceVerified("c", []Event{{ID: "p", ContractRef: ref, Kind: "proposed", OccurredAt: "2026-01-01T00:00:00Z", SourceDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}); err == nil {
		t.Fatal("missing relationship digest accepted")
	}
}

func TestFabricatedProofRecordTypeDoesNotPromoteRuntimeAxes(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "action-contract-interop", "v1", "expected", "compensation", "pac-4b7f1402784256ce.json"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := actioncontract.ParseProposal(raw)
	if err != nil {
		t.Fatal(err)
	}
	register, _, err := RegisterAndPackets([]actioncontract.Proposal{p}, nil)
	if err != nil {
		t.Fatal(err)
	}
	c := register.Contracts[0]
	record := proof.Record{RecordID: "fabricated", SourceProduct: "gait", RecordType: "execution", Metadata: map[string]any{"evidence_kind": "not_gait_lifecycle", "gait_authoritative": true, "gait_fixture_only": false, "gait_verification_state": "verified"}, Event: map[string]any{"contract_ref": map[string]any{"id": c.ID, "kind": "action_contract", "digest": c.CausalRef.Digest, "schema_id": c.CausalRef.SchemaID, "source_product": c.CausalRef.SourceProduct}}, Integrity: proof.Integrity{RecordHash: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}
	_, packets, err := RegisterAndPackets([]actioncontract.Proposal{p}, []proof.Record{record})
	if err != nil {
		t.Fatal(err)
	}
	if packets[c.ID].AxisStates["execution"] != "missing" {
		t.Fatalf("fabricated execution label promoted: %+v", packets[c.ID].AxisStates)
	}
}

func TestVerifiedLifecycleProjectsOnlyExactRefsAndSemanticGaps(t *testing.T) {
	contractRef := verifiedRef("c")
	activationRef := map[string]any{"id": "gact-1", "kind": "activated_action_contract", "digest": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "source": "gait", "source_product": "gait", "schema_id": "https://gait.dev/schemas/v1/activated-action-contract-artifact.schema.json", "schema_version": "1"}
	record := proof.Record{RecordID: "gait-lifecycle", Source: "gait", SourceProduct: "gait", Integrity: proof.Integrity{RecordHash: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}, Metadata: map[string]any{"evidence_kind": "gait_lifecycle", "gait_verification_state": "verified", "gait_authoritative": true, "gait_fixture_only": false, "gait_source_artifact_digest": "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", "gait_source_artifact_digests": []string{"sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"}, "gait_derived_evidence_digests": []string{"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"}}, Event: map[string]any{"contract_ref": contractRef, "activation_ref": activationRef, "evidence_refs": []map[string]any{{"id": "exec-1", "kind": "execution", "digest": "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "source": "gait", "source_product": "gait", "schema_id": "https://gait.dev/schemas/v1/execution-evidence.schema.json", "schema_version": "1"}, {"id": "effect-1", "kind": "effect_event", "digest": "sha256:1111111111111111111111111111111111111111111111111111111111111111", "source": "gait", "source_product": "gait", "schema_id": "https://gait.dev/schemas/v1/effect-event.schema.json", "schema_version": "1"}, {"id": "contain-1", "kind": "containment", "digest": "sha256:2222222222222222222222222222222222222222222222222222222222222222", "source": "gait", "source_product": "gait", "schema_id": "https://gait.dev/schemas/v1/containment-evidence.schema.json", "schema_version": "1"}}, "gait_execution": "succeeded", "gait_effect": "validated", "containment_status": "unresolved", "gait_lifecycle_events": []map[string]any{{"id": "p", "kind": "proposal_ingested", "occurred_at": "2026-01-01T00:00:00Z"}, {"id": "a", "kind": "activated", "occurred_at": "2026-01-01T00:00:01Z"}, {"id": "x", "kind": "execution_succeeded", "occurred_at": "2026-01-01T00:00:02Z"}, {"id": "f", "kind": "effect_validated", "occurred_at": "2026-01-01T00:00:03Z"}, {"id": "c", "kind": "containment_unresolved", "occurred_at": "2026-01-01T00:00:04Z"}}}}
	compRefs := []map[string]any{{"id": "comp-required", "kind": "compensation", "digest": "sha256:3333333333333333333333333333333333333333333333333333333333333333", "source": "gait", "source_product": "gait", "schema_id": "https://gait.dev/schemas/v1/action-contract/compensation-evidence.schema.json", "schema_version": "1"}, {"id": "comp-started", "kind": "compensation", "digest": "sha256:4444444444444444444444444444444444444444444444444444444444444444", "source": "gait", "source_product": "gait", "schema_id": "https://gait.dev/schemas/v1/action-contract/compensation-evidence.schema.json", "schema_version": "1"}, {"id": "comp-completed", "kind": "compensation", "digest": "sha256:5555555555555555555555555555555555555555555555555555555555555555", "source": "gait", "source_product": "gait", "schema_id": "https://gait.dev/schemas/v1/action-contract/compensation-evidence.schema.json", "schema_version": "1"}}
	record.Event["evidence_refs"] = append(record.Event["evidence_refs"].([]map[string]any), compRefs...)
	record.Event["compensation_status"] = "completed"
	record.Event["gait_lifecycle_events"] = append(record.Event["gait_lifecycle_events"].([]map[string]any), []map[string]any{{"id": "cr", "kind": "compensation_required", "occurred_at": "2026-01-01T00:00:05Z", "evidence_ref": compRefs[0]}, {"id": "cs", "kind": "compensation_started", "occurred_at": "2026-01-01T00:00:06Z", "evidence_ref": compRefs[1]}, {"id": "cc", "kind": "compensation_completed", "occurred_at": "2026-01-01T00:00:07Z", "evidence_ref": compRefs[2]}}...)
	evidence := verifiedLifecycleEvidence(record, contractRef)
	seen := map[string]Evidence{}
	for _, item := range evidence {
		seen[item.Kind] = item
	}
	for _, missing := range []string{"approval", "credential", "delegation", "confirmation"} {
		if _, ok := seen[missing]; ok {
			t.Fatalf("unbound %s axis was promoted", missing)
		}
	}
	for _, absent := range []string{"proof", "freshness", "resource_lifecycle"} {
		if _, ok := seen[absent]; ok {
			t.Fatalf("unbound %s axis was synthesized", absent)
		}
	}
	if seen["execution"].Attributes["state"] != "succeeded" || seen["effect"].Attributes["state"] != "validated" || seen["containment"].Attributes["state"] != "gap" {
		t.Fatalf("semantic axes not projected correctly: %+v", seen)
	}
	if seen["execution"].Ref.ID != "exec-1" || seen["execution"].Ref.Kind != "execution" || seen["effect"].Ref.ID != "effect-1" || seen["effect"].Ref.Kind != "effect_event" {
		t.Fatalf("source refs were rewritten: execution=%+v effect=%+v", seen["execution"].Ref, seen["effect"].Ref)
	}
	if seen["compensation"].Ref.ID != "comp-completed" {
		t.Fatalf("compensation source ref was rewritten or collapsed: %+v", seen["compensation"].Ref)
	}
	if seen["execution"].OccurredAt != "2026-01-01T00:00:02Z" || seen["effect"].OccurredAt != "2026-01-01T00:00:03Z" || seen["containment"].OccurredAt != "2026-01-01T00:00:04Z" {
		t.Fatalf("source lifecycle timestamps were flattened: execution=%s effect=%s containment=%s", seen["execution"].OccurredAt, seen["effect"].OccurredAt, seen["containment"].OccurredAt)
	}
	if seen["enforcement"].Provenance.Kind != "gait_lifecycle_source" || seen["enforcement"].Provenance.SourceProduct != "gait" {
		t.Fatalf("raw lifecycle provenance missing: %+v", seen["enforcement"].Provenance)
	}
	if seen["enforcement"].Ref.ID != "gact-1" || seen["enforcement"].Attributes["event_id"] != record.RecordID {
		t.Fatalf("activation source ref or run-specific event identity was not preserved: %+v", seen["enforcement"])
	}
	if seen["enforcement"].Attributes["source_digest"] != record.Integrity.RecordHash {
		t.Fatalf("restart event did not receive a run-specific signed source digest: %+v", seen["enforcement"])
	}
	withoutActivation := record
	withoutActivation.Event = map[string]any{"contract_ref": contractRef, "evidence_refs": record.Event["evidence_refs"], "gait_execution": "succeeded"}
	if got := verifiedLifecycleEvidence(withoutActivation, contractRef); len(got) != 0 {
		t.Fatalf("missing activation_ref promoted lifecycle: %+v", got)
	}
}

func TestRepeatedActivationEventsRetainSharedRefAndUseUniqueSignedDigests(t *testing.T) {
	activation := verifiedRef("gact-1")
	activation.Kind = "activated_action_contract"
	contract := verifiedRef("contract")
	packet := Packet{Evidence: []Evidence{
		{Kind: "enforcement", Ref: activation, ContractRef: contract, Attributes: map[string]string{"event_id": "run-1", "source_digest": "sha256:1111111111111111111111111111111111111111111111111111111111111111"}},
		{Kind: "enforcement", Ref: activation, ContractRef: contract, Attributes: map[string]string{"event_id": "run-2", "source_digest": "sha256:2222222222222222222222222222222222222222222222222222222222222222"}},
	}}
	events := EventsFromPacket(packet)
	if len(events) != 2 || events[0].ID != "run-1" || events[1].ID != "run-2" || events[0].SourceDigest == events[1].SourceDigest {
		t.Fatalf("repeated activation events lost run-specific identity/digests: %+v", events)
	}
	if events[0].ContractRef.ID != contract.ID || events[0].Kind != "activated" || events[1].Kind != "activated" {
		t.Fatalf("repeated activation events changed lifecycle mapping: %+v", events)
	}
	if events[0].SourceDigest == activation.Digest || events[1].SourceDigest == activation.Digest {
		t.Fatalf("repeated activation events fell back to shared activation digest: %+v", events)
	}
}

func TestTelemetryIdentifierMatchNeverBecomesAuthoritative(t *testing.T) {
	span := TraceSpan{TraceID: "0123456789abcdef0123456789abcdef", SpanID: "0123456789abcdef", StartTime: "2026-01-01T00:00:00Z", EndTime: "2026-01-01T00:00:01Z", Source: "otel", Attributes: map[string]string{"contract.id": "c"}}
	span.Digest, _ = Digest(span)
	raw, _ := json.Marshal(map[string]any{"spans": []TraceSpan{span}})
	result, err := IngestTelemetry(raw, time.Date(2026, 1, 1, 0, 0, 2, 0, time.UTC), time.Hour, "c")
	if err != nil {
		t.Fatal(err)
	}
	if result.CorrelationState == "authoritative" || result.CorrelationState != "reported_match" {
		t.Fatalf("telemetry spoof became authoritative: %+v", result)
	}
}

func TestGateParentOrderingAndMapImmutability(t *testing.T) {
	keyRaw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "governance", "v1", "gait-gate", "fixture-signing-key.public.b64"))
	if err != nil {
		t.Fatal(err)
	}
	pubBytes, err := base64.StdEncoding.DecodeString(string(keyRaw))
	if err != nil {
		t.Fatal(err)
	}
	rootRaw, _ := os.ReadFile(filepath.Join("..", "..", "testdata", "governance", "v1", "gait-gate", "delegation-root.json"))
	childRaw, _ := os.ReadFile(filepath.Join("..", "..", "testdata", "governance", "v1", "gait-gate", "delegation-child-tightened.json"))
	var parent map[string]any
	if err := json.Unmarshal(rootRaw, &parent); err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(parent)
	if _, err = VerifyGateArtifact(childRaw, ed25519.PublicKey(pubBytes), time.Date(2026, 8, 1, 12, 30, 0, 0, time.UTC), parent); err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(parent)
	if string(before) != string(after) {
		t.Fatal("gate verification mutated parent map")
	}
	if result := VerifyGateChain([][]byte{childRaw, rootRaw}, ed25519.PublicKey(pubBytes), time.Date(2026, 8, 1, 12, 30, 0, 0, time.UTC)); !result.AuthorityLineage {
		t.Fatalf("child-first chain rejected: %+v", result)
	}
	forged, _ := os.ReadFile(filepath.Join("..", "..", "testdata", "governance", "v1", "gait-gate", "invalid", "wrong-parent-digest.json"))
	result := VerifyGateChain([][]byte{rootRaw, forged}, ed25519.PublicKey(pubBytes), time.Date(2026, 8, 1, 12, 30, 0, 0, time.UTC))
	if result.AuthorityLineage || len(result.Gaps) == 0 {
		t.Fatalf("forged parent digest accepted: %+v", result)
	}
	duplicate := VerifyGateChain([][]byte{rootRaw, rootRaw}, ed25519.PublicKey(pubBytes), time.Date(2026, 8, 1, 12, 30, 0, 0, time.UTC))
	if duplicate.AuthorityLineage || !containsGateReason(duplicate.Gaps, ReasonGateDuplicate) {
		t.Fatalf("duplicate gate token accepted: %+v", duplicate)
	}
	if err := gateSubset(map[string]any{"scope": []any{"write"}}, map[string]any{"scope": []any{"admin"}}); err == nil || err.Error() != ReasonGateAuthority {
		t.Fatalf("delegated scope expansion accepted: %v", err)
	}
	depthRoot := map[string]any{"max_descendant_depth": float64(2), "depth": float64(0)}
	depthChild := map[string]any{"max_descendant_depth": float64(1), "depth": float64(1)}
	if err := gateSubset(depthRoot, depthChild); err != nil {
		t.Fatalf("depth-budget-preserving child rejected: %v", err)
	}
	depthGrandchild := map[string]any{"max_descendant_depth": float64(0), "depth": float64(2)}
	if err := gateSubset(depthChild, depthGrandchild); err != nil {
		t.Fatalf("three-hop depth-budget-preserving descendant rejected: %v", err)
	}
	repeatedBudget := map[string]any{"max_descendant_depth": float64(2), "depth": float64(1)}
	if err := gateSubset(depthRoot, repeatedBudget); err == nil || err.Error() != ReasonGateAuthority {
		t.Fatalf("delegated child repeated depth budget: %v", err)
	}
	beyondBudget := map[string]any{"max_descendant_depth": float64(0), "depth": float64(3)}
	if err := gateSubset(depthRoot, beyondBudget); err == nil || err.Error() != ReasonGateAuthority {
		t.Fatalf("delegated child exceeded depth budget: %v", err)
	}
	bindingsRoot := map[string]any{
		"contract_digest":         "contract",
		"policy_digest":           "policy",
		"origin_authority_digest": "origin",
	}
	bindingsChild := map[string]any{
		"contract_digest":         "contract",
		"policy_digest":           "policy",
		"origin_authority_digest": "origin",
	}
	if err := gateSubset(bindingsRoot, bindingsChild); err != nil {
		t.Fatalf("matching delegated bindings rejected: %v", err)
	}
	for _, key := range []string{"contract_digest", "policy_digest", "origin_authority_digest"} {
		child := cloneGateMap(bindingsChild)
		delete(child, key)
		if err := gateSubset(bindingsRoot, child); err == nil || err.Error() != ReasonGateParent {
			t.Fatalf("delegated child omitted %s binding: %v", key, err)
		}
	}
	limitsRoot := map[string]any{"max_operations": float64(4), "max_targets": float64(2)}
	limitsChild := map[string]any{"max_operations": float64(3), "max_targets": float64(1)}
	if err := gateSubset(limitsRoot, limitsChild); err != nil {
		t.Fatalf("bounded delegated limits rejected: %v", err)
	}
	for _, key := range []string{"max_operations", "max_targets"} {
		child := cloneGateMap(limitsChild)
		delete(child, key)
		if err := gateSubset(limitsRoot, child); err == nil || err.Error() != ReasonGateAuthority {
			t.Fatalf("delegated child omitted %s bound: %v", key, err)
		}
	}
	var malformedParent map[string]any
	if err := json.Unmarshal(childRaw, &malformedParent); err != nil {
		t.Fatal(err)
	}
	delete(malformedParent, "parent_token_id")
	malformedRaw, err := json.Marshal(malformedParent)
	if err != nil {
		t.Fatal(err)
	}
	malformedResult := VerifyGateChain([][]byte{malformedRaw}, ed25519.PublicKey(pubBytes), time.Date(2026, 8, 1, 12, 30, 0, 0, time.UTC))
	if malformedResult.AuthorityLineage || !containsGateReason(malformedResult.Gaps, ReasonGateParent) {
		t.Fatalf("dangling parent digest accepted as root: %+v", malformedResult)
	}
	malformedParent["parent_token_id"] = float64(7)
	malformedRaw, err = json.Marshal(malformedParent)
	if err != nil {
		t.Fatal(err)
	}
	malformedResult = VerifyGateChain([][]byte{malformedRaw}, ed25519.PublicKey(pubBytes), time.Date(2026, 8, 1, 12, 30, 0, 0, time.UTC))
	if malformedResult.AuthorityLineage || !containsGateReason(malformedResult.Gaps, ReasonGateParent) {
		t.Fatalf("mistyped parent reference accepted as root: %+v", malformedResult)
	}
}

func TestGateRejectsIncompleteAuthorityShape(t *testing.T) {
	base := map[string]any{
		"schema_id":          "gait.gate.delegation_token",
		"schema_version":     "1.0.0",
		"producer_version":   "v1.5.0",
		"token_id":           "root",
		"delegator_identity": "root@example",
		"delegate_identity":  "child@example",
		"created_at":         "2026-08-01T12:00:00Z",
		"expires_at":         "2026-08-01T14:00:00Z",
		"scope":              []any{"write"},
	}
	if err := validateGateShape(base); err != nil {
		t.Fatalf("complete gate shape rejected: %v", err)
	}
	unsupported := cloneGateMap(base)
	unsupported["schema_id"] = "gait.gate.unknown_token"
	if err := validateGateShape(unsupported); err == nil || err.Error() != ReasonMalformed {
		t.Fatalf("unsupported gate schema accepted: %v", err)
	}
	badVersion := cloneGateMap(base)
	badVersion["schema_version"] = "9.9.9"
	if err := validateGateShape(badVersion); err == nil || err.Error() != ReasonMalformed {
		t.Fatalf("unsupported gate schema version accepted: %v", err)
	}
	approval := cloneGateMap(base)
	approval["schema_id"] = "gait.gate.approval_token"
	approval["approver_identity"] = "owner@example"
	delete(approval, "delegator_identity")
	delete(approval, "delegate_identity")
	for _, key := range []string{"reason_code", "intent_digest", "policy_digest"} {
		candidate := cloneGateMap(approval)
		delete(candidate, key)
		if err := validateGateShape(candidate); err == nil || err.Error() != ReasonMalformed {
			t.Fatalf("approval gate omitted %s field: %v", key, err)
		}
	}
	for _, key := range []string{"schema_id", "schema_version", "delegator_identity", "delegate_identity", "scope"} {
		candidate := cloneGateMap(base)
		delete(candidate, key)
		if err := validateGateShape(candidate); err == nil || err.Error() != ReasonMalformed {
			t.Fatalf("missing %s gate field accepted: %v", key, err)
		}
	}
	for _, identities := range [][2]string{{"", "child@example"}, {"root@example", ""}, {"same@example", "same@example"}} {
		candidate := cloneGateMap(base)
		candidate["delegator_identity"], candidate["delegate_identity"] = identities[0], identities[1]
		if err := validateGateShape(candidate); err == nil || err.Error() != ReasonMalformed {
			t.Fatalf("invalid identity pair accepted: %v", err)
		}
	}
}

func containsGateReason(gaps []GateGap, reason string) bool {
	for _, gap := range gaps {
		if gap.Code == reason {
			return true
		}
	}
	return false
}

func verifiedRef(id string) Ref {
	return Ref{ID: id, Kind: "action_contract", Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Source: "wrkr", SourceProduct: "wrkr", SchemaID: RegisterSchemaID, SchemaVersion: SchemaVersion}
}

func TestPacketSignatureTamperAndWrongKey(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	contractRef := Ref{ID: "c", Kind: "action_contract", Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Source: "wrkr", SourceProduct: "wrkr", SchemaID: RegisterSchemaID, SchemaVersion: SchemaVersion}
	evidenceRef := Ref{ID: "e", Kind: "approval", Digest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Source: "gait", SourceProduct: "gait", SchemaID: RegisterSchemaID, SchemaVersion: SchemaVersion}
	p, err := BuildPacket(Contract{ID: "c", FamilyID: "f", Revision: 1, Action: "a", Target: "t", Environment: "e", Provenance: contractRef}, []Evidence{{Kind: "approval", Ref: evidenceRef, ContractRef: contractRef, Provenance: contractRef, OccurredAt: "2026-01-01T00:00:00Z"}})
	if err != nil {
		t.Fatal(err)
	}
	p, err = SignPacket(p, priv)
	if err != nil {
		t.Fatal(err)
	}
	if err = VerifySignedPacket(p, pub); err != nil {
		t.Fatal(err)
	}
	wrong, _, _ := ed25519.GenerateKey(rand.Reader)
	if err = VerifySignedPacket(p, wrong); err == nil {
		t.Fatal("wrong key accepted")
	}
	p.ReasonCodes = append(p.ReasonCodes, "tampered")
	if err = VerifySignedPacket(p, pub); err == nil {
		t.Fatal("tampered packet accepted")
	}
}

func TestCustomerRedactionPreservesJoinsAndRemovesSecrets(t *testing.T) {
	ref := Ref{ID: "p", Kind: "proposal", Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Source: "wrkr", SourceProduct: "wrkr", SchemaID: RegisterSchemaID, SchemaVersion: SchemaVersion}
	r := RedactRegister(Register{SchemaID: RegisterSchemaID, SchemaVersion: SchemaVersion, Contracts: []Contract{{ID: "c", FamilyID: "f", Revision: 1, Action: "a", Target: "t", Environment: "e", Authorization: []map[string]any{{"requirement_id": "keep", "required_constraint": "raw-secret"}}, Provenance: ref}}})
	raw, _ := json.Marshal(r)
	if bytes.Contains(raw, []byte("raw-secret")) || !bytes.Contains(raw, []byte("keep")) {
		t.Fatalf("redaction leaked or removed join: %s", raw)
	}
}

func TestMultiLevelGateFailuresAreHighSeverityGaps(t *testing.T) {
	keyRaw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "governance", "v1", "gait-gate", "fixture-signing-key.public.b64"))
	if err != nil {
		t.Fatal(err)
	}
	pubRaw, err := base64.StdEncoding.DecodeString(string(keyRaw))
	if err != nil {
		t.Fatal(err)
	}
	rootRaw, _ := os.ReadFile(filepath.Join("..", "..", "testdata", "governance", "v1", "gait-gate", "delegation-root.json"))
	childRaw, _ := os.ReadFile(filepath.Join("..", "..", "testdata", "governance", "v1", "gait-gate", "invalid", "action-expansion.json"))
	result := VerifyGateChain([][]byte{rootRaw, childRaw}, ed25519.PublicKey(pubRaw), time.Date(2026, 8, 1, 12, 30, 0, 0, time.UTC))
	if result.AuthorityLineage || len(result.Gaps) == 0 || result.Gaps[0].Severity != "high" {
		t.Fatalf("authority expansion not surfaced: %+v", result)
	}
}
