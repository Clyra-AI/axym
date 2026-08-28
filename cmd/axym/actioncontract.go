package main

import (
	"crypto/ed25519"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/Clyra-AI/axym/core/ingest/actioncontract"
	"github.com/spf13/cobra"
)

func newActionContractCmd(stdout io.Writer, stderr io.Writer, global *globalFlags) *cobra.Command {
	var storeDir, activationPath, selectionPath, trustedKey, verifyAt string
	var allowDevelopmentSigning bool
	root := &cobra.Command{Use: "action-contract", Short: "Consume bounded Wrkr and Gait Action Contract artifacts"}
	consume := &cobra.Command{
		Use:   "consume <proposed_action_contract.json>",
		Short: "Consume exactly one Wrkr proposal without asserting authority",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return emitActionContractError(stdout, stderr, global, &actioncontract.ValidationError{Reasons: []string{actioncontract.ReasonSelectionRequired}})
			}
			proposal, err := actioncontract.ReadProposal(args[0])
			if err != nil {
				return emitActionContractError(stdout, stderr, global, err)
			}
			proposalEvaluation := time.Now().UTC()
			if strings.TrimSpace(verifyAt) != "" {
				proposalEvaluation, err = time.Parse(time.RFC3339Nano, verifyAt)
				if err != nil {
					return emitActionContractError(stdout, stderr, global, fmt.Errorf("verify-at must be RFC3339Nano: %w", err))
				}
			}
			validation := actioncontract.ValidateProposal(proposal, actioncontract.ValidationOptions{Now: proposalEvaluation})
			if !actioncontract.AcceptableSemanticProposal(validation) {
				return emitActionContractError(stdout, stderr, global, &actioncontract.ValidationError{Reasons: validation.ReasonCodes})
			}
			classification := actioncontract.ClassIncomplete
			if !validation.Valid {
				classification = actioncontract.ClassUnverifiable
				if containsActionContractReason(validation.ReasonCodes, actioncontract.ReasonProposalContradictory) {
					classification = actioncontract.ClassMismatched
				}
			}
			semantic := actioncontract.SemanticResult{ProposalValid: validation.Valid, Classification: classification, ReasonCodes: validation.ReasonCodes}
			var activationToPersist *actioncontract.Activation
			var conformanceToPersist *actioncontract.ConformanceResult
			var activationVerification *actioncontract.ActivationVerification
			if strings.TrimSpace(activationPath) != "" {
				activation, activationErr := actioncontract.ReadActivation(activationPath)
				if activationErr != nil {
					return emitActionContractError(stdout, stderr, global, activationErr)
				}
				var selection *actioncontract.SelectionEvidence
				if strings.TrimSpace(selectionPath) != "" {
					selected, selectionErr := actioncontract.ReadSelectionManifest(selectionPath, proposal)
					if selectionErr != nil {
						return emitActionContractError(stdout, stderr, global, selectionErr)
					}
					selection = &selected
				}
				var pub ed25519.PublicKey
				if strings.TrimSpace(trustedKey) != "" {
					loaded, keyErr := loadGovernanceKey(trustedKey)
					if keyErr != nil {
						return emitActionContractError(stdout, stderr, global, keyErr)
					}
					pub = loaded
				}
				when := time.Time{}
				if strings.TrimSpace(verifyAt) != "" {
					when = proposalEvaluation
				}
				activationResult := actioncontract.ValidateActivation(activation, actioncontract.ValidationOptions{Now: when, PublicKey: pub, Proposal: &proposal, Selection: selection, AllowDevelopmentSigning: allowDevelopmentSigning})
				if !activationResult.Valid {
					return emitActionContractError(stdout, stderr, global, &actioncontract.ValidationError{Reasons: activationResult.ReasonCodes})
				}
				conformance := actioncontract.CompareProposalActivation(proposal, activation, validation, activationResult)
				semantic.ActivationReady = conformance.Valid
				semantic.Classification = conformance.Classification
				semantic.ReasonCodes = actioncontract.SortedStrings(append(append([]string{}, validation.ReasonCodes...), append(activationResult.ReasonCodes, conformance.ReasonCodes...)...))
				activationToPersist, conformanceToPersist = &activation, &conformance
				activationVerification = &actioncontract.ActivationVerification{Result: activationResult, PublicKey: pub}
			}
			if strings.TrimSpace(storeDir) != "" {
				if _, err := actioncontract.PersistProposal(storeDir, proposal); err != nil {
					return &cliError{code: exitRuntimeFailure, msg: err.Error()}
				}
				if activationToPersist != nil {
					if activationVerification == nil {
						return emitActionContractError(stdout, stderr, global, fmt.Errorf("activation verification context is missing"))
					}
					if _, err := actioncontract.PersistActivation(storeDir, *activationToPersist, conformanceToPersist, *activationVerification); err != nil {
						return &cliError{code: exitRuntimeFailure, msg: err.Error()}
					}
				}
			}
			receipt := actioncontract.Receipt{
				Consumer: actioncontract.ConsumerName, Version: actioncontract.CurrentConsumerVersion(),
				ScenarioID: filepath.Base(filepath.Dir(args[0])), ArtifactSHA256: proposal.RawSHA256,
				Status: actioncontract.StatusPass, SelfAttestation: false,
				ProposalArtifactID: proposal.ArtifactID, ContractID: proposal.ContractID,
				ContractFamilyID: proposal.ContractFamilyID, Revision: proposal.Revision,
				ResolutionKey:   proposal.ResolutionKey,
				CorrelationRefs: actioncontract.SortedStrings(append(append(append([]string{}, proposal.SourceScanRefs...), proposal.CompositionRefs...), proposal.CreationEvidence...)),
				SchemaVersions:  map[string]string{"artifact": proposal.SchemaVersion, "contract": proposal.Producer.ContractSchemaVersion},
				SemanticResult:  semantic,
			}
			if global.JSON {
				_ = printJSON(stdout, envelope{OK: true, Command: "action-contract consume", Data: receipt})
			} else if !global.Quiet {
				printText(stdout, fmt.Sprintf("action contract %s: %s", receipt.ScenarioID, receipt.Status), global.Quiet)
			}
			return nil
		},
	}
	consume.Flags().StringVar(&storeDir, "store-dir", "", "Managed Axym store for exact producer artifact retention")
	consume.Flags().StringVar(&activationPath, "activation", "", "Optional exact Gait activation artifact")
	consume.Flags().StringVar(&selectionPath, "selection", "", "Optional exact current-selection manifest required for activation verification")
	consume.Flags().StringVar(&trustedKey, "trusted-key", "", "Trusted public-key artifact for activation verification")
	consume.Flags().StringVar(&verifyAt, "verify-at", "", "Verification time (RFC3339); required for activation")
	consume.Flags().BoolVar(&allowDevelopmentSigning, "allow-development-signing", false, "Allow fixture-only development signing (never authoritative)")
	root.AddCommand(consume)
	return root
}

func containsActionContractReason(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func emitActionContractError(stdout io.Writer, stderr io.Writer, global *globalFlags, err error) error {
	reasons := actioncontract.StableReasonCodes(err)
	code := exitInvalidInput
	if len(reasons) > 0 && reasons[0] != actioncontract.ReasonInputUnreadable && reasons[0] != actioncontract.ReasonSelectionRequired {
		code = exitVerificationFailed
	}
	if global.JSON {
		reason := actioncontract.ReasonInputUnreadable
		if len(reasons) > 0 {
			reason = reasons[0]
		}
		_ = printJSON(stdout, envelope{OK: false, Command: "action-contract consume", Error: &errorEnvelope{Reason: reason, Message: err.Error()}})
	} else if !global.Quiet {
		_, _ = fmt.Fprintln(stderr, err.Error())
	}
	return &cliError{code: code, msg: err.Error()}
}
