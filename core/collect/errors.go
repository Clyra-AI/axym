package collect

import "fmt"

const (
	ReasonCollectorError  = "COLLECTOR_ERROR"
	ReasonContextCanceled = "COLLECT_CONTEXT_CANCELED"
	ReasonMalformed       = "MALFORMED_PAYLOAD"
	ReasonRuntime         = "COLLECT_RUNTIME_ERROR"
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

type reasonCoder interface {
	ReasonCode() string
}
