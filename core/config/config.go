package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Clyra-AI/axym/core/compliance/threshold"
	configschema "github.com/Clyra-AI/axym/schemas/v1/config"
	"gopkg.in/yaml.v3"
)

const (
	DefaultPolicyFile = "axym-policy.yaml"
	VersionV1         = "v1"
)

type Policy struct {
	Version    string         `yaml:"version" json:"version"`
	Defaults   Defaults       `yaml:"defaults" json:"defaults"`
	Compliance Compliance     `yaml:"compliance,omitempty" json:"compliance,omitempty"`
	PolicyPath string         `yaml:"-" json:"-"`
	raw        map[string]any // preserves deterministic json marshaling inputs
}

type Defaults struct {
	StoreDir   string   `yaml:"store_dir" json:"store_dir"`
	Frameworks []string `yaml:"frameworks" json:"frameworks"`
}

type Compliance struct {
	GlobalMinCoverage    *float64           `yaml:"global_min_coverage,omitempty" json:"global_min_coverage,omitempty"`
	FrameworkMinCoverage map[string]float64 `yaml:"framework_min_coverage,omitempty" json:"framework_min_coverage,omitempty"`
}

func DefaultPolicy() Policy {
	return Policy{
		Version: VersionV1,
		Defaults: Defaults{
			StoreDir:   ".axym",
			Frameworks: []string{"eu-ai-act", "soc2"},
		},
		Compliance: Compliance{
			FrameworkMinCoverage: map[string]float64{},
		},
	}
}

func Discover(explicitPath string) (Policy, bool, error) {
	path := strings.TrimSpace(explicitPath)
	if path != "" {
		policy, err := Load(path)
		return policy, true, err
	}
	if _, err := os.Stat(DefaultPolicyFile); err == nil {
		policy, loadErr := Load(DefaultPolicyFile)
		return policy, true, loadErr
	} else if !errors.Is(err, os.ErrNotExist) {
		return Policy{}, false, fmt.Errorf("stat %s: %w", DefaultPolicyFile, err)
	}
	return DefaultPolicy(), false, nil
}

func Load(path string) (Policy, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return Policy{}, fmt.Errorf("policy path is required")
	}
	// #nosec G304 -- policy path is explicit user input or default local file.
	raw, err := os.ReadFile(trimmed)
	if err != nil {
		return Policy{}, fmt.Errorf("read policy file: %w", err)
	}
	policy, err := parse(raw)
	if err != nil {
		return Policy{}, err
	}
	policy.PolicyPath = trimmed
	return policy, nil
}

func WriteDefault(path string, overwrite bool) (Policy, bool, error) {
	target := strings.TrimSpace(path)
	if target == "" {
		target = DefaultPolicyFile
	}
	if _, err := os.Stat(target); err == nil && !overwrite {
		policy, loadErr := Load(target)
		return policy, false, loadErr
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Policy{}, false, fmt.Errorf("stat policy file: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil && filepath.Dir(target) != "." {
		return Policy{}, false, fmt.Errorf("create policy directory: %w", err)
	}
	content := defaultPolicyYAML()
	if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
		return Policy{}, false, fmt.Errorf("write policy file: %w", err)
	}
	policy, err := Load(target)
	if err != nil {
		return Policy{}, false, err
	}
	return policy, true, nil
}

func (p Policy) ThresholdPolicy() threshold.PolicyConfig {
	out := threshold.PolicyConfig{
		GlobalMinCoverage: p.Compliance.GlobalMinCoverage,
	}
	if len(p.Compliance.FrameworkMinCoverage) > 0 {
		out.FrameworkMinCoverage = make(map[string]float64, len(p.Compliance.FrameworkMinCoverage))
		for key, value := range p.Compliance.FrameworkMinCoverage {
			out.FrameworkMinCoverage[strings.ToLower(strings.TrimSpace(key))] = value
		}
	}
	return out
}

func (p Policy) ResolveStoreDir(explicit string) string {
	value := strings.TrimSpace(explicit)
	if value != "" {
		return value
	}
	if resolved := strings.TrimSpace(p.Defaults.StoreDir); resolved != "" {
		return resolved
	}
	return ".axym"
}

func (p Policy) ResolveFrameworks(explicit []string) []string {
	normalizedExplicit := normalizeFrameworks(explicit)
	if len(normalizedExplicit) > 0 {
		return normalizedExplicit
	}
	normalizedDefault := normalizeFrameworks(p.Defaults.Frameworks)
	if len(normalizedDefault) > 0 {
		return normalizedDefault
	}
	return []string{"eu-ai-act", "soc2"}
}

func parse(raw []byte) (Policy, error) {
	var typed Policy
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(&typed); err != nil {
		return Policy{}, fmt.Errorf("decode policy yaml: %w", err)
	}

	var generic map[string]any
	if err := yaml.Unmarshal(raw, &generic); err != nil {
		return Policy{}, fmt.Errorf("decode policy yaml map: %w", err)
	}
	jsonRaw, err := json.Marshal(generic)
	if err != nil {
		return Policy{}, fmt.Errorf("marshal policy json: %w", err)
	}
	if err := configschema.ValidatePolicy(jsonRaw); err != nil {
		return Policy{}, fmt.Errorf("policy schema validation failed: %w", err)
	}

	typed.Version = strings.TrimSpace(typed.Version)
	typed.Defaults.StoreDir = strings.TrimSpace(typed.Defaults.StoreDir)
	typed.Defaults.Frameworks = normalizeFrameworks(typed.Defaults.Frameworks)
	typed.Compliance.FrameworkMinCoverage = normalizeThresholdMap(typed.Compliance.FrameworkMinCoverage)
	typed.raw = generic
	return typed, nil
}

func normalizeFrameworks(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized := trimmed
		if !isLikelyPath(trimmed) {
			normalized = strings.ToLower(trimmed)
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

func isLikelyPath(value string) bool {
	if value == "" {
		return false
	}
	if filepath.IsAbs(value) {
		return true
	}
	if strings.HasPrefix(value, ".") {
		return true
	}
	return strings.ContainsAny(value, `/\`)
}

func normalizeThresholdMap(values map[string]float64) map[string]float64 {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]float64, len(values))
	for frameworkID, minimum := range values {
		key := strings.ToLower(strings.TrimSpace(frameworkID))
		if key == "" {
			continue
		}
		out[key] = minimum
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func defaultPolicyYAML() string {
	return strings.TrimSpace(`
version: v1
defaults:
  store_dir: .axym
  frameworks:
    - eu-ai-act
    - soc2
compliance:
  framework_min_coverage: {}
`) + "\n"
}
