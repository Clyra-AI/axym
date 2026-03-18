package contracts

import (
	"strings"
	"testing"
)

func TestCommandSurfaceDocsParity(t *testing.T) {
	t.Parallel()

	readme := readRepoFile(t, "README.md")
	commandGuide := readRepoFile(t, "docs/commands/axym.md")
	llmDoc := readRepoFile(t, "docs-site/public/llm/axym.md")
	required := []string{
		"axym init",
		"axym collect",
		"axym map",
		"axym gaps",
		"axym regress",
		"axym bundle",
		"axym review",
		"axym verify",
		"axym record add",
		"axym override create",
		"axym ingest",
		"axym replay",
	}
	for _, command := range required {
		if !strings.Contains(readme, command) {
			t.Fatalf("README missing command %q", command)
		}
		if !strings.Contains(commandGuide, command) {
			t.Fatalf("docs/commands/axym.md missing command %q", command)
		}
		if !strings.Contains(llmDoc, command) {
			t.Fatalf("docs-site/public/llm/axym.md missing command %q", command)
		}
	}
}

func TestCommandInstallSurfaceDocsParity(t *testing.T) {
	t.Parallel()

	readme := readRepoFile(t, "README.md")
	commandGuide := readRepoFile(t, "docs/commands/axym.md")
	llmDoc := readRepoFile(t, "docs-site/public/llm/axym.md")
	required := []string{
		"brew install Clyra-AI/tap/axym",
		"go build ./cmd/axym",
		"./axym version --json",
	}
	for _, snippet := range required {
		if !strings.Contains(readme, snippet) {
			t.Fatalf("README missing install snippet %q", snippet)
		}
		if !strings.Contains(commandGuide, snippet) {
			t.Fatalf("docs/commands/axym.md missing install snippet %q", snippet)
		}
		if !strings.Contains(llmDoc, snippet) {
			t.Fatalf("docs-site/public/llm/axym.md missing install snippet %q", snippet)
		}
	}
}
