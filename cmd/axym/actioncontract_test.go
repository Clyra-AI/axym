package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Clyra-AI/axym/core/ingest/actioncontract"
)

func TestActionContractConsumeJSONReceiptAndExitContract(t *testing.T) {
	fixture := filepath.Join("..", "..", "testdata", "action-contract-interop", "v1", "expected", "customer-data-to-egress", "pac-6dcee5a6d9a65e8c.json")
	var stdout, stderr bytes.Buffer
	if exit := execute([]string{"--json", "action-contract", "consume", fixture}, &stdout, &stderr); exit != exitSuccess {
		t.Fatalf("consume exit=%d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}
	var envelope struct {
		OK   bool                   `json:"ok"`
		Data actioncontract.Receipt `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.OK || envelope.Data.Consumer != actioncontract.ConsumerName || envelope.Data.Status != actioncontract.StatusPass || envelope.Data.SelfAttestation || envelope.Data.SemanticResult.ExecutionClaim || envelope.Data.SemanticResult.EffectClaim {
		t.Fatalf("unexpected consumer receipt: %s", stdout.String())
	}
	if len(envelope.Data.CorrelationRefs) == 0 || envelope.Data.ProposalArtifactID == "" || envelope.Data.ContractID == "" {
		t.Fatalf("receipt did not preserve proposal identity and refs: %s", stdout.String())
	}
}

func TestActionContractConsumeRejectsZeroOrMultiplePaths(t *testing.T) {
	for _, args := range [][]string{
		{"--json", "action-contract", "consume"},
		{"--json", "action-contract", "consume", "one.json", "two.json"},
	} {
		var stdout, stderr bytes.Buffer
		if exit := execute(args, &stdout, &stderr); exit != exitInvalidInput {
			t.Fatalf("args=%v exit=%d want=%d stdout=%s stderr=%s", args, exit, exitInvalidInput, stdout.String(), stderr.String())
		}
	}
}

func TestActionContractConsumeKeepsArtifactBytesUnchanged(t *testing.T) {
	fixture := filepath.Join("..", "..", "testdata", "action-contract-interop", "v1", "expected", "customer-data-to-egress", "pac-6dcee5a6d9a65e8c.json")
	before, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if exit := execute([]string{"--json", "action-contract", "consume", fixture}, &stdout, &stderr); exit != exitSuccess {
		t.Fatalf("consume exit=%d stderr=%s", exit, stderr.String())
	}
	after, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("consumer modified producer-native artifact bytes")
	}
}
