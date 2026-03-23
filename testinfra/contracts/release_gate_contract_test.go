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
		"checksums.txt.pem",
		"local-cosign.pub",
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

func TestReleaseWorkflowSmokeBinaryStaysOutOfDistRoot(t *testing.T) {
	t.Parallel()

	release := readRepoFile(t, ".github/workflows/release.yml")
	required := []string{
		"go build -o .tmp/axym-release-smoke ./cmd/axym",
		"--release-binary .tmp/axym-release-smoke",
	}
	for _, snippet := range required {
		if !strings.Contains(release, snippet) {
			t.Fatalf("release workflow missing smoke-binary safeguard %q", snippet)
		}
	}
	if strings.Contains(release, "go build -o dist/axym ./cmd/axym") {
		t.Fatal("release workflow must not build the smoke binary into dist/")
	}
}

func TestReleaseWorkflowUsesPinnedToolingAndHostedVerificationPaths(t *testing.T) {
	t.Parallel()

	release := readRepoFile(t, ".github/workflows/release.yml")
	required := []string{
		"version: v2.14.1",
		"id-token: write",
		"cosign sign-blob --yes --output-signature dist/checksums.txt.sig --output-certificate dist/checksums.txt.pem dist/checksums.txt",
		"actions/attest-build-provenance@v4",
		"dist/github-attestation.provenance.json",
	}
	for _, snippet := range required {
		if !strings.Contains(release, snippet) {
			t.Fatalf("release workflow missing hosted verification snippet %q", snippet)
		}
	}
}
