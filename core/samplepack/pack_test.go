package samplepack

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateWritesDeterministicAssets(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "axym-sample")

	result, err := Create(target)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if result.Path != target {
		t.Fatalf("path mismatch: got %s want %s", result.Path, target)
	}
	if len(result.Files) != len(sampleAssets) {
		t.Fatalf("file count mismatch: got %d want %d", len(result.Files), len(sampleAssets))
	}
	for i, item := range sampleAssets {
		got := result.Files[i]
		wantPath := filepath.Join(target, filepath.FromSlash(item.RelPath))
		if got.Path != wantPath {
			t.Fatalf("file path mismatch: got %s want %s", got.Path, wantPath)
		}
		raw, err := os.ReadFile(wantPath)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", wantPath, err)
		}
		if string(raw) != item.Contents {
			t.Fatalf("asset mismatch for %s", item.RelPath)
		}
	}
	if len(result.NextSteps) != 7 {
		t.Fatalf("next step count mismatch: %d", len(result.NextSteps))
	}
}

func TestCreateRejectsInvalidOrExistingTarget(t *testing.T) {
	t.Parallel()

	if _, err := Create("."); err == nil || !strings.Contains(err.Error(), "current directory") {
		t.Fatalf("expected current directory validation error, got %v", err)
	}

	root := t.TempDir()
	target := filepath.Join(root, "axym-sample")
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if _, err := Create(target); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected existing target error, got %v", err)
	}
}

func TestCreateCleansTempDirWhenFinalizeFails(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "axym-sample")
	ops := defaultFileOps()
	var tempDir string
	ops.rename = func(oldPath string, newPath string) error {
		tempDir = oldPath
		return errors.New("rename blocked")
	}

	_, err := createWithOps(target, ops)
	if err == nil || !strings.Contains(err.Error(), "rename blocked") {
		t.Fatalf("expected rename failure, got %v", err)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("expected target to be absent, got err=%v", statErr)
	}
	if tempDir == "" {
		t.Fatal("expected temp dir capture")
	}
	if _, statErr := os.Stat(tempDir); !os.IsNotExist(statErr) {
		t.Fatalf("expected temp dir cleanup, got err=%v", statErr)
	}
}
