package bundleschema

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed *.schema.json
var schemaFS embed.FS

var (
	oscalOnce sync.Once
	oscalDef  *jsonschema.Schema
	oscalErr  error

	execOnce sync.Once
	execDef  *jsonschema.Schema
	execErr  error

	signingKeyOnce sync.Once
	signingKeyDef  *jsonschema.Schema
	signingKeyErr  error
)

func ValidateOSCAL(data []byte) error {
	schema, err := compiledOSCAL()
	if err != nil {
		return err
	}
	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("decode oscal payload: %w", err)
	}
	if err := schema.Validate(payload); err != nil {
		return fmt.Errorf("validate oscal payload: %w", err)
	}
	return nil
}

func ValidateExecutiveSummary(data []byte) error {
	schema, err := compiledExecutiveSummary()
	if err != nil {
		return err
	}
	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("decode executive summary: %w", err)
	}
	if err := schema.Validate(payload); err != nil {
		return fmt.Errorf("validate executive summary: %w", err)
	}
	return nil
}

func ValidateRecordSigningKey(data []byte) error {
	schema, err := compiledRecordSigningKey()
	if err != nil {
		return err
	}
	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("decode record signing key: %w", err)
	}
	if err := schema.Validate(payload); err != nil {
		return fmt.Errorf("validate record signing key: %w", err)
	}
	return nil
}

func compiledOSCAL() (*jsonschema.Schema, error) {
	oscalOnce.Do(func() {
		oscalDef, oscalErr = compile("oscal-component-definition-v1_1.schema.json")
	})
	if oscalErr != nil {
		return nil, oscalErr
	}
	return oscalDef, nil
}

func compiledExecutiveSummary() (*jsonschema.Schema, error) {
	execOnce.Do(func() {
		execDef, execErr = compile("executive-summary-v1.schema.json")
	})
	if execErr != nil {
		return nil, execErr
	}
	return execDef, nil
}

func compiledRecordSigningKey() (*jsonschema.Schema, error) {
	signingKeyOnce.Do(func() {
		signingKeyDef, signingKeyErr = compile("record-signing-key-v1.schema.json")
	})
	if signingKeyErr != nil {
		return nil, signingKeyErr
	}
	return signingKeyDef, nil
}

func compile(name string) (*jsonschema.Schema, error) {
	raw, err := schemaFS.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read schema %q: %w", name, err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(name, strings.NewReader(string(raw))); err != nil {
		return nil, fmt.Errorf("add schema resource %q: %w", name, err)
	}
	compiled, err := compiler.Compile(name)
	if err != nil {
		return nil, fmt.Errorf("compile schema %q: %w", name, err)
	}
	return compiled, nil
}
