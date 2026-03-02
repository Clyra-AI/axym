package safety

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	ReasonUnsafePath = "unsafe_operation"
	ExitUnsafePath   = 8
	ManagedMarker    = ".axym-managed"
)

type Error struct {
	ReasonCode string
	Message    string
	ExitCode   int
	Err        error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return fmt.Sprintf("%s: %s", e.ReasonCode, e.Message)
	}
	return fmt.Sprintf("%s: %s: %v", e.ReasonCode, e.Message, e.Err)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// EnsureManagedOutputDir enforces "non-empty + non-managed => fail" with a
// marker file trust contract (marker must be a regular file).
func EnsureManagedOutputDir(path string) error {
	if path == "" {
		return &Error{ReasonCode: ReasonUnsafePath, Message: "output path is required", ExitCode: ExitUnsafePath}
	}
	clean := filepath.Clean(path)
	info, err := os.Lstat(clean)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(clean, 0o700); err != nil {
				return &Error{ReasonCode: ReasonUnsafePath, Message: "create output path", ExitCode: ExitUnsafePath, Err: err}
			}
			return createMarker(filepath.Join(clean, ManagedMarker))
		}
		return &Error{ReasonCode: ReasonUnsafePath, Message: "inspect output path", ExitCode: ExitUnsafePath, Err: err}
	}
	if !info.IsDir() {
		return &Error{ReasonCode: ReasonUnsafePath, Message: "output path is not a directory", ExitCode: ExitUnsafePath}
	}

	entries, err := os.ReadDir(clean)
	if err != nil {
		return &Error{ReasonCode: ReasonUnsafePath, Message: "list output path", ExitCode: ExitUnsafePath, Err: err}
	}
	markerPath := filepath.Join(clean, ManagedMarker)
	if len(entries) == 0 {
		return createMarker(markerPath)
	}

	markerInfo, err := os.Lstat(markerPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Error{ReasonCode: ReasonUnsafePath, Message: "non-empty output path is unmanaged", ExitCode: ExitUnsafePath}
		}
		return &Error{ReasonCode: ReasonUnsafePath, Message: "inspect marker", ExitCode: ExitUnsafePath, Err: err}
	}
	if markerInfo.Mode()&os.ModeSymlink != 0 {
		return &Error{ReasonCode: ReasonUnsafePath, Message: "marker must be a regular file", ExitCode: ExitUnsafePath}
	}
	if markerInfo.IsDir() || !markerInfo.Mode().IsRegular() {
		return &Error{ReasonCode: ReasonUnsafePath, Message: "marker must be a regular file", ExitCode: ExitUnsafePath}
	}
	return nil
}

func createMarker(path string) error {
	if err := os.WriteFile(path, []byte("managed\n"), 0o600); err != nil {
		return &Error{ReasonCode: ReasonUnsafePath, Message: fmt.Sprintf("write marker %s", ManagedMarker), ExitCode: ExitUnsafePath, Err: err}
	}
	return nil
}
