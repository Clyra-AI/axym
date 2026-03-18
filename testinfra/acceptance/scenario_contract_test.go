package acceptance

import (
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
