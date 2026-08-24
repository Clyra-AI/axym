package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/Clyra-AI/axym/core/ingest/gait/evidence"
	"github.com/Clyra-AI/proof"
	proofcanon "github.com/Clyra-AI/proof/canon"
)

const (
	ReasonLifecycleAuthorityRequired = "GAIT_LIFECYCLE_AUTHORITY_REQUIRED"
	ReasonLifecyclePackMismatch      = "GAIT_LIFECYCLE_PACK_MISMATCH"
	ReasonLifecycleTranslation       = "GAIT_LIFECYCLE_TRANSLATION_FAILED"
	LifecycleTranslationVersion      = "v1"
)

// TranslateLifecycle converts one verified Gait lifecycle pack into one
// aggregate Proof record. The aggregate is an observation of Gait's verified
// boundary evidence; it is never an execution command or an Axym decision.
func TranslateLifecycle(result evidence.VerificationResult, pack evidence.LifecyclePack) (*proof.Record, error) {
	if !result.Valid || !result.Authoritative || result.FixtureOnly || !result.EvidenceSet.Verified || !result.EvidenceSet.Authoritative || result.EvidenceSet.FixtureOnly || result.EvidenceSet.VerificationState != "verified" {
		return nil, &Error{ReasonCode: ReasonLifecycleAuthorityRequired, Message: "lifecycle result is not verified authoritative evidence"}
	}
	if len(pack.Records) == 0 {
		return nil, &Error{ReasonCode: ReasonLifecyclePackMismatch, Message: "lifecycle pack is empty"}
	}
	setID, err := lifecycleSetID(pack)
	if err != nil || setID != result.EvidenceSet.EvidenceSetID {
		return nil, &Error{ReasonCode: ReasonLifecyclePackMismatch, Message: "lifecycle pack does not match verified evidence set"}
	}
	if result.EvidenceSet.SourceProduct != evidence.GaitProducer || result.EvidenceSet.ProducerVersion == "" {
		return nil, &Error{ReasonCode: ReasonLifecycleTranslation, Message: "lifecycle producer metadata is incomplete"}
	}
	if len(result.EvidenceSet.SourceCommit) != 40 || !validHex(result.EvidenceSet.SourceCommit) || !containsDigest(result.EvidenceSet.SourceArtifactDigests, pack.SourceArtifactDigest) || !validDigests(result.EvidenceSet.SourceArtifactDigests) || !validDigests(result.EvidenceSet.DerivedEvidenceDigests) {
		return nil, &Error{ReasonCode: ReasonLifecyclePackMismatch, Message: "lifecycle provenance metadata does not match the verified source"}
	}
	timestamp, err := lifecycleTimestamp(pack)
	if err != nil {
		return nil, &Error{ReasonCode: ReasonLifecycleTranslation, Message: "lifecycle timestamp is invalid", Err: err}
	}

	contractRef, activationRef, refs := lifecycleRefs(pack)
	if contractRef == nil || activationRef == nil || len(refs) == 0 {
		return nil, &Error{ReasonCode: ReasonLifecyclePackMismatch, Message: "lifecycle contract, activation, and evidence references are required"}
	}
	authoritativeSuccess := result.Snapshot.ExecutionStatus == "succeeded" &&
		result.Snapshot.EffectStatus == "validated" &&
		result.Snapshot.ContainmentStatus == "completed" &&
		(result.Snapshot.CompensationStatus == "" || result.Snapshot.CompensationStatus == "completed")

	event := map[string]any{
		"test_name":                "gait_lifecycle_conformance",
		"status":                   "passed",
		"evidence_set_id":          result.EvidenceSet.EvidenceSetID,
		"producer":                 evidence.GaitProducer,
		"producer_version":         result.EvidenceSet.ProducerVersion,
		"source_commit":            result.EvidenceSet.SourceCommit,
		"translation_version":      LifecycleTranslationVersion,
		"gait_execution":           result.Snapshot.ExecutionStatus,
		"gait_effect":              result.Snapshot.EffectStatus,
		"containment_status":       result.Snapshot.ContainmentStatus,
		"compensation_status":      result.Snapshot.CompensationStatus,
		"authoritative_success":    authoritativeSuccess,
		"contract_ref":             *contractRef,
		"activation_ref":           *activationRef,
		"evidence_refs":            refs,
		"source_artifact_digest":   pack.SourceArtifactDigest,
		"source_artifact_digests":  append([]string(nil), result.EvidenceSet.SourceArtifactDigests...),
		"derived_evidence_digests": append([]string(nil), result.EvidenceSet.DerivedEvidenceDigests...),
		"reason_codes":             append([]string(nil), result.ReasonCodes...),
	}
	metadata := map[string]any{
		"evidence_kind":                 "gait_lifecycle",
		"gait_translation":              LifecycleTranslationVersion,
		"gait_evidence_set_id":          result.EvidenceSet.EvidenceSetID,
		"gait_producer_version":         result.EvidenceSet.ProducerVersion,
		"gait_source_commit":            result.EvidenceSet.SourceCommit,
		"gait_verification_state":       result.EvidenceSet.VerificationState,
		"gait_authoritative":            true,
		"gait_fixture_only":             false,
		"gait_execution":                result.Snapshot.ExecutionStatus,
		"gait_effect":                   result.Snapshot.EffectStatus,
		"gait_containment_status":       result.Snapshot.ContainmentStatus,
		"gait_compensation_status":      result.Snapshot.CompensationStatus,
		"gait_authoritative_success":    authoritativeSuccess,
		"gait_source_artifact_digests":  append([]string(nil), result.EvidenceSet.SourceArtifactDigests...),
		"gait_derived_evidence_digests": append([]string(nil), result.EvidenceSet.DerivedEvidenceDigests...),
		"gait_source_artifact_digest":   pack.SourceArtifactDigest,
		"gait_lifecycle_record_ids":     lifecycleRecordIDs(pack),
	}
	allRefs := append([]evidence.Ref{*contractRef, *activationRef}, refs...)
	relationship := &proof.Relationship{EntityRefs: refsToProof(allRefs)}
	record, err := proof.NewRecord(proof.RecordOpts{
		Timestamp: timestamp, Source: evidence.GaitProducer, SourceProduct: evidence.GaitProducer,
		AgentID: "gait://" + result.EvidenceSet.ProducerVersion, Type: "test_result",
		Event: event, Metadata: metadata, Relationship: relationship,
		Controls: proof.Controls{},
	})
	if err != nil {
		return nil, &Error{ReasonCode: ReasonLifecycleTranslation, Message: "build lifecycle proof record", Err: err}
	}
	return record, nil
}

