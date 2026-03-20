package bundle

import (
	"bytes"
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
	"github.com/Clyra-AI/axym/core/export/manifest"
	"github.com/Clyra-AI/axym/core/export/oscal"
	"github.com/Clyra-AI/axym/core/export/safety"
	"github.com/Clyra-AI/axym/core/gaps"
	"github.com/Clyra-AI/axym/core/identitygovernance"
	"github.com/Clyra-AI/axym/core/review/grade"
	"github.com/Clyra-AI/axym/core/store"
	coreverify "github.com/Clyra-AI/axym/core/verify"
	verifysupport "github.com/Clyra-AI/axym/core/verifysupport"
	bundleschema "github.com/Clyra-AI/axym/schemas/v1/bundle"
	"github.com/Clyra-AI/proof"
	"gopkg.in/yaml.v3"
)

const (
	DefaultStoreDir       = ".axym"
	DefaultOutputDir      = "axym-evidence"
	FixedTimestampRFC3339 = "2000-01-01T00:00:00Z"
	SummaryVersion        = "v1"

	ReasonBundleBuild  = "bundle_build_failed"
	ReasonInvalidInput = "invalid_input"
	ReasonUnsafePath   = safety.ReasonUnsafePath
)

type BuildRequest struct {
	AuditName    string
	FrameworkIDs []string
	StoreDir     string
	OutputDir    string
}

type Result struct {
	Path         string       `json:"path"`
	Files        int          `json:"files"`
	Algo         string       `json:"algo"`
	Frameworks   []string     `json:"frameworks"`
	AuditName    string       `json:"audit"`
	Verification verifyResult `json:"verification"`
	Compliance   Compliance   `json:"compliance"`
}

