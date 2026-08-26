package actioncontract

import "testing"

func TestConformanceClassificationMatrix(t *testing.T) {
	proposal, err := ReadProposal("../../../testdata/action-contract-interop/v1/expected/package-to-release/pac-0d9384785d3b213a.json")
	if err != nil {
		t.Fatal(err)
	}
	proposalResult := ValidateProposal(proposal, ValidationOptions{})
	activation, _ := makeTestActivation(t, proposal, "context_only", nil)
	activationResult := ValidationResult{Status: StatusPass, Valid: true}
	if got := CompareProposalActivation(proposal, activation, proposalResult, activationResult); got.Classification != ClassContextual || !got.NonBinding {
		t.Fatalf("contextual=%+v", got)
	}
	activation.ContractID = "pac-ffffffff"
	if got := CompareProposalActivation(proposal, activation, proposalResult, activationResult); got.Classification != ClassMismatched {
		t.Fatalf("mismatched=%+v", got)
	}
	activation.ContractID = proposal.ContractID
	activation.ActivationMode = "unsupported"
	if got := CompareProposalActivation(proposal, activation, proposalResult, activationResult); got.Classification != ClassIncomplete {
		t.Fatalf("incomplete=%+v", got)
	}
	if got := CompareProposalActivation(proposal, activation, proposalResult, ValidationResult{Status: StatusInvalid}); got.Classification != ClassUnverifiable {
		t.Fatalf("unverifiable=%+v", got)
	}
	activation.ActivationMode = "enforce_floor"
	activation.ExplicitExceptions = []string{"composition_ref"}
	if got := CompareProposalActivation(proposal, activation, proposalResult, activationResult); got.Classification != ClassExplicitlyExcepted {
		t.Fatalf("excepted=%+v", got)
	}
}
