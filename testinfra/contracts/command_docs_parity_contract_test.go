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

	contract := loadFirstValueSampleContract(t)
	required := append(launchStoryBoundarySnippets(), launchStoryCommandSnippets(contract)...)
	required = append(required, launchStoryOutcomeSnippets(contract)...)
	required = append(required, contract.RequiredArtifacts...)

	for _, doc := range loadLaunchNarrativeDocs(t) {
		requireDocContainsAll(t, doc, required)
		requireDocOmitsAll(t, doc, staleLaunchStorySnippets())
	}
}

func TestOperatorDocsAreLinkedFromLaunchSurfaces(t *testing.T) {
	t.Parallel()

	readme := readRepoFile(t, "README.md")
	commandGuide := readRepoFile(t, "docs/commands/axym.md")
	llmDoc := readRepoFile(t, "docs-site/public/llm/axym.md")
	llmIndex := readRepoFile(t, "docs-site/public/llms.txt")
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
	for _, snippet := range readmeRequired {
		if !strings.Contains(llmIndex, snippet) {
			t.Fatalf("docs-site/public/llms.txt missing operator doc link %q", snippet)
		}
	}
}

func TestCommandInstallSurfaceDocsParity(t *testing.T) {
	t.Parallel()

	for _, doc := range loadLaunchNarrativeDocs(t) {
		requireDocContainsAll(t, doc, launchInstallSurfaceSnippets())
	}
}

func TestLaunchStoryDocsSequence(t *testing.T) {
	t.Parallel()

	contract := loadFirstValueSampleContract(t)
	for _, doc := range loadLaunchNarrativeDocs(t) {
		requireDocOrdering(t, doc, launchStoryOrderedSnippets(contract))
	}
}

func TestLaunchDocsIndexReferencesSourceOfTruth(t *testing.T) {
	t.Parallel()

	llmIndex := readRepoFile(t, "docs-site/public/llms.txt")
	required := []string{
		"LICENSE",
		"CHANGELOG.md",
		"CODE_OF_CONDUCT.md",
		"CONTRIBUTING.md",
		"SECURITY.md",
		"README.md",
		"docs/commands/axym.md",
		"docs-site/public/llm/axym.md",
		"docs/operator/quickstart.md",
		"docs/operator/integration-model.md",
		"docs/operator/integration-boundary.mmd",
		".github/ISSUE_TEMPLATE/bug_report.yml",
		".github/ISSUE_TEMPLATE/feature_request.yml",
		".github/pull_request_template.md",
	}
	for _, snippet := range required {
		if !strings.Contains(llmIndex, snippet) {
			t.Fatalf("docs-site/public/llms.txt missing source-of-truth snippet %q", snippet)
		}
	}
}
