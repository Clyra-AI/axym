package contracts

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseGoNoGoScriptSyntaxAndCoverage(t *testing.T) {
	t.Parallel()

	scriptPath := filepath.Join(testRepoRoot(t), "scripts", "release_go_nogo.sh")
	if out, err := exec.Command("bash", "-n", scriptPath).CombinedOutput(); err != nil {
		t.Fatalf("bash -n release_go_nogo.sh: %v output=%s", err, string(out))
	}

	script := readRepoFile(t, "scripts/release_go_nogo.sh")
	required := []string{
		"sha256sum -c",
		"cosign verify-blob",
		"brew install Clyra-AI/tap/axym",
		"go build -o",
		"version --json",
	}
	for _, snippet := range required {
		if !strings.Contains(script, snippet) {
			t.Fatalf("release_go_nogo.sh missing snippet %q", snippet)
		}
	}
}

func TestGoReleaserDefinesHomebrewTap(t *testing.T) {
	t.Parallel()

	content := readRepoFile(t, ".goreleaser.yaml")
	required := []string{
		"brews:",
		"owner: Clyra-AI",
		"name: tap",
	}
	for _, snippet := range required {
		if !strings.Contains(content, snippet) {
			t.Fatalf(".goreleaser.yaml missing Homebrew config snippet %q", snippet)
		}
	}
}
