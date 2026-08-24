package gait

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Clyra-AI/axym/core/ingest/gait/evidence"
	"github.com/Clyra-AI/axym/core/ingest/gait/pack"
	"github.com/Clyra-AI/axym/core/ingest/gait/translate"
	"github.com/Clyra-AI/axym/core/normalize"
	"github.com/Clyra-AI/axym/core/store"
	"github.com/Clyra-AI/axym/core/store/dedupe"
	"github.com/Clyra-AI/proof"
)

const (
	ReasonNoInput                       = "NO_INPUT"
	ReasonInvalidInput                  = "GAIT_INVALID_INPUT"
	ReasonPackReadFailed                = "GAIT_PACK_READ_FAILED"
	ReasonTranslationFailed             = "GAIT_TRANSLATION_FAILED"
	ReasonAppendFailed                  = "GAIT_CHAIN_APPEND_FAILED"
	ReasonUnsupportedNative             = "GAIT_UNSUPPORTED_NATIVE_TYPE"
	ReasonInvalidNativeRecord           = "GAIT_INVALID_NATIVE_RECORD"
	ReasonContextCanceled               = "GAIT_CONTEXT_CANCELED"
	ReasonLifecycleVerificationRequired = "GAIT_LIFECYCLE_VERIFICATION_REQUIRED"
	ReasonLifecycleVerificationFailed   = "GAIT_LIFECYCLE_VERIFICATION_FAILED"
)

type Request struct {
	InputPaths            []string
	Store                 *store.Store
	LifecycleVerification *evidence.VerificationOptions
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
	LifecycleVerified         int                      `json:"lifecycle_verified"`
	LifecycleAuthoritative    int                      `json:"lifecycle_authoritative"`
	LifecycleTranslated       int                      `json:"lifecycle_translated"`
	LifecycleEvidenceSets     []evidence.EvidenceSet   `json:"lifecycle_evidence_sets,omitempty"`
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
	ReasonCode  string
	Message     string
	ReasonCodes []string
	Err         error
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

type stagedPack struct {
	passthrough []*proof.Record
	native      []*proof.Record
	lifecycle   []*proof.Record
	artifacts   []*proof.Record
}

func Ingest(ctx context.Context, req Request) (Result, error) {
	if req.Store == nil {
		return Result{}, &Error{ReasonCode: ReasonInvalidInput, Message: "store is required"}
	}
	paths := normalizeInputPaths(req.InputPaths)
	result := Result{Source: "gait", InputFiles: len(paths), ReasonCodes: []string{}}
	if len(paths) == 0 {
		result.ReasonCodes = []string{ReasonNoInput}
		return result, nil
	}
	chain, err := req.Store.LoadChain()
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonAppendFailed, Message: "load chain for gait correlation", Err: err}
	}
	correlationRecords := append([]proof.Record(nil), chain.Records...)
	staged := make([]stagedPack, 0, len(paths))
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
		stage := stagedPack{passthrough: append([]*proof.Record(nil), packResult.ProofRecords...)}
		if len(packResult.LifecyclePacks) > 0 {
			if req.LifecycleVerification == nil {
				return Result{}, &Error{ReasonCode: ReasonLifecycleVerificationRequired, Message: "lifecycle evidence requires explicit trusted verification options"}
			}
			for _, lifecycle := range packResult.LifecyclePacks {
				verification := evidence.VerifyLifecyclePack(lifecycle, *req.LifecycleVerification)
				result.ReasonCodes = append(result.ReasonCodes, verification.ReasonCodes...)
				if !verification.Valid {
					return Result{}, &Error{ReasonCode: ReasonLifecycleVerificationFailed, Message: "lifecycle evidence verification failed", ReasonCodes: append([]string(nil), verification.ReasonCodes...)}
				}
				result.LifecycleVerified++
				result.LifecycleEvidenceSets = append(result.LifecycleEvidenceSets, verification.EvidenceSet)
				if verification.Authoritative {
					result.LifecycleAuthoritative++
					record, translateErr := translate.TranslateLifecycle(verification, lifecycle)
					if translateErr != nil {
						return Result{}, wrapTranslateError("translate verified Gait lifecycle evidence", translateErr)
					}
					stage.lifecycle = append(stage.lifecycle, record)
					result.LifecycleTranslated++
					result.Translated++
				}
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
			stage.native = append(stage.native, record)
		}
		for _, passthrough := range stage.passthrough {
			if passthrough != nil {
				correlationRecords = append(correlationRecords, *passthrough)
			}
		}
		for _, record := range stage.native {
			if record != nil {
				correlationRecords = append(correlationRecords, *record)
			}
		}
		for _, record := range stage.lifecycle {
			if record != nil {
				correlationRecords = append(correlationRecords, *record)
			}
		}
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
			stage.artifacts = append(stage.artifacts, record)
			correlationRecords = append(correlationRecords, *record)
		}
		staged = append(staged, stage)
	}
	for _, stage := range staged {
		ordered := append(append(append(append([]*proof.Record{}, stage.passthrough...), stage.native...), stage.lifecycle...), stage.artifacts...)
		for _, record := range ordered {
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
	var lifecycleErr *pack.LifecycleError
	if errors.As(err, &lifecycleErr) {
		return &Error{ReasonCode: lifecycleErr.ReasonCode, Message: fmt.Sprintf("read gait pack %s", path), Err: lifecycleErr}
	}
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
