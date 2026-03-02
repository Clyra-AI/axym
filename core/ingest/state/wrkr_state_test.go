package state

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestWithLockedStateRecoversFromStaleLock(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager := NewWrkrManager(root)
	lockPath := filepath.Join(root, wrkrLockFile)
	if err := os.WriteFile(lockPath, []byte("stale"), 0o600); err != nil {
		t.Fatalf("write lock file: %v", err)
	}

	old := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(lockPath, old, old); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
	manager.now = func() time.Time {
		return old.Add(manager.lockTTL + time.Minute)
	}

	err := manager.WithLockedState(func(st *WrkrState) error {
		st.PrivilegeBaseline["agent-a"] = []string{"read"}
		return nil
	})
	if err != nil {
		t.Fatalf("WithLockedState: %v", err)
	}
	if _, statErr := os.Stat(manager.StatePath()); statErr != nil {
		t.Fatalf("expected state file after stale-lock recovery: %v", statErr)
	}
}

func TestWithLockedStateDoesNotReclaimActiveLongRunningLock(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	holder := NewWrkrManager(root)
	holder.lockTTL = 150 * time.Millisecond
	holder.lockHeartbeat = 20 * time.Millisecond

	entered := make(chan struct{})
	release := make(chan struct{})
	holderErr := make(chan error, 1)
	go func() {
		holderErr <- holder.WithLockedState(func(st *WrkrState) error {
			close(entered)
			<-release
			return nil
		})
	}()

	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for holder lock acquisition")
	}

	time.Sleep(250 * time.Millisecond)

	contender := NewWrkrManager(root)
	contender.lockTTL = 150 * time.Millisecond
	contender.lockHeartbeat = 20 * time.Millisecond

	err := contender.WithLockedState(func(st *WrkrState) error { return nil })
	if !errors.Is(err, ErrStateLocked) {
		t.Fatalf("expected ErrStateLocked for active long-running lock, got %v", err)
	}

	close(release)
	select {
	case err := <-holderErr:
		if err != nil {
			t.Fatalf("holder lock failed: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for holder completion")
	}
}
