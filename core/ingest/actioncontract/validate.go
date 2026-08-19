package actioncontract

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	proofcanon "github.com/Clyra-AI/proof/canon"
	proofsign "github.com/Clyra-AI/proof/signing"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v5"
)

const maxArtifactBytes int64 = 16 << 20

//go:embed schemaassets/*.json
var schemaAssets embed.FS

var (
	artifactIDPattern   = regexp.MustCompile(`^paca-[a-f0-9]{16}$`)
	activationIDPattern = regexp.MustCompile(`^gact-[a-f0-9]{16}$`)
	contractIDPattern   = regexp.MustCompile(`^pac-[a-f0-9]{8,64}$`)
	familyIDPattern     = regexp.MustCompile(`^pacf-[a-f0-9]{8,64}$`)
	digestPattern       = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
	compileOnce         sync.Once
	compiled            map[string]*jsonschema.Schema
	compileErr          error
)

const (
	ReasonMalformed             = "artifact_malformed"
	ReasonSchema                = "schema_validation_failed"
	ReasonUnsupportedSchema     = "schema_unsupported"
	ReasonUnsupportedProducer   = "producer_unsupported"
	ReasonReportOnly            = "report_only_required"
	ReasonIdentity              = "identity_mismatch"
	ReasonDigest                = "canonical_digest_mismatch"
	ReasonContractDigest        = "contract_digest_mismatch"
	ReasonRevision              = "revision_mismatch"
	ReasonBinding               = "proposal_binding_mismatch"
	ReasonSignature             = "signature_invalid"
	ReasonSignatureUnverifiable = "signature_unverifiable"
	ReasonValidity              = "validity_invalid"
	ReasonExpired               = "activation_expired"
	ReasonNotYetValid           = "activation_not_yet_valid"
	ReasonCurrentRevision       = "revision_not_current"
	ReasonSelectionRequired     = "selection_evidence_required"
	ReasonSelectionMismatch     = "selection_mismatch"
	ReasonSelectionNotCurrent   = "selection_not_current"
	ReasonSelectionAmbiguous    = "selection_ambiguous"
	ReasonEvaluationTime        = "evaluation_time_required"
	ReasonDevelopmentSigning    = "development_signing_not_allowed"
	ReasonDevelopmentUnverified = "development_signing_unverifiable"
	ReasonInputUnreadable       = "artifact_unreadable"
)

type ValidationError struct{ Reasons []string }

func (e *ValidationError) Error() string {
	if e == nil || len(e.Reasons) == 0 {
		return "action contract validation failed"
	}
	return "action contract validation failed: " + strings.Join(e.Reasons, ",")
}

// StableReasonCodes converts any ingestion error into the bounded public
// reason vocabulary. Dynamic OS/parser text is never emitted as a reason.
func StableReasonCodes(err error) []string {
	if err == nil {
		return nil
	}
	var validationErr *ValidationError
	if errors.As(err, &validationErr) && validationErr != nil && len(validationErr.Reasons) > 0 {
		return sortedUnique(validationErr.Reasons)
	}
	return []string{ReasonInputUnreadable}
}

func sortedUnique(values []string) []string {
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			seen[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

// SortedStrings provides deterministic de-duplication for receipt references.
func SortedStrings(values []string) []string { return sortedUnique(values) }

func addReason(reasons *[]string, reason string) {
	for _, existing := range *reasons {
		if existing == reason {
			return
		}
	}
	*reasons = append(*reasons, reason)
}

func rejectDuplicateJSONKeys(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var walk func() error
	walk = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]struct{}{}
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return err
				}
				key := keyToken.(string)
				if _, exists := seen[key]; exists {
					return fmt.Errorf("duplicate JSON key %q", key)
				}
				seen[key] = struct{}{}
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		default:
			return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
		}
	}
	if err := walk(); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return err
	}
	return nil
}

func decodeMap(raw []byte) (map[string]any, error) {
	if err := rejectDuplicateJSONKeys(raw); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var data map[string]any
	if err := decoder.Decode(&data); err != nil || data == nil {
		if err == nil {
			err = errors.New("object required")
		}
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing JSON value")
	}
	return data, nil
}

