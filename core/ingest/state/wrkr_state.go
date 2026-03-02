package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	wrkrStateFile = "wrkr-last-ingest.json"
	wrkrLockFile  = "wrkr-last-ingest.lock"
	wrkrLockTTL   = 5 * time.Minute
)

var ErrStateLocked = errors.New("wrkr ingest state is locked")

type WrkrState struct {
	Version           string              `json:"version"`
	UpdatedAt         string              `json:"updated_at"`
	PrivilegeBaseline map[string][]string `json:"privilege_baseline"`
}

type WrkrManager struct {
	rootDir string
	now     func() time.Time
}

func NewWrkrManager(rootDir string) *WrkrManager {
	return &WrkrManager{
		rootDir: rootDir,
		now:     func() time.Time { return time.Now().UTC().Truncate(time.Second) },
	}
}

func (m *WrkrManager) StatePath() string {
	return filepath.Join(m.rootDir, wrkrStateFile)
}

func (m *WrkrManager) WithLockedState(fn func(*WrkrState) error) error {
	if fn == nil {
		return fmt.Errorf("state callback is required")
	}
	if err := os.MkdirAll(m.rootDir, 0o700); err != nil {
		return fmt.Errorf("create wrkr state directory: %w", err)
	}

	lockPath := filepath.Join(m.rootDir, wrkrLockFile)
	lockFile, err := m.acquireLock(lockPath)
	if err != nil {
		return err
	}
	_ = lockFile.Close()
	defer func() { _ = os.Remove(lockPath) }()

	state, err := m.load()
	if err != nil {
		return err
	}
	if err := fn(state); err != nil {
		return err
	}

	state.Version = "v1"
	state.UpdatedAt = m.now().Format(time.RFC3339)
	normalizeBaseline(state)
	return writeJSONAtomic(m.StatePath(), state)
}

func (m *WrkrManager) acquireLock(lockPath string) (*os.File, error) {
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err == nil {
		return lockFile, nil
	}
	if !errors.Is(err, os.ErrExist) {
		return nil, fmt.Errorf("acquire wrkr state lock: %w", err)
	}

	stale, staleErr := m.lockIsStale(lockPath)
	if staleErr != nil {
		return nil, fmt.Errorf("inspect wrkr state lock: %w", staleErr)
	}
	if !stale {
		return nil, ErrStateLocked
	}
	if err := os.Remove(lockPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("remove stale wrkr state lock: %w", err)
	}
	lockFile, err = os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, ErrStateLocked
		}
		return nil, fmt.Errorf("acquire wrkr state lock: %w", err)
	}
	return lockFile, nil
}

func (m *WrkrManager) lockIsStale(lockPath string) (bool, error) {
	info, err := os.Stat(lockPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	age := m.now().Sub(info.ModTime().UTC())
	return age > wrkrLockTTL, nil
}

func (m *WrkrManager) load() (*WrkrState, error) {
	raw, err := os.ReadFile(m.StatePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &WrkrState{
				Version:           "v1",
				PrivilegeBaseline: map[string][]string{},
			}, nil
		}
		return nil, fmt.Errorf("read wrkr ingest state: %w", err)
	}

	var state WrkrState
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, fmt.Errorf("decode wrkr ingest state: %w", err)
	}
	if state.PrivilegeBaseline == nil {
		state.PrivilegeBaseline = map[string][]string{}
	}
	normalizeBaseline(&state)
	return &state, nil
}

func normalizeBaseline(state *WrkrState) {
	if state == nil {
		return
	}
	for principal, privileges := range state.PrivilegeBaseline {
		if len(privileges) == 0 {
			state.PrivilegeBaseline[principal] = []string{}
			continue
		}
		seen := make(map[string]struct{}, len(privileges))
		normalized := make([]string, 0, len(privileges))
		for _, privilege := range privileges {
			if privilege == "" {
				continue
			}
			if _, ok := seen[privilege]; ok {
				continue
			}
			seen[privilege] = struct{}{}
			normalized = append(normalized, privilege)
		}
		sort.Strings(normalized)
		state.PrivilegeBaseline[principal] = normalized
	}
}

func writeJSONAtomic(path string, payload any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".wrkr-state-*")
	if err != nil {
		return fmt.Errorf("create state temp file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}
	if _, err := tmp.Write(raw); err != nil {
		cleanup()
		return fmt.Errorf("write state temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close state temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("rename state temp file: %w", err)
	}
	return nil
}
