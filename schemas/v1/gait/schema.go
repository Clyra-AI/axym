// Package gaitschema exposes Axym's public Gait lifecycle configuration
// contract to runtime callers without relying on workspace-relative files.
package gaitschema

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v5"
)

var missingPropertyPattern = regexp.MustCompile(`^missing properties: '([a-z][a-z0-9_]*)'`)

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

// SafeValidationDetail returns only the failing JSON pointer and schema
// keyword. It intentionally excludes the rejected value and validator message.
func SafeValidationDetail(err error) string {
	var validationErr *jsonschema.ValidationError
	if !errors.As(err, &validationErr) {
		return ""
	}
	leaf := validationErr
	for len(leaf.Causes) > 0 {
		leaf = leaf.Causes[0]
	}
	field := strings.TrimSpace(leaf.InstanceLocation)
	if field == "" {
		field = "/"
	}
	keyword := strings.TrimSpace(leaf.KeywordLocation)
	if index := strings.LastIndex(keyword, "/"); index >= 0 {
		keyword = keyword[index+1:]
	}
	if keyword == "" {
		keyword = "schema"
	}
	if keyword == "required" {
		if match := missingPropertyPattern.FindStringSubmatch(leaf.Message); len(match) == 2 {
			if field == "/" {
				field += match[1]
			} else {
				field += "/" + match[1]
			}
		}
	}
	return fmt.Sprintf(" at %s (%s)", field, keyword)
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