func compiledSchemas() (map[string]*jsonschema.Schema, error) {
	compileOnce.Do(func() {
		compiler := jsonschema.NewCompiler()
		compiled = map[string]*jsonschema.Schema{}
		resources := map[string]string{
			ProposalSchemaID: "proposed-action-contract-artifact.schema.json",
			"https://wrkr.dev/schemas/v1/proposed-action-contract-v3.schema.json": "proposed-action-contract-v3.schema.json",
			ActivationSchemaID: "activated-action-contract-artifact.schema.json",
			ReceiptSchemaID:    "consumer-receipt.schema.json",
		}
		for id, file := range resources {
			raw, err := schemaAssets.ReadFile("schemaassets/" + file)
			if err != nil {
				compileErr = err
				return
			}
			if err := compiler.AddResource(id, bytes.NewReader(raw)); err != nil {
				compileErr = err
				return
			}
		}
		for id := range resources {
			schema, err := compiler.Compile(id)
			if err != nil {
				compileErr = err
				return
			}
			compiled[id] = schema
		}
	})
	return compiled, compileErr
}

func validateSchema(raw []byte, schemaID string) error {
	sch, err := compiledSchemas()
	if err != nil {
		return err
	}
	data, err := decodeMap(raw)
	if err != nil {
		return err
	}
	return sch[schemaID].Validate(data)
}

