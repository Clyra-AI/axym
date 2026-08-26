package governance

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/Clyra-AI/axym/core/ingest/actioncontract"
	"github.com/Clyra-AI/axym/core/store"
	"github.com/Clyra-AI/proof"
)

const FixedProposalTimestamp = "2000-01-01T00:00:00Z"

// RegisterAndPackets projects verified producer-native proposals into the
// governed Axym register. It intentionally accepts proposals separately from
// records: a proposal is report-only evidence and never an execution grant.
func RegisterAndPackets(proposals []actioncontract.Proposal, records []proof.Record) (Register, map[string]Packet, error) {
	// Legacy direct callers cannot establish trusted producer provenance;
	// intentionally keep this projection proposal-only and partial.
	return registerAndPackets(proposals, nil)
}

// RegisterAndPacketsVerified is the production path. It verifies every
// signed Proof record before any Gait lifecycle aggregate can promote axes.
func RegisterAndPacketsVerified(proposals []actioncontract.Proposal, records []proof.Record, publicKey proof.PublicKey) (Register, map[string]Packet, error) {
	for i := range records {
		if err := proof.Verify(&records[i], publicKey); err != nil {
			return Register{}, nil, fmt.Errorf("verified record %d: %w", i, err)
		}
		if lifecycleRecord(records[i]) {
			if _, err := store.VerifyLifecycleReceipt(&records[i], publicKey.Public); err != nil {
				return Register{}, nil, fmt.Errorf("verified record %d lacks an ingest receipt: %w", i, err)
			}
		}
	}
	return registerAndPackets(proposals, records)
}

func RegisterAndPacketsVerifiedWithRegistry(proposals []actioncontract.Proposal, records []proof.Record, publicKey proof.PublicKey, registryRoot string) (Register, map[string]Packet, error) {
	for i := range records {
		if err := proof.Verify(&records[i], publicKey); err != nil {
			return Register{}, nil, fmt.Errorf("verified record %d: %w", i, err)
		}
		if lifecycleRecord(records[i]) {
			if _, err := store.VerifyLifecycleReceipt(&records[i], publicKey.Public); err != nil {
				return Register{}, nil, fmt.Errorf("verified record %d lacks an ingest receipt: %w", i, err)
			}
			if _, ok := lifecycleRegistryEntry(records[i]); !ok || !VerifyRegisteredLifecycle(registryRoot, records[i], publicKey) {
				return Register{}, nil, fmt.Errorf("lifecycle record %s lacks trusted Gait verification registry entry", records[i].RecordID)
			}
		}
	}
	return registerAndPackets(proposals, records)
}

func registerAndPackets(proposals []actioncontract.Proposal, records []proof.Record) (Register, map[string]Packet, error) {
	register := Register{SchemaID: RegisterSchemaID, SchemaVersion: SchemaVersion}
	packets := map[string]Packet{}
	for _, proposal := range proposals {
		contract, err := contractFromProposal(proposal)
		if err != nil {
			return Register{}, nil, err
		}
		register.Contracts = append(register.Contracts, contract)
		evidence := evidenceForContract(contract, proposal, records)
		packet, err := BuildPacket(contract, evidence)
		if err != nil {
			return Register{}, nil, err
		}
		packets[contract.ID] = packet
	}
	normalized, err := NormalizeRegister(register)
	if err != nil {
		return Register{}, nil, err
	}
	register = normalized
	source, err := Digest(register.Contracts)
	if err != nil {
		return Register{}, nil, err
	}
	register.SourceDigest = source
	return register, packets, nil
}

