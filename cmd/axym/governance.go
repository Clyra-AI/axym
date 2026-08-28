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
	"github.com/Clyra-AI/axym/core/ingest/actioncontract"
	"github.com/Clyra-AI/axym/core/store"
	coreverify "github.com/Clyra-AI/axym/core/verify"
	"github.com/Clyra-AI/axym/core/verifysupport"
	"github.com/Clyra-AI/proof"
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
		var telemetrySummary *governance.TelemetryResult
		if strings.ToLower(kind) == "boundary" {
			var a governance.BoundaryAttestation
			if e := json.Unmarshal(raw, &a); e != nil {
				return emitGovernanceError(e, stdout, stderr, global)
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
			vt, e := parseGovernanceTime(verifyAt)
			if e != nil {
				return emitGovernanceInvalid(e.Error(), stdout, stderr, global)
			}
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
				return emitGovernanceVerifyError(e, stdout, stderr, global)
			}
			if len(t.ReasonCodes) > 0 {
				return emitGovernanceVerifyError(fmt.Errorf("%s", strings.Join(t.ReasonCodes, ",")), stdout, stderr, global)
			}
			redactedSummary := t
			redactedSummary.Spans = make([]governance.TraceSpan, len(t.Spans))
			for i, span := range t.Spans {
				redacted := governance.RedactTelemetrySpan(span)
				redacted.Digest = ""
				redactedDigest, digestErr := governance.Digest(redacted)
				if digestErr != nil {
					return emitGovernanceError(digestErr, stdout, stderr, global)
				}
				redacted.Digest = redactedDigest
				redactedSummary.Spans[i] = redacted
			}
			telemetrySummary = &redactedSummary
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
		if telemetrySummary != nil {
			data["telemetry"] = telemetrySummary
		}
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
	addGovernanceProjectionCommands(cmd, stdout, stderr, global)
	return cmd
}

func addGovernanceProjectionCommands(parent *cobra.Command, stdout, stderr io.Writer, global *globalFlags) {
	newProjection := func(name, short string, graph bool) *cobra.Command {
		var storeDir, contractID, format string
		projection := &cobra.Command{Use: name, Short: short, RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(storeDir) == "" || strings.TrimSpace(contractID) == "" {
				return emitGovernanceInvalid("--store-dir and --contract-id are required", stdout, stderr, global)
			}
			st, err := store.OpenReadOnly(store.Config{RootDir: storeDir})
			if err != nil {
				return emitGovernanceError(err, stdout, stderr, global)
			}
			chain, err := st.LoadChain()
			if err != nil {
				return emitGovernanceError(err, stdout, stderr, global)
			}
			publicKey := proof.PublicKey{}
			if len(chain.Records) > 0 {
				if _, verifyErr := coreverify.VerifyChainFromStoreDir(storeDir); verifyErr != nil {
					return emitGovernanceVerifyError(verifyErr, stdout, stderr, global)
				}
				publicKey, err = verifysupport.LoadStorePublicKey(storeDir)
				if err != nil {
					return emitGovernanceVerifyError(err, stdout, stderr, global)
				}
			}
			native, err := actioncontract.LoadStored(storeDir)
			if err != nil {
				return emitGovernanceError(err, stdout, stderr, global)
			}
			proposals := make([]actioncontract.Proposal, 0)
			for _, item := range native {
				if item.Envelope.ArtifactType != "proposal" {
					continue
				}
				p, parseErr := actioncontract.ParseProposal(item.Raw)
				if parseErr != nil {
					return emitGovernanceError(parseErr, stdout, stderr, global)
				}
				validation := actioncontract.ValidateProposal(p, actioncontract.ValidationOptions{SkipTemporalValidation: true})
				if !actioncontract.AcceptableSemanticProposal(validation) {
					return emitGovernanceVerifyError(&actioncontract.ValidationError{Reasons: validation.ReasonCodes}, stdout, stderr, global)
				}
				proposals = append(proposals, p)
			}
			_, packets, err := governance.RegisterAndPacketsVerifiedWithRegistry(proposals, chain.Records, publicKey, storeDir)
			if err != nil {
				return emitGovernanceVerifyError(err, stdout, stderr, global)
			}
			packet, ok := packets[contractID]
			if !ok {
				return emitGovernanceInvalid("contract has no stored governed packet", stdout, stderr, global)
			}
			packet = governance.RedactPacket(packet)
			events := governanceProjectionEvents(packet)
			timeline, timelineErr := governance.ProjectTimeline(contractID, events)
			if timelineErr != nil {
				timeline = governance.Timeline{ContractID: contractID, Events: events, State: governance.State{ContractID: contractID, Status: "gap", Events: governanceProjectionEventIDs(events), SourceDigests: governanceProjectionEventDigests(events), ReasonCodes: []string{"LIFECYCLE_ILLEGAL_TRANSITION"}}}
			}
			if graph {
				value, graphErr := governance.ProjectGraph(timeline)
				if graphErr != nil {
					return emitGovernanceError(graphErr, stdout, stderr, global)
				}
				if strings.EqualFold(format, "markdown") {
					printText(stdout, governanceGraphMarkdown(value), global.Quiet)
					return nil
				}
				return printJSON(stdout, envelope{OK: true, Command: "governance graph", Data: value})
			}
			if strings.EqualFold(format, "markdown") {
				printText(stdout, governanceTimelineMarkdown(timeline), global.Quiet)
				return nil
			}
			return printJSON(stdout, envelope{OK: true, Command: "governance " + name, Data: timeline})
		}}
		projection.Flags().StringVar(&storeDir, "store-dir", ".axym", "Path to local chain store")
		projection.Flags().StringVar(&contractID, "contract-id", "", "Action Contract ID")
		projection.Flags().StringVar(&format, "format", "json", "Output format (json|markdown)")
		return projection
	}
	parent.AddCommand(newProjection("timeline", "Project deterministic governed lifecycle timeline", false))
	parent.AddCommand(newProjection("graph", "Project deterministic governed lifecycle graph", true))
	parent.AddCommand(newProjection("replay", "Replay governed lifecycle events with the governance reducer", false))
}

func governanceProjectionEvents(packet governance.Packet) []governance.Event {
	return governance.EventsFromPacket(packet)
}
func governanceProjectionEventIDs(events []governance.Event) []string {
	out := make([]string, 0, len(events))
	for _, item := range events {
		out = append(out, item.ID)
	}
	return out
}
func governanceProjectionEventDigests(events []governance.Event) []string {
	out := make([]string, 0, len(events))
	for _, item := range events {
		if item.SourceDigest != "" {
			out = append(out, item.SourceDigest)
		}
	}
	return out
}
func governanceTimelineMarkdown(t governance.Timeline) string {
	return fmt.Sprintf("# Governed Action Contract Timeline\n\n- Contract: %s\n- Status: %s\n- Complete: %t\n- Reasons: %s\n- Events: %d\n", t.ContractID, t.State.Status, t.State.Complete, strings.Join(t.State.ReasonCodes, ", "), len(t.Events))
}
func governanceGraphMarkdown(g governance.Graph) string {
	return fmt.Sprintf("# Governed Action Contract Graph\n\n- Nodes: %d\n- Edges: %d\n- Replay status: %s\n", len(g.Nodes), len(g.Edges), g.ReplayState.Status)
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
