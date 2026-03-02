package contracts

import (
	"encoding/json"
	"path/filepath"
	"testing"

	gapsschema "github.com/Clyra-AI/axym/schemas/v1/gaps"
)

func TestGapsSchemaContract(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	stdout, exit := runAxymContract(t, "collect", "--fixture-dir", filepath.Join(testRepoRoot(t), "fixtures", "collectors"), "--store-dir", storeDir, "--json")
	if exit != 0 {
		t.Fatalf("collect setup failed: exit=%d output=%s", exit, stdout)
	}
	gapsOut, gapsExit := runAxymContract(t, "gaps", "--frameworks", "eu-ai-act,soc2", "--store-dir", storeDir, "--json")
	if gapsExit != 0 {
		t.Fatalf("gaps exit mismatch: %d output=%s", gapsExit, gapsOut)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(gapsOut), &payload); err != nil {
		t.Fatalf("decode gaps output: %v", err)
	}
	data, _ := payload["data"].(map[string]any)
	raw, err := json.Marshal(map[string]any{
		"summary": data["summary"],
		"grade":   data["grade"],
		"gaps":    data["gaps"],
	})
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	if err := gapsschema.Validate(raw); err != nil {
		t.Fatalf("gaps schema validation failed: %v payload=%s", err, string(raw))
	}
}
