package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	coregovernance "github.com/Clyra-AI/axym/core/governance"
	"github.com/Clyra-AI/axym/core/ingest/actioncontract"
	"github.com/Clyra-AI/axym/core/verifysupport"
	"github.com/Clyra-AI/proof"
)

func TestGovernanceEmitRejectsMalformedVerifyAtForBoundaryAndJudge(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keyRaw, err := verifysupport.MarshalBundlePublicKey(structuredPublicKey(pub))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	keyPath, inputPath := filepath.Join(root, "key.json"), filepath.Join(root, "input.json")
	if err = os.WriteFile(keyPath, keyRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"boundary", "judge"} {
		if err = os.WriteFile(inputPath, []byte(`{}`), 0o600); err != nil {
			t.Fatal(err)
		}
		var out, stderr bytes.Buffer
		code := execute([]string{"governance", "emit", "--kind", kind, "--input", inputPath, "--trusted-key", keyPath, "--store-dir", filepath.Join(root, kind), "--verify-at", "not-a-time", "--json"}, &out, &stderr)
		if code != exitInvalidInput {
			t.Fatalf("%s verify-at code=%d out=%s err=%s", kind, code, out.String(), stderr.String())
		}
		var envelope map[string]any
		if err = json.Unmarshal(out.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope["ok"] != false {
			t.Fatalf("%s malformed verify-at accepted: %s", kind, out.String())
		}
	}
}

func TestGovernanceEmitTelemetryVerificationFailureUsesExitTwo(t *testing.T) {
	now := time.Now().UTC()
	span := coregovernance.TraceSpan{TraceID: "0123456789abcdef0123456789abcdef", SpanID: "0123456789abcdef", StartTime: now.Add(-time.Second).Format(time.RFC3339Nano), EndTime: now.Format(time.RFC3339Nano), Source: "otel", Attributes: map[string]string{}}
	span.Digest, _ = coregovernance.Digest(span)
	span.Digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	raw, _ := json.Marshal(map[string]any{"spans": []coregovernance.TraceSpan{span}})
	root := t.TempDir()
	input := filepath.Join(root, "telemetry.json")
	if err := os.WriteFile(input, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	if code := execute([]string{"governance", "emit", "--kind", "telemetry", "--input", input, "--store-dir", filepath.Join(root, "store"), "--json"}, &out, &stderr); code != exitVerificationFailed {
		t.Fatalf("telemetry verification exit=%d output=%s", code, out.String())
	}
}

func TestGovernanceReplayJSONAndMarkdownAreDeterministic(t *testing.T) {
	proposalRaw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "action-contract-interop", "v1", "expected", "package-to-release", "pac-0d9384785d3b213a.json"))
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := actioncontract.ParseProposal(proposalRaw)
	if err != nil {
		t.Fatal(err)
	}
	storeDir := filepath.Join(t.TempDir(), "store")
	if _, err = actioncontract.PersistProposal(storeDir, proposal); err != nil {
		t.Fatal(err)
	}
	var first, second, markdown, stderr bytes.Buffer
	args := []string{"governance", "replay", "--store-dir", storeDir, "--contract-id", proposal.ContractID, "--json"}
	if code := execute(args, &first, &stderr); code != 0 {
		t.Fatalf("replay json: %d %s", code, stderr.String())
	}
	if code := execute(args, &second, &stderr); code != 0 {
		t.Fatalf("replay json second: %d", code)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("replay JSON drift")
	}
	if code := execute([]string{"governance", "replay", "--store-dir", storeDir, "--contract-id", proposal.ContractID, "--format", "markdown"}, &markdown, &stderr); code != 0 || !bytes.Contains(markdown.Bytes(), []byte("Governed Action Contract Timeline")) {
		t.Fatalf("replay markdown: %d %s", code, markdown.String())
	}
}

func structuredPublicKey(pub ed25519.PublicKey) proof.PublicKey { return proof.PublicKey{Public: pub} }
