package bundle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	corebundle "github.com/Clyra-AI/axym/core/bundle"
	corecollect "github.com/Clyra-AI/axym/core/collect"
	"github.com/Clyra-AI/axym/core/collector"
	"github.com/Clyra-AI/axym/core/policy/sink"
	"github.com/Clyra-AI/axym/core/store"
)

func TestContextEngineeringBundleArtifactsAreDeterministic(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	governancePath := filepath.Join(repoRoot, "fixtures", "governance", "context_engineering.jsonl")

	req := collector.Request{
		Now:                  time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC),
		GovernanceEventFiles: []string{governancePath},
	}
	registry, err := corecollect.BuildRegistry(req)
	if err != nil {
		t.Fatalf("BuildRegistry: %v", err)
	}

	storeDir := filepath.Join(t.TempDir(), "store")
	st, err := store.New(store.Config{RootDir: storeDir, ComplianceMode: true})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	runner := corecollect.Runner{Registry: registry, Store: st, SinkMode: sink.ModeFailClosed}
	if _, err := runner.Run(context.Background(), req, false); err != nil {
		t.Fatalf("runner.Run: %v", err)
	}

	firstDir := filepath.Join(t.TempDir(), "bundle-a")
	firstResult, err := corebundle.Build(corebundle.BuildRequest{
		AuditName:    "Q3-2026",
		FrameworkIDs: []string{"eu-ai-act", "soc2"},
		StoreDir:     storeDir,
		OutputDir:    firstDir,
	})
	if err != nil {
		t.Fatalf("first bundle build: %v", err)
	}

	secondDir := filepath.Join(t.TempDir(), "bundle-b")
	secondResult, err := corebundle.Build(corebundle.BuildRequest{
		AuditName:    "Q3-2026",
		FrameworkIDs: []string{"eu-ai-act", "soc2"},
		StoreDir:     storeDir,
		OutputDir:    secondDir,
	})
	if err != nil {
		t.Fatalf("second bundle build: %v", err)
	}

	firstRaw, err := os.ReadFile(filepath.Join(firstResult.Path, "raw-records.jsonl"))
	if err != nil {
		t.Fatalf("read first raw records: %v", err)
	}
	if !strings.Contains(string(firstRaw), `"context_event_class":"context_engineering"`) {
		t.Fatalf("raw records missing context engineering evidence: %s", string(firstRaw))
	}
	if !strings.Contains(string(firstRaw), `"context_artifact_digest":"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"`) {
		t.Fatalf("raw records missing context digest: %s", string(firstRaw))
	}

	firstDigest, err := digestDir(firstResult.Path)
	if err != nil {
		t.Fatalf("digest first bundle: %v", err)
	}
	secondDigest, err := digestDir(secondResult.Path)
	if err != nil {
		t.Fatalf("digest second bundle: %v", err)
	}
	if firstDigest != secondDigest {
		t.Fatalf("bundle digest mismatch: first=%s second=%s", firstDigest, secondDigest)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func digestDir(root string) (string, error) {
	h := sha256.New()
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		h.Write([]byte(filepath.ToSlash(rel)))
		h.Write([]byte{0})
		h.Write(raw)
		h.Write([]byte{0})
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
