package verify

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const managedMarker = ".axym-managed"

func EnsureManagedTempDir(path string) error {
	if path == "" {
		return &Error{ReasonCode: ReasonUnsafePath, Message: "temp path is required", ExitCode: 8}
	}
	clean := filepath.Clean(path)
	info, err := os.Lstat(clean)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(clean, 0o700); err != nil {
				return &Error{ReasonCode: ReasonUnsafePath, Message: "create temp path", ExitCode: 8, Err: err}
			}
			return createMarker(filepath.Join(clean, managedMarker))
		}
		return &Error{ReasonCode: ReasonUnsafePath, Message: "inspect temp path", ExitCode: 8, Err: err}
	}
	if !info.IsDir() {
		return &Error{ReasonCode: ReasonUnsafePath, Message: "temp path is not a directory", ExitCode: 8}
	}

	entries, err := os.ReadDir(clean)
	if err != nil {
		return &Error{ReasonCode: ReasonUnsafePath, Message: "list temp path", ExitCode: 8, Err: err}
	}
	markerPath := filepath.Join(clean, managedMarker)
	if len(entries) == 0 {
		return createMarker(markerPath)
	}

	markerInfo, err := os.Lstat(markerPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Error{ReasonCode: ReasonUnsafePath, Message: "non-empty temp path is unmanaged", ExitCode: 8}
		}
		return &Error{ReasonCode: ReasonUnsafePath, Message: "inspect marker", ExitCode: 8, Err: err}
	}
	if markerInfo.Mode()&os.ModeSymlink != 0 {
		return &Error{ReasonCode: ReasonUnsafePath, Message: "marker must be a regular file", ExitCode: 8}
	}
	if markerInfo.IsDir() {
		return &Error{ReasonCode: ReasonUnsafePath, Message: "marker must be a regular file", ExitCode: 8}
	}
	return nil
}

func createMarker(path string) error {
	if err := os.WriteFile(path, []byte("managed\n"), 0o600); err != nil {
		return &Error{ReasonCode: ReasonUnsafePath, Message: fmt.Sprintf("write marker %s", managedMarker), ExitCode: 8, Err: err}
	}
	return nil
}
