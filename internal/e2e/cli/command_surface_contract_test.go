package cli

import (
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
