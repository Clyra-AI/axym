//go:build scenario

package scenarios

import "testing"

func TestAcceptanceScenarios(t *testing.T) {
	fixtures, err := LoadFixtures()
	if err != nil {
		t.Fatalf("LoadFixtures: %v", err)
	}

	for _, fixture := range fixtures {
		fixture := fixture
		if fixture.ID == "ac13_regression_exit5" {
			continue
		}
		t.Run(fixture.ID, func(t *testing.T) {
			runScenarioFixture(t, fixture)
		})
	}
}
