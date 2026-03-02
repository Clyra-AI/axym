package regressschema

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed regress-baseline-v1.schema.json
var schemaFS embed.FS

var (
	compileOnce sync.Once
	compiled    *jsonschema.Schema
	compileErr  error
)

func ValidateBaseline(data []byte) error {
	schema, err := compiledSchema()
	if err != nil {
		return err
	}
	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("decode regress baseline payload: %w", err)
	}
	if err := schema.Validate(payload); err != nil {
		return fmt.Errorf("validate regress baseline payload: %w", err)
	}
	return nil
}

func compiledSchema() (*jsonschema.Schema, error) {
	compileOnce.Do(func() {
		raw, err := schemaFS.ReadFile("regress-baseline-v1.schema.json")
		if err != nil {
			compileErr = fmt.Errorf("read regress schema: %w", err)
			return
		}
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("regress-baseline-v1.schema.json", strings.NewReader(string(raw))); err != nil {
			compileErr = fmt.Errorf("add schema resource: %w", err)
			return
		}
		compiled, compileErr = compiler.Compile("regress-baseline-v1.schema.json")
		if compileErr != nil {
			compileErr = fmt.Errorf("compile regress schema: %w", compileErr)
		}
	})
	if compileErr != nil {
		return nil, compileErr
	}
	return compiled, nil
}
