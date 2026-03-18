//go:build scenario

package scenarios

import "testing"

func TestAC13RegressionSpec(t *testing.T) {
	fixtures, err := LoadFixtures()
	if err != nil {
		t.Fatalf("LoadFixtures: %v", err)
	}
	fixture, ok := FixtureByID(fixtures, "ac13_regression_exit5")
	if !ok {
		t.Fatal("ac13_regression_exit5 fixture not found")
	}
	runScenarioFixture(t, fixture)
}
