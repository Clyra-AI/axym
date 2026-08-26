package bundle

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Clyra-AI/axym/core/ingest/actioncontract"
	"github.com/Clyra-AI/axym/core/store"
	coreverify "github.com/Clyra-AI/axym/core/verify/bundle"
)

func TestBundleCarriesNativeGovernanceArtifactsAndVerifies(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "action-contract-interop", "v1", "expected", "package-to-release", "pac-0d9384785d3b213a.json"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := actioncontract.ParseProposal(raw)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	storeDir, outDir := filepath.Join(root, "store"), filepath.Join(root, "bundle")
	if _, err = store.New(store.Config{RootDir: storeDir}); err != nil {
		t.Fatal(err)
	}
	if _, err = actioncontract.PersistProposal(storeDir, p); err != nil {
		t.Fatal(err)
	}
	if _, err = Build(BuildRequest{AuditName: "governance-fixture", StoreDir: storeDir, OutputDir: outDir}); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"action-contract/proposals/" + p.ArtifactID + ".json", "action-contract/proposals/" + p.ArtifactID + ".envelope.json", "action-contract-register.json", "action-contract/packets/" + p.ContractID + ".json", "action-contract/packets/" + p.ContractID + ".md", "action-contract/timelines/" + p.ContractID + ".json", "action-contract/graphs/" + p.ContractID + ".json"} {
		if _, err := os.Stat(filepath.Join(outDir, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}
	got, _ := os.ReadFile(filepath.Join(outDir, "action-contract", "proposals", p.ArtifactID+".json"))
	if !bytes.Equal(got, raw) {
		t.Fatal("native proposal bytes changed in bundle")
	}
	if _, err := coreverify.Verify(outDir, nil); err != nil {
		t.Fatalf("bundle verify: %v", err)
	}
	var packet map[string]any
	packetRaw, _ := os.ReadFile(filepath.Join(outDir, "action-contract", "packets", p.ContractID+".json"))
	if json.Unmarshal(packetRaw, &packet) != nil || packet["signature"] == nil {
		t.Fatal("packet is not signed")
	}
	timelinePath := filepath.Join(outDir, "action-contract", "timelines", p.ContractID+".json")
	timelineRaw, _ := os.ReadFile(timelinePath)
	if err = os.WriteFile(timelinePath, append(timelineRaw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = coreverify.Verify(outDir, nil); err == nil {
		t.Fatal("tampered timeline accepted")
	}
}

func TestBundleFailureLeavesManagedOutputUnaccepted(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "bundle")
	if _, err := Build(BuildRequest{AuditName: "fixture", StoreDir: filepath.Join(root, "store"), OutputDir: output, FrameworkIDs: []string{"does-not-exist"}}); err == nil {
		t.Fatal("invalid bundle build accepted")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("newly-created managed output remained: %v", err)
	}
}

func TestBundleFailurePreservesExistingManagedOutput(t *testing.T) {
	root := t.TempDir()
	storeDir, output := filepath.Join(root, "store"), filepath.Join(root, "bundle")
	if _, err := Build(BuildRequest{AuditName: "fixture", StoreDir: storeDir, OutputDir: output}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(output, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Build(BuildRequest{AuditName: "fixture", StoreDir: storeDir, OutputDir: output, FrameworkIDs: []string{"does-not-exist"}}); err == nil {
		t.Fatal("invalid rebuild accepted")
	}
	after, err := os.ReadFile(filepath.Join(output, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("failed rebuild changed existing bundle")
	}
}
