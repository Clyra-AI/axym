package safety

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEnsureManagedOutputDirRejectsNonManagedNonEmptyDirectory(t *testing.T) {
	t.Parallel()

	out := filepath.Join(t.TempDir(), "out")
	if err := os.MkdirAll(out, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(out, "foreign.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := EnsureManagedOutputDir(out)
	if err == nil {
		t.Fatal("expected error for unmanaged non-empty directory")
	}
	sErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if sErr.ExitCode != ExitUnsafePath {
		t.Fatalf("exit mismatch: got %d want %d", sErr.ExitCode, ExitUnsafePath)
	}
}

func TestEnsureManagedOutputDirCreatesMarkerForEmptyDirectory(t *testing.T) {
	t.Parallel()

	out := filepath.Join(t.TempDir(), "out")
	if err := os.MkdirAll(out, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := EnsureManagedOutputDir(out); err != nil {
		t.Fatalf("EnsureManagedOutputDir: %v", err)
	}
	info, err := os.Lstat(filepath.Join(out, ManagedMarker))
	if err != nil {
		t.Fatalf("marker stat: %v", err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("marker mode mismatch: %v", info.Mode())
	}
}

func TestEnsureManagedOutputDirRejectsSymlinkMarker(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on some Windows environments")
	}

	out := filepath.Join(t.TempDir(), "out")
	if err := os.MkdirAll(out, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(out, "payload.json"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("WriteFile payload: %v", err)
	}
	target := filepath.Join(out, "target.txt")
	if err := os.WriteFile(target, []byte("managed"), 0o600); err != nil {
		t.Fatalf("WriteFile target: %v", err)
	}
	if err := os.Symlink(target, filepath.Join(out, ManagedMarker)); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	err := EnsureManagedOutputDir(out)
	if err == nil {
		t.Fatal("expected symlink marker rejection")
	}
}
