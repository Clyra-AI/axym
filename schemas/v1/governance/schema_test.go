package governance

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEmbeddedSchemasMatchCheckedInAssets(t *testing.T) {
	for _, item := range []struct {
		name     string
		embedded []byte
	}{
		{"action-contract-register.schema.json", RegisterSchema()},
		{"action-contract-evidence-packet.schema.json", PacketSchema()},
	} {
		raw, err := os.ReadFile(filepath.Join(item.name))
		if err != nil {
			t.Fatal(err)
		}
		if string(raw) != string(item.embedded) {
			t.Fatalf("embedded schema drift: %s", item.name)
		}
	}
}
