package redact

import "testing"

func TestApplySupportsHashMaskOmit(t *testing.T) {
	t.Parallel()

	event := map[string]any{
		"token": "s3cr3t",
		"body": map[string]any{
			"prompt": "unsafe",
		},
		"drop_me": "bye",
	}
	metadata := map[string]any{"owner": "alice"}

	eventOut, metadataOut, err := Apply(event, metadata, Config{
		EventRules: []Rule{
			{Path: "token", Action: ActionHash},
			{Path: "body.prompt", Action: ActionMask},
			{Path: "drop_me", Action: ActionOmit},
		},
		MetadataRules: []Rule{{Path: "owner", Action: ActionMask}},
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if eventOut["token"] == "s3cr3t" {
		t.Fatal("token was not hashed")
	}
	if _, exists := eventOut["drop_me"]; exists {
		t.Fatal("drop_me should be omitted")
	}
	body := eventOut["body"].(map[string]any)
	if body["prompt"] != "***" {
		t.Fatalf("masked value mismatch: got %v", body["prompt"])
	}
	if metadataOut["owner"] != "***" {
		t.Fatalf("metadata mask mismatch: got %v", metadataOut["owner"])
	}
	if event["token"] != "s3cr3t" {
		t.Fatal("input event mutated")
	}
}

func TestApplyHashIsDeterministic(t *testing.T) {
	t.Parallel()

	event := map[string]any{"value": map[string]any{"a": 1.0, "b": "x"}}

	outA, _, err := Apply(event, nil, Config{EventRules: []Rule{{Path: "value", Action: ActionHash}}})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	outB, _, err := Apply(event, nil, Config{EventRules: []Rule{{Path: "value", Action: ActionHash}}})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if outA["value"] != outB["value"] {
		t.Fatalf("hash mismatch across runs: %v vs %v", outA["value"], outB["value"])
	}
}

func TestApplyRejectsUnsupportedAction(t *testing.T) {
	t.Parallel()

	_, _, err := Apply(map[string]any{"x": 1}, nil, Config{EventRules: []Rule{{Path: "x", Action: Action("rotate")}}})
	if err == nil {
		t.Fatal("expected unsupported action error")
	}
}

func TestApplyRedactsArrayElements(t *testing.T) {
	event := map[string]any{"items": []any{map[string]any{"token": "one"}, map[string]any{"token": "two"}}}
	got, _, err := Apply(event, nil, Config{EventRules: []Rule{{Path: "items.*.token", Action: ActionMask}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range got["items"].([]any) {
		if item.(map[string]any)["token"] != "***" {
			t.Fatalf("array item was not redacted: %#v", item)
		}
	}
}

func TestApplyActionContractRefsAndSecretsInNestedArrays(t *testing.T) {
	event := map[string]any{"evidence": []any{map[string]any{"trace_id": "trace-secret", "provenance": map[string]any{"token": "secret"}, "ref": "stable"}}}
	got, _, err := Apply(event, nil, Config{EventRules: []Rule{{Path: "evidence.*.trace_id", Action: ActionHash}, {Path: "evidence.*.provenance.token", Action: ActionOmit}}})
	if err != nil {
		t.Fatal(err)
	}
	item := got["evidence"].([]any)[0].(map[string]any)
	if item["trace_id"] == "trace-secret" {
		t.Fatal("trace id not hashed")
	}
	if _, ok := item["provenance"].(map[string]any)["token"]; ok {
		t.Fatal("provenance secret retained")
	}
	if item["ref"] != "stable" {
		t.Fatal("non-sensitive reference changed")
	}
}
