package collect

import (
	"context"
	"fmt"

	"github.com/Clyra-AI/axym/core/collector"
	"github.com/Clyra-AI/axym/core/normalize"
	"github.com/Clyra-AI/axym/core/policy/sink"
	"github.com/Clyra-AI/axym/core/proofemit"
	"github.com/Clyra-AI/axym/core/record"
	"github.com/Clyra-AI/axym/core/redact"
	"github.com/Clyra-AI/axym/core/store"
)

type Runner struct {
	Registry  *collector.Registry
	Store     *store.Store
	SinkMode  sink.Mode
	Redaction redact.Config
}

func (r *Runner) Run(ctx context.Context, req collector.Request, dryRun bool) (Result, error) {
	if r == nil || r.Registry == nil {
		return Result{}, &Error{ReasonCode: ReasonRuntime, Message: "collector registry is required", ExitCode: 1}
	}
	if !dryRun && r.Store == nil {
		return Result{}, &Error{ReasonCode: ReasonRuntime, Message: "store is required for collect writes", ExitCode: 1}
	}

	result := Result{DryRun: dryRun, Sources: []SourceSummary{}, ReasonCodes: []string{}}
	var emitter *proofemit.Emitter
	if !dryRun {
		emitter = &proofemit.Emitter{Store: r.Store, SinkMode: r.SinkMode}
	}

	for _, sourceCollector := range r.Registry.Ordered() {
		summary := SourceSummary{Name: sourceCollector.Name(), Status: "ok"}
		collectorResult, err := sourceCollector.Collect(ctx, req)
		if err != nil {
			reason := ReasonCollectorError
			if rc, ok := err.(reasonCoder); ok {
				reason = rc.ReasonCode()
			}
			summary.Status = "failed"
			summary.ReasonCodes = uniqueSorted([]string{reason})
			summary.Error = err.Error()
			result.Failures++
			result.ReasonCodes = append(result.ReasonCodes, reason)
			result.Sources = append(result.Sources, summary)
			continue
		}

		summary.WouldCapture = len(collectorResult.Candidates)
		result.WouldCapture += summary.WouldCapture
		summary.ReasonCodes = append(summary.ReasonCodes, collectorResult.ReasonCodes...)

		for _, candidate := range collectorResult.Candidates {
			input := normalize.Input{
				SourceType:    candidate.SourceType,
				Source:        candidate.Source,
				SourceProduct: candidate.SourceProduct,
				RecordType:    candidate.RecordType,
				AgentID:       candidate.AgentID,
				Timestamp:     candidate.Timestamp,
				Event:         candidate.Event,
				Metadata:      candidate.Metadata,
				Controls: normalize.Controls{
					PermissionsEnforced: candidate.Controls.PermissionsEnforced,
					ApprovedScope:       candidate.Controls.ApprovedScope,
				},
			}

			if dryRun {
				if _, err := record.NormalizeAndBuild(input, r.Redaction); err != nil {
					summary.Rejected++
					result.Rejected++
					reason := record.ReasonCode(err)
					if reason == "" {
						reason = ReasonMalformed
					}
					summary.ReasonCodes = append(summary.ReasonCodes, reason)
					result.ReasonCodes = append(result.ReasonCodes, reason)
					continue
				}
				summary.Captured++
				result.Captured++
				continue
			}

			emitResult, emitErr := emitter.Emit(proofemit.EmitInput{Normalized: input, Redaction: r.Redaction})
			if emitErr != nil {
				if sinkErr, ok := emitErr.(*proofemit.SinkFailureError); ok {
					return result, &Error{ReasonCode: sinkErr.ReasonCode, Message: sinkErr.Message, ExitCode: 1, Err: emitErr}
				}
				summary.Rejected++
				result.Rejected++
				reason := record.ReasonCode(emitErr)
				if reason == "" {
					reason = ReasonMalformed
				}
				summary.ReasonCodes = append(summary.ReasonCodes, reason)
				result.ReasonCodes = append(result.ReasonCodes, reason)
				continue
			}

			if emitResult.Degraded {
				result.Degraded = true
				if emitResult.ReasonCode != "" {
					summary.ReasonCodes = append(summary.ReasonCodes, emitResult.ReasonCode)
					result.ReasonCodes = append(result.ReasonCodes, emitResult.ReasonCode)
				}
			}
			if emitResult.Deduped {
				result.Deduped++
				summary.ReasonCodes = append(summary.ReasonCodes, "DEDUPED")
				continue
			}
			if !emitResult.Appended {
				summary.Rejected++
				result.Rejected++
				summary.ReasonCodes = append(summary.ReasonCodes, ReasonRuntime)
				result.ReasonCodes = append(result.ReasonCodes, ReasonRuntime)
				continue
			}

			summary.Captured++
			result.Captured++
			result.Appended++
			result.RecordCount = emitResult.RecordCount
			result.HeadHash = emitResult.HeadHash
		}

		summary.ReasonCodes = uniqueSorted(summary.ReasonCodes)
		if summary.Status == "ok" {
			switch {
			case summary.Captured == 0 && summary.WouldCapture == 0:
				summary.Status = "empty"
			case summary.Rejected > 0:
				summary.Status = "partial"
			}
		}
		result.Sources = append(result.Sources, summary)
	}

	result.ReasonCodes = uniqueSorted(result.ReasonCodes)
	for i := range result.Sources {
		result.Sources[i].ReasonCodes = uniqueSorted(result.Sources[i].ReasonCodes)
	}
	if dryRun && result.Appended != 0 {
		return Result{}, fmt.Errorf("dry-run must not append records")
	}
	return result, nil
}
