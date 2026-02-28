package governanceeventschema

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed governance-event.schema.json
var schemaFS embed.FS

var (
	compileOnce sync.Once
	compiled    *jsonschema.Schema
	compileErr  error
)

func Validate(data []byte) error {
	schema, err := compiledSchema()
	if err != nil {
		return err
	}
	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("decode governance event: %w", err)
	}
	if err := schema.Validate(payload); err != nil {
		return fmt.Errorf("validate governance event: %w", err)
	}
	return nil
}

func compiledSchema() (*jsonschema.Schema, error) {
	compileOnce.Do(func() {
		raw, err := schemaFS.ReadFile("governance-event.schema.json")
		if err != nil {
			compileErr = fmt.Errorf("read governance event schema: %w", err)
			return
		}
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("governance-event.schema.json", strings.NewReader(string(raw))); err != nil {
			compileErr = fmt.Errorf("add schema resource: %w", err)
			return
		}
		compiled, compileErr = compiler.Compile("governance-event.schema.json")
		if compileErr != nil {
			compileErr = fmt.Errorf("compile governance event schema: %w", compileErr)
		}
	})
	if compileErr != nil {
		return nil, compileErr
	}
	return compiled, nil
}
