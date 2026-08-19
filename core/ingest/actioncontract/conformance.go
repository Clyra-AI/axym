package actioncontract

import "strings"

// CompareProposalActivation classifies only structural and binding
// conformance. It does not decide whether a policy is correct and does not
// claim that activation proves execution, effects, containment, or authority.
func CompareProposalActivation(proposal Proposal, activation Activation, proposalResult ValidationResult, activationResult ValidationResult) ConformanceResult {
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
		result.Valid = true
		add("context_only_non_binding")
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
	if len(activation.ExplicitExceptions) > 0 {
		result.Classification = ClassExplicitlyExcepted
		result.Valid = true
		result.Exceptions = sortedUnique(append([]string(nil), activation.ExplicitExceptions...))
		add("explicit_exception_present")
	} else if activation.ActivationMode == "enforce_floor" {
		result.Classification = ClassTightened
		result.Valid = true
		add("enforce_floor_preserves_required_fields")
	} else {
		result.Classification = ClassExact
		result.Valid = true
	}
	result.ReasonCodes = sortedUnique(result.ReasonCodes)
	return result
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
	field = strings.ToLower(strings.TrimSpace(field))
	for _, exception := range exceptions {
		if strings.Contains(strings.ToLower(exception), field) {
			return true
		}
	}
	return false
}
