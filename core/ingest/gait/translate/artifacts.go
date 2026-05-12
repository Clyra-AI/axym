package translate

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Clyra-AI/proof"
)

const (
	ArtifactTypeAuthorizationBundle  = "gate_authorization_bundle"
	ArtifactTypeAuthorizationProfile = "packspec_authorization_profile"
	ArtifactTypeCredentialPosture    = "credential_posture" // #nosec G101 -- descriptive artifact type, not a credential value.
	ArtifactTypeFreezeWindow         = "freeze_window"
	ArtifactTypeKillSwitch           = "kill_switch"
	ArtifactTypeEnforcementExplain   = "enforcement_explain"
	ArtifactTypeSandboxPolicy        = "sandbox_policy"
	ArtifactTypeTrustGraduation      = "trust_graduation"

	ReasonInvalidAuthorizationArtifact    = "GAIT_INVALID_AUTHORIZATION_ARTIFACT"
	ReasonMissingAuthorizationBundleID    = "GAIT_MISSING_AUTHORIZATION_BUNDLE_ID"
	ReasonDuplicateAuthorizationBundleID  = "GAIT_DUPLICATE_AUTHORIZATION_BUNDLE_ID"
	ReasonUnsupportedAuthorizationProfile = "GAIT_UNSUPPORTED_AUTHORIZATION_PROFILE_VERSION"
	ReasonUnsupportedArtifactType         = "GAIT_UNSUPPORTED_ARTIFACT_TYPE"
	ReasonUnverifiableLinkedRefs          = "GAIT_UNVERIFIABLE_LINKED_REFS"
)

var supportedArtifactTypes = map[string]struct{}{
	ArtifactTypeAuthorizationBundle:  {},
	ArtifactTypeAuthorizationProfile: {},
	ArtifactTypeCredentialPosture:    {},
	ArtifactTypeFreezeWindow:         {},
	ArtifactTypeKillSwitch:           {},
	ArtifactTypeEnforcementExplain:   {},
	ArtifactTypeSandboxPolicy:        {},
	ArtifactTypeTrustGraduation:      {},
}

type ArtifactBase struct {
	Kind               string               `json:"kind"`
	ArtifactID         string               `json:"artifact_id"`
	BundleID           string               `json:"bundle_id"`
	Timestamp          string               `json:"timestamp"`
	Decision           string               `json:"decision,omitempty"`
	TraceID            string               `json:"trace_id,omitempty"`
	IntentDigest       string               `json:"intent_digest,omitempty"`
	PolicyDigest       string               `json:"policy_digest,omitempty"`
	SchemaVersion      string               `json:"schema_version,omitempty"`
	ProfileVersion     string               `json:"profile_version,omitempty"`
	Source             string               `json:"source,omitempty"`
	SourceProduct      string               `json:"source_product"`
	SourceArtifactPath string               `json:"source_artifact_path"`
	TargetKind         string               `json:"target_kind,omitempty"`
	TargetID           string               `json:"target_id,omitempty"`
	Environment        string               `json:"environment,omitempty"`
	AgentID            string               `json:"agent_id,omitempty"`
	Verification       VerificationMetadata `json:"verification,omitempty"`
	Links              ArtifactLinks        `json:"links,omitempty"`
}

type VerificationMetadata struct {
	Status         string   `json:"status,omitempty"`
	SchemaValid    bool     `json:"schema_valid"`
	SignatureValid *bool    `json:"signature_valid,omitempty"`
	ReasonCodes    []string `json:"reason_codes,omitempty"`
}

type ArtifactLinks struct {
	DecisionTraceRefs      []string `json:"decision_trace_refs,omitempty"`
	ApprovalAuditRefs      []string `json:"approval_audit_refs,omitempty"`
	CredentialEvidenceRefs []string `json:"credential_evidence_refs,omitempty"`
	ActionOutcomeRefs      []string `json:"action_outcome_refs,omitempty"`
	ProofRecordIDs         []string `json:"proof_record_ids,omitempty"`
}

type AuthorizationBundle struct {
	ArtifactBase `json:",inline"`
}

type CredentialPosture struct {
	ArtifactBase              `json:",inline"`
	StandingCredentialBlocked bool     `json:"standing_credential_blocked"`
	JITRequired               bool     `json:"jit_required"`
	BrokerSource              string   `json:"broker_source,omitempty"`
	Issuer                    string   `json:"issuer,omitempty"`
	TTLSeconds                int      `json:"ttl_seconds,omitempty"`
	Scope                     string   `json:"scope,omitempty"`
	BindingProofRef           string   `json:"binding_proof_ref,omitempty"`
	ActionRefs                []string `json:"action_refs,omitempty"`
	SecretRedacted            bool     `json:"secret_redacted,omitempty"`
}

type FreezeWindow struct {
	ArtifactBase `json:",inline"`
	State        string `json:"state,omitempty"`
	ApprovalRef  string `json:"approval_ref,omitempty"`
	Reason       string `json:"reason,omitempty"`
	Explain      string `json:"explain,omitempty"`
	Stale        bool   `json:"stale,omitempty"`
}

type KillSwitch struct {
	ArtifactBase         `json:",inline"`
	State                string `json:"state,omitempty"`
	BlockedDispatchProof string `json:"blocked_dispatch_proof,omitempty"`
	Expiry               string `json:"expiry,omitempty"`
	Actor                string `json:"actor,omitempty"`
}

type EnforcementExplain struct {
	ArtifactBase      `json:",inline"`
	MissingFields     []string `json:"missing_fields,omitempty"`
	CredentialPosture string   `json:"credential_posture,omitempty"`
	FreezeState       string   `json:"freeze_state,omitempty"`
	KillSwitchState   string   `json:"kill_switch_state,omitempty"`
	SandboxState      string   `json:"sandbox_state,omitempty"`
	ReasonCodes       []string `json:"reason_codes,omitempty"`
}

