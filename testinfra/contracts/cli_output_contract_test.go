package contracts

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCLIOutputRootHelpIncludesGlobalFlags(t *testing.T) {
	t.Parallel()

	stdout, exit := runAxymContract(t, "--help")
	if exit != 0 {
		t.Fatalf("unexpected exit %d output=%s", exit, stdout)
	}
	for _, want := range []string{"--json", "--quiet", "--explain"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("missing %s from root help output=%s", want, stdout)
		}
	}
}

func TestCLIOutputErrorEnvelopeContract(t *testing.T) {
	t.Parallel()

	stdout, exit := runAxymContract(t, "verify", "--json")
	if exit != 6 {
		t.Fatalf("exit mismatch: got %d want 6 output=%s", exit, stdout)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("decode output json: %v output=%s", err, stdout)
	}
	if payload["ok"] != false {
		t.Fatalf("expected ok=false output=%s", stdout)
	}
	errorPayload, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("missing error envelope output=%s", stdout)
	}
	if strings.TrimSpace(toString(errorPayload["reason"])) == "" {
		t.Fatalf("missing non-empty reason output=%s", stdout)
	}
	if strings.TrimSpace(toString(errorPayload["message"])) == "" {
		t.Fatalf("missing non-empty message output=%s", stdout)
	}
	if _, ok := errorPayload["break_index"]; ok {
		t.Fatalf("unexpected break_index on non-verify error output=%s", stdout)
	}
}

func TestCLIOutputCommandHelpCarriesGlobalJSONFlag(t *testing.T) {
	t.Parallel()

	commands := []string{
		"init",
		"collect",
		"record",
		"ingest",
		"map",
		"gaps",
		"regress",
		"review",
		"override",
		"replay",
		"bundle",
		"verify",
		"version",
	}
	for _, command := range commands {
		command := command
		t.Run(command, func(t *testing.T) {
			t.Parallel()
			stdout, exit := runAxymContract(t, command, "--help")
			if exit != 0 {
				t.Fatalf("unexpected exit %d output=%s", exit, stdout)
			}
			if !strings.Contains(stdout, "--json") {
				t.Fatalf("missing --json global flag output=%s", stdout)
			}
		})
	}
}

func toString(value any) string {
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}
