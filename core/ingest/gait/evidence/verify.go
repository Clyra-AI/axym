package evidence

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	proofcanon "github.com/Clyra-AI/proof/canon"
	proofsign "github.com/Clyra-AI/proof/signing"
)

func ParseLifecyclePack(raw []byte) (LifecyclePack, error) {
	if err := rejectDuplicateKeys(raw); err != nil {
		return LifecyclePack{}, fmt.Errorf("%s: %w", ReasonMalformed, err)
	}
	var pack LifecyclePack
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&pack); err != nil {
		return LifecyclePack{}, fmt.Errorf("%s: %w", ReasonUnknownField, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return LifecyclePack{}, fmt.Errorf("%s: trailing JSON", ReasonMalformed)
	}
	if len(pack.Records) == 0 {
		return LifecyclePack{}, errors.New(ReasonEvidenceMissing)
	}
	return pack, nil
}

func VerifyLifecyclePack(pack LifecyclePack, options VerificationOptions) VerificationResult {
	result := VerificationResult{FixtureOnly: options.AllowFixtureOnly}
	result.EvidenceSet = EvidenceSet{
		EvidenceSetID:         "gait_lifecycle_v1",
		SourceProduct:         GaitProducer,
		ProducerVersion:       FixtureTag,
		FixtureOnly:           options.AllowFixtureOnly,
		VerificationState:     "unverified",
		SourceArtifactDigests: []string{},
	}
	add := func(reason string) {
		if reason == "" {
			return
		}
		for _, existing := range result.ReasonCodes {
			if existing == reason {
				return
			}
		}
		result.ReasonCodes = append(result.ReasonCodes, reason)
	}
	if len(options.TrustedPublicKey) != ed25519.PublicKeySize {
		add(ReasonTrustedKeyRequired)
		return finish(result)
	}
	if options.EvaluationTime.IsZero() {
		add(ReasonReadinessInvalid)
		return finish(result)
	}
	if !options.ActivationNotBefore.IsZero() && options.EvaluationTime.Before(options.ActivationNotBefore) {
		add(ReasonActivationNotYetValid)
	}
	if !options.ActivationNotAfter.IsZero() && !options.EvaluationTime.Before(options.ActivationNotAfter) {
		add(ReasonActivationExpired)
	}
	if len(pack.Records) == 0 {
		add(ReasonEvidenceMissing)
		return finish(result)
	}
	var previous time.Time
	seen := map[string]struct{}{}
	var binding *EvidenceBinding
	var executionStarted, executionTerminal, effectRecorded, effectValidated, containmentRequested, containmentTerminal bool
	var executionOutcome string
	var executionRef, effectRef, containmentRef, compensationRef Ref
	var compensationNeeded, compensationRequired, compensationStarted, compensationCompleted bool
	for i := range pack.Records {
		record := &pack.Records[i]
		if record.SchemaID != LifecycleSchemaID || record.SchemaVersion != LifecycleSchemaVersion {
			add(ReasonSchemaUnsupported)
			continue
		}
		if record.RecordID == "" {
			add(ReasonMalformed)
			continue
		}
		if _, ok := seen[record.RecordID]; ok {
			add(ReasonReplay)
			continue
		}
		seen[record.RecordID] = struct{}{}
		tm, err := parseTime(record.OccurredAt)
		if err != nil {
			add(ReasonMalformed)
			continue
		}
		if !previous.IsZero() && tm.Before(previous) {
			add(ReasonEvidenceOrder)
		}
		previous = tm
		if err := verifyLifecycleRecord(*record, options.TrustedPublicKey); err != nil {
			add(mapReason(err))
			continue
		}
		if err := validateRecordLineage(*record, options, binding); err != nil {
			add(mapReason(err))
			continue
		}
		if typed := typedBinding(*record); typed != nil {
			if binding == nil {
				copyBinding := *typed
				binding = &copyBinding
			} else if !sameBinding(*binding, *typed) {
				add(ReasonLineageMismatch)
			}
		}
		if typed := typedEvidenceRef(*record); typed != nil {
			if !containsRef(record.EvidenceRefs, *typed) {
				add(ReasonLineageMissing)
			}
		}
		switch record.Kind {
		case "proposal_ingested":
		case "activation_requested":
		case "precondition_evaluated":
		case "decision_ready":
		case "activated":
		case "execution_started":
			if record.Execution == nil || record.Execution.Outcome != "started" || executionStarted || executionTerminal {
				add(ReasonEvidenceOrder)
				continue
			}
			executionStarted = true
			executionRef = evidenceRef("execution", record.Execution.EvidenceID, record.Execution.CanonicalContentDigest, ExecutionSchemaID)
			compensationNeeded = record.Execution.CompensationRequired
			result.Snapshot.ExecutionStatus = "started"
		case "execution_blocked":
			if record.Execution == nil || record.Execution.Outcome != "blocked" || executionStarted || executionTerminal {
				add(ReasonEvidenceOrder)
				continue
			}
			executionTerminal = true
			executionOutcome = "blocked"
			executionRef = evidenceRef("execution", record.Execution.EvidenceID, record.Execution.CanonicalContentDigest, ExecutionSchemaID)
			compensationNeeded = record.Execution.CompensationRequired
			result.Snapshot.ExecutionStatus = executionOutcome
		case "execution_succeeded", "execution_failed":
			if record.Execution == nil || executionTerminal || !executionStarted {
				add(ReasonEvidenceOrder)
				continue
			}
			want := strings.TrimPrefix(record.Kind, "execution_")
			if record.Execution.Outcome != want {
				add(ReasonOutcomeInvalid)
				continue
			}
			executionTerminal = true
			executionOutcome = want
			executionRef = evidenceRef("execution", record.Execution.EvidenceID, record.Execution.CanonicalContentDigest, ExecutionSchemaID)
			compensationNeeded = compensationNeeded || record.Execution.CompensationRequired
			result.Snapshot.ExecutionStatus = executionOutcome
		case "effect_recorded":
			if record.Effect == nil || record.Effect.Outcome != "recorded" || !executionTerminal || executionOutcome != "succeeded" || effectRecorded {
				add(ReasonEvidenceOrder)
				continue
			}
			if !sameRef(record.Effect.ExecutionRef, executionRef) || !hasCausal(record.Effect.Binding, executionRef) {
				add(ReasonLineageMismatch)
				continue
			}
			effectRecorded = true
			effectRef = evidenceRef("effect_event", record.Effect.EvidenceID, record.Effect.CanonicalContentDigest, EffectSchemaID)
			result.Snapshot.EffectStatus = "recorded"
		case "effect_validated":
			if record.Effect == nil || record.Effect.Outcome != "validated" || !effectRecorded || effectValidated {
				add(ReasonEvidenceOrder)
				continue
			}
			if !sameRef(record.Effect.ExecutionRef, executionRef) || !sameRef(record.Effect.EffectRef, record.Effect.EffectRef) || !hasCausal(record.Effect.Binding, effectRef) {
				add(ReasonLineageMismatch)
				continue
			}
			effectValidated = true
			effectRef = evidenceRef("effect_event", record.Effect.EvidenceID, record.Effect.CanonicalContentDigest, EffectSchemaID)
			result.Snapshot.EffectStatus = "validated"
		case "containment_requested":
			if record.Containment == nil || record.Containment.Outcome != "requested" || !effectValidated || containmentRequested || containmentTerminal {
				add(ReasonEvidenceOrder)
				continue
			}
			if !sameRef(record.Containment.ExecutionRef, executionRef) || !sameRef(record.Containment.EffectRef, effectRef) || !hasCausal(record.Containment.Binding, effectRef) {
				add(ReasonLineageMismatch)
				continue
			}
			containmentRequested = true
			containmentRef = evidenceRef("containment", record.Containment.EvidenceID, record.Containment.CanonicalContentDigest, ContainmentSchemaID)
			result.Snapshot.ContainmentStatus = "requested"
		case "containment_completed", "containment_partial", "containment_unresolved":
			if record.Containment == nil || !containmentRequested || containmentTerminal {
				add(ReasonEvidenceOrder)
				continue
			}
			want := strings.TrimPrefix(record.Kind, "containment_")
			if record.Containment.Outcome != want || !sameRef(record.Containment.ExecutionRef, executionRef) || !sameRef(record.Containment.EffectRef, effectRef) || !hasCausal(record.Containment.Binding, containmentRef) {
				add(ReasonLineageMismatch)
				continue
			}
			containmentTerminal = true
			result.Snapshot.ContainmentStatus = want
		case "compensation_required":
			if record.Compensation == nil || record.Compensation.Outcome != "required" || !executionTerminal || !compensationNeeded || compensationRequired {
				add(ReasonEvidenceOrder)
				continue
			}
			if !sameRef(record.Compensation.ExecutionRef, executionRef) || !hasCausal(record.Compensation.Binding, executionRef) {
				add(ReasonLineageMismatch)
				continue
			}
			compensationRequired = true
			compensationRef = evidenceRef("compensation", record.Compensation.EvidenceID, record.Compensation.CanonicalContentDigest, CompensationSchemaID)
			result.Snapshot.CompensationStatus = "required"
		case "compensation_started":
			if record.Compensation == nil || record.Compensation.Outcome != "started" || !compensationRequired || compensationStarted || compensationCompleted {
				add(ReasonEvidenceOrder)
				continue
			}
			if !sameRef(record.Compensation.ExecutionRef, executionRef) || !hasCausal(record.Compensation.Binding, compensationRef) {
				add(ReasonLineageMismatch)
				continue
			}
			compensationStarted = true
			compensationRef = evidenceRef("compensation", record.Compensation.EvidenceID, record.Compensation.CanonicalContentDigest, CompensationSchemaID)
			result.Snapshot.CompensationStatus = "started"
		case "compensation_completed":
			if record.Compensation == nil || record.Compensation.Outcome != "completed" || !compensationStarted || compensationCompleted {
				add(ReasonEvidenceOrder)
				continue
			}
			if !sameRef(record.Compensation.ExecutionRef, executionRef) || !hasCausal(record.Compensation.Binding, compensationRef) {
				add(ReasonLineageMismatch)
				continue
			}
			compensationCompleted = true
			result.Snapshot.CompensationStatus = "completed"
		default:
			add(ReasonOutcomeInvalid)
		}
	}
	if binding == nil {
		add(ReasonLineageMissing)
	}
	if len(result.ReasonCodes) == 0 {
		if executionOutcome == "blocked" || executionOutcome == "failed" {
			if result.Snapshot.EffectStatus != "" || result.Snapshot.ContainmentStatus != "" {
				add(ReasonEvidenceOrder)
			}
		} else if executionOutcome != "succeeded" {
			add(ReasonEvidenceMissing)
		} else if result.Snapshot.EffectStatus != "validated" {
			add(ReasonEvidenceMissing)
		} else if result.Snapshot.ContainmentStatus != "completed" && result.Snapshot.ContainmentStatus != "partial" && result.Snapshot.ContainmentStatus != "unresolved" {
			add(ReasonEvidenceMissing)
		}
		if compensationNeeded && !compensationCompleted {
			add(ReasonEvidenceMissing)
		}
	}
	if len(result.ReasonCodes) == 0 {
		result.Valid = true
		result.EvidenceSet.Verified = true
		result.EvidenceSet.VerificationState = "verified"
		result.EvidenceSet.Execution = result.Snapshot.ExecutionStatus
		result.EvidenceSet.Effect = result.Snapshot.EffectStatus
		result.EvidenceSet.ContainmentStatus = result.Snapshot.ContainmentStatus
		result.EvidenceSet.CompensationStatus = result.Snapshot.CompensationStatus
		result.Authoritative = !options.AllowFixtureOnly
		result.EvidenceSet.Authoritative = result.Authoritative
		if options.AllowFixtureOnly {
			add(ReasonFixtureNonAuthoritative)
			result.Valid = true
		}
	}
	return finish(result)
}

