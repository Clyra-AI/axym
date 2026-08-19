// Package actioncontract implements Axym's bounded consumer boundary for the
// currently released Wrkr proposal and Gait activation artifacts. It keeps
// producer-native bytes and fields intact; it never turns a recommendation
// into Axym or Gait authority.
package actioncontract

import (
	"crypto/ed25519"
	"time"

	proofsign "github.com/Clyra-AI/proof/signing"
)

const (
	ProposalSchemaID          = "https://wrkr.dev/schemas/v1/proposed-action-contract-artifact.schema.json"
	ProposalSchemaVersion     = "1"
	ProposalContractVersion   = "3"
	ProposalProducer          = "wrkr"
	ActivationSchemaID        = "https://gait.dev/schemas/v1/activated-action-contract-artifact.schema.json"
	ActivationSchemaVersion   = "1"
	ActivationContractVersion = "1"
	ReceiptSchemaID           = "https://axym.dev/schemas/v1/action-contract-consumer-receipt.schema.json"
	ReceiptSchemaVersion      = "1"
	ConsumerName              = "axym"
	ConsumerVersion           = "v0.6.1"
)

const (
	StatusPass         = "pass"
	StatusInvalid      = "invalid"
	StatusUnverifiable = "unverifiable"

	ClassExact              = "exact"
	ClassTightened          = "tightened"
	ClassContextual         = "contextual"
	ClassExplicitlyExcepted = "explicitly_excepted"
	ClassMismatched         = "mismatched"
	ClassWeakened           = "weakened"
	ClassIncomplete         = "incomplete"
	ClassUnverifiable       = "unverifiable"
)

type ProducerMetadata struct {
	Name                  string `json:"name"`
	ArtifactSchemaVersion string `json:"artifact_schema_version"`
	ContractSchemaVersion string `json:"contract_schema_version"`
}

type VariantMetadata struct {
	ShareProfile string `json:"share_profile"`
	Redacted     bool   `json:"redacted"`
}

// Proposal is a typed identity projection plus the untouched producer bytes
// and decoded contract map. Raw is never re-emitted as a rewritten proposal.
type Proposal struct {
	Raw                    []byte         `json:"-"`
	Data                   map[string]any `json:"-"`
	SchemaID               string
	SchemaVersion          string
	ArtifactID             string
	ContractID             string
	ContractFamilyID       string
	Revision               int
	Producer               ProducerMetadata
	SourceScanRefs         []string
	CompositionRefs        []string
	ResolutionKey          string
	CreationEvidence       []string
	CanonicalContentDigest string
	ReportOnly             bool
	Contract               map[string]any
	RawSHA256              string `json:"-"`
}

type Signature = proofsign.Signature

type ActivationProposalRef struct {
	ArtifactID             string `json:"artifact_id"`
	CanonicalContentDigest string `json:"canonical_content_digest"`
	ContractID             string `json:"contract_id"`
	ContractFamilyID       string `json:"contract_family_id"`
	Revision               int    `json:"revision"`
	SchemaID               string `json:"schema_id"`
	SchemaVersion          string `json:"schema_version"`
	ContractSchemaVersion  string `json:"contract_schema_version"`
}

type Validity struct {
	NotBefore string `json:"not_before"`
	NotAfter  string `json:"not_after,omitempty"`
}

type Activation struct {
	Raw                 []byte                `json:"-"`
	Data                map[string]any        `json:"-"`
	SchemaID            string                `json:"schema_id"`
	SchemaVersion       string                `json:"schema_version"`
	ArtifactID          string                `json:"artifact_id"`
	ContractID          string                `json:"contract_id"`
	ContractFamilyID    string                `json:"contract_family_id"`
	Revision            int                   `json:"revision"`
	Producer            ProducerMetadata      `json:"producer"`
	Proposal            ActivationProposalRef `json:"proposal"`
	PolicyDigest        string                `json:"policy_digest"`
	ActivatingPrincipal string                `json:"activating_principal"`
	AuthorityRefs       []string              `json:"authority_refs"`
	Target              string                `json:"target"`
	Environment         string                `json:"environment"`
	ActivationMode      string                `json:"activation_mode"`
	Validity            Validity              `json:"validity"`
	ExplicitExceptions  []string              `json:"explicit_exceptions"`
	ReportOnly          bool                  `json:"report_only"`
	DevelopmentSigning  bool                  `json:"development_signing"`
	Signature           Signature             `json:"signature"`
	RawSHA256           string                `json:"-"`
}

