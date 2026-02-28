package record

import (
	"errors"
	"fmt"
)

const (
	ReasonInvalidRecord = "invalid_record"
	ReasonSchemaError   = "schema_error"
	ReasonMappingError  = "mapping_error"
)

type InvalidInputError struct {
	Reason  string
	Message string
	Err     error
}

func (e *InvalidInputError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return fmt.Sprintf("%s: %s", e.Reason, e.Message)
	}
	return fmt.Sprintf("%s: %s: %v", e.Reason, e.Message, e.Err)
}

func (e *InvalidInputError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func NewInvalidInputError(reason string, message string, err error) error {
	return &InvalidInputError{Reason: reason, Message: message, Err: err}
}

func IsInvalidInput(err error) bool {
	var target *InvalidInputError
	return errors.As(err, &target)
}

func ReasonCode(err error) string {
	var target *InvalidInputError
	if errors.As(err, &target) {
		return target.Reason
	}
	return ""
}
