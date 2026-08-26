package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestActionContractConsumerNegativeScenariosRemainValidReceipts(t *testing.T) {
	for _, scenario := range []string{"approval-expiry", "compensation", "customer-data-to-egress", "excessive-child-authority", "failed-effect-validation", "package-to-release", "secret-to-network", "supersession", "workflow-to-deploy"} {
		path := filepath.Join("..", "..", "testdata", "action-contract-interop", "v1", "expected", scenario)
		entries, err := os.ReadDir(path)
		if err != nil {
			t.Fatal(err)
		}
		var artifact string
		for _, entry := range entries {
			if filepath.Ext(entry.Name()) == ".json" && entry.Name() != "activated-action-contract.json" {
				artifact = filepath.Join(path, entry.Name())
				break
			}
		}
		var out, stderr bytes.Buffer
		if code := execute([]string{"action-contract", "consume", artifact, "--json"}, &out, &stderr); code != 0 {
			t.Fatalf("%s rejected: code=%d out=%s err=%s", scenario, code, out.String(), stderr.String())
		}
		var envelope struct {
			Data struct {
				Status   string `json:"status"`
				Semantic struct {
					Classification string `json:"classification"`
				} `json:"semantic_result"`
			} `json:"data"`
		}
		if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Data.Status != "pass" || envelope.Data.Semantic.Classification == "" {
			t.Fatalf("%s receipt is not semantic pass: %s", scenario, out.String())
		}
	}
}

func TestActionContractStructuralFailurePersistsNothing(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "bad.json")
	store := filepath.Join(root, "store")
	if err := os.WriteFile(source, []byte(`{"schema_id":"https://wrkr.dev/schemas/v1/proposed-action-contract-artifact.schema.json","schema_version":"1","unknown":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	if code := execute([]string{"action-contract", "consume", source, "--store-dir", store, "--json"}, &out, &stderr); code == 0 {
		t.Fatal("structurally invalid proposal accepted")
	}
	if _, err := os.Stat(filepath.Join(store, "action-contract")); !os.IsNotExist(err) {
		t.Fatalf("invalid proposal was persisted: %v", err)
	}
	validPath := filepath.Join(root, "tampered.json")
	valid, err := os.ReadFile(filepath.Join("..", "..", "testdata", "action-contract-interop", "v1", "expected", "package-to-release", "pac-0d9384785d3b213a.json"))
	if err != nil {
		t.Fatal(err)
	}
	valid[len(valid)-2] ^= 1
	if err = os.WriteFile(validPath, valid, 0o600); err != nil {
		t.Fatal(err)
	}
	var tamperedOut, tamperedErr bytes.Buffer
	if code := execute([]string{"action-contract", "consume", validPath, "--store-dir", filepath.Join(root, "tampered-store"), "--json"}, &tamperedOut, &tamperedErr); code == 0 {
		t.Fatal("tampered proposal accepted")
	}
	if _, err = os.Stat(filepath.Join(root, "tampered-store", "action-contract")); !os.IsNotExist(err) {
		t.Fatalf("tampered proposal was persisted: %v", err)
	}
}

func TestActionContractStoreFailureIsRuntimeError(t *testing.T) {
	fixture := filepath.Join("..", "..", "testdata", "action-contract-interop", "v1", "expected", "customer-data-to-egress", "pac-6dcee5a6d9a65e8c.json")
	root := t.TempDir()
	storePath := filepath.Join(root, "store-file")
	if err := os.WriteFile(storePath, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	if code := execute([]string{"action-contract", "consume", fixture, "--store-dir", storePath, "--json"}, &out, &stderr); code != exitRuntimeFailure {
		t.Fatalf("store failure exit=%d want=%d output=%s stderr=%s", code, exitRuntimeFailure, out.String(), stderr.String())
	}
}
