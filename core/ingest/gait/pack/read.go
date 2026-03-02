package pack

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Clyra-AI/axym/core/ingest/gait/translate"
	"github.com/Clyra-AI/proof"
)

type Result struct {
	ProofRecords  []*proof.Record
	NativeRecords []translate.NativeRecord
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
	nativeRecords, err := parseNativeJSONLFile(cleaned)
	if err != nil {
		return Result{}, err
	}
	return Result{NativeRecords: nativeRecords}, nil
}

func readDirectory(dir string) (Result, error) {
	result := Result{}
	foundSupportedEntry := false
	proofPath := filepath.Join(dir, "proof_records.jsonl")
	if _, err := os.Stat(proofPath); err == nil {
		foundSupportedEntry = true
		proofRecords, parseErr := parseProofJSONLFile(proofPath)
		if parseErr != nil {
			return Result{}, parseErr
		}
		result.ProofRecords = append(result.ProofRecords, proofRecords...)
	}

	nativePath := filepath.Join(dir, "native_records.jsonl")
	if _, err := os.Stat(nativePath); err == nil {
		foundSupportedEntry = true
		nativeRecords, parseErr := parseNativeJSONLFile(nativePath)
		if parseErr != nil {
			return Result{}, parseErr
		}
		result.NativeRecords = append(result.NativeRecords, nativeRecords...)
	}
	if !foundSupportedEntry {
		return Result{}, fmt.Errorf("gait pack directory %s must contain at least one of proof_records.jsonl or native_records.jsonl", dir)
	}
	return result, nil
}

func readZip(path string) (Result, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return Result{}, fmt.Errorf("open gait pack zip: %w", err)
	}
	defer func() { _ = reader.Close() }()

	result := Result{}
	entries := make([]*zip.File, 0, len(reader.File))
	for _, file := range reader.File {
		entries = append(entries, file)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	for _, entry := range entries {
		name := strings.ToLower(filepath.Base(entry.Name))
		if name != "proof_records.jsonl" && name != "native_records.jsonl" {
			continue
		}
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
		}
	}
	return result, nil
}

func parseProofJSONLFile(path string) ([]*proof.Record, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read proof_records.jsonl: %w", err)
	}
	return parseProofJSONL(raw)
}

func parseNativeJSONLFile(path string) ([]translate.NativeRecord, error) {
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
