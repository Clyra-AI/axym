package actioncontract

import (
	"fmt"
	"sort"
	"strings"
)

// CompareProposalActivation classifies only structural and binding
// conformance. It does not decide whether a policy is correct and does not
// claim that activation proves execution, effects, containment, or authority.
func CompareProposalActivation(proposal Proposal, activation Activation, proposalResult ValidationResult, activationResult ValidationResult) ConformanceResult {
	return compareProposalActivation(proposal, activation, proposalResult, activationResult, nil)
}

// CompareProposalActivationWithProjection performs the same bounded
// comparison with an explicit control projection. Tightening is reported only
// when these actual values prove it; activation mode alone is insufficient.
func CompareProposalActivationWithProjection(proposal Proposal, activation Activation, proposalResult ValidationResult, activationResult ValidationResult, projection *ActivationProjection) ConformanceResult {
	return compareProposalActivation(proposal, activation, proposalResult, activationResult, projection)
}

func compareProposalActivation(proposal Proposal, activation Activation, proposalResult ValidationResult, activationResult ValidationResult, projection *ActivationProjection) ConformanceResult {
	result := ConformanceResult{Classification: ClassUnverifiable, Valid: false}
	add := func(reason string) { result.ReasonCodes = append(result.ReasonCodes, reason) }
	if proposalResult.Status == StatusInvalid || activationResult.Status == StatusInvalid {
		result.Classification = ClassUnverifiable
		add("validation_failed")
		result.ReasonCodes = sortedUnique(result.ReasonCodes)
		return result
	}
	if proposal.ArtifactID == "" || activation.ArtifactID == "" {
		result.Classification = ClassIncomplete
		add("artifact_id_missing")
		return result
	}
	result.CheckedFields = []string{"artifact_id", "canonical_content_digest", "contract_id", "contract_family_id", "revision", "schema_versions"}
	if activation.Proposal.ArtifactID != proposal.ArtifactID || activation.Proposal.CanonicalContentDigest != proposal.CanonicalContentDigest || activation.ContractID != proposal.ContractID || activation.ContractFamilyID != proposal.ContractFamilyID || activation.Revision != proposal.Revision {
		result.Classification = ClassMismatched
		add("proposal_binding_mismatch")
		result.ReasonCodes = sortedUnique(result.ReasonCodes)
		return result
	}
	if activation.ActivationMode == "context_only" {
		result.Classification = ClassContextual
		result.NonBinding = true
		result.Valid = proposalResult.Status == StatusPass && activationResult.Status == StatusPass
		add("context_only_non_binding")
		if !result.Valid {
			add("verification_unavailable")
		}
		result.ReasonCodes = sortedUnique(result.ReasonCodes)
		return result
	}
	if proposalResult.Status != StatusPass || activationResult.Status != StatusPass {
		result.Classification = ClassUnverifiable
		add("verification_unavailable")
		result.ReasonCodes = sortedUnique(result.ReasonCodes)
		return result
	}
	if activation.ActivationMode != "enforce_floor" && activation.ActivationMode != "required" {
		result.Classification = ClassIncomplete
		add("activation_mode_unrecognized")
		result.ReasonCodes = sortedUnique(result.ReasonCodes)
		return result
	}
	missing := requiredProposalFields(proposal)
	result.MissingFields = missing
	if len(missing) > 0 {
		for _, field := range missing {
			if !containsException(activation.ExplicitExceptions, field) {
				add("missing:" + field)
			}
		}
		if len(result.ReasonCodes) == 0 {
			result.Classification = ClassExplicitlyExcepted
			result.Valid = true
		} else {
			result.Classification = ClassIncomplete
		}
		result.ReasonCodes = sortedUnique(result.ReasonCodes)
		return result
	}
	if projection != nil {
		compareProjectedControls(&result, proposal, activation, projection)
		result.ReasonCodes = sortedUnique(result.ReasonCodes)
		return result
	}
	if unknown := unknownExceptionRefs(activation.ExplicitExceptions); len(unknown) > 0 {
		result.Classification = ClassMismatched
		add("unknown_exception_ref:" + unknown[0])
	} else if len(activation.ExplicitExceptions) > 0 {
		result.Classification = ClassExplicitlyExcepted
		result.Valid = true
		result.Exceptions = sortedUnique(append([]string(nil), activation.ExplicitExceptions...))
		add("explicit_exception_present")
	} else if activation.ActivationMode == "enforce_floor" {
		// The released Gait activation schema carries no independent control
		// projection. Binding the exact proposal proves preservation, but mode
		// alone cannot prove tightening, so this remains exact until an actual
		// value projection is supplied by a later producer artifact.
		result.Classification = ClassExact
		result.Valid = true
		add("enforce_floor_preserves_bound_proposal")
	} else {
		result.Classification = ClassExact
		result.Valid = true
	}
	result.ReasonCodes = sortedUnique(result.ReasonCodes)
	return result
}

