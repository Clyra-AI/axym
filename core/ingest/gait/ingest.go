package gait

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Clyra-AI/axym/core/ingest/gait/pack"
	"github.com/Clyra-AI/axym/core/ingest/gait/translate"
	"github.com/Clyra-AI/axym/core/store"
	"github.com/Clyra-AI/axym/core/store/dedupe"
	"github.com/Clyra-AI/proof"
)

const (
	ReasonNoInput             = "NO_INPUT"
	ReasonInvalidInput        = "GAIT_INVALID_INPUT"
	ReasonPackReadFailed      = "GAIT_PACK_READ_FAILED"
	ReasonTranslationFailed   = "GAIT_TRANSLATION_FAILED"
	ReasonAppendFailed        = "GAIT_CHAIN_APPEND_FAILED"
	ReasonUnsupportedNative   = "GAIT_UNSUPPORTED_NATIVE_TYPE"
	ReasonInvalidNativeRecord = "GAIT_INVALID_NATIVE_RECORD"
	ReasonContextCanceled     = "GAIT_CONTEXT_CANCELED"
)

type Request struct {
	InputPaths []string
	Store      *store.Store
}

type Result struct {
	Source       string   `json:"source"`
	InputFiles   int      `json:"input_files"`
	NativeParsed int      `json:"native_parsed"`
	ProofParsed  int      `json:"proof_parsed"`
	Appended     int      `json:"appended"`
	Deduped      int      `json:"deduped"`
	Rejected     int      `json:"rejected"`
	RecordCount  int      `json:"record_count"`
	HeadHash     string   `json:"head_hash,omitempty"`
	ReasonCodes  []string `json:"reason_codes"`
	Translated   int      `json:"translated"`
	Passthrough  int      `json:"passthrough"`
}

type Error struct {
	ReasonCode string
	Message    string
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

func Ingest(ctx context.Context, req Request) (Result, error) {
	if req.Store == nil {
		return Result{}, &Error{ReasonCode: ReasonInvalidInput, Message: "store is required"}
	}
	paths := normalizeInputPaths(req.InputPaths)
	result := Result{
		Source:      "gait",
		InputFiles:  len(paths),
		ReasonCodes: []string{},
	}
	if len(paths) == 0 {
		result.ReasonCodes = []string{ReasonNoInput}
		return result, nil
	}

	for _, path := range paths {
		select {
		case <-ctx.Done():
			return Result{}, &Error{ReasonCode: ReasonContextCanceled, Message: "ingest canceled", Err: ctx.Err()}
		default:
		}

		packResult, err := pack.Read(path)
		if err != nil {
			return Result{}, &Error{ReasonCode: ReasonPackReadFailed, Message: fmt.Sprintf("read gait pack %s", path), Err: err}
		}
		result.ProofParsed += len(packResult.ProofRecords)
		result.Passthrough += len(packResult.ProofRecords)
		result.NativeParsed += len(packResult.NativeRecords)

		for _, passthrough := range packResult.ProofRecords {
			if err := appendRecord(req.Store, passthrough, &result); err != nil {
				return Result{}, err
			}
		}
		for _, native := range packResult.NativeRecords {
			record, err := translate.Translate(native)
			if err != nil {
				result.Rejected++
				result.ReasonCodes = append(result.ReasonCodes, ReasonTranslationFailed)
				if tErr, ok := err.(*translate.Error); ok {
					switch tErr.ReasonCode {
					case translate.ReasonUnsupportedNativeType:
						result.ReasonCodes = append(result.ReasonCodes, ReasonUnsupportedNative)
					case translate.ReasonInvalidNativeRecord:
						result.ReasonCodes = append(result.ReasonCodes, ReasonInvalidNativeRecord)
					}
				}
				continue
			}
			result.Translated++
			if err := appendRecord(req.Store, record, &result); err != nil {
				return Result{}, err
			}
		}
	}

	result.ReasonCodes = uniqueSorted(result.ReasonCodes)
	return result, nil
}

func appendRecord(st *store.Store, record *proof.Record, result *Result) error {
	key, err := dedupe.BuildKey(record.SourceProduct, record.RecordType, record.Event)
	if err != nil {
		result.Rejected++
		result.ReasonCodes = append(result.ReasonCodes, ReasonTranslationFailed)
		return nil
	}
	appendResult, err := st.Append(record, key)
	if err != nil {
		return &Error{ReasonCode: ReasonAppendFailed, Message: "append gait record", Err: err}
	}
	result.RecordCount = appendResult.RecordCount
	result.HeadHash = appendResult.HeadHash
	if appendResult.Deduped {
		result.Deduped++
		return nil
	}
	if appendResult.Appended {
		result.Appended++
		return nil
	}
	result.Rejected++
	result.ReasonCodes = append(result.ReasonCodes, ReasonAppendFailed)
	return nil
}

func normalizeInputPaths(raw []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, candidate := range raw {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func uniqueSorted(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, reason := range in {
		if reason == "" {
			continue
		}
		if _, ok := seen[reason]; ok {
			continue
		}
		seen[reason] = struct{}{}
		out = append(out, reason)
	}
	sort.Strings(out)
	return out
}