func finish(result VerificationResult) VerificationResult {
	sort.Strings(result.ReasonCodes)
	result.EvidenceSet.ReasonCodes = append([]string(nil), result.ReasonCodes...)
	return result
}

func VerifyLifecycleRecord(record LifecycleRecord, publicKey ed25519.PublicKey) (bool, error) {
	return verifyLifecycleRecord(record, publicKey) == nil, verifyLifecycleRecord(record, publicKey)
}

func verifyLifecycleRecord(record LifecycleRecord, publicKey ed25519.PublicKey) error {
	if record.SchemaID != LifecycleSchemaID || record.SchemaVersion != LifecycleSchemaVersion || record.RecordID == "" {
		return errors.New(ReasonSchemaUnsupported)
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return errors.New(ReasonTrustedKeyRequired)
	}
	if err := validateCorrelation(record.Correlation, record.ContractRef); err != nil {
		return fmt.Errorf("%s: %w", ReasonCorrelationInvalid, err)
	}
	if err := validateTyped(record); err != nil {
		return err
	}
	if typed := typedTimes(record); typed != nil {
		recordTime, err := parseTime(record.OccurredAt)
		if err != nil || recordTime.Before(typed.occurred) || recordTime.After(typed.freshUntil) {
			return errors.New(ReasonTimeWindow)
		}
	}
	if typed := typedProvenance(record); typed != nil {
		if err := verifyTypedProvenance(*typed, publicKey); err != nil {
			return err
		}
	}
	digest, err := lifecycleDigest(record)
	if err != nil || record.Signature.SignedDigest != strings.TrimPrefix(digest, "sha256:") {
		return errors.New(ReasonDigestMismatch)
	}
	if record.Signature.KeyID != proofsign.KeyID(publicKey) {
		return errors.New(ReasonSignatureInvalid)
	}
	ok, err := proofsign.VerifyDigestHex(publicKey, record.Signature)
	if err != nil || !ok {
		return errors.New(ReasonSignatureInvalid)
	}
	if record.RecordID != "gait-lr-"+strings.TrimPrefix(digest, "sha256:")[:16] {
		return errors.New(ReasonDigestMismatch)
	}
	return nil
}

