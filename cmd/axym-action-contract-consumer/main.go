// Command axym-action-contract-consumer is the executable entrypoint used by
// the Wrkr Action Contract conformance harness. It consumes exactly one
// producer artifact and emits a deterministic, non-authoritative receipt.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Clyra-AI/axym/core/ingest/actioncontract"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] == "" {
		fmt.Fprintln(os.Stderr, "usage: axym-action-contract-consumer <one-proposed-action-contract.json>")
		os.Exit(6)
	}
	path := os.Args[1]
	proposal, err := actioncontract.ReadProposal(path)
	if err != nil {
		artifactDigest := actioncontract.RawDigest(nil)
		if info, statErr := os.Lstat(path); statErr == nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 {
			// #nosec G304 -- explicit consumer path.
			if raw, readErr := os.ReadFile(path); readErr == nil {
				artifactDigest = actioncontract.RawDigest(raw)
			}
		}
		writeReceipt(actioncontract.Receipt{Consumer: actioncontract.ConsumerName, Version: actioncontract.ConsumerVersion, ScenarioID: filepath.Base(filepath.Dir(path)), ArtifactSHA256: artifactDigest, Status: actioncontract.StatusInvalid, SelfAttestation: false, SchemaVersions: map[string]string{"artifact": actioncontract.ProposalSchemaVersion, "contract": actioncontract.ProposalContractVersion}, SemanticResult: actioncontract.SemanticResult{Classification: actioncontract.ClassUnverifiable, ReasonCodes: []string{err.Error()}, ExecutionClaim: false, EffectClaim: false}})
		os.Exit(2)
	}
	validation := actioncontract.ValidateProposal(proposal, actioncontract.ValidationOptions{})
	status := actioncontract.StatusPass
	if !validation.Valid {
		status = actioncontract.StatusInvalid
	}
	correlationRefs := append([]string(nil), proposal.SourceScanRefs...)
	correlationRefs = append(correlationRefs, proposal.CompositionRefs...)
	correlationRefs = append(correlationRefs, proposal.CreationEvidence...)
	receipt := actioncontract.Receipt{
		Consumer:           actioncontract.ConsumerName,
		Version:            actioncontract.ConsumerVersion,
		ScenarioID:         filepath.Base(filepath.Dir(path)),
		ArtifactSHA256:     proposal.RawSHA256,
		Status:             status,
		SelfAttestation:    false,
		ProposalArtifactID: proposal.ArtifactID,
		ContractID:         proposal.ContractID,
		ContractFamilyID:   proposal.ContractFamilyID,
		Revision:           proposal.Revision,
		ResolutionKey:      proposal.ResolutionKey,
		CorrelationRefs:    actioncontract.SortedStrings(correlationRefs),
		SchemaVersions:     map[string]string{"artifact": proposal.SchemaVersion, "contract": proposal.Producer.ContractSchemaVersion},
		SemanticResult:     actioncontract.SemanticResult{ProposalValid: validation.Valid, ActivationReady: false, Classification: actioncontract.ClassIncomplete, ReasonCodes: validation.ReasonCodes, ExecutionClaim: false, EffectClaim: false},
	}
	writeReceipt(receipt)
	if !validation.Valid {
		os.Exit(2)
	}
}

func writeReceipt(receipt actioncontract.Receipt) {
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_, _ = os.Stdout.Write(append(encoded, '\n'))
}
