package governance

import (
	"crypto/ed25519"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSignedRegisterUsesInlineDigestAndKeyBinding(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "governance", "v1", "producer-fixture", "action-contract-register.json"))
	if err != nil {
		t.Fatal(err)
	}
	var register Register
	if err := json.Unmarshal(raw, &register); err != nil {
		t.Fatal(err)
	}
	seed := make([]byte, ed25519.SeedSize)
	seed[0] = 1
	priv := ed25519.NewKeyFromSeed(seed)
	signed, err := SignRegister(register, priv)
	if err != nil {
		t.Fatal(err)
	}
	if signed.Signature == nil || signed.Digest == "" {
		t.Fatal("register did not receive inline digest and signature")
	}
	if err := VerifySignedRegister(signed, priv.Public().(ed25519.PublicKey)); err != nil {
		t.Fatal(err)
	}
	signed.Contracts[0].Action = "tampered"
	if err := VerifySignedRegister(signed, priv.Public().(ed25519.PublicKey)); err == nil {
		t.Fatal("tampered signed register accepted")
	}
}
