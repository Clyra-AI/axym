package bundleartifacts

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/Clyra-AI/axym/core/identitygovernance"
	"github.com/Clyra-AI/axym/core/ingest/gait/translate"
	"github.com/Clyra-AI/proof"
)

const (
	VersionV1 = "v1"

	FileAuthorizationRegister    = "authorization-register.json"
	FileInsuranceEvidenceProfile = "insurance-evidence-profile.json"
	FileCredentialPosture        = "credential-posture-register.json" // #nosec G101 -- descriptive artifact filename, not a credential value.
	FileFreezeWindowCoverage     = "freeze-window-coverage.json"
	FileKillSwitchCoverage       = "kill-switch-coverage.json"
	FileEnforcementExplain       = "enforcement-explain-register.json"
	FileSandboxCoverage          = "sandbox-coverage.json"
	FileControlMaturity          = "control-maturity.json"

	StatusCovered = "covered"
	StatusPartial = "partial"
	StatusGap     = "gap"
)

type LinkedProofRef struct {
	RecordID   string `json:"record_id"`
	RecordHash string `json:"record_hash,omitempty"`
	RecordType string `json:"record_type,omitempty"`
}

type AuthorizationRegister struct {
	Version string                       `json:"version"`
	Entries []AuthorizationRegisterEntry `json:"entries"`
}

type AuthorizationRegisterEntry struct {
	BundleID               string                         `json:"bundle_id"`
	Status                 string                         `json:"status"`
	SourceProduct          string                         `json:"source_product"`
	SourceArtifactType     string                         `json:"source_artifact_type"`
	SourceArtifactID       string                         `json:"source_artifact_id"`
	SourceArtifactPath     string                         `json:"source_artifact_path"`
	SourceVersion          string                         `json:"source_version,omitempty"`
	Decision               string                         `json:"decision,omitempty"`
	TraceID                string                         `json:"trace_id,omitempty"`
	IntentDigest           string                         `json:"intent_digest,omitempty"`
	PolicyDigest           string                         `json:"policy_digest,omitempty"`
	Verification           translate.VerificationMetadata `json:"verification"`
	ApprovalAuditRefs      []string                       `json:"approval_audit_refs,omitempty"`
	CredentialEvidenceRefs []string                       `json:"credential_evidence_refs,omitempty"`
	ActionOutcomeRefs      []string                       `json:"action_outcome_refs,omitempty"`
	LinkedProofRefs        []LinkedProofRef               `json:"linked_proof_refs"`
	GapReasonCodes         []string                       `json:"gap_reason_codes,omitempty"`
}

type CredentialPostureRegister struct {
	Version string                   `json:"version"`
	Entries []CredentialPostureEntry `json:"entries"`
}

type CredentialPostureEntry struct {
	BundleID                  string           `json:"bundle_id"`
	ArtifactID                string           `json:"artifact_id"`
	Status                    string           `json:"status"`
	SourceProduct             string           `json:"source_product"`
	SourceArtifactPath        string           `json:"source_artifact_path"`
	StandingCredentialBlocked bool             `json:"standing_credential_blocked"`
	JITRequired               bool             `json:"jit_required"`
	BrokerSource              string           `json:"broker_source,omitempty"`
	Issuer                    string           `json:"issuer,omitempty"`
	TTLSeconds                int              `json:"ttl_seconds,omitempty"`
	Scope                     string           `json:"scope,omitempty"`
	BindingProofRef           string           `json:"binding_proof_ref,omitempty"`
	ActionRefs                []string         `json:"action_refs,omitempty"`
	ProofRefs                 []LinkedProofRef `json:"proof_refs"`
	GapReasonCodes            []string         `json:"gap_reason_codes,omitempty"`
}

type FreezeWindowCoverage struct {
	Version string                      `json:"version"`
	Entries []FreezeWindowCoverageEntry `json:"entries"`
}

type FreezeWindowCoverageEntry struct {
	BundleID           string           `json:"bundle_id"`
	ArtifactID         string           `json:"artifact_id"`
	Status             string           `json:"status"`
	Environment        string           `json:"environment,omitempty"`
	TargetKind         string           `json:"target_kind,omitempty"`
	TargetID           string           `json:"target_id,omitempty"`
	PolicyDigest       string           `json:"policy_digest,omitempty"`
	ApprovalRef        string           `json:"approval_ref,omitempty"`
	Reason             string           `json:"reason,omitempty"`
	Explain            string           `json:"explain,omitempty"`
	SourceProduct      string           `json:"source_product"`
	SourceArtifactPath string           `json:"source_artifact_path"`
	ProofRefs          []LinkedProofRef `json:"proof_refs"`
	GapReasonCodes     []string         `json:"gap_reason_codes,omitempty"`
}

type KillSwitchCoverage struct {
	Version string                    `json:"version"`
	Entries []KillSwitchCoverageEntry `json:"entries"`
}

type KillSwitchCoverageEntry struct {
	BundleID             string           `json:"bundle_id"`
	ArtifactID           string           `json:"artifact_id"`
	Status               string           `json:"status"`
	Environment          string           `json:"environment,omitempty"`
	TargetKind           string           `json:"target_kind,omitempty"`
	TargetID             string           `json:"target_id,omitempty"`
	PolicyDigest         string           `json:"policy_digest,omitempty"`
	BlockedDispatchProof string           `json:"blocked_dispatch_proof,omitempty"`
	Expiry               string           `json:"expiry,omitempty"`
	Actor                string           `json:"actor,omitempty"`
	SourceProduct        string           `json:"source_product"`
	SourceArtifactPath   string           `json:"source_artifact_path"`
	ProofRefs            []LinkedProofRef `json:"proof_refs"`
	GapReasonCodes       []string         `json:"gap_reason_codes,omitempty"`
}

type EnforcementExplainRegister struct {
	Version string                    `json:"version"`
	Entries []EnforcementExplainEntry `json:"entries"`
}

