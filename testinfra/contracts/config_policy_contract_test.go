package contracts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitWritesPolicyContract(t *testing.T) {
	t.Parallel()

	policyPath := filepath.Join(t.TempDir(), "axym-policy.yaml")
	stdout, exit := runAxymContract(t, "init", "--policy-path", policyPath, "--json")
	if exit != 0 {
		t.Fatalf("unexpected exit %d output=%s", exit, stdout)
	}
	if _, err := os.Stat(policyPath); err != nil {
		t.Fatalf("policy file not created: %v", err)
	}
}

func TestInvalidPolicyContractExitCode(t *testing.T) {
	t.Parallel()

	stdout, exit := runAxymContract(t, "map", "--policy-config", filepath.Join(testRepoRoot(t), "fixtures", "policy", "invalid-missing-defaults.yaml"), "--json")
	if exit != 6 {
		t.Fatalf("exit mismatch: got %d want 6 output=%s", exit, stdout)
	}
}
