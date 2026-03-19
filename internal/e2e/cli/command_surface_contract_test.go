package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandSurfaceIncludesPrimaryCommands(t *testing.T) {
	t.Parallel()

	stdout, exit := runAxymCLI(t, "--help")
	if exit != 0 {
		t.Fatalf("unexpected exit %d output=%s", exit, stdout)
	}
	required := []string{
		"init",
		"collect",
		"map",
		"gaps",
		"regress",
		"bundle",
		"review",
		"verify",
		"record",
		"override",
		"ingest",
		"replay",
	}
	for _, command := range required {
		if !strings.Contains(stdout, command) {
			t.Fatalf("missing command %q in root help output=%s", command, stdout)
		}
	}
}

func TestInvalidPolicyConfigFailsClosedWithExit6(t *testing.T) {
	t.Parallel()

	stdout, exit := runAxymCLI(t, "map", "--policy-config", "fixtures/policy/invalid-missing-defaults.yaml", "--json")
	if exit != 6 {
		t.Fatalf("exit mismatch: got %d want 6 output=%s", exit, stdout)
	}
	if !strings.Contains(strings.ToLower(stdout), "invalid policy config") {
		t.Fatalf("expected invalid policy config reason output=%s", stdout)
	}
}

func TestInitHelpIncludesSamplePackFlag(t *testing.T) {
	t.Parallel()

	stdout, exit := runAxymCLI(t, "init", "--help")
	if exit != 0 {
		t.Fatalf("unexpected exit %d output=%s", exit, stdout)
	}
	if !strings.Contains(stdout, "--sample-pack") {
		t.Fatalf("missing --sample-pack flag in init help output=%s", stdout)
	}
}

func TestInitSamplePackJSONContract(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	stdout, exit := runAxymCLI(
		t,
		"init",
		"--store-dir", filepath.Join(root, "store"),
		"--policy-path", filepath.Join(root, "axym-policy.yaml"),
		"--sample-pack", filepath.Join(root, "axym-sample"),
		"--json",
	)
	if exit != 0 {
		t.Fatalf("unexpected exit %d output=%s", exit, stdout)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("decode output: %v output=%s", err, stdout)
	}
	data, _ := payload["data"].(map[string]any)
	samplePack, ok := data["sample_pack"].(map[string]any)
	if !ok {
		t.Fatalf("expected sample_pack object output=%s", stdout)
	}
	files, _ := samplePack["files"].([]any)
	if len(files) != 3 {
		t.Fatalf("expected 3 created files output=%s", stdout)
	}
	nextSteps, _ := samplePack["next_steps"].([]any)
	if len(nextSteps) != 7 {
		t.Fatalf("expected 7 next steps output=%s", stdout)
	}
}
