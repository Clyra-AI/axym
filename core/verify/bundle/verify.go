package bundle

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Clyra-AI/axym/core/compliance/coverage"
	"github.com/Clyra-AI/axym/core/compliance/framework"
	"github.com/Clyra-AI/axym/core/compliance/match"
	"github.com/Clyra-AI/axym/core/identitygovernance"
	"github.com/Clyra-AI/axym/core/review/grade"
	bundleschema "github.com/Clyra-AI/axym/schemas/v1/bundle"
	"github.com/Clyra-AI/proof"
	"gopkg.in/yaml.v3"
)

const (
	ReasonBundleVerify       = "bundle_verify_failed"
	ReasonBundleCompleteness = "bundle_completeness_failed"
	ReasonInvalidInput       = "invalid_input"
	ReasonSchemaViolation    = "schema_violation"
)

type Error struct {
	ReasonCode string
	Message    string
	ExitCode   int
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

type Result struct {
	Path               string     `json:"path"`
	Files              int        `json:"files"`
	Algo               string     `json:"algo"`
	Cryptographic      bool       `json:"cryptographic"`
	ComplianceVerified bool       `json:"compliance_verified"`
	OSCALValid         bool       `json:"oscal_valid"`
	Compliance         Compliance `json:"compliance,omitempty"`
}

type Compliance struct {
	RequiredRecordTypes []string                  `json:"required_record_types"`
	ObservedRecordTypes []string                  `json:"observed_record_types"`
	MissingRecordTypes  []string                  `json:"missing_record_types"`
	IncompleteControls  int                       `json:"incomplete_controls"`
	ControlsMissing     int                       `json:"controls_missing_fields"`
	Complete            bool                      `json:"complete"`
	Grade               grade.Result              `json:"grade"`
	IdentityGovernance  identitygovernance.Digest `json:"identity_governance"`
}

type executiveSummary struct {
	Version    string     `json:"version"`
	Audit      string     `json:"audit"`
	Frameworks []string   `json:"frameworks"`
	Compliance Compliance `json:"compliance"`
}

func Verify(path string, frameworkIDs []string) (Result, error) {
	manifest, err := proof.VerifyBundle(path, proof.BundleVerifyOpts{})
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleVerify, Message: "bundle verification failed", ExitCode: 2, Err: err}
	}
	result := Result{
		Path:          path,
		Files:         len(manifest.Files),
		Algo:          manifest.AlgoID,
		Cryptographic: true,
	}

	summaryPath := filepath.Join(path, "executive-summary.json")
	// #nosec G304 -- bundle verification intentionally reads artifacts from the explicit bundle root.
	summaryRaw, err := os.ReadFile(summaryPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Legacy proof-only bundles are still cryptographically valid.
			return result, nil
		}
		return Result{}, &Error{ReasonCode: ReasonInvalidInput, Message: "read executive summary", ExitCode: 6, Err: err}
	}
	if err := bundleschema.ValidateExecutiveSummary(summaryRaw); err != nil {
		return Result{}, &Error{ReasonCode: ReasonSchemaViolation, Message: "executive summary schema validation failed", ExitCode: 3, Err: err}
	}
	var summary executiveSummary
	if err := json.Unmarshal(summaryRaw, &summary); err != nil {
		return Result{}, &Error{ReasonCode: ReasonInvalidInput, Message: "decode executive summary", ExitCode: 6, Err: err}
	}

	normalizedFrameworks := normalizeFrameworkIDs(frameworkIDs)
	if len(normalizedFrameworks) == 0 {
		normalizedFrameworks = normalizeFrameworkIDs(summary.Frameworks)
	}
	definitions, err := framework.LoadMany(normalizedFrameworks)
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonInvalidInput, Message: "load frameworks", ExitCode: 6, Err: err}
	}

	// #nosec G304 -- bundle verification intentionally reads artifacts from the explicit bundle root.
	chainRaw, err := os.ReadFile(filepath.Join(path, "chain.json"))
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonInvalidInput, Message: "read chain artifact", ExitCode: 6, Err: err}
	}
	var chain proof.Chain
	if err := json.Unmarshal(chainRaw, &chain); err != nil {
		return Result{}, &Error{ReasonCode: ReasonInvalidInput, Message: "decode chain artifact", ExitCode: 6, Err: err}
	}
	chainVerification, err := proof.VerifyChain(&chain)
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleVerify, Message: "verify chain artifact", ExitCode: 2, Err: err}
	}
	if !chainVerification.Intact {
		return Result{}, &Error{ReasonCode: ReasonBundleVerify, Message: "chain integrity check failed", ExitCode: 2}
	}

	matchResult := match.Evaluate(definitions, chain.Records, match.Options{ExcludeInvalidEvidence: true})
	coverageReport := coverage.Build(matchResult)
	recomputed := buildCompliance(definitions, coverageReport, chain.Records)
	if !equalCompliance(summary.Compliance, recomputed) {
		return Result{}, &Error{ReasonCode: ReasonBundleCompleteness, Message: "executive summary compliance does not match recomputed output", ExitCode: 2}
	}
	if err := verifyIdentityArtifacts(path, chain.Records); err != nil {
		return Result{}, err
	}

	// #nosec G304 -- bundle verification intentionally reads artifacts from the explicit bundle root.
	gradeArtifactRaw, err := os.ReadFile(filepath.Join(path, "auditability-grade.yaml"))
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonInvalidInput, Message: "read grade artifact", ExitCode: 6, Err: err}
	}
	var gradeArtifact struct {
		Grade grade.Result `yaml:"grade"`
	}
	if err := yaml.Unmarshal(gradeArtifactRaw, &gradeArtifact); err != nil {
		return Result{}, &Error{ReasonCode: ReasonInvalidInput, Message: "decode grade artifact", ExitCode: 6, Err: err}
	}
	if gradeArtifact.Grade != recomputed.Grade {
		return Result{}, &Error{ReasonCode: ReasonBundleCompleteness, Message: "grade artifact does not match recomputed grade", ExitCode: 2}
	}

	// #nosec G304 -- bundle verification intentionally reads artifacts from the explicit bundle root.
	oscalRaw, err := os.ReadFile(filepath.Join(path, "oscal-v1.1", "component-definition.json"))
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonInvalidInput, Message: "read oscal export", ExitCode: 6, Err: err}
	}
	if err := bundleschema.ValidateOSCAL(oscalRaw); err != nil {
		return Result{}, &Error{ReasonCode: ReasonSchemaViolation, Message: "oscal schema validation failed", ExitCode: 3, Err: err}
	}

	result.ComplianceVerified = true
	result.OSCALValid = true
	result.Compliance = recomputed
	return result, nil
}

