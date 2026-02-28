package proofemit

import (
	"fmt"

	"github.com/Clyra-AI/axym/core/normalize"
	"github.com/Clyra-AI/axym/core/policy/sink"
	"github.com/Clyra-AI/axym/core/record"
	"github.com/Clyra-AI/axym/core/redact"
	"github.com/Clyra-AI/axym/core/store"
	"github.com/Clyra-AI/axym/core/store/dedupe"
	"github.com/Clyra-AI/proof"
)

type Emitter struct {
	Store    *store.Store
	SinkMode sink.Mode
}

type EmitInput struct {
	Normalized normalize.Input
	Redaction  redact.Config
}

type Result struct {
	Record      *proof.Record
	Appended    bool
	Deduped     bool
	HeadHash    string
	RecordCount int
	Degraded    bool
	ReasonCode  string
	Message     string
}

type SinkFailureError struct {
	ReasonCode string
	Message    string
	Err        error
}

func (e *SinkFailureError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return e.ReasonCode + ": " + e.Message
	}
	return e.ReasonCode + ": " + e.Message + ": " + e.Err.Error()
}

func (e *SinkFailureError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (em *Emitter) Emit(input EmitInput) (Result, error) {
	if em.Store == nil {
		return Result{}, fmt.Errorf("store is required")
	}

	rec, err := record.NormalizeAndBuild(input.Normalized, input.Redaction)
	if err != nil {
		return Result{}, err
	}
	key, err := dedupe.BuildKey(rec.SourceProduct, rec.RecordType, rec.Event)
	if err != nil {
		return Result{}, fmt.Errorf("build dedupe key: %w", err)
	}

	appendResult, err := em.Store.Append(rec, key)
	if err != nil {
		decision := sink.OnSinkFailure(em.SinkMode, err)
		if !decision.Allow {
			return Result{}, &SinkFailureError{
				ReasonCode: decision.ReasonCode,
				Message:    decision.Message,
				Err:        err,
			}
		}
		return Result{
			Record:     rec,
			Appended:   false,
			Deduped:    false,
			Degraded:   true,
			ReasonCode: decision.ReasonCode,
			Message:    decision.Message,
		}, nil
	}

	return Result{
		Record:      rec,
		Appended:    appendResult.Appended,
		Deduped:     appendResult.Deduped,
		HeadHash:    appendResult.HeadHash,
		RecordCount: appendResult.RecordCount,
	}, nil
}