type EnforcementExplainEntry struct {
	BundleID           string           `json:"bundle_id"`
	ArtifactID         string           `json:"artifact_id"`
	Status             string           `json:"status"`
	Decision           string           `json:"decision,omitempty"`
	MissingFields      []string         `json:"missing_fields,omitempty"`
	CredentialPosture  string           `json:"credential_posture,omitempty"`
	FreezeState        string           `json:"freeze_state,omitempty"`
	KillSwitchState    string           `json:"kill_switch_state,omitempty"`
	SandboxState       string           `json:"sandbox_state,omitempty"`
	SourceProduct      string           `json:"source_product"`
	SourceArtifactPath string           `json:"source_artifact_path"`
	ProofRefs          []LinkedProofRef `json:"proof_refs"`
	ReasonCodes        []string         `json:"reason_codes,omitempty"`
}

type SandboxCoverage struct {
	Version string                 `json:"version"`
	Entries []SandboxCoverageEntry `json:"entries"`
}

type SandboxCoverageEntry struct {
	BundleID            string           `json:"bundle_id"`
	ArtifactID          string           `json:"artifact_id"`
	Status              string           `json:"status"`
	Path                string           `json:"path,omitempty"`
	NetworkMode         string           `json:"network_mode,omitempty"`
	WritablePaths       []string         `json:"writable_paths,omitempty"`
	EnvExposure         []string         `json:"env_exposure,omitempty"`
	TimeoutSeconds      int              `json:"timeout_seconds,omitempty"`
	FilesystemIsolation string           `json:"filesystem_isolation,omitempty"`
	PolicyResult        string           `json:"policy_result,omitempty"`
	SourceProduct       string           `json:"source_product"`
	SourceArtifactPath  string           `json:"source_artifact_path"`
	ProofRefs           []LinkedProofRef `json:"proof_refs"`
	GapReasonCodes      []string         `json:"gap_reason_codes,omitempty"`
}

type ControlMaturity struct {
	Version string                 `json:"version"`
	Entries []ControlMaturityEntry `json:"entries"`
}

type ControlMaturityEntry struct {
	Path                    string           `json:"path"`
	BundleID                string           `json:"bundle_id"`
	ArtifactID              string           `json:"artifact_id"`
	CurrentStage            string           `json:"current_stage,omitempty"`
	PriorStage              string           `json:"prior_stage,omitempty"`
	Drift                   string           `json:"drift"`
	SourceProduct           string           `json:"source_product"`
	SourceArtifactPath      string           `json:"source_artifact_path"`
	ProofRefs               []LinkedProofRef `json:"proof_refs"`
	RemainingGapReasonCodes []string         `json:"remaining_gap_reason_codes,omitempty"`
	ApprovalFatigueSignals  []string         `json:"approval_fatigue_signals,omitempty"`
}

type InsuranceEvidenceProfile struct {
	Version                 string                   `json:"version"`
	RequiredEvidenceClasses []string                 `json:"required_evidence_classes"`
	Entries                 []InsuranceEvidenceEntry `json:"entries"`
}

type InsuranceEvidenceEntry struct {
	Name               string   `json:"name"`
	Status             string   `json:"status"`
	SourceProducts     []string `json:"source_products"`
	SourceArtifactRefs []string `json:"source_artifact_refs,omitempty"`
	ProofRecordRefs    []string `json:"proof_record_refs,omitempty"`
	CompletenessScore  float64  `json:"completeness_score"`
	ReasonCodes        []string `json:"reason_codes,omitempty"`
	ActionableGaps     []string `json:"actionable_gaps,omitempty"`
}

type Outputs struct {
	AuthorizationRegister      *AuthorizationRegister
	InsuranceEvidenceProfile   *InsuranceEvidenceProfile
	CredentialPostureRegister  *CredentialPostureRegister
	FreezeWindowCoverage       *FreezeWindowCoverage
	KillSwitchCoverage         *KillSwitchCoverage
	EnforcementExplainRegister *EnforcementExplainRegister
	SandboxCoverage            *SandboxCoverage
	ControlMaturity            *ControlMaturity
}

type sourceArtifactRecord struct {
	Artifact translate.SourceArtifact
	Record   proof.Record
}

func Build(records []proof.Record) Outputs {
	sources := extractSourceArtifactRecords(records)
	authRegister := buildAuthorizationRegister(sources, records)
	credentialRegister := buildCredentialPostureRegister(sources, records)
	freezeCoverage := buildFreezeWindowCoverage(sources, records)
	killCoverage := buildKillSwitchCoverage(sources, records)
	explainRegister := buildExplainRegister(sources, records)
	sandboxCoverage := buildSandboxCoverage(sources, records)
	controlMaturity := buildControlMaturity(sources, records)
	insuranceProfile := buildInsuranceEvidenceProfile(records, authRegister, credentialRegister, freezeCoverage, killCoverage, explainRegister, sandboxCoverage, controlMaturity)

	return Outputs{
		AuthorizationRegister:      authRegister,
		InsuranceEvidenceProfile:   insuranceProfile,
		CredentialPostureRegister:  credentialRegister,
		FreezeWindowCoverage:       freezeCoverage,
		KillSwitchCoverage:         killCoverage,
		EnforcementExplainRegister: explainRegister,
		SandboxCoverage:            sandboxCoverage,
		ControlMaturity:            controlMaturity,
	}
}

func (o Outputs) Files() (map[string][]byte, error) {
	out := map[string][]byte{}
	if o.AuthorizationRegister != nil {
		raw, err := json.MarshalIndent(o.AuthorizationRegister, "", "  ")
		if err != nil {
			return nil, err
		}
		out[FileAuthorizationRegister] = raw
	}
	if o.InsuranceEvidenceProfile != nil {
		raw, err := json.MarshalIndent(o.InsuranceEvidenceProfile, "", "  ")
		if err != nil {
			return nil, err
		}
		out[FileInsuranceEvidenceProfile] = raw
	}
	if o.CredentialPostureRegister != nil {
		raw, err := json.MarshalIndent(o.CredentialPostureRegister, "", "  ")
		if err != nil {
			return nil, err
		}
		out[FileCredentialPosture] = raw
	}
	if o.FreezeWindowCoverage != nil {
		raw, err := json.MarshalIndent(o.FreezeWindowCoverage, "", "  ")
		if err != nil {
			return nil, err
		}
		out[FileFreezeWindowCoverage] = raw
	}
	if o.KillSwitchCoverage != nil {
		raw, err := json.MarshalIndent(o.KillSwitchCoverage, "", "  ")
		if err != nil {
			return nil, err
		}
		out[FileKillSwitchCoverage] = raw
	}
	if o.EnforcementExplainRegister != nil {
		raw, err := json.MarshalIndent(o.EnforcementExplainRegister, "", "  ")
		if err != nil {
			return nil, err
		}
		out[FileEnforcementExplain] = raw
	}
	if o.SandboxCoverage != nil {
		raw, err := json.MarshalIndent(o.SandboxCoverage, "", "  ")
		if err != nil {
			return nil, err
		}
		out[FileSandboxCoverage] = raw
	}
	if o.ControlMaturity != nil {
		raw, err := json.MarshalIndent(o.ControlMaturity, "", "  ")
		if err != nil {
			return nil, err
		}
		out[FileControlMaturity] = raw
	}
	return out, nil
}

