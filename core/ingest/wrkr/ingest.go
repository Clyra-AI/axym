package wrkr

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Clyra-AI/axym/core/ingest/state"
	"github.com/Clyra-AI/axym/core/normalize"
	"github.com/Clyra-AI/axym/core/review/privilegedrift"
	"github.com/Clyra-AI/axym/core/store"
	"github.com/Clyra-AI/axym/core/store/dedupe"
	"github.com/Clyra-AI/proof"
)

const (
	ReasonNoInput             = "NO_INPUT"
	ReasonInvalidInput        = "WRKR_INVALID_INPUT"
	ReasonInvalidPayload      = "WRKR_INVALID_PAYLOAD"
	ReasonUnsupportedType     = "WRKR_UNSUPPORTED_RECORD_TYPE"
	ReasonStateUnavailable    = "WRKR_STATE_UNAVAILABLE"
	ReasonStateLocked         = "WRKR_STATE_LOCKED"
	ReasonAppendFailed        = "WRKR_CHAIN_APPEND_FAILED"
	ReasonContextCanceled     = "WRKR_CONTEXT_CANCELED"
	ReasonPrivilegeEscalation = privilegedrift.ReasonClassUnapprovedPrivilegeEscalation
)

var supportedRecordTypes = map[string]struct{}{
	"tool_invocation":    {},
	"decision":           {},
	"policy_enforcement": {},
	"approval":           {},
	"risk_assessment":    {},
	"scan_finding":       {},
}

type Request struct {
	InputPaths []string
	Store      *store.Store
	StateDir   string
}

type Result struct {
	Source        string                   `json:"source"`
	InputFiles    int                      `json:"input_files"`
	Parsed        int                      `json:"parsed"`
	Appended      int                      `json:"appended"`
	Deduped       int                      `json:"deduped"`
	Rejected      int                      `json:"rejected"`
	RecordCount   int                      `json:"record_count"`
	HeadHash      string                   `json:"head_hash,omitempty"`
	StatePath     string                   `json:"state_path"`
	ReasonCodes   []string                 `json:"reason_codes"`
	DriftGaps     []privilegedrift.Gap     `json:"drift_gaps"`
	IdentityViews []normalize.IdentityView `json:"identity_views,omitempty"`
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
	stateDir := req.StateDir
	if strings.TrimSpace(stateDir) == "" {
		stateDir = req.Store.RootDir()
	}
	manager := state.NewWrkrManager(stateDir)
	result := Result{
		Source:      "wrkr",
		StatePath:   manager.StatePath(),
		ReasonCodes: []string{},
		DriftGaps:   []privilegedrift.Gap{},
	}

	paths := normalizePaths(req.InputPaths)
	result.InputFiles = len(paths)
	if len(paths) == 0 {
		result.ReasonCodes = []string{ReasonNoInput}
		return result, nil
	}

	records, rejectCodes, err := loadRecords(paths)
	result.ReasonCodes = append(result.ReasonCodes, rejectCodes...)
	result.Rejected += len(rejectCodes)
	if err != nil {
		return Result{}, err
	}
	result.Parsed = len(records)
	for _, rec := range records {
		appendIdentityView(&result, normalize.IdentityViewFromRecord(rec))
	}
	if len(records) == 0 {
		result.ReasonCodes = uniqueSorted(result.ReasonCodes)
		return result, nil
	}

	sort.Slice(records, func(i, j int) bool {
		ti := records[i].Timestamp.UTC()
		tj := records[j].Timestamp.UTC()
		if !ti.Equal(tj) {
			return ti.Before(tj)
		}
		if records[i].RecordID != records[j].RecordID {
			return records[i].RecordID < records[j].RecordID
		}
		return records[i].Integrity.RecordHash < records[j].Integrity.RecordHash
	})

	var observations []privilegedrift.Observation
	for _, rec := range records {
		if obs, ok := observationFromRecord(rec); ok {
			observations = append(observations, obs)
		}
	}

	if err := manager.WithLockedState(func(st *state.WrkrState) error {
		select {
		case <-ctx.Done():
			return &Error{ReasonCode: ReasonContextCanceled, Message: "ingest canceled", Err: ctx.Err()}
		default:
		}

		updated, gaps := privilegedrift.Analyze(st.PrivilegeBaseline, observations)
		st.PrivilegeBaseline = updated
		result.DriftGaps = gaps
		if len(gaps) > 0 {
			result.ReasonCodes = append(result.ReasonCodes, ReasonPrivilegeEscalation)
		}

		for _, rec := range records {
			key, err := dedupe.BuildKey(rec.SourceProduct, rec.RecordType, rec.Event)
			if err != nil {
				result.Rejected++
				result.ReasonCodes = append(result.ReasonCodes, ReasonInvalidPayload)
				continue
			}
			appendResult, err := req.Store.Append(rec, key)
			if err != nil {
				return &Error{ReasonCode: ReasonAppendFailed, Message: "append wrkr record", Err: err}
			}
			result.RecordCount = appendResult.RecordCount
			result.HeadHash = appendResult.HeadHash
			if appendResult.Deduped {
				result.Deduped++
				continue
			}
			if appendResult.Appended {
				result.Appended++
				continue
			}
			result.Rejected++
			result.ReasonCodes = append(result.ReasonCodes, ReasonAppendFailed)
		}
		return nil
	}); err != nil {
		if errors.Is(err, state.ErrStateLocked) {
			return Result{}, &Error{ReasonCode: ReasonStateLocked, Message: "wrkr ingest state lock is held", Err: err}
		}
		var ingestErr *Error
		if errors.As(err, &ingestErr) {
			return Result{}, ingestErr
		}
		return Result{}, &Error{ReasonCode: ReasonStateUnavailable, Message: "load or persist wrkr ingest state", Err: err}
	}

	result.ReasonCodes = uniqueSorted(result.ReasonCodes)
	return result, nil
}