type SandboxPolicy struct {
	ArtifactBase        `json:",inline"`
	Path                string   `json:"path,omitempty"`
	NetworkMode         string   `json:"network_mode,omitempty"`
	WritablePaths       []string `json:"writable_paths,omitempty"`
	EnvExposure         []string `json:"env_exposure,omitempty"`
	TimeoutSeconds      int      `json:"timeout_seconds,omitempty"`
	FilesystemIsolation string   `json:"filesystem_isolation,omitempty"`
	PolicyResult        string   `json:"policy_result,omitempty"`
	GapReasonCodes      []string `json:"gap_reason_codes,omitempty"`
}

type TrustGraduation struct {
	ArtifactBase            `json:",inline"`
	Path                    string   `json:"path,omitempty"`
	CurrentStage            string   `json:"current_stage,omitempty"`
	PriorStage              string   `json:"prior_stage,omitempty"`
	PostureState            string   `json:"posture_state,omitempty"`
	ApprovalFatigueSignals  []string `json:"approval_fatigue_signals,omitempty"`
	RemainingGapReasonCodes []string `json:"remaining_gap_reason_codes,omitempty"`
}

type SourceArtifact struct {
	AuthorizationBundle *AuthorizationBundle `json:"authorization_bundle,omitempty"`
	CredentialPosture   *CredentialPosture   `json:"credential_posture,omitempty"`
	FreezeWindow        *FreezeWindow        `json:"freeze_window,omitempty"`
	KillSwitch          *KillSwitch          `json:"kill_switch,omitempty"`
	EnforcementExplain  *EnforcementExplain  `json:"enforcement_explain,omitempty"`
	SandboxPolicy       *SandboxPolicy       `json:"sandbox_policy,omitempty"`
	TrustGraduation     *TrustGraduation     `json:"trust_graduation,omitempty"`
}

func (a SourceArtifact) Kind() string {
	switch {
	case a.AuthorizationBundle != nil:
		return a.AuthorizationBundle.Kind
	case a.CredentialPosture != nil:
		return a.CredentialPosture.Kind
	case a.FreezeWindow != nil:
		return a.FreezeWindow.Kind
	case a.KillSwitch != nil:
		return a.KillSwitch.Kind
	case a.EnforcementExplain != nil:
		return a.EnforcementExplain.Kind
	case a.SandboxPolicy != nil:
		return a.SandboxPolicy.Kind
	case a.TrustGraduation != nil:
		return a.TrustGraduation.Kind
	default:
		return ""
	}
}

func (a SourceArtifact) Base() ArtifactBase {
	switch {
	case a.AuthorizationBundle != nil:
		return a.AuthorizationBundle.ArtifactBase
	case a.CredentialPosture != nil:
		return a.CredentialPosture.ArtifactBase
	case a.FreezeWindow != nil:
		return a.FreezeWindow.ArtifactBase
	case a.KillSwitch != nil:
		return a.KillSwitch.ArtifactBase
	case a.EnforcementExplain != nil:
		return a.EnforcementExplain.ArtifactBase
	case a.SandboxPolicy != nil:
		return a.SandboxPolicy.ArtifactBase
	case a.TrustGraduation != nil:
		return a.TrustGraduation.ArtifactBase
	default:
		return ArtifactBase{}
	}
}

func (a SourceArtifact) ProofRecordIDs() []string {
	return append([]string(nil), a.Base().Links.ProofRecordIDs...)
}

func (a SourceArtifact) BundleID() string {
	return a.Base().BundleID
}

func (a SourceArtifact) ArtifactID() string {
	return a.Base().ArtifactID
}

func (a SourceArtifact) SourcePath() string {
	return a.Base().SourceArtifactPath
}

type rawBase struct {
	ArtifactType           string               `json:"artifact_type,omitempty"`
	ProfileType            string               `json:"profile_type,omitempty"`
	ArtifactID             string               `json:"artifact_id,omitempty"`
	ID                     string               `json:"id,omitempty"`
	BundleID               string               `json:"bundle_id,omitempty"`
	Timestamp              string               `json:"timestamp,omitempty"`
	CreatedAt              string               `json:"created_at,omitempty"`
	IssuedAt               string               `json:"issued_at,omitempty"`
	Decision               string               `json:"decision,omitempty"`
	TraceID                string               `json:"trace_id,omitempty"`
	IntentDigest           string               `json:"intent_digest,omitempty"`
	PolicyDigest           string               `json:"policy_digest,omitempty"`
	SchemaVersion          string               `json:"schema_version,omitempty"`
	ProfileVersion         string               `json:"profile_version,omitempty"`
	Source                 string               `json:"source,omitempty"`
	SourceProduct          string               `json:"source_product,omitempty"`
	TargetKind             string               `json:"target_kind,omitempty"`
	TargetID               string               `json:"target_id,omitempty"`
	Environment            string               `json:"environment,omitempty"`
	AgentID                string               `json:"agent_id,omitempty"`
	Verification           VerificationMetadata `json:"verification,omitempty"`
	DecisionTraceRefs      []string             `json:"decision_trace_refs,omitempty"`
	ApprovalAuditRefs      []string             `json:"approval_audit_refs,omitempty"`
	CredentialEvidenceRefs []string             `json:"credential_evidence_refs,omitempty"`
	ActionOutcomeRefs      []string             `json:"action_outcome_refs,omitempty"`
	LinkedRecordIDs        []string             `json:"linked_record_ids,omitempty"`
}

type rawAuthorizationArtifact struct {
	rawBase
	CredentialPosture  *rawCredentialArtifact `json:"credential_posture,omitempty"`
	FreezeWindow       *rawFreezeWindow       `json:"freeze_window,omitempty"`
	KillSwitch         *rawKillSwitch         `json:"kill_switch,omitempty"`
	EnforcementExplain *rawEnforcementExplain `json:"enforcement_explain,omitempty"`
	SandboxPolicy      *rawSandboxPolicy      `json:"sandbox,omitempty"`
	TrustGraduation    *rawTrustGraduation    `json:"trust_graduation,omitempty"`
}