func validateTyped(record LifecycleRecord) error {
	count := 0
	if record.Execution != nil {
		count++
	}
	if record.Effect != nil {
		count++
	}
	if record.Containment != nil {
		count++
	}
	if record.Compensation != nil {
		count++
	}
	if count > 1 {
		return errors.New(ReasonEvidenceOrder)
	}
	if record.Execution != nil {
		if err := validateExecution(*record.Execution); err != nil {
			return err
		}
		if record.Kind != "execution_started" && record.Kind != "execution_succeeded" && record.Kind != "execution_failed" && record.Kind != "execution_blocked" {
			return errors.New(ReasonOutcomeInvalid)
		}
	}
	if record.Effect != nil {
		if err := validateEffect(*record.Effect); err != nil {
			return err
		}
		if record.Kind != "effect_recorded" && record.Kind != "effect_validated" {
			return errors.New(ReasonOutcomeInvalid)
		}
	}
	if record.Containment != nil {
		if err := validateContainment(*record.Containment); err != nil {
			return err
		}
		if !strings.HasPrefix(record.Kind, "containment_") {
			return errors.New(ReasonOutcomeInvalid)
		}
	}
	if record.Compensation != nil {
		if err := validateCompensation(*record.Compensation); err != nil {
			return err
		}
		if !strings.HasPrefix(record.Kind, "compensation_") {
			return errors.New(ReasonOutcomeInvalid)
		}
	}
	return nil
}

