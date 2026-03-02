package verify

import (
	"errors"

	"github.com/Clyra-AI/axym/core/export/safety"
)

func EnsureManagedTempDir(path string) error {
	err := safety.EnsureManagedOutputDir(path)
	if err == nil {
		return nil
	}
	var sErr *safety.Error
	if !errors.As(err, &sErr) {
		return err
	}
	return &Error{
		ReasonCode: sErr.ReasonCode,
		Message:    sErr.Message,
		ExitCode:   sErr.ExitCode,
		Err:        sErr.Err,
	}
}
