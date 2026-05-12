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

var compiledSchemas sync.Map

func ValidateOSCAL(data []byte) error {
	schema, err := compiledSchema("oscal-component-definition-v1_1.schema.json")
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
	schema, err := compiledSchema("executive-summary-v1.schema.json")
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
	schema, err := compiledSchema("record-signing-key-v1.schema.json")
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

func ValidateAuthorizationRegister(data []byte) error {
	return validate("authorization-register-v1.schema.json", "authorization register", data)
}

func ValidateInsuranceEvidenceProfile(data []byte) error {
	return validate("insurance-evidence-profile-v1.schema.json", "insurance evidence profile", data)
}

func ValidateCredentialPostureRegister(data []byte) error {
	return validate("credential-posture-register-v1.schema.json", "credential posture register", data)
}

func ValidateFreezeWindowCoverage(data []byte) error {
	return validate("freeze-window-coverage-v1.schema.json", "freeze-window coverage", data)
}

func ValidateKillSwitchCoverage(data []byte) error {
	return validate("kill-switch-coverage-v1.schema.json", "kill-switch coverage", data)
}

func ValidateEnforcementExplainRegister(data []byte) error {
	return validate("enforcement-explain-register-v1.schema.json", "enforcement explain register", data)
}

func ValidateSandboxCoverage(data []byte) error {
	return validate("sandbox-coverage-v1.schema.json", "sandbox coverage", data)
}

func ValidateControlMaturity(data []byte) error {
	return validate("control-maturity-v1.schema.json", "control maturity", data)
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

func compiledSchema(name string) (*jsonschema.Schema, error) {
	if cached, ok := compiledSchemas.Load(name); ok {
		entry := cached.(compiledSchemaEntry)
		return entry.schema, entry.err
	}
	schema, err := compile(name)
	compiledSchemas.Store(name, compiledSchemaEntry{schema: schema, err: err})
	return schema, err
}

type compiledSchemaEntry struct {
	schema *jsonschema.Schema
	err    error
}

func validate(name string, label string, data []byte) error {
	schema, err := compiledSchema(name)
	if err != nil {
		return err
	}
	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("decode %s: %w", label, err)
	}
	if err := schema.Validate(payload); err != nil {
		return fmt.Errorf("validate %s: %w", label, err)
	}
	return nil
}