func extractSourceArtifactRecords(records []proof.Record) []sourceArtifactRecord {
	out := make([]sourceArtifactRecord, 0)
	for _, record := range sortedRecords(records) {
		artifact, ok := translate.ExtractSourceArtifact(record)
		if !ok {
			continue
		}
		out = append(out, sourceArtifactRecord{Artifact: artifact, Record: record})
	}
	return out
}

func buildAuthorizationRegister(sources []sourceArtifactRecord, records []proof.Record) *AuthorizationRegister {
	entries := make([]AuthorizationRegisterEntry, 0)
	for _, item := range sources {
		if item.Artifact.Kind() != translate.ArtifactTypeAuthorizationBundle && item.Artifact.Kind() != translate.ArtifactTypeAuthorizationProfile {
			continue
		}
		base := item.Artifact.Base()
		linkedRefs, gapReasons := linkedProofRefs(item, sources, records)
		status := "verified"
		switch {
		case !base.Verification.SchemaValid:
			status = "schema_invalid"
		case base.Verification.SignatureValid != nil && !*base.Verification.SignatureValid:
			status = "signature_failed"
		case containsReason(gapReasons, "missing_proof_record_ref"):
			status = "missing_link"
		case len(gapReasons) > 0:
			status = "partial"
		}
		version := firstNonEmpty(base.SchemaVersion, base.ProfileVersion)
		entries = append(entries, AuthorizationRegisterEntry{
			BundleID:               base.BundleID,
			Status:                 status,
			SourceProduct:          base.SourceProduct,
			SourceArtifactType:     base.Kind,
			SourceArtifactID:       base.ArtifactID,
			SourceArtifactPath:     base.SourceArtifactPath,
			SourceVersion:          version,
			Decision:               base.Decision,
			TraceID:                base.TraceID,
			IntentDigest:           base.IntentDigest,
			PolicyDigest:           base.PolicyDigest,
			Verification:           base.Verification,
			ApprovalAuditRefs:      append([]string(nil), base.Links.ApprovalAuditRefs...),
			CredentialEvidenceRefs: append([]string(nil), base.Links.CredentialEvidenceRefs...),
			ActionOutcomeRefs:      append([]string(nil), base.Links.ActionOutcomeRefs...),
			LinkedProofRefs:        linkedRefs,
			GapReasonCodes:         gapReasons,
		})
	}
	if len(entries) == 0 {
		return nil
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].BundleID != entries[j].BundleID {
			return entries[i].BundleID < entries[j].BundleID
		}
		return entries[i].SourceArtifactPath < entries[j].SourceArtifactPath
	})
	return &AuthorizationRegister{Version: VersionV1, Entries: entries}
}

func buildCredentialPostureRegister(sources []sourceArtifactRecord, records []proof.Record) *CredentialPostureRegister {
	entries := make([]CredentialPostureEntry, 0)
	for _, item := range sources {
		if item.Artifact.CredentialPosture == nil {
			continue
		}
		artifact := item.Artifact.CredentialPosture
		linkedRefs, gapReasons := linkedProofRefs(item, sources, records)
		status := credentialStatus(artifact, gapReasons)
		entries = append(entries, CredentialPostureEntry{
			BundleID:                  artifact.BundleID,
			ArtifactID:                artifact.ArtifactID,
			Status:                    status,
			SourceProduct:             artifact.SourceProduct,
			SourceArtifactPath:        artifact.SourceArtifactPath,
			StandingCredentialBlocked: artifact.StandingCredentialBlocked,
			JITRequired:               artifact.JITRequired,
			BrokerSource:              artifact.BrokerSource,
			Issuer:                    artifact.Issuer,
			TTLSeconds:                artifact.TTLSeconds,
			Scope:                     artifact.Scope,
			BindingProofRef:           artifact.BindingProofRef,
			ActionRefs:                append([]string(nil), artifact.ActionRefs...),
			ProofRefs:                 linkedRefs,
			GapReasonCodes:            gapReasons,
		})
	}
	if len(entries) == 0 {
		return nil
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].BundleID != entries[j].BundleID {
			return entries[i].BundleID < entries[j].BundleID
		}
		return entries[i].ArtifactID < entries[j].ArtifactID
	})
	return &CredentialPostureRegister{Version: VersionV1, Entries: entries}
}

func buildFreezeWindowCoverage(sources []sourceArtifactRecord, records []proof.Record) *FreezeWindowCoverage {
	entries := make([]FreezeWindowCoverageEntry, 0)
	for _, item := range sources {
		if item.Artifact.FreezeWindow == nil {
			continue
		}
		artifact := item.Artifact.FreezeWindow
		linkedRefs, gapReasons := linkedProofRefs(item, sources, records)
		status := freezeStatus(artifact)
		if status == "missing" {
			gapReasons = uniqueStrings(append(gapReasons, "missing_freeze_state"))
		}
		entries = append(entries, FreezeWindowCoverageEntry{
			BundleID:           artifact.BundleID,
			ArtifactID:         artifact.ArtifactID,
			Status:             status,
			Environment:        artifact.Environment,
			TargetKind:         artifact.TargetKind,
			TargetID:           artifact.TargetID,
			PolicyDigest:       artifact.PolicyDigest,
			ApprovalRef:        artifact.ApprovalRef,
			Reason:             artifact.Reason,
			Explain:            artifact.Explain,
			SourceProduct:      artifact.SourceProduct,
			SourceArtifactPath: artifact.SourceArtifactPath,
			ProofRefs:          linkedRefs,
			GapReasonCodes:     gapReasons,
		})
	}
	if len(entries) == 0 {
		return nil
	}
	sort.Slice(entries, func(i, j int) bool {
		left := entries[i]
		right := entries[j]
		if left.Environment != right.Environment {
			return left.Environment < right.Environment
		}
		if left.TargetKind != right.TargetKind {
			return left.TargetKind < right.TargetKind
		}
		if left.TargetID != right.TargetID {
			return left.TargetID < right.TargetID
		}
		if left.PolicyDigest != right.PolicyDigest {
			return left.PolicyDigest < right.PolicyDigest
		}
		return left.BundleID < right.BundleID
	})
	return &FreezeWindowCoverage{Version: VersionV1, Entries: entries}
}

