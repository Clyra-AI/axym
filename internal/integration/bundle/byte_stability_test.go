package bundle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	corebundle "github.com/Clyra-AI/axym/core/bundle"
	corecollect "github.com/Clyra-AI/axym/core/collect"
	"github.com/Clyra-AI/axym/core/collector"
	"github.com/Clyra-AI/axym/core/policy/sink"
	"github.com/Clyra-AI/axym/core/store"
)

func TestBundleByteStability(t *testing.T) {
	t.Parallel()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	fixtureDir := filepath.Join(repoRoot, "fixtures", "collectors")

	req := collector.Request{
		Now:        time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
		FixtureDir: fixtureDir,
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
		t.Fatalf("collect runner: %v", err)
	}

	outDir := filepath.Join(t.TempDir(), "bundle")
	if _, err := corebundle.Build(corebundle.BuildRequest{
		AuditName:    "Q3-2026",
		FrameworkIDs: []string{"eu-ai-act", "soc2"},
		StoreDir:     storeDir,
		OutputDir:    outDir,
	}); err != nil {
		t.Fatalf("first bundle build: %v", err)
	}
	first, firstFiles, err := snapshotDigest(outDir)
	if err != nil {
		t.Fatalf("snapshot first: %v", err)
	}

	if _, err := corebundle.Build(corebundle.BuildRequest{
		AuditName:    "Q3-2026",
		FrameworkIDs: []string{"eu-ai-act", "soc2"},
		StoreDir:     storeDir,
		OutputDir:    outDir,
	}); err != nil {
		t.Fatalf("second bundle build: %v", err)
	}
	second, secondFiles, err := snapshotDigest(outDir)
	if err != nil {
		t.Fatalf("snapshot second: %v", err)
	}

	if first != second {
		changed := make([]string, 0)
		for path, left := range firstFiles {
			right, ok := secondFiles[path]
			if !ok {
				changed = append(changed, path+" missing_on_second")
				continue
			}
			if left != right {
				changed = append(changed, path+": "+left+" -> "+right)
			}
		}
		for path := range secondFiles {
			if _, ok := firstFiles[path]; !ok {
				changed = append(changed, path+" added_on_second")
			}
		}
		sort.Strings(changed)
		t.Fatalf("bundle output is not byte-stable\nfirst:  %s\nsecond: %s\nchanges: %v", first, second, changed)
	}
}

func snapshotDigest(root string) (string, map[string]string, error) {
	files := make([]string, 0)
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
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return "", nil, err
	}
	sort.Strings(files)

	h := sha256.New()
	perFile := make(map[string]string, len(files))
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return "", nil, err
		}
		sum := sha256.Sum256(data)
		perFile[rel] = hex.EncodeToString(sum[:])
		h.Write([]byte(rel))
		h.Write([]byte{0})
		h.Write(data)
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), perFile, nil
}
