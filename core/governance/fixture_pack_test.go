package governance

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	governanceschema "github.com/Clyra-AI/axym/schemas/v1/governance"
)

func TestCheckedInProducerFixturePackIsPinnedAndQuarantined(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "governance", "v1", "producer-fixture")
	var manifest struct {
		FixtureVersion string `json:"fixture_version"`
		Quarantine     bool   `json:"quarantine"`
		Authoritative  bool   `json:"authoritative"`
		Sources        map[string]struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"sources"`
		Files []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"files"`
	}
	raw, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &manifest); err != nil || manifest.FixtureVersion != "1" || !manifest.Quarantine || manifest.Authoritative || len(manifest.Files) != 5 || len(manifest.Sources) < 6 {
		t.Fatalf("invalid fixture manifest: %s", raw)
	}
	for _, file := range manifest.Files {
		payload, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(file.Path)))
		if readErr != nil {
			t.Fatal(readErr)
		}
		sum := sha256.Sum256(payload)
		if got := "sha256:" + hex.EncodeToString(sum[:]); got != file.SHA256 {
			t.Fatalf("fixture drift %s: %s != %s", file.Path, got, file.SHA256)
		}
	}
	for name, source := range manifest.Sources {
		payload, readErr := os.ReadFile(filepath.Join("..", "..", source.Path))
		if readErr != nil {
			t.Fatalf("source %s: %v", name, readErr)
		}
		sum := sha256.Sum256(payload)
		if got := "sha256:" + hex.EncodeToString(sum[:]); got != source.SHA256 {
			t.Fatalf("source drift %s", name)
		}
	}
	registerRaw, err := os.ReadFile(filepath.Join(root, "action-contract-register.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = governanceschema.ValidateRegister(registerRaw); err != nil {
		t.Fatal(err)
	}
	packetRaw, err := os.ReadFile(filepath.Join(root, "action-contract-evidence-packet.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = governanceschema.ValidatePacket(packetRaw); err != nil {
		t.Fatal(err)
	}
	var packet Packet
	if err = json.Unmarshal(packetRaw, &packet); err != nil {
		t.Fatal(err)
	}
	keyRaw, err := os.ReadFile(filepath.Join(root, "fixture-signing-key.public.b64"))
	if err != nil {
		t.Fatal(err)
	}
	public, err := base64.StdEncoding.DecodeString(string(keyRaw))
	if err != nil || len(public) != ed25519.PublicKeySize {
		t.Fatalf("invalid fixture key: %v", err)
	}
	if err = VerifySignedPacket(packet, ed25519.PublicKey(public)); err != nil {
		t.Fatalf("fixture packet signature: %v", err)
	}
	if packet.Contract.ID != "pac-4b7f1402784256ce" || packet.Contract.Provenance.Digest != "sha256:bfb32cdce650b2ea969059ae0816df2637f7345e70b08a67d4c23684489bf154" || len(packet.Evidence) < 17 {
		t.Fatalf("fixture does not preserve cross-product joins: %+v", packet.Contract)
	}
}