func buildKillSwitchCoverage(sources []sourceArtifactRecord, records []proof.Record) *KillSwitchCoverage {
	entries := make([]KillSwitchCoverageEntry, 0)
	for _, item := range sources {
		if item.Artifact.KillSwitch == nil {
			continue
		}
		artifact := item.Artifact.KillSwitch
		linkedRefs, gapReasons := linkedProofRefs(item, sources, records)
		status := killSwitchStatus(artifact)
		if status == "missing" {
			gapReasons = uniqueStrings(append(gapReasons, "missing_kill_switch_state"))
		}
		entries = append(entries, KillSwitchCoverageEntry{
			BundleID:             artifact.BundleID,
			ArtifactID:           artifact.ArtifactID,
			Status:               status,
			Environment:          artifact.Environment,
			TargetKind:           artifact.TargetKind,
			TargetID:             artifact.TargetID,
			PolicyDigest:         artifact.PolicyDigest,
			BlockedDispatchProof: artifact.BlockedDispatchProof,
			Expiry:               artifact.Expiry,
			Actor:                artifact.Actor,
			SourceProduct:        artifact.SourceProduct,
			SourceArtifactPath:   artifact.SourceArtifactPath,
			ProofRefs:            linkedRefs,
			GapReasonCodes:       gapReasons,
		})
	}
	if len(entries) == 0 {
		return nil
	}
	sort.Slice(entries, func(i, j int) bool {
		left := entries[i]
		right := entries[j]
		if left.Environment != right.Environment {
			return left.Environment < right.Environment
		}
		if left.TargetKind != right.TargetKind {
			return left.TargetKind < right.TargetKind
		}
		if left.TargetID != right.TargetID {
			return left.TargetID < right.TargetID
		}
		if left.PolicyDigest != right.PolicyDigest {
			return left.PolicyDigest < right.PolicyDigest
		}
		return left.BundleID < right.BundleID
	})
	return &KillSwitchCoverage{Version: VersionV1, Entries: entries}
}

func buildExplainRegister(sources []sourceArtifactRecord, records []proof.Record) *EnforcementExplainRegister {
	entries := make([]EnforcementExplainEntry, 0)
	for _, item := range sources {
		if item.Artifact.EnforcementExplain == nil {
			continue
		}
		artifact := item.Artifact.EnforcementExplain
		linkedRefs, gapReasons := linkedProofRefs(item, sources, records)
		status := "verified"
		if len(artifact.MissingFields) > 0 || len(gapReasons) > 0 {
			status = "partial"
		}
		entries = append(entries, EnforcementExplainEntry{
			BundleID:           artifact.BundleID,
			ArtifactID:         artifact.ArtifactID,
			Status:             status,
			Decision:           artifact.Decision,
			MissingFields:      append([]string(nil), artifact.MissingFields...),
			CredentialPosture:  artifact.CredentialPosture,
			FreezeState:        artifact.FreezeState,
			KillSwitchState:    artifact.KillSwitchState,
			SandboxState:       artifact.SandboxState,
			SourceProduct:      artifact.SourceProduct,
			SourceArtifactPath: artifact.SourceArtifactPath,
			ProofRefs:          linkedRefs,
			ReasonCodes:        uniqueStrings(append(append([]string(nil), artifact.ReasonCodes...), gapReasons...)),
		})
	}
	if len(entries) == 0 {
		return nil
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].BundleID != entries[j].BundleID {
			return entries[i].BundleID < entries[j].BundleID
		}
		return entries[i].ArtifactID < entries[j].ArtifactID
	})
	return &EnforcementExplainRegister{Version: VersionV1, Entries: entries}
}

func buildSandboxCoverage(sources []sourceArtifactRecord, records []proof.Record) *SandboxCoverage {
	entries := make([]SandboxCoverageEntry, 0)
	for _, item := range sources {
		if item.Artifact.SandboxPolicy == nil {
			continue
		}
		artifact := item.Artifact.SandboxPolicy
		linkedRefs, gapReasons := linkedProofRefs(item, sources, records)
		gapReasons = uniqueStrings(append(gapReasons, artifact.GapReasonCodes...))
		status := sandboxStatus(artifact, gapReasons)
		entries = append(entries, SandboxCoverageEntry{
			BundleID:            artifact.BundleID,
			ArtifactID:          artifact.ArtifactID,
			Status:              status,
			Path:                artifact.Path,
			NetworkMode:         artifact.NetworkMode,
			WritablePaths:       append([]string(nil), artifact.WritablePaths...),
			EnvExposure:         append([]string(nil), artifact.EnvExposure...),
			TimeoutSeconds:      artifact.TimeoutSeconds,
			FilesystemIsolation: artifact.FilesystemIsolation,
			PolicyResult:        artifact.PolicyResult,
			SourceProduct:       artifact.SourceProduct,
			SourceArtifactPath:  artifact.SourceArtifactPath,
			ProofRefs:           linkedRefs,
			GapReasonCodes:      gapReasons,
		})
	}
	if len(entries) == 0 {
		return nil
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Path != entries[j].Path {
			return entries[i].Path < entries[j].Path
		}
		if entries[i].BundleID != entries[j].BundleID {
			return entries[i].BundleID < entries[j].BundleID
		}
		return entries[i].ArtifactID < entries[j].ArtifactID
	})
	return &SandboxCoverage{Version: VersionV1, Entries: entries}
}

