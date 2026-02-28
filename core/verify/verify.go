package verify

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
	Path  string `json:"path"`
	Files int    `json:"files"`
	Algo  string `json:"algo"`
}

func VerifyChainFromStoreDir(storeDir string) (ChainResult, error) {
	chainPath := filepath.Join(storeDir, "chain.json")
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

func VerifyBundle(path string) (BundleResult, error) {
	manifest, err := proof.VerifyBundle(path, proof.BundleVerifyOpts{})
	if err != nil {
		return BundleResult{}, &Error{ReasonCode: ReasonBundleVerify, Message: "bundle verification failed", ExitCode: 2, Err: err}
	}
	return BundleResult{Path: path, Files: len(manifest.Files), Algo: manifest.AlgoID}, nil
}