func validateExecution(item ExecutionEvidence) error {
	if item.SchemaID != ExecutionSchemaID || item.SchemaVersion != EvidenceSchemaVersion || item.Outcome != "started" && item.Outcome != "succeeded" && item.Outcome != "failed" && item.Outcome != "blocked" {
		return errors.New(ReasonSchemaUnsupported)
	}
	return validateCommon(item.EvidenceID, "gait-exec-", item.OccurredAt, item.FreshUntil, item.ReasonCode, item.Binding, item.EventRef, item.Provenance, item.CanonicalContentDigest, item)
}
func validateEffect(item EffectEvent) error {
	if item.SchemaID != EffectSchemaID || item.SchemaVersion != EvidenceSchemaVersion || item.Outcome != "recorded" && item.Outcome != "validated" {
		return errors.New(ReasonSchemaUnsupported)
	}
	if err := validateCommon(item.EvidenceID, "gait-effect-", item.OccurredAt, item.FreshUntil, item.ReasonCode, item.Binding, item.EventRef, item.Provenance, item.CanonicalContentDigest, item); err != nil {
		return err
	}
	if !validRefKind(item.ExecutionRef, "execution", ExecutionSchemaID) || !validRef(item.EffectRef) {
		return errors.New(ReasonLineageMissing)
	}
	return nil
}
func validateContainment(item ContainmentEvidence) error {
	if item.SchemaID != ContainmentSchemaID || item.SchemaVersion != EvidenceSchemaVersion || item.Outcome != "requested" && item.Outcome != "completed" && item.Outcome != "partial" && item.Outcome != "unresolved" {
		return errors.New(ReasonSchemaUnsupported)
	}
	if err := validateCommon(item.EvidenceID, "gait-containment-", item.OccurredAt, item.FreshUntil, item.ReasonCode, item.Binding, item.EventRef, item.Provenance, item.CanonicalContentDigest, item); err != nil {
		return err
	}
	if !validRefKind(item.ExecutionRef, "execution", ExecutionSchemaID) || !validRef(item.EffectRef) || !validRef(item.ContainmentRef) {
		return errors.New(ReasonLineageMissing)
	}
	return nil
}
func validateCompensation(item CompensationEvidence) error {
	if item.SchemaID != CompensationSchemaID || item.SchemaVersion != EvidenceSchemaVersion || item.Outcome != "required" && item.Outcome != "started" && item.Outcome != "completed" {
		return errors.New(ReasonSchemaUnsupported)
	}
	if err := validateCommon(item.EvidenceID, "gait-compensation-", item.OccurredAt, item.FreshUntil, item.ReasonCode, item.Binding, item.EventRef, item.Provenance, item.CanonicalContentDigest, item); err != nil {
		return err
	}
	if !validRefKind(item.ExecutionRef, "execution", ExecutionSchemaID) || !validRef(item.RequirementRef) {
		return errors.New(ReasonLineageMissing)
	}
	return nil
}

