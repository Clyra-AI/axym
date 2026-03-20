package acceptance

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	internalscenarios "github.com/Clyra-AI/axym/internal/scenarios"
)

func TestScenarioFixtureCoverageAndGoldens(t *testing.T) {
	t.Parallel()

	repoRoot, err := internalscenarios.RepoRoot()
	if err != nil {
		t.Fatalf("RepoRoot: %v", err)
	}
	fixtures, err := internalscenarios.LoadFixturesFromRoot(repoRoot)
	if err != nil {
		t.Fatalf("LoadFixturesFromRoot: %v", err)
	}
	goldens, err := internalscenarios.LoadGoldenResultsFromRoot(repoRoot)
	if err != nil {
		t.Fatalf("LoadGoldenResultsFromRoot: %v", err)
	}

	if len(fixtures) != 17 {
		t.Fatalf("expected 17 scenario fixtures, got %d", len(fixtures))
	}

	seenIDs := map[string]struct{}{}
	seenCriteria := map[string]int{}
	for _, fixture := range fixtures {
		if fixture.ID == "" {
			t.Fatal("fixture id must be non-empty")
		}
		if _, ok := seenIDs[fixture.ID]; ok {
			t.Fatalf("duplicate fixture id: %s", fixture.ID)
		}
		seenIDs[fixture.ID] = struct{}{}
		if fixture.Title == "" {
			t.Fatalf("fixture %s missing title", fixture.ID)
		}
		if fixture.PassCommand == "" {
			t.Fatalf("fixture %s missing pass command", fixture.ID)
		}
		if _, ok := goldens[fixture.ID]; !ok {
			t.Fatalf("fixture %s missing golden output", fixture.ID)
		}
		for _, criterion := range fixture.AcceptanceCriteria {
			seenCriteria[criterion]++
		}
	}

	for i := 1; i <= 17; i++ {
		key := "AC" + itoa(i)
		if seenCriteria[key] == 0 {
			t.Fatalf("missing coverage for %s", key)
		}
	}
}

func TestCrossProductScenarioCommandsAreDeclared(t *testing.T) {
	t.Parallel()

	fixtures, err := internalscenarios.LoadFixtures()
	if err != nil {
		t.Fatalf("LoadFixtures: %v", err)
	}
	fixture, ok := internalscenarios.FixtureByID(fixtures, "ac09_mixed_source_chain_parity")
	if !ok {
		t.Fatal("ac09_mixed_source_chain_parity fixture missing")
	}
	if fixture.PassCommand == "" {
		t.Fatal("ac09 pass command must be declared")
	}
	if !contains(fixture.PassCommand, "axym verify --chain") || !contains(fixture.PassCommand, "proof verify --chain") {
		t.Fatalf("ac09 pass command must declare both verification runtimes: %s", fixture.PassCommand)
	}
}

func TestAuditorHandoffScenarioUsesProofBundleVerification(t *testing.T) {
	t.Parallel()

	fixtures, err := internalscenarios.LoadFixtures()
	if err != nil {
		t.Fatalf("LoadFixtures: %v", err)
	}
	fixture, ok := internalscenarios.FixtureByID(fixtures, "ac03_auditor_handoff_bundle_verify")
	if !ok {
		t.Fatal("ac03_auditor_handoff_bundle_verify fixture missing")
	}
	if !contains(fixture.PassCommand, "proof verify --bundle") {
		t.Fatalf("ac03 pass command must declare proof bundle verification: %s", fixture.PassCommand)
	}
}

