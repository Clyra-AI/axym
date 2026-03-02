package contracts

import (
	"strings"
	"testing"
)

func TestCommandSurfaceDocsParity(t *testing.T) {
	t.Parallel()

	readme := readRepoFile(t, "README.md")
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
		if !strings.Contains(llmDoc, command) {
			t.Fatalf("docs-site/public/llm/axym.md missing command %q", command)
		}
	}
}
