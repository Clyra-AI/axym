package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	recordschema "github.com/Clyra-AI/axym/schemas/v1/record"
)

func TestNormalizedRecordSchemaContract(t *testing.T) {
	t.Parallel()

	valid := []byte(`{
		"source": "mcp",
		"source_product": "axym",
		"record_type": "tool_invocation",
		"timestamp": "2026-02-28T12:00:00Z",
		"event": {
			"tool_name": "fetch",
			"actor_identity": "agent://requester",
			"downstream_identity": "agent://executor",
			"owner_identity": "owner://payments",
			"policy_digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"approval_token_ref": "approval://chg-123",
			"target_kind": "tool",
			"target_id": "fetch"
		},
		"metadata": {"env": "ci"},
		"relationship": {
			"policy_ref": {"policy_digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
		},
		"controls": {"permissions_enforced": true}
	}`)
	if err := recordschema.ValidateNormalized(valid); err != nil {
		t.Fatalf("valid schema payload rejected: %v", err)
	}

	invalid := []byte(`{
		"source": "mcp",
		"record_type": "tool_invocation",
		"timestamp": "2026-02-28T12:00:00Z",
		"event": {"tool_name": "fetch"},
		"controls": {"permissions_enforced": true}
	}`)
	if err := recordschema.ValidateNormalized(invalid); err == nil {
		t.Fatal("invalid payload accepted; source_product should be required")
	}
}

func TestRecordAddRejectsUnknownTypeContract(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	recordPath := filepath.Join(root, "record.json")
	payload := []byte(`{
  "record_id": "contract-unknown-type",
  "record_version": "v1",
  "timestamp": "2026-03-18T00:00:00Z",
  "source": "manual",
  "source_product": "axym",
  "agent_id": "agent-1",
  "record_type": "does_not_exist",
  "event": {"anything": true},
  "controls": {"permissions_enforced": true}
}`)
	if err := os.WriteFile(recordPath, payload, 0o600); err != nil {
		t.Fatalf("write record fixture: %v", err)
	}
	out, exit := runAxymContract(t, "record", "add", "--input", recordPath, "--store-dir", filepath.Join(root, "store"), "--json")
	if exit != 3 {
		t.Fatalf("exit mismatch: got %d want 3 output=%s", exit, out)
	}
	var envelope map[string]any
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("decode output: %v output=%s", err, out)
	}
	errObj, _ := envelope["error"].(map[string]any)
	if errObj["reason"] != "schema_violation" {
		t.Fatalf("reason mismatch: got %v output=%s", errObj["reason"], out)
	}
}

func TestManualRecordSchemaContract(t *testing.T) {
	t.Parallel()

	root := testRepoRoot(t)
	requiredPaths := []string{
		"schemas/v1/record/manual-input.schema.json",
		"schemas/v1/record/README.md",
		"schemas/v1/record/examples/decision.v1.json",
		"schemas/v1/record/examples/approval.v1.json",
	}
	for _, rel := range requiredPaths {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("manual record contract asset missing: %s: %v", rel, err)
		}
	}

	valid := []byte(`{
		"record_version": "v1",
		"record_id": "contract-valid-v1",
		"timestamp": "2026-03-18T00:00:00Z",
		"source": "manual",
		"source_product": "axym",
		"agent_id": "agent-1",
		"record_type": "decision",
		"event": {"action": "allow"},
		"controls": {"permissions_enforced": true}
	}`)
	if err := recordschema.ValidateManualInput(valid); err != nil {
		t.Fatalf("valid manual payload rejected: %v", err)
	}

	legacy := []byte(`{
		"record_version": "1.0",
		"record_id": "contract-legacy-version",
		"timestamp": "2026-03-18T00:00:00Z",
		"source": "manual",
		"source_product": "axym",
		"agent_id": "agent-1",
		"record_type": "decision",
		"event": {"action": "allow"},
		"controls": {"permissions_enforced": true}
	}`)
	normalized, err := recordschema.NormalizeManualInput(legacy)
	if err != nil {
		t.Fatalf("normalize legacy manual payload: %v", err)
	}
	var normalizedPayload map[string]any
	if err := json.Unmarshal(normalized, &normalizedPayload); err != nil {
		t.Fatalf("decode normalized payload: %v", err)
	}
	if normalizedPayload["record_version"] != "v1" {
		t.Fatalf("legacy record_version was not normalized: %v", normalizedPayload["record_version"])
	}
	if err := recordschema.ValidateManualInput(normalized); err != nil {
		t.Fatalf("normalized legacy payload rejected: %v", err)
	}

	missingRequired := []byte(`{
		"record_version": "v1",
		"record_id": "contract-missing-source-product",
		"timestamp": "2026-03-18T00:00:00Z",
		"source": "manual",
		"agent_id": "agent-1",
		"record_type": "decision",
		"event": {"action": "allow"},
		"controls": {"permissions_enforced": true}
	}`)
	if err := recordschema.ValidateManualInput(missingRequired); err == nil {
		t.Fatal("manual payload missing required source_product was accepted")
	}
}

func TestManualRecordContractDocsAreLinkedFromLaunchSurfaces(t *testing.T) {
	t.Parallel()

	required := map[string]string{
		"README.md":                          "schemas/v1/record/README.md",
		"docs/commands/axym.md":              "../../schemas/v1/record/README.md",
		"docs/operator/quickstart.md":        "../../schemas/v1/record/README.md",
		"docs/operator/integration-model.md": "../../schemas/v1/record/README.md",
		"docs-site/public/llm/axym.md":       "../../../schemas/v1/record/README.md",
	}
	for path, snippet := range required {
		content := readRepoFile(t, path)
		if !strings.Contains(content, snippet) {
			t.Fatalf("%s missing manual record contract link %q", path, snippet)
		}
	}
}