func buildCompliance(definitions []framework.Definition, report coverage.Report, records []proof.Record) Compliance {
	required := map[string]struct{}{}
	for _, def := range definitions {
		for _, control := range def.Controls {
			for _, recordType := range control.RequiredRecordTypes {
				trimmed := strings.TrimSpace(recordType)
				if trimmed == "" {
					continue
				}
				required[trimmed] = struct{}{}
			}
		}
	}
	observed := map[string]struct{}{}
	for _, record := range records {
		trimmed := strings.TrimSpace(record.RecordType)
		if trimmed == "" {
			continue
		}
		observed[trimmed] = struct{}{}
	}

	requiredTypes := setToSortedSlice(required)
	observedTypes := setToSortedSlice(observed)
	missingTypes := make([]string, 0)
	for _, requiredType := range requiredTypes {
		if _, ok := observed[requiredType]; ok {
			continue
		}
		missingTypes = append(missingTypes, requiredType)
	}

	incomplete := 0
	missingFields := 0
	for _, fw := range report.Frameworks {
		for _, control := range fw.Controls {
			if control.Status != "covered" {
				incomplete++
			}
			if len(control.MissingFields) > 0 {
				missingFields++
			}
		}
	}
	identityArtifacts := identitygovernance.Build(records)

	return Compliance{
		RequiredRecordTypes: requiredTypes,
		ObservedRecordTypes: observedTypes,
		MissingRecordTypes:  missingTypes,
		IncompleteControls:  incomplete,
		ControlsMissing:     missingFields,
		Complete:            len(missingTypes) == 0 && incomplete == 0 && missingFields == 0,
		Grade:               grade.Derive(report),
		IdentityGovernance:  identityArtifacts.Digest,
	}
}

func verifyIdentityArtifacts(path string, records []proof.Record) error {
	artifacts := identitygovernance.Build(records)
	checks := []struct {
		rel  string
		want any
	}{
		{rel: "identity-chain-summary.json", want: artifacts.ChainSummary},
		{rel: "ownership-register.json", want: artifacts.OwnershipRegister},
		{rel: "privilege-drift-report.json", want: artifacts.PrivilegeDriftReport},
		{rel: "delegated-chain-exceptions.json", want: artifacts.DelegatedChainExceptions},
	}
	for _, check := range checks {
		// #nosec G304 -- bundle verification intentionally reads artifacts from the explicit bundle root.
		gotRaw, err := os.ReadFile(filepath.Join(path, check.rel))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return &Error{ReasonCode: ReasonBundleCompleteness, Message: "missing identity artifact " + check.rel, ExitCode: 2, Err: err}
			}
			return &Error{ReasonCode: ReasonInvalidInput, Message: "read identity artifact " + check.rel, ExitCode: 6, Err: err}
		}
		wantRaw, err := identitygovernance.MarshalIndent(check.want)
		if err != nil {
			return &Error{ReasonCode: ReasonBundleVerify, Message: "marshal recomputed identity artifact " + check.rel, ExitCode: 2, Err: err}
		}
		if string(gotRaw) != string(wantRaw) {
			return &Error{ReasonCode: ReasonBundleCompleteness, Message: "identity artifact does not match recomputed output: " + check.rel, ExitCode: 2}
		}
	}
	return nil
}

func setToSortedSlice(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func equalCompliance(left Compliance, right Compliance) bool {
	leftRaw, err := json.Marshal(left)
	if err != nil {
		return false
	}
	rightRaw, err := json.Marshal(right)
	if err != nil {
		return false
	}
	return string(leftRaw) == string(rightRaw)
}

func normalizeFrameworkIDs(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, candidate := range in {
		for _, part := range strings.Split(candidate, ",") {
			id := strings.ToLower(strings.TrimSpace(part))
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}