// TranslateLifecycleEvidence is the descriptive alias used by sibling
// integration callers.
func TranslateLifecycleEvidence(result evidence.VerificationResult, pack evidence.LifecyclePack) (*proof.Record, error) {
	return TranslateLifecycle(result, pack)
}

func lifecycleTimestamp(pack evidence.LifecyclePack) (time.Time, error) {
	last := time.Time{}
	for _, record := range pack.Records {
		parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(record.OccurredAt))
		if err != nil {
			return time.Time{}, err
		}
		if parsed.After(last) {
			last = parsed
		}
	}
	if last.IsZero() {
		return time.Time{}, errors.New("lifecycle timestamp missing")
	}
	return last.UTC(), nil
}

func lifecycleRefs(pack evidence.LifecyclePack) (*evidence.Ref, *evidence.Ref, []evidence.Ref) {
	var contract, activation *evidence.Ref
	seen := map[string]struct{}{}
	refs := make([]evidence.Ref, 0)
	for _, record := range pack.Records {
		if contract == nil {
			copy := record.ContractRef
			contract = &copy
		}
		if activation == nil && record.ActivationRef != nil {
			copy := *record.ActivationRef
			activation = &copy
		}
		for _, ref := range record.EvidenceRefs {
			key := ref.Kind + "|" + ref.ID + "|" + ref.Digest + "|" + ref.SchemaID + "|" + ref.SchemaVersion + "|" + ref.SourceProduct
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			refs = append(refs, ref)
		}
	}
	sort.Slice(refs, func(i, j int) bool {
		left := refs[i].Kind + "|" + refs[i].ID + "|" + refs[i].Digest + "|" + refs[i].SchemaID + "|" + refs[i].SchemaVersion + "|" + refs[i].SourceProduct
		right := refs[j].Kind + "|" + refs[j].ID + "|" + refs[j].Digest + "|" + refs[j].SchemaID + "|" + refs[j].SchemaVersion + "|" + refs[j].SourceProduct
		return left < right
	})
	return contract, activation, refs
}

func refsToProof(refs []evidence.Ref) []proof.RelationshipRef {
	out := make([]proof.RelationshipRef, 0, len(refs))
	for _, ref := range refs {
		// The event retains Gait's exact ref. Proof's graph projection adds a
		// source namespace so producer-specific kinds remain portable.
		kind := strings.TrimSpace(ref.SourceProduct) + "." + strings.TrimSpace(ref.Kind)
		out = append(out, proof.RelationshipRef{Kind: kind, ID: ref.ID, Digest: ref.Digest, SchemaID: ref.SchemaID, SchemaVersion: ref.SchemaVersion, SourceProduct: ref.SourceProduct})
	}
	return out
}

func lifecycleRecordIDs(pack evidence.LifecyclePack) []string {
	ids := make([]string, 0, len(pack.Records))
	for _, record := range pack.Records {
		if record.RecordID != "" {
			ids = append(ids, record.RecordID)
		}
	}
	return ids
}

func lifecycleSetID(pack evidence.LifecyclePack) (string, error) {
	if digest := strings.TrimSpace(pack.SourceArtifactDigest); validSHA256Digest(digest) {
		return "gait_lifecycle_v1:" + strings.TrimPrefix(digest, "sha256:")[:16], nil
	}
	raw, err := json.Marshal(pack)
	if err != nil {
		return "", err
	}
	digest, err := proofcanon.DigestJCS(raw)
	if err != nil {
		return "", err
	}
	digest = strings.TrimPrefix(digest, "sha256:")
	if len(digest) < 16 {
		return "", errors.New("lifecycle digest too short")
	}
	return "gait_lifecycle_v1:" + digest[:16], nil
}

func containsDigest(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func validDigests(values []string) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if !validSHA256Digest(value) {
			return false
		}
	}
	return true
}

func validHex(value string) bool {
	_, err := hex.DecodeString(value)
	return err == nil
}

func validSHA256Digest(value string) bool {
	value = strings.TrimPrefix(strings.TrimSpace(value), "sha256:")
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

// LifecycleTranslationDigest is a stable digest of the translated aggregate's
// semantic payload, useful to downstream conformance tests before signing.
func LifecycleTranslationDigest(record *proof.Record) string {
	if record == nil {
		return ""
	}
	raw, err := json.Marshal(struct {
		Event    map[string]any `json:"event"`
		Metadata map[string]any `json:"metadata"`
	}{record.Event, record.Metadata})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}
