package store

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestShouldIgnoreWindowsDirSyncError(t *testing.T) {
	t.Parallel()

	tempChild := filepath.Join(os.TempDir(), "axym-test")
	outside := filepath.Join(string(os.PathSeparator), "var", "lib", "axym")
	permErr := &os.PathError{Op: "sync", Path: tempChild, Err: os.ErrPermission}

	if !shouldIgnoreWindowsDirSyncError("windows", tempChild, permErr) {
		t.Fatal("expected windows temp permission error to be ignored")
	}
	if shouldIgnoreWindowsDirSyncError("windows", outside, permErr) {
		t.Fatal("expected non-temp path permission error to be enforced")
	}
	nonPermErr := &os.PathError{Op: "sync", Path: tempChild, Err: errors.New("other")}
	if shouldIgnoreWindowsDirSyncError("windows", tempChild, nonPermErr) {
		t.Fatal("expected non-permission error to be enforced")
	}
	if shouldIgnoreWindowsDirSyncError(runtime.GOOS, tempChild, permErr) && runtime.GOOS != "windows" {
		t.Fatal("expected non-windows runtime to enforce permission error")
	}
}
