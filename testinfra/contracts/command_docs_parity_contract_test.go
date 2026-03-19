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

func TestLaunchStoryDocsParity(t *testing.T) {
	t.Parallel()

	readme := readRepoFile(t, "README.md")
	commandGuide := readRepoFile(t, "docs/commands/axym.md")
	llmDoc := readRepoFile(t, "docs-site/public/llm/axym.md")
	required := []string{
		"Smoke test",
		"Sample proof path",
		"Real integration path",
		"./axym init --sample-pack ./axym-sample --json",
		"./axym collect --json --governance-event-file ./axym-sample/governance/context_engineering.jsonl",
		"./axym record add --input ./axym-sample/records/approval.json --json",
		"./axym record add --input ./axym-sample/records/risk_assessment.json --json",
		"Built-in collectors",
		"Plugin collectors",
		"Manual record append",
		"Sibling ingest",
	}
	for _, snippet := range required {
		if !strings.Contains(readme, snippet) {
			t.Fatalf("README missing launch snippet %q", snippet)
		}
		if !strings.Contains(commandGuide, snippet) {
			t.Fatalf("docs/commands/axym.md missing launch snippet %q", snippet)
		}
		if !strings.Contains(llmDoc, snippet) {
			t.Fatalf("docs-site/public/llm/axym.md missing launch snippet %q", snippet)
		}
	}
}

func TestOperatorDocsAreLinkedFromLaunchSurfaces(t *testing.T) {
	t.Parallel()

	readme := readRepoFile(t, "README.md")
	commandGuide := readRepoFile(t, "docs/commands/axym.md")
	llmDoc := readRepoFile(t, "docs-site/public/llm/axym.md")
	readmeRequired := []string{
		"docs/operator/quickstart.md",
		"docs/operator/integration-model.md",
	}
	commandGuideRequired := []string{
		"../operator/quickstart.md",
		"../operator/integration-model.md",
	}
	llmDocRequired := []string{
		"../../../docs/operator/quickstart.md",
		"../../../docs/operator/integration-model.md",
	}
	for _, snippet := range readmeRequired {
		if !strings.Contains(readme, snippet) {
			t.Fatalf("README missing operator doc link %q", snippet)
		}
	}
	for _, snippet := range commandGuideRequired {
		if !strings.Contains(commandGuide, snippet) {
			t.Fatalf("docs/commands/axym.md missing operator doc link %q", snippet)
		}
	}
	for _, snippet := range llmDocRequired {
		if !strings.Contains(llmDoc, snippet) {
			t.Fatalf("docs-site/public/llm/axym.md missing operator doc link %q", snippet)
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
