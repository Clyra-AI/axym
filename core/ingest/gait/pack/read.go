package pack

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Clyra-AI/axym/core/ingest/gait/evidence"
	"github.com/Clyra-AI/axym/core/ingest/gait/translate"
	"github.com/Clyra-AI/proof"
)

type Result struct {
	ProofRecords   []*proof.Record
	NativeRecords  []translate.NativeRecord
	Artifacts      []translate.SourceArtifact
	LifecyclePacks []evidence.LifecyclePack
	ReasonCodes    []string
}

type LifecycleError struct {
	ReasonCode string
	Message    string
	Err        error
}

func (e *LifecycleError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return fmt.Sprintf("%s: %s", e.ReasonCode, e.Message)
	}
	return fmt.Sprintf("%s: %s: %v", e.ReasonCode, e.Message, e.Err)
}

func (e *LifecycleError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func Read(path string) (Result, error) {
	cleaned := strings.TrimSpace(path)
	if cleaned == "" {
		return Result{}, nil
	}
	info, err := os.Stat(cleaned)
	if err != nil {
		return Result{}, fmt.Errorf("stat gait pack path: %w", err)
	}
	if info.IsDir() {
		return readDirectory(cleaned)
	}
	if strings.EqualFold(filepath.Ext(cleaned), ".zip") {
		return readZip(cleaned)
	}
	if strings.EqualFold(filepath.Base(cleaned), "proof_records.jsonl") {
		proofRecords, err := parseProofJSONLFile(cleaned)
		if err != nil {
			return Result{}, err
		}
		return Result{ProofRecords: proofRecords}, nil
	}
	if strings.EqualFold(filepath.Base(cleaned), "lifecycle.json") {
		raw, err := os.ReadFile(cleaned) // #nosec G304 -- explicit user-provided pack path.
		if err != nil {
			return Result{}, fmt.Errorf("read lifecycle evidence: %w", err)
		}
		lifecycle, err := parseLifecycle(raw)
		if err != nil {
			return Result{}, err
		}
		return Result{LifecyclePacks: []evidence.LifecyclePack{lifecycle}}, nil
	}
	if strings.EqualFold(filepath.Ext(cleaned), ".json") {
		artifacts, err := parseSourceArtifactFile(cleaned, filepath.ToSlash(filepath.Base(cleaned)))
		if err != nil {
			return Result{}, err
		}
		if len(artifacts) == 0 {
			return Result{}, &translate.Error{
				ReasonCode: translate.ReasonInvalidAuthorizationArtifact,
				Message:    fmt.Sprintf("unsupported gait source artifact file %s", cleaned),
			}
		}
		return Result{Artifacts: artifacts}, nil
	}
	nativeRecords, err := parseNativeJSONLFile(cleaned)
	if err != nil {
		return Result{}, err
	}
	return Result{NativeRecords: nativeRecords}, nil
}

func readDirectory(dir string) (Result, error) {
	result := Result{}
	foundSupportedEntry := false
	entries := make([]string, 0)
	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		entries = append(entries, path)
		return nil
	}); err != nil {
		return Result{}, fmt.Errorf("walk gait pack directory: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		left, _ := filepath.Rel(dir, entries[i])
		right, _ := filepath.Rel(dir, entries[j])
		return filepath.ToSlash(left) < filepath.ToSlash(right)
	})
	for _, entry := range entries {
		rel, err := filepath.Rel(dir, entry)
		if err != nil {
			return Result{}, fmt.Errorf("compute gait pack relative path: %w", err)
		}
		rel = filepath.ToSlash(rel)
		name := strings.ToLower(filepath.Base(entry))
		switch {
		case name == "proof_records.jsonl":
			foundSupportedEntry = true
			proofRecords, parseErr := parseProofJSONLFile(entry)
			if parseErr != nil {
				return Result{}, parseErr
			}
			result.ProofRecords = append(result.ProofRecords, proofRecords...)
		case name == "native_records.jsonl":
			foundSupportedEntry = true
			nativeRecords, parseErr := parseNativeJSONLFile(entry)
			if parseErr != nil {
				return Result{}, parseErr
			}
			result.NativeRecords = append(result.NativeRecords, nativeRecords...)
		case name == "lifecycle.json":
			foundSupportedEntry = true
			raw, parseErr := os.ReadFile(entry) // #nosec G304 -- discovered below the explicit pack root.
			if parseErr != nil {
				return Result{}, fmt.Errorf("read lifecycle evidence: %w", parseErr)
			}
			lifecycle, parseErr := parseLifecycle(raw)
			if parseErr != nil {
				return Result{}, parseErr
			}
			result.LifecyclePacks = append(result.LifecyclePacks, lifecycle)
		case strings.EqualFold(filepath.Ext(entry), ".json"):
			artifacts, reasons, parseErr := parseSourceArtifactEntryFile(entry, rel)
			if parseErr != nil {
				return Result{}, parseErr
			}
			if len(artifacts) > 0 || len(reasons) > 0 {
				foundSupportedEntry = true
			}
			result.Artifacts = append(result.Artifacts, artifacts...)
			result.ReasonCodes = append(result.ReasonCodes, reasons...)
		}
	}
	if !foundSupportedEntry {
		return Result{}, fmt.Errorf("gait pack directory %s must contain at least one supported proof, native, or source artifact entry", dir)
	}
	if err := validateAuthorizationBundleIDs(result.Artifacts); err != nil {
		return Result{}, err
	}
	result.ReasonCodes = uniqueSorted(result.ReasonCodes)
	return result, nil
}