func loadRecords(paths []string) ([]*proof.Record, []string, error) {
	out := make([]*proof.Record, 0)
	reasons := make([]string, 0)
	for _, path := range paths {
		records, errs, err := readRecords(path)
		reasons = append(reasons, errs...)
		if err != nil {
			return nil, reasons, &Error{ReasonCode: ReasonInvalidInput, Message: fmt.Sprintf("read %s", path), Err: err}
		}
		out = append(out, records...)
	}
	return out, reasons, nil
}

func readRecords(path string) ([]*proof.Record, []string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, nil, err
	}
	if info.IsDir() {
		candidate := filepath.Join(path, "proof_records.jsonl")
		if _, err := os.Stat(candidate); err != nil {
			return nil, nil, fmt.Errorf("directory %s must contain proof_records.jsonl", path)
		}
		return readJSONLRecords(candidate)
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".jsonl":
		return readJSONLRecords(path)
	case ".json":
		return readJSONRecords(path)
	default:
		return nil, nil, fmt.Errorf("unsupported wrkr input file extension for %s", path)
	}
}

func readJSONLRecords(path string) ([]*proof.Record, []string, error) {
	// #nosec G304 -- Wrkr ingest reads the explicit user-provided input path.
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = file.Close() }()

	records := make([]*proof.Record, 0)
	reasons := make([]string, 0)
	scanner := bufio.NewScanner(file)
	buffer := make([]byte, 0, 1024*64)
	scanner.Buffer(buffer, 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record proof.Record
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			reasons = append(reasons, ReasonInvalidPayload)
			continue
		}
		if !isSupportedRecordType(record.RecordType) {
			reasons = append(reasons, ReasonUnsupportedType)
			continue
		}
		if err := proof.ValidateRecord(&record); err != nil {
			reasons = append(reasons, ReasonInvalidPayload)
			continue
		}
		records = append(records, &record)
	}
	if err := scanner.Err(); err != nil {
		return nil, reasons, err
	}
	return records, reasons, nil
}

func readJSONRecords(path string) ([]*proof.Record, []string, error) {
	// #nosec G304 -- Wrkr ingest reads the explicit user-provided input path.
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	var list []proof.Record
	if err := json.Unmarshal(raw, &list); err == nil {
		return filterAndValidate(list)
	}

	var payload struct {
		Records []proof.Record `json:"records"`
	}
	if err := json.Unmarshal(raw, &payload); err == nil {
		return filterAndValidate(payload.Records)
	}
	return nil, nil, fmt.Errorf("unsupported json record format in %s", path)
}

func filterAndValidate(records []proof.Record) ([]*proof.Record, []string, error) {
	out := make([]*proof.Record, 0, len(records))
	reasons := make([]string, 0)
	for i := range records {
		record := records[i]
		if !isSupportedRecordType(record.RecordType) {
			reasons = append(reasons, ReasonUnsupportedType)
			continue
		}
		if err := proof.ValidateRecord(&record); err != nil {
			reasons = append(reasons, ReasonInvalidPayload)
			continue
		}
		out = append(out, &record)
	}
	return out, reasons, nil
}

func isSupportedRecordType(recordType string) bool {
	_, ok := supportedRecordTypes[strings.TrimSpace(recordType)]
	return ok
}

func observationFromRecord(record *proof.Record) (privilegedrift.Observation, bool) {
	if record == nil {
		return privilegedrift.Observation{}, false
	}
	principal := firstString(
		stringFromMap(record.Metadata, "principal_id"),
		stringFromMap(record.Event, "principal_id"),
		strings.TrimSpace(record.AgentID),
	)
	privilege := firstString(
		stringFromMap(record.Event, "privilege"),
		stringFromMap(record.Event, "scope"),
		stringFromMap(record.Metadata, "scope"),
	)
	if principal == "" || privilege == "" {
		return privilegedrift.Observation{}, false
	}
	approved := boolFromMap(record.Event, "approved") || boolFromMap(record.Metadata, "approved")
	return privilegedrift.Observation{
		Principal: principal,
		Privilege: privilege,
		Approved:  approved,
		RecordID:  record.RecordID,
	}, true
}

func normalizePaths(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, raw := range in {
		candidate := strings.TrimSpace(raw)
		if candidate == "" {
			continue
		}
		cleaned := filepath.Clean(candidate)
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		out = append(out, cleaned)
	}
	sort.Strings(out)
	return out
}

func uniqueSorted(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, candidate := range in {
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
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

func firstString(candidates ...string) string {
	for _, candidate := range candidates {
		trimmed := strings.TrimSpace(candidate)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func stringFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func boolFromMap(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	v, ok := m[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}