func contractFromProposal(p actioncontract.Proposal) (Contract, error) {
	if p.ArtifactID == "" || p.ContractID == "" || p.ContractFamilyID == "" || p.Revision < 1 {
		return Contract{}, fmt.Errorf("proposal identity is incomplete")
	}
	ref := Ref{ID: p.ArtifactID, Kind: "proposal", Digest: p.RawSHA256, Source: "wrkr", SourceProduct: p.Producer.Name, SchemaID: p.SchemaID, SchemaVersion: p.SchemaVersion}
	if !validRef(ref) {
		return Contract{}, fmt.Errorf("proposal provenance is incomplete")
	}
	target := constraintValue(p.Contract, "target_identity")
	if target == "" {
		target = constraintValue(p.Contract, "target_class")
	}
	if target == "" {
		target = "unknown"
	}
	environment := constraintValue(p.Contract, "environment")
	if environment == "" {
		environment = "unknown"
	}
	action := stringFieldAC(p.Contract, "expected_outcome_class")
	if action == "" {
		action = "proposed_action"
	}
	c := Contract{ID: p.ContractID, FamilyID: p.ContractFamilyID, Revision: p.Revision, Action: action, Target: target, Environment: environment, PolicyDigest: stringFieldAC(p.Contract, "policy_digest"), Owner: stringFieldAC(p.Contract, "owner"), Provenance: ref}
	c.CausalRef = Ref{ID: p.ContractID, Kind: "action_contract", Digest: p.CanonicalContentDigest, Source: "wrkr", SourceProduct: p.Producer.Name, SchemaID: "https://wrkr.dev/schemas/v1/proposed-action-contract-v3.schema.json", SchemaVersion: p.Producer.ContractSchemaVersion}
	// Preserve each governed axis as its own array. Missing and empty remain
	// distinguishable to downstream completeness evaluators.
	c.Authorization = objectArrayAC(p.Contract, "authority_requirements")
	c.Preconditions = objectArrayAC(p.Contract, "preconditions")
	if value, ok := p.Contract["confirmation_requirement"].(map[string]any); ok {
		c.Confirmation = []map[string]any{value}
	}
	if value, ok := p.Contract["approval_requirement"].(map[string]any); ok {
		c.Approval = []map[string]any{value}
	}
	c.Credential = objectArrayAC(p.Contract, "credential_requirements")
	c.Effect = objectArrayAC(p.Contract, "effects")
	c.Compensation = objectOrArrayAC(p.Contract, "compensation_requirement")
	if len(c.Compensation) == 0 {
		// Accept the earlier plural spelling for older producer artifacts, but
		// prefer Wrkr v3's mandatory compensation_requirement object.
		c.Compensation = objectArrayAC(p.Contract, "compensation")
	}
	c.Outcome = objectArrayAC(p.Contract, "outcomes")
	return c, nil
}

func evidenceForContract(c Contract, p actioncontract.Proposal, records []proof.Record) []Evidence {
	proposalRef := c.Provenance
	contractRef := c.CausalRef
	if !validRef(contractRef) {
		contractRef = Ref{ID: c.ID, Kind: "action_contract", Digest: proposalRef.Digest, Source: proposalRef.Source, SourceProduct: proposalRef.SourceProduct, SchemaID: RegisterSchemaID, SchemaVersion: SchemaVersion}
	}
	// The proposal is the root lifecycle event. Keep it even when verified
	// Gait evidence exists so replay always starts from a legal state-machine
	// transition rather than an activation/runtime event.
	out := []Evidence{{Kind: "proposal", Ref: proposalRef, OccurredAt: FixedProposalTimestamp, ContractRef: contractRef, Provenance: proposalRef}}
	for _, record := range records {
		if !recordBelongs(record, c) {
			continue
		}
		out = append(out, verifiedLifecycleEvidence(record, contractRef)...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ref.ID < out[j].Ref.ID })
	return out
}

