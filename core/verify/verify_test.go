package verify

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Clyra-AI/proof"
)

func TestVerifyChainAgreementWithProof(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	if err := os.MkdirAll(storeDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	chain := proof.NewChain("agreement")
	record, err := proof.NewRecord(proof.RecordOpts{
		Source:        "axym",
		SourceProduct: "axym",
		Type:          "tool_invocation",
		Timestamp:     time.Date(2026, 2, 28, 16, 0, 0, 0, time.UTC),
		Event:         map[string]any{"tool_name": "fetch"},
		Controls:      proof.Controls{PermissionsEnforced: true},
	})
	if err != nil {
		t.Fatalf("NewRecord: %v", err)
	}
	if err := proof.AppendToChain(chain, record); err != nil {
		t.Fatalf("AppendToChain: %v", err)
	}
	raw, err := json.MarshalIndent(chain, "", "  ")
	if err != nil {
		t.Fatalf("marshal chain: %v", err)
	}
	if err := os.WriteFile(filepath.Join(storeDir, "chain.json"), raw, 0o600); err != nil {
		t.Fatalf("write chain: %v", err)
	}

	coreResult, err := VerifyChainFromStoreDir(storeDir)
	if err != nil {
		t.Fatalf("VerifyChainFromStoreDir: %v", err)
	}
	proofResult, err := proof.VerifyChain(chain)
	if err != nil {
		t.Fatalf("proof.VerifyChain: %v", err)
	}
	if coreResult.Intact != proofResult.Intact || coreResult.Count != proofResult.Count {
		t.Fatalf("mismatch core vs proof: core=%+v proof=%+v", coreResult, proofResult)
	}

	chain.Records[0].Event["tool_name"] = "tampered"
	raw, err = json.MarshalIndent(chain, "", "  ")
	if err != nil {
		t.Fatalf("marshal tampered chain: %v", err)
	}
	if err := os.WriteFile(filepath.Join(storeDir, "chain.json"), raw, 0o600); err != nil {
		t.Fatalf("write tampered chain: %v", err)
	}

	_, coreErr := VerifyChainFromStoreDir(storeDir)
	if coreErr == nil {
		t.Fatal("expected tampered chain error")
	}
	var verifyErr *Error
	if !errors.As(coreErr, &verifyErr) {
		t.Fatalf("expected verify.Error, got %T", coreErr)
	}
	proofResult, err = proof.VerifyChain(chain)
	if err != nil {
		t.Fatalf("proof.VerifyChain tampered: %v", err)
	}
	if proofResult.Intact {
		t.Fatal("proof.VerifyChain should fail on tampered chain")
	}
}
