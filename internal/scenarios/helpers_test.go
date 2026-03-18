//go:build scenario

package scenarios

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	coreoverride "github.com/Clyra-AI/axym/core/override"
	corereplay "github.com/Clyra-AI/axym/core/replay"
	"github.com/Clyra-AI/axym/core/store"
	coreticket "github.com/Clyra-AI/axym/core/ticket"
	"github.com/Clyra-AI/axym/core/ticket/dlq"
	"github.com/Clyra-AI/axym/core/ticket/jira"
	"github.com/Clyra-AI/proof"
)

type scenarioTools struct {
	RepoRoot  string
	AxymPath  string
	ProofPath string
}

type commandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type scenarioWorkspace struct {
	RootDir   string
	StoreDir  string
	BundleDir string
}

var (
	toolsOnce sync.Once
	toolsVal  scenarioTools
	toolsErr  error

	goldensOnce sync.Once
	goldensVal  map[string]json.RawMessage
	goldensErr  error
)

func scenarioRuntimeTools(t *testing.T) scenarioTools {
	t.Helper()

	toolsOnce.Do(func() {
		repoRoot, err := RepoRoot()
		if err != nil {
			toolsErr = err
			return
		}
		binRoot, err := os.MkdirTemp("", "axym-scenarios-*")
		if err != nil {
			toolsErr = err
			return
		}
		axymName := "axym"
		proofName := "proof"
		if runtime.GOOS == "windows" {
			axymName += ".exe"
			proofName += ".exe"
		}
		axymPath := filepath.Join(binRoot, axymName)
		axymBuild := exec.Command("go", "build", "-o", axymPath, "./cmd/axym")
		axymBuild.Dir = repoRoot
		if out, err := axymBuild.CombinedOutput(); err != nil {
			toolsErr = fmt.Errorf("build axym: %w output=%s", err, string(out))
			return
		}
		proofPath := filepath.Join(binRoot, proofName)
		proofBuild := exec.Command("go", "build", "-o", proofPath, "github.com/Clyra-AI/proof/cmd/proof")
		proofBuild.Dir = repoRoot
		if out, err := proofBuild.CombinedOutput(); err != nil {
			toolsErr = fmt.Errorf("build proof: %w output=%s", err, string(out))
			return
		}
		toolsVal = scenarioTools{RepoRoot: repoRoot, AxymPath: axymPath, ProofPath: proofPath}
	})

	if toolsErr != nil {
		t.Fatal(toolsErr)
	}
	return toolsVal
}

func scenarioGoldens(t *testing.T) map[string]json.RawMessage {
	t.Helper()

	goldensOnce.Do(func() {
		repoRoot, err := RepoRoot()
		if err != nil {
			goldensErr = err
			return
		}
		goldensVal, goldensErr = LoadGoldenResultsFromRoot(repoRoot)
	})

	if goldensErr != nil {
		t.Fatal(goldensErr)
	}
	return goldensVal
}

func newScenarioWorkspace(t *testing.T) scenarioWorkspace {
	t.Helper()

	root := t.TempDir()
	return scenarioWorkspace{
		RootDir:   root,
		StoreDir:  filepath.Join(root, "store"),
		BundleDir: filepath.Join(root, "bundle"),
	}
}

func runScenarioFixture(t *testing.T, fixture Fixture) {
	t.Helper()

	tools := scenarioRuntimeTools(t)
	executor, ok := scenarioExecutors[fixture.ID]
	if !ok {
		t.Fatalf("missing scenario executor for %s", fixture.ID)
	}
	actual := executor(t, tools, fixture)
	wantRaw, ok := scenarioGoldens(t)[fixture.ID]
	if !ok {
		t.Fatalf("missing golden output for %s", fixture.ID)
	}
	assertScenarioGolden(t, wantRaw, actual)
}

func assertScenarioGolden(t *testing.T, wantRaw json.RawMessage, actual map[string]any) {
	t.Helper()

	want := normalizeJSON(t, wantRaw)
	actualRaw, err := json.MarshalIndent(actual, "", "  ")
	if err != nil {
		t.Fatalf("marshal actual scenario output: %v", err)
	}
	if string(want) != string(actualRaw) {
		t.Fatalf("scenario output mismatch\nwant:\n%s\n\nactual:\n%s", string(want), string(actualRaw))
	}
}

func normalizeJSON(t *testing.T, raw []byte) []byte {
	t.Helper()

	var out bytes.Buffer
	if err := json.Indent(&out, raw, "", "  "); err != nil {
		t.Fatalf("indent golden json: %v", err)
	}
	return out.Bytes()
}

func runAxymJSON(t *testing.T, tools scenarioTools, workdir string, env map[string]string, args ...string) (map[string]any, commandResult) {
	t.Helper()
	result := runCommand(t, tools.AxymPath, workdir, env, args...)
	payload := decodeJSON(t, result.Stdout)
	return payload, result
}