func verifiedLifecycleEvidence(record proof.Record, contractRef Ref) []Evidence {
	if !lifecycleRecord(record) || record.Event == nil || record.Integrity.RecordHash == "" {
		return nil
	}
	if stringFieldAC(record.Metadata, "evidence_kind") != "gait_lifecycle" || stringFieldAC(record.Metadata, "gait_verification_state") != "verified" || !boolFieldAC(record.Metadata, "gait_authoritative") || boolFieldAC(record.Metadata, "gait_fixture_only") {
		return nil
	}
	sourceDigest := stringFieldAC(record.Metadata, "gait_source_artifact_digest")
	sourceDigests := stringSliceAC(record.Metadata, "gait_source_artifact_digests")
	derivedDigests := stringSliceAC(record.Metadata, "gait_derived_evidence_digests")
	if !validDigestAC(sourceDigest) || !allValidDigestsAC(sourceDigests) || !allValidDigestsAC(derivedDigests) || !nonEmptyArrayAC(record.Event["evidence_refs"]) {
		return nil
	}
	occurred := record.Timestamp.UTC().Format(time.RFC3339Nano)
	base := Ref{ID: record.RecordID, Kind: "gait_lifecycle", Digest: record.Integrity.RecordHash, Source: record.Source, SourceProduct: "gait", SchemaID: "https://github.com/Clyra-AI/proof/schemas/v1/proof-record-v1.schema.json", SchemaVersion: "1.0"}
	if !validDigestAC(base.Digest) {
		return nil
	}
	refs := lifecycleRefsFromValue(record.Event["evidence_refs"])
	refsByKind := map[string]Ref{}
	validEvidenceRefCount := 0
	for _, ref := range refs {
		if ref.ID != "" && validDigestAC(ref.Digest) && ref.SchemaID != "" && ref.SchemaVersion != "" && ref.SourceProduct != "" {
			refsByKind[strings.ToLower(ref.Kind)] = ref
			validEvidenceRefCount++
		}
	}
	activation := lifecycleRefFromValue(record.Event["activation_ref"])
	contract := lifecycleRefFromValue(record.Event["contract_ref"])
	if !validRef(activation) || strings.ToLower(activation.Kind) != "activated_action_contract" || !validRef(contract) || strings.ToLower(contract.Kind) != "action_contract" || validEvidenceRefCount == 0 {
		return nil
	}
	if contract.ID != contractRef.ID || contract.Digest != contractRef.Digest || contract.SchemaID != contractRef.SchemaID || contract.SchemaVersion != contractRef.SchemaVersion || contract.SourceProduct != contractRef.SourceProduct {
		return nil
	}
	if validRef(activation) {
		refsByKind[strings.ToLower(activation.Kind)] = activation
		refsByKind["enforcement"] = activation
	}
	if validRef(contract) {
		refsByKind[strings.ToLower(contract.Kind)] = contract
	}
	values := []string{}
	for _, kind := range []string{"authority", "readiness", "preconditions", "confirmation", "approval", "credential", "delegation", "enforcement", "resource_lifecycle", "proof", "freshness", "correlation"} {
		if _, ok := refsByKind[kind]; ok {
			values = append(values, kind)
		}
	}
	if _, ok := refsByKind["action_contract"]; ok {
		values = append(values, "authority", "readiness", "correlation")
	}
	if _, ok := refsByKind["activated_action_contract"]; ok {
		values = append(values, "enforcement")
	}
	if _, ok := refsByKind["execution"]; ok && stringFieldAC(record.Event, "gait_execution") != "" {
		values = append(values, "execution")
	}
	if _, ok := refsByKind["effect_event"]; ok && stringFieldAC(record.Event, "gait_effect") != "" {
		values = append(values, "effect")
	}
	if _, ok := refsByKind["containment"]; ok && (stringFieldAC(record.Event, "containment_status") != "" || stringFieldAC(record.Event, "gait_containment_status") != "") {
		values = append(values, "containment")
	}
	if _, ok := refsByKind["compensation"]; ok && (stringFieldAC(record.Event, "compensation_status") != "" || stringFieldAC(record.Metadata, "gait_compensation_status") != "") {
		values = append(values, "compensation")
	}
	seen := map[string]bool{}
	out := make([]Evidence, 0, len(values))
	for _, kind := range values {
		if seen[kind] {
			continue
		}
		seen[kind] = true
		ref := base
		if source, ok := refsByKind[kind]; ok {
			ref = source
		} else if sourceKind, ok := map[string]string{"effect": "effect_event"}[kind]; ok {
			if source, exists := refsByKind[sourceKind]; exists {
				ref = source
			}
		}
		attributes := map[string]string{"state": lifecycleState(record, kind)}
		transitions := lifecycleTransitions(record, kind)
		if len(transitions) > 0 {
			for _, transition := range transitions {
				transitionRef := ref
				if validRef(transition.Ref) {
					transitionRef = transition.Ref
				}
				transitionAttributes := map[string]string{
					"state":    lifecycleTransitionState(kind, transition.Kind),
					"event_id": transition.ID,
				}
				if transition.SourceDigest != "" {
					transitionAttributes["source_digest"] = transition.SourceDigest
				}
				if lifecycleTransitionTerminal(kind, transition.Kind) {
					transitionAttributes["terminal"] = "true"
				}
				out = append(out, Evidence{Kind: kind, Ref: transitionRef, OccurredAt: transition.OccurredAt, ContractRef: contractRef, Attributes: transitionAttributes, Provenance: lifecycleProvenance(record, refs, sourceDigest)})
			}
			continue
		}
		if kind == "enforcement" {
			// The producer activation reference is immutable and may be shared
			// by repeated runs. Keep it as the source ref, but use the signed
			// lifecycle record ID as the event identity so each restart remains
			// replayable without duplicate reducer IDs.
			attributes["event_id"] = record.RecordID
			attributes["source_digest"] = record.Integrity.RecordHash
		}
		out = append(out, Evidence{Kind: kind, Ref: ref, OccurredAt: lifecycleEvidenceOccurredAt(record, kind, occurred), ContractRef: contractRef, Attributes: attributes, Provenance: lifecycleProvenance(record, refs, sourceDigest)})
	}
	return out
}

