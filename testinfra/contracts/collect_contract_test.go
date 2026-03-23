package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCollectJSONEnvelopeContract(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	stdout, exit := runAxymContract(t, "collect", "--dry-run", "--fixture-dir", filepath.Join(testRepoRoot(t), "fixtures", "collectors"), "--store-dir", storeDir, "--json")
	if exit != 0 {
		t.Fatalf("unexpected exit %d output=%s", exit, stdout)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if payload["command"] != "collect" {
		t.Fatalf("command mismatch: got %v", payload["command"])
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("missing data envelope: %s", stdout)
	}
	if _, ok := data["sources"].([]any); !ok {
		t.Fatalf("missing sources summary: %s", stdout)
	}
	if _, ok := data["reason_codes"]; !ok {
		t.Fatalf("missing reason_codes: %s", stdout)
	}
}

func TestCollectGovernanceContextEngineeringContract(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	governancePath := filepath.Join(testRepoRoot(t), "fixtures", "governance", "context_engineering.jsonl")
	stdout, exit := runAxymContract(t, "collect", "--governance-event-file", governancePath, "--store-dir", storeDir, "--json")
	if exit != 0 {
		t.Fatalf("unexpected exit %d output=%s", exit, stdout)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("missing data envelope: %s", stdout)
	}
	if captured, _ := data["captured"].(float64); captured != 3 {
		t.Fatalf("captured mismatch: %s", stdout)
	}
	sources, ok := data["sources"].([]any)
	if !ok {
		t.Fatalf("missing sources summary: %s", stdout)
	}
	for _, candidate := range sources {
		source, _ := candidate.(map[string]any)
		if source["name"] != "governanceevent" {
			continue
		}
		if source["captured"] != float64(3) {
			t.Fatalf("governance source captured mismatch: %v", source)
		}
		if source["status"] != "ok" {
			t.Fatalf("governance source status mismatch: %v", source)
		}
		return
	}
	t.Fatalf("governanceevent source summary missing: %s", stdout)
}

func TestCollectGovernanceEventNoInputReasonCodeContract(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	stdout, exit := runAxymContract(t, "collect", "--dry-run", "--store-dir", storeDir, "--json")
	if exit != 0 {
		t.Fatalf("unexpected exit %d output=%s", exit, stdout)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("missing data envelope: %s", stdout)
	}
	sources, ok := data["sources"].([]any)
	if !ok {
		t.Fatalf("missing sources summary: %s", stdout)
	}
	for _, candidate := range sources {
		source, _ := candidate.(map[string]any)
		if source["name"] != "governanceevent" {
			continue
		}
		if source["status"] != "empty" {
			t.Fatalf("governanceevent source status mismatch: %v", source)
		}
		reasons, _ := source["reason_codes"].([]any)
		if len(reasons) != 1 || reasons[0] != "NO_INPUT" {
			t.Fatalf("governanceevent reason codes mismatch: %v", source)
		}
		return
	}
	t.Fatalf("governanceevent source summary missing: %s", stdout)
}

func TestCollectPluginEmptyMetadataRoundTripContract(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	pluginPath := filepath.Join(root, "plugin.go")
	pluginSource := []byte("package main\nimport \"fmt\"\nfunc main(){fmt.Println(`{\"source_type\":\"plugin\",\"source\":\"custom\",\"source_product\":\"axym\",\"record_type\":\"tool_invocation\",\"agent_id\":\"agent-1\",\"timestamp\":\"2026-03-18T00:00:00Z\",\"event\":{\"tool_name\":\"scan\"},\"metadata\":{},\"controls\":{\"permissions_enforced\":true}}`)}\n")
	if err := os.WriteFile(pluginPath, pluginSource, 0o600); err != nil {
		t.Fatalf("write plugin source: %v", err)
	}

	storeDir := filepath.Join(root, "store")
	stdout, exit := runAxymContract(t, "collect", "--plugin-timeout", "60s", "--plugin", "go run "+pluginPath, "--store-dir", storeDir, "--json")
	if exit != 0 {
		t.Fatalf("unexpected collect exit %d output=%s", exit, stdout)
	}
	var collectPayload map[string]any
	if err := json.Unmarshal([]byte(stdout), &collectPayload); err != nil {
		t.Fatalf("decode collect json: %v", err)
	}
	data, _ := collectPayload["data"].(map[string]any)
	if appended, _ := data["appended"].(float64); appended != 1 {
		t.Fatalf("expected appended=1 output=%s", stdout)
	}

	verifyOut, verifyExit := runAxymContract(t, "verify", "--chain", "--store-dir", storeDir, "--json")
	if verifyExit != 0 {
		t.Fatalf("unexpected verify exit %d output=%s", verifyExit, verifyOut)
	}
	var verifyPayload map[string]any
	if err := json.Unmarshal([]byte(verifyOut), &verifyPayload); err != nil {
		t.Fatalf("decode verify json: %v", err)
	}
	verifyData, _ := verifyPayload["data"].(map[string]any)
	verification, _ := verifyData["verification"].(map[string]any)
	if verification["intact"] != true {
		t.Fatalf("expected intact=true output=%s", verifyOut)
	}
	if verification["count"] != float64(1) {
		t.Fatalf("expected count=1 output=%s", verifyOut)
	}
}

func TestCollectPluginRelationshipRoundTripContract(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	pluginPath := filepath.Join(root, "plugin.go")
	pluginSource := []byte("package main\nimport \"fmt\"\nfunc main(){fmt.Println(`{\"source_type\":\"plugin\",\"source\":\"custom\",\"source_product\":\"axym\",\"record_type\":\"tool_invocation\",\"agent_id\":\"agent-1\",\"timestamp\":\"2026-03-18T00:00:00Z\",\"event\":{\"tool_name\":\"scan\"},\"metadata\":{},\"relationship\":{\"parent_ref\":{\"kind\":\"trace\",\"id\":\"trace-123\"},\"entity_refs\":[{\"kind\":\"resource\",\"id\":\"db://prod\"}],\"policy_ref\":{\"policy_digest\":\"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\"}},\"controls\":{\"permissions_enforced\":true}}`)}\n")
	if err := os.WriteFile(pluginPath, pluginSource, 0o600); err != nil {
		t.Fatalf("write plugin source: %v", err)
	}

	storeDir := filepath.Join(root, "store")
	stdout, exit := runAxymContract(t, "collect", "--plugin-timeout", "60s", "--plugin", "go run "+pluginPath, "--store-dir", storeDir, "--json")
	if exit != 0 {
		t.Fatalf("unexpected collect exit %d output=%s", exit, stdout)
	}

	raw, err := os.ReadFile(filepath.Join(storeDir, "chain.json"))
	if err != nil {
		t.Fatalf("read chain: %v", err)
	}
	var chain map[string]any
	if err := json.Unmarshal(raw, &chain); err != nil {
		t.Fatalf("decode chain: %v", err)
	}
	records, _ := chain["records"].([]any)
	if len(records) != 1 {
		t.Fatalf("expected one record, got %d", len(records))
	}
	record, _ := records[0].(map[string]any)
	relationship, _ := record["relationship"].(map[string]any)
	parent, _ := relationship["parent_ref"].(map[string]any)
	if parent["kind"] != "trace" || parent["id"] != "trace-123" {
		t.Fatalf("parent ref mismatch: %+v", parent)
	}
	policy, _ := relationship["policy_ref"].(map[string]any)
	if policy["policy_digest"] != "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("policy ref mismatch: %+v", policy)
	}
	entityRefs, _ := relationship["entity_refs"].([]any)
	foundResource := false
	for _, item := range entityRefs {
		ref, _ := item.(map[string]any)
		if ref["kind"] == "resource" && ref["id"] == "db://prod" {
			foundResource = true
			break
		}
	}
	if !foundResource {
		t.Fatalf("resource relationship ref missing: %+v", entityRefs)
	}
}
