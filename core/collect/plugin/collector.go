package plugin

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Clyra-AI/axym/core/collect/collectorerr"
	"github.com/Clyra-AI/axym/core/collector"
	"github.com/Clyra-AI/proof"
)

const (
	ReasonExecFailed = "PLUGIN_EXEC_FAILED"
	ReasonTimeout    = "PLUGIN_TIMEOUT"
	ReasonMalformed  = "PLUGIN_MALFORMED_JSONL"
)

type Collector struct {
	Command string
	Timeout time.Duration
	NameID  string
	Env     []string
}

type configPayload struct {
	Name string `json:"name"`
	Now  string `json:"now"`
}

type outputRecord struct {
	SourceType    string              `json:"source_type"`
	Source        string              `json:"source"`
	SourceProduct string              `json:"source_product"`
	RecordType    string              `json:"record_type"`
	AgentID       string              `json:"agent_id"`
	Timestamp     string              `json:"timestamp"`
	Event         map[string]any      `json:"event"`
	Metadata      map[string]any      `json:"metadata"`
	Relationship  *proof.Relationship `json:"relationship,omitempty"`
	Controls      struct {
		PermissionsEnforced bool   `json:"permissions_enforced"`
		ApprovedScope       string `json:"approved_scope"`
	} `json:"controls"`
}

func (c Collector) Name() string {
	if strings.TrimSpace(c.NameID) != "" {
		return c.NameID
	}
	parts := strings.Fields(c.Command)
	if len(parts) == 0 {
		return "plugin"
	}
	return "plugin:" + filepath.Base(parts[0])
}

func (c Collector) Collect(ctx context.Context, req collector.Request) (collector.Result, error) {
	commandParts := strings.Fields(c.Command)
	if len(commandParts) == 0 {
		return collector.Result{}, nil
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = req.PluginTimeout
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	collectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	payloadNow := req.Now.UTC().Format(time.RFC3339)
	if req.Now.IsZero() {
		payloadNow = "2026-02-28T00:00:00Z"
	}
	cfgRaw, err := json.Marshal(configPayload{Name: c.Name(), Now: payloadNow})
	if err != nil {
		return collector.Result{}, collectorerr.New(ReasonExecFailed, "marshal plugin config", err)
	}

	// #nosec G204 -- plugin command is an explicit user-supplied integration hook.
	cmd := exec.CommandContext(collectCtx, commandParts[0], commandParts[1:]...)
	cmd.Stdin = bytes.NewReader(cfgRaw)
	if len(c.Env) > 0 {
		cmd.Env = append(cmd.Environ(), c.Env...)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	if collectCtx.Err() != nil {
		if errors.Is(collectCtx.Err(), context.DeadlineExceeded) {
			return collector.Result{}, collectorerr.New(ReasonTimeout, "plugin timed out", collectCtx.Err())
		}
		return collector.Result{}, collectorerr.New(ReasonExecFailed, "plugin context failed", collectCtx.Err())
	}
	if runErr != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = runErr.Error()
		}
		return collector.Result{}, collectorerr.New(ReasonExecFailed, message, runErr)
	}

	candidates := []collector.Candidate{}
	scanner := bufio.NewScanner(bytes.NewReader(stdout.Bytes()))
	line := 0
	for scanner.Scan() {
		line++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}
		var out outputRecord
		if err := json.Unmarshal([]byte(raw), &out); err != nil {
			return collector.Result{}, collectorerr.New(ReasonMalformed, fmt.Sprintf("decode line %d", line), err)
		}
		timestamp, err := time.Parse(time.RFC3339, out.Timestamp)
		if err != nil {
			return collector.Result{}, collectorerr.New(ReasonMalformed, fmt.Sprintf("parse timestamp on line %d", line), err)
		}
		sourceType := out.SourceType
		if sourceType == "" {
			sourceType = "plugin"
		}
		sourceProduct := out.SourceProduct
		if sourceProduct == "" {
			sourceProduct = "axym"
		}
		candidates = append(candidates, collector.Candidate{
			SourceType:    sourceType,
			Source:        out.Source,
			SourceProduct: sourceProduct,
			RecordType:    out.RecordType,
			AgentID:       out.AgentID,
			Timestamp:     timestamp.UTC().Truncate(time.Second),
			Event:         out.Event,
			Metadata:      out.Metadata,
			Relationship:  out.Relationship,
			Controls: collector.Controls{
				PermissionsEnforced: out.Controls.PermissionsEnforced,
				ApprovedScope:       out.Controls.ApprovedScope,
			},
		})
	}
	if err := scanner.Err(); err != nil {
		return collector.Result{}, collectorerr.New(ReasonMalformed, "scan plugin output", err)
	}
	return collector.Result{Candidates: candidates, ReasonCodes: []string{"CAPTURED"}}, nil
}