func lifecycleRecord(record proof.Record) bool {
	value, _ := record.Metadata["evidence_kind"].(string)
	return record.SourceProduct == "gait" && value == "gait_lifecycle"
}

func lifecycleState(record proof.Record, kind string) string {
	value := ""
	switch kind {
	case "execution":
		value = stringFieldAC(record.Event, "gait_execution")
		if strings.EqualFold(value, "blocked") {
			return "blocked"
		}
	case "effect":
		value = stringFieldAC(record.Event, "gait_effect")
	case "containment":
		value = stringFieldAC(record.Event, "containment_status")
		if value == "" {
			value = stringFieldAC(record.Event, "gait_containment_status")
		}
		if value == "partial" || value == "requested" {
			return value
		}
	case "compensation":
		value = stringFieldAC(record.Event, "compensation_status")
		if value == "" {
			value = stringFieldAC(record.Metadata, "gait_compensation_status")
		}
		value = strings.ToLower(value)
		switch value {
		case "required", "not_required", "started", "requested", "completed", "succeeded", "failed", "rejected", "unresolved", "unknown":
			return value
		}
	}
	value = strings.ToLower(value)
	if value == "failed" || value == "rejected" || (value == "unresolved" && kind == "containment") {
		return "gap"
	}
	if value == "unknown" || value == "unresolved" || value == "started" || value == "requested" {
		return "unknown"
	}
	return "present"
}

// lifecycleEvidenceOccurredAt preserves producer event order in the Axym
// projection. The aggregate timestamp is only a compatibility fallback for
// older translated records that predate gait_lifecycle_events.
func lifecycleEvidenceOccurredAt(record proof.Record, kind, fallback string) string {
	preferred := map[string][]string{
		"authority":    {"proposal_ingested"},
		"readiness":    {"decision_ready"},
		"correlation":  {"proposal_ingested"},
		"enforcement":  {"activated"},
		"execution":    {"execution_started", "execution_blocked", "execution_succeeded", "execution_failed"},
		"effect":       {"effect_recorded", "effect_validated"},
		"containment":  {"containment_requested", "containment_completed", "containment_partial", "containment_unresolved"},
		"compensation": {"compensation_required", "compensation_started", "compensation_completed"},
	}
	kinds := preferred[kind]
	if len(kinds) == 0 {
		return fallback
	}
	wanted := map[string]struct{}{}
	for _, value := range kinds {
		wanted[value] = struct{}{}
	}
	var selected string
	if values, ok := record.Event["gait_lifecycle_events"].([]any); ok {
		for _, value := range values {
			item, ok := value.(map[string]any)
			if !ok {
				continue
			}
			if _, ok := wanted[stringFieldAC(item, "kind")]; ok && stringFieldAC(item, "occurred_at") != "" {
				selected = stringFieldAC(item, "occurred_at")
			}
		}
	}
	if values, ok := record.Event["gait_lifecycle_events"].([]map[string]any); ok {
		for _, item := range values {
			if _, ok := wanted[stringFieldAC(item, "kind")]; ok && stringFieldAC(item, "occurred_at") != "" {
				selected = stringFieldAC(item, "occurred_at")
			}
		}
	}
	if selected != "" {
		return selected
	}
	return fallback
}

