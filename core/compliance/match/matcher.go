package match

import (
	"fmt"
	"sort"
	"strings"
	"time"

	compliancecontext "github.com/Clyra-AI/axym/core/compliance/context"
	"github.com/Clyra-AI/axym/core/compliance/framework"
	"github.com/Clyra-AI/axym/core/compliance/threshold"
	"github.com/Clyra-AI/axym/core/normalize"
	"github.com/Clyra-AI/proof"
)

const (
	RecordStatusMatched = "matched"
	RecordStatusPartial = "partial"
	RecordStatusFailed  = "failed"

	ControlStatusCovered = "covered"
	ControlStatusPartial = "partial"
	ControlStatusGap     = "gap"

	ReasonMissingActorLinkage      = "MISSING_ACTOR_LINKAGE"
	ReasonMissingDownstreamLinkage = "MISSING_DOWNSTREAM_LINKAGE"
	ReasonMissingOwnerLinkage      = "MISSING_OWNER_LINKAGE"
	ReasonMissingTargetLinkage     = "MISSING_TARGET_LINKAGE"
	ReasonMissingPolicyBinding     = "MISSING_POLICY_BINDING"
	ReasonMissingApprovalBinding   = "MISSING_APPROVAL_BINDING"
	ReasonIncompleteDelegation     = "INCOMPLETE_DELEGATION_CHAIN"
	ReasonUnapprovedPrivilegeDrift = "UNAPPROVED_PRIVILEGE_DRIFT"
)

type Options struct {
	ExcludeInvalidEvidence bool
}

type Result struct {
	Frameworks []FrameworkResult `json:"frameworks"`
	Summary    Summary           `json:"summary"`
}

type Summary struct {
	FrameworkCount int `json:"framework_count"`
	ControlCount   int `json:"control_count"`
	CoveredCount   int `json:"covered_count"`
	PartialCount   int `json:"partial_count"`
	GapCount       int `json:"gap_count"`
}

type FrameworkResult struct {
	ID            string          `json:"id"`
	Version       string          `json:"version"`
	Title         string          `json:"title"`
	Controls      []ControlResult `json:"controls"`
	CoverageRatio float64         `json:"coverage_ratio"`
}

type ControlResult struct {
	FrameworkID         string        `json:"framework_id"`
	ControlID           string        `json:"control_id"`
	Title               string        `json:"title"`
	RequiredRecordTypes []string      `json:"required_record_types"`
	RequiredFields      []string      `json:"required_fields"`
	MinimumFrequency    string        `json:"minimum_frequency"`
	Status              string        `json:"status"`
	CandidateCount      int           `json:"candidate_count"`
	MatchedCount        int           `json:"matched_count"`
	PartialCount        int           `json:"partial_count"`
	FailedCount         int           `json:"failed_count"`
	InvalidExcluded     int           `json:"invalid_excluded"`
	ReasonCodes         []string      `json:"reason_codes"`
	Rationale           string        `json:"rationale"`
	Evidence            []RecordMatch `json:"evidence"`
}

type RecordMatch struct {
	RecordID    string                            `json:"record_id"`
	RecordType  string                            `json:"record_type"`
	Timestamp   string                            `json:"timestamp"`
	Status      string                            `json:"status"`
	Matched     []string                          `json:"matched_fields"`
	Missing     []string                          `json:"missing_fields,omitempty"`
	ReasonCodes []string                          `json:"reason_codes"`
	Context     compliancecontext.EvidenceContext `json:"context"`
	Weights     compliancecontext.Weights         `json:"weights"`
}

