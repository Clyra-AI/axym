package governance

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGaitControlLifecycleArtifactsVerifyOuterAndNestedIntegrity(t *testing.T) {
	pubRaw, e := os.ReadFile(filepath.Join("..", "..", "testdata", "governance", "v1", "gait-control", "../gait-control/fixture-signing-key.public.b64"))
	if e != nil {
		t.Fatal(e)
	}
	pubBytes, e := base64.StdEncoding.DecodeString(string(pubRaw))
	if e != nil {
		t.Fatal(e)
	}
	pub := ed25519.PublicKey(pubBytes)
	base := filepath.Join("..", "..", "testdata", "governance", "v1", "gait-control")
	entries, _ := os.ReadDir(base)
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		raw, e := os.ReadFile(filepath.Join(base, entry.Name(), "lifecycle.json"))
		if e != nil {
			t.Fatal(e)
		}
		res, e := VerifyControlLifecycle(raw, pub)
		if e != nil {
			t.Fatalf("control fixture %s: %v", entry.Name(), e)
		}
		if res.Authoritative || !res.Quarantine {
			t.Fatalf("authority escalation %s", entry.Name())
		}
		count++
	}
	if count != 7 {
		t.Fatalf("control fixture count %d", count)
	}
}

func TestGaitControlLifecycleRejectsTamperWrongKeyAndOrder(t *testing.T) {
	base := filepath.Join("..", "..", "testdata", "governance", "v1", "gait-control")
	publicRaw, err := os.ReadFile(filepath.Join(base, "fixture-signing-key.public.b64"))
	if err != nil {
		t.Fatal(err)
	}
	publicBytes, err := base64.StdEncoding.DecodeString(string(publicRaw))
	if err != nil {
		t.Fatal(err)
	}
	publicKey := ed25519.PublicKey(publicBytes)
	raw, err := os.ReadFile(filepath.Join(base, "stop-requested-acknowledged", "lifecycle.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyControlLifecycle(raw, make(ed25519.PublicKey, ed25519.PublicKeySize)); err == nil {
		t.Fatal("wrong key accepted")
	}
	var document struct {
		Records []map[string]any `json:"records"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	document.Records[len(document.Records)-1]["kind"] = "stop_failed"
	tampered, _ := json.Marshal(document)
	if _, err := VerifyControlLifecycle(tampered, publicKey); err == nil {
		t.Fatal("content tamper accepted")
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	last := document.Records[len(document.Records)-1]
	control := last["control"].(map[string]any)
	control["affected_scope"] = []any{"different"}
	tampered, _ = json.Marshal(document)
	if _, err := VerifyControlLifecycle(tampered, publicKey); err == nil {
		t.Fatal("nested control tamper accepted")
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	document.Records[0], document.Records[1] = document.Records[1], document.Records[0]
	reordered, _ := json.Marshal(document)
	if _, err := VerifyControlLifecycle(reordered, publicKey); err == nil {
		t.Fatal("reordered lifecycle accepted")
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	document.Records = append(document.Records, document.Records[len(document.Records)-1])
	duplicated, _ := json.Marshal(document)
	if _, err := VerifyControlLifecycle(duplicated, publicKey); err == nil {
		t.Fatal("duplicate lifecycle record accepted")
	}
}
