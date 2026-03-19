package verify

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	bundleverify "github.com/Clyra-AI/axym/core/verify/bundle"
	"github.com/Clyra-AI/proof"
)

const (
	ReasonChainTamper  = "chain_tamper_detected"
	ReasonChainRead    = "chain_read_failed"
	ReasonBundleVerify = "bundle_verify_failed"
	ReasonUnsafePath   = "unsafe_operation"
)

type Error struct {
	ReasonCode string
	Message    string
	ExitCode   int
	BreakIndex int
	BreakPoint string
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

type ChainResult struct {
	Intact     bool   `json:"intact"`
	Count      int    `json:"count"`
	HeadHash   string `json:"head_hash,omitempty"`
	BreakIndex int    `json:"break_index,omitempty"`
	BreakPoint string `json:"break_point,omitempty"`
}

type BundleResult struct {
	Path               string         `json:"path"`
	Files              int            `json:"files"`
	Algo               string         `json:"algo"`
	Cryptographic      bool           `json:"cryptographic"`
	ComplianceVerified bool           `json:"compliance_verified"`
	OSCALValid         bool           `json:"oscal_valid"`
	Compliance         map[string]any `json:"compliance,omitempty"`
}

func VerifyChainFromStoreDir(storeDir string) (ChainResult, error) {
	chainPath := filepath.Join(storeDir, "chain.json")
	// #nosec G304 -- chain path is derived from the explicit store directory contract.
	raw, err := os.ReadFile(chainPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ChainResult{Intact: true, Count: 0}, nil
		}
		return ChainResult{}, &Error{ReasonCode: ReasonChainRead, Message: "read chain", ExitCode: 2, Err: err}
	}
	var chain proof.Chain
	if err := json.Unmarshal(raw, &chain); err != nil {
		return ChainResult{}, &Error{ReasonCode: ReasonChainRead, Message: "decode chain", ExitCode: 2, Err: err}
	}
	verification, err := proof.VerifyChain(&chain)
	if err != nil {
		return ChainResult{}, &Error{ReasonCode: ReasonChainRead, Message: "verify chain", ExitCode: 2, Err: err}
	}
	if !verification.Intact {
		return ChainResult{}, &Error{
			ReasonCode: ReasonChainTamper,
			Message:    "chain integrity check failed",
			ExitCode:   2,
			BreakIndex: verification.BreakIndex,
			BreakPoint: verification.BreakPoint,
		}
	}
	return ChainResult{
		Intact:   true,
		Count:    verification.Count,
		HeadHash: verification.HeadHash,
	}, nil
}

func VerifyBundle(path string, frameworkIDs []string) (BundleResult, error) {
	result, err := bundleverify.Verify(path, frameworkIDs)
	if err != nil {
		var bErr *bundleverify.Error
		if errors.As(err, &bErr) {
			return BundleResult{}, &Error{
				ReasonCode: bErr.ReasonCode,
				Message:    bErr.Message,
				ExitCode:   bErr.ExitCode,
				Err:        bErr.Err,
			}
		}
		return BundleResult{}, &Error{ReasonCode: ReasonBundleVerify, Message: "bundle verification failed", ExitCode: 2, Err: err}
	}

	compliance := map[string]any{
		"required_record_types":   result.Compliance.RequiredRecordTypes,
		"observed_record_types":   result.Compliance.ObservedRecordTypes,
		"missing_record_types":    result.Compliance.MissingRecordTypes,
		"incomplete_controls":     result.Compliance.IncompleteControls,
		"controls_missing_fields": result.Compliance.ControlsMissing,
		"complete":                result.Compliance.Complete,
		"grade":                   result.Compliance.Grade,
		"identity_governance":     result.Compliance.IdentityGovernance,
	}
	if !result.ComplianceVerified {
		compliance = nil
	}
	return BundleResult{
		Path:               result.Path,
		Files:              result.Files,
		Algo:               result.Algo,
		Cryptographic:      result.Cryptographic,
		ComplianceVerified: result.ComplianceVerified,
		OSCALValid:         result.OSCALValid,
		Compliance:         compliance,
	}, nil
}
