package main

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Clyra-AI/axym/core/governance"
	"github.com/Clyra-AI/axym/core/store"
	"github.com/Clyra-AI/axym/core/verifysupport"
	"github.com/spf13/cobra"
)

func newGovernanceCmd(stdout io.Writer, stderr io.Writer, global *globalFlags) *cobra.Command {
	var input, kind, contractID, trustedKey string
	var verifyAt string
	var maxAge time.Duration
	cmd := &cobra.Command{Use: "governance", Short: "Project governed action evidence (read-only)", SilenceUsage: true}
	ingest := &cobra.Command{Use: "ingest", Short: "Ingest OTLP/boundary or advisory Judge JSON", RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(input) == "" || strings.TrimSpace(kind) == "" {
			return emitGovernanceInvalid("--input and --kind are required", stdout, stderr, global)
		}
		// #nosec G304 -- input path is explicit operator input.
		raw, err := os.ReadFile(input)
		if err != nil {
			return emitGovernanceInvalid(err.Error(), stdout, stderr, global)
		}
		var data any
		switch strings.ToLower(kind) {
		case "otlp":
			result, e := governance.IngestOTLP(raw, governance.OTLPOptions{Now: time.Now().UTC(), MaxAge: maxAge, ContractID: contractID})
			if e != nil {
				return emitGovernanceError(e, stdout, stderr, global)
			}
			data = result
		case "boundary":
			var a governance.BoundaryAttestation
			if e := json.Unmarshal(raw, &a); e != nil {
				return emitGovernanceInvalid(e.Error(), stdout, stderr, global)
			}
			pub, e := loadGovernanceKey(trustedKey)
			if e != nil {
				return emitGovernanceInvalid("--trusted-key is required", stdout, stderr, global)
			}
			vt, e := parseGovernanceTime(verifyAt)
			if e != nil {
				return emitGovernanceInvalid(e.Error(), stdout, stderr, global)
			}
			if e = governance.VerifyBoundary(a, pub, vt, contractID); e != nil {
				return emitGovernanceVerifyError(e, stdout, stderr, global)
			}
			data = a
		case "telemetry":
			result, e := governance.IngestTelemetry(raw, time.Now().UTC(), maxAge, contractID)
			if e != nil {
				return emitGovernanceError(e, stdout, stderr, global)
			}
			data = result
		case "judge":
			evidence, e := governance.ParseJudge(raw)
			if e != nil {
				return emitGovernanceError(e, stdout, stderr, global)
			}
			pub, e := loadGovernanceKey(trustedKey)
			if e != nil {
				return emitGovernanceInvalid("--trusted-key is required and must be a public-key artifact", stdout, stderr, global)
			}
			vt, e := parseGovernanceTime(verifyAt)
			if e != nil {
				return emitGovernanceInvalid(e.Error(), stdout, stderr, global)
			}
			if e = governance.VerifyJudge(evidence, pub, vt, contractID); e != nil {
				return emitGovernanceVerifyError(e, stdout, stderr, global)
			}
			result, e := governance.ProjectJudge(evidence, contractID)
			if e != nil {
				return emitGovernanceError(e, stdout, stderr, global)
			}
			data = result
		default:
			return emitGovernanceInvalid("--kind must be one of: otlp, telemetry, boundary, judge", stdout, stderr, global)
		}
		if global.JSON {
			return printJSON(stdout, envelope{OK: true, Command: "governance ingest", Data: data})
		}
		if !global.Quiet {
			printText(stdout, fmt.Sprintf("governance ingest %s: advisory/read-only projection", kind), false)
		}
		return nil
	}}
	ingest.Flags().StringVar(&input, "input", "", "JSON input path")
	ingest.Flags().StringVar(&kind, "kind", "", "Input kind (otlp|telemetry|boundary|judge)")
	ingest.Flags().StringVar(&contractID, "contract-id", "", "Expected Action Contract ID")
	ingest.Flags().DurationVar(&maxAge, "max-age", 24*time.Hour, "Maximum telemetry age")
	ingest.Flags().StringVar(&trustedKey, "trusted-key", "", "Trusted public-key artifact for Judge/boundary verification")
	ingest.Flags().StringVar(&verifyAt, "verify-at", time.Now().UTC().Format(time.RFC3339Nano), "Verification time (RFC3339)")
	cmd.AddCommand(ingest)
	var emitStore string
	emit := &cobra.Command{Use: "emit", Short: "Append a verified governance projection to the local chain", RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(input) == "" || strings.TrimSpace(kind) == "" || strings.TrimSpace(emitStore) == "" {
			return emitGovernanceInvalid("--input, --kind, and --store-dir are required", stdout, stderr, global)
		}
		// #nosec G304 -- input path is explicit operator input.
		raw, err := os.ReadFile(input)
		if err != nil {
			return emitGovernanceInvalid(err.Error(), stdout, stderr, global)
		}
		st, err := store.New(store.Config{RootDir: emitStore})
		if err != nil {
			return emitGovernanceError(err, stdout, stderr, global)
		}
		var results []any
		if strings.ToLower(kind) == "boundary" {
			var a governance.BoundaryAttestation
			if e := json.Unmarshal(raw, &a); e != nil {
				return emitGovernanceError(e, stdout, stderr, global)
			}
			pub, e := loadGovernanceKey(trustedKey)
			if e != nil {
				return emitGovernanceInvalid("--trusted-key is required", stdout, stderr, global)
			}
			vt, _ := time.Parse(time.RFC3339Nano, verifyAt)
			if e = governance.VerifyBoundary(a, pub, vt, contractID); e != nil {
				return emitGovernanceVerifyError(e, stdout, stderr, global)
			}
			tm, _ := time.Parse(time.RFC3339Nano, a.ObservedAt)
			rec, e := governance.ToProofRecord("policy_enforcement", a.Source, a.Source, a.ID, tm, map[string]any{"boundary": a}, []governance.Ref{a.ContractRef})
			if e != nil {
				return emitGovernanceError(e, stdout, stderr, global)
			}
			ar, e := governance.AppendProjection(st, rec)
			if e != nil {
				return emitGovernanceError(e, stdout, stderr, global)
			}
			results = append(results, ar)
		} else if strings.ToLower(kind) == "judge" {
			j, e := governance.ParseJudge(raw)
			if e != nil {
				return emitGovernanceError(e, stdout, stderr, global)
			}
			pub, e := loadGovernanceKey(trustedKey)
			if e != nil {
				return emitGovernanceInvalid("--trusted-key is required", stdout, stderr, global)
			}
			vt, _ := time.Parse(time.RFC3339Nano, verifyAt)
			if e = governance.VerifyJudge(j, pub, vt, contractID); e != nil {
				return emitGovernanceVerifyError(e, stdout, stderr, global)
			}
			p, e := governance.ProjectJudge(j, contractID)
			if e != nil {
				return emitGovernanceError(e, stdout, stderr, global)
			}
			tm, _ := time.Parse(time.RFC3339Nano, j.ObservedAt)
			rec, e := governance.ToProofRecord("test_result", "judge", "judge", p.EvidenceID, tm, map[string]any{"judge": p}, []governance.Ref{j.ContractRef})
			if e != nil {
				return emitGovernanceError(e, stdout, stderr, global)
			}
			a, e := governance.AppendProjection(st, rec)
			if e != nil {
				return emitGovernanceError(e, stdout, stderr, global)
			}
			results = append(results, a)
		} else {
			if strings.ToLower(kind) != "telemetry" && strings.ToLower(kind) != "otlp" {
				return emitGovernanceInvalid("--kind must be one of: otlp, telemetry, boundary, judge", stdout, stderr, global)
			}
			var t governance.TelemetryResult
			var e error
			if strings.ToLower(kind) == "otlp" {
				t, e = governance.IngestOTLP(raw, governance.OTLPOptions{Now: time.Now().UTC(), MaxAge: maxAge, ContractID: contractID})
			} else {
				t, e = governance.IngestTelemetry(raw, time.Now().UTC(), maxAge, contractID)
			}
			if e != nil {
				return emitGovernanceError(e, stdout, stderr, global)
			}
			if len(t.ReasonCodes) > 0 {
				return emitGovernanceError(fmt.Errorf("%s", strings.Join(t.ReasonCodes, ",")), stdout, stderr, global)
			}
			for _, s := range t.Spans {
				originalDigest := s.Digest
				tm, e := time.Parse(time.RFC3339Nano, s.StartTime)
				if e != nil {
					return emitGovernanceError(e, stdout, stderr, global)
				}
				s = governance.RedactTelemetrySpan(s)
				s.Digest = ""
				redactedDigest, e := governance.Digest(s)
				if e != nil {
					return emitGovernanceError(e, stdout, stderr, global)
				}
				s.Digest = redactedDigest
				rec, e := governance.ToProofRecord("tool_invocation", s.Source, "telemetry", s.SpanID, tm, map[string]any{"trace": s, "source_digest": originalDigest}, []governance.Ref{{Kind: "evidence", ID: s.SpanID, Digest: originalDigest, Source: s.Source, SourceProduct: "telemetry", SchemaID: "https://axym.dev/schemas/v1/governance/trace-span.schema.json", SchemaVersion: "v1"}})
				if e != nil {
					return emitGovernanceError(e, stdout, stderr, global)
				}
				a, e := governance.AppendProjection(st, rec)
				if e != nil {
					return emitGovernanceError(e, stdout, stderr, global)
				}
				results = append(results, a)
			}
		}
		data := map[string]any{"appended": results}
		if global.JSON {
			return printJSON(stdout, envelope{OK: true, Command: "governance emit", Data: data})
		}
		printText(stdout, fmt.Sprintf("governance emit: %d projection(s)", len(results)), global.Quiet)
		return nil
	}}
	emit.Flags().StringVar(&input, "input", "", "JSON input path")
	emit.Flags().StringVar(&kind, "kind", "", "Input kind (otlp|telemetry|boundary|judge)")
	emit.Flags().StringVar(&contractID, "contract-id", "", "Expected Action Contract ID")
	emit.Flags().DurationVar(&maxAge, "max-age", 24*time.Hour, "Maximum telemetry age")
	emit.Flags().StringVar(&emitStore, "store-dir", "", "Explicit local chain store (required for writes)")
	emit.Flags().StringVar(&trustedKey, "trusted-key", "", "Trusted public-key artifact for Judge/boundary verification")
	emit.Flags().StringVar(&verifyAt, "verify-at", time.Now().UTC().Format(time.RFC3339Nano), "Verification time (RFC3339)")
	cmd.AddCommand(emit)
	return cmd
}

