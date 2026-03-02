package contracts

import (
	"os"
	"path/filepath"
	"testing"

	bundleschema "github.com/Clyra-AI/axym/schemas/v1/bundle"
)

func TestBundleOSCALSchemaContract(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	bundleDir := filepath.Join(t.TempDir(), "bundle")
	if out, exit := runAxymContract(t,
		"collect",
		"--fixture-dir", filepath.Join(testRepoRoot(t), "fixtures", "collectors"),
		"--store-dir", storeDir,
		"--json",
	); exit != 0 {
		t.Fatalf("collect setup failed: exit=%d output=%s", exit, out)
	}
	if out, exit := runAxymContract(t,
		"bundle",
		"--audit", "Q3-2026",
		"--frameworks", "eu-ai-act,soc2",
		"--store-dir", storeDir,
		"--output", bundleDir,
		"--json",
	); exit != 0 {
		t.Fatalf("bundle setup failed: exit=%d output=%s", exit, out)
	}
	raw, err := os.ReadFile(filepath.Join(bundleDir, "oscal-v1.1", "component-definition.json"))
	if err != nil {
		t.Fatalf("read oscal output: %v", err)
	}
	if err := bundleschema.ValidateOSCAL(raw); err != nil {
		t.Fatalf("oscal schema validation failed: %v", err)
	}
}