func readZip(path string) (Result, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return Result{}, fmt.Errorf("open gait pack zip: %w", err)
	}
	defer func() { _ = reader.Close() }()

	result := Result{}
	entries := append(make([]*zip.File, 0, len(reader.File)), reader.File...)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	for _, entry := range entries {
		name := strings.ToLower(filepath.Base(entry.Name))
		switch {
		case name == "proof_records.jsonl", name == "native_records.jsonl", name == "lifecycle.json", strings.EqualFold(filepath.Ext(entry.Name), ".json"):
			fh, err := entry.Open()
			if err != nil {
				return Result{}, fmt.Errorf("open zip entry %s: %w", entry.Name, err)
			}
			data, err := io.ReadAll(fh)
			_ = fh.Close()
			if err != nil {
				return Result{}, fmt.Errorf("read zip entry %s: %w", entry.Name, err)
			}
			switch name {
			case "proof_records.jsonl":
				records, parseErr := parseProofJSONL(data)
				if parseErr != nil {
					return Result{}, parseErr
				}
				result.ProofRecords = append(result.ProofRecords, records...)
			case "native_records.jsonl":
				records, parseErr := parseNativeJSONL(data)
				if parseErr != nil {
					return Result{}, parseErr
				}
				result.NativeRecords = append(result.NativeRecords, records...)
			case "lifecycle.json":
				lifecycle, parseErr := parseLifecycle(data)
				if parseErr != nil {
					return Result{}, parseErr
				}
				result.LifecyclePacks = append(result.LifecyclePacks, lifecycle)
			default:
				artifacts, reasons, parseErr := parseSourceArtifactData(data, filepath.ToSlash(entry.Name))
				if parseErr != nil {
					return Result{}, parseErr
				}
				result.Artifacts = append(result.Artifacts, artifacts...)
				result.ReasonCodes = append(result.ReasonCodes, reasons...)
			}
		}
	}
	if err := validateAuthorizationBundleIDs(result.Artifacts); err != nil {
		return Result{}, err
	}
	result.ReasonCodes = uniqueSorted(result.ReasonCodes)
	return result, nil
}

func parseLifecycle(raw []byte) (evidence.LifecyclePack, error) {
	lifecycle, err := evidence.ParseLifecyclePack(raw)
	if err == nil {
		return lifecycle, nil
	}
	reason := evidence.ReasonMalformed
	for _, candidate := range []string{evidence.ReasonUnknownField, evidence.ReasonSchemaUnsupported, evidence.ReasonEvidenceMissing, evidence.ReasonMalformed} {
		if strings.Contains(err.Error(), candidate) {
			reason = candidate
			break
		}
	}
	return evidence.LifecyclePack{}, &LifecycleError{ReasonCode: reason, Message: "parse Gait lifecycle evidence", Err: err}
}

func parseProofJSONLFile(path string) ([]*proof.Record, error) {
	// #nosec G304 -- Gait ingest reads the explicit user-provided pack entry path.
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read proof_records.jsonl: %w", err)
	}
	return parseProofJSONL(raw)
}

func parseNativeJSONLFile(path string) ([]translate.NativeRecord, error) {
	// #nosec G304 -- Gait ingest reads the explicit user-provided pack entry path.
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read native records: %w", err)
	}
	return parseNativeJSONL(raw)
}

func parseProofJSONL(raw []byte) ([]*proof.Record, error) {
	records := make([]*proof.Record, 0)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record proof.Record
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, fmt.Errorf("decode proof record line: %w", err)
		}
		if err := proof.ValidateRecord(&record); err != nil {
			return nil, fmt.Errorf("validate proof record line: %w", err)
		}
		records = append(records, &record)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func parseNativeJSONL(raw []byte) ([]translate.NativeRecord, error) {
	records := make([]translate.NativeRecord, 0)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record translate.NativeRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, fmt.Errorf("decode native record line: %w", err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func parseSourceArtifactFile(path string, sourcePath string) ([]translate.SourceArtifact, error) {
	// #nosec G304 -- Gait ingest reads the explicit user-provided source artifact path.
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read gait source artifact: %w", err)
	}
	artifacts, _, err := parseSourceArtifactData(raw, sourcePath)
	return artifacts, err
}

func parseSourceArtifactEntryFile(path string, sourcePath string) ([]translate.SourceArtifact, []string, error) {
	// #nosec G304 -- Gait ingest reads the explicit user-provided pack entry path.
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read gait source artifact: %w", err)
	}
	return parseSourceArtifactData(raw, sourcePath)
}

func parseSourceArtifactData(raw []byte, sourcePath string) ([]translate.SourceArtifact, []string, error) {
	artifacts, err := translate.ParseSourceArtifact(raw, sourcePath)
	if err == nil {
		return artifacts, nil, nil
	}
	var tErr *translate.Error
	if !strings.EqualFold(filepath.Ext(sourcePath), ".json") {
		return nil, nil, err
	}
	if errors.As(err, &tErr) && tErr.ReasonCode == translate.ReasonUnsupportedArtifactType {
		return nil, []string{translate.ReasonUnsupportedArtifactType}, nil
	}
	return nil, nil, err
}

func validateAuthorizationBundleIDs(artifacts []translate.SourceArtifact) error {
	seen := map[string]struct{}{}
	for _, artifact := range artifacts {
		kind := artifact.Kind()
		if kind != translate.ArtifactTypeAuthorizationBundle && kind != translate.ArtifactTypeAuthorizationProfile {
			continue
		}
		bundleID := strings.TrimSpace(artifact.BundleID())
		if bundleID == "" {
			continue
		}
		if _, ok := seen[bundleID]; ok {
			return &translate.Error{
				ReasonCode: translate.ReasonDuplicateAuthorizationBundleID,
				Message:    fmt.Sprintf("duplicate authorization bundle_id %q", bundleID),
			}
		}
		seen[bundleID] = struct{}{}
	}
	return nil
}

func uniqueSorted(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}