func validateCommon(id, prefix, occurred, fresh, reason string, binding EvidenceBinding, event Ref, provenance Provenance, digest string, value any) error {
	if reason == "" || id == "" || !strings.HasPrefix(id, prefix) || !validRef(event) {
		return errors.New(ReasonMalformed)
	}
	if err := validateBinding(binding); err != nil {
		return err
	}
	start, err := parseTime(occurred)
	if err != nil {
		return errors.New(ReasonTimeWindow)
	}
	end, err := parseTime(fresh)
	if err != nil || end.Before(start) {
		return errors.New(ReasonTimeWindow)
	}
	if provenance.Producer != GaitProducer || provenance.Writer == "" || provenance.Verifier == "" || provenance.PublicKey == "" || provenance.Signature.Alg != "ed25519" {
		return errors.New(ReasonProducerInvalid)
	}
	if !validDigest(digest) {
		return errors.New(ReasonDigestMismatch)
	}
	computed, err := canonicalEvidenceDigest(value)
	if err != nil || computed != digest || provenance.Signature.SignedDigest != strings.TrimPrefix(digest, "sha256:") {
		return errors.New(ReasonDigestMismatch)
	}
	return nil
}

func validateBinding(binding EvidenceBinding) error {
	if binding.ContractFamilyID == "" || binding.Revision < 1 || len(binding.ProofRefs) == 0 || len(binding.CausalRefs) == 0 {
		return errors.New(ReasonLineageMissing)
	}
	refs := []Ref{binding.ContractRef, binding.ActivationRef, binding.RuntimeActionRef, binding.ReadinessRef, binding.DecisionRef, binding.PolicyRef, binding.TargetRef, binding.EnvironmentRef}
	for _, ref := range refs {
		if !validRef(ref) {
			return errors.New(ReasonLineageMissing)
		}
	}
	if !validRefKind(binding.ContractRef, "action_contract", "https://wrkr.dev/schemas/v1/proposed-action-contract-v3.schema.json") || binding.ContractRef.SourceProduct != WrkrProducer {
		return errors.New(ReasonLineageMismatch)
	}
	if !validRefKind(binding.ActivationRef, "activated_action_contract", "https://gait.dev/schemas/v1/activated-action-contract-artifact.schema.json") || binding.ActivationRef.SourceProduct != GaitProducer {
		return errors.New(ReasonLineageMismatch)
	}
	if binding.RuntimeActionRef.Kind != "runtime_action" || binding.ReadinessRef.Kind != "readiness" || binding.DecisionRef.Kind != "decision" || binding.PolicyRef.Kind != "policy" || binding.TargetRef.Kind != "target" || binding.EnvironmentRef.Kind != "environment" {
		return errors.New(ReasonLineageMismatch)
	}
	if binding.Correlation.BindingMode != DigestBound || binding.Correlation.ProfileVersion != "1.0" || binding.Correlation.ContractRef == nil || !sameRef(*binding.Correlation.ContractRef, binding.ContractRef) || binding.Correlation.ContentDigest != binding.ContractRef.Digest {
		return errors.New(ReasonCorrelationInvalid)
	}
	for _, ref := range append(append([]Ref{}, binding.ProofRefs...), binding.CausalRefs...) {
		if !validRef(ref) {
			return errors.New(ReasonLineageMissing)
		}
	}
	return nil
}

