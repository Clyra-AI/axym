package scenarios

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	FixtureManifestRelPath = "scenarios/axym/fixtures.yaml"
	GoldenResultsRelPath   = "scenarios/axym/golden/results.json"
)

type Fixture struct {
	ID                 string   `yaml:"id"`
	Title              string   `yaml:"title"`
	AcceptanceCriteria []string `yaml:"acceptance_criteria"`
	PassCommand        string   `yaml:"pass_command"`
}

type fixtureManifest struct {
	Scenarios []Fixture `yaml:"scenarios"`
}

func RepoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..")), nil
}

func LoadFixtures() ([]Fixture, error) {
	repoRoot, err := RepoRoot()
	if err != nil {
		return nil, err
	}
	return LoadFixturesFromRoot(repoRoot)
}

func LoadFixturesFromRoot(repoRoot string) ([]Fixture, error) {
	raw, err := os.ReadFile(filepath.Join(repoRoot, FixtureManifestRelPath))
	if err != nil {
		return nil, err
	}
	var manifest fixtureManifest
	if err := yaml.Unmarshal(raw, &manifest); err != nil {
		return nil, fmt.Errorf("decode scenario fixtures: %w", err)
	}
	out := make([]Fixture, 0, len(manifest.Scenarios))
	for _, fixture := range manifest.Scenarios {
		fixture.ID = strings.TrimSpace(strings.ToLower(fixture.ID))
		fixture.Title = strings.TrimSpace(fixture.Title)
		fixture.PassCommand = strings.TrimSpace(fixture.PassCommand)
		fixture.AcceptanceCriteria = normalizeCriteria(fixture.AcceptanceCriteria)
		out = append(out, fixture)
	}
	return out, nil
}

func LoadGoldenResults() (map[string]json.RawMessage, error) {
	repoRoot, err := RepoRoot()
	if err != nil {
		return nil, err
	}
	return LoadGoldenResultsFromRoot(repoRoot)
}

func LoadGoldenResultsFromRoot(repoRoot string) (map[string]json.RawMessage, error) {
	raw, err := os.ReadFile(filepath.Join(repoRoot, GoldenResultsRelPath))
	if err != nil {
		return nil, err
	}
	var results map[string]json.RawMessage
	if err := json.Unmarshal(raw, &results); err != nil {
		return nil, fmt.Errorf("decode scenario golden results: %w", err)
	}
	return results, nil
}

func FixtureByID(fixtures []Fixture, id string) (Fixture, bool) {
	needle := strings.TrimSpace(strings.ToLower(id))
	for _, fixture := range fixtures {
		if fixture.ID == needle {
			return fixture, true
		}
	}
	return Fixture{}, false
}

func normalizeCriteria(criteria []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(criteria))
	for _, criterion := range criteria {
		normalized := strings.ToUpper(strings.TrimSpace(criterion))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	sort.Strings(out)
	return out
}