func ParseProposal(raw []byte) (Proposal, error) {
	data, err := decodeMap(raw)
	if err != nil {
		return Proposal{}, &ValidationError{Reasons: []string{ReasonMalformed}}
	}
	if err := validateSchema(raw, ProposalSchemaID); err != nil {
		return Proposal{}, &ValidationError{Reasons: []string{ReasonSchema}}
	}
	var typed struct {
		SchemaID               string           `json:"schema_id"`
		SchemaVersion          string           `json:"schema_version"`
		ArtifactID             string           `json:"artifact_id"`
		ContractID             string           `json:"contract_id"`
		ContractFamilyID       string           `json:"contract_family_id"`
		Revision               int              `json:"revision"`
		Producer               ProducerMetadata `json:"producer"`
		SourceScanRefs         []string         `json:"source_scan_refs"`
		CompositionRefs        []string         `json:"composition_refs"`
		ResolutionKey          string           `json:"resolution_key"`
		CreationEvidence       []string         `json:"creation_evidence"`
		CanonicalContentDigest string           `json:"canonical_content_digest"`
		ReportOnly             bool             `json:"report_only"`
		Contract               map[string]any   `json:"contract"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&typed); err != nil {
		return Proposal{}, &ValidationError{Reasons: []string{ReasonMalformed}}
	}
	return Proposal{Raw: append([]byte(nil), raw...), Data: data, SchemaID: typed.SchemaID, SchemaVersion: typed.SchemaVersion, ArtifactID: typed.ArtifactID, ContractID: typed.ContractID, ContractFamilyID: typed.ContractFamilyID, Revision: typed.Revision, Producer: typed.Producer, SourceScanRefs: typed.SourceScanRefs, CompositionRefs: typed.CompositionRefs, ResolutionKey: typed.ResolutionKey, CreationEvidence: typed.CreationEvidence, CanonicalContentDigest: typed.CanonicalContentDigest, ReportOnly: typed.ReportOnly, Contract: typed.Contract, RawSHA256: rawDigest(raw)}, nil
}

func ReadProposal(path string) (Proposal, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return Proposal{}, &ValidationError{Reasons: []string{ReasonMalformed}}
	}
	raw, err := readStableArtifact(path)
	if err != nil {
		return Proposal{}, err
	}
	return ParseProposal(raw)
}

func ValidateProposal(proposal Proposal, options ValidationOptions) ValidationResult {
	result := ValidationResult{Status: StatusInvalid, RawSHA256: proposal.RawSHA256}
	var reasons []string
	add := func(reason string) { addReason(&reasons, reason) }
	if err := validateSchema(proposal.Raw, ProposalSchemaID); err != nil {
		add(ReasonSchema)
	}
	if proposal.SchemaID != ProposalSchemaID || proposal.SchemaVersion != ProposalSchemaVersion {
		add(ReasonUnsupportedSchema)
	}
	if proposal.Producer.Name != ProposalProducer || proposal.Producer.ArtifactSchemaVersion != ProposalSchemaVersion {
		add(ReasonUnsupportedProducer)
	}
	if proposal.Producer.ContractSchemaVersion != ProposalContractVersion {
		add(ReasonUnsupportedSchema)
	}
	if !proposal.ReportOnly {
		add(ReasonReportOnly)
	}
	if !artifactIDPattern.MatchString(proposal.ArtifactID) || !contractIDPattern.MatchString(proposal.ContractID) || !familyIDPattern.MatchString(proposal.ContractFamilyID) || proposal.Revision < 1 {
		add(ReasonIdentity)
	}
	if len(nonEmpty(proposal.SourceScanRefs)) == 0 || len(nonEmpty(proposal.CompositionRefs)) == 0 || len(nonEmpty(proposal.CreationEvidence)) == 0 {
		add(ReasonIdentity)
	}
	if proposal.Contract == nil || stringField(proposal.Contract, "contract_id") != proposal.ContractID || stringField(proposal.Contract, "contract_family_id") != proposal.ContractFamilyID || intField(proposal.Contract, "revision") != proposal.Revision || stringField(proposal.Contract, "contract_version") != ProposalContractVersion || stringField(proposal.Contract, "contract_kind") != "proposed_action_contract" || !boolField(proposal.Contract, "report_only") {
		add(ReasonIdentity)
	}
	if composition := stringField(proposal.Contract, "composition_ref"); composition == "" || !containsString(proposal.CompositionRefs, composition) {
		add(ReasonIdentity)
	}
	if resolution := strings.TrimSpace(proposal.ResolutionKey); resolution != "" && strings.TrimSpace(stringField(proposal.Contract, "resolution_key")) != "" && resolution != stringField(proposal.Contract, "resolution_key") {
		add(ReasonIdentity)
	}
	if proposal.Revision > 1 && strings.TrimSpace(stringField(proposal.Contract, "supersedes_ref")) == "" {
		add(ReasonRevision)
	}
	if digest, err := proposalCanonicalDigest(proposal.Data); err != nil || digest != proposal.CanonicalContentDigest {
		add(ReasonDigest)
	} else {
		result.CanonicalContentDigest = digest
	}
	embedded := stringField(proposal.Contract, "contract_content_digest")
	if !digestPattern.MatchString(embedded) || proposedContractContentDigest(proposal.Contract) != embedded {
		add(ReasonContractDigest)
	}
	if proposedContractFamilyID(proposal.Contract) != proposal.ContractFamilyID || proposedContractID(proposal.Contract) != proposal.ContractID {
		add(ReasonIdentity)
	}
	if options.ExpectedRevision > 0 && options.ExpectedRevision != proposal.Revision {
		add(ReasonRevision)
	}
	now := options.Now
	if expires := stringField(proposal.Contract, "expires_at"); expires != "" {
		if now.IsZero() {
			add(ReasonEvaluationTime)
		} else if parsed, err := parseUTCInstant(expires); err != nil || parsed.Before(now.UTC()) {
			add(ReasonValidity)
		}
	}
	result.ReasonCodes = sortedUnique(reasons)
	result.Valid = len(result.ReasonCodes) == 0
	if result.Valid {
		result.Status = StatusPass
	} else if hasReason(result.ReasonCodes, ReasonEvaluationTime) {
		result.Status = StatusUnverifiable
	}
	return result
}

func hasReason(reasons []string, wanted string) bool {
	for _, reason := range reasons {
		if reason == wanted {
			return true
		}
	}
	return false
}

func ParseActivation(raw []byte) (Activation, error) {
	data, err := decodeMap(raw)
	if err != nil {
		return Activation{}, &ValidationError{Reasons: []string{ReasonMalformed}}
	}
	if err := validateSchema(raw, ActivationSchemaID); err != nil {
		return Activation{}, &ValidationError{Reasons: []string{ReasonSchema}}
	}
	var typed Activation
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&typed); err != nil {
		return Activation{}, &ValidationError{Reasons: []string{ReasonMalformed}}
	}
	typed.Raw = append([]byte(nil), raw...)
	typed.Data = data
	typed.RawSHA256 = rawDigest(raw)
	return typed, nil
}

func ReadActivation(path string) (Activation, error) {
	raw, err := readStableArtifact(path)
	if err != nil {
		return Activation{}, err
	}
	return ParseActivation(raw)
}

func readStableArtifact(path string) ([]byte, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, &ValidationError{Reasons: []string{ReasonMalformed}}
	}
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return nil, &ValidationError{Reasons: []string{ReasonInputUnreadable}}
	}
	if err := rejectSymlinkAncestors(abs); err != nil {
		return nil, err
	}
	initial, err := os.Lstat(abs)
	if err != nil {
		return nil, &ValidationError{Reasons: []string{ReasonInputUnreadable}}
	}
	if initial.Mode()&os.ModeSymlink != 0 || !initial.Mode().IsRegular() {
		return nil, &ValidationError{Reasons: []string{ReasonMalformed}}
	}
	file, err := os.Open(abs) // #nosec G304 -- explicit consumer path after lexical safety checks.
	if err != nil {
		return nil, &ValidationError{Reasons: []string{ReasonInputUnreadable}}
	}
	defer func() { _ = file.Close() }()
	descriptorInfo, err := file.Stat()
	if err != nil || !descriptorInfo.Mode().IsRegular() || descriptorInfo.Size() > maxArtifactBytes {
		return nil, &ValidationError{Reasons: []string{ReasonMalformed}}
	}
	first, err := readBounded(file)
	if err != nil {
		return nil, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, &ValidationError{Reasons: []string{ReasonInputUnreadable}}
	}
	second, err := readBounded(file)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(first, second) {
		return nil, &ValidationError{Reasons: []string{ReasonInputUnreadable}}
	}
	finalInfo, err := file.Stat()
	if err != nil || finalInfo.Size() != descriptorInfo.Size() {
		return nil, &ValidationError{Reasons: []string{ReasonInputUnreadable}}
	}
	current, err := os.Lstat(abs)
	if err != nil || current.Mode()&os.ModeSymlink != 0 || !current.Mode().IsRegular() || !os.SameFile(descriptorInfo, current) {
		return nil, &ValidationError{Reasons: []string{ReasonInputUnreadable}}
	}
	return first, nil
}

func readBounded(file *os.File) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(file, maxArtifactBytes+1))
	if err != nil {
		return nil, &ValidationError{Reasons: []string{ReasonInputUnreadable}}
	}
	if int64(len(raw)) > maxArtifactBytes {
		return nil, &ValidationError{Reasons: []string{ReasonMalformed}}
	}
	return raw, nil
}

func rejectSymlinkAncestors(path string) error {
	current := filepath.Dir(path)
	for {
		info, err := os.Lstat(current)
		if err != nil {
			return &ValidationError{Reasons: []string{ReasonInputUnreadable}}
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return &ValidationError{Reasons: []string{ReasonMalformed}}
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}

func selectionMatchesProposal(selection SelectionEvidence, proposal Proposal) bool {
	return selection.CandidateCount <= 1 && selection.Current && selection.ArtifactID == proposal.ArtifactID && selection.ArtifactSHA256 == rawDigest(proposal.Raw) && selection.CanonicalContentDigest == proposal.CanonicalContentDigest && selection.ContractID == proposal.ContractID && selection.ContractFamilyID == proposal.ContractFamilyID && selection.Revision == proposal.Revision
}

func parseUTCInstant(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

// ParseSelectionManifest parses the Gait current-selection manifest shape and
// returns one exact selection assertion for proposal. It rejects ambiguous,
// stale, superseded, and digest-mismatched selections.
func ParseSelectionManifest(raw []byte, proposal Proposal) (SelectionEvidence, error) {
	if err := rejectDuplicateJSONKeys(raw); err != nil {
		return SelectionEvidence{}, &ValidationError{Reasons: []string{ReasonMalformed}}
	}
	var manifest struct {
		FixtureVersion string `json:"fixture_version"`
		Producer       struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"producer"`
		Schemas struct {
			Artifact string `json:"artifact"`
			Contract string `json:"contract"`
		} `json:"schemas"`
		Scenarios []struct {
			ArtifactID             string `json:"artifact_id"`
			ArtifactSHA256         string `json:"artifact_sha256"`
			CanonicalContentDigest string `json:"canonical_content_digest"`
			ContractID             string `json:"contract_id"`
			ContractFamilyID       string `json:"contract_family_id"`
			Revision               int    `json:"revision"`
			Current                bool   `json:"current"`
		} `json:"scenarios"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return SelectionEvidence{}, &ValidationError{Reasons: []string{ReasonMalformed}}
	}
	if manifest.FixtureVersion != "1" || manifest.Producer.Name != ProposalProducer || manifest.Schemas.Artifact != ProposalSchemaVersion || manifest.Schemas.Contract != ProposalContractVersion {
		return SelectionEvidence{}, &ValidationError{Reasons: []string{ReasonSelectionMismatch}}
	}
	var selected SelectionEvidence
	found := 0
	maxRevision := 0
	for _, scenario := range manifest.Scenarios {
		if scenario.ContractFamilyID == proposal.ContractFamilyID && scenario.Revision > maxRevision {
			maxRevision = scenario.Revision
		}
		if scenario.ArtifactID != proposal.ArtifactID && scenario.ContractID != proposal.ContractID {
			continue
		}
		found++
		selected = SelectionEvidence{
			ArtifactID: scenario.ArtifactID, ArtifactSHA256: scenario.ArtifactSHA256,
			CanonicalContentDigest: scenario.CanonicalContentDigest, ContractID: scenario.ContractID,
			ContractFamilyID: scenario.ContractFamilyID, Revision: scenario.Revision,
			Current: scenario.Current, CandidateCount: found,
		}
	}
	if found == 0 {
		return SelectionEvidence{}, &ValidationError{Reasons: []string{ReasonSelectionRequired}}
	}
	if found > 1 {
		return SelectionEvidence{}, &ValidationError{Reasons: []string{ReasonSelectionAmbiguous}}
	}
	if !selected.Current || maxRevision > selected.Revision {
		return SelectionEvidence{}, &ValidationError{Reasons: []string{ReasonSelectionNotCurrent}}
	}
	if !selectionMatchesProposal(selected, proposal) {
		return SelectionEvidence{}, &ValidationError{Reasons: []string{ReasonSelectionMismatch}}
	}
	return selected, nil
}

// ReadSelectionManifest reads one explicit, stable current-selection
// manifest and binds it to proposal without scanning for alternatives.
func ReadSelectionManifest(path string, proposal Proposal) (SelectionEvidence, error) {
	raw, err := readStableArtifact(path)
	if err != nil {
		return SelectionEvidence{}, err
	}
	return ParseSelectionManifest(raw, proposal)
}

func ValidateActivation(activation Activation, options ValidationOptions) ValidationResult {
	result := ValidationResult{Status: StatusUnverifiable, RawSHA256: activation.RawSHA256}
	var reasons []string
	add := func(reason string) { addReason(&reasons, reason) }
	if err := validateSchema(activation.Raw, ActivationSchemaID); err != nil {
		add(ReasonSchema)
	}
	if activation.SchemaID != ActivationSchemaID || activation.SchemaVersion != ActivationSchemaVersion || activation.Producer.Name != "gait" || activation.Producer.ArtifactSchemaVersion != ActivationSchemaVersion || activation.Producer.ContractSchemaVersion != ActivationContractVersion || activation.ReportOnly {
		add(ReasonUnsupportedSchema)
	}
	if activation.DevelopmentSigning {
		if !options.AllowDevelopmentSigning {
			add(ReasonDevelopmentSigning)
		} else {
			add(ReasonDevelopmentUnverified)
		}
	}
	if !activationIDPattern.MatchString(activation.ArtifactID) || !contractIDPattern.MatchString(activation.ContractID) || !familyIDPattern.MatchString(activation.ContractFamilyID) || activation.Revision < 1 || activation.PolicyDigest == "" || activation.ActivatingPrincipal == "" || len(nonEmpty(activation.AuthorityRefs)) == 0 || activation.Target == "" || activation.Environment == "" {
		add(ReasonIdentity)
	}
	if activation.Proposal.ContractID != activation.ContractID || activation.Proposal.ContractFamilyID != activation.ContractFamilyID || activation.Proposal.Revision != activation.Revision || activation.Proposal.SchemaID != ProposalSchemaID || activation.Proposal.SchemaVersion != ProposalSchemaVersion || activation.Proposal.ContractSchemaVersion != ProposalContractVersion {
		add(ReasonBinding)
	}
	if options.Selection == nil {
		add(ReasonSelectionRequired)
	} else if options.Selection.CandidateCount > 1 {
		add(ReasonSelectionAmbiguous)
	} else if !options.Selection.Current {
		add(ReasonSelectionNotCurrent)
	} else if options.Proposal == nil || !selectionMatchesProposal(*options.Selection, *options.Proposal) {
		add(ReasonSelectionMismatch)
	}
	if options.Proposal == nil {
		add(ReasonSignatureUnverifiable)
	} else {
		proposalResult := ValidateProposal(*options.Proposal, options)
		if !proposalResult.Valid || activation.Proposal.ArtifactID != options.Proposal.ArtifactID || activation.Proposal.CanonicalContentDigest != options.Proposal.CanonicalContentDigest {
			add(ReasonBinding)
		}
	}
	if activation.ActivationMode != "context_only" && activation.ActivationMode != "enforce_floor" && activation.ActivationMode != "required" {
		add(ReasonIdentity)
	}
	if options.Now.IsZero() {
		add(ReasonEvaluationTime)
	}
	if parsed, err := parseUTCInstant(activation.Validity.NotBefore); err != nil {
		add(ReasonValidity)
	} else if !options.Now.IsZero() && options.Now.UTC().Before(parsed) {
		add(ReasonNotYetValid)
	}
	if activation.Validity.NotAfter != "" {
		if parsed, err := parseUTCInstant(activation.Validity.NotAfter); err != nil {
			add(ReasonValidity)
		} else if !options.Now.IsZero() && !options.Now.UTC().Before(parsed) {
			add(ReasonExpired)
		}
		if before, beforeErr := parseUTCInstant(activation.Validity.NotBefore); beforeErr == nil {
			if after, afterErr := parseUTCInstant(activation.Validity.NotAfter); afterErr == nil && !after.After(before) {
				add(ReasonValidity)
			}
		}
	}
	if options.ExpectedRevision > 0 && options.ExpectedRevision != activation.Revision {
		add(ReasonCurrentRevision)
	}
	if len(options.PublicKey) == ed25519.PublicKeySize && len(reasons) == 0 {
		digest, err := activationSignableDigest(activation)
		if err != nil {
			add(ReasonSignature)
		} else if activation.Signature.SignedDigest != strings.TrimPrefix(digest, "sha256:") {
			add(ReasonSignature)
		} else if valid, err := proofsign.VerifyDigestHex(options.PublicKey, activation.Signature); err != nil || !valid {
			add(ReasonSignature)
		}
		if len(reasons) == 0 {
			wantID := "gact-" + strings.TrimPrefix(digest, "sha256:")[:16]
			if activation.ArtifactID != wantID {
				add(ReasonIdentity)
			}
		}
	} else if len(reasons) == 0 {
		add(ReasonSignatureUnverifiable)
	}
	result.ReasonCodes = sortedUnique(reasons)
	result.Valid = len(result.ReasonCodes) == 0 && !activation.DevelopmentSigning
	if result.Valid {
		result.Status = StatusPass
	}
	return result
}

func activationSignableDigest(activation Activation) (string, error) {
	data := map[string]any{}
	for key, value := range activation.Data {
		if key != "signature" {
			data[key] = value
		}
	}
	data["artifact_id"] = ""
	data["signature"] = map[string]any{"alg": "", "key_id": "", "sig": ""}
	encoded, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return proofcanon.DigestJCS(encoded)
}

func signActivationForTest(data map[string]any, privateKey ed25519.PrivateKey) (map[string]any, error) {
	copyData := map[string]any{}
	for key, value := range data {
		copyData[key] = value
	}
	copyData["artifact_id"] = ""
	copyData["signature"] = map[string]any{"alg": "", "key_id": "", "sig": ""}
	encoded, err := json.Marshal(copyData)
	if err != nil {
		return nil, err
	}
	digest, err := proofcanon.DigestJCS(encoded)
	if err != nil {
		return nil, err
	}
	sig, err := proofsign.SignDigestHex(privateKey, strings.TrimPrefix(digest, "sha256:"))
	if err != nil {
		return nil, err
	}
	copyData["signature"] = sig
	copyData["artifact_id"] = "gact-" + strings.TrimPrefix(digest, "sha256:")[:16]
	return copyData, nil
}

func rawDigest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// RawDigest returns the byte SHA-256 used by consumer receipts.
func RawDigest(raw []byte) string { return rawDigest(raw) }

func DecodeSignature(data map[string]any) (Signature, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return Signature{}, err
	}
	var signature Signature
	if err := json.Unmarshal(raw, &signature); err != nil {
		return Signature{}, err
	}
	if signature.Alg != proofsign.AlgEd25519 {
		return Signature{}, fmt.Errorf("unsupported signature algorithm %q", signature.Alg)
	}
	if _, err := base64.StdEncoding.DecodeString(signature.Sig); err != nil {
		return Signature{}, err
	}
	return signature, nil
}

func nonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

func containsString(values []string, wanted string) bool {
	wanted = strings.TrimSpace(wanted)
	for _, value := range values {
		if strings.TrimSpace(value) == wanted {
			return true
		}
	}
	return false
}

func stringField(object map[string]any, key string) string {
	value, _ := object[key].(string)
	return strings.TrimSpace(value)
}

func boolField(object map[string]any, key string) bool {
	value, _ := object[key].(bool)
	return value
}

func intField(object map[string]any, key string) int {
	switch value := object[key].(type) {
	case int:
		return value
	case float64:
		return int(value)
	case json.Number:
		parsed, _ := value.Int64()
		return int(parsed)
	default:
		return 0
	}
}

func stringArray(object map[string]any, key string) []string {
	items, _ := object[key].([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if value, ok := item.(string); ok {
			out = append(out, strings.TrimSpace(value))
		}
	}
	return out
}

func objectArray(object map[string]any, key string) []map[string]any {
	items, _ := object[key].([]any)
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if value, ok := item.(map[string]any); ok {
			out = append(out, value)
		}
	}
	return out
}

func proposedContractFamilyID(contract map[string]any) string {
	constraints := make([]string, 0)
	for _, item := range objectArray(contract, "target_constraints") {
		constraints = append(constraints, stringField(item, "key")+"="+stringField(item, "value"))
	}
	raw := strings.Join([]string{stringField(contract, "composition_ref"), stringField(contract, "required_credential_mode") + "|" + strconv.Itoa(intField(contract, "maximum_delegation_depth")) + "|" + stringField(contract, "expected_outcome_class") + "|" + stringField(contract, "resolution_key"), strings.Join(constraints, "|")}, "\x1f")
	return "pacf-" + shortHash(raw)
}

func proposedContractID(contract map[string]any) string {
	digest := proposedContractContentDigest(contract)
	if digest == "" {
		return ""
	}
	return "pac-" + shortHash(proposedContractFamilyID(contract)+"|"+digest+"|"+stringField(contract, "contract_version"))
}

func proposedContractContentDigest(contract map[string]any) string {
	parts := []string{"version=" + stringField(contract, "contract_version"), "kind=" + stringField(contract, "contract_kind"), "composition=" + stringField(contract, "composition_ref"), "resolution=" + stringField(contract, "resolution_key"), "credential_mode=" + stringField(contract, "required_credential_mode"), "delegation_depth=" + strconv.Itoa(intField(contract, "maximum_delegation_depth")), "outcome=" + stringField(contract, "expected_outcome_class"), "compensation=" + strconv.FormatBool(boolField(contract, "compensation_required")), "expires_at=" + stringField(contract, "expires_at"), "report_only=" + strconv.FormatBool(boolField(contract, "report_only")), "readiness=" + stringField(contract, "readiness_state"), "revision=" + strconv.Itoa(intField(contract, "revision")), "supersedes=" + stringField(contract, "supersedes_ref"), "authority_readiness=" + stringField(contract, "authority_readiness_state")}
	for _, entry := range []struct{ key, prefix string }{{"allowed_transitions", "allow="}, {"prohibited_transitions", "prohibit="}, {"approval_required_transitions", "approval="}} {
		for _, item := range objectArray(contract, entry.key) {
			parts = append(parts, entry.prefix+strings.Join([]string{stringField(item, "transition_id"), stringField(item, "from_stage_id"), stringField(item, "to_stage_id"), stringField(item, "from_role"), stringField(item, "to_role"), stringField(item, "reason")}, "|"))
		}
	}
	for _, item := range objectArray(contract, "target_constraints") {
		parts = append(parts, "target="+stringField(item, "key")+"="+stringField(item, "value"))
	}
	for _, value := range stringArray(contract, "evidence_requirements") {
		parts = append(parts, "evidence="+value)
	}
	for _, value := range stringArray(contract, "acceptable_countersigners") {
		parts = append(parts, "countersigner="+value)
	}
	for _, value := range stringArray(contract, "source_digests") {
		parts = append(parts, "digest="+value)
	}
	for _, value := range stringArray(contract, "reason_codes") {
		parts = append(parts, "reason="+value)
	}
	for _, item := range objectArray(contract, "authority_requirements") {
		parts = append(parts, "authority="+strings.Join([]string{stringField(item, "requirement_id"), stringField(item, "kind"), stringField(item, "required_constraint"), stringField(item, "observed_value"), stringField(item, "evidence_state"), stringField(item, "freshness_state"), strings.Join(stringArray(item, "evidence_refs"), ","), strings.Join(stringArray(item, "reason_codes"), ",")}, "|"))
	}
	for _, item := range objectArray(contract, "preconditions") {
		parts = append(parts, "precondition="+strings.Join([]string{stringField(item, "requirement_id"), stringField(item, "kind"), stringField(item, "required_constraint"), stringField(item, "observed_value"), stringField(item, "observed_result"), strings.Join(stringArray(item, "acceptable_producers"), ","), stringField(item, "max_age"), stringField(item, "evidence_state"), stringField(item, "freshness_state"), strings.Join(stringArray(item, "evidence_refs"), ","), strings.Join(stringArray(item, "reason_codes"), ",")}, "|"))
	}
	if item, ok := contract["confirmation_requirement"].(map[string]any); ok {
		parts = append(parts, "confirmation="+strings.Join([]string{stringField(item, "mode"), strconv.FormatBool(boolField(item, "required")), stringField(item, "evidence_state"), stringField(item, "freshness_state"), strings.Join(stringArray(item, "evidence_refs"), ","), strings.Join(stringArray(item, "reason_codes"), ",")}, "|"))
	}
	if item, ok := contract["approval_requirement"].(map[string]any); ok {
		parts = append(parts, "approval="+strings.Join([]string{strconv.FormatBool(boolField(item, "required")), strings.Join(stringArray(item, "approver_roles"), ","), strconv.Itoa(intField(item, "minimum_approvals")), strings.Join(stringArray(item, "separation_of_duties"), ","), stringField(item, "scope_digest"), stringField(item, "validity_window"), strings.Join(stringArray(item, "reapproval_triggers"), ","), stringField(item, "evidence_state"), stringField(item, "freshness_state"), strings.Join(stringArray(item, "evidence_refs"), ","), strings.Join(stringArray(item, "reason_codes"), ",")}, "|"))
	}
	if item, ok := contract["compensation_requirement"].(map[string]any); ok {
		parts = append(parts, "compensation="+strings.Join([]string{strconv.FormatBool(boolField(item, "required")), stringField(item, "kind"), stringField(item, "procedure_ref"), stringField(item, "target"), stringField(item, "execution_window"), strconv.FormatBool(boolField(item, "verification_required")), strings.Join(stringArray(item, "acceptable_producers"), ","), stringField(item, "evidence_state"), stringField(item, "freshness_state"), strings.Join(stringArray(item, "evidence_refs"), ","), strings.Join(stringArray(item, "reason_codes"), ",")}, "|"))
	}
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func shortHash(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:8])
}

func proposalCanonicalDigest(data map[string]any) (string, error) {
	payload := map[string]any{}
	for _, key := range []string{"schema_id", "schema_version", "contract_id", "contract_family_id", "revision", "producer", "source_scan_refs", "composition_refs", "creation_evidence", "variant", "report_only", "contract"} {
		if value, ok := data[key]; ok {
			payload[key] = value
		}
	}
	if rawProducer, ok := data["producer"]; ok {
		encodedProducer, err := json.Marshal(rawProducer)
		if err != nil {
			return "", err
		}
		var producer ProducerMetadata
		if err := json.Unmarshal(encodedProducer, &producer); err != nil {
			return "", err
		}
		payload["producer"] = producer
	}
	if value, ok := data["resolution_key"]; ok && strings.TrimSpace(fmt.Sprint(value)) != "" {
		payload["resolution_key"] = value
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	digest, err := proofcanon.DigestJCS(encoded)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(digest, "sha256:") {
		digest = "sha256:" + digest
	}
	return digest, nil
}