func validateCorrelation(c Correlation, contract Ref) error {
	if c.BindingMode != DigestBound || c.ProfileVersion != "1.0" || c.ContractRef == nil || !sameRef(*c.ContractRef, contract) || c.ContentDigest != contract.Digest {
		return errors.New(ReasonCorrelationInvalid)
	}
	return nil
}

func validateRecordLineage(record LifecycleRecord, options VerificationOptions, current *EvidenceBinding) error {
	if options.ExpectedContract.ID != "" && !sameRef(record.ContractRef, options.ExpectedContract) {
		return errors.New(ReasonLineageMismatch)
	}
	if options.ExpectedFamily != "" && record.ContractFamilyID != options.ExpectedFamily {
		return errors.New(ReasonLineageMismatch)
	}
	if options.ExpectedRevision > 0 && record.Revision != options.ExpectedRevision {
		return errors.New(ReasonLineageMismatch)
	}
	if record.ProposalRef != nil && !sameRef(*record.ProposalRef, record.ContractRef) {
		return errors.New(ReasonLineageMismatch)
	}
	if strings.HasPrefix(record.Kind, "execution_") || strings.HasPrefix(record.Kind, "effect_") || strings.HasPrefix(record.Kind, "containment_") || strings.HasPrefix(record.Kind, "compensation_") || record.Kind == "activated" {
		if record.ActivationRef == nil {
			return errors.New(ReasonLineageMissing)
		}
	}
	if record.ActivationRef != nil && options.ExpectedActivation.ID != "" && !sameRef(*record.ActivationRef, options.ExpectedActivation) {
		return errors.New(ReasonLineageMismatch)
	}
	if typed := typedBinding(record); typed != nil {
		if typed.ContractFamilyID != record.ContractFamilyID || typed.Revision != record.Revision || !sameRef(typed.ContractRef, record.ContractRef) || record.ProposalRef == nil || record.ActivationRef == nil || !sameRef(typed.ActivationRef, *record.ActivationRef) {
			return errors.New(ReasonLineageMismatch)
		}
		if current != nil && !sameBinding(*current, *typed) {
			return errors.New(ReasonLineageMismatch)
		}
		if options.ExpectedRuntimeDigest != "" && typed.RuntimeActionRef.Digest != options.ExpectedRuntimeDigest {
			return errors.New(ReasonLineageMismatch)
		}
		if options.ExpectedReadinessDigest != "" && (typed.ReadinessRef.Digest != options.ExpectedReadinessDigest || typed.DecisionRef.Digest != options.ExpectedReadinessDigest) {
			return errors.New(ReasonLineageMismatch)
		}
		if options.ExpectedPolicyDigest != "" && typed.PolicyRef.Digest != options.ExpectedPolicyDigest {
			return errors.New(ReasonLineageMismatch)
		}
		if options.ExpectedTarget != "" && typed.TargetRef.ID != options.ExpectedTarget {
			return errors.New(ReasonLineageMismatch)
		}
		if options.ExpectedEnvironment != "" && typed.EnvironmentRef.ID != options.ExpectedEnvironment {
			return errors.New(ReasonLineageMismatch)
		}
	}
	return nil
}

