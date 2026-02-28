package collectorerr

import "fmt"

type Error struct {
	reason string
	msg    string
	err    error
}

func New(reason string, msg string, err error) *Error {
	return &Error{reason: reason, msg: msg, err: err}
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.err == nil {
		return fmt.Sprintf("%s: %s", e.reason, e.msg)
	}
	return fmt.Sprintf("%s: %s: %v", e.reason, e.msg, e.err)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func (e *Error) ReasonCode() string {
	if e == nil {
		return ""
	}
	return e.reason
}