type rawCredentialArtifact struct {
	rawBase
	StandingCredentialBlocked *bool    `json:"standing_credential_blocked,omitempty"`
	JITRequired               *bool    `json:"jit_required,omitempty"`
	BrokerSource              string   `json:"broker_source,omitempty"`
	Issuer                    string   `json:"issuer,omitempty"`
	TTLSeconds                int      `json:"ttl_seconds,omitempty"`
	Scope                     string   `json:"scope,omitempty"`
	BindingProofRef           string   `json:"binding_proof_ref,omitempty"`
	ActionRefs                []string `json:"action_refs,omitempty"`
}

type rawFreezeWindow struct {
	rawBase
	State       string `json:"state,omitempty"`
	ApprovalRef string `json:"approval_ref,omitempty"`
	Reason      string `json:"reason,omitempty"`
	Explain     string `json:"explain,omitempty"`
	Stale       bool   `json:"stale,omitempty"`
}

type rawKillSwitch struct {
	rawBase
	State                string `json:"state,omitempty"`
	BlockedDispatchProof string `json:"blocked_dispatch_proof,omitempty"`
	Expiry               string `json:"expiry,omitempty"`
	Actor                string `json:"actor,omitempty"`
}

type rawEnforcementExplain struct {
	rawBase
	MissingFields     []string `json:"missing_fields,omitempty"`
	CredentialPosture string   `json:"credential_posture,omitempty"`
	FreezeState       string   `json:"freeze_state,omitempty"`
	KillSwitchState   string   `json:"kill_switch_state,omitempty"`
	SandboxState      string   `json:"sandbox_state,omitempty"`
	ReasonCodes       []string `json:"reason_codes,omitempty"`
}

type rawSandboxPolicy struct {
	rawBase
	Path                string   `json:"path,omitempty"`
	NetworkMode         string   `json:"network_mode,omitempty"`
	WritablePaths       []string `json:"writable_paths,omitempty"`
	EnvExposure         []string `json:"env_exposure,omitempty"`
	TimeoutSeconds      int      `json:"timeout_seconds,omitempty"`
	FilesystemIsolation string   `json:"filesystem_isolation,omitempty"`
	PolicyResult        string   `json:"policy_result,omitempty"`
	GapReasonCodes      []string `json:"gap_reason_codes,omitempty"`
}

type rawTrustGraduation struct {
	rawBase
	Path                    string   `json:"path,omitempty"`
	CurrentStage            string   `json:"current_stage,omitempty"`
	PriorStage              string   `json:"prior_stage,omitempty"`
	PostureState            string   `json:"posture_state,omitempty"`
	ApprovalFatigueSignals  []string `json:"approval_fatigue_signals,omitempty"`
	RemainingGapReasonCodes []string `json:"remaining_gap_reason_codes,omitempty"`
}

func ParseSourceArtifact(raw []byte, sourcePath string) ([]SourceArtifact, error) {
	artifactType, err := detectArtifactType(raw)
	if err != nil {
		return nil, &Error{ReasonCode: ReasonInvalidAuthorizationArtifact, Message: "decode gait source artifact", Err: err}
	}
	if artifactType == "" {
		return nil, nil
	}
	if _, ok := supportedArtifactTypes[artifactType]; !ok {
		return nil, &Error{
			ReasonCode: ReasonUnsupportedArtifactType,
			Message:    fmt.Sprintf("unsupported gait source artifact type %q", artifactType),
		}
	}

	switch artifactType {
	case ArtifactTypeAuthorizationBundle, ArtifactTypeAuthorizationProfile:
		return parseAuthorizationArtifact(raw, sourcePath, artifactType)
	case ArtifactTypeCredentialPosture:
		return parseCredentialArtifact(raw, sourcePath)
	case ArtifactTypeFreezeWindow:
		return parseFreezeWindowArtifact(raw, sourcePath)
	case ArtifactTypeKillSwitch:
		return parseKillSwitchArtifact(raw, sourcePath)
	case ArtifactTypeEnforcementExplain:
		return parseExplainArtifact(raw, sourcePath)
	case ArtifactTypeSandboxPolicy:
		return parseSandboxArtifact(raw, sourcePath)
	case ArtifactTypeTrustGraduation:
		return parseTrustArtifact(raw, sourcePath)
	default:
		return nil, nil
	}
}