type lifecycleTransition struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	OccurredAt   string `json:"occurred_at"`
	SourceDigest string `json:"source_digest"`
	Ref          Ref    `json:"evidence_ref"`
}

func lifecycleTransitions(record proof.Record, axis string) []lifecycleTransition {
	var wanted map[string]struct{}
	switch axis {
	case "execution":
		wanted = map[string]struct{}{"execution_started": {}, "execution_blocked": {}, "execution_succeeded": {}, "execution_failed": {}}
	case "effect":
		wanted = map[string]struct{}{"effect_recorded": {}, "effect_validated": {}}
	case "compensation":
		wanted = map[string]struct{}{"compensation_required": {}, "compensation_started": {}, "compensation_completed": {}}
	case "containment":
		wanted = map[string]struct{}{"containment_requested": {}, "containment_completed": {}, "containment_partial": {}, "containment_unresolved": {}}
	default:
		return nil
	}
	raw, err := json.Marshal(record.Event["gait_lifecycle_events"])
	if err != nil {
		return nil
	}
	var values []lifecycleTransition
	if json.Unmarshal(raw, &values) != nil {
		return nil
	}
	out := make([]lifecycleTransition, 0, len(values))
	for _, value := range values {
		if _, ok := wanted[value.Kind]; ok && value.ID != "" && value.OccurredAt != "" {
			out = append(out, value)
		}
	}
	return out
}

func lifecycleTransitionState(axis, sourceKind string) string {
	switch sourceKind {
	case "execution_started", "compensation_started":
		return "started"
	case "execution_blocked":
		return "blocked"
	case "execution_succeeded":
		return "succeeded"
	case "effect_recorded":
		return "recorded"
	case "effect_validated":
		return "validated"
	case "execution_failed", "compensation_failed":
		return "failed"
	case "compensation_required":
		return "required"
	case "compensation_completed":
		return "completed"
	case "containment_requested":
		return "requested"
	case "containment_partial":
		return "partial"
	case "containment_completed":
		return "completed"
	case "containment_unresolved":
		return "gap"
	default:
		return "unknown"
	}
}

func lifecycleTransitionTerminal(axis, sourceKind string) bool {
	switch axis {
	case "execution":
		return sourceKind == "execution_blocked" || sourceKind == "execution_succeeded" || sourceKind == "execution_failed"
	case "compensation":
		return sourceKind == "compensation_completed" || sourceKind == "compensation_failed"
	case "effect":
		return sourceKind == "effect_validated"
	case "containment":
		return sourceKind == "containment_completed" || sourceKind == "containment_partial" || sourceKind == "containment_unresolved"
	default:
		return false
	}
}

func lifecycleProvenance(record proof.Record, refs []Ref, sourceDigest string) Ref {
	for _, ref := range refs {
		if ref.Digest == sourceDigest {
			return ref
		}
	}
	return Ref{ID: sourceDigest, Kind: "gait_lifecycle_source", Digest: sourceDigest, Source: "gait", SourceProduct: "gait", SchemaID: "https://gait.dev/schemas/v1/runtime-lifecycle-record.schema.json", SchemaVersion: "1"}
}