func TestInstalledBinaryFirstValueSamplePath(t *testing.T) {
	t.Parallel()

	repoRoot := testRepoRoot(t)
	contractPath := filepath.Join(repoRoot, "scenarios", "axym", "first_value_sample", "contract.json")
	rawContract, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("ReadFile(contract): %v", err)
	}
	var contract struct {
		Frameworks                  string   `json:"frameworks"`
		SamplePackPath              string   `json:"sample_pack_path"`
		GovernanceEventCaptureCount int      `json:"governance_event_capture_count"`
		FinalChainRecordCount       int      `json:"final_chain_record_count"`
		CoveredControlCount         int      `json:"covered_control_count"`
		ControlCount                int      `json:"control_count"`
		Grade                       string   `json:"grade"`
		BundleComplete              bool     `json:"bundle_complete"`
		WeakRecordCount             int      `json:"weak_record_count"`
		ChainIntact                 bool     `json:"chain_intact"`
		RemainingGapControlIDs      []string `json:"remaining_gap_control_ids"`
		RequiredArtifacts           []string `json:"required_bundle_artifacts"`
	}
	if err := json.Unmarshal(rawContract, &contract); err != nil {
		t.Fatalf("Unmarshal(contract): %v", err)
	}

	binaryPath := buildAxymAcceptanceBinary(t, repoRoot)
	workdir := t.TempDir()
	samplePackPath := contract.SamplePackPath
	if samplePackPath == "" {
		samplePackPath = "./axym-sample"
	}

	initPayload := runAcceptanceJSON(t, binaryPath, workdir, "init", "--sample-pack", samplePackPath, "--json")
	initData, _ := initPayload["data"].(map[string]any)
	if _, ok := initData["sample_pack"].(map[string]any); !ok {
		t.Fatalf("expected sample_pack data output=%v", initPayload)
	}

	collectPayload := runAcceptanceJSON(t, binaryPath, workdir, "collect", "--json", "--governance-event-file", filepath.Join(samplePackPath, "governance", "context_engineering.jsonl"))
	collectData, _ := collectPayload["data"].(map[string]any)
	if got := intValue(t, collectData, "captured"); got != contract.GovernanceEventCaptureCount {
		t.Fatalf("captured mismatch: got %d want %d output=%v", got, contract.GovernanceEventCaptureCount, collectPayload)
	}
	if got := intValue(t, collectData, "record_count"); got != contract.GovernanceEventCaptureCount {
		t.Fatalf("record_count mismatch after collect: got %d want %d output=%v", got, contract.GovernanceEventCaptureCount, collectPayload)
	}
	sources, _ := collectData["sources"].([]any)
	foundGovernanceSource := false
	for _, candidate := range sources {
		source, _ := candidate.(map[string]any)
		if source["name"] != "governanceevent" {
			continue
		}
		foundGovernanceSource = true
		if got := intValue(t, source, "captured"); got != contract.GovernanceEventCaptureCount {
			t.Fatalf("governanceevent source captured mismatch: got %d want %d output=%v", got, contract.GovernanceEventCaptureCount, source)
		}
	}
	if !foundGovernanceSource {
		t.Fatalf("governanceevent source summary missing output=%v", collectPayload)
	}

	approvalPayload := runAcceptanceJSON(t, binaryPath, workdir, "record", "add", "--input", filepath.Join(samplePackPath, "records", "approval.json"), "--json")
	approvalData, _ := approvalPayload["data"].(map[string]any)
	if got := intValue(t, approvalData, "record_count"); got != contract.GovernanceEventCaptureCount+1 {
		t.Fatalf("record_count mismatch after approval append: got %d want %d output=%v", got, contract.GovernanceEventCaptureCount+1, approvalPayload)
	}

	riskPayload := runAcceptanceJSON(t, binaryPath, workdir, "record", "add", "--input", filepath.Join(samplePackPath, "records", "risk_assessment.json"), "--json")
	riskData, _ := riskPayload["data"].(map[string]any)
	if got := intValue(t, riskData, "record_count"); got != contract.FinalChainRecordCount {
		t.Fatalf("record_count mismatch after risk append: got %d want %d output=%v", got, contract.FinalChainRecordCount, riskPayload)
	}

	mapPayload := runAcceptanceJSON(t, binaryPath, workdir, "map", "--frameworks", contract.Frameworks, "--json")
	mapData, _ := mapPayload["data"].(map[string]any)
	mapSummary, _ := mapData["summary"].(map[string]any)
	if got := intValue(t, mapSummary, "covered_count"); got != contract.CoveredControlCount {
		t.Fatalf("covered_count mismatch: got %d want %d output=%v", got, contract.CoveredControlCount, mapPayload)
	}
	if got := intValue(t, mapSummary, "control_count"); got != contract.ControlCount {
		t.Fatalf("control_count mismatch: got %d want %d output=%v", got, contract.ControlCount, mapPayload)
	}

	gapsPayload := runAcceptanceJSON(t, binaryPath, workdir, "gaps", "--frameworks", contract.Frameworks, "--json")
	gapsData, _ := gapsPayload["data"].(map[string]any)
	grade, _ := gapsData["grade"].(map[string]any)
	if letter, _ := grade["letter"].(string); letter != contract.Grade {
		t.Fatalf("grade mismatch: got %q want %q output=%v", letter, contract.Grade, gapsPayload)
	}
	gaps, _ := gapsData["gaps"].([]any)
	if len(contract.RemainingGapControlIDs) > 0 {
		seenGaps := map[string]struct{}{}
		for _, candidate := range gaps {
			gap, _ := candidate.(map[string]any)
			if controlID, _ := gap["control_id"].(string); controlID != "" {
				seenGaps[controlID] = struct{}{}
			}
		}
		for _, want := range contract.RemainingGapControlIDs {
			if _, ok := seenGaps[want]; !ok {
				t.Fatalf("remaining gap %q missing output=%v", want, gapsPayload)
			}
		}
	}

	bundlePayload := runAcceptanceJSON(t, binaryPath, workdir, "bundle", "--audit", "sample", "--frameworks", contract.Frameworks, "--json")
	bundleData, _ := bundlePayload["data"].(map[string]any)
	bundlePath, _ := bundleData["path"].(string)
	if bundlePath != "" && !filepath.IsAbs(bundlePath) {
		bundlePath = filepath.Join(workdir, bundlePath)
	}
	for _, rel := range contract.RequiredArtifacts {
		if _, err := os.Stat(filepath.Join(workdir, rel)); err == nil {
			continue
		}
		if bundlePath == "" {
			t.Fatalf("bundle path missing for required artifact check: %v", bundlePayload)
		}
		if _, err := os.Stat(filepath.Join(bundlePath, rel)); err != nil {
			t.Fatalf("missing required bundle artifact %s: %v", rel, err)
		}
	}
	compliance, _ := bundleData["compliance"].(map[string]any)
	if complete := boolValue(t, compliance, "complete"); complete != contract.BundleComplete {
		t.Fatalf("bundle complete mismatch: got %t want %t output=%v", complete, contract.BundleComplete, bundlePayload)
	}
	bundleGrade, _ := compliance["grade"].(map[string]any)
	if letter, _ := bundleGrade["letter"].(string); letter != contract.Grade {
		t.Fatalf("bundle grade mismatch: got %q want %q output=%v", letter, contract.Grade, bundlePayload)
	}
	identityGovernance, _ := compliance["identity_governance"].(map[string]any)
	if got := intValue(t, identityGovernance, "weak_record_count"); got != contract.WeakRecordCount {
		t.Fatalf("weak_record_count mismatch: got %d want %d output=%v", got, contract.WeakRecordCount, bundlePayload)
	}
	bundleVerification, _ := bundleData["verification"].(map[string]any)
	if intact := boolValue(t, bundleVerification, "intact"); intact != contract.ChainIntact {
		t.Fatalf("bundle verification intact mismatch: got %t want %t output=%v", intact, contract.ChainIntact, bundlePayload)
	}
	if got := intValue(t, bundleVerification, "count"); got != contract.FinalChainRecordCount {
		t.Fatalf("bundle verification count mismatch: got %d want %d output=%v", got, contract.FinalChainRecordCount, bundlePayload)
	}

	verifyPayload := runAcceptanceJSON(t, binaryPath, workdir, "verify", "--chain", "--json")
	verifyData, _ := verifyPayload["data"].(map[string]any)
	verification, _ := verifyData["verification"].(map[string]any)
	if intact := boolValue(t, verification, "intact"); intact != contract.ChainIntact {
		t.Fatalf("expected intact chain output=%v", verifyPayload)
	}
	if got := intValue(t, verification, "count"); got != contract.FinalChainRecordCount {
		t.Fatalf("verify count mismatch: got %d want %d output=%v", got, contract.FinalChainRecordCount, verifyPayload)
	}
}