func TranslateSourceArtifact(artifact SourceArtifact) (*proof.Record, error) {
	base := artifact.Base()
	event := map[string]any{
		"artifact_id":   base.ArtifactID,
		"artifact_type": base.Kind,
		"bundle_id":     base.BundleID,
	}
	setString(event, "decision", base.Decision)
	setString(event, "trace_id", base.TraceID)
	setString(event, "intent_digest", base.IntentDigest)
	setString(event, "policy_digest", base.PolicyDigest)
	setString(event, "target_kind", base.TargetKind)
	setString(event, "target_id", base.TargetID)
	setString(event, "environment", base.Environment)

	metadata := map[string]any{
		"gait_artifact_type":             base.Kind,
		"gait_source_artifact_id":        base.ArtifactID,
		"gait_source_artifact_path":      base.SourceArtifactPath,
		"gait_source_product":            base.SourceProduct,
		"gait_verification_status":       strings.TrimSpace(base.Verification.Status),
		"gait_verification_schema_valid": base.Verification.SchemaValid,
	}
	if base.SchemaVersion != "" {
		metadata["gait_schema_version"] = base.SchemaVersion
	}
	if base.ProfileVersion != "" {
		metadata["gait_profile_version"] = base.ProfileVersion
	}
	if base.Verification.SignatureValid != nil {
		metadata["gait_signature_valid"] = *base.Verification.SignatureValid
	}
	if len(base.Verification.ReasonCodes) > 0 {
		metadata["gait_verification_reason_codes"] = append([]string(nil), base.Verification.ReasonCodes...)
	}
	if len(base.Links.DecisionTraceRefs) > 0 {
		metadata["gait_decision_trace_refs"] = append([]string(nil), base.Links.DecisionTraceRefs...)
	}
	if len(base.Links.ApprovalAuditRefs) > 0 {
		metadata["gait_approval_audit_refs"] = append([]string(nil), base.Links.ApprovalAuditRefs...)
	}
	if len(base.Links.CredentialEvidenceRefs) > 0 {
		metadata["gait_credential_evidence_refs"] = append([]string(nil), base.Links.CredentialEvidenceRefs...)
	}
	if len(base.Links.ActionOutcomeRefs) > 0 {
		metadata["gait_action_outcome_refs"] = append([]string(nil), base.Links.ActionOutcomeRefs...)
	}
	if len(base.Links.ProofRecordIDs) > 0 {
		metadata["gait_proof_record_ids"] = append([]string(nil), base.Links.ProofRecordIDs...)
	}

	recordType := "policy_enforcement"

	switch {
	case artifact.CredentialPosture != nil:
		payload := artifact.CredentialPosture
		event["standing_credential_blocked"] = payload.StandingCredentialBlocked
		event["jit_required"] = payload.JITRequired
		setString(event, "broker_source", payload.BrokerSource)
		setString(event, "issuer", payload.Issuer)
		if payload.TTLSeconds > 0 {
			event["ttl_seconds"] = payload.TTLSeconds
		}
		setString(event, "scope", payload.Scope)
		setString(event, "binding_proof_ref", payload.BindingProofRef)
		if len(payload.ActionRefs) > 0 {
			metadata["gait_action_refs"] = append([]string(nil), payload.ActionRefs...)
		}
		if payload.SecretRedacted {
			metadata["gait_secret_redacted"] = true
		}
	case artifact.FreezeWindow != nil:
		payload := artifact.FreezeWindow
		setString(event, "state", payload.State)
		setString(event, "approval_ref", payload.ApprovalRef)
		setString(event, "reason", payload.Reason)
		setString(event, "explain", payload.Explain)
		if payload.Stale {
			event["stale"] = true
		}
	case artifact.KillSwitch != nil:
		payload := artifact.KillSwitch
		setString(event, "state", payload.State)
		setString(event, "blocked_dispatch_proof", payload.BlockedDispatchProof)
		setString(event, "expiry", payload.Expiry)
		setString(event, "actor", payload.Actor)
	case artifact.EnforcementExplain != nil:
		recordType = "decision"
		payload := artifact.EnforcementExplain
		if len(payload.MissingFields) > 0 {
			event["missing_fields"] = append([]string(nil), payload.MissingFields...)
		}
		setString(event, "credential_posture", payload.CredentialPosture)
		setString(event, "freeze_state", payload.FreezeState)
		setString(event, "kill_switch_state", payload.KillSwitchState)
		setString(event, "sandbox_state", payload.SandboxState)
		if len(payload.ReasonCodes) > 0 {
			metadata["gait_reason_codes"] = append([]string(nil), payload.ReasonCodes...)
		}
	case artifact.SandboxPolicy != nil:
		payload := artifact.SandboxPolicy
		setString(event, "path", payload.Path)
		setString(event, "network_mode", payload.NetworkMode)
		if len(payload.WritablePaths) > 0 {
			event["writable_paths"] = append([]string(nil), payload.WritablePaths...)
		}
		if len(payload.EnvExposure) > 0 {
			event["env_exposure"] = append([]string(nil), payload.EnvExposure...)
		}
		if payload.TimeoutSeconds > 0 {
			event["timeout_seconds"] = payload.TimeoutSeconds
		}
		setString(event, "filesystem_isolation", payload.FilesystemIsolation)
		setString(event, "policy_result", payload.PolicyResult)
		if len(payload.GapReasonCodes) > 0 {
			metadata["gait_gap_reason_codes"] = append([]string(nil), payload.GapReasonCodes...)
		}
	case artifact.TrustGraduation != nil:
		recordType = "risk_assessment"
		payload := artifact.TrustGraduation
		setString(event, "path", payload.Path)
		setString(event, "current_stage", payload.CurrentStage)
		setString(event, "prior_stage", payload.PriorStage)
		setString(event, "posture_state", payload.PostureState)
		if len(payload.ApprovalFatigueSignals) > 0 {
			metadata["gait_approval_fatigue_signals"] = append([]string(nil), payload.ApprovalFatigueSignals...)
		}
		if len(payload.RemainingGapReasonCodes) > 0 {
			metadata["gait_gap_reason_codes"] = append([]string(nil), payload.RemainingGapReasonCodes...)
		}
	}

	timestamp, err := parseTimestamp(base.Timestamp)
	if err != nil {
		return nil, &Error{
			ReasonCode: ReasonInvalidAuthorizationArtifact,
			Message:    "gait source artifact timestamp is required and must be RFC3339",
			Err:        err,
		}
	}
	source := strings.TrimSpace(base.Source)
	if source == "" {
		source = "gait"
	}
	record, err := proof.NewRecord(proof.RecordOpts{
		Timestamp:     timestamp,
		Source:        source,
		SourceProduct: firstNonEmpty(base.SourceProduct, "gait"),
		AgentID:       strings.TrimSpace(base.AgentID),
		Type:          recordType,
		Event:         event,
		Metadata:      metadata,
		Controls: proof.Controls{
			PermissionsEnforced: true,
			ApprovedScope:       "gait-source-artifact",
		},
	})
	if err != nil {
		return nil, &Error{
			ReasonCode: ReasonInvalidAuthorizationArtifact,
			Message:    "build translated gait source artifact proof record",
			Err:        err,
		}
	}
	return record, nil
}

