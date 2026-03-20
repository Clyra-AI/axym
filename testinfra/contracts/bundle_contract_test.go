package contracts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	verifysupport "github.com/Clyra-AI/axym/core/verifysupport"
	"github.com/Clyra-AI/proof"
)

func TestBundleAndVerifyBundleJSONEnvelopeContract(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	bundleDir := filepath.Join(t.TempDir(), "bundle")

	collectOut, collectExit := runAxymContract(t,
		"collect",
		"--fixture-dir", filepath.Join(testRepoRoot(t), "fixtures", "collectors"),
		"--store-dir", storeDir,
		"--json",
	)
	if collectExit != 0 {
		t.Fatalf("collect setup failed: exit=%d output=%s", collectExit, collectOut)
	}

	bundleOut, bundleExit := runAxymContract(t,
		"bundle",
		"--audit", "Q3-2026",
		"--frameworks", "eu-ai-act,soc2",
		"--store-dir", storeDir,
		"--output", bundleDir,
		"--json",
	)
	if bundleExit != 0 {
		t.Fatalf("bundle failed: exit=%d output=%s", bundleExit, bundleOut)
	}
	var bundlePayload map[string]any
	if err := json.Unmarshal([]byte(bundleOut), &bundlePayload); err != nil {
		t.Fatalf("decode bundle output: %v output=%s", err, bundleOut)
	}
	if bundlePayload["command"] != "bundle" {
		t.Fatalf("bundle command mismatch: %s", bundleOut)
	}
	if bundlePayload["ok"] != true {
		t.Fatalf("bundle expected ok=true: %s", bundleOut)
	}

	verifyOut, verifyExit := runAxymContract(t, "verify", "--bundle", bundleDir, "--json")
	if verifyExit != 0 {
		t.Fatalf("verify bundle failed: exit=%d output=%s", verifyExit, verifyOut)
	}
	var verifyPayload map[string]any
	if err := json.Unmarshal([]byte(verifyOut), &verifyPayload); err != nil {
		t.Fatalf("decode verify output: %v output=%s", err, verifyOut)
	}
	if verifyPayload["command"] != "verify" {
		t.Fatalf("verify command mismatch: %s", verifyOut)
	}
	data, _ := verifyPayload["data"].(map[string]any)
	verification, _ := data["verification"].(map[string]any)
	if verification["cryptographic"] != true {
		t.Fatalf("expected cryptographic=true: %s", verifyOut)
	}
	if verification["compliance_verified"] != true {
		t.Fatalf("expected compliance_verified=true: %s", verifyOut)
	}
	compliance, _ := verification["compliance"].(map[string]any)
	if _, ok := compliance["identity_governance"].(map[string]any); !ok {
		t.Fatalf("expected identity_governance envelope: %s", verifyOut)
	}
}

func TestVerifyBundleInvalidOSCALContractExit(t *testing.T) {
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
	oscalPath := filepath.Join(bundleDir, "oscal-v1.1", "component-definition.json")
	if err := os.WriteFile(oscalPath, []byte(`{"bad":true}`), 0o600); err != nil {
		t.Fatalf("tamper oscal: %v", err)
	}
	if err := updateManifestHash(bundleDir, filepath.ToSlash(filepath.Join("oscal-v1.1", "component-definition.json"))); err != nil {
		t.Fatalf("update manifest hash: %v", err)
	}
	if err := resignManifest(storeDir, bundleDir); err != nil {
		t.Fatalf("resign manifest: %v", err)
	}
	out, exit := runAxymContract(t, "verify", "--bundle", bundleDir, "--json")
	if exit != 3 {
		t.Fatalf("exit mismatch: got=%d want=3 output=%s", exit, out)
	}
}

func updateManifestHash(bundleDir string, relPath string) error {
	payload, err := os.ReadFile(filepath.Join(bundleDir, filepath.FromSlash(relPath)))
	if err != nil {
		return err
	}
	sum := sha256.Sum256(payload)
	want := "sha256:" + hex.EncodeToString(sum[:])

	manifestPath := filepath.Join(bundleDir, "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var manifest struct {
		Files []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"files"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return err
	}
	for i := range manifest.Files {
		if manifest.Files[i].Path == relPath {
			manifest.Files[i].SHA256 = want
		}
	}
	out, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(manifestPath, out, 0o600)
}

func resignManifest(storeDir string, bundleDir string) error {
	signingKey, err := verifysupport.LoadStoreSigningKey(storeDir)
	if err != nil {
		return err
	}
	manifestPath := filepath.Join(bundleDir, "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var manifest map[string]any
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return err
	}
	delete(manifest, "signatures")
	cleaned, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(manifestPath, cleaned, 0o600); err != nil {
		return err
	}
	_, err = proof.SignBundleFile(bundleDir, signingKey)
	return err
}
