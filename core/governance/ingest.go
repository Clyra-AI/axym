package governance

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	proofsign "github.com/Clyra-AI/proof/signing"
	"io"
	"regexp"
	"sort"
	"time"
)

const (
	ReasonMalformed     = "GOVERNANCE_MALFORMED"
	ReasonTampered      = "GOVERNANCE_DIGEST_MISMATCH"
	ReasonStale         = "GOVERNANCE_STALE"
	ReasonOutOfScope    = "GOVERNANCE_OUT_OF_SCOPE"
	ReasonAdvisoryOnly  = "JUDGE_ADVISORY_ONLY"
	ReasonLimitExceeded = "GOVERNANCE_LIMIT_EXCEEDED"
)
const MaxTelemetrySpans = 10000

var traceIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)
var spanIDPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)

type TraceSpan struct {
	TraceID      string            `json:"trace_id"`
	SpanID       string            `json:"span_id"`
	ParentSpanID string            `json:"parent_span_id,omitempty"`
	Name         string            `json:"name"`
	StartTime    string            `json:"start_time"`
	EndTime      string            `json:"end_time"`
	Attributes   map[string]string `json:"attributes,omitempty"`
	Source       string            `json:"source"`
	Digest       string            `json:"digest"`
}
type BoundaryAttestation struct {
	SchemaID           string              `json:"schema_id"`
	SchemaVersion      string              `json:"schema_version"`
	ID                 string              `json:"id"`
	Boundary           string              `json:"boundary"`
	ContractRef        Ref                 `json:"contract_ref"`
	ObservedAt         string              `json:"observed_at"`
	FreshUntil         string              `json:"fresh_until"`
	Scope              string              `json:"scope"`
	Source             string              `json:"source"`
	Digest             string              `json:"digest"`
	Signature          proofsign.Signature `json:"signature"`
	Advisory           bool                `json:"advisory"`
	ExecutionAuthority bool                `json:"execution_authority"`
}
type TelemetryResult struct {
	Spans         []TraceSpan           `json:"spans"`
	Attestations  []BoundaryAttestation `json:"attestations"`
	SourceDigests []string              `json:"source_digests"`
	ReasonCodes   []string              `json:"reason_codes,omitempty"`
}
type JudgeEvidence struct {
	SchemaID           string              `json:"schema_id"`
	SchemaVersion      string              `json:"schema_version"`
	ID                 string              `json:"id"`
	ContractRef        Ref                 `json:"contract_ref"`
	Verdict            string              `json:"verdict"`
	ObservedAt         string              `json:"observed_at"`
	FreshUntil         string              `json:"fresh_until"`
	Explanation        string              `json:"explanation,omitempty"`
	Source             string              `json:"source"`
	Digest             string              `json:"digest"`
	ProviderVersion    string              `json:"provider_version"`
	Signature          proofsign.Signature `json:"signature"`
	Advisory           bool                `json:"advisory"`
	ExecutionAuthority bool                `json:"execution_authority"`
}
type JudgeProjection struct {
	EvidenceID         string   `json:"evidence_id"`
	Verdict            string   `json:"verdict"`
	Advisory           bool     `json:"advisory"`
	ExecutionAuthority bool     `json:"execution_authority"`
	ReasonCodes        []string `json:"reason_codes"`
	SourceDigest       string   `json:"source_digest"`
}

func parseOne(raw []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return fmt.Errorf("%s: %w", ReasonMalformed, err)
	}
	var tail any
	if err := d.Decode(&tail); err != io.EOF {
		return fmt.Errorf("%s: trailing input", ReasonMalformed)
	}
	return nil
}
func rawDigest(raw []byte) string {
	s := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(s[:])
}
func validDigest(d string) bool { return digestPattern.MatchString(d) }
func IngestTelemetry(raw []byte, now time.Time, maxAge time.Duration, contractID string) (TelemetryResult, error) {
	var doc struct {
		Spans        []TraceSpan           `json:"spans"`
		Attestations []BoundaryAttestation `json:"attestations"`
	}
	if err := parseOne(raw, &doc); err != nil {
		return TelemetryResult{}, err
	}
	if len(doc.Spans) > MaxTelemetrySpans {
		return TelemetryResult{}, fmt.Errorf("%s", ReasonLimitExceeded)
	}
	out := TelemetryResult{Spans: doc.Spans, Attestations: doc.Attestations, SourceDigests: []string{rawDigest(raw)}}
	for _, s := range out.Spans {
		if !traceIDPattern.MatchString(s.TraceID) || !spanIDPattern.MatchString(s.SpanID) || !validDigest(s.Digest) {
			out.ReasonCodes = append(out.ReasonCodes, ReasonMalformed)
			continue
		}
		st, e1 := time.Parse(time.RFC3339Nano, s.StartTime)
		et, e2 := time.Parse(time.RFC3339Nano, s.EndTime)
		if e1 != nil || e2 != nil || !et.After(st) {
			out.ReasonCodes = append(out.ReasonCodes, ReasonMalformed)
		}
		if !now.IsZero() && maxAge > 0 && now.Sub(et) > maxAge {
			out.ReasonCodes = append(out.ReasonCodes, ReasonStale)
		}
		if contractID != "" && s.Attributes["contract.id"] != "" && s.Attributes["contract.id"] != contractID {
			out.ReasonCodes = append(out.ReasonCodes, ReasonOutOfScope)
		}
	}
	sort.Strings(out.ReasonCodes)
	return out, nil
}
func ParseJudge(raw []byte) (JudgeEvidence, error) {
	var j JudgeEvidence
	if err := parseOne(raw, &j); err != nil {
		return j, err
	}
	return j, nil
}