func parseAuthorizationArtifact(raw []byte, sourcePath string, artifactType string) ([]SourceArtifact, error) {
	var payload rawAuthorizationArtifact
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, &Error{ReasonCode: ReasonInvalidAuthorizationArtifact, Message: "decode authorization artifact", Err: err}
	}
	base, err := buildBase(payload.rawBase, artifactType, sourcePath, true)
	if err != nil {
		return nil, err
	}
	if artifactType == ArtifactTypeAuthorizationProfile {
		version := normalizeVersion(payload.ProfileVersion)
		if version != "v1" {
			return nil, &Error{
				ReasonCode: ReasonUnsupportedAuthorizationProfile,
				Message:    fmt.Sprintf("unsupported authorization profile version %q", strings.TrimSpace(payload.ProfileVersion)),
			}
		}
		base.ProfileVersion = version
	}

	out := []SourceArtifact{
		{AuthorizationBundle: &AuthorizationBundle{ArtifactBase: base}},
	}
	if payload.CredentialPosture != nil {
		cred := toCredentialPosture(base, *payload.CredentialPosture)
		out = append(out, SourceArtifact{CredentialPosture: &cred})
	}
	if payload.FreezeWindow != nil {
		freeze := toFreezeWindow(base, *payload.FreezeWindow)
		out = append(out, SourceArtifact{FreezeWindow: &freeze})
	}
	if payload.KillSwitch != nil {
		kill := toKillSwitch(base, *payload.KillSwitch)
		out = append(out, SourceArtifact{KillSwitch: &kill})
	}
	if payload.EnforcementExplain != nil {
		explain := toEnforcementExplain(base, *payload.EnforcementExplain)
		out = append(out, SourceArtifact{EnforcementExplain: &explain})
	}
	if payload.SandboxPolicy != nil {
		sandbox := toSandboxPolicy(base, *payload.SandboxPolicy)
		out = append(out, SourceArtifact{SandboxPolicy: &sandbox})
	}
	if payload.TrustGraduation != nil {
		trust := toTrustGraduation(base, *payload.TrustGraduation)
		out = append(out, SourceArtifact{TrustGraduation: &trust})
	}
	return out, nil
}

func parseCredentialArtifact(raw []byte, sourcePath string) ([]SourceArtifact, error) {
	var payload rawCredentialArtifact
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, &Error{ReasonCode: ReasonInvalidAuthorizationArtifact, Message: "decode credential posture artifact", Err: err}
	}
	base, err := buildBase(payload.rawBase, ArtifactTypeCredentialPosture, sourcePath, false)
	if err != nil {
		return nil, err
	}
	cred := toCredentialPosture(base, payload)
	return []SourceArtifact{{CredentialPosture: &cred}}, nil
}

func parseFreezeWindowArtifact(raw []byte, sourcePath string) ([]SourceArtifact, error) {
	var payload rawFreezeWindow
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, &Error{ReasonCode: ReasonInvalidAuthorizationArtifact, Message: "decode freeze-window artifact", Err: err}
	}
	base, err := buildBase(payload.rawBase, ArtifactTypeFreezeWindow, sourcePath, false)
	if err != nil {
		return nil, err
	}
	freeze := toFreezeWindow(base, payload)
	return []SourceArtifact{{FreezeWindow: &freeze}}, nil
}

func parseKillSwitchArtifact(raw []byte, sourcePath string) ([]SourceArtifact, error) {
	var payload rawKillSwitch
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, &Error{ReasonCode: ReasonInvalidAuthorizationArtifact, Message: "decode kill-switch artifact", Err: err}
	}
	base, err := buildBase(payload.rawBase, ArtifactTypeKillSwitch, sourcePath, false)
	if err != nil {
		return nil, err
	}
	kill := toKillSwitch(base, payload)
	return []SourceArtifact{{KillSwitch: &kill}}, nil
}

func parseExplainArtifact(raw []byte, sourcePath string) ([]SourceArtifact, error) {
	var payload rawEnforcementExplain
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, &Error{ReasonCode: ReasonInvalidAuthorizationArtifact, Message: "decode enforcement explain artifact", Err: err}
	}
	base, err := buildBase(payload.rawBase, ArtifactTypeEnforcementExplain, sourcePath, false)
	if err != nil {
		return nil, err
	}
	explain := toEnforcementExplain(base, payload)
	return []SourceArtifact{{EnforcementExplain: &explain}}, nil
}

func parseSandboxArtifact(raw []byte, sourcePath string) ([]SourceArtifact, error) {
	var payload rawSandboxPolicy
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, &Error{ReasonCode: ReasonInvalidAuthorizationArtifact, Message: "decode sandbox artifact", Err: err}
	}
	base, err := buildBase(payload.rawBase, ArtifactTypeSandboxPolicy, sourcePath, false)
	if err != nil {
		return nil, err
	}
	sandbox := toSandboxPolicy(base, payload)
	return []SourceArtifact{{SandboxPolicy: &sandbox}}, nil
}

func parseTrustArtifact(raw []byte, sourcePath string) ([]SourceArtifact, error) {
	var payload rawTrustGraduation
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, &Error{ReasonCode: ReasonInvalidAuthorizationArtifact, Message: "decode trust-graduation artifact", Err: err}
	}
	base, err := buildBase(payload.rawBase, ArtifactTypeTrustGraduation, sourcePath, false)
	if err != nil {
		return nil, err
	}
	trust := toTrustGraduation(base, payload)
	return []SourceArtifact{{TrustGraduation: &trust}}, nil
}

func detectArtifactType(raw []byte) (string, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", err
	}
	artifactType := strings.ToLower(strings.TrimSpace(rawString(payload["artifact_type"])))
	if artifactType != "" {
		return artifactType, nil
	}
	profileType := strings.ToLower(strings.TrimSpace(rawString(payload["profile_type"])))
	if profileType == "authorization" {
		return ArtifactTypeAuthorizationProfile, nil
	}
	return "", nil
}