func compareProjectedControls(result *ConformanceResult, proposal Proposal, activation Activation, projection *ActivationProjection) {
	proposalValues := proposalControlValues(proposal)
	actualValues := projectionControlValues(projection)
	result.CheckedFields = append(result.CheckedFields, "authority", "preconditions", "confirmation", "approval", "credential_mode", "delegation_depth", "expected_outcome", "forbidden_effects", "compensation", "validity")
	result.CheckedFields = sortedUnique(result.CheckedFields)
	unknown := unknownExceptionRefs(activation.ExplicitExceptions)
	if len(unknown) > 0 {
		result.Classification = ClassMismatched
		result.Valid = false
		result.ReasonCodes = append(result.ReasonCodes, "unknown_exception_ref:"+unknown[0])
		return
	}
	weakened := make([]string, 0)
	tightened := false
	for field, want := range proposalValues {
		got, ok := actualValues[field]
		if !ok {
			if containsException(activation.ExplicitExceptions, field) {
				result.Exceptions = append(result.Exceptions, field)
				continue
			}
			weakened = append(weakened, field)
			continue
		}
		if got == want {
			continue
		}
		if projectedValueTightens(field, want, got) {
			tightened = true
			continue
		}
		if containsException(activation.ExplicitExceptions, field) {
			result.Exceptions = append(result.Exceptions, field)
			continue
		}
		weakened = append(weakened, field)
	}
	if len(weakened) > 0 {
		sort.Strings(weakened)
		result.Classification = ClassWeakened
		result.Valid = false
		result.ReasonCodes = append(result.ReasonCodes, "weakened:"+weakened[0])
		return
	}
	if len(result.Exceptions) > 0 {
		result.Classification = ClassExplicitlyExcepted
		result.Valid = true
		result.Exceptions = sortedUnique(result.Exceptions)
		result.ReasonCodes = append(result.ReasonCodes, "explicit_exception_present")
		return
	}
	if tightened {
		result.Classification = ClassTightened
		result.Valid = true
		result.ReasonCodes = append(result.ReasonCodes, "actual_controls_tightened")
		return
	}
	result.Classification = ClassExact
	result.Valid = true
}

func proposalControlValues(proposal Proposal) map[string]string {
	contract := proposal.Contract
	values := map[string]string{
		"credential_mode":  stringField(contract, "required_credential_mode"),
		"delegation_depth": fmt.Sprint(intField(contract, "maximum_delegation_depth")),
		"expected_outcome": stringField(contract, "expected_outcome_class"),
		"compensation":     fmt.Sprint(boolField(contract, "compensation_required")),
		"validity":         stringField(contract, "expires_at"),
	}
	values["authority"] = joinRequirements(objectArray(contract, "authority_requirements"))
	values["preconditions"] = joinRequirements(objectArray(contract, "preconditions"))
	if item, ok := contract["confirmation_requirement"].(map[string]any); ok {
		values["confirmation"] = stringField(item, "mode") + "|" + fmt.Sprint(boolField(item, "required"))
	}
	if item, ok := contract["approval_requirement"].(map[string]any); ok {
		values["approval"] = fmt.Sprint(boolField(item, "required")) + "|" + fmt.Sprint(intField(item, "minimum_approvals"))
	}
	values["forbidden_effects"] = joinEffectRequirements(contract)
	return values
}

func projectionControlValues(projection *ActivationProjection) map[string]string {
	values := map[string]string{
		"authority":         strings.Join(sortedUnique(projection.AuthorityRefs), "|"),
		"preconditions":     strings.Join(sortedUnique(projection.Preconditions), "|"),
		"confirmation":      projection.ConfirmationRequirement,
		"approval":          projection.ApprovalRequirement,
		"credential_mode":   strings.TrimSpace(projection.CredentialMode),
		"delegation_depth":  fmt.Sprint(projection.DelegationDepth),
		"expected_outcome":  strings.TrimSpace(projection.ExpectedOutcomeClass),
		"forbidden_effects": strings.Join(sortedUnique(projection.ForbiddenEffects), "|"),
		"validity":          strings.TrimSpace(projection.ValidityNotAfter),
	}
	if projection.CompensationRequired != nil {
		values["compensation"] = fmt.Sprint(*projection.CompensationRequired)
	}
	return values
}

func joinRequirements(items []map[string]any) string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, stringField(item, "requirement_id")+"="+stringField(item, "required_constraint"))
	}
	return strings.Join(sortedUnique(values), "|")
}

