package context

import (
	"sort"
	"strings"

	"github.com/Clyra-AI/proof"
)

type EvidenceContext struct {
	DataClass       string `json:"data_class"`
	EndpointClass   string `json:"endpoint_class"`
	RiskLevel       string `json:"risk_level"`
	DiscoveryMethod string `json:"discovery_method"`
}

type Weights struct {
	DataClassWeight     int `json:"data_class_weight"`
	EndpointClassWeight int `json:"endpoint_class_weight"`
	RiskLevelWeight     int `json:"risk_level_weight"`
	DiscoveryWeight     int `json:"discovery_weight"`
	Total               int `json:"total"`
}

func Enrich(record proof.Record) EvidenceContext {
	ctx := EvidenceContext{
		DataClass:       firstNonEmpty(lowerString(record.Metadata["data_class"]), lowerString(record.Event["data_class"])),
		EndpointClass:   firstNonEmpty(lowerString(record.Event["endpoint_class"]), classifyEndpoint(record)),
		RiskLevel:       firstNonEmpty(lowerString(record.Metadata["risk_level"]), lowerString(record.Event["risk_level"]), classifyRisk(record)),
		DiscoveryMethod: firstNonEmpty(lowerString(record.Metadata["discovery_method"]), lowerString(record.Metadata["evidence_source"]), strings.ToLower(strings.TrimSpace(record.Source))),
	}
	if ctx.DataClass == "" {
		ctx.DataClass = classifyDataClass(record)
	}
	if ctx.EndpointClass == "" {
		ctx.EndpointClass = "unknown"
	}
	if ctx.RiskLevel == "" {
		ctx.RiskLevel = "unknown"
	}
	if ctx.DiscoveryMethod == "" {
		ctx.DiscoveryMethod = "unknown"
	}
	return ctx
}

func Score(ctx EvidenceContext) Weights {
	weights := Weights{
		DataClassWeight:     scoreDataClass(ctx.DataClass),
		EndpointClassWeight: scoreEndpointClass(ctx.EndpointClass),
		RiskLevelWeight:     scoreRiskLevel(ctx.RiskLevel),
		DiscoveryWeight:     scoreDiscovery(ctx.DiscoveryMethod),
	}
	weights.Total = weights.DataClassWeight + weights.EndpointClassWeight + weights.RiskLevelWeight + weights.DiscoveryWeight
	return weights
}

func SortContextsStable(in []EvidenceContext) {
	sort.SliceStable(in, func(i, j int) bool {
		if in[i].RiskLevel != in[j].RiskLevel {
			return in[i].RiskLevel < in[j].RiskLevel
		}
		if in[i].DataClass != in[j].DataClass {
			return in[i].DataClass < in[j].DataClass
		}
		if in[i].EndpointClass != in[j].EndpointClass {
			return in[i].EndpointClass < in[j].EndpointClass
		}
		return in[i].DiscoveryMethod < in[j].DiscoveryMethod
	})
}

func lowerString(value any) string {
	asString, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(asString))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func classifyDataClass(record proof.Record) string {
	recordType := strings.ToLower(strings.TrimSpace(record.RecordType))
	source := strings.ToLower(strings.TrimSpace(record.Source))
	switch {
	case strings.Contains(recordType, "risk"):
		return "restricted"
	case strings.Contains(recordType, "decision") || strings.Contains(source, "llm"):
		return "sensitive"
	case strings.Contains(recordType, "deployment"):
		return "internal"
	default:
		return "unknown"
	}
}

func classifyEndpoint(record proof.Record) string {
	tool := lowerString(record.Event["tool_name"])
	endpoint := lowerString(record.Event["endpoint"])
	value := firstNonEmpty(tool, endpoint)
	switch {
	case strings.Contains(value, "admin") || strings.Contains(value, "root"):
		return "admin"
	case strings.Contains(value, "write") || strings.Contains(value, "delete") || strings.Contains(value, "update"):
		return "write"
	case strings.Contains(value, "read") || strings.Contains(value, "get"):
		return "read"
	case value == "":
		return "unknown"
	default:
		return "execute"
	}
}

func classifyRisk(record proof.Record) string {
	recordType := strings.ToLower(strings.TrimSpace(record.RecordType))
	switch recordType {
	case "risk_assessment", "approval", "human_oversight":
		return "high"
	case "decision", "policy_enforcement", "permission_check":
		return "medium"
	case "tool_invocation", "deployment", "model_change", "data_pipeline_run":
		return "low"
	default:
		return "unknown"
	}
}

func scoreDataClass(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "restricted":
		return 4
	case "sensitive":
		return 3
	case "internal":
		return 2
	case "public":
		return 1
	default:
		return 0
	}
}

func scoreEndpointClass(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "admin":
		return 4
	case "write":
		return 3
	case "read", "execute":
		return 2
	case "unknown":
		return 1
	default:
		return 0
	}
}

func scoreRiskLevel(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func scoreDiscovery(value string) int {
	v := strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.Contains(v, "runtime") || strings.Contains(v, "mcp") || strings.Contains(v, "llm"):
		return 3
	case strings.Contains(v, "ci") || strings.Contains(v, "githubactions") || strings.Contains(v, "pipeline"):
		return 2
	case strings.Contains(v, "ingest") || strings.Contains(v, "wrkr") || strings.Contains(v, "gait"):
		return 1
	case v == "unknown" || v == "":
		return 0
	default:
		return 1
	}
}