func typedBinding(record LifecycleRecord) *EvidenceBinding {
	switch {
	case record.Execution != nil:
		return &record.Execution.Binding
	case record.Effect != nil:
		return &record.Effect.Binding
	case record.Containment != nil:
		return &record.Containment.Binding
	case record.Compensation != nil:
		return &record.Compensation.Binding
	default:
		return nil
	}
}

func typedProvenance(record LifecycleRecord) *Provenance {
	switch {
	case record.Execution != nil:
		return &record.Execution.Provenance
	case record.Effect != nil:
		return &record.Effect.Provenance
	case record.Containment != nil:
		return &record.Containment.Provenance
	case record.Compensation != nil:
		return &record.Compensation.Provenance
	default:
		return nil
	}
}

type evidenceTimes struct {
	occurred   time.Time
	freshUntil time.Time
}

func typedTimes(record LifecycleRecord) *evidenceTimes {
	var occurred, fresh string
	switch {
	case record.Execution != nil:
		occurred, fresh = record.Execution.OccurredAt, record.Execution.FreshUntil
	case record.Effect != nil:
		occurred, fresh = record.Effect.OccurredAt, record.Effect.FreshUntil
	case record.Containment != nil:
		occurred, fresh = record.Containment.OccurredAt, record.Containment.FreshUntil
	case record.Compensation != nil:
		occurred, fresh = record.Compensation.OccurredAt, record.Compensation.FreshUntil
	default:
		return nil
	}
	start, err := parseTime(occurred)
	if err != nil {
		return &evidenceTimes{}
	}
	end, err := parseTime(fresh)
	if err != nil {
		return &evidenceTimes{}
	}
	return &evidenceTimes{occurred: start, freshUntil: end}
}

func verifyTypedProvenance(provenance Provenance, publicKey ed25519.PublicKey) error {
	declared, err := base64.StdEncoding.DecodeString(strings.TrimSpace(provenance.PublicKey))
	if err != nil || len(declared) != ed25519.PublicKeySize || !bytes.Equal(declared, publicKey) {
		return errors.New(ReasonPublicKeyMismatch)
	}
	if provenance.Signature.KeyID != proofsign.KeyID(publicKey) {
		return errors.New(ReasonSignatureInvalid)
	}
	ok, err := proofsign.VerifyDigestHex(publicKey, provenance.Signature)
	if err != nil || !ok {
		return errors.New(ReasonSignatureInvalid)
	}
	return nil
}
func typedEvidenceRef(record LifecycleRecord) *Ref {
	switch {
	case record.Execution != nil:
		r := evidenceRef("execution", record.Execution.EvidenceID, record.Execution.CanonicalContentDigest, ExecutionSchemaID)
		return &r
	case record.Effect != nil:
		r := evidenceRef("effect_event", record.Effect.EvidenceID, record.Effect.CanonicalContentDigest, EffectSchemaID)
		return &r
	case record.Containment != nil:
		r := evidenceRef("containment", record.Containment.EvidenceID, record.Containment.CanonicalContentDigest, ContainmentSchemaID)
		return &r
	case record.Compensation != nil:
		r := evidenceRef("compensation", record.Compensation.EvidenceID, record.Compensation.CanonicalContentDigest, CompensationSchemaID)
		return &r
	default:
		return nil
	}
}
func evidenceRef(kind, id, digest, schema string) Ref {
	return Ref{Kind: kind, ID: id, Digest: digest, SchemaID: schema, SchemaVersion: EvidenceSchemaVersion, SourceProduct: GaitProducer}
}
func validRef(ref Ref) bool { return ref.Kind != "" && ref.ID != "" && validDigest(ref.Digest) }
func validRefKind(ref Ref, kind, schema string) bool {
	return validRef(ref) && ref.Kind == kind && ref.SchemaID == schema && ref.SchemaVersion == EvidenceSchemaVersion
}
func sameRef(a, b Ref) bool {
	return a.Kind == b.Kind && a.ID == b.ID && a.Digest == b.Digest && (a.SchemaID == "" || b.SchemaID == "" || a.SchemaID == b.SchemaID) && (a.SchemaVersion == "" || b.SchemaVersion == "" || a.SchemaVersion == b.SchemaVersion) && (a.SourceProduct == "" || b.SourceProduct == "" || a.SourceProduct == b.SourceProduct)
}
func containsRef(refs []Ref, want Ref) bool {
	for _, ref := range refs {
		if sameRef(ref, want) {
			return true
		}
	}
	return false
}
func hasCausal(binding EvidenceBinding, want Ref) bool {
	for _, ref := range binding.CausalRefs {
		if sameRef(ref, want) {
			return true
		}
	}
	return false
}
func sameBinding(a, b EvidenceBinding) bool {
	a.CausalRefs = nil
	b.CausalRefs = nil
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return bytes.Equal(left, right)
}