type ValidationOptions struct {
	Now                     time.Time
	PublicKey               ed25519.PublicKey
	Proposal                *Proposal
	Selection               *SelectionEvidence
	ExpectedRevision        int
	AllowDevelopmentSigning bool
}

// SelectionEvidence is the trusted current-selection assertion supplied by
// Gait or an explicit Axym integration boundary. It binds the activation to
// one exact proposal, family, revision, and byte digest.
type SelectionEvidence struct {
	ArtifactID             string `json:"artifact_id"`
	ArtifactSHA256         string `json:"artifact_sha256"`
	CanonicalContentDigest string `json:"canonical_content_digest"`
	ContractID             string `json:"contract_id"`
	ContractFamilyID       string `json:"contract_family_id"`
	Revision               int    `json:"revision"`
	Current                bool   `json:"current"`
	CandidateCount         int    `json:"candidate_count,omitempty"`
}

type ValidationResult struct {
	Valid                  bool     `json:"valid"`
	Status                 string   `json:"status"`
	ReasonCodes            []string `json:"reason_codes,omitempty"`
	CanonicalContentDigest string   `json:"canonical_content_digest,omitempty"`
	RawSHA256              string   `json:"artifact_sha256,omitempty"`
}

type ConformanceResult struct {
	Classification string   `json:"classification"`
	Valid          bool     `json:"valid"`
	NonBinding     bool     `json:"non_binding"`
	ReasonCodes    []string `json:"reason_codes,omitempty"`
	CheckedFields  []string `json:"checked_fields,omitempty"`
	MissingFields  []string `json:"missing_fields,omitempty"`
	Exceptions     []string `json:"explicit_exceptions,omitempty"`
}

// ActivationProjection is an optional Axym-owned observation of controls
// emitted by a future activation producer. The released Gait activation
// artifact does not duplicate proposal controls, so the default comparison
// relies on the exact proposal digest and never infers tightening from mode.
type ActivationProjection struct {
	AuthorityRefs           []string
	Preconditions           []string
	ConfirmationRequirement string
	ApprovalRequirement     string
	CredentialMode          string
	DelegationDepth         int
	ExpectedOutcomeClass    string
	ForbiddenEffects        []string
	CompensationRequired    *bool
	ValidityNotBefore       string
	ValidityNotAfter        string
}

type Receipt struct {
	Consumer           string            `json:"consumer"`
	Version            string            `json:"version"`
	ScenarioID         string            `json:"scenario_id"`
	ArtifactSHA256     string            `json:"artifact_sha256"`
	Status             string            `json:"status"`
	SelfAttestation    bool              `json:"self_attestation"`
	ProposalArtifactID string            `json:"proposal_artifact_id,omitempty"`
	ContractID         string            `json:"contract_id,omitempty"`
	ContractFamilyID   string            `json:"contract_family_id,omitempty"`
	Revision           int               `json:"revision,omitempty"`
	ResolutionKey      string            `json:"resolution_key,omitempty"`
	CorrelationRefs    []string          `json:"correlation_refs,omitempty"`
	SchemaVersions     map[string]string `json:"schema_versions"`
	SemanticResult     SemanticResult    `json:"semantic_result"`
}

type SemanticResult struct {
	ProposalValid   bool     `json:"proposal_valid"`
	ActivationReady bool     `json:"activation_ready"`
	Classification  string   `json:"classification"`
	ReasonCodes     []string `json:"reason_codes,omitempty"`
	ExecutionClaim  bool     `json:"execution_claim"`
	EffectClaim     bool     `json:"effect_claim"`
}
