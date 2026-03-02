package dlq

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Clyra-AI/axym/core/store"
)

const defaultFile = "ticket-dlq.jsonl"

type Queue struct {
	rootDir string
	file    string
}

type Entry struct {
	System      string   `json:"system"`
	ChangeID    string   `json:"change_id"`
	PayloadHash string   `json:"payload_hash"`
	ReasonCodes []string `json:"reason_codes"`
	Attempts    int      `json:"attempts"`
	OccurredAt  string   `json:"occurred_at"`
}

func New(rootDir string) (*Queue, error) {
	cleaned := filepath.Clean(strings.TrimSpace(rootDir))
	if cleaned == "" || cleaned == "." {
		return nil, fmt.Errorf("dlq root dir is required")
	}
	if err := os.MkdirAll(cleaned, 0o700); err != nil {
		return nil, fmt.Errorf("create dlq root: %w", err)
	}
	return &Queue{rootDir: cleaned, file: filepath.Join(cleaned, defaultFile)}, nil
}

func (q *Queue) Path() string {
	if q == nil {
		return ""
	}
	return q.file
}

func (q *Queue) Enqueue(entry Entry) (string, error) {
	if q == nil {
		return "", fmt.Errorf("dlq queue is nil")
	}
	if err := ensureRegularFilePath(q.file); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(entry)
	if err != nil {
		return "", fmt.Errorf("marshal dlq entry: %w", err)
	}
	encoded = append(encoded, '\n')

	fh, err := os.OpenFile(q.file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return "", fmt.Errorf("open dlq file: %w", err)
	}
	defer func() { _ = fh.Close() }()
	if _, err := fh.Write(encoded); err != nil {
		return "", fmt.Errorf("write dlq entry: %w", err)
	}
	if err := fh.Sync(); err != nil {
		return "", fmt.Errorf("sync dlq entry: %w", err)
	}
	return q.file, nil
}

func ensureRegularFilePath(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat dlq path: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("dlq path must not be symlink")
	}
	if info.IsDir() {
		return fmt.Errorf("dlq path must be regular file")
	}
	return nil
}

func ReadAll(path string) ([]Entry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, fmt.Errorf("read dlq: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []Entry{}, nil
	}
	entries := make([]Entry, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		var entry Entry
		if err := json.Unmarshal([]byte(trimmed), &entry); err != nil {
			return nil, fmt.Errorf("decode dlq line: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func AtomicWrite(path string, data []byte) error {
	return store.WriteJSONAtomic(path, data, true)
}
