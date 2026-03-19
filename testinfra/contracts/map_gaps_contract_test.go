package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMapAndGapsJSONEnvelopeContract(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	stdout, exit := runAxymContract(t, "collect", "--fixture-dir", filepath.Join(testRepoRoot(t), "fixtures", "collectors"), "--store-dir", storeDir, "--json")
	if exit != 0 {
		t.Fatalf("collect setup failed: exit=%d output=%s", exit, stdout)
	}

	mapOut, mapExit := runAxymContract(t, "map", "--frameworks", "eu-ai-act,soc2", "--store-dir", storeDir, "--json")
	if mapExit != 0 {
		t.Fatalf("map exit mismatch: %d output=%s", mapExit, mapOut)
	}
	var mapPayload map[string]any
	if err := json.Unmarshal([]byte(mapOut), &mapPayload); err != nil {
		t.Fatalf("decode map output: %v", err)
	}
	if mapPayload["command"] != "map" {
		t.Fatalf("map command mismatch: %s", mapOut)
	}
	if mapPayload["ok"] != true {
		t.Fatalf("map expected ok=true: %s", mapOut)
	}
	mapData, _ := mapPayload["data"].(map[string]any)
	if _, ok := mapData["frameworks"].([]any); !ok {
		t.Fatalf("map missing frameworks data: %s", mapOut)
	}

	gapsOut, gapsExit := runAxymContract(t, "gaps", "--frameworks", "eu-ai-act,soc2", "--store-dir", storeDir, "--json")
	if gapsExit != 0 {
		t.Fatalf("gaps exit mismatch: %d output=%s", gapsExit, gapsOut)
	}
	var gapsPayload map[string]any
	if err := json.Unmarshal([]byte(gapsOut), &gapsPayload); err != nil {
		t.Fatalf("decode gaps output: %v", err)
	}
	if gapsPayload["command"] != "gaps" {
		t.Fatalf("gaps command mismatch: %s", gapsOut)
	}
	gapsData, _ := gapsPayload["data"].(map[string]any)
	if _, ok := gapsData["gaps"].([]any); !ok {
		t.Fatalf("gaps missing ranked list: %s", gapsOut)
	}
	if _, ok := gapsData["grade"].(map[string]any); !ok {
		t.Fatalf("gaps missing grade data: %s", gapsOut)
	}
}

func TestMapAndGapsDoNotCreateSigningKeysOnFreshStore(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mapStore := filepath.Join(root, "map-store")
	mapOut, mapExit := runAxymContract(t, "map", "--store-dir", mapStore, "--json")
	if mapExit != 0 {
		t.Fatalf("map exit mismatch: %d output=%s", mapExit, mapOut)
	}
	if _, err := os.Stat(filepath.Join(mapStore, "signing-key.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no map signing key side effect, got err=%v", err)
	}

	gapsStore := filepath.Join(root, "gaps-store")
	gapsOut, gapsExit := runAxymContract(t, "gaps", "--store-dir", gapsStore, "--json")
	if gapsExit != 0 {
		t.Fatalf("gaps exit mismatch: %d output=%s", gapsExit, gapsOut)
	}
	if _, err := os.Stat(filepath.Join(gapsStore, "signing-key.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no gaps signing key side effect, got err=%v", err)
	}
}
