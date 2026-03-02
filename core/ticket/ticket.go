package ticket

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Clyra-AI/axym/core/store"
	"github.com/Clyra-AI/axym/core/store/dedupe"
	"github.com/Clyra-AI/axym/core/ticket/dlq"
	"github.com/Clyra-AI/proof"
)

const (
	ReasonInvalidInput = "TICKET_INVALID_INPUT"
	ReasonRateLimited  = "TICKET_RATE_LIMITED"
	ReasonRemoteError  = "TICKET_REMOTE_ERROR"
	ReasonRejected     = "TICKET_REJECTED"
	ReasonDLQ          = "TICKET_DLQ"
	ReasonAppendFailed = "TICKET_CHAIN_APPEND_FAILED"

	StatusAttached = "attached"
	StatusDLQ      = "dlq"
)

type Adapter interface {
	Name() string
	Attach(ctx context.Context, req AttachRequest) (Response, error)
}

type AttachRequest struct {
	ChangeID    string
	PayloadHash string
	Payload     map[string]any
}

type Response struct {
	StatusCode int
	Message    string
}

type Request struct {
	System      string
	ChangeID    string
	PayloadHash string
	Payload     map[string]any
	OpenedAt    time.Time
	SLA         time.Duration
	Source      string
	AgentID     string
}

type Result struct {
	System      string   `json:"system"`
	ChangeID    string   `json:"change_id"`
	Status      string   `json:"status"`
	Attempts    int      `json:"attempts"`
	ReasonCodes []string `json:"reason_codes"`
	DLQPath     string   `json:"dlq_path,omitempty"`
	SLAWithin   bool     `json:"sla_within"`
	OccurredAt  string   `json:"occurred_at"`
	RecordID    string   `json:"record_id,omitempty"`
}

type Processor struct {
	Adapter     Adapter
	Store       *store.Store
	DLQ         *dlq.Queue
	Clock       func() time.Time
	MaxAttempts int
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

func (p Processor) Process(ctx context.Context, req Request) (Result, error) {
	if p.Adapter == nil {
		return Result{}, &Error{ReasonCode: ReasonInvalidInput, Message: "adapter is required"}
	}
	if p.Store == nil {
		return Result{}, &Error{ReasonCode: ReasonInvalidInput, Message: "store is required"}
	}
	if strings.TrimSpace(req.ChangeID) == "" {
		return Result{}, &Error{ReasonCode: ReasonInvalidInput, Message: "change_id is required"}
	}
	system := strings.TrimSpace(req.System)
	if system == "" {
		system = p.Adapter.Name()
	}
	if system == "" {
		system = "ticket"
	}
	maxAttempts := p.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	clock := p.Clock
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC().Truncate(time.Second) }
	}
	now := clock().UTC().Truncate(time.Second)
	reasons := make([]string, 0)

	status := StatusDLQ
	attempts := 0
	dlqPath := ""
	for attempts = 1; attempts <= maxAttempts; attempts++ {
		select {
		case <-ctx.Done():
			return Result{}, &Error{ReasonCode: ReasonRemoteError, Message: "context canceled", Err: ctx.Err()}
		default:
		}

		resp, err := p.Adapter.Attach(ctx, AttachRequest{
			ChangeID:    req.ChangeID,
			PayloadHash: req.PayloadHash,
			Payload:     req.Payload,
		})
		if err != nil {
			reasons = append(reasons, ReasonRemoteError)
			if attempts < maxAttempts {
				continue
			}
			break
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			status = StatusAttached
			break
		}
		if resp.StatusCode == 429 {
			reasons = append(reasons, ReasonRateLimited)
			if attempts < maxAttempts {
				continue
			}
			break
		}
		if resp.StatusCode >= 500 {
			reasons = append(reasons, ReasonRemoteError)
			if attempts < maxAttempts {
				continue
			}
			break
		}
		reasons = append(reasons, ReasonRejected)
		break
	}

	slaWithin := evaluateSLA(now, req.OpenedAt, req.SLA)
	if status != StatusAttached {
		reasons = append(reasons, ReasonDLQ)
		if p.DLQ != nil {
			var err error
			dlqPath, err = p.DLQ.Enqueue(dlq.Entry{
				System:      system,
				ChangeID:    req.ChangeID,
				PayloadHash: req.PayloadHash,
				ReasonCodes: uniqueSorted(reasons),
				Attempts:    attempts,
				OccurredAt:  now.Format(time.RFC3339),
			})
			if err != nil {
				return Result{}, &Error{ReasonCode: ReasonDLQ, Message: "persist dlq entry", Err: err}
			}
		}
	}

	result := Result{
		System:      system,
		ChangeID:    req.ChangeID,
		Status:      status,
		Attempts:    attempts,
		ReasonCodes: uniqueSorted(reasons),
		DLQPath:     dlqPath,
		SLAWithin:   slaWithin,
		OccurredAt:  now.Format(time.RFC3339),
	}
	recordID, err := p.emitResultRecord(now, req, result)
	if err != nil {
		return Result{}, err
	}
	result.RecordID = recordID
	return result, nil
}

func (p Processor) emitResultRecord(now time.Time, req Request, result Result) (string, error) {
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "axym-ticket-bridge"
	}
	event := map[string]any{
		"kind":         "ticket_attachment",
		"system":       result.System,
		"change_id":    result.ChangeID,
		"status":       result.Status,
		"attempts":     result.Attempts,
		"sla_within":   result.SLAWithin,
		"reason_codes": append([]string(nil), result.ReasonCodes...),
	}
	if result.DLQPath != "" {
		event["dlq_path"] = result.DLQPath
	}
	metadata := map[string]any{
		"payload_hash": req.PayloadHash,
		"occurred_at":  result.OccurredAt,
		"sla_within":   result.SLAWithin,
	}
	record, err := proof.NewRecord(proof.RecordOpts{
		Source:        source,
		SourceProduct: "axym",
		AgentID:       strings.TrimSpace(req.AgentID),
		Type:          "policy_enforcement",
		Timestamp:     now,
		Event:         event,
		Metadata:      metadata,
		Controls:      proof.Controls{PermissionsEnforced: true, ApprovedScope: "ticket:" + result.ChangeID},
	})
	if err != nil {
		return "", &Error{ReasonCode: ReasonAppendFailed, Message: "build ticket evidence record", Err: err}
	}
	key, err := dedupe.BuildKey(record.SourceProduct, record.RecordType, record.Event)
	if err != nil {
		return "", &Error{ReasonCode: ReasonAppendFailed, Message: "build dedupe key", Err: err}
	}
	appendResult, err := p.Store.Append(record, key)
	if err != nil {
		return "", &Error{ReasonCode: ReasonAppendFailed, Message: "append ticket evidence record", Err: err}
	}
	if appendResult.Deduped {
		return record.RecordID, nil
	}
	return appendResult.RecordID, nil
}

func evaluateSLA(now time.Time, openedAt time.Time, sla time.Duration) bool {
	if openedAt.IsZero() || sla <= 0 {
		return true
	}
	return !now.After(openedAt.UTC().Add(sla))
}

func uniqueSorted(in []string) []string {
	set := map[string]struct{}{}
	for _, candidate := range in {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for candidate := range set {
		out = append(out, candidate)
	}
	sort.Strings(out)
	return out
}

func IsReason(err error, reasonCode string) bool {
	var ticketErr *Error
	if !errors.As(err, &ticketErr) {
		return false
	}
	return ticketErr.ReasonCode == reasonCode
}