type verifyResult struct {
	Intact   bool   `json:"intact"`
	Count    int    `json:"count"`
	HeadHash string `json:"head_hash,omitempty"`
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

func Build(req BuildRequest) (Result, error) {
	req = normalizeRequest(req)
	if strings.TrimSpace(req.AuditName) == "" {
		return Result{}, &Error{ReasonCode: ReasonInvalidInput, Message: "audit name is required", ExitCode: 6}
	}
	if err := safety.EnsureManagedOutputDir(req.OutputDir); err != nil {
		return Result{}, wrapSafetyError(err)
	}

	st, err := store.New(store.Config{RootDir: req.StoreDir, ComplianceMode: true})
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "initialize local store", ExitCode: 1, Err: err}
	}
	signingKey, err := verifysupport.LoadStoreSigningKey(req.StoreDir)
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "load signing key", ExitCode: 1, Err: err}
	}
	chain, err := st.LoadChain()
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "load chain", ExitCode: 1, Err: err}
	}
	chainSnapshot, err := cloneChain(chain)
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "clone chain", ExitCode: 1, Err: err}
	}
	recordSnapshot := append([]proof.Record(nil), chainSnapshot.Records...)
	chainVerification, err := coreverify.VerifyChainFromStoreDir(req.StoreDir)
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "verify chain prior to bundling", ExitCode: 2, Err: err}
	}

	definitions, err := framework.LoadMany(req.FrameworkIDs)
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonInvalidInput, Message: "load frameworks", ExitCode: 6, Err: err}
	}
	matchResult := match.Evaluate(definitions, append([]proof.Record(nil), recordSnapshot...), match.Options{ExcludeInvalidEvidence: true})
	coverageReport := coverage.Build(matchResult)
	gapReport := gaps.Build(coverageReport)
	compliance := buildCompliance(definitions, coverageReport, recordSnapshot, gapReport.Grade)
	identityArtifacts := identitygovernance.Build(recordSnapshot)

	artifacts := map[string][]byte{}
	rawChain, err := json.MarshalIndent(chainSnapshot, "", "  ")
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "marshal chain", ExitCode: 1, Err: err}
	}
	artifacts["chain.json"] = rawChain

	rawRecords, err := marshalRawRecords(recordSnapshot)
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "marshal raw records", ExitCode: 1, Err: err}
	}
	artifacts["raw-records.jsonl"] = rawRecords

	chainYAML, err := yaml.Marshal(struct {
		Intact   bool   `yaml:"intact"`
		Count    int    `yaml:"count"`
		HeadHash string `yaml:"head_hash,omitempty"`
	}{
		Intact:   chainVerification.Intact,
		Count:    chainVerification.Count,
		HeadHash: chainVerification.HeadHash,
	})
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "marshal chain verification artifact", ExitCode: 1, Err: err}
	}
	artifacts["chain-verification.yaml"] = chainYAML

	recordSigningKeyRaw, err := verifysupport.MarshalBundlePublicKey(proof.PublicKey{
		KeyID:  signingKey.KeyID,
		Public: signingKey.Public,
	})
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "marshal record signing key artifact", ExitCode: 1, Err: err}
	}
	if err := bundleschema.ValidateRecordSigningKey(recordSigningKeyRaw); err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "validate record signing key schema", ExitCode: 3, Err: err}
	}
	artifacts[verifysupport.BundlePublicKeyArtifact] = recordSigningKeyRaw

	gradeYAML, err := yaml.Marshal(struct {
		Version        string       `yaml:"version"`
		Audit          string       `yaml:"audit"`
		Frameworks     []string     `yaml:"frameworks"`
		FixedTimestamp string       `yaml:"fixed_timestamp"`
		Grade          grade.Result `yaml:"grade"`
		Summary        gaps.Summary `yaml:"summary"`
		Compliance     Compliance   `yaml:"compliance"`
	}{
		Version:        SummaryVersion,
		Audit:          req.AuditName,
		Frameworks:     append([]string(nil), req.FrameworkIDs...),
		FixedTimestamp: FixedTimestampRFC3339,
		Grade:          gapReport.Grade,
		Summary:        gapReport.Summary,
		Compliance:     compliance,
	})
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "marshal grade artifact", ExitCode: 1, Err: err}
	}
	artifacts["auditability-grade.yaml"] = gradeYAML

	identityChainRaw, err := identitygovernance.MarshalIndent(identityArtifacts.ChainSummary)
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "marshal identity-chain summary", ExitCode: 1, Err: err}
	}
	artifacts["identity-chain-summary.json"] = identityChainRaw

	ownershipRaw, err := identitygovernance.MarshalIndent(identityArtifacts.OwnershipRegister)
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "marshal ownership register", ExitCode: 1, Err: err}
	}
	artifacts["ownership-register.json"] = ownershipRaw

	privilegeRaw, err := identitygovernance.MarshalIndent(identityArtifacts.PrivilegeDriftReport)
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "marshal privilege drift report", ExitCode: 1, Err: err}
	}
	artifacts["privilege-drift-report.json"] = privilegeRaw

	delegatedRaw, err := identitygovernance.MarshalIndent(identityArtifacts.DelegatedChainExceptions)
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "marshal delegated-chain exceptions", ExitCode: 1, Err: err}
	}
	artifacts["delegated-chain-exceptions.json"] = delegatedRaw

	executiveSummary := struct {
		Version        string     `json:"version"`
		Audit          string     `json:"audit"`
		Frameworks     []string   `json:"frameworks"`
		RecordCount    int        `json:"record_count"`
		FixedTimestamp string     `json:"fixed_timestamp"`
		Compliance     Compliance `json:"compliance"`
	}{
		Version:        SummaryVersion,
		Audit:          req.AuditName,
		Frameworks:     append([]string(nil), req.FrameworkIDs...),
		RecordCount:    len(recordSnapshot),
		FixedTimestamp: FixedTimestampRFC3339,
		Compliance:     compliance,
	}
	execRaw, err := json.MarshalIndent(executiveSummary, "", "  ")
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "marshal executive summary", ExitCode: 1, Err: err}
	}
	if err := bundleschema.ValidateExecutiveSummary(execRaw); err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "validate executive summary schema", ExitCode: 3, Err: err}
	}
	artifacts["executive-summary.json"] = execRaw
	artifacts["executive-summary.pdf"] = buildExecutiveSummaryPDF(req.AuditName, req.FrameworkIDs, len(recordSnapshot), coverageReport, gapReport)

	retentionRaw, err := json.MarshalIndent(defaultRetentionMatrix(), "", "  ")
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "marshal retention matrix", ExitCode: 1, Err: err}
	}
	artifacts["retention-matrix.json"] = retentionRaw

	artifacts["boundary-contract.md"] = []byte(buildBoundaryContract(req.AuditName, req.FrameworkIDs))
	if overrideArtifact, err := loadOptionalStoreArtifact(req.StoreDir, filepath.Join("overrides", "overrides.jsonl")); err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "load override artifact", ExitCode: 1, Err: err}
	} else if len(overrideArtifact) > 0 {
		artifacts[filepath.ToSlash(filepath.Join("overrides", "overrides.jsonl"))] = overrideArtifact
	}

	oscalDoc := oscal.Build(req.AuditName, coverageReport)
	oscalRaw, err := oscal.Marshal(oscalDoc)
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "generate oscal export", ExitCode: 3, Err: err}
	}
	artifacts[filepath.ToSlash(filepath.Join("oscal-v1.1", "component-definition.json"))] = oscalRaw

	paths := make([]string, 0, len(artifacts))
	for rel, payload := range artifacts {
		paths = append(paths, rel)
		if err := writeArtifact(req.OutputDir, rel, payload); err != nil {
			return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: fmt.Sprintf("write artifact %s", rel), ExitCode: 1, Err: err}
		}
	}
	sort.Strings(paths)

	bundleManifest, err := manifest.Build(req.OutputDir, paths)
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "build manifest", ExitCode: 1, Err: err}
	}
	manifestRaw, err := json.MarshalIndent(bundleManifest, "", "  ")
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "marshal manifest", ExitCode: 1, Err: err}
	}
	if err := writeArtifact(req.OutputDir, "manifest.json", manifestRaw); err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "write manifest", ExitCode: 1, Err: err}
	}

	signedManifest, err := proof.SignBundleFile(req.OutputDir, signingKey)
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "sign bundle", ExitCode: 2, Err: err}
	}
	if _, err := proof.VerifyBundle(req.OutputDir, proof.BundleVerifyOpts{
		VerifySignatures: true,
		PublicKey: proof.PublicKey{
			KeyID:  signingKey.KeyID,
			Public: signingKey.Public,
		},
	}); err != nil {
		return Result{}, &Error{ReasonCode: ReasonBundleBuild, Message: "verify signed bundle", ExitCode: 2, Err: err}
	}

	return Result{
		Path:       req.OutputDir,
		Files:      len(signedManifest.Files),
		Algo:       signedManifest.AlgoID,
		Frameworks: append([]string(nil), req.FrameworkIDs...),
		AuditName:  req.AuditName,
		Verification: verifyResult{
			Intact:   chainVerification.Intact,
			Count:    chainVerification.Count,
			HeadHash: chainVerification.HeadHash,
		},
		Compliance: compliance,
	}, nil
}

