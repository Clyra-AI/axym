package override_replay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	corebundle "github.com/Clyra-AI/axym/core/bundle"
	coreoverride "github.com/Clyra-AI/axym/core/override"
	corereplay "github.com/Clyra-AI/axym/core/replay"
	coreverify "github.com/Clyra-AI/axym/core/verify"
)

func TestOverrideAppendOnlyAndTamperDetection(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	fixedNow := time.Date(2026, 9, 15, 15, 0, 0, 0, time.UTC)
	first, err := coreoverride.Create(coreoverride.Request{
		Bundle:   "Q3-2026",
		Reason:   "fixture",
		Signer:   "ops-key",
		StoreDir: storeDir,
		Now:      func() time.Time { return fixedNow },
	})
	if err != nil {
		t.Fatalf("first override create: %v", err)
	}
	second, err := coreoverride.Create(coreoverride.Request{
		Bundle:   "Q3-2026",
		Reason:   "fixture-2",
		Signer:   "ops-key",
		StoreDir: storeDir,
		Now:      func() time.Time { return fixedNow.Add(time.Minute) },
	})
	if err != nil {
		t.Fatalf("second override create: %v", err)
	}
	if first.RecordID == second.RecordID {
		t.Fatalf("override records must remain append-only and unique: first=%s second=%s", first.RecordID, second.RecordID)
	}

	rawOverrides, err := os.ReadFile(first.ArtifactPath)
	if err != nil {
		t.Fatalf("read override artifact: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(rawOverrides)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two append-only override entries, got=%d", len(lines))
	}

	chainPath := filepath.Join(storeDir, "chain.json")
	rawChain, err := os.ReadFile(chainPath)
	if err != nil {
		t.Fatalf("read chain: %v", err)
	}
	var chain map[string]any
	if err := json.Unmarshal(rawChain, &chain); err != nil {
		t.Fatalf("decode chain: %v", err)
	}
	records := chain["records"].([]any)
	firstRecord := records[0].(map[string]any)
	event := firstRecord["event"].(map[string]any)
	event["reason"] = "tampered"
	mutated, err := json.MarshalIndent(chain, "", "  ")
	if err != nil {
		t.Fatalf("marshal tampered chain: %v", err)
	}
	if err := os.WriteFile(chainPath, mutated, 0o600); err != nil {
		t.Fatalf("write tampered chain: %v", err)
	}
	if _, err := coreverify.VerifyChainFromStoreDir(storeDir); err == nil {
		t.Fatal("expected tamper detection after override mutation")
	}
}

func TestReplayCertificationDeterministicFields(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	result, err := corereplay.Run(corereplay.Request{
		Model:    "payments-agent",
		Tier:     "A",
		StoreDir: storeDir,
		Now:      func() time.Time { return time.Date(2026, 9, 15, 16, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("replay run: %v", err)
	}
	if result.Tier != "A" {
		t.Fatalf("tier mismatch: %+v", result)
	}
	if result.BlastRadius["description"] != "organization-wide" {
		t.Fatalf("blast radius summary mismatch: %+v", result.BlastRadius)
	}
}

func TestOverrideVisibleInBundle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	outputDir := filepath.Join(root, "bundle")
	if _, err := coreoverride.Create(coreoverride.Request{
		Bundle:   "Q3-2026",
		Reason:   "fixture",
		Signer:   "ops-key",
		StoreDir: storeDir,
		Now:      func() time.Time { return time.Date(2026, 9, 15, 17, 0, 0, 0, time.UTC) },
	}); err != nil {
		t.Fatalf("override create: %v", err)
	}

	if _, err := corebundle.Build(corebundle.BuildRequest{
		AuditName:    "Q3-2026",
		FrameworkIDs: []string{"eu-ai-act", "soc2"},
		StoreDir:     storeDir,
		OutputDir:    outputDir,
	}); err != nil {
		t.Fatalf("bundle build: %v", err)
	}

	rawRecords, err := os.ReadFile(filepath.Join(outputDir, "raw-records.jsonl"))
	if err != nil {
		t.Fatalf("read raw-records: %v", err)
	}
	content := string(rawRecords)
	if !strings.Contains(content, "\"kind\":\"override\"") {
		t.Fatalf("expected override evidence in bundle raw records: %s", content)
	}
}