func Evaluate(definitions []framework.Definition, records []proof.Record, opts Options) Result {
	sortedRecords := sortedRecords(records)
	frameworkResults := make([]FrameworkResult, 0, len(definitions))
	summary := Summary{FrameworkCount: len(definitions)}

	for _, definition := range definitions {
		frameworkResult := FrameworkResult{
			ID:      definition.ID,
			Version: definition.Version,
			Title:   definition.Title,
		}
		frameworkResult.Controls = make([]ControlResult, 0, len(definition.Controls))
		covered := 0
		for _, control := range definition.Controls {
			controlResult := evaluateControl(control, sortedRecords, opts)
			frameworkResult.Controls = append(frameworkResult.Controls, controlResult)
			summary.ControlCount++
			switch controlResult.Status {
			case ControlStatusCovered:
				summary.CoveredCount++
				covered++
			case ControlStatusPartial:
				summary.PartialCount++
			default:
				summary.GapCount++
			}
		}
		if len(frameworkResult.Controls) > 0 {
			frameworkResult.CoverageRatio = roundRatio(float64(covered) / float64(len(frameworkResult.Controls)))
		}
		frameworkResults = append(frameworkResults, frameworkResult)
	}

	sort.Slice(frameworkResults, func(i, j int) bool {
		return frameworkResults[i].ID < frameworkResults[j].ID
	})
	return Result{Frameworks: frameworkResults, Summary: summary}
}

func evaluateControl(control framework.Control, records []proof.Record, opts Options) ControlResult {
	requiredFields := uniqueSorted(control.RequiredFields)
	requiredTypes := uniqueSorted(control.RequiredRecordTypes)
	requiredTypeSet := make(map[string]struct{}, len(requiredTypes))
	for _, recordType := range requiredTypes {
		requiredTypeSet[recordType] = struct{}{}
	}

	result := ControlResult{
		FrameworkID:         control.FrameworkID,
		ControlID:           control.ID,
		Title:               control.Title,
		RequiredRecordTypes: requiredTypes,
		RequiredFields:      requiredFields,
		MinimumFrequency:    control.MinimumFrequency,
		ReasonCodes:         []string{},
		Evidence:            []RecordMatch{},
	}

	matchedInFrequencyWindow := 0
	windowStart, hasWindow := frequencyWindowStart(control.MinimumFrequency, records)

	for _, record := range records {
		recordType := strings.ToLower(strings.TrimSpace(record.RecordType))
		if _, ok := requiredTypeSet[recordType]; !ok {
			continue
		}
		result.CandidateCount++
		matchedFields, missingFields := evaluateFields(record, requiredFields)

		if opts.ExcludeInvalidEvidence {
			if invalid, invalidCode := threshold.IsInvalidEvidenceClass(record); invalid {
				result.InvalidExcluded++
				result.FailedCount++
				result.Evidence = append(result.Evidence, buildRecordMatch(record, matchedFields, missingFields, RecordStatusFailed, []string{"INVALID_EVIDENCE", invalidCode}))
				continue
			}
		}

		identityMissing, identityReasons := IdentityWeaknesses(record)
		missingFields = uniqueSorted(append(missingFields, identityMissing...))

		recordStatus := RecordStatusFailed
		reasons := append([]string{"MISSING_REQUIRED_FIELDS"}, identityReasons...)
		switch {
		case len(missingFields) == 0:
			recordStatus = RecordStatusMatched
			reasons = []string{"MATCHED"}
			result.MatchedCount++
			if !hasWindow || !record.Timestamp.Before(windowStart) {
				matchedInFrequencyWindow++
			}
		case len(matchedFields) > 0:
			recordStatus = RecordStatusPartial
			reasons = append([]string{"PARTIAL_FIELD_MATCH"}, identityReasons...)
			result.PartialCount++
		default:
			reasons = append(reasons, identityReasons...)
			result.FailedCount++
		}
		result.Evidence = append(result.Evidence, buildRecordMatch(record, matchedFields, missingFields, recordStatus, reasons))
	}

	reasons := make([]string, 0, 3)
	requiredMatches := minimumMatches(control.MinimumFrequency)
	frequencyMet := matchedInFrequencyWindow >= requiredMatches

	switch {
	case result.MatchedCount > 0 && frequencyMet:
		result.Status = ControlStatusCovered
		reasons = append(reasons, "CONTROL_COVERED")
	case result.MatchedCount > 0 || result.PartialCount > 0:
		result.Status = ControlStatusPartial
		reasons = append(reasons, "CONTROL_PARTIAL")
	default:
		result.Status = ControlStatusGap
		reasons = append(reasons, "CONTROL_GAP")
	}
	if !frequencyMet {
		reasons = append(reasons, "FREQUENCY_NOT_MET")
	}
	if result.CandidateCount == 0 {
		reasons = append(reasons, "NO_RECORD_TYPE_MATCH")
	}
	if result.InvalidExcluded > 0 {
		reasons = append(reasons, threshold.ReasonInvalidEvidence)
	}
	for _, evidence := range result.Evidence {
		for _, reason := range evidence.ReasonCodes {
			if isIdentityReasonCode(reason) {
				reasons = append(reasons, reason)
			}
		}
	}
	result.ReasonCodes = uniqueSorted(reasons)
	result.Rationale = buildRationale(result, requiredMatches, matchedInFrequencyWindow)
	return result
}

