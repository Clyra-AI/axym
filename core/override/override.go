package override

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Clyra-AI/axym/core/store"
	"github.com/Clyra-AI/axym/core/store/dedupe"
	"github.com/Clyra-AI/proof"
)

const (
	ReasonInvalidInput = "OVERRIDE_INVALID_INPUT"
	ReasonAppendFailed = "OVERRIDE_APPEND_FAILED"
)

type Request struct {
	Bundle    string
	Reason    string
	Signer    string
	StoreDir  string
	ExpiresAt time.Time
	Now       func() time.Time
}

type Result struct {
	RecordID     string `json:"record_id"`
	Bundle       string `json:"bundle"`
	Signer       string `json:"signer"`
	Reason       string `json:"reason"`
	ExpiresAt    string `json:"expires_at"`
	ArtifactPath string `json:"artifact_path"`
	HeadHash     string `json:"head_hash"`
	RecordCount  int    `json:"record_count"`
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

func Create(req Request) (Result, error) {
	bundle := strings.TrimSpace(req.Bundle)
	reason := strings.TrimSpace(req.Reason)
	signer := strings.TrimSpace(req.Signer)
	if bundle == "" || reason == "" || signer == "" {
		return Result{}, &Error{ReasonCode: ReasonInvalidInput, Message: "bundle, reason, and signer are required"}
	}
	storeDir := strings.TrimSpace(req.StoreDir)
	if storeDir == "" {
		storeDir = ".axym"
	}
	clock := req.Now
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC().Truncate(time.Second) }
	}
	now := clock().UTC().Truncate(time.Second)
	expiresAt := req.ExpiresAt.UTC().Truncate(time.Second)
	if expiresAt.IsZero() {
		expiresAt = now.Add(24 * time.Hour)
	}
	if !expiresAt.After(now) {
		return Result{}, &Error{ReasonCode: ReasonInvalidInput, Message: "expires_at must be in the future"}
	}

	evidenceStore, err := store.New(store.Config{RootDir: storeDir, ComplianceMode: true})
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonAppendFailed, Message: "initialize local store", Err: err}
	}
	record, err := proof.NewRecord(proof.RecordOpts{
		Source:        "axym-override",
		SourceProduct: "axym",
		Type:          "approval",
		Timestamp:     now,
		Event: map[string]any{
			"kind":       "override",
			"bundle":     bundle,
			"reason":     reason,
			"signer":     signer,
			"expires_at": expiresAt.Format(time.RFC3339),
			"status":     "approved",
			"decision":   "allow",
		},
		Metadata: map[string]any{
			"override":     true,
			"bundle":       bundle,
			"signer":       signer,
			"reason_codes": []string{"OVERRIDE_APPROVED"},
		},
		Controls: proof.Controls{PermissionsEnforced: true, ApprovedScope: "override:" + bundle},
	})
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonAppendFailed, Message: "build override proof record", Err: err}
	}
	key, err := dedupe.BuildKey(record.SourceProduct, record.RecordType, record.Event)
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonAppendFailed, Message: "build override dedupe key", Err: err}
	}
	appendResult, err := evidenceStore.Append(record, key)
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonAppendFailed, Message: "append override proof record", Err: err}
	}

	artifactPath := filepath.Join(storeDir, "overrides", "overrides.jsonl")
	if appendResult.Appended {
		artifact, err := json.Marshal(map[string]any{
			"record_id":            record.RecordID,
			"bundle":               bundle,
			"reason":               reason,
			"signer":               signer,
			"expires_at":           expiresAt.Format(time.RFC3339),
			"timestamp":            now.Format(time.RFC3339),
			"record_hash":          record.Integrity.RecordHash,
			"previous_record_hash": record.Integrity.PreviousRecordHash,
		})
		if err != nil {
			return Result{}, &Error{ReasonCode: ReasonAppendFailed, Message: "marshal override artifact", Err: err}
		}
		if err := appendLine(artifactPath, artifact); err != nil {
			return Result{}, &Error{ReasonCode: ReasonAppendFailed, Message: "persist override artifact", Err: err}
		}
	}

	return Result{
		RecordID:     appendResult.RecordID,
		Bundle:       bundle,
		Signer:       signer,
		Reason:       reason,
		ExpiresAt:    expiresAt.Format(time.RFC3339),
		ArtifactPath: artifactPath,
		HeadHash:     appendResult.HeadHash,
		RecordCount:  appendResult.RecordCount,
	}, nil
}

func appendLine(path string, payload []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := ensureRegularFile(path); err != nil {
		return err
	}
	// #nosec G304 -- override artifact path is derived from the managed store root and regular-file checked.
	fh, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = fh.Close() }()
	if _, err := fh.Write(append(payload, '\n')); err != nil {
		return err
	}
	return fh.Sync()
}

func ensureRegularFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("override artifact path must not be symlink")
	}
	if info.IsDir() {
		return fmt.Errorf("override artifact path must be regular file")
	}
	return nil
}