func joinEffectRequirements(contract map[string]any) string {
	values := make([]string, 0)
	for _, item := range objectArray(contract, "preconditions") {
		kind := stringField(item, "kind")
		if kind == "forbidden_effect" || kind == "expected_effect" {
			values = append(values, stringField(item, "required_constraint"))
		}
	}
	return strings.Join(sortedUnique(values), "|")
}

func projectedValueTightens(field, want, got string) bool {
	switch field {
	case "credential_mode":
		return credentialStrength(got) > credentialStrength(want)
	case "delegation_depth":
		return intValue(got) < intValue(want)
	case "forbidden_effects", "authority", "preconditions":
		return isSuperset(got, want)
	case "approval":
		return approvalStrength(got) > approvalStrength(want)
	case "compensation":
		return want == "false" && got == "true"
	case "validity":
		if want == "" {
			return got != ""
		}
		wantTime, wantErr := parseUTCInstant(want)
		gotTime, gotErr := parseUTCInstant(got)
		return wantErr == nil && gotErr == nil && gotTime.Before(wantTime)
	default:
		return false
	}
}

func credentialStrength(value string) int {
	switch strings.TrimSpace(value) {
	case "ephemeral":
		return 3
	case "scoped":
		return 2
	case "standing_or_unknown":
		return 1
	default:
		return 0
	}
}

func intValue(value string) int {
	var parsed int
	_, _ = fmt.Sscan(value, &parsed)
	return parsed
}

func approvalStrength(value string) int {
	parts := strings.Split(value, "|")
	if len(parts) != 2 {
		return 0
	}
	strength := intValue(parts[1])
	if parts[0] == "true" {
		strength += 100
	}
	return strength
}

func isSuperset(got, want string) bool {
	wantSet := map[string]struct{}{}
	for _, item := range strings.Split(want, "|") {
		if strings.TrimSpace(item) != "" {
			wantSet[item] = struct{}{}
		}
	}
	gotSet := map[string]struct{}{}
	for _, item := range strings.Split(got, "|") {
		if strings.TrimSpace(item) != "" {
			gotSet[item] = struct{}{}
		}
	}
	for item := range wantSet {
		if _, ok := gotSet[item]; !ok {
			return false
		}
	}
	return len(gotSet) > len(wantSet)
}

func requiredProposalFields(proposal Proposal) []string {
	missing := make([]string, 0)
	contract := proposal.Contract
	if strings.TrimSpace(stringField(contract, "composition_ref")) == "" {
		missing = append(missing, "composition_ref")
	}
	if strings.TrimSpace(stringField(contract, "contract_content_digest")) == "" {
		missing = append(missing, "contract_content_digest")
	}
	if len(objectArray(contract, "preconditions")) == 0 {
		missing = append(missing, "preconditions")
	}
	if len(objectArray(contract, "authority_requirements")) == 0 {
		missing = append(missing, "authority_requirements")
	}
	if item, ok := contract["confirmation_requirement"].(map[string]any); !ok || strings.TrimSpace(stringField(item, "mode")) == "" {
		missing = append(missing, "confirmation_requirement")
	}
	return sortedUnique(missing)
}

func containsException(exceptions []string, field string) bool {
	field = normalizeExceptionRef(field)
	aliases := map[string]string{
		"authority": "authority_requirements", "confirmation": "confirmation_requirement",
		"approval": "approval_requirement", "compensation": "compensation_requirement",
	}
	accepted := map[string]struct{}{field: {}}
	if alias, ok := aliases[field]; ok {
		accepted[alias] = struct{}{}
	}
	for canonical, alias := range aliases {
		if field == alias {
			accepted[canonical] = struct{}{}
		}
	}
	for _, exception := range exceptions {
		if _, ok := accepted[normalizeExceptionRef(exception)]; ok {
			return true
		}
	}
	return false
}

func unknownExceptionRefs(exceptions []string) []string {
	known := map[string]struct{}{}
	for _, field := range []string{"authority", "authority_requirements", "preconditions", "confirmation", "confirmation_requirement", "approval", "approval_requirement", "credential_mode", "delegation_depth", "expected_outcome", "forbidden_effects", "compensation", "compensation_requirement", "validity", "composition_ref", "contract_content_digest"} {
		known[field] = struct{}{}
	}
	unknown := make([]string, 0)
	for _, exception := range sortedUnique(exceptions) {
		if _, ok := known[normalizeExceptionRef(exception)]; !ok {
			unknown = append(unknown, normalizeExceptionRef(exception))
		}
	}
	return unknown
}

func normalizeExceptionRef(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