func buildBase(raw rawBase, fallbackKind string, sourcePath string, requireBundleID bool) (ArtifactBase, error) {
	kind := firstNonEmpty(strings.ToLower(strings.TrimSpace(raw.ArtifactType)), fallbackKind)
	bundleID := strings.TrimSpace(raw.BundleID)
	if requireBundleID && bundleID == "" {
		return ArtifactBase{}, &Error{ReasonCode: ReasonMissingAuthorizationBundleID, Message: "bundle_id is required"}
	}
	if !requireBundleID && bundleID == "" {
		bundleID = firstNonEmpty(strings.TrimSpace(raw.BundleID), strings.TrimSpace(raw.ID))
	}
	base := ArtifactBase{
		Kind:               kind,
		ArtifactID:         firstNonEmpty(strings.TrimSpace(raw.ArtifactID), strings.TrimSpace(raw.ID), derivedArtifactID(bundleID, kind)),
		BundleID:           bundleID,
		Timestamp:          firstNonEmpty(strings.TrimSpace(raw.Timestamp), strings.TrimSpace(raw.CreatedAt), strings.TrimSpace(raw.IssuedAt)),
		Decision:           strings.TrimSpace(raw.Decision),
		TraceID:            strings.TrimSpace(raw.TraceID),
		IntentDigest:       strings.TrimSpace(raw.IntentDigest),
		PolicyDigest:       strings.TrimSpace(raw.PolicyDigest),
		SchemaVersion:      normalizeVersion(raw.SchemaVersion),
		ProfileVersion:     normalizeVersion(raw.ProfileVersion),
		Source:             strings.TrimSpace(raw.Source),
		SourceProduct:      firstNonEmpty(strings.TrimSpace(raw.SourceProduct), "gait"),
		SourceArtifactPath: filepathSlash(sourcePath),
		TargetKind:         strings.TrimSpace(raw.TargetKind),
		TargetID:           strings.TrimSpace(raw.TargetID),
		Environment:        strings.TrimSpace(raw.Environment),
		AgentID:            strings.TrimSpace(raw.AgentID),
		Verification: VerificationMetadata{
			Status:         strings.TrimSpace(raw.Verification.Status),
			SchemaValid:    raw.Verification.SchemaValid,
			SignatureValid: raw.Verification.SignatureValid,
			ReasonCodes:    uniqueStrings(raw.Verification.ReasonCodes),
		},
		Links: ArtifactLinks{
			DecisionTraceRefs:      uniqueStrings(raw.DecisionTraceRefs),
			ApprovalAuditRefs:      uniqueStrings(raw.ApprovalAuditRefs),
			CredentialEvidenceRefs: uniqueStrings(raw.CredentialEvidenceRefs),
			ActionOutcomeRefs:      uniqueStrings(raw.ActionOutcomeRefs),
			ProofRecordIDs:         uniqueStrings(raw.LinkedRecordIDs),
		},
	}
	if base.Timestamp == "" {
		return ArtifactBase{}, &Error{ReasonCode: ReasonInvalidAuthorizationArtifact, Message: "artifact timestamp is required"}
	}
	return base, nil
}

func toCredentialPosture(base ArtifactBase, raw rawCredentialArtifact) CredentialPosture {
	next := base
	next.Kind = ArtifactTypeCredentialPosture
	next.ArtifactID = firstNonEmpty(strings.TrimSpace(raw.ArtifactID), strings.TrimSpace(raw.ID), derivedArtifactID(base.BundleID, ArtifactTypeCredentialPosture))
	standingBlocked := raw.StandingCredentialBlocked != nil && *raw.StandingCredentialBlocked
	jitRequired := raw.JITRequired != nil && *raw.JITRequired
	return CredentialPosture{
		ArtifactBase:              next,
		StandingCredentialBlocked: standingBlocked,
		JITRequired:               jitRequired,
		BrokerSource:              strings.TrimSpace(raw.BrokerSource),
		Issuer:                    strings.TrimSpace(raw.Issuer),
		TTLSeconds:                raw.TTLSeconds,
		Scope:                     strings.TrimSpace(raw.Scope),
		BindingProofRef:           strings.TrimSpace(raw.BindingProofRef),
		ActionRefs:                uniqueStrings(raw.ActionRefs),
	}
}

func toFreezeWindow(base ArtifactBase, raw rawFreezeWindow) FreezeWindow {
	next := base
	next.Kind = ArtifactTypeFreezeWindow
	next.ArtifactID = firstNonEmpty(strings.TrimSpace(raw.ArtifactID), strings.TrimSpace(raw.ID), derivedArtifactID(base.BundleID, ArtifactTypeFreezeWindow))
	return FreezeWindow{
		ArtifactBase: next,
		State:        strings.TrimSpace(raw.State),
		ApprovalRef:  strings.TrimSpace(raw.ApprovalRef),
		Reason:       strings.TrimSpace(raw.Reason),
		Explain:      strings.TrimSpace(raw.Explain),
		Stale:        raw.Stale,
	}
}

func toKillSwitch(base ArtifactBase, raw rawKillSwitch) KillSwitch {
	next := base
	next.Kind = ArtifactTypeKillSwitch
	next.ArtifactID = firstNonEmpty(strings.TrimSpace(raw.ArtifactID), strings.TrimSpace(raw.ID), derivedArtifactID(base.BundleID, ArtifactTypeKillSwitch))
	return KillSwitch{
		ArtifactBase:         next,
		State:                strings.TrimSpace(raw.State),
		BlockedDispatchProof: strings.TrimSpace(raw.BlockedDispatchProof),
		Expiry:               strings.TrimSpace(raw.Expiry),
		Actor:                strings.TrimSpace(raw.Actor),
	}
}

func toEnforcementExplain(base ArtifactBase, raw rawEnforcementExplain) EnforcementExplain {
	next := base
	next.Kind = ArtifactTypeEnforcementExplain
	next.ArtifactID = firstNonEmpty(strings.TrimSpace(raw.ArtifactID), strings.TrimSpace(raw.ID), derivedArtifactID(base.BundleID, ArtifactTypeEnforcementExplain))
	return EnforcementExplain{
		ArtifactBase:      next,
		MissingFields:     uniqueStrings(raw.MissingFields),
		CredentialPosture: strings.TrimSpace(raw.CredentialPosture),
		FreezeState:       strings.TrimSpace(raw.FreezeState),
		KillSwitchState:   strings.TrimSpace(raw.KillSwitchState),
		SandboxState:      strings.TrimSpace(raw.SandboxState),
		ReasonCodes:       uniqueStrings(raw.ReasonCodes),
	}
}

