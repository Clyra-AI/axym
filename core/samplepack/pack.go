package samplepack

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultDirPerm  = 0o700
	defaultFilePerm = 0o600
)

type Result struct {
	Path      string   `json:"path"`
	Files     []File   `json:"files"`
	NextSteps []string `json:"next_steps"`
}

type File struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
}

type asset struct {
	RelPath  string
	Kind     string
	Contents string
}

type fileOps struct {
	mkdirAll  func(string, os.FileMode) error
	mkdtemp   func(string, string) (string, error)
	writeFile func(string, []byte, os.FileMode) error
	rename    func(string, string) error
	removeAll func(string) error
	stat      func(string) (os.FileInfo, error)
}

var sampleAssets = []asset{
	{
		RelPath: "governance/context_engineering.jsonl",
		Kind:    "governance_events",
		Contents: strings.TrimSpace(`
{"event_type":"instruction_rewrite","source":"agent-fw","timestamp":"2026-03-18T12:00:00Z","actor":{"id":"agent-1","type":"agent"},"action":"rewrite","target":{"kind":"instruction_set","id":"system-prompt"},"context":{"previous_hash":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","current_hash":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","artifact_kind":"instruction_set","reason_code":"POLICY_REFRESH"}}
{"event_type":"context_reset","source":"agent-fw","timestamp":"2026-03-18T12:05:00Z","actor":{"id":"agent-1","type":"agent"},"action":"reset","target":{"kind":"context_window","id":"conversation-memory"},"context":{"previous_hash":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","current_hash":"sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd","reason_code":"SESSION_BOUNDARY"}}
{"event_type":"knowledge_import","source":"agent-fw","timestamp":"2026-03-18T12:10:00Z","actor":{"id":"agent-1","type":"agent"},"action":"import","target":{"kind":"knowledge_artifact","id":"kb:policy-pack"},"context":{"artifact_digest":"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee","artifact_kind":"knowledge_artifact","source_uri":"repo://policy/pack","reason_code":"KNOWLEDGE_SYNC","approval_ref":"chg-42"}}
`) + "\n",
	},
	{
		RelPath: "records/approval.json",
		Kind:    "proof_record",
		Contents: strings.TrimSpace(`
{
  "record_version": "v1",
  "record_id": "sample-approval-001",
  "source": "manual",
  "source_product": "axym",
  "agent_id": "ops-reviewer",
  "record_type": "approval",
  "timestamp": "2026-03-18T12:15:00Z",
  "event": {
    "decision": "allow",
    "approver": "ops-reviewer",
    "scope": "context-pack-change"
  },
  "metadata": {
    "reason_code": "HUMAN_APPROVED"
  },
  "controls": {
    "permissions_enforced": true,
    "approved_scope": "sample:first-value"
  }
}
`) + "\n",
	},
	{
		RelPath: "records/risk_assessment.json",
		Kind:    "proof_record",
		Contents: strings.TrimSpace(`
{
  "record_version": "v1",
  "record_id": "sample-risk-001",
  "source": "manual",
  "source_product": "axym",
  "agent_id": "risk-analyst",
  "record_type": "risk_assessment",
  "timestamp": "2026-03-18T12:20:00Z",
  "event": {
    "risk_id": "risk-sample-001",
    "severity": "medium",
    "summary": "Context change requires documented evaluation"
  },
  "metadata": {
    "reason_code": "RISK_REVIEWED"
  },
  "controls": {
    "permissions_enforced": true,
    "approved_scope": "sample:first-value"
  }
}
`) + "\n",
	},
}

func Create(targetDir string) (Result, error) {
	return createWithOps(targetDir, defaultFileOps())
}

func ValidateTarget(targetDir string) (string, error) {
	trimmed := strings.TrimSpace(targetDir)
	if trimmed == "" {
		return "", fmt.Errorf("sample pack path is required")
	}

	clean := filepath.Clean(trimmed)
	if clean == "." {
		return "", fmt.Errorf("sample pack path must not be the current directory")
	}
	if isFilesystemRoot(clean) {
		return "", fmt.Errorf("sample pack path must not be a filesystem root")
	}
	return clean, nil
}

func createWithOps(targetDir string, ops fileOps) (Result, error) {
	targetDir, err := ValidateTarget(targetDir)
	if err != nil {
		return Result{}, err
	}

	if _, err := ops.stat(targetDir); err == nil {
		return Result{}, fmt.Errorf("sample pack target already exists: %s", targetDir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Result{}, fmt.Errorf("stat sample pack target: %w", err)
	}

	parentDir := filepath.Dir(targetDir)
	if err := ops.mkdirAll(parentDir, defaultDirPerm); err != nil {
		return Result{}, fmt.Errorf("create sample pack parent: %w", err)
	}

	tempDir, err := ops.mkdtemp(parentDir, ".axym-samplepack-*")
	if err != nil {
		return Result{}, fmt.Errorf("create sample pack temp dir: %w", err)
	}
	cleanupTemp := true
	defer func() {
		if cleanupTemp {
			_ = ops.removeAll(tempDir)
		}
	}()

	for _, item := range sampleAssets {
		path := filepath.Join(tempDir, filepath.FromSlash(item.RelPath))
		if err := ops.mkdirAll(filepath.Dir(path), defaultDirPerm); err != nil {
			return Result{}, fmt.Errorf("create sample pack dir: %w", err)
		}
		if err := ops.writeFile(path, []byte(item.Contents), defaultFilePerm); err != nil {
			return Result{}, fmt.Errorf("write sample pack asset %s: %w", item.RelPath, err)
		}
	}

	if err := ops.rename(tempDir, targetDir); err != nil {
		return Result{}, fmt.Errorf("finalize sample pack: %w", err)
	}
	cleanupTemp = false

	files := make([]File, 0, len(sampleAssets))
	for _, item := range sampleAssets {
		files = append(files, File{
			Path: filepath.Join(targetDir, filepath.FromSlash(item.RelPath)),
			Kind: item.Kind,
		})
	}

	return Result{
		Path:      targetDir,
		Files:     files,
		NextSteps: nextSteps(targetDir),
	}, nil
}

func nextSteps(targetDir string) []string {
	return []string{
		fmt.Sprintf("axym collect --json --governance-event-file %s", filepath.Join(targetDir, "governance", "context_engineering.jsonl")),
		fmt.Sprintf("axym record add --input %s --json", filepath.Join(targetDir, "records", "approval.json")),
		fmt.Sprintf("axym record add --input %s --json", filepath.Join(targetDir, "records", "risk_assessment.json")),
		"axym map --frameworks eu-ai-act,soc2 --json",
		"axym gaps --frameworks eu-ai-act,soc2 --json",
		"axym bundle --audit sample --frameworks eu-ai-act,soc2 --json",
		"axym verify --chain --json",
	}
}

func defaultFileOps() fileOps {
	return fileOps{
		mkdirAll:  os.MkdirAll,
		mkdtemp:   os.MkdirTemp,
		writeFile: os.WriteFile,
		rename:    os.Rename,
		removeAll: os.RemoveAll,
		stat:      os.Stat,
	}
}

func isFilesystemRoot(path string) bool {
	volume := filepath.VolumeName(path)
	trimmed := strings.TrimPrefix(path, volume)
	return trimmed == string(os.PathSeparator)
}
