// Package gaitschema exposes Axym's public Gait lifecycle configuration
// contract to runtime callers without relying on workspace-relative files.
package gaitschema

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed lifecycle-verification-config-v1.schema.json
var schemaFS embed.FS

var (
	compileOnce sync.Once
	compiled    *jsonschema.Schema
	compileErr  error
)

// ValidateLifecycleVerificationConfig validates the versioned public JSON
// contract used by --gait-lifecycle-verification.
func ValidateLifecycleVerificationConfig(data []byte) error {
	schema, err := compiledSchema()
	if err != nil {
		return err
	}
	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("decode Gait lifecycle verification config: %w", err)
	}
	if err := schema.Validate(payload); err != nil {
		return fmt.Errorf("validate Gait lifecycle verification config: %w", err)
	}
	return nil
}

func compiledSchema() (*jsonschema.Schema, error) {
	compileOnce.Do(func() {
		raw, err := schemaFS.ReadFile("lifecycle-verification-config-v1.schema.json")
		if err != nil {
			compileErr = fmt.Errorf("read Gait lifecycle verification schema: %w", err)
			return
		}
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("lifecycle-verification-config-v1.schema.json", strings.NewReader(string(raw))); err != nil {
			compileErr = fmt.Errorf("add Gait lifecycle verification schema: %w", err)
			return
		}
		compiled, compileErr = compiler.Compile("lifecycle-verification-config-v1.schema.json")
		if compileErr != nil {
			compileErr = fmt.Errorf("compile Gait lifecycle verification schema: %w", compileErr)
		}
	})
	return compiled, compileErr
}
