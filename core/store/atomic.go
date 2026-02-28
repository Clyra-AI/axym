package store

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func WriteJSONAtomic(path string, data []byte, fsync bool) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, defaultDirPerm); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".axym-tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}
	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("write temp file: %w", err)
	}
	if fsync {
		if err := tmp.Sync(); err != nil {
			cleanup()
			return fmt.Errorf("sync temp file: %w", err)
		}
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("rename temp file: %w", err)
	}
	if fsync {
		if err := syncDir(dir); err != nil {
			return err
		}
	}
	return nil
}

func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open directory for sync: %w", err)
	}
	defer func() { _ = dir.Close() }()
	if err := dir.Sync(); err != nil {
		// Windows runners may deny directory sync on temp paths; writes are still durable to file path.
		if runtime.GOOS == "windows" && os.IsPermission(err) {
			return nil
		}
		return fmt.Errorf("sync directory: %w", err)
	}
	return nil
}
