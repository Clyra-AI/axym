package governance

import (
	"embed"
	"encoding/json"
	"fmt"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v5"
	"strings"
)

// Schemas are embedded so validation and offline consumers use the same
// versioned assets shipped by Axym.
//
//go:embed action-contract-register.schema.json action-contract-evidence-packet.schema.json judge-evidence.schema.json boundary-attestation.schema.json gait-verification-receipt.schema.json
var assets embed.FS

func RegisterSchema() []byte {
	b, _ := assets.ReadFile("action-contract-register.schema.json")
	return b
}
func PacketSchema() []byte {
	b, _ := assets.ReadFile("action-contract-evidence-packet.schema.json")
	return b
}
func JudgeSchema() []byte    { b, _ := assets.ReadFile("judge-evidence.schema.json"); return b }
func BoundarySchema() []byte { b, _ := assets.ReadFile("boundary-attestation.schema.json"); return b }
func GaitVerificationReceiptSchema() []byte {
	b, _ := assets.ReadFile("gait-verification-receipt.schema.json")
	return b
}
func ValidJSONSchema(name string) bool {
	var v any
	b := RegisterSchema()
	if name == "packet" {
		b = PacketSchema()
	}
	return json.Unmarshal(b, &v) == nil
}

func ValidateRegister(data []byte) error { return validate(data, "register") }
func ValidatePacket(data []byte) error   { return validate(data, "packet") }
func validate(data []byte, name string) error {
	raw := RegisterSchema()
	resource := "register.json"
	if name == "packet" {
		raw = PacketSchema()
		resource = "packet.json"
	}
	c := jsonschema.NewCompiler()
	registerRaw := RegisterSchema()
	if err := c.AddResource("action-contract-register.schema.json", strings.NewReader(string(registerRaw))); err != nil {
		return err
	}
	if err := c.AddResource("https://axym.dev/schemas/v1/governance/action-contract-register.schema.json", strings.NewReader(string(registerRaw))); err != nil {
		return err
	}
	if err := c.AddResource(resource, strings.NewReader(string(raw))); err != nil {
		return err
	}
	s, err := c.Compile(resource)
	if err != nil {
		return fmt.Errorf("compile governance schema: %w", err)
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	return s.Validate(value)
}