func buildControlMaturity(sources []sourceArtifactRecord, records []proof.Record) *ControlMaturity {
	entries := make([]ControlMaturityEntry, 0)
	for _, item := range sources {
		if item.Artifact.TrustGraduation == nil {
			continue
		}
		artifact := item.Artifact.TrustGraduation
		linkedRefs, gapReasons := linkedProofRefs(item, sources, records)
		entries = append(entries, ControlMaturityEntry{
			Path:                    firstNonEmpty(artifact.Path, artifact.TargetID),
			BundleID:                artifact.BundleID,
			ArtifactID:              artifact.ArtifactID,
			CurrentStage:            artifact.CurrentStage,
			PriorStage:              artifact.PriorStage,
			Drift:                   maturityDrift(artifact.CurrentStage, artifact.PriorStage),
			SourceProduct:           artifact.SourceProduct,
			SourceArtifactPath:      artifact.SourceArtifactPath,
			ProofRefs:               linkedRefs,
			RemainingGapReasonCodes: uniqueStrings(append([]string(nil), append(artifact.RemainingGapReasonCodes, gapReasons...)...)),
			ApprovalFatigueSignals:  append([]string(nil), artifact.ApprovalFatigueSignals...),
		})
	}
	if len(entries) == 0 {
		return nil
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Path != entries[j].Path {
			return entries[i].Path < entries[j].Path
		}
		if entries[i].BundleID != entries[j].BundleID {
			return entries[i].BundleID < entries[j].BundleID
		}
		return entries[i].ArtifactID < entries[j].ArtifactID
	})
	return &ControlMaturity{Version: VersionV1, Entries: entries}
}

func buildInsuranceEvidenceProfile(
	records []proof.Record,
	authRegister *AuthorizationRegister,
	credentialRegister *CredentialPostureRegister,
	freezeCoverage *FreezeWindowCoverage,
	killCoverage *KillSwitchCoverage,
	explainRegister *EnforcementExplainRegister,
	sandboxCoverage *SandboxCoverage,
	controlMaturity *ControlMaturity,
) *InsuranceEvidenceProfile {
	if authRegister == nil && credentialRegister == nil && freezeCoverage == nil && killCoverage == nil && explainRegister == nil && sandboxCoverage == nil && controlMaturity == nil && !hasWrkrEvidence(records) {
		return nil
	}
	identityArtifacts := identitygovernance.Build(records)
	required := []string{
		"inventory",
		"ownership",
		"scoped_authority",
		"approval_gates",
		"kill_switch",
		"drift_tracking",
		"incident_reconstruction",
		"jit_credentials",
		"freeze_windows",
		"sandboxing",
		"proof_of_enforcement",
	}
	entries := []InsuranceEvidenceEntry{
		buildInventoryEntry(records),
		buildOwnershipEntry(records, identityArtifacts),
		buildScopedAuthorityEntry(authRegister),
		buildApprovalGatesEntry(authRegister),
		buildKillSwitchEntry(killCoverage),
		buildDriftTrackingEntry(records, identityArtifacts),
		buildIncidentReconstructionEntry(authRegister, explainRegister),
		buildJITCredentialsEntry(credentialRegister),
		buildFreezeWindowsEntry(freezeCoverage),
		buildSandboxingEntry(sandboxCoverage),
		buildProofOfEnforcementEntry(authRegister),
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})
	return &InsuranceEvidenceProfile{
		Version:                 VersionV1,
		RequiredEvidenceClasses: required,
		Entries:                 entries,
	}
}

func linkedProofRefs(item sourceArtifactRecord, sources []sourceArtifactRecord, records []proof.Record) ([]LinkedProofRef, []string) {
	base := item.Artifact.Base()
	seen := map[string]struct{}{}
	refs := make([]LinkedProofRef, 0)
	addRef := func(record proof.Record) {
		id := strings.TrimSpace(record.RecordID)
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		refs = append(refs, LinkedProofRef{
			RecordID:   id,
			RecordHash: strings.TrimSpace(record.Integrity.RecordHash),
			RecordType: strings.TrimSpace(record.RecordType),
		})
	}

	addRef(item.Record)
	for _, other := range sources {
		if other.Artifact.BundleID() == base.BundleID && base.BundleID != "" {
			addRef(other.Record)
		}
	}

	missing := make([]string, 0)
	for _, wantID := range base.Links.ProofRecordIDs {
		found := false
		for _, record := range records {
			if strings.TrimSpace(record.RecordID) == wantID {
				addRef(record)
				found = true
			}
		}
		if !found {
			missing = append(missing, "missing_proof_record_ref")
		}
	}
	if len(base.Links.DecisionTraceRefs) > 0 {
		traceMatched := false
		for _, record := range records {
			if matchesAnyTrace(record, append([]string{base.TraceID}, base.Links.DecisionTraceRefs...)...) {
				addRef(record)
				traceMatched = true
			}
		}
		if !traceMatched {
			missing = append(missing, "missing_trace_ref")
		}
	}
	if len(base.Links.ApprovalAuditRefs) > 0 {
		approvalMatched := false
		for _, record := range records {
			if matchesAnyApproval(record, base.Links.ApprovalAuditRefs...) {
				addRef(record)
				approvalMatched = true
			}
		}
		if !approvalMatched {
			missing = append(missing, "missing_approval_ref")
		}
	}
	if len(base.Links.CredentialEvidenceRefs) > 0 {
		credentialMatched := false
		for _, record := range records {
			if matchesAnyValue(record, "artifact_id", "binding_proof_ref", base.Links.CredentialEvidenceRefs...) {
				addRef(record)
				credentialMatched = true
			}
		}
		if !credentialMatched {
			missing = append(missing, "missing_credential_ref")
		}
	}
	if len(base.Links.ActionOutcomeRefs) > 0 {
		actionMatched := false
		for _, record := range records {
			if matchesActionOutcome(record, base.Links.ActionOutcomeRefs...) {
				addRef(record)
				actionMatched = true
			}
		}
		if !actionMatched {
			missing = append(missing, "missing_action_outcome_ref")
		}
	}

	sort.Slice(refs, func(i, j int) bool {
		return refs[i].RecordID < refs[j].RecordID
	})
	return refs, uniqueStrings(missing)
}

