package main

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/Clyra-AI/axym/core/ingest/actioncontract"
	"github.com/spf13/cobra"
)

func newActionContractCmd(stdout io.Writer, stderr io.Writer, global *globalFlags) *cobra.Command {
	root := &cobra.Command{Use: "action-contract", Short: "Consume bounded Wrkr and Gait Action Contract artifacts"}
	consume := &cobra.Command{
		Use:   "consume <proposed_action_contract.json>",
		Short: "Consume exactly one Wrkr proposal without asserting authority",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return emitActionContractError(stdout, stderr, global, fmt.Errorf("exactly one proposed_action_contract path is required"))
			}
			proposal, err := actioncontract.ReadProposal(args[0])
			if err != nil {
				return emitActionContractError(stdout, stderr, global, err)
			}
			validation := actioncontract.ValidateProposal(proposal, actioncontract.ValidationOptions{})
			receipt := actioncontract.Receipt{
				Consumer: actioncontract.ConsumerName, Version: actioncontract.ConsumerVersion,
				ScenarioID: filepath.Base(filepath.Dir(args[0])), ArtifactSHA256: proposal.RawSHA256,
				Status: actioncontract.StatusPass, SelfAttestation: false,
				ProposalArtifactID: proposal.ArtifactID, ContractID: proposal.ContractID,
				ContractFamilyID: proposal.ContractFamilyID, Revision: proposal.Revision,
				ResolutionKey:   proposal.ResolutionKey,
				CorrelationRefs: actioncontract.SortedStrings(append(append(append([]string{}, proposal.SourceScanRefs...), proposal.CompositionRefs...), proposal.CreationEvidence...)),
				SchemaVersions:  map[string]string{"artifact": proposal.SchemaVersion, "contract": proposal.Producer.ContractSchemaVersion},
				SemanticResult:  actioncontract.SemanticResult{ProposalValid: validation.Valid, Classification: actioncontract.ClassIncomplete, ReasonCodes: validation.ReasonCodes},
			}
			if !validation.Valid {
				receipt.Status = actioncontract.StatusInvalid
			}
			if global.JSON {
				_ = printJSON(stdout, envelope{OK: validation.Valid, Command: "action-contract consume", Data: receipt})
			} else if !global.Quiet {
				printText(stdout, fmt.Sprintf("action contract %s: %s", receipt.ScenarioID, receipt.Status), global.Quiet)
			}
			if !validation.Valid {
				return &cliError{code: exitVerificationFailed, msg: "action contract validation failed"}
			}
			return nil
		},
	}
	root.AddCommand(consume)
	return root
}

func emitActionContractError(stdout io.Writer, stderr io.Writer, global *globalFlags, err error) error {
	if global.JSON {
		_ = printJSON(stdout, envelope{OK: false, Command: "action-contract consume", Error: &errorEnvelope{Reason: "invalid_input", Message: err.Error()}})
	} else if !global.Quiet {
		_, _ = fmt.Fprintln(stderr, err.Error())
	}
	return &cliError{code: exitInvalidInput, msg: err.Error()}
}
