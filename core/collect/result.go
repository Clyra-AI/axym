package collect

type SourceSummary struct {
	Name         string   `json:"name"`
	Status       string   `json:"status"`
	WouldCapture int      `json:"would_capture"`
	Captured     int      `json:"captured"`
	Rejected     int      `json:"rejected"`
	ReasonCodes  []string `json:"reason_codes,omitempty"`
	Error        string   `json:"error,omitempty"`
}

type Result struct {
	DryRun       bool            `json:"dry_run"`
	Sources      []SourceSummary `json:"sources"`
	WouldCapture int             `json:"would_capture"`
	Captured     int             `json:"captured"`
	Rejected     int             `json:"rejected"`
	Appended     int             `json:"appended"`
	Deduped      int             `json:"deduped"`
	RecordCount  int             `json:"record_count,omitempty"`
	HeadHash     string          `json:"head_hash,omitempty"`
	Degraded     bool            `json:"degraded"`
	Failures     int             `json:"failures"`
	ReasonCodes  []string        `json:"reason_codes"`
}