func buildRecordMatch(record proof.Record, matched []string, missing []string, status string, reasons []string) RecordMatch {
	ctx := compliancecontext.Enrich(record)
	weights := compliancecontext.Score(ctx)
	matchedFields := uniqueSorted(matched)
	missingFields := uniqueSorted(missing)
	return RecordMatch{
		RecordID:    strings.TrimSpace(record.RecordID),
		RecordType:  strings.ToLower(strings.TrimSpace(record.RecordType)),
		Timestamp:   record.Timestamp.UTC().Format(time.RFC3339),
		Status:      status,
		Matched:     matchedFields,
		Missing:     missingFields,
		ReasonCodes: uniqueSorted(reasons),
		Context:     ctx,
		Weights:     weights,
	}
}

func buildRationale(result ControlResult, requiredMatches int, matchedInWindow int) string {
	return fmt.Sprintf(
		"status=%s matched=%d partial=%d failed=%d candidates=%d required_matches=%d matched_in_window=%d invalid_excluded=%d",
		result.Status,
		result.MatchedCount,
		result.PartialCount,
		result.FailedCount,
		result.CandidateCount,
		requiredMatches,
		matchedInWindow,
		result.InvalidExcluded,
	)
}

func evaluateFields(record proof.Record, required []string) ([]string, []string) {
	matched := make([]string, 0, len(required))
	missing := make([]string, 0, len(required))
	for _, field := range required {
		if recordFieldExists(record, field) {
			matched = append(matched, field)
			continue
		}
		missing = append(missing, field)
	}
	return matched, missing
}

func recordFieldExists(record proof.Record, field string) bool {
	field = strings.TrimSpace(field)
	if field == "" {
		return false
	}
	switch field {
	case "record_id":
		return strings.TrimSpace(record.RecordID) != ""
	case "timestamp":
		return !record.Timestamp.IsZero()
	case "source":
		return strings.TrimSpace(record.Source) != ""
	case "source_product":
		return strings.TrimSpace(record.SourceProduct) != ""
	case "record_type":
		return strings.TrimSpace(record.RecordType) != ""
	case "event":
		return record.Event != nil
	case "metadata":
		return record.Metadata != nil
	}

	segments := strings.Split(field, ".")
	if len(segments) == 0 {
		return false
	}
	head := strings.ToLower(strings.TrimSpace(segments[0]))
	rest := segments[1:]

	switch head {
	case "event":
		if len(rest) == 1 {
			if exists, ok := normalizedIdentityFieldExists(record, rest[0]); ok {
				return exists
			}
		}
		return nestedMapKeyExists(record.Event, rest)
	case "metadata":
		return nestedMapKeyExists(record.Metadata, rest)
	case "integrity":
		if len(rest) == 1 && rest[0] == "record_hash" {
			return strings.TrimSpace(record.Integrity.RecordHash) != ""
		}
		if len(rest) == 1 && rest[0] == "previous_record_hash" {
			return strings.TrimSpace(record.Integrity.PreviousRecordHash) != ""
		}
		if len(rest) == 1 && rest[0] == "signing_key_id" {
			return strings.TrimSpace(record.Integrity.SigningKeyID) != ""
		}
		if len(rest) == 1 && rest[0] == "signature" {
			return strings.TrimSpace(record.Integrity.Signature) != ""
		}
		return false
	default:
		return false
	}
}

