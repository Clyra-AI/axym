// Package evidence is Axym's read-only consumer boundary for the Gait v1.5.0
// Action Contract lifecycle evidence pack. It intentionally keeps the Gait
// producer contract separate from Axym's proof-record and framework models.
package evidence

import (
	"crypto/ed25519"
	"encoding/json"
	"time"

	proofsign "github.com/Clyra-AI/proof/signing"
)

const (
	LifecycleSchemaID      = "https://gait.dev/schemas/v1/runtime-lifecycle-record.schema.json"
	LifecycleSchemaVersion = "1"
	ContractSchemaID       = "https://wrkr.dev/schemas/v1/proposed-action-contract-v3.schema.json"
	ContractSchemaVersion  = "3"
	ActivationSchemaID     = "https://gait.dev/schemas/v1/activated-action-contract-artifact.schema.json"
	RuntimeActionSchemaID  = "https://gait.dev/schemas/v1/runtime-action.schema.json"
	ReadinessSchemaID      = "https://gait.dev/schemas/v1/runtime-readiness.schema.json"
	ExecutionSchemaID      = "https://gait.dev/schemas/v1/action-contract/execution-evidence.schema.json"
	EffectSchemaID         = "https://gait.dev/schemas/v1/action-contract/effect-event.schema.json"
	ContainmentSchemaID    = "https://gait.dev/schemas/v1/action-contract/containment-evidence.schema.json"
	CompensationSchemaID   = "https://gait.dev/schemas/v1/action-contract/compensation-evidence.schema.json"
	EvidenceSchemaVersion  = "1"
	GaitProducer           = "gait"
	WrkrProducer           = "wrkr"
	DigestBound            = "digest_bound"
	FixtureTag             = "v1.5.0"
	FixtureCommit          = "10f8b91b316c30c2202a580847dfdd3509bff458"
	FixtureKeyID           = "42571c8843a10df565fd17a97a236c3552e8e4b7ff2a0b48bf524409279771d9"
)

const (
	ReasonSchemaUnsupported       = "gait_schema_unsupported"
	ReasonUnknownField            = "gait_unknown_field"
	ReasonMalformed               = "gait_malformed"
	ReasonDigestMismatch          = "gait_digest_mismatch"
	ReasonSignatureInvalid        = "gait_signature_invalid"
	ReasonTrustedKeyRequired      = "gait_trusted_key_required"
	ReasonPublicKeyMismatch       = "gait_public_key_mismatch"
	ReasonProducerInvalid         = "gait_producer_invalid"
	ReasonFixtureNonAuthoritative = "gait_fixture_non_authoritative"
	ReasonLineageMissing          = "gait_lineage_missing"
	ReasonLineageMismatch         = "gait_lineage_mismatch"
	ReasonCorrelationInvalid      = "gait_correlation_invalid"
	ReasonEvidenceMissing         = "gait_evidence_missing"
	ReasonEvidenceOrder           = "gait_evidence_order_invalid"
	ReasonReplay                  = "gait_replay"
	ReasonTimeWindow              = "gait_time_window_invalid"
	ReasonOutcomeInvalid          = "gait_outcome_invalid"
	ReasonActivationExpired       = "gait_activation_expired"
	ReasonActivationNotYetValid   = "gait_activation_not_yet_valid"
	ReasonReadinessInvalid        = "gait_readiness_invalid"
	ReasonRuntimeInvalid          = "gait_runtime_invalid"
	ReasonScenarioInvalid         = "gait_scenario_invalid"
	ReasonSourceProvenanceInvalid = "gait_source_provenance_invalid"
)

type Ref struct {
	Digest        string `json:"digest"`
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	SourceProduct string `json:"source_product"`
}

type Correlation struct {
	ProfileVersion string `json:"profile_version"`
	ContractRef    *Ref   `json:"contract_ref,omitempty"`
	EventRef       *Ref   `json:"event_ref,omitempty"`
	CausalRef      *Ref   `json:"causal_ref,omitempty"`
	ContentDigest  string `json:"content_digest"`
	BindingMode    string `json:"binding_mode"`
}

