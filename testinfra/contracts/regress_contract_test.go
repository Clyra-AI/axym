package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	regressschema "github.com/Clyra-AI/axym/schemas/v1/regress"
)

func TestRegressExit5Contract(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeBaseline := filepath.Join(root, "baseline-store")
	storeCurrent := filepath.Join(root, "current-store")
	baselinePath := filepath.Join(root, "baseline.json")
	frameworkPath := filepath.Join(testRepoRoot(t), "fixtures", "frameworks", "regress-minimal.yaml")
	recordPath := filepath.Join(testRepoRoot(t), "fixtures", "records", "decision.json")

	if out, exit := runAxymContract(t, "record", "add", "--input", recordPath, "--store-dir", storeBaseline, "--json"); exit != 0 {
		t.Fatalf("record add exit mismatch: %d output=%s", exit, out)
	}
	if out, exit := runAxymContract(t, "regress", "init", "--baseline", baselinePath, "--frameworks", frameworkPath, "--store-dir", storeBaseline, "--json"); exit != 0 {
		t.Fatalf("regress init exit mismatch: %d output=%s", exit, out)
	}
	out, exit := runAxymContract(t, "regress", "run", "--baseline", baselinePath, "--frameworks", frameworkPath, "--store-dir", storeCurrent, "--json")
	if exit != 5 {
		t.Fatalf("regress run exit mismatch: got %d want 5 output=%s", exit, out)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("decode output json: %v output=%s", err, out)
	}
	if payload["command"] != "regress" {
		t.Fatalf("command mismatch: %v", payload["command"])
	}
}

func TestRegressBaselineSchemaContract(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeBaseline := filepath.Join(root, "baseline-store")
	baselinePath := filepath.Join(root, "baseline.json")
	frameworkPath := filepath.Join(testRepoRoot(t), "fixtures", "frameworks", "regress-minimal.yaml")
	recordPath := filepath.Join(testRepoRoot(t), "fixtures", "records", "decision.json")

	if out, exit := runAxymContract(t, "record", "add", "--input", recordPath, "--store-dir", storeBaseline, "--json"); exit != 0 {
		t.Fatalf("record add exit mismatch: %d output=%s", exit, out)
	}
	if out, exit := runAxymContract(t, "regress", "init", "--baseline", baselinePath, "--frameworks", frameworkPath, "--store-dir", storeBaseline, "--json"); exit != 0 {
		t.Fatalf("regress init exit mismatch: %d output=%s", exit, out)
	}
	raw, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatalf("read baseline file: %v", err)
	}
	if err := regressschema.ValidateBaseline(raw); err != nil {
		t.Fatalf("validate baseline schema: %v", err)
	}
}
