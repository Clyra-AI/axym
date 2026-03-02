package stitch

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Clyra-AI/proof"
)

const ReasonChainSessionGap = "CHAIN_SESSION_GAP"

type Config struct {
	MaxGap time.Duration
}

type Gap struct {
	ReasonCode      string `json:"reason_code"`
	FromSession     string `json:"from_session"`
	ToSession       string `json:"to_session"`
	GapStart        string `json:"gap_start"`
	GapEnd          string `json:"gap_end"`
	MissingSeconds  int64  `json:"missing_seconds"`
	BoundaryTrigger string `json:"boundary_trigger"`
}

type Result struct {
	Intact bool  `json:"intact"`
	Gaps   []Gap `json:"gaps"`
}

func Analyze(records []proof.Record, cfg Config) Result {
	maxGap := cfg.MaxGap
	if maxGap <= 0 {
		maxGap = 30 * time.Minute
	}
	if len(records) < 2 {
		return Result{Intact: true, Gaps: []Gap{}}
	}

	gaps := make([]Gap, 0)
	for i := 1; i < len(records); i++ {
		prev := records[i-1]
		curr := records[i]
		prevTS := prev.Timestamp.UTC()
		currTS := curr.Timestamp.UTC()
		if !currTS.After(prevTS) {
			continue
		}
		delta := currTS.Sub(prevTS)
		if delta <= maxGap {
			continue
		}
		trigger := boundaryTrigger(prev, curr)
		gaps = append(gaps, Gap{
			ReasonCode:      ReasonChainSessionGap,
			FromSession:     sessionID(prev),
			ToSession:       sessionID(curr),
			GapStart:        prevTS.Format(time.RFC3339),
			GapEnd:          currTS.Format(time.RFC3339),
			MissingSeconds:  int64(delta.Seconds()),
			BoundaryTrigger: trigger,
		})
	}
	sort.Slice(gaps, func(i, j int) bool {
		if gaps[i].GapStart != gaps[j].GapStart {
			return gaps[i].GapStart < gaps[j].GapStart
		}
		if gaps[i].GapEnd != gaps[j].GapEnd {
			return gaps[i].GapEnd < gaps[j].GapEnd
		}
		return gaps[i].BoundaryTrigger < gaps[j].BoundaryTrigger
	})
	return Result{
		Intact: len(gaps) == 0,
		Gaps:   gaps,
	}
}

func boundaryTrigger(prev proof.Record, curr proof.Record) string {
	prevSession := sessionID(prev)
	currSession := sessionID(curr)
	if prevSession != "" && currSession != "" && prevSession != currSession {
		return "session_id_change"
	}
	if hasCheckpoint(prev) || hasCheckpoint(curr) {
		return "checkpoint_boundary"
	}
	return "timestamp_discontinuity"
}

func sessionID(record proof.Record) string {
	if fromMetadata := stringFromMap(record.Metadata, "session_id"); fromMetadata != "" {
		return fromMetadata
	}
	if fromEvent := stringFromMap(record.Event, "session_id"); fromEvent != "" {
		return fromEvent
	}
	return "unknown"
}

func hasCheckpoint(record proof.Record) bool {
	if boolFromMap(record.Event, "checkpoint") {
		return true
	}
	if stringFromMap(record.Event, "checkpoint_id") != "" {
		return true
	}
	return false
}

func stringFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	value, ok := m[key]
	if !ok {
		return ""
	}
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return fmt.Sprintf("%v", value)
}

func boolFromMap(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	value, ok := m[key]
	if !ok {
		return false
	}
	v, ok := value.(bool)
	return ok && v
}