func matchesAnyTrace(record proof.Record, values ...string) bool {
	if matchesValues(values, stringFromEvent(record, "trace_id")) {
		return true
	}
	return matchesListValues(values, stringSliceFromMetadata(record, "gait_decision_trace_refs"))
}

func matchesAnyApproval(record proof.Record, values ...string) bool {
	if matchesValues(values,
		stringFromEvent(record, "approval_token_ref"),
		stringFromEvent(record, "approval_ref"),
		stringFromMetadata(record, "approval_token_ref"),
	) {
		return true
	}
	return matchesListValues(values, stringSliceFromMetadata(record, "gait_approval_audit_refs"))
}

func matchesActionOutcome(record proof.Record, values ...string) bool {
	if matchesAnyValue(record, "action_outcome_ref", "", values...) {
		return true
	}
	return matchesListValues(values, stringSliceFromMetadata(record, "gait_action_refs"))
}

func matchesAnyValue(record proof.Record, eventKey string, metadataKey string, values ...string) bool {
	want := uniqueStrings(values)
	if len(want) == 0 {
		return false
	}
	candidates := []string{}
	if eventKey != "" {
		candidates = append(candidates, stringFromEvent(record, eventKey))
	}
	if metadataKey != "" {
		candidates = append(candidates, stringFromMetadata(record, metadataKey))
	}
	for _, candidate := range candidates {
		if containsString(want, candidate) {
			return true
		}
	}
	return false
}

func matchesListValues(values []string, lists ...[]string) bool {
	want := uniqueStrings(values)
	if len(want) == 0 {
		return false
	}
	for _, listed := range lists {
		for _, candidate := range listed {
			if containsString(want, candidate) {
				return true
			}
		}
	}
	return false
}

func matchesValues(values []string, candidates ...string) bool {
	want := uniqueStrings(values)
	if len(want) == 0 {
		return false
	}
	for _, candidate := range candidates {
		if containsString(want, candidate) {
			return true
		}
	}
	return false
}

func credentialStatus(artifact *translate.CredentialPosture, gapReasons []string) string {
	switch {
	case strings.TrimSpace(artifact.BindingProofRef) == "":
		return "missing_binding"
	case artifact.TTLSeconds <= 0:
		return "ttl_missing"
	case strings.TrimSpace(artifact.Scope) == "":
		return "scope_missing"
	case strings.TrimSpace(artifact.BrokerSource) == "":
		return "broker_missing"
	case strings.TrimSpace(artifact.Issuer) == "":
		return "issuer_missing"
	case artifact.JITRequired && artifact.StandingCredentialBlocked:
		return "jit_verified"
	case artifact.StandingCredentialBlocked:
		return "standing_blocked"
	case len(gapReasons) > 0:
		return "missing_binding"
	default:
		return "standing_blocked"
	}
}

func freezeStatus(artifact *translate.FreezeWindow) string {
	state := strings.ToLower(strings.TrimSpace(artifact.State))
	switch {
	case artifact.Stale || state == "stale":
		return "stale"
	case state == "enforced":
		return "enforced"
	case state == "bypassed" || state == "bypassed_by_approval" || strings.TrimSpace(artifact.ApprovalRef) != "":
		return "bypassed_by_approval"
	case state == "evaluated" || state == "allow" || state == "blocked":
		return "evaluated"
	default:
		return "missing"
	}
}

func killSwitchStatus(artifact *translate.KillSwitch) string {
	state := strings.ToLower(strings.TrimSpace(artifact.State))
	switch {
	case state == "stale":
		return "stale"
	case state == "expired":
		return "expired"
	case strings.TrimSpace(artifact.BlockedDispatchProof) != "":
		return "blocked_dispatch"
	case state == "active":
		return "active"
	default:
		return "missing"
	}
}

func sandboxStatus(artifact *translate.SandboxPolicy, gapReasons []string) string {
	switch {
	case len(gapReasons) > 0:
		return StatusPartial
	case strings.TrimSpace(artifact.NetworkMode) == "" || strings.TrimSpace(artifact.FilesystemIsolation) == "" || strings.TrimSpace(artifact.PolicyResult) == "":
		return StatusGap
	default:
		return StatusCovered
	}
}

func maturityDrift(currentStage string, priorStage string) string {
	current := stageRank(currentStage)
	prior := stageRank(priorStage)
	switch {
	case strings.TrimSpace(priorStage) == "":
		return "introduced"
	case current < prior:
		return "regressed"
	case current > prior:
		return "advanced"
	default:
		return "unchanged"
	}
}

func stageRank(stage string) int {
	switch strings.TrimSpace(stage) {
	case "observe":
		return 0
	case "dry_run":
		return 1
	case "read_only_allow":
		return 2
	case "approval_gated_write":
		return 3
	case "brokered_write":
		return 4
	case "blocked_destructive":
		return 5
	default:
		return -1
	}
}

func buildInventoryEntry(records []proof.Record) InsuranceEvidenceEntry {
	if !hasWrkrEvidence(records) {
		return insuranceEntry("inventory", StatusGap, nil, nil, 0, []string{"missing_wrkr_inventory"}, []string{"ingest Wrkr inventory evidence"})
	}
	return insuranceEntry("inventory", StatusCovered, []string{"wrkr"}, wrkrRecordIDs(records), 1.0, nil, nil)
}

func buildOwnershipEntry(records []proof.Record, identityArtifacts identitygovernance.Artifacts) InsuranceEvidenceEntry {
	if len(identityArtifacts.OwnershipRegister.Entries) == 0 {
		return insuranceEntry("ownership", StatusGap, []string{"wrkr"}, nil, 0, []string{"missing_owner_identity"}, []string{"capture owner or approver linkage"})
	}
	proofRefs := make([]string, 0)
	for _, entry := range identityArtifacts.OwnershipRegister.Entries {
		proofRefs = append(proofRefs, entry.RecordIDs...)
	}
	return insuranceEntry("ownership", StatusCovered, []string{"wrkr", "axym"}, uniqueStrings(proofRefs), 1.0, nil, nil)
}

