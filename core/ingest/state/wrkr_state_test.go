package state

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWithLockedStatePersistsDeterministicBaseline(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager := NewWrkrManager(root)

	err := manager.WithLockedState(func(st *WrkrState) error {
		st.PrivilegeBaseline["agent-a"] = []string{"write", "read", "read"}
		return nil
	})
	if err != nil {
		t.Fatalf("WithLockedState: %v", err)
	}

	raw, err := os.ReadFile(manager.StatePath())
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("expected state file contents")
	}
}

func TestWithLockedStateReturnsErrStateLocked(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager := NewWrkrManager(root)
	lockPath := filepath.Join(root, wrkrLockFile)
	if err := os.WriteFile(lockPath, []byte("held"), 0o600); err != nil {
		t.Fatalf("write lock file: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(lockPath) })

	err := manager.WithLockedState(func(st *WrkrState) error { return nil })
	if !errors.Is(err, ErrStateLocked) {
		t.Fatalf("expected ErrStateLocked, got %v", err)
	}
}