type EvidenceBinding struct {
	ContractFamilyID string      `json:"contract_family_id"`
	Revision         int         `json:"revision"`
	ContractRef      Ref         `json:"contract_ref"`
	ActivationRef    Ref         `json:"activation_ref"`
	RuntimeActionRef Ref         `json:"runtime_action_ref"`
	ReadinessRef     Ref         `json:"readiness_ref"`
	DecisionRef      Ref         `json:"decision_ref"`
	PolicyRef        Ref         `json:"policy_ref"`
	TargetRef        Ref         `json:"target_ref"`
	EnvironmentRef   Ref         `json:"environment_ref"`
	ProofRefs        []Ref       `json:"proof_refs"`
	CausalRefs       []Ref       `json:"causal_refs"`
	Correlation      Correlation `json:"correlation"`
}

type Provenance struct {
	Producer  string              `json:"producer"`
	Writer    string              `json:"writer"`
	Verifier  string              `json:"verifier"`
	PublicKey string              `json:"public_key"`
	Signature proofsign.Signature `json:"signature"`
}

type ExecutionEvidence struct {
	SchemaID               string          `json:"schema_id"`
	SchemaVersion          string          `json:"schema_version"`
	EvidenceID             string          `json:"evidence_id"`
	Binding                EvidenceBinding `json:"binding"`
	EventRef               Ref             `json:"event_ref"`
	OccurredAt             string          `json:"occurred_at"`
	FreshUntil             string          `json:"fresh_until"`
	Outcome                string          `json:"outcome"`
	ReasonCode             string          `json:"reason_code"`
	CompensationRequired   bool            `json:"compensation_required"`
	Provenance             Provenance      `json:"provenance"`
	CanonicalContentDigest string          `json:"canonical_content_digest"`
}

type EffectEvent struct {
	SchemaID               string          `json:"schema_id"`
	SchemaVersion          string          `json:"schema_version"`
	EvidenceID             string          `json:"evidence_id"`
	Binding                EvidenceBinding `json:"binding"`
	EventRef               Ref             `json:"event_ref"`
	ExecutionRef           Ref             `json:"execution_ref"`
	EffectRef              Ref             `json:"effect_ref"`
	OccurredAt             string          `json:"occurred_at"`
	FreshUntil             string          `json:"fresh_until"`
	Outcome                string          `json:"outcome"`
	ReasonCode             string          `json:"reason_code"`
	Provenance             Provenance      `json:"provenance"`
	CanonicalContentDigest string          `json:"canonical_content_digest"`
}

type ContainmentEvidence struct {
	SchemaID               string          `json:"schema_id"`
	SchemaVersion          string          `json:"schema_version"`
	EvidenceID             string          `json:"evidence_id"`
	Binding                EvidenceBinding `json:"binding"`
	EventRef               Ref             `json:"event_ref"`
	ExecutionRef           Ref             `json:"execution_ref"`
	EffectRef              Ref             `json:"effect_ref"`
	ContainmentRef         Ref             `json:"containment_ref"`
	OccurredAt             string          `json:"occurred_at"`
	FreshUntil             string          `json:"fresh_until"`
	Outcome                string          `json:"outcome"`
	ReasonCode             string          `json:"reason_code"`
	Provenance             Provenance      `json:"provenance"`
	CanonicalContentDigest string          `json:"canonical_content_digest"`
}

type CompensationEvidence struct {
	SchemaID               string          `json:"schema_id"`
	SchemaVersion          string          `json:"schema_version"`
	EvidenceID             string          `json:"evidence_id"`
	Binding                EvidenceBinding `json:"binding"`
	EventRef               Ref             `json:"event_ref"`
	RequirementRef         Ref             `json:"requirement_ref"`
	ExecutionRef           Ref             `json:"execution_ref"`
	OccurredAt             string          `json:"occurred_at"`
	FreshUntil             string          `json:"fresh_until"`
	Outcome                string          `json:"outcome"`
	ReasonCode             string          `json:"reason_code"`
	Provenance             Provenance      `json:"provenance"`
	CanonicalContentDigest string          `json:"canonical_content_digest"`
}