func contains(value string, needle string) bool {
	return len(value) >= len(needle) && stringIndex(value, needle) >= 0
}

func stringIndex(value string, needle string) int {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

func itoa(value int) string {
	if value < 10 {
		return string(rune('0' + value))
	}
	return "1" + string(rune('0'+(value%10)))
}

func testRepoRoot(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}

func buildAxymAcceptanceBinary(t *testing.T, repoRoot string) string {
	t.Helper()

	binaryName := "axym"
	if runtime.GOOS == "windows" {
		binaryName = "axym.exe"
	}
	binaryPath := filepath.Join(t.TempDir(), binaryName)
	build := exec.Command("go", "build", "-o", binaryPath, "./cmd/axym")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build axym: %v output=%s", err, string(out))
	}
	return binaryPath
}

func runAcceptanceJSON(t *testing.T, binaryPath string, workdir string, args ...string) map[string]any {
	t.Helper()

	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Fatalf("run %v exit=%d output=%s", args, exitErr.ExitCode(), string(out))
		}
		t.Fatalf("run %v: %v output=%s", args, err, string(out))
	}

	var payload map[string]any
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode output for %v: %v output=%s", args, err, string(out))
	}
	return payload
}

func intValue(t *testing.T, values map[string]any, key string) int {
	t.Helper()

	raw, ok := values[key]
	if !ok {
		t.Fatalf("missing numeric key %q in %+v", key, values)
	}
	number, ok := raw.(float64)
	if !ok {
		t.Fatalf("key %q is not numeric in %+v", key, values)
	}
	return int(number)
}

func boolValue(t *testing.T, values map[string]any, key string) bool {
	t.Helper()

	raw, ok := values[key]
	if !ok {
		t.Fatalf("missing bool key %q in %+v", key, values)
	}
	flag, ok := raw.(bool)
	if !ok {
		t.Fatalf("key %q is not bool in %+v", key, values)
	}
	return flag
}