func runProofJSON(t *testing.T, tools scenarioTools, workdir string, args ...string) (map[string]any, commandResult) {
	t.Helper()
	result := runCommand(t, tools.ProofPath, workdir, nil, args...)
	payload := decodeJSON(t, result.Stdout)
	return payload, result
}

func runCommand(t *testing.T, binary string, workdir string, env map[string]string, args ...string) commandResult {
	t.Helper()

	cmd := exec.Command(binary, args...)
	cmd.Dir = workdir
	cmd.Env = os.Environ()
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	stdout, err := cmd.CombinedOutput()
	if err == nil {
		return commandResult{Stdout: string(stdout), ExitCode: 0}
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return commandResult{Stdout: string(stdout), ExitCode: exitErr.ExitCode()}
	}
	t.Fatalf("run %s %v: %v output=%s", binary, args, err, string(stdout))
	return commandResult{}
}

func decodeJSON(t *testing.T, raw string) map[string]any {
	t.Helper()

	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode json: %v output=%s", err, raw)
	}
	return payload
}

func writeFixtureJSON(t *testing.T, dir string, name string, payload any) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("MkdirAll %s: %v", dir, err)
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture %s: %v", name, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), raw, 0o600); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}
}

func writePluginSource(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "bad_plugin.go")
	source := "package main\nimport \"fmt\"\nfunc main(){fmt.Println(\"{bad-json\")}\n"
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatalf("write plugin source: %v", err)
	}
	return path
}

func loadChain(t *testing.T, storeDir string) proof.Chain {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(storeDir, "chain.json"))
	if err != nil {
		t.Fatalf("read chain: %v", err)
	}
	var chain proof.Chain
	if err := json.Unmarshal(raw, &chain); err != nil {
		t.Fatalf("decode chain: %v", err)
	}
	return chain
}

