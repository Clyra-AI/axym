package bundle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildRequiresAuditName(t *testing.T) {
	t.Parallel()

	_, err := Build(BuildRequest{})
	if err == nil {
		t.Fatal("expected audit validation error")
	}
	bErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if bErr.ExitCode != 6 {
		t.Fatalf("exit mismatch: got %d want 6", bErr.ExitCode)
	}
}

func TestBuildRejectsUnmanagedOutputPath(t *testing.T) {
	t.Parallel()

	outDir := filepath.Join(t.TempDir(), "bundle")
	if err := os.MkdirAll(outDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "foreign.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := Build(BuildRequest{
		AuditName: "Q3-2026",
		OutputDir: outDir,
		StoreDir:  filepath.Join(t.TempDir(), "store"),
	})
	if err == nil {
		t.Fatal("expected unmanaged output rejection")
	}
	bErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if bErr.ExitCode != 8 {
		t.Fatalf("exit mismatch: got %d want 8", bErr.ExitCode)
	}
}