type LifecycleRecord struct {
	SchemaID         string                `json:"schema_id"`
	SchemaVersion    string                `json:"schema_version"`
	RecordID         string                `json:"record_id"`
	Kind             string                `json:"kind"`
	OccurredAt       string                `json:"occurred_at"`
	ContractRef      Ref                   `json:"contract_ref"`
	ContractFamilyID string                `json:"contract_family_id,omitempty"`
	Revision         int                   `json:"revision"`
	ProposalRef      *Ref                  `json:"proposal_ref,omitempty"`
	ActivationRef    *Ref                  `json:"activation_ref,omitempty"`
	PreconditionRefs []Ref                 `json:"precondition_refs,omitempty"`
	Decision         json.RawMessage       `json:"decision,omitempty"`
	EvidenceRefs     []Ref                 `json:"evidence_refs,omitempty"`
	Execution        *ExecutionEvidence    `json:"execution,omitempty"`
	Effect           *EffectEvent          `json:"effect,omitempty"`
	Containment      *ContainmentEvidence  `json:"containment,omitempty"`
	Compensation     *CompensationEvidence `json:"compensation,omitempty"`
	ReasonCodes      []string              `json:"reason_codes,omitempty"`
	Correlation      Correlation           `json:"correlation"`
	ImmutableObject  json.RawMessage       `json:"immutable_object,omitempty"`
	Signature        proofsign.Signature   `json:"signature"`
}

type LifecyclePack struct {
	Records              []LifecycleRecord `json:"records"`
	SourceArtifactDigest string            `json:"-"`
}

type VerificationOptions struct {
	TrustedPublicKey        ed25519.PublicKey
	EvaluationTime          time.Time
	AllowFixtureOnly        bool
	ExpectedContract        Ref
	ExpectedFamily          string
	ExpectedRevision        int
	ExpectedActivation      Ref
	ExpectedRuntimeDigest   string
	ExpectedReadinessDigest string
	ExpectedPolicyDigest    string
	ExpectedTarget          string
	ExpectedEnvironment     string
	ExpectedProducerVersion string
	ExpectedSourceCommit    string
	ExpectedLifecycleDigest string
	ActivationNotBefore     time.Time
	ActivationNotAfter      time.Time
}

type Snapshot struct {
	ExecutionStatus    string `json:"execution_status,omitempty"`
	EffectStatus       string `json:"effect_status,omitempty"`
	ContainmentStatus  string `json:"containment_status,omitempty"`
	CompensationStatus string `json:"compensation_status,omitempty"`
	CurrentStatus      string `json:"current_status"`
}

type EvidenceSet struct {
	EvidenceSetID          string   `json:"evidence_set_id"`
	SourceProduct          string   `json:"source_product"`
	ProducerVersion        string   `json:"producer_version"`
	SourceCommit           string   `json:"source_commit"`
	Execution              string   `json:"gait_execution,omitempty"`
	Effect                 string   `json:"gait_effect,omitempty"`
	ContainmentStatus      string   `json:"containment_status,omitempty"`
	CompensationStatus     string   `json:"compensation_status,omitempty"`
	Verified               bool     `json:"verified"`
	Authoritative          bool     `json:"authoritative"`
	FixtureOnly            bool     `json:"fixture_only"`
	VerificationState      string   `json:"verification_state"`
	SourceArtifactDigests  []string `json:"source_artifact_digests"`
	DerivedEvidenceDigests []string `json:"derived_evidence_digests"`
	ReasonCodes            []string `json:"reason_codes,omitempty"`
}

type VerificationResult struct {
	Valid         bool        `json:"valid"`
	Authoritative bool        `json:"authoritative"`
	FixtureOnly   bool        `json:"fixture_only"`
	ReasonCodes   []string    `json:"reason_codes,omitempty"`
	Snapshot      Snapshot    `json:"snapshot"`
	EvidenceSet   EvidenceSet `json:"evidence_set"`
}