func lifecycleRefsFromValue(value any) []Ref {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var values []Ref
	if json.Unmarshal(raw, &values) != nil {
		return nil
	}
	for i := range values {
		if values[i].Source == "" {
			values[i].Source = values[i].SourceProduct
		}
	}
	return values
}
func lifecycleRefFromValue(value any) Ref {
	raw, err := json.Marshal(value)
	if err != nil {
		return Ref{}
	}
	var ref Ref
	_ = json.Unmarshal(raw, &ref)
	if ref.Source == "" {
		ref.Source = ref.SourceProduct
	}
	return ref
}

func validDigestAC(value string) bool { return digestPattern.MatchString(strings.TrimSpace(value)) }
func boolFieldAC(object map[string]any, key string) bool {
	value, _ := object[key].(bool)
	return value
}
func stringSliceAC(object map[string]any, key string) []string {
	out := make([]string, 0)
	switch values := object[key].(type) {
	case []string:
		out = append(out, values...)
	case []any:
		for _, value := range values {
			if s, ok := value.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}
func allValidDigestsAC(values []string) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if !validDigestAC(value) {
			return false
		}
	}
	return true
}
func nonEmptyArrayAC(value any) bool {
	rv := reflect.ValueOf(value)
	return rv.IsValid() && (rv.Kind() == reflect.Array || rv.Kind() == reflect.Slice) && rv.Len() > 0
}

func recordBelongs(record proof.Record, contract Contract) bool {
	contractID := contract.ID
	if value, ok := record.Event["contract_ref"].(map[string]any); ok {
		id, _ := value["id"].(string)
		kind, _ := value["kind"].(string)
		contractDigest, _ := value["digest"].(string)
		contractSchema, _ := value["schema_id"].(string)
		contractSourceProduct, _ := value["source_product"].(string)
		if kind != "action_contract" {
			return false
		}
		return exactContractJoin(id, contractID, contract.Provenance.Digest, contract.Provenance.SchemaID, contract.Provenance.SourceProduct, contractDigest, contractSchema, contractSourceProduct) || exactContractJoin(id, contractID, contract.CausalRef.Digest, contract.CausalRef.SchemaID, contract.CausalRef.SourceProduct, contractDigest, contractSchema, contractSourceProduct)
	}
	// A bare contract_id is intentionally not a causal join: it lacks the
	// digest/schema/source binding needed to distinguish revisions.
	if record.Relationship != nil {
		for _, ref := range record.Relationship.EntityRefs {
			if ref.Kind == "action_contract" && (exactContractJoin(ref.ID, contractID, contract.Provenance.Digest, contract.Provenance.SchemaID, contract.Provenance.SourceProduct, ref.Digest, ref.SchemaID, ref.SourceProduct) || exactContractJoin(ref.ID, contractID, contract.CausalRef.Digest, contract.CausalRef.SchemaID, contract.CausalRef.SourceProduct, ref.Digest, ref.SchemaID, ref.SourceProduct)) {
				return true
			}
		}
	}
	return false
}

func exactContractJoin(id, contractID, expectedDigest, expectedSchema, expectedSource, digest, schemaID, sourceProduct string) bool {
	return id == contractID && digest != "" && schemaID != "" && sourceProduct != "" && digest == expectedDigest && schemaID == expectedSchema && sourceProduct == expectedSource
}

func constraintValue(contract map[string]any, key string) string {
	for _, item := range objectArrayAC(contract, "target_constraints") {
		if stringFieldAC(item, "key") == key {
			return stringFieldAC(item, "value")
		}
	}
	return ""
}

func objectArrayAC(object map[string]any, key string) []map[string]any {
	values, _ := object[key].([]map[string]any)
	if values != nil {
		return values
	}
	items, _ := object[key].([]any)
	out := make([]map[string]any, 0, len(items))
	for _, value := range items {
		if item, ok := value.(map[string]any); ok {
			out = append(out, item)
		}
	}
	return out
}

func objectOrArrayAC(object map[string]any, key string) []map[string]any {
	if value, ok := object[key].(map[string]any); ok {
		return []map[string]any{value}
	}
	return objectArrayAC(object, key)
}

func stringFieldAC(object map[string]any, key string) string {
	value, _ := object[key].(string)
	return strings.TrimSpace(value)
}