func buildScopedAuthorityEntry(register *AuthorizationRegister) InsuranceEvidenceEntry {
	if register == nil || len(register.Entries) == 0 {
		return insuranceEntry("scoped_authority", StatusGap, []string{"gait"}, nil, 0, []string{"missing_authorization_bundle"}, []string{"ingest Gait authorization bundle"})
	}
	reasonCodes := []string{}
	status := StatusCovered
	sourceRefs := make([]string, 0, len(register.Entries))
	proofRefs := make([]string, 0)
	for _, entry := range register.Entries {
		sourceRefs = append(sourceRefs, entry.SourceArtifactID)
		if entry.PolicyDigest == "" {
			status = StatusPartial
			reasonCodes = append(reasonCodes, "missing_policy_digest")
		}
		if entry.Status != "verified" {
			status = StatusPartial
			reasonCodes = append(reasonCodes, "authorization_linkage_partial")
		}
		for _, ref := range entry.LinkedProofRefs {
			proofRefs = append(proofRefs, ref.RecordID)
		}
	}
	score := scoreForStatus(status)
	return insuranceEntry("scoped_authority", status, []string{"gait"}, uniqueStrings(sourceRefs), score, uniqueStrings(reasonCodes), nil).withProofRefs(uniqueStrings(proofRefs))
}

func buildApprovalGatesEntry(register *AuthorizationRegister) InsuranceEvidenceEntry {
	if register == nil || len(register.Entries) == 0 {
		return insuranceEntry("approval_gates", StatusGap, []string{"gait"}, nil, 0, []string{"missing_authorization_bundle"}, []string{"ingest Gait approval evidence"})
	}
	status := StatusCovered
	reasonCodes := []string{}
	sourceRefs := make([]string, 0, len(register.Entries))
	proofRefs := make([]string, 0)
	for _, entry := range register.Entries {
		sourceRefs = append(sourceRefs, entry.SourceArtifactID)
		if len(entry.ApprovalAuditRefs) == 0 {
			status = StatusPartial
			reasonCodes = append(reasonCodes, "missing_approval_ref")
		}
		for _, ref := range entry.LinkedProofRefs {
			proofRefs = append(proofRefs, ref.RecordID)
		}
	}
	return insuranceEntry("approval_gates", status, []string{"gait"}, uniqueStrings(sourceRefs), scoreForStatus(status), uniqueStrings(reasonCodes), nil).withProofRefs(uniqueStrings(proofRefs))
}

func buildKillSwitchEntry(coverage *KillSwitchCoverage) InsuranceEvidenceEntry {
	if coverage == nil || len(coverage.Entries) == 0 {
		return insuranceEntry("kill_switch", StatusGap, []string{"gait"}, nil, 0, []string{"missing_kill_switch_state"}, []string{"ingest Gait kill-switch evidence"})
	}
	status := StatusCovered
	reasonCodes := []string{}
	sourceRefs := make([]string, 0)
	proofRefs := make([]string, 0)
	for _, entry := range coverage.Entries {
		sourceRefs = append(sourceRefs, entry.ArtifactID)
		if entry.Status == "stale" || entry.Status == "expired" || entry.Status == "missing" {
			status = StatusPartial
			reasonCodes = append(reasonCodes, "kill_switch_not_current")
		}
		for _, ref := range entry.ProofRefs {
			proofRefs = append(proofRefs, ref.RecordID)
		}
	}
	return insuranceEntry("kill_switch", status, []string{"gait"}, uniqueStrings(sourceRefs), scoreForStatus(status), uniqueStrings(reasonCodes), nil).withProofRefs(uniqueStrings(proofRefs))
}

func buildDriftTrackingEntry(records []proof.Record, identityArtifacts identitygovernance.Artifacts) InsuranceEvidenceEntry {
	if !hasWrkrEvidence(records) {
		return insuranceEntry("drift_tracking", StatusGap, []string{"wrkr"}, nil, 0, []string{"missing_wrkr_posture"}, []string{"ingest Wrkr posture evidence"})
	}
	proofRefs := make([]string, 0)
	for _, finding := range identityArtifacts.PrivilegeDriftReport.Findings {
		proofRefs = append(proofRefs, finding.RecordID)
	}
	status := StatusCovered
	if len(proofRefs) == 0 {
		status = StatusPartial
	}
	return insuranceEntry("drift_tracking", status, []string{"wrkr", "axym"}, uniqueStrings(proofRefs), scoreForStatus(status), nil, nil)
}

func buildIncidentReconstructionEntry(register *AuthorizationRegister, explain *EnforcementExplainRegister) InsuranceEvidenceEntry {
	if register == nil || explain == nil {
		return insuranceEntry("incident_reconstruction", StatusGap, []string{"gait", "proof"}, nil, 0, []string{"missing_traceable_runtime_evidence"}, []string{"ingest authorization bundles and explain output"})
	}
	proofRefs := make([]string, 0)
	for _, entry := range register.Entries {
		for _, ref := range entry.LinkedProofRefs {
			proofRefs = append(proofRefs, ref.RecordID)
		}
	}
	return insuranceEntry("incident_reconstruction", StatusCovered, []string{"gait", "proof"}, uniqueStrings(proofRefs), 1.0, nil, nil)
}

func buildJITCredentialsEntry(register *CredentialPostureRegister) InsuranceEvidenceEntry {
	if register == nil || len(register.Entries) == 0 {
		return insuranceEntry("jit_credentials", StatusGap, []string{"gait"}, nil, 0, []string{"missing_jit_evidence"}, []string{"ingest Gait credential posture evidence"})
	}
	status := StatusCovered
	reasonCodes := []string{}
	sourceRefs := make([]string, 0)
	proofRefs := make([]string, 0)
	for _, entry := range register.Entries {
		sourceRefs = append(sourceRefs, entry.ArtifactID)
		if entry.Status != "jit_verified" && entry.Status != "standing_blocked" {
			status = StatusPartial
			reasonCodes = append(reasonCodes, entry.Status)
		}
		for _, ref := range entry.ProofRefs {
			proofRefs = append(proofRefs, ref.RecordID)
		}
	}
	return insuranceEntry("jit_credentials", status, []string{"gait"}, uniqueStrings(sourceRefs), scoreForStatus(status), uniqueStrings(reasonCodes), nil).withProofRefs(uniqueStrings(proofRefs))
}

