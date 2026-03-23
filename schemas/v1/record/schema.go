package recordschema

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed normalized-input.schema.json manual-input.schema.json
var schemaFS embed.FS

var (
	normalizedCompileOnce sync.Once
	normalizedCompiled    *jsonschema.Schema
	normalizedCompileErr  error

	manualCompileOnce sync.Once
	manualCompiled    *jsonschema.Schema
	manualCompileErr  error
)

func ValidateNormalized(data []byte) error {
	schema, err := normalizedCompiledSchema()
	if err != nil {
		return err
	}
	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("decode normalized payload: %w", err)
	}
	if err := schema.Validate(payload); err != nil {
		return fmt.Errorf("validate normalized payload: %w", err)
	}
	return nil
}

func NormalizeManualInput(data []byte) ([]byte, error) {
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("decode manual payload: %w", err)
	}

	switch raw := payload["record_version"].(type) {
	case nil:
		payload["record_version"] = "v1"
	case string:
		trimmed := strings.TrimSpace(raw)
		switch trimmed {
		case "", "1.0", "v1":
			payload["record_version"] = "v1"
		default:
			payload["record_version"] = trimmed
		}
	}

	normalized, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal normalized manual payload: %w", err)
	}
	return normalized, nil
}

func ValidateManualInput(data []byte) error {
	schema, err := manualCompiledSchema()
	if err != nil {
		return err
	}
	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("decode manual payload: %w", err)
	}
	if err := schema.Validate(payload); err != nil {
		return fmt.Errorf("validate manual payload: %w", err)
	}
	return nil
}

func normalizedCompiledSchema() (*jsonschema.Schema, error) {
	normalizedCompileOnce.Do(func() {
		raw, err := schemaFS.ReadFile("normalized-input.schema.json")
		if err != nil {
			normalizedCompileErr = fmt.Errorf("read normalized schema: %w", err)
			return
		}
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("normalized-input.schema.json", strings.NewReader(string(raw))); err != nil {
			normalizedCompileErr = fmt.Errorf("add schema resource: %w", err)
			return
		}
		normalizedCompiled, normalizedCompileErr = compiler.Compile("normalized-input.schema.json")
		if normalizedCompileErr != nil {
			normalizedCompileErr = fmt.Errorf("compile normalized schema: %w", normalizedCompileErr)
		}
	})
	if normalizedCompileErr != nil {
		return nil, normalizedCompileErr
	}
	return normalizedCompiled, nil
}

func manualCompiledSchema() (*jsonschema.Schema, error) {
	manualCompileOnce.Do(func() {
		raw, err := schemaFS.ReadFile("manual-input.schema.json")
		if err != nil {
			manualCompileErr = fmt.Errorf("read manual schema: %w", err)
			return
		}
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("manual-input.schema.json", strings.NewReader(string(raw))); err != nil {
			manualCompileErr = fmt.Errorf("add schema resource: %w", err)
			return
		}
		manualCompiled, manualCompileErr = compiler.Compile("manual-input.schema.json")
		if manualCompileErr != nil {
			manualCompileErr = fmt.Errorf("compile manual schema: %w", manualCompileErr)
		}
	})
	if manualCompileErr != nil {
		return nil, manualCompileErr
	}
	return manualCompiled, nil
}
