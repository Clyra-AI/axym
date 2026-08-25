package governance

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type OTLPOptions struct {
	MaxBytes          int
	MaxResourceGroups int
	MaxScopes         int
	MaxSpans          int
	MaxAttributes     int
	MaxString         int
	Now               time.Time
	MaxAge            time.Duration
	ContractID        string
}

const (
	defaultOTLPMaxBytes  = 10 << 20
	defaultOTLPMaxGroups = 100
	defaultOTLPMaxScopes = 500
	defaultOTLPMaxAttrs  = 100
	defaultOTLPMaxString = 4096
)

func IngestOTLP(raw []byte, opt OTLPOptions) (TelemetryResult, error) {
	if opt.MaxBytes == 0 {
		opt.MaxBytes = defaultOTLPMaxBytes
	}
	if len(raw) > opt.MaxBytes {
		return TelemetryResult{}, fmt.Errorf("%s", ReasonLimitExceeded)
	}
	var root struct {
		ResourceSpans []struct {
			Resource struct {
				Attributes []map[string]any `json:"attributes"`
			} `json:"resource"`
			ScopeSpans []struct {
				Scope map[string]any   `json:"scope"`
				Spans []map[string]any `json:"spans"`
			} `json:"scopeSpans"`
		} `json:"resourceSpans"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return TelemetryResult{}, fmt.Errorf("%s", ReasonMalformed)
	}
	if len(root.ResourceSpans) > limit(opt.MaxResourceGroups, defaultOTLPMaxGroups) {
		return TelemetryResult{}, fmt.Errorf("%s", ReasonLimitExceeded)
	}
	spans := []TraceSpan{}
	scopes := 0
	for _, rg := range root.ResourceSpans {
		resource := attrsMap(rg.Resource.Attributes)
		for _, sg := range rg.ScopeSpans {
			scopes++
			if scopes > limit(opt.MaxScopes, defaultOTLPMaxScopes) {
				return TelemetryResult{}, fmt.Errorf("%s", ReasonLimitExceeded)
			}
			scope := stringValue(sg.Scope["name"])
			for _, rawSpan := range sg.Spans {
				if len(spans) >= limit(opt.MaxSpans, MaxTelemetrySpans) {
					return TelemetryResult{}, fmt.Errorf("%s", ReasonLimitExceeded)
				}
				a := mergeAttrs(resource, attrsMap(rawSpan["attributes"]))
				trace := strings.ToLower(stringValue(rawSpan["traceId"]))
				sid := strings.ToLower(stringValue(rawSpan["spanId"]))
				if !traceIDPattern.MatchString(trace) || !spanIDPattern.MatchString(sid) {
					return TelemetryResult{}, fmt.Errorf("%s", ReasonMalformed)
				}
				start, ok1 := otlpTime(rawSpan["startTimeUnixNano"])
				if !ok1 {
					var e error
					start, e = time.Parse(time.RFC3339Nano, stringValue(rawSpan["startTime"]))
					ok1 = e == nil
				}
				end, ok2 := otlpTime(rawSpan["endTimeUnixNano"])
				if !ok2 {
					var e error
					end, e = time.Parse(time.RFC3339Nano, stringValue(rawSpan["endTime"]))
					ok2 = e == nil
				}
				if !ok1 || !ok2 || end.Before(start) {
					return TelemetryResult{}, fmt.Errorf("%s", ReasonMalformed)
				}
				s := TraceSpan{TraceID: trace, SpanID: sid, ParentSpanID: strings.ToLower(stringValue(rawSpan["parentSpanId"])), Name: stringValue(rawSpan["name"]), StartTime: start.UTC().Format(time.RFC3339Nano), EndTime: end.UTC().Format(time.RFC3339Nano), Attributes: a, Source: scope}
				s.Digest, _ = Digest(s)
				spans = append(spans, s)
			}
		}
	}
	seen := map[string]bool{}
	for _, s := range spans {
		k := s.TraceID + ":" + s.SpanID
		if seen[k] {
			return TelemetryResult{}, fmt.Errorf("%s", ReasonTampered)
		}
		seen[k] = true
	}
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].TraceID == spans[j].TraceID {
			return spans[i].SpanID < spans[j].SpanID
		}
		return spans[i].TraceID < spans[j].TraceID
	})
	out := TelemetryResult{Spans: spans, SourceDigests: []string{rawDigest(raw)}}
	if opt.ContractID != "" {
		for _, s := range spans {
			if id := s.Attributes["axym.contract.id"]; id == "" || id != opt.ContractID {
				out.ReasonCodes = append(out.ReasonCodes, ReasonOutOfScope)
			}
		}
	}
	if !opt.Now.IsZero() && opt.MaxAge > 0 {
		for _, s := range spans {
			et, _ := time.Parse(time.RFC3339Nano, s.EndTime)
			if opt.Now.Sub(et) > opt.MaxAge {
				out.ReasonCodes = append(out.ReasonCodes, ReasonStale)
			}
		}
	}
	sort.Strings(out.ReasonCodes)
	return out, nil
}
func limit(v, d int) int {
	if v > 0 {
		return v
	}
	return d
}
func stringValue(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case map[string]any:
		for _, k := range []string{"stringValue", "string_value", "value"} {
			if s, ok := x[k].(string); ok {
				return s
			}
		}
	}
	return ""
}
func attrsMap(v any) map[string]string {
	out := map[string]string{}
	switch arr := v.(type) {
	case []any:
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				addAttr(out, m)
			}
		}
	case []map[string]any:
		for _, m := range arr {
			addAttr(out, m)
		}
	}
	return out
}
func addAttr(out map[string]string, m map[string]any) {
	k := stringValue(m["key"])
	val := stringValue(m["value"])
	if k != "" {
		out[k] = val
	}
}
func mergeAttrs(a, b map[string]string) map[string]string {
	o := map[string]string{}
	for k, v := range a {
		o[k] = v
	}
	for k, v := range b {
		o[k] = v
	}
	return o
}
func otlpTime(v any) (time.Time, bool) {
	s := stringValue(v)
	if s == "" {
		return time.Time{}, false
	}
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return time.Time{}, false
		}
		n = n*10 + int64(c-'0')
	}
	return time.Unix(0, n), true
}