func normalizeRequest(req BuildRequest) BuildRequest {
	if req.StoreDir == "" {
		req.StoreDir = DefaultStoreDir
	}
	if req.OutputDir == "" {
		req.OutputDir = DefaultOutputDir
	}
	req.FrameworkIDs = normalizeFrameworkIDs(req.FrameworkIDs)
	return req
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
	if len(out) == 0 {
		out = append(out, "eu-ai-act", "soc2")
	}
	sort.Strings(out)
	return out
}

func wrapSafetyError(err error) error {
	var sErr *safety.Error
	if !errors.As(err, &sErr) {
		return &Error{ReasonCode: ReasonUnsafePath, Message: "unsafe output path", ExitCode: 8, Err: err}
	}
	return &Error{
		ReasonCode: sErr.ReasonCode,
		Message:    sErr.Message,
		ExitCode:   sErr.ExitCode,
		Err:        sErr.Err,
	}
}

func buildCompliance(definitions []framework.Definition, report coverage.Report, records []proof.Record, gradeResult grade.Result) Compliance {
	required := map[string]struct{}{}
	for _, def := range definitions {
		for _, control := range def.Controls {
			for _, recordType := range control.RequiredRecordTypes {
				required[strings.TrimSpace(recordType)] = struct{}{}
			}
		}
	}
	observed := map[string]struct{}{}
	for _, record := range records {
		observed[strings.TrimSpace(record.RecordType)] = struct{}{}
	}

	requiredTypes := setToSortedSlice(required)
	observedTypes := setToSortedSlice(observed)

	missing := make([]string, 0)
	for _, want := range requiredTypes {
		if _, ok := observed[want]; ok {
			continue
		}
		missing = append(missing, want)
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
		MissingRecordTypes:  missing,
		IncompleteControls:  incomplete,
		ControlsMissing:     missingFields,
		Complete:            len(missing) == 0 && incomplete == 0 && missingFields == 0,
		Grade:               gradeResult,
		IdentityGovernance:  identityArtifacts.Digest,
	}
}

func setToSortedSlice(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}

func marshalRawRecords(records []proof.Record) ([]byte, error) {
	var out strings.Builder
	for _, record := range records {
		raw, err := json.Marshal(record)
		if err != nil {
			return nil, err
		}
		out.Write(raw)
		out.WriteString("\n")
	}
	return []byte(out.String()), nil
}

func writeArtifact(root string, rel string, payload []byte) error {
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return store.WriteJSONAtomic(path, payload, true)
}