func buildFreezeWindowsEntry(coverage *FreezeWindowCoverage) InsuranceEvidenceEntry {
	if coverage == nil || len(coverage.Entries) == 0 {
		return insuranceEntry("freeze_windows", StatusGap, []string{"gait"}, nil, 0, []string{"missing_freeze_window_evidence"}, []string{"ingest Gait freeze-window evidence"})
	}
	status := StatusCovered
	reasonCodes := []string{}
	sourceRefs := make([]string, 0)
	proofRefs := make([]string, 0)
	for _, entry := range coverage.Entries {
		sourceRefs = append(sourceRefs, entry.ArtifactID)
		if entry.Status == "stale" || entry.Status == "missing" {
			status = StatusPartial
			reasonCodes = append(reasonCodes, entry.Status)
		}
		for _, ref := range entry.ProofRefs {
			proofRefs = append(proofRefs, ref.RecordID)
		}
	}
	return insuranceEntry("freeze_windows", status, []string{"gait"}, uniqueStrings(sourceRefs), scoreForStatus(status), uniqueStrings(reasonCodes), nil).withProofRefs(uniqueStrings(proofRefs))
}

func buildSandboxingEntry(coverage *SandboxCoverage) InsuranceEvidenceEntry {
	if coverage == nil || len(coverage.Entries) == 0 {
		return insuranceEntry("sandboxing", StatusGap, []string{"gait"}, nil, 0, []string{"missing_sandbox_evidence"}, []string{"ingest Gait sandbox evidence"})
	}
	status := StatusCovered
	reasonCodes := []string{}
	sourceRefs := make([]string, 0)
	proofRefs := make([]string, 0)
	for _, entry := range coverage.Entries {
		sourceRefs = append(sourceRefs, entry.ArtifactID)
		if entry.Status != StatusCovered {
			status = StatusPartial
			reasonCodes = append(reasonCodes, entry.GapReasonCodes...)
		}
		for _, ref := range entry.ProofRefs {
			proofRefs = append(proofRefs, ref.RecordID)
		}
	}
	return insuranceEntry("sandboxing", status, []string{"gait"}, uniqueStrings(sourceRefs), scoreForStatus(status), uniqueStrings(reasonCodes), nil).withProofRefs(uniqueStrings(proofRefs))
}

func buildProofOfEnforcementEntry(register *AuthorizationRegister) InsuranceEvidenceEntry {
	if register == nil || len(register.Entries) == 0 {
		return insuranceEntry("proof_of_enforcement", StatusGap, []string{"gait", "proof"}, nil, 0, []string{"missing_authorization_register"}, []string{"build authorization register from Gait evidence"})
	}
	proofRefs := make([]string, 0)
	for _, entry := range register.Entries {
		for _, ref := range entry.LinkedProofRefs {
			proofRefs = append(proofRefs, ref.RecordID)
		}
	}
	status := StatusCovered
	reasonCodes := []string{}
	for _, entry := range register.Entries {
		if entry.Status != "verified" {
			status = StatusPartial
			reasonCodes = append(reasonCodes, entry.GapReasonCodes...)
		}
	}
	return insuranceEntry("proof_of_enforcement", status, []string{"gait", "proof"}, uniqueStrings(proofRefs), scoreForStatus(status), uniqueStrings(reasonCodes), nil)
}

func insuranceEntry(name string, status string, sourceProducts []string, sourceRefs []string, score float64, reasonCodes []string, actionableGaps []string) InsuranceEvidenceEntry {
	products := uniqueStrings(sourceProducts)
	if products == nil {
		products = []string{}
	}
	return InsuranceEvidenceEntry{
		Name:               name,
		Status:             status,
		SourceProducts:     products,
		SourceArtifactRefs: uniqueStrings(sourceRefs),
		CompletenessScore:  score,
		ReasonCodes:        uniqueStrings(reasonCodes),
		ActionableGaps:     uniqueStrings(actionableGaps),
	}
}

func (entry InsuranceEvidenceEntry) withProofRefs(proofRefs []string) InsuranceEvidenceEntry {
	entry.ProofRecordRefs = uniqueStrings(proofRefs)
	return entry
}

func scoreForStatus(status string) float64 {
	switch status {
	case StatusCovered:
		return 1.0
	case StatusPartial:
		return 0.5
	default:
		return 0
	}
}

func hasWrkrEvidence(records []proof.Record) bool {
	for _, record := range records {
		if strings.TrimSpace(record.SourceProduct) == "wrkr" {
			return true
		}
	}
	return false
}

func wrkrRecordIDs(records []proof.Record) []string {
	out := make([]string, 0)
	for _, record := range records {
		if strings.TrimSpace(record.SourceProduct) == "wrkr" {
			out = append(out, strings.TrimSpace(record.RecordID))
		}
	}
	return uniqueStrings(out)
}

func stringFromEvent(record proof.Record, key string) string {
	value, _ := record.Event[key].(string)
	return strings.TrimSpace(value)
}

func stringFromMetadata(record proof.Record, key string) string {
	value, _ := record.Metadata[key].(string)
	return strings.TrimSpace(value)
}

func stringSliceFromMetadata(record proof.Record, key string) []string {
	raw, ok := record.Metadata[key]
	if !ok {
		return nil
	}
	switch typed := raw.(type) {
	case []string:
		return uniqueStrings(typed)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if value, ok := item.(string); ok {
				if trimmed := strings.TrimSpace(value); trimmed != "" {
					out = append(out, trimmed)
				}
			}
		}
		return uniqueStrings(out)
	default:
		return nil
	}
}

func sortedRecords(records []proof.Record) []proof.Record {
	out := append([]proof.Record(nil), records...)
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Timestamp.Equal(out[j].Timestamp) {
			return out[i].Timestamp.Before(out[j].Timestamp)
		}
		if out[i].RecordID != out[j].RecordID {
			return out[i].RecordID < out[j].RecordID
		}
		return out[i].Integrity.RecordHash < out[j].Integrity.RecordHash
	})
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}

func containsString(values []string, target string) bool {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" {
		return false
	}
	for _, value := range values {
		if value == trimmed {
			return true
		}
	}
	return false
}

func containsReason(values []string, target string) bool {
	return containsString(values, target)
}
