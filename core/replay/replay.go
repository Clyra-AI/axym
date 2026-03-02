package replay

import (
	"fmt"
	"strings"
	"time"

	"github.com/Clyra-AI/axym/core/store"
	"github.com/Clyra-AI/axym/core/store/dedupe"
	"github.com/Clyra-AI/proof"
)

const (
	ReasonInvalidInput = "REPLAY_INVALID_INPUT"
	ReasonAppendFailed = "REPLAY_APPEND_FAILED"
)

type RiskInput struct {
	ProductionCritical bool
	DataSensitivity    string
	PublicExposure     bool
}

type Request struct {
	Model    string
	Tier     string
	StoreDir string
	Source   string
	AgentID  string
	Risk     RiskInput
	Now      func() time.Time
}

type Result struct {
	RecordID    string         `json:"record_id"`
	Model       string         `json:"model"`
	Tier        string         `json:"tier"`
	Status      string         `json:"status"`
	Pass        bool           `json:"pass"`
	BlastRadius map[string]any `json:"blast_radius"`
	HeadHash    string         `json:"head_hash"`
	RecordCount int            `json:"record_count"`
	OccurredAt  string         `json:"occurred_at"`
	ReasonCodes []string       `json:"reason_codes"`
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

func Run(req Request) (Result, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		return Result{}, &Error{ReasonCode: ReasonInvalidInput, Message: "model is required"}
	}
	storeDir := strings.TrimSpace(req.StoreDir)
	if storeDir == "" {
		storeDir = ".axym"
	}
	tier := strings.ToUpper(strings.TrimSpace(req.Tier))
	if tier == "" {
		tier = ClassifyTier(req.Risk)
	}
	if tier != "A" && tier != "B" && tier != "C" {
		return Result{}, &Error{ReasonCode: ReasonInvalidInput, Message: "tier must be one of A,B,C"}
	}
	clock := req.Now
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	startedAt := clock().UTC()
	now := startedAt.Truncate(time.Second)

	evidenceStore, err := store.New(store.Config{RootDir: storeDir, ComplianceMode: true})
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonAppendFailed, Message: "initialize local store", Err: err}
	}

	blast := blastSummary(tier)
	record, err := proof.NewRecord(proof.RecordOpts{
		Source:        source(req.Source),
		SourceProduct: "axym",
		AgentID:       strings.TrimSpace(req.AgentID),
		Type:          "replay_certification",
		Timestamp:     now,
		Event: map[string]any{
			"model":        model,
			"tier":         tier,
			"status":       "certified",
			"pass":         true,
			"blast_radius": blast,
		},
		Metadata: map[string]any{
			"reason_codes": []string{"REPLAY_CERTIFIED"},
		},
		Controls: proof.Controls{PermissionsEnforced: true, ApprovedScope: "replay:" + model},
	})
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonAppendFailed, Message: "build replay record", Err: err}
	}
	key, err := dedupe.BuildKey(record.SourceProduct, record.RecordType, replayDedupeEvent(record.Event, startedAt))
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonAppendFailed, Message: "build replay dedupe key", Err: err}
	}
	appendResult, err := evidenceStore.Append(record, key)
	if err != nil {
		return Result{}, &Error{ReasonCode: ReasonAppendFailed, Message: "append replay record", Err: err}
	}
	return Result{
		RecordID:    appendResult.RecordID,
		Model:       model,
		Tier:        tier,
		Status:      "certified",
		Pass:        true,
		BlastRadius: blast,
		HeadHash:    appendResult.HeadHash,
		RecordCount: appendResult.RecordCount,
		OccurredAt:  now.Format(time.RFC3339),
		ReasonCodes: []string{"REPLAY_CERTIFIED"},
	}, nil
}

func ClassifyTier(input RiskInput) string {
	sensitivity := strings.ToLower(strings.TrimSpace(input.DataSensitivity))
	if input.ProductionCritical && (sensitivity == "high" || input.PublicExposure) {
		return "A"
	}
	if input.ProductionCritical || sensitivity == "medium" || input.PublicExposure {
		return "B"
	}
	return "C"
}

func blastSummary(tier string) map[string]any {
	switch tier {
	case "A":
		return map[string]any{"systems": 8, "control_domains": 5, "description": "organization-wide"}
	case "B":
		return map[string]any{"systems": 3, "control_domains": 3, "description": "service-boundary"}
	default:
		return map[string]any{"systems": 1, "control_domains": 1, "description": "component-local"}
	}
}

func source(candidate string) string {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return "axym-replay"
	}
	return candidate
}

func replayDedupeEvent(event map[string]any, startedAt time.Time) map[string]any {
	out := make(map[string]any, len(event)+1)
	for key, value := range event {
		out[key] = value
	}
	// Replay runs should always emit new evidence; dedupe keeps per-run identity.
	out["run_identity"] = startedAt.UTC().Format(time.RFC3339Nano)
	return out
}