func loadGovernanceKey(path string) (ed25519.PublicKey, error) {
	// #nosec G304 -- trusted-key path is explicit operator input.
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pk, err := verifysupport.UnmarshalBundlePublicKey(raw)
	if err != nil {
		return nil, err
	}
	return ed25519.PublicKey(pk.Public), nil
}
func parseGovernanceTime(v string) (time.Time, error) {
	t, e := time.Parse(time.RFC3339Nano, v)
	if e != nil {
		return time.Time{}, fmt.Errorf("verify-at must be RFC3339Nano: %w", e)
	}
	return t, nil
}

func emitGovernanceInvalid(message string, stdout, stderr io.Writer, global *globalFlags) error {
	if global.JSON {
		_ = printJSON(stdout, envelope{OK: false, Command: "governance ingest", Error: &errorEnvelope{Reason: "invalid_input", Message: message}})
	} else if !global.Quiet {
		_, _ = fmt.Fprintln(stderr, message)
	}
	return &cliError{code: exitInvalidInput, msg: message}
}
func emitGovernanceError(err error, stdout, stderr io.Writer, global *globalFlags) error {
	reason := governance.ReasonMalformed
	if strings.Contains(err.Error(), governance.ReasonTampered) {
		reason = governance.ReasonTampered
	}
	if global.JSON {
		_ = printJSON(stdout, envelope{OK: false, Command: "governance ingest", Error: &errorEnvelope{Reason: reason, Message: err.Error()}})
	} else if !global.Quiet {
		_, _ = fmt.Fprintln(stderr, err)
	}
	return &cliError{code: exitInvalidInput, msg: err.Error()}
}
func emitGovernanceVerifyError(err error, stdout, stderr io.Writer, global *globalFlags) error {
	if global.JSON {
		_ = printJSON(stdout, envelope{OK: false, Command: "governance ingest", Error: &errorEnvelope{Reason: "verification_failed", Message: err.Error()}})
	} else if !global.Quiet {
		_, _ = fmt.Fprintln(stderr, err)
	}
	return &cliError{code: exitVerificationFailed, msg: err.Error()}
}