func toSandboxPolicy(base ArtifactBase, raw rawSandboxPolicy) SandboxPolicy {
	next := base
	next.Kind = ArtifactTypeSandboxPolicy
	next.ArtifactID = firstNonEmpty(strings.TrimSpace(raw.ArtifactID), strings.TrimSpace(raw.ID), derivedArtifactID(base.BundleID, ArtifactTypeSandboxPolicy))
	return SandboxPolicy{
		ArtifactBase:        next,
		Path:                strings.TrimSpace(raw.Path),
		NetworkMode:         strings.TrimSpace(raw.NetworkMode),
		WritablePaths:       uniqueStrings(raw.WritablePaths),
		EnvExposure:         uniqueStrings(raw.EnvExposure),
		TimeoutSeconds:      raw.TimeoutSeconds,
		FilesystemIsolation: strings.TrimSpace(raw.FilesystemIsolation),
		PolicyResult:        strings.TrimSpace(raw.PolicyResult),
		GapReasonCodes:      uniqueStrings(raw.GapReasonCodes),
	}
}

func toTrustGraduation(base ArtifactBase, raw rawTrustGraduation) TrustGraduation {
	next := base
	next.Kind = ArtifactTypeTrustGraduation
	next.ArtifactID = firstNonEmpty(strings.TrimSpace(raw.ArtifactID), strings.TrimSpace(raw.ID), derivedArtifactID(base.BundleID, ArtifactTypeTrustGraduation))
	return TrustGraduation{
		ArtifactBase:            next,
		Path:                    strings.TrimSpace(raw.Path),
		CurrentStage:            strings.TrimSpace(raw.CurrentStage),
		PriorStage:              strings.TrimSpace(raw.PriorStage),
		PostureState:            strings.TrimSpace(raw.PostureState),
		ApprovalFatigueSignals:  uniqueStrings(raw.ApprovalFatigueSignals),
		RemainingGapReasonCodes: uniqueStrings(raw.RemainingGapReasonCodes),
	}
}

