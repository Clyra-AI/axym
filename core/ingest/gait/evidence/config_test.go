package evidence

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v5"
)

func TestParseVerificationConfigStrictAndCallerBound(t *testing.T) {
	key, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	config := validVerificationConfig(key)
	raw, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	options, err := ParseVerificationConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if options.ExpectedContract.ID != config["expected_contract"].(map[string]any)["id"] || options.ExpectedLifecycleDigest == "" {
		t.Fatalf("config bindings not preserved: %+v", options)
	}

	unknown := append([]byte(`{"unknown":true,`), raw[1:]...)
	if _, err := ParseVerificationConfig(unknown); err == nil || !strings.Contains(err.Error(), ReasonConfigInvalid) {
		t.Fatalf("unknown field accepted: %v", err)
	}
	duplicate := append([]byte(`{"expected_family":"a","`), raw[1:]...)
	if _, err := ParseVerificationConfig(duplicate); err == nil || !strings.Contains(err.Error(), ReasonConfigInvalid) {
		t.Fatalf("duplicate field accepted: %v", err)
	}
}

func TestParseVerificationConfigRejectsFixtureKeyUnlessExplicit(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "testdata", "gait-action-contract-evidence", "v1")
	raw, err := os.ReadFile(filepath.Join(root, "fixture-signing-key.public.b64"))
	if err != nil {
		t.Fatal(err)
	}
	keyBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	key := ed25519.PublicKey(keyBytes)
	config := validVerificationConfig(key)
	if _, err := ParseVerificationConfig(mustMarshalConfig(config)); err == nil {
		t.Fatal("fixture-like key unexpectedly accepted")
	}
	config["allow_fixture_only"] = true
	config["expected_lifecycle_digest"] = "sha256:fcb0085b5af73b8a42aa09c25c09f6510d4eb39b8c06a0eb4e16bcbded4fffa2"
	if _, err := ParseVerificationConfig(mustMarshalConfig(config)); err != nil {
		t.Fatalf("explicit exact released fixture config rejected: %v", err)
	}
}

func TestPublicVerificationConfigSchemaAcceptsExample(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "schemas", "v1", "gait")
	schemaRaw, err := os.ReadFile(filepath.Join(root, "lifecycle-verification-config-v1.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	exampleRaw, err := os.ReadFile(filepath.Join(root, "lifecycle-verification-config.example.json"))
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("lifecycle-verification-config-v1.schema.json", strings.NewReader(string(schemaRaw))); err != nil {
		t.Fatal(err)
	}
	compiled, err := compiler.Compile("lifecycle-verification-config-v1.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var example any
	if err := json.Unmarshal(exampleRaw, &example); err != nil {
		t.Fatal(err)
	}
	if err := compiled.Validate(example); err != nil {
		t.Fatalf("public example does not satisfy schema: %v", err)
	}
	if _, err := ParseVerificationConfig(exampleRaw); err != nil {
		t.Fatalf("public example does not satisfy runtime parser: %v", err)
	}
}

func validVerificationConfig(key ed25519.PublicKey) map[string]any {
	digest := "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	ref := func(kind, id, schema, version, source string) map[string]any {
		return map[string]any{"kind": kind, "id": id, "digest": digest, "schema_id": schema, "schema_version": version, "source_product": source}
	}
	return map[string]any{
		"trusted_public_key": base64.StdEncoding.EncodeToString(key),
		"expected_contract":  ref("action_contract", "pac-4b7f1402784256ce", ContractSchemaID, ContractSchemaVersion, WrkrProducer),
		"expected_family":    "pacf-55f758ded9e42f84", "expected_revision": 1,
		"expected_activation":     ref("activated_action_contract", "gact-4aad73ff9f3c7e5a", ActivationSchemaID, EvidenceSchemaVersion, GaitProducer),
		"expected_runtime_digest": digest, "expected_readiness_digest": digest, "expected_policy_digest": digest, "expected_lifecycle_digest": digest,
		"expected_target": "target:fixture", "expected_environment": "test", "expected_producer_version": FixtureTag, "expected_source_commit": FixtureCommit,
		"evaluation_time": "2026-07-20T00:00:00Z", "activation_not_before": "2026-01-01T00:00:00Z", "activation_not_after": "2027-01-01T00:00:00Z",
		"allow_fixture_only": false,
	}
}

func mustMarshalConfig(value map[string]any) []byte { raw, _ := json.Marshal(value); return raw }