func SignJudge(j JudgeEvidence, priv ed25519.PrivateKey) (JudgeEvidence, error) {
	j.Signature = proofsign.Signature{}
	j.Digest = ""
	d, e := Digest(j)
	if e != nil {
		return j, e
	}
	j.Digest = d
	s, e := proofsign.SignDigestHex(priv, d)
	if e != nil {
		return j, e
	}
	j.Signature = s
	return j, nil
}
func SignBoundary(a BoundaryAttestation, priv ed25519.PrivateKey) (BoundaryAttestation, error) {
	a.Signature = proofsign.Signature{}
	a.Digest = ""
	d, e := Digest(a)
	if e != nil {
		return a, e
	}
	a.Digest = d
	s, e := proofsign.SignDigestHex(priv, d)
	if e != nil {
		return a, e
	}
	a.Signature = s
	return a, nil
}
func VerifyBoundary(a BoundaryAttestation, pub ed25519.PublicKey, now time.Time, contractID string) error {
	if a.SchemaID == "" || a.SchemaVersion == "" || !a.Advisory || a.ExecutionAuthority || !validRef(a.ContractRef) || !validDigest(a.Digest) {
		return fmt.Errorf("%s", ReasonMalformed)
	}
	if contractID != "" && a.ContractRef.ID != contractID {
		return fmt.Errorf("%s", ReasonOutOfScope)
	}
	fresh, e := time.Parse(time.RFC3339Nano, a.FreshUntil)
	if e != nil || now.After(fresh) {
		return fmt.Errorf("%s", ReasonStale)
	}
	c := a
	c.Digest = ""
	c.Signature = proofsign.Signature{}
	d, _ := Digest(c)
	if d != a.Digest {
		return fmt.Errorf("%s", ReasonTampered)
	}
	ok, e := proofsign.VerifyDigestHex(pub, a.Signature)
	if e != nil || !ok {
		return fmt.Errorf("%s", ReasonTampered)
	}
	return nil
}
func VerifyJudge(j JudgeEvidence, pub ed25519.PublicKey, now time.Time, contractID string) error {
	if j.SchemaID == "" || j.SchemaVersion == "" || j.ProviderVersion == "" || !j.Advisory || j.ExecutionAuthority || !validRef(j.ContractRef) || !validDigest(j.Digest) {
		return fmt.Errorf("%s", ReasonMalformed)
	}
	if contractID != "" && j.ContractRef.ID != contractID {
		return fmt.Errorf("%s", ReasonOutOfScope)
	}
	fresh, e := time.Parse(time.RFC3339Nano, j.FreshUntil)
	if e != nil || now.After(fresh) {
		return fmt.Errorf("%s", ReasonStale)
	}
	c := j
	c.Digest = ""
	c.Signature = proofsign.Signature{}
	d, _ := Digest(c)
	if d != j.Digest {
		return fmt.Errorf("%s", ReasonTampered)
	}
	ok, e := proofsign.VerifyDigestHex(pub, j.Signature)
	if e != nil || !ok {
		return fmt.Errorf("%s", ReasonTampered)
	}
	return nil
}
func ProjectJudge(j JudgeEvidence, contractID string) (JudgeProjection, error) {
	if j.ID == "" || !validRef(j.ContractRef) || j.Verdict == "" || j.Source == "" || !validDigest(j.Digest) {
		return JudgeProjection{}, fmt.Errorf("%s", ReasonMalformed)
	}
	p := JudgeProjection{EvidenceID: j.ID, Verdict: j.Verdict, Advisory: true, ExecutionAuthority: false, ReasonCodes: []string{ReasonAdvisoryOnly}, SourceDigest: j.Digest}
	if contractID != "" && j.ContractRef.ID != contractID {
		p.ReasonCodes = append(p.ReasonCodes, ReasonOutOfScope)
	}
	return p, nil
}