func nestedMapKeyExists(container map[string]any, path []string) bool {
	if len(path) == 0 {
		return container != nil
	}
	if container == nil {
		return false
	}
	current := any(container)
	for _, segment := range path {
		key := strings.TrimSpace(segment)
		switch node := current.(type) {
		case map[string]any:
			value, ok := node[key]
			if !ok {
				return false
			}
			current = value
		default:
			return false
		}
	}
	if current == nil {
		return false
	}
	if value, ok := current.(string); ok {
		return strings.TrimSpace(value) != ""
	}
	return true
}

func minimumMatches(minimumFrequency string) int {
	trimmed := strings.ToLower(strings.TrimSpace(minimumFrequency))
	switch trimmed {
	case "", "continuous", "per-event", "daily", "weekly", "monthly", "quarterly":
		return 1
	default:
		return 1
	}
}

func frequencyWindowStart(minimumFrequency string, records []proof.Record) (time.Time, bool) {
	if len(records) == 0 {
		return time.Time{}, false
	}
	latest := records[len(records)-1].Timestamp.UTC()
	switch strings.ToLower(strings.TrimSpace(minimumFrequency)) {
	case "quarterly":
		return latest.AddDate(0, -3, 0), true
	case "monthly":
		return latest.AddDate(0, -1, 0), true
	case "weekly":
		return latest.AddDate(0, 0, -7), true
	case "daily":
		return latest.Add(-24 * time.Hour), true
	default:
		return time.Time{}, false
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

func uniqueSorted(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(in))
	for _, item := range in {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	result := make([]string, 0, len(out))
	for i, item := range out {
		if i == 0 || item != out[i-1] {
			result = append(result, item)
		}
	}
	return result
}

func roundRatio(in float64) float64 {
	if in <= 0 {
		return 0
	}
	if in >= 1 {
		return 1
	}
	return float64(int(in*10000+0.5)) / 10000
}

func IdentityWeaknesses(record proof.Record) ([]string, []string) {
	recordType := strings.ToLower(strings.TrimSpace(record.RecordType))
	if !identityGovernedRecordType(recordType) {
		return nil, nil
	}
	view := normalize.IdentityViewFromRecord(&record)
	missing := make([]string, 0, 8)
	reasons := make([]string, 0, 8)

	if strings.TrimSpace(view.ActorIdentity) == "" {
		missing = append(missing, "event.actor_identity")
		reasons = append(reasons, ReasonMissingActorLinkage)
	}
	if strings.TrimSpace(view.DownstreamIdentity) == "" {
		missing = append(missing, "event.downstream_identity")
		reasons = append(reasons, ReasonMissingDownstreamLinkage)
	}
	if strings.TrimSpace(view.TargetKind) == "" {
		missing = append(missing, "event.target_kind")
	}
	if strings.TrimSpace(view.TargetID) == "" {
		missing = append(missing, "event.target_id")
	}
	if strings.TrimSpace(view.TargetKind) == "" || strings.TrimSpace(view.TargetID) == "" {
		reasons = append(reasons, ReasonMissingTargetLinkage)
	}
	if len(view.DelegationChain) == 0 {
		missing = append(missing, "event.delegation_chain")
		reasons = append(reasons, ReasonIncompleteDelegation)
	}
	if requiresOwnerLinkage(recordType) && strings.TrimSpace(view.OwnerIdentity) == "" {
		missing = append(missing, "event.owner_identity")
		reasons = append(reasons, ReasonMissingOwnerLinkage)
	}
	if requiresPolicyBinding(recordType) && strings.TrimSpace(view.PolicyDigest) == "" {
		missing = append(missing, "event.policy_digest")
		reasons = append(reasons, ReasonMissingPolicyBinding)
	}
	if requiresApprovalBinding(record, view) && strings.TrimSpace(view.ApprovalTokenRef) == "" {
		missing = append(missing, "event.approval_token_ref")
		reasons = append(reasons, ReasonMissingApprovalBinding)
	}
	if unapprovedPrivilegeDrift(record, view) {
		reasons = append(reasons, ReasonUnapprovedPrivilegeDrift)
	}
	return uniqueSorted(missing), uniqueSorted(reasons)
}

func normalizedIdentityFieldExists(record proof.Record, field string) (bool, bool) {
	view := normalize.IdentityViewFromRecord(&record)
	switch strings.TrimSpace(field) {
	case "actor_identity":
		return strings.TrimSpace(view.ActorIdentity) != "", true
	case "downstream_identity":
		return strings.TrimSpace(view.DownstreamIdentity) != "", true
	case "owner_identity":
		return strings.TrimSpace(view.OwnerIdentity) != "", true
	case "policy_digest":
		return strings.TrimSpace(view.PolicyDigest) != "", true
	case "approval_token_ref":
		return strings.TrimSpace(view.ApprovalTokenRef) != "", true
	case "target_kind":
		return strings.TrimSpace(view.TargetKind) != "", true
	case "target_id":
		return strings.TrimSpace(view.TargetID) != "", true
	case "delegation_chain":
		return len(view.DelegationChain) > 0, true
	default:
		return false, false
	}
}

func identityGovernedRecordType(recordType string) bool {
	switch recordType {
	case "tool_invocation", "decision", "policy_enforcement", "approval", "scan_finding", "deployment", "data_pipeline_run", "permission_check", "compiled_action":
		return true
	default:
		return false
	}
}

func requiresOwnerLinkage(recordType string) bool {
	switch recordType {
	case "decision", "policy_enforcement", "approval", "scan_finding", "deployment", "data_pipeline_run", "permission_check", "compiled_action":
		return true
	default:
		return false
	}
}

func requiresPolicyBinding(recordType string) bool {
	switch recordType {
	case "decision", "policy_enforcement", "approval", "scan_finding", "deployment", "data_pipeline_run", "permission_check", "compiled_action":
		return true
	default:
		return false
	}
}

func requiresApprovalBinding(record proof.Record, view normalize.IdentityView) bool {
	recordType := strings.ToLower(strings.TrimSpace(record.RecordType))
	if recordType == "approval" {
		return true
	}
	if value := strings.TrimSpace(normalizeFieldString(record.Event, "approval_token_ref")); value != "" {
		return true
	}
	if value := strings.TrimSpace(normalizeFieldString(record.Event, "context_approval_ref")); value != "" {
		return true
	}
	if value := strings.TrimSpace(normalizeFieldString(record.Event, "approval_ref")); value != "" {
		return true
	}
	if record.Controls.HumanOversight != nil && record.Controls.HumanOversight.ApprovalRequired {
		return true
	}
	return unapprovedPrivilegeDrift(record, view)
}

func unapprovedPrivilegeDrift(record proof.Record, view normalize.IdentityView) bool {
	if strings.ToLower(strings.TrimSpace(record.RecordType)) != "scan_finding" {
		return false
	}
	if strings.TrimSpace(view.TargetKind) != "privilege" && strings.TrimSpace(normalizeFieldString(record.Event, "privilege")) == "" {
		return false
	}
	approved, ok := normalizeFieldBool(record.Event, "approved")
	if !ok {
		approved, ok = normalizeFieldBool(record.Metadata, "approved")
	}
	return ok && !approved && strings.TrimSpace(view.ApprovalTokenRef) == ""
}

func normalizeFieldString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	value, _ := m[key].(string)
	return strings.TrimSpace(value)
}

func normalizeFieldBool(m map[string]any, key string) (bool, bool) {
	if m == nil {
		return false, false
	}
	value, ok := m[key].(bool)
	return value, ok
}

func isIdentityReasonCode(reason string) bool {
	switch strings.TrimSpace(reason) {
	case ReasonMissingActorLinkage,
		ReasonMissingDownstreamLinkage,
		ReasonMissingOwnerLinkage,
		ReasonMissingTargetLinkage,
		ReasonMissingPolicyBinding,
		ReasonMissingApprovalBinding,
		ReasonIncompleteDelegation,
		ReasonUnapprovedPrivilegeDrift:
		return true
	default:
		return false
	}
}