func canonicalEvidenceDigest(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	var obj map[string]any
	if err = json.Unmarshal(raw, &obj); err != nil {
		return "", err
	}
	delete(obj, "canonical_content_digest")
	delete(obj, "evidence_id")
	if p, ok := obj["provenance"].(map[string]any); ok {
		delete(p, "signature")
	}
	return proofcanon.DigestJCS(mustJSON(obj))
}
func lifecycleDigest(record LifecycleRecord) (string, error) {
	copy := record
	copy.RecordID = ""
	copy.Signature = proofsign.Signature{}
	raw, err := json.Marshal(copy)
	if err != nil {
		return "", err
	}
	return proofcanon.DigestJCS(raw)
}
func mustJSON(value any) []byte { raw, _ := json.Marshal(value); return raw }
func validDigest(value string) bool {
	value = strings.TrimPrefix(strings.TrimSpace(value), "sha256:")
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
func parseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
}
func canonicalDigest(raw []byte) (string, error) {
	var obj any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", err
	}
	out, err := proofcanon.DigestJCS(mustJSON(obj))
	return out, err
}
func RawDigest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func DecodePublicKey(raw []byte) (ed25519.PublicKey, error) {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(decoded) != ed25519.PublicKeySize {
		return nil, errors.New(ReasonTrustedKeyRequired)
	}
	return ed25519.PublicKey(decoded), nil
}

func mapReason(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	for _, reason := range []string{ReasonSchemaUnsupported, ReasonDigestMismatch, ReasonSignatureInvalid, ReasonTrustedKeyRequired, ReasonPublicKeyMismatch, ReasonProducerInvalid, ReasonLineageMissing, ReasonLineageMismatch, ReasonCorrelationInvalid, ReasonEvidenceMissing, ReasonEvidenceOrder, ReasonReplay, ReasonTimeWindow, ReasonOutcomeInvalid, ReasonReadinessInvalid, ReasonRuntimeInvalid} {
		if strings.Contains(msg, reason) {
			return reason
		}
	}
	return ReasonMalformed
}

func rejectDuplicateKeys(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var walk func() error
	walk = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]struct{}{}
			for decoder.More() {
				tok, err := decoder.Token()
				if err != nil {
					return err
				}
				key, ok := tok.(string)
				if !ok {
					return errors.New("object key required")
				}
				if _, exists := seen[key]; exists {
					return fmt.Errorf("duplicate key %q", key)
				}
				seen[key] = struct{}{}
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		default:
			return fmt.Errorf("unexpected delimiter %q", delim)
		}
	}
	if err := walk(); err != nil {
		return err
	}
	var trailing any
	err := decoder.Decode(&trailing)
	if !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("trailing JSON")
		}
		return err
	}
	return nil
}