func loadOptionalStoreArtifact(storeDir string, rel string) ([]byte, error) {
	path := filepath.Join(storeDir, filepath.FromSlash(rel))
	// #nosec G304 -- optional artifact path is derived from the managed local store root.
	payload, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return payload, nil
}

func buildBoundaryContract(audit string, frameworks []string) string {
	lines := []string{
		"# Axym Boundary Contract",
		"",
		"- Version: v1",
		"- Audit: " + audit,
		"- Frameworks: " + strings.Join(frameworks, ","),
		"- Fixed timestamp strategy: " + FixedTimestampRFC3339,
		"- Proof owns cryptographic verification; Axym layers deterministic compliance interpretation.",
		"- Axym proves identity-governed action in software delivery; it does not replace IAM, PAM, or IGA systems.",
		"- Upstream identity systems remain authoritative for identity lifecycle, credential issuance, entitlements, and interactive access control.",
		"- Bundle identity artifacts summarize actor, downstream identity, owner or approver, delegation chain, policy binding, and privilege-drift exceptions.",
		"- Output safety contract: non-empty unmanaged output paths fail with exit 8.",
		"- Raw records are included as `raw-records.jsonl` and chain material as `chain.json`.",
	}
	return strings.Join(lines, "\n") + "\n"
}

func defaultRetentionMatrix() map[string]any {
	return map[string]any{
		"version": "v1",
		"entries": []map[string]string{
			{"artifact": "chain.json", "retention": "7y", "rationale": "chain integrity audit trail"},
			{"artifact": "raw-records.jsonl", "retention": "7y", "rationale": "portable evidence replay"},
			{"artifact": "overrides/overrides.jsonl", "retention": "7y", "rationale": "signed manual approvals and exceptions"},
			{"artifact": "executive-summary.json", "retention": "3y", "rationale": "audit handoff summary"},
			{"artifact": "executive-summary.pdf", "retention": "3y", "rationale": "board-ready summary"},
			{"artifact": "oscal-v1.1/component-definition.json", "retention": "3y", "rationale": "framework interchange"},
		},
	}
}

func cloneChain(in *proof.Chain) (*proof.Chain, error) {
	raw, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var out proof.Chain
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func buildExecutiveSummaryPDF(audit string, frameworks []string, recordCount int, report coverage.Report, gapReport gaps.Report) []byte {
	lines := []string{
		fmt.Sprintf("Axym Executive Summary (%s)", audit),
		fmt.Sprintf("Frameworks: %s", strings.Join(frameworks, ", ")),
		fmt.Sprintf("AI systems in scope: %d", max(1, len(report.Frameworks))),
		fmt.Sprintf("Proof records collected: %d", recordCount),
	}
	for _, fw := range report.Frameworks {
		lines = append(lines, fmt.Sprintf("Coverage %s: %.2f%%", fw.FrameworkID, fw.Coverage*100))
	}
	limit := 3
	if len(gapReport.Gaps) < limit {
		limit = len(gapReport.Gaps)
	}
	for i := 0; i < limit; i++ {
		item := gapReport.Gaps[i]
		lines = append(lines, fmt.Sprintf("Top gap %d: %s/%s - %s (%s)", i+1, item.FrameworkID, item.ControlID, item.Remediation, item.Effort))
	}

	text := "BT\n/F1 12 Tf\n50 760 Td\n"
	for i, line := range lines {
		if i > 0 {
			text += "0 -16 Td\n"
		}
		text += fmt.Sprintf("(%s) Tj\n", pdfEscape(line))
	}
	text += "ET\n"

	var out bytes.Buffer
	write := func(s string) { _, _ = out.WriteString(s) }
	offsets := make([]int, 6)
	write("%PDF-1.4\n")
	offsets[1] = out.Len()
	write("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	offsets[2] = out.Len()
	write("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	offsets[3] = out.Len()
	write("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n")
	offsets[4] = out.Len()
	write(fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n%sendstream\nendobj\n", len(text), text))
	offsets[5] = out.Len()
	write("5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")
	xrefOffset := out.Len()
	write("xref\n0 6\n")
	write("0000000000 65535 f \n")
	for i := 1; i <= 5; i++ {
		write(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	write("trailer\n<< /Size 6 /Root 1 0 R >>\n")
	write(fmt.Sprintf("startxref\n%d\n%%%%EOF\n", xrefOffset))
	return out.Bytes()
}

func pdfEscape(input string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)")
	return replacer.Replace(input)
}
