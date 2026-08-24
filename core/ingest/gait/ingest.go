package gait

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Clyra-AI/axym/core/ingest/gait/pack"
	"github.com/Clyra-AI/axym/core/ingest/gait/translate"
	"github.com/Clyra-AI/axym/core/normalize"
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
	Source                    string                   `json:"source"`
	InputFiles                int                      `json:"input_files"`
	NativeParsed              int                      `json:"native_parsed"`
	ProofParsed               int                      `json:"proof_parsed"`
	AuthorizationBundles      int                      `json:"authorization_bundles"`
	AuthorizationProfiles     int                      `json:"authorization_profiles"`
	ControlArtifacts          int                      `json:"control_artifacts"`
	LifecycleParsed           int                      `json:"lifecycle_parsed"`
	Appended                  int                      `json:"appended"`
	Deduped                   int                      `json:"deduped"`
	Rejected                  int                      `json:"rejected"`
	RecordCount               int                      `json:"record_count"`
	HeadHash                  string                   `json:"head_hash,omitempty"`
	ReasonCodes               []string                 `json:"reason_codes"`
	Translated                int                      `json:"translated"`
	SourceArtifactsTranslated int                      `json:"source_artifacts_translated"`
	Passthrough               int                      `json:"passthrough"`
	IdentityViews             []normalize.IdentityView `json:"identity_views,omitempty"`
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
			return Result{}, wrapPackError(path, err)
		}
		result.ProofParsed += len(packResult.ProofRecords)
		result.Passthrough += len(packResult.ProofRecords)
		result.NativeParsed += len(packResult.NativeRecords)
		result.LifecycleParsed += len(packResult.LifecyclePacks)
		result.ReasonCodes = append(result.ReasonCodes, packResult.ReasonCodes...)

		translatedNative := make([]*proof.Record, 0, len(packResult.NativeRecords))
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
			translatedNative = append(translatedNative, record)
		}

		chain, err := req.Store.LoadChain()
		if err != nil {
			return Result{}, &Error{ReasonCode: ReasonAppendFailed, Message: "load chain for gait correlation", Err: err}
		}
		correlationRecords := append([]proof.Record(nil), chain.Records...)
		for _, passthrough := range packResult.ProofRecords {
			if passthrough == nil {
				continue
			}
			correlationRecords = append(correlationRecords, *passthrough)
		}
		for _, translated := range translatedNative {
			if translated == nil {
				continue
			}
			correlationRecords = append(correlationRecords, *translated)
		}
		translatedArtifacts := make([]*proof.Record, 0, len(packResult.Artifacts))
		for _, artifact := range packResult.Artifacts {
			incrementArtifactCounts(&result, artifact)
			if err := translate.ValidateSourceArtifactLinks(artifact, correlationRecords); err != nil {
				return Result{}, wrapTranslateError("validate gait source artifact links", err)
			}
			record, err := translate.TranslateSourceArtifact(artifact)
			if err != nil {
				return Result{}, wrapTranslateError("translate gait source artifact", err)
			}
			result.SourceArtifactsTranslated++
			translatedArtifacts = append(translatedArtifacts, record)
			correlationRecords = append(correlationRecords, *record)
		}

		for _, passthrough := range packResult.ProofRecords {
			appendIdentityView(&result, normalize.IdentityViewFromRecord(passthrough))
			if err := appendRecord(req.Store, passthrough, &result); err != nil {
				return Result{}, err
			}
		}
		for _, record := range translatedNative {
			appendIdentityView(&result, normalize.IdentityViewFromRecord(record))
			if err := appendRecord(req.Store, record, &result); err != nil {
				return Result{}, err
			}
		}
		for _, record := range translatedArtifacts {
			appendIdentityView(&result, normalize.IdentityViewFromRecord(record))
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

func appendIdentityView(result *Result, view normalize.IdentityView) {
	if result == nil || view.Empty() {
		return
	}
	result.IdentityViews = append(result.IdentityViews, view)
}

func wrapPackError(path string, err error) error {
	var tErr *translate.Error
	if errors.As(err, &tErr) {
		return &Error{ReasonCode: tErr.ReasonCode, Message: fmt.Sprintf("read gait pack %s", path), Err: tErr}
	}
	return &Error{ReasonCode: ReasonPackReadFailed, Message: fmt.Sprintf("read gait pack %s", path), Err: err}
}

func wrapTranslateError(message string, err error) error {
	var tErr *translate.Error
	if errors.As(err, &tErr) {
		return &Error{ReasonCode: tErr.ReasonCode, Message: message, Err: tErr}
	}
	return &Error{ReasonCode: ReasonTranslationFailed, Message: message, Err: err}
}

func incrementArtifactCounts(result *Result, artifact translate.SourceArtifact) {
	if result == nil {
		return
	}
	switch artifact.Kind() {
	case translate.ArtifactTypeAuthorizationBundle:
		result.AuthorizationBundles++
	case translate.ArtifactTypeAuthorizationProfile:
		result.AuthorizationProfiles++
	default:
		result.ControlArtifacts++
	}
}
