package recordschema

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed normalized-input.schema.json
var schemaFS embed.FS

var (
	compileOnce sync.Once
	compiled    *jsonschema.Schema
	compileErr  error
)

func ValidateNormalized(data []byte) error {
	schema, err := compiledSchema()
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

func compiledSchema() (*jsonschema.Schema, error) {
	compileOnce.Do(func() {
		raw, err := schemaFS.ReadFile("normalized-input.schema.json")
		if err != nil {
			compileErr = fmt.Errorf("read normalized schema: %w", err)
			return
		}
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("normalized-input.schema.json", strings.NewReader(string(raw))); err != nil {
			compileErr = fmt.Errorf("add schema resource: %w", err)
			return
		}
		compiled, compileErr = compiler.Compile("normalized-input.schema.json")
		if compileErr != nil {
			compileErr = fmt.Errorf("compile normalized schema: %w", compileErr)
		}
	})
	if compileErr != nil {
		return nil, compileErr
	}
	return compiled, nil
}
