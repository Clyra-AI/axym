package contracts

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestComplianceThresholdExitContract(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	stdout, exit := runAxymContract(t, "collect", "--fixture-dir", filepath.Join(testRepoRoot(t), "fixtures", "collectors"), "--store-dir", storeDir, "--json")
	if exit != 0 {
		t.Fatalf("collect setup failed: exit=%d output=%s", exit, stdout)
	}

	mapOut, mapExit := runAxymContract(t, "map", "--frameworks", "eu-ai-act", "--store-dir", storeDir, "--min-coverage", "1", "--json")
	if mapExit != 5 {
		t.Fatalf("map exit mismatch: got %d want 5 output=%s", mapExit, mapOut)
	}
	assertThresholdError(t, mapOut)

	gapsOut, gapsExit := runAxymContract(t, "gaps", "--frameworks", "eu-ai-act", "--store-dir", storeDir, "--min-coverage", "1", "--json")
	if gapsExit != 5 {
		t.Fatalf("gaps exit mismatch: got %d want 5 output=%s", gapsExit, gapsOut)
	}
	assertThresholdError(t, gapsOut)
}

func assertThresholdError(t *testing.T, output string) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	errObj, _ := payload["error"].(map[string]any)
	if errObj["reason"] != "COVERAGE_THRESHOLD_NOT_MET" {
		t.Fatalf("reason mismatch: %s", output)
	}
	data, _ := payload["data"].(map[string]any)
	thresholdObj, _ := data["threshold"].(map[string]any)
	if thresholdObj["passed"] != false {
		t.Fatalf("threshold object mismatch: %s", output)
	}
}
