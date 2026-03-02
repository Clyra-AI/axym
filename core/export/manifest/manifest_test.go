package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildDeterministicOrdering(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("b"), 0o600); err != nil {
		t.Fatalf("WriteFile b: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a"), 0o600); err != nil {
		t.Fatalf("WriteFile a: %v", err)
	}

	left, err := Build(root, []string{"b.txt", "a.txt"})
	if err != nil {
		t.Fatalf("Build left: %v", err)
	}
	right, err := Build(root, []string{"a.txt", "b.txt"})
	if err != nil {
		t.Fatalf("Build right: %v", err)
	}

	leftRaw, err := json.Marshal(left)
	if err != nil {
		t.Fatalf("Marshal left: %v", err)
	}
	rightRaw, err := json.Marshal(right)
	if err != nil {
		t.Fatalf("Marshal right: %v", err)
	}
	if string(leftRaw) != string(rightRaw) {
		t.Fatalf("manifest not deterministic: left=%s right=%s", string(leftRaw), string(rightRaw))
	}
}