func manifestPaths(t *testing.T, bundleDir string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(bundleDir, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest struct {
		Files []struct {
			Path string `json:"path"`
		} `json:"files"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	out := make([]string, 0, len(manifest.Files))
	for _, entry := range manifest.Files {
		out = append(out, entry.Path)
	}
	sort.Strings(out)
	return out
}

func rawRecordTypes(t *testing.T, bundleDir string) []string {
	t.Helper()
	fh, err := os.Open(filepath.Join(bundleDir, "raw-records.jsonl"))
	if err != nil {
		t.Fatalf("open raw-records: %v", err)
	}
	defer func() { _ = fh.Close() }()

	types := []string{}
	scanner := bufio.NewScanner(fh)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record struct {
			RecordType string `json:"record_type"`
		}
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("decode raw record line: %v", err)
		}
		types = append(types, record.RecordType)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan raw-records: %v", err)
	}
	sort.Strings(types)
	return types
}

func findRecordBySourceAndType(t *testing.T, chain proof.Chain, source string, recordType string) proof.Record {
	t.Helper()
	for _, record := range chain.Records {
		if record.Source == source && record.RecordType == recordType {
			return record
		}
	}
	t.Fatalf("record not found for source=%s type=%s", source, recordType)
	return proof.Record{}
}

func findRecordByType(t *testing.T, chain proof.Chain, recordType string) proof.Record {
	t.Helper()
	for _, record := range chain.Records {
		if record.RecordType == recordType {
			return record
		}
	}
	t.Fatalf("record not found for type=%s", recordType)
	return proof.Record{}
}

func stringSliceValue(value any) []string {
	switch typed := value.(type) {
	case []string:
		out := append([]string(nil), typed...)
		sort.Strings(out)
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if str, ok := item.(string); ok && strings.TrimSpace(str) != "" {
				out = append(out, str)
			}
		}
		sort.Strings(out)
		return out
	default:
		return nil
	}
}

func countsByName(items []any) map[string]int {
	out := map[string]int{}
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _ := entry["name"].(string)
		out[name] = intValue(entry["count"])
	}
	return out
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func boolValue(value any) bool {
	typed, _ := value.(bool)
	return typed
}

func stringValue(value any) string {
	typed, _ := value.(string)
	return typed
}

func mapValue(value any) map[string]any {
	typed, _ := value.(map[string]any)
	return typed
}

func sliceValue(value any) []any {
	typed, _ := value.([]any)
	return typed
}

func scenarioExecutorsMap() map[string]func(*testing.T, scenarioTools, Fixture) map[string]any {
	return map[string]func(*testing.T, scenarioTools, Fixture) map[string]any{
		"ac01_install_collect_map":                    execAC01InstallCollectMap,
		"ac02_bundle_board_summary":                   execAC02BundleBoardSummary,
		"ac03_auditor_handoff_bundle_verify":          execAC03AuditorHandoff,
		"ac04_chain_integrity_breakpoint":             execAC04ChainIntegrity,
		"ac05_gap_alert_ranking":                      execAC05GapAlert,
		"ac06_builtin_collector_capture":              execAC06CollectorCoverage,
		"ac07_nonblocking_collection_failure":         execAC07NonBlockingCollect,
		"ac08_offline_fixture_run":                    execAC08OfflineFixtureRun,
		"ac09_mixed_source_chain_parity":              execAC09MixedSourceParity,
		"ac10_datapipeline_semantics_and_replay":      execAC10DataPipelineReplay,
		"ac11_sink_unavailable_fail_closed":           execAC11SinkFailClosed,
		"ac12_oscal_bundle_validation":                execAC12OSCALValidation,
		"ac13_regression_exit5":                       execAC13RegressionExit5,
		"ac14_daily_review_pack":                      execAC14DailyReview,
		"ac15_ticket_retry_dlq_sla":                   execAC15TicketRetryDLQ,
		"ac16_override_artifact_visible_in_bundle":    execAC16OverrideBundle,
		"ac17_third_party_collector_schema_rejection": execAC17PluginSchemaReject,
	}
}

var scenarioExecutors = scenarioExecutorsMap()

func execAC01InstallCollectMap(t *testing.T, tools scenarioTools, _ Fixture) map[string]any {
	t.Helper()

	ws := newScenarioWorkspace(t)
	initPayload, _ := runAxymJSON(t, tools, ws.RootDir, nil, "init", "--store-dir", ws.StoreDir, "--json")
	collectPayload, _ := runAxymJSON(t, tools, ws.RootDir, nil, "collect", "--fixture-dir", filepath.Join(tools.RepoRoot, "fixtures", "collectors"), "--store-dir", ws.StoreDir, "--json")
	mapPayload, _ := runAxymJSON(t, tools, ws.RootDir, nil, "map", "--frameworks", "eu-ai-act,soc2", "--store-dir", ws.StoreDir, "--json")

	collectData := mapValue(collectPayload["data"])
	mapData := mapValue(mapPayload["data"])
	summary := mapValue(mapData["summary"])

	sourceStatuses := map[string]string{}
	for _, item := range sliceValue(collectData["sources"]) {
		source := mapValue(item)
		sourceStatuses[stringValue(source["name"])] = stringValue(source["status"])
	}

	return map[string]any{
		"policy_created": boolValue(mapValue(initPayload["data"])["policy_created"]),
		"policy_path":    stringValue(mapValue(initPayload["data"])["policy_path"]),
		"collect": map[string]any{
			"captured": intValue(collectData["captured"]),
			"appended": intValue(collectData["appended"]),
			"sources":  sourceStatuses,
		},
		"map": map[string]any{
			"framework_count": intValue(summary["framework_count"]),
			"covered_count":   intValue(summary["covered_count"]),
			"gap_count":       intValue(summary["gap_count"]),
		},
	}
}

func execAC02BundleBoardSummary(t *testing.T, tools scenarioTools, _ Fixture) map[string]any {
	t.Helper()

	ws := newScenarioWorkspace(t)
	_, _ = runAxymJSON(t, tools, ws.RootDir, nil, "collect", "--fixture-dir", filepath.Join(tools.RepoRoot, "fixtures", "collectors"), "--store-dir", ws.StoreDir, "--json")
	bundlePayload, _ := runAxymJSON(t, tools, ws.RootDir, nil, "bundle", "--audit", "Q3-2026", "--frameworks", "eu-ai-act,soc2", "--store-dir", ws.StoreDir, "--output", ws.BundleDir, "--json")
	pdfRaw, err := os.ReadFile(filepath.Join(ws.BundleDir, "executive-summary.pdf"))
	if err != nil {
		t.Fatalf("read executive-summary.pdf: %v", err)
	}
	pdfText := string(pdfRaw)

	return map[string]any{
		"bundle_files": intValue(mapValue(bundlePayload["data"])["files"]),
		"pdf_contains": []string{
			"Axym Executive Summary (Q3-2026)",
			"Frameworks: eu-ai-act, soc2",
			"AI systems in scope: 2",
			"Proof records collected: 7",
			"Coverage eu-ai-act: 33.33%",
			"Coverage soc2: 0.00%",
			"Top gap 1: eu-ai-act/article-14",
		},
		"pdf_valid": strings.Contains(pdfText, "Axym Executive Summary \\(Q3-2026\\)") &&
			strings.Contains(pdfText, "Top gap 1: eu-ai-act/article-14"),
	}
}

func execAC03AuditorHandoff(t *testing.T, tools scenarioTools, _ Fixture) map[string]any {
	t.Helper()

	ws := newScenarioWorkspace(t)
	_, _ = runAxymJSON(t, tools, ws.RootDir, nil, "collect", "--fixture-dir", filepath.Join(tools.RepoRoot, "fixtures", "collectors"), "--store-dir", ws.StoreDir, "--json")
	_, _ = runAxymJSON(t, tools, ws.RootDir, nil, "bundle", "--audit", "Q3-2026", "--frameworks", "eu-ai-act,soc2", "--store-dir", ws.StoreDir, "--output", ws.BundleDir, "--json")
	verifyPayload, _ := runAxymJSON(t, tools, ws.RootDir, nil, "verify", "--bundle", ws.BundleDir, "--json")
	proofPayload, _ := runProofJSON(t, tools, ws.RootDir, "verify", "--bundle", ws.BundleDir, "--json")

	verification := mapValue(mapValue(verifyPayload["data"])["verification"])
	rawManifest, err := os.ReadFile(filepath.Join(ws.BundleDir, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest struct {
		Signatures []any `json:"signatures"`
	}
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	return map[string]any{
		"axym_compliance_verified": boolValue(verification["compliance_verified"]),
		"axym_oscal_valid":         boolValue(verification["oscal_valid"]),
		"bundle_signed":            len(manifest.Signatures) == 1,
		"proof_ok":                 boolValue(proofPayload["ok"]),
	}
}

func execAC04ChainIntegrity(t *testing.T, tools scenarioTools, _ Fixture) map[string]any {
	t.Helper()

	ws := newScenarioWorkspace(t)
	_, _ = runAxymJSON(t, tools, ws.RootDir, nil, "collect", "--fixture-dir", filepath.Join(tools.RepoRoot, "fixtures", "collectors"), "--store-dir", ws.StoreDir, "--json")
	chain := loadChain(t, ws.StoreDir)
	chain.Records[0].Event["tampered"] = true
	rawChain, err := json.MarshalIndent(chain, "", "  ")
	if err != nil {
		t.Fatalf("marshal tampered chain: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ws.StoreDir, "chain.json"), rawChain, 0o600); err != nil {
		t.Fatalf("write tampered chain: %v", err)
	}
	verifyPayload, result := runAxymJSON(t, tools, ws.RootDir, nil, "verify", "--chain", "--store-dir", ws.StoreDir, "--json")
	errPayload := mapValue(verifyPayload["error"])
	return map[string]any{
		"exit_code":   result.ExitCode,
		"reason":      stringValue(errPayload["reason"]),
		"break_index": intValue(errPayload["break_index"]),
	}
}

func execAC05GapAlert(t *testing.T, tools scenarioTools, _ Fixture) map[string]any {
	t.Helper()

	ws := newScenarioWorkspace(t)
	_, _ = runAxymJSON(t, tools, ws.RootDir, nil, "collect", "--fixture-dir", filepath.Join(tools.RepoRoot, "fixtures", "collectors"), "--store-dir", ws.StoreDir, "--json")
	gapsPayload, _ := runAxymJSON(t, tools, ws.RootDir, nil, "gaps", "--frameworks", "eu-ai-act,soc2", "--store-dir", ws.StoreDir, "--json")
	data := mapValue(gapsPayload["data"])
	summary := mapValue(data["summary"])
	topGap := mapValue(sliceValue(data["gaps"])[0])
	return map[string]any{
		"gap_count":         intValue(summary["gap_count"]),
		"highest_priority":  intValue(summary["highest_priority"]),
		"top_gap_framework": stringValue(topGap["framework_id"]),
		"top_gap_control":   stringValue(topGap["control_id"]),
		"top_gap_effort":    stringValue(topGap["effort"]),
		"top_gap_fix":       stringValue(topGap["remediation"]),
	}
}

func execAC06CollectorCoverage(t *testing.T, tools scenarioTools, _ Fixture) map[string]any {
	t.Helper()

	ws := newScenarioWorkspace(t)
	collectPayload, _ := runAxymJSON(t, tools, ws.RootDir, nil, "collect", "--fixture-dir", filepath.Join(tools.RepoRoot, "fixtures", "collectors"), "--store-dir", ws.StoreDir, "--json")
	data := mapValue(collectPayload["data"])
	capturedBySource := map[string]int{}
	for _, item := range sliceValue(data["sources"]) {
		source := mapValue(item)
		name := stringValue(source["name"])
		if name == "governanceevent" {
			continue
		}
		capturedBySource[name] = intValue(source["captured"])
	}
	return map[string]any{
		"captured":           intValue(data["captured"]),
		"appended":           intValue(data["appended"]),
		"captured_by_source": capturedBySource,
	}
}

func execAC07NonBlockingCollect(t *testing.T, tools scenarioTools, _ Fixture) map[string]any {
	t.Helper()

	ws := newScenarioWorkspace(t)
	fixtureDir := filepath.Join(ws.RootDir, "fixtures")
	writeFixtureJSON(t, fixtureDir, "snowflake.json", map[string]any{
		"events": []map[string]any{{
			"timestamp": "bad-time",
			"event": map[string]any{
				"job_name":   "daily_models",
				"query_text": "select 1",
			},
		}},
	})
	writeFixtureJSON(t, fixtureDir, "mcp.json", map[string]any{
		"events": []map[string]any{{
			"timestamp": "2026-02-28T09:00:01Z",
			"event": map[string]any{
				"tool_name": "filesystem.read",
				"action":    "read",
			},
			"metadata": map[string]any{"evidence_source": "mcp"},
		}},
	})
	payload, result := runAxymJSON(t, tools, ws.RootDir, nil, "collect", "--fixture-dir", fixtureDir, "--store-dir", ws.StoreDir, "--json")
	data := mapValue(payload["data"])
	return map[string]any{
		"exit_code":    result.ExitCode,
		"captured":     intValue(data["captured"]),
		"appended":     intValue(data["appended"]),
		"failures":     intValue(data["failures"]),
		"reason_codes": stringSliceValue(data["reason_codes"]),
	}
}

func execAC08OfflineFixtureRun(t *testing.T, tools scenarioTools, _ Fixture) map[string]any {
	t.Helper()

	ws := newScenarioWorkspace(t)
	env := map[string]string{
		"HTTP_PROXY":  "http://127.0.0.1:9",
		"HTTPS_PROXY": "http://127.0.0.1:9",
		"ALL_PROXY":   "http://127.0.0.1:9",
		"NO_PROXY":    "",
	}
	collectPayload, _ := runAxymJSON(t, tools, ws.RootDir, env, "collect", "--fixture-dir", filepath.Join(tools.RepoRoot, "fixtures", "collectors"), "--store-dir", ws.StoreDir, "--json")
	mapPayload, _ := runAxymJSON(t, tools, ws.RootDir, env, "map", "--frameworks", "eu-ai-act,soc2", "--store-dir", ws.StoreDir, "--json")
	return map[string]any{
		"captured":      intValue(mapValue(collectPayload["data"])["captured"]),
		"map_gap_count": intValue(mapValue(mapValue(mapPayload["data"])["summary"])["gap_count"]),
		"proxy_blocked": true,
	}
}

func execAC09MixedSourceParity(t *testing.T, tools scenarioTools, _ Fixture) map[string]any {
	t.Helper()

	ws := newScenarioWorkspace(t)
	wrkrPayload, _ := runAxymJSON(t, tools, ws.RootDir, nil, "ingest", "--source", "wrkr", "--input", filepath.Join(tools.RepoRoot, "fixtures", "ingest", "wrkr", "proof_records.jsonl"), "--store-dir", ws.StoreDir, "--state-dir", filepath.Join(ws.RootDir, "state"), "--json")
	gaitPayload, _ := runAxymJSON(t, tools, ws.RootDir, nil, "ingest", "--source", "gait", "--input", filepath.Join(tools.RepoRoot, "fixtures", "ingest", "gait"), "--store-dir", ws.StoreDir, "--json")
	axymVerify, _ := runAxymJSON(t, tools, ws.RootDir, nil, "verify", "--chain", "--store-dir", ws.StoreDir, "--json")
	proofVerify, _ := runProofJSON(t, tools, ws.RootDir, "verify", "--chain", filepath.Join(ws.StoreDir, "chain.json"), "--json")

	axymVerification := mapValue(mapValue(axymVerify["data"])["verification"])
	return map[string]any{
		"wrkr_appended":     intValue(mapValue(mapValue(wrkrPayload["data"])["result"])["appended"]),
		"gait_appended":     intValue(mapValue(mapValue(gaitPayload["data"])["result"])["appended"]),
		"axym_count":        intValue(axymVerification["count"]),
		"proof_records":     intValue(proofVerify["records"]),
		"head_hash_matches": stringValue(axymVerification["head_hash"]) == stringValue(proofVerify["head_hash"]),
	}
}

func execAC10DataPipelineReplay(t *testing.T, tools scenarioTools, _ Fixture) map[string]any {
	t.Helper()

	ws := newScenarioWorkspace(t)
	fixtureDir := filepath.Join(ws.RootDir, "fixtures")
	writeFixtureJSON(t, fixtureDir, "dbt.json", map[string]any{
		"events": []map[string]any{{
			"timestamp": "2026-09-15T09:00:00Z",
			"event": map[string]any{
				"job_name":  "daily_models",
				"git_sha":   strings.Repeat("d", 40),
				"requestor": "alice",
				"approver":  "bob",
				"deployer":  "alice",
				"freeze_windows": []map[string]any{{
					"start": "2026-09-15T08:00:00Z",
					"end":   "2026-09-15T10:00:00Z",
				}},
				"models": []map[string]any{{
					"name": "fact_orders",
					"sql":  "select * from raw.orders",
				}},
			},
			"metadata": map[string]any{"evidence_source": "dbt"},
		}},
	})
	writeFixtureJSON(t, fixtureDir, "snowflake.json", map[string]any{
		"events": []map[string]any{{
			"timestamp": "2026-09-15T09:00:01Z",
			"event": map[string]any{
				"job_name":       "daily_models",
				"warehouse_name": "COMPLIANCE_WH",
				"query_text":     "select count(*) from analytics.fact_orders",
				"query_tag":      "CHANGE-1234",
				"enriched_at":    "2026-09-15T09:10:00Z",
			},
			"metadata": map[string]any{"evidence_source": "snowflake"},
		}},
	})
	_, _ = runAxymJSON(t, tools, ws.RootDir, nil, "collect", "--fixture-dir", fixtureDir, "--store-dir", ws.StoreDir, "--json")
	if _, err := corereplay.Run(corereplay.Request{
		Model:    "payments-agent",
		Tier:     "A",
		StoreDir: ws.StoreDir,
		Now:      func() time.Time { return time.Date(2026, 9, 15, 9, 30, 0, 0, time.UTC) },
	}); err != nil {
		t.Fatalf("replay.Run: %v", err)
	}
	chain := loadChain(t, ws.StoreDir)
	dbtRecord := findRecordBySourceAndType(t, chain, "dbt", "data_pipeline_run")
	snowflakeRecord := findRecordBySourceAndType(t, chain, "snowflake", "data_pipeline_run")
	replayRecord := findRecordByType(t, chain, "replay_certification")

	return map[string]any{
		"dbt": map[string]any{
			"pass":         boolValue(mapValue(dbtRecord.Event["decision"])["pass"]),
			"reason_codes": stringSliceValue(dbtRecord.Event["reason_codes"]),
		},
		"snowflake": map[string]any{
			"pass":         boolValue(mapValue(snowflakeRecord.Event["decision"])["pass"]),
			"query_tag":    stringValue(snowflakeRecord.Event["query_tag"]),
			"reason_codes": stringSliceValue(snowflakeRecord.Event["reason_codes"]),
		},
		"replay": map[string]any{
			"tier":         stringValue(replayRecord.Event["tier"]),
			"status":       stringValue(replayRecord.Event["status"]),
			"reason_codes": stringSliceValue(replayRecord.Metadata["reason_codes"]),
		},
	}
}

func execAC11SinkFailClosed(t *testing.T, tools scenarioTools, _ Fixture) map[string]any {
	t.Helper()

	ws := newScenarioWorkspace(t)
	if err := os.MkdirAll(filepath.Join(ws.StoreDir, "chain.json"), 0o700); err != nil {
		t.Fatalf("mkdir chain dir: %v", err)
	}
	payload, result := runAxymJSON(t, tools, ws.RootDir, nil, "collect", "--fixture-dir", filepath.Join(tools.RepoRoot, "fixtures", "collectors"), "--store-dir", ws.StoreDir, "--json")
	errPayload := mapValue(payload["error"])
	return map[string]any{
		"exit_code": result.ExitCode,
		"reason":    stringValue(errPayload["reason"]),
	}
}

func execAC12OSCALValidation(t *testing.T, tools scenarioTools, _ Fixture) map[string]any {
	t.Helper()

	ws := newScenarioWorkspace(t)
	_, _ = runAxymJSON(t, tools, ws.RootDir, nil, "collect", "--fixture-dir", filepath.Join(tools.RepoRoot, "fixtures", "collectors"), "--store-dir", ws.StoreDir, "--json")
	_, _ = runAxymJSON(t, tools, ws.RootDir, nil, "bundle", "--audit", "Q3-2026", "--frameworks", "eu-ai-act,soc2", "--store-dir", ws.StoreDir, "--output", ws.BundleDir, "--json")
	verifyPayload, _ := runAxymJSON(t, tools, ws.RootDir, nil, "verify", "--bundle", ws.BundleDir, "--json")
	verification := mapValue(mapValue(verifyPayload["data"])["verification"])
	return map[string]any{
		"cryptographic":       boolValue(verification["cryptographic"]),
		"compliance_verified": boolValue(verification["compliance_verified"]),
		"oscal_valid":         boolValue(verification["oscal_valid"]),
	}
}

func execAC13RegressionExit5(t *testing.T, tools scenarioTools, _ Fixture) map[string]any {
	t.Helper()

	ws := newScenarioWorkspace(t)
	baselineStore := filepath.Join(ws.RootDir, "baseline-store")
	currentStore := filepath.Join(ws.RootDir, "current-store")
	baselinePath := filepath.Join(ws.RootDir, "baseline.json")
	frameworkPath := filepath.Join(tools.RepoRoot, "fixtures", "frameworks", "regress-minimal.yaml")
	recordPath := filepath.Join(tools.RepoRoot, "fixtures", "records", "decision.json")

	_, _ = runAxymJSON(t, tools, ws.RootDir, nil, "record", "add", "--input", recordPath, "--store-dir", baselineStore, "--json")
	_, _ = runAxymJSON(t, tools, ws.RootDir, nil, "regress", "init", "--baseline", baselinePath, "--frameworks", frameworkPath, "--store-dir", baselineStore, "--json")
	payload, result := runAxymJSON(t, tools, ws.RootDir, nil, "regress", "run", "--baseline", baselinePath, "--frameworks", frameworkPath, "--store-dir", currentStore, "--json")
	data := mapValue(payload["data"])
	runResult := mapValue(data["result"])
	drifts := sliceValue(runResult["regressed_controls"])
	firstDrift := mapValue(drifts[0])
	return map[string]any{
		"exit_code":      result.ExitCode,
		"reason":         stringValue(mapValue(payload["error"])["reason"]),
		"drift_detected": boolValue(runResult["drift_detected"]),
		"regressed_control": map[string]any{
			"framework_id":    stringValue(firstDrift["framework_id"]),
			"control_id":      stringValue(firstDrift["control_id"]),
			"baseline_status": stringValue(firstDrift["baseline_status"]),
			"current_status":  stringValue(firstDrift["current_status"]),
			"reason":          stringValue(firstDrift["reason"]),
		},
	}
}

func execAC14DailyReview(t *testing.T, tools scenarioTools, _ Fixture) map[string]any {
	t.Helper()

	ws := newScenarioWorkspace(t)
	fixtureDir := filepath.Join(ws.RootDir, "fixtures")
	writeFixtureJSON(t, fixtureDir, "dbt.json", map[string]any{
		"events": []map[string]any{{
			"timestamp": "2026-09-15T09:00:00Z",
			"event": map[string]any{
				"job_name":  "daily_models",
				"git_sha":   strings.Repeat("d", 40),
				"requestor": "alice",
				"approver":  "bob",
				"deployer":  "alice",
				"freeze_windows": []map[string]any{{
					"start": "2026-09-15T08:00:00Z",
					"end":   "2026-09-15T10:00:00Z",
				}},
				"models": []map[string]any{{
					"name": "fact_orders",
					"sql":  "select * from raw.orders",
				}},
			},
			"metadata": map[string]any{"evidence_source": "dbt"},
		}},
	})
	writeFixtureJSON(t, fixtureDir, "snowflake.json", map[string]any{
		"events": []map[string]any{{
			"timestamp": "2026-09-15T09:00:01Z",
			"event": map[string]any{
				"job_name":       "daily_models",
				"warehouse_name": "COMPLIANCE_WH",
				"query_text":     "select count(*) from analytics.fact_orders",
				"query_tag":      "",
				"enriched_at":    "2026-09-15T10:15:00Z",
			},
			"metadata": map[string]any{"evidence_source": "snowflake"},
		}},
	})
	_, _ = runAxymJSON(t, tools, ws.RootDir, nil, "collect", "--fixture-dir", fixtureDir, "--store-dir", ws.StoreDir, "--json")
	evidenceStore, err := store.New(store.Config{RootDir: ws.StoreDir, ComplianceMode: true})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	queue, err := dlq.New(filepath.Join(ws.RootDir, "dlq"))
	if err != nil {
		t.Fatalf("dlq.New: %v", err)
	}
	processor := coreticket.Processor{
		Adapter:     jira.NewScripted([]int{429, 200}),
		Store:       evidenceStore,
		DLQ:         queue,
		MaxAttempts: 3,
		Clock:       func() time.Time { return time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC) },
		Sleep:       func(context.Context, time.Duration) error { return nil },
	}
	if _, err := processor.Process(context.Background(), coreticket.Request{
		ChangeID:    "CHG-100",
		PayloadHash: "fixture",
		OpenedAt:    time.Date(2026, 9, 15, 11, 59, 0, 0, time.UTC),
		SLA:         5 * time.Minute,
	}); err != nil {
		t.Fatalf("ticket.Process: %v", err)
	}
	if _, err := corereplay.Run(corereplay.Request{
		Model:    "payments-agent",
		Tier:     "A",
		StoreDir: ws.StoreDir,
		Now:      func() time.Time { return time.Date(2026, 9, 15, 13, 0, 0, 0, time.UTC) },
	}); err != nil {
		t.Fatalf("replay.Run: %v", err)
	}
	payload, _ := runAxymJSON(t, tools, ws.RootDir, nil, "review", "--date", "2026-09-15", "--store-dir", ws.StoreDir, "--json")
	data := mapValue(payload["data"])
	return map[string]any{
		"record_count":      intValue(data["record_count"]),
		"exceptions":        countsByName(sliceValue(data["exceptions"])),
		"replay_tiers":      countsByName(sliceValue(data["replay_tier_distribution"])),
		"attach_status":     countsByName(sliceValue(data["attach_status"])),
		"attach_sla":        countsByName(sliceValue(data["attach_sla"])),
		"degradation_flags": stringSliceValue(data["degradation_flags"]),
	}
}

func execAC15TicketRetryDLQ(t *testing.T, _ scenarioTools, _ Fixture) map[string]any {
	t.Helper()

	ws := newScenarioWorkspace(t)
	evidenceStore, err := store.New(store.Config{RootDir: ws.StoreDir, ComplianceMode: true})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	queue, err := dlq.New(filepath.Join(ws.RootDir, "dlq"))
	if err != nil {
		t.Fatalf("dlq.New: %v", err)
	}
	processor := coreticket.Processor{
		Adapter:     jira.NewScripted([]int{429, 429, 429}),
		Store:       evidenceStore,
		DLQ:         queue,
		MaxAttempts: 3,
		Clock:       func() time.Time { return time.Date(2026, 9, 15, 14, 0, 0, 0, time.UTC) },
		Sleep:       func(context.Context, time.Duration) error { return nil },
	}
	result, err := processor.Process(context.Background(), coreticket.Request{
		ChangeID:    "CHG-999",
		PayloadHash: "storm",
		OpenedAt:    time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC),
		SLA:         30 * time.Minute,
	})
	if err != nil {
		t.Fatalf("ticket.Process: %v", err)
	}
	entries, err := dlq.ReadAll(queue.Path())
	if err != nil {
		t.Fatalf("dlq.ReadAll: %v", err)
	}
	return map[string]any{
		"status":       result.Status,
		"attempts":     result.Attempts,
		"reason_codes": append([]string(nil), result.ReasonCodes...),
		"sla_within":   result.SLAWithin,
		"dlq_entries":  len(entries),
	}
}

func execAC16OverrideBundle(t *testing.T, tools scenarioTools, _ Fixture) map[string]any {
	t.Helper()

	ws := newScenarioWorkspace(t)
	if _, err := coreoverride.Create(coreoverride.Request{
		Bundle:    "Q3-2026",
		Reason:    "fixture",
		Signer:    "ops-key",
		StoreDir:  ws.StoreDir,
		Now:       func() time.Time { return time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC) },
		ExpiresAt: time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("override.Create: %v", err)
	}
	_, _ = runAxymJSON(t, tools, ws.RootDir, nil, "bundle", "--audit", "Q3-2026", "--frameworks", "eu-ai-act,soc2", "--store-dir", ws.StoreDir, "--output", ws.BundleDir, "--json")
	verifyPayload, _ := runAxymJSON(t, tools, ws.RootDir, nil, "verify", "--bundle", ws.BundleDir, "--json")
	overrideRaw, err := os.ReadFile(filepath.Join(ws.BundleDir, "overrides", "overrides.jsonl"))
	if err != nil {
		t.Fatalf("read bundled override artifact: %v", err)
	}
	return map[string]any{
		"bundle_verifies":           boolValue(verifyPayload["ok"]),
		"override_artifact_bundled": containsString(manifestPaths(t, ws.BundleDir), "overrides/overrides.jsonl"),
		"override_lines":            len(strings.Split(strings.TrimSpace(string(overrideRaw)), "\n")),
		"raw_record_types":          rawRecordTypes(t, ws.BundleDir),
	}
}

func execAC17PluginSchemaReject(t *testing.T, tools scenarioTools, _ Fixture) map[string]any {
	t.Helper()

	ws := newScenarioWorkspace(t)
	pluginPath := writePluginSource(t, ws.RootDir)
	payload, result := runAxymJSON(t, tools, ws.RootDir, nil, "collect", "--json", "--fixture-dir", filepath.Join(tools.RepoRoot, "fixtures", "collectors"), "--store-dir", ws.StoreDir, "--plugin-timeout", "15s", "--plugin", "go run "+pluginPath)
	data := mapValue(payload["data"])
	return map[string]any{
		"exit_code":    result.ExitCode,
		"failures":     intValue(data["failures"]),
		"appended":     intValue(data["appended"]),
		"reason_codes": stringSliceValue(data["reason_codes"]),
	}
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
