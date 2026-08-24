package evidence

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	gaitschema "github.com/Clyra-AI/axym/schemas/v1/gait"
	proofsign "github.com/Clyra-AI/proof/signing"
)

const (
	VerificationConfigMaxBytes = 1 << 20
	ReasonConfigInvalid        = "gait_verification_config_invalid"
)

// verificationConfig is intentionally a file-backed, caller-owned boundary.
// Public keys and all lineage bindings must be supplied by the caller; no
// producer artifact is trusted to select its own verification context.
type verificationConfig struct {
	TrustedPublicKey        string `json:"trusted_public_key"`
	ExpectedContract        Ref    `json:"expected_contract"`
	ExpectedFamily          string `json:"expected_family"`
	ExpectedRevision        int    `json:"expected_revision"`
	ExpectedActivation      Ref    `json:"expected_activation"`
	ExpectedRuntimeDigest   string `json:"expected_runtime_digest"`
	ExpectedReadinessDigest string `json:"expected_readiness_digest"`
	ExpectedPolicyDigest    string `json:"expected_policy_digest"`
	ExpectedTarget          string `json:"expected_target"`
	ExpectedEnvironment     string `json:"expected_environment"`
	ExpectedProducerVersion string `json:"expected_producer_version"`
	ExpectedSourceCommit    string `json:"expected_source_commit"`
	ExpectedLifecycleDigest string `json:"expected_lifecycle_digest"`
	EvaluationTime          string `json:"evaluation_time"`
	ActivationNotBefore     string `json:"activation_not_before"`
	ActivationNotAfter      string `json:"activation_not_after"`
	AllowFixtureOnly        bool   `json:"allow_fixture_only"`
}

// LoadVerificationConfig loads a strict JSON verification context. The file
// is configuration, not evidence: its key and bindings are caller-owned.
func LoadVerificationConfig(path string) (VerificationOptions, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return VerificationOptions{}, configError("path is required")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return VerificationOptions{}, configError("regular non-symlink file required")
	}
	if info.Size() > VerificationConfigMaxBytes {
		return VerificationOptions{}, configError("configuration is too large")
	}
	// #nosec G304 -- the caller explicitly selected the verification config.
	file, err := os.Open(path)
	if err != nil {
		return VerificationOptions{}, configError("read failed")
	}
	defer func() { _ = file.Close() }()
	raw, err := io.ReadAll(io.LimitReader(file, VerificationConfigMaxBytes+1))
	if err != nil || int64(len(raw)) > VerificationConfigMaxBytes {
		return VerificationOptions{}, configError("read failed")
	}
	return ParseVerificationConfig(raw)
}

// ParseVerificationConfig parses the strict JSON representation used by the
// CLI and integrations. It rejects duplicate and unknown fields recursively.
func ParseVerificationConfig(raw []byte) (VerificationOptions, error) {
	if err := rejectDuplicateKeys(raw); err != nil {
		return VerificationOptions{}, configError("duplicate or malformed JSON")
	}
	if err := gaitschema.ValidateLifecycleVerificationConfig(raw); err != nil {
		return VerificationOptions{}, configError("schema validation failed" + gaitschema.SafeValidationDetail(err))
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var cfg verificationConfig
	if err := decoder.Decode(&cfg); err != nil {
		return VerificationOptions{}, configError("unknown or malformed field")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return VerificationOptions{}, configError("trailing JSON")
	}
	keyRaw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(cfg.TrustedPublicKey))
	if err != nil || len(keyRaw) != ed25519.PublicKeySize {
		return VerificationOptions{}, configError("trusted_public_key must be standard base64 Ed25519 public key")
	}
	key := ed25519.PublicKey(keyRaw)
	if proofKeyID := proofsign.KeyID(key); proofKeyID == "" {
		return VerificationOptions{}, configError("trusted public key is invalid")
	} else if proofKeyID == FixtureKeyID && !cfg.AllowFixtureOnly {
		return VerificationOptions{}, configError("fixture key requires allow_fixture_only=true")
	}
	if !validRefKind(cfg.ExpectedContract, "action_contract", ContractSchemaID, ContractSchemaVersion) || cfg.ExpectedContract.SourceProduct != WrkrProducer || cfg.ExpectedFamily == "" || cfg.ExpectedRevision < 1 || !validRefKind(cfg.ExpectedActivation, "activated_action_contract", ActivationSchemaID, EvidenceSchemaVersion) || cfg.ExpectedActivation.SourceProduct != GaitProducer || !validDigest(cfg.ExpectedRuntimeDigest) || !validDigest(cfg.ExpectedReadinessDigest) || !validDigest(cfg.ExpectedPolicyDigest) || !validDigest(cfg.ExpectedLifecycleDigest) || strings.TrimSpace(cfg.ExpectedTarget) == "" || strings.TrimSpace(cfg.ExpectedEnvironment) == "" || strings.TrimSpace(cfg.ExpectedProducerVersion) == "" || !validCommit(cfg.ExpectedSourceCommit) {
		return VerificationOptions{}, configError("complete exact lineage bindings are required")
	}
	if proofsign.KeyID(key) == FixtureKeyID && (strings.TrimSpace(cfg.ExpectedProducerVersion) != FixtureTag || strings.TrimSpace(cfg.ExpectedSourceCommit) != FixtureCommit || !releasedFixtureDigest(cfg.ExpectedLifecycleDigest)) {
		return VerificationOptions{}, configError("fixture key requires exact released Gait v1.5.0 provenance")
	}
	evaluation, err := parseConfigTime(cfg.EvaluationTime)
	if err != nil {
		return VerificationOptions{}, configError("evaluation_time must be RFC3339")
	}
	notBefore, err := parseConfigTime(cfg.ActivationNotBefore)
	if err != nil {
		return VerificationOptions{}, configError("activation_not_before must be RFC3339")
	}
	notAfter, err := parseConfigTime(cfg.ActivationNotAfter)
	if err != nil || !notAfter.After(notBefore) {
		return VerificationOptions{}, configError("activation_not_after must be RFC3339 after not_before")
	}
	return VerificationOptions{
		TrustedPublicKey: key, EvaluationTime: evaluation, AllowFixtureOnly: cfg.AllowFixtureOnly,
		ExpectedContract: cfg.ExpectedContract, ExpectedFamily: cfg.ExpectedFamily, ExpectedRevision: cfg.ExpectedRevision,
		ExpectedActivation: cfg.ExpectedActivation, ExpectedRuntimeDigest: cfg.ExpectedRuntimeDigest,
		ExpectedReadinessDigest: cfg.ExpectedReadinessDigest, ExpectedPolicyDigest: cfg.ExpectedPolicyDigest,
		ExpectedTarget: cfg.ExpectedTarget, ExpectedEnvironment: cfg.ExpectedEnvironment,
		ExpectedProducerVersion: cfg.ExpectedProducerVersion, ExpectedSourceCommit: cfg.ExpectedSourceCommit, ExpectedLifecycleDigest: cfg.ExpectedLifecycleDigest,
		ActivationNotBefore: notBefore, ActivationNotAfter: notAfter,
	}, nil
}

func parseConfigTime(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, errors.New("time required")
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || parsed.IsZero() {
		return time.Time{}, errors.New("invalid time")
	}
	return parsed.UTC(), nil
}

func configError(message string) error { return fmt.Errorf("%s: %s", ReasonConfigInvalid, message) }