func ValidateSourceArtifactLinks(artifact SourceArtifact, records []proof.Record) error {
	want := artifact.ProofRecordIDs()
	if len(want) == 0 {
		return nil
	}
	have := map[string]struct{}{}
	for _, record := range records {
		id := strings.TrimSpace(record.RecordID)
		if id == "" {
			continue
		}
		have[id] = struct{}{}
	}
	missing := make([]string, 0)
	for _, id := range want {
		if _, ok := have[id]; ok {
			continue
		}
		missing = append(missing, id)
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return &Error{
		ReasonCode: ReasonUnverifiableLinkedRefs,
		Message:    fmt.Sprintf("unverifiable linked proof record refs: %s", strings.Join(missing, ",")),
	}
}

func IsTranslatedSourceArtifact(record proof.Record) bool {
	_, ok := ExtractSourceArtifact(record)
	return ok
}

func ExtractSourceArtifact(record proof.Record) (SourceArtifact, bool) {
	kind := strings.TrimSpace(stringFromMap(record.Metadata, "gait_artifact_type"))
	if kind == "" || strings.TrimSpace(record.SourceProduct) != "gait" {
		return SourceArtifact{}, false
	}
	base := ArtifactBase{
		Kind:               kind,
		ArtifactID:         strings.TrimSpace(stringFromMap(record.Event, "artifact_id")),
		BundleID:           strings.TrimSpace(stringFromMap(record.Event, "bundle_id")),
		Timestamp:          record.Timestamp.UTC().Format(time.RFC3339),
		Decision:           strings.TrimSpace(stringFromMap(record.Event, "decision")),
		TraceID:            strings.TrimSpace(stringFromMap(record.Event, "trace_id")),
		IntentDigest:       strings.TrimSpace(stringFromMap(record.Event, "intent_digest")),
		PolicyDigest:       strings.TrimSpace(stringFromMap(record.Event, "policy_digest")),
		SchemaVersion:      strings.TrimSpace(stringFromMap(record.Metadata, "gait_schema_version")),
		ProfileVersion:     strings.TrimSpace(stringFromMap(record.Metadata, "gait_profile_version")),
		Source:             strings.TrimSpace(record.Source),
		SourceProduct:      strings.TrimSpace(record.SourceProduct),
		SourceArtifactPath: strings.TrimSpace(stringFromMap(record.Metadata, "gait_source_artifact_path")),
		TargetKind:         strings.TrimSpace(stringFromMap(record.Event, "target_kind")),
		TargetID:           strings.TrimSpace(stringFromMap(record.Event, "target_id")),
		Environment:        strings.TrimSpace(stringFromMap(record.Event, "environment")),
		AgentID:            strings.TrimSpace(record.AgentID),
		Verification: VerificationMetadata{
			Status:      strings.TrimSpace(stringFromMap(record.Metadata, "gait_verification_status")),
			SchemaValid: boolFromMap(record.Metadata, "gait_verification_schema_valid"),
			ReasonCodes: stringSliceFromMap(record.Metadata, "gait_verification_reason_codes"),
		},
		Links: ArtifactLinks{
			DecisionTraceRefs:      stringSliceFromMap(record.Metadata, "gait_decision_trace_refs"),
			ApprovalAuditRefs:      stringSliceFromMap(record.Metadata, "gait_approval_audit_refs"),
			CredentialEvidenceRefs: stringSliceFromMap(record.Metadata, "gait_credential_evidence_refs"),
			ActionOutcomeRefs:      stringSliceFromMap(record.Metadata, "gait_action_outcome_refs"),
			ProofRecordIDs:         stringSliceFromMap(record.Metadata, "gait_proof_record_ids"),
		},
	}
	if signatureValid, ok := optionalBoolFromMap(record.Metadata, "gait_signature_valid"); ok {
		base.Verification.SignatureValid = &signatureValid
	}

	switch kind {
	case ArtifactTypeAuthorizationBundle, ArtifactTypeAuthorizationProfile:
		return SourceArtifact{AuthorizationBundle: &AuthorizationBundle{ArtifactBase: base}}, true
	case ArtifactTypeCredentialPosture:
		return SourceArtifact{CredentialPosture: &CredentialPosture{
			ArtifactBase:              base,
			StandingCredentialBlocked: boolFromMap(record.Event, "standing_credential_blocked"),
			JITRequired:               boolFromMap(record.Event, "jit_required"),
			BrokerSource:              strings.TrimSpace(stringFromMap(record.Event, "broker_source")),
			Issuer:                    strings.TrimSpace(stringFromMap(record.Event, "issuer")),
			TTLSeconds:                intFromMap(record.Event, "ttl_seconds"),
			Scope:                     strings.TrimSpace(stringFromMap(record.Event, "scope")),
			BindingProofRef:           strings.TrimSpace(stringFromMap(record.Event, "binding_proof_ref")),
			ActionRefs:                stringSliceFromMap(record.Metadata, "gait_action_refs"),
			SecretRedacted:            boolFromMap(record.Metadata, "gait_secret_redacted"),
		}}, true
	case ArtifactTypeFreezeWindow:
		return SourceArtifact{FreezeWindow: &FreezeWindow{
			ArtifactBase: base,
			State:        strings.TrimSpace(stringFromMap(record.Event, "state")),
			ApprovalRef:  strings.TrimSpace(stringFromMap(record.Event, "approval_ref")),
			Reason:       strings.TrimSpace(stringFromMap(record.Event, "reason")),
			Explain:      strings.TrimSpace(stringFromMap(record.Event, "explain")),
			Stale:        boolFromMap(record.Event, "stale"),
		}}, true
	case ArtifactTypeKillSwitch:
		return SourceArtifact{KillSwitch: &KillSwitch{
			ArtifactBase:         base,
			State:                strings.TrimSpace(stringFromMap(record.Event, "state")),
			BlockedDispatchProof: strings.TrimSpace(stringFromMap(record.Event, "blocked_dispatch_proof")),
			Expiry:               strings.TrimSpace(stringFromMap(record.Event, "expiry")),
			Actor:                strings.TrimSpace(stringFromMap(record.Event, "actor")),
		}}, true
	case ArtifactTypeEnforcementExplain:
		return SourceArtifact{EnforcementExplain: &EnforcementExplain{
			ArtifactBase:      base,
			MissingFields:     stringSliceFromMap(record.Event, "missing_fields"),
			CredentialPosture: strings.TrimSpace(stringFromMap(record.Event, "credential_posture")),
			FreezeState:       strings.TrimSpace(stringFromMap(record.Event, "freeze_state")),
			KillSwitchState:   strings.TrimSpace(stringFromMap(record.Event, "kill_switch_state")),
			SandboxState:      strings.TrimSpace(stringFromMap(record.Event, "sandbox_state")),
			ReasonCodes:       stringSliceFromMap(record.Metadata, "gait_reason_codes"),
		}}, true
	case ArtifactTypeSandboxPolicy:
		return SourceArtifact{SandboxPolicy: &SandboxPolicy{
			ArtifactBase:        base,
			Path:                strings.TrimSpace(stringFromMap(record.Event, "path")),
			NetworkMode:         strings.TrimSpace(stringFromMap(record.Event, "network_mode")),
			WritablePaths:       stringSliceFromMap(record.Event, "writable_paths"),
			EnvExposure:         stringSliceFromMap(record.Event, "env_exposure"),
			TimeoutSeconds:      intFromMap(record.Event, "timeout_seconds"),
			FilesystemIsolation: strings.TrimSpace(stringFromMap(record.Event, "filesystem_isolation")),
			PolicyResult:        strings.TrimSpace(stringFromMap(record.Event, "policy_result")),
			GapReasonCodes:      stringSliceFromMap(record.Metadata, "gait_gap_reason_codes"),
		}}, true
	case ArtifactTypeTrustGraduation:
		return SourceArtifact{TrustGraduation: &TrustGraduation{
			ArtifactBase:            base,
			Path:                    strings.TrimSpace(stringFromMap(record.Event, "path")),
			CurrentStage:            strings.TrimSpace(stringFromMap(record.Event, "current_stage")),
			PriorStage:              strings.TrimSpace(stringFromMap(record.Event, "prior_stage")),
			PostureState:            strings.TrimSpace(stringFromMap(record.Event, "posture_state")),
			ApprovalFatigueSignals:  stringSliceFromMap(record.Metadata, "gait_approval_fatigue_signals"),
			RemainingGapReasonCodes: stringSliceFromMap(record.Metadata, "gait_gap_reason_codes"),
		}}, true
	default:
		return SourceArtifact{}, false
	}
}

func stringFromMap(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func stringSliceFromMap(values map[string]any, key string) []string {
	if values == nil {
		return nil
	}
	raw, ok := values[key]
	if !ok {
		return nil
	}
	items, ok := raw.([]any)
	if !ok {
		if typed, ok := raw.([]string); ok {
			return uniqueStrings(typed)
		}
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if value, ok := item.(string); ok {
			value = strings.TrimSpace(value)
			if value != "" {
				out = append(out, value)
			}
		}
	}
	return uniqueStrings(out)
}

func boolFromMap(values map[string]any, key string) bool {
	value, ok := optionalBoolFromMap(values, key)
	return ok && value
}

func optionalBoolFromMap(values map[string]any, key string) (bool, bool) {
	if values == nil {
		return false, false
	}
	value, ok := values[key]
	if !ok {
		return false, false
	}
	typed, ok := value.(bool)
	return typed, ok
}

func intFromMap(values map[string]any, key string) int {
	if values == nil {
		return 0
	}
	value, ok := values[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func rawString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return value
	}
	return ""
}

func normalizeVersion(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	switch trimmed {
	case "", "v1", "1", "1.0":
		if trimmed == "" {
			return ""
		}
		return "v1"
	default:
		return trimmed
	}
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

func filepathSlash(path string) string {
	return strings.ReplaceAll(strings.TrimSpace(path), "\\", "/")
}

func derivedArtifactID(bundleID string, kind string) string {
	return strings.Trim(strings.TrimSpace(bundleID)+":"+strings.TrimSpace(kind), ":")
}

func setString(values map[string]any, key string, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	values[key] = strings.TrimSpace(value)
}
