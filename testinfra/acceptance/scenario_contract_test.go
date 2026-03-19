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
		Frameworks          string `json:"frameworks"`
		MinimumCoveredCount int    `json:"minimum_covered_count"`
		ForbiddenGrade      string `json:"forbidden_grade"`
		SamplePackPath      string `json:"sample_pack_path"`
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

	runAcceptanceJSON(t, binaryPath, workdir, "collect", "--json", "--governance-event-file", filepath.Join(samplePackPath, "governance", "context_engineering.jsonl"))
	runAcceptanceJSON(t, binaryPath, workdir, "record", "add", "--input", filepath.Join(samplePackPath, "records", "approval.json"), "--json")
	runAcceptanceJSON(t, binaryPath, workdir, "record", "add", "--input", filepath.Join(samplePackPath, "records", "risk_assessment.json"), "--json")

	mapPayload := runAcceptanceJSON(t, binaryPath, workdir, "map", "--frameworks", contract.Frameworks, "--json")
	mapData, _ := mapPayload["data"].(map[string]any)
	mapSummary, _ := mapData["summary"].(map[string]any)
	if got := int(mapSummary["covered_count"].(float64)); got < contract.MinimumCoveredCount {
		t.Fatalf("covered_count mismatch: got %d want >= %d output=%v", got, contract.MinimumCoveredCount, mapPayload)
	}

	gapsPayload := runAcceptanceJSON(t, binaryPath, workdir, "gaps", "--frameworks", contract.Frameworks, "--json")
	gapsData, _ := gapsPayload["data"].(map[string]any)
	grade, _ := gapsData["grade"].(map[string]any)
	if letter := grade["letter"]; letter == contract.ForbiddenGrade {
		t.Fatalf("unexpected grade %v output=%v", letter, gapsPayload)
	}

	runAcceptanceJSON(t, binaryPath, workdir, "bundle", "--audit", "sample", "--frameworks", contract.Frameworks, "--json")
	verifyPayload := runAcceptanceJSON(t, binaryPath, workdir, "verify", "--chain", "--json")
	verifyData, _ := verifyPayload["data"].(map[string]any)
	verification, _ := verifyData["verification"].(map[string]any)
	if verification["intact"] != true {
		t.Fatalf("expected intact chain output=%v", verifyPayload)
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
