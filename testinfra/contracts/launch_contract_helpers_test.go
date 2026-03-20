package contracts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type firstValueSampleContract struct {
	Frameworks                  string   `json:"frameworks"`
	SamplePackPath              string   `json:"sample_pack_path"`
	GovernanceEventCaptureCount int      `json:"governance_event_capture_count"`
	FinalChainRecordCount       int      `json:"final_chain_record_count"`
	CoveredControlCount         int      `json:"covered_control_count"`
	ControlCount                int      `json:"control_count"`
	Grade                       string   `json:"grade"`
	BundleComplete              bool     `json:"bundle_complete"`
	WeakRecordCount             int      `json:"weak_record_count"`
	ChainIntact                 bool     `json:"chain_intact"`
	RemainingGapControlIDs      []string `json:"remaining_gap_control_ids"`
	RequiredArtifacts           []string `json:"required_bundle_artifacts"`
}

type launchDocFixture struct {
	path    string
	content string
}

func loadFirstValueSampleContract(t *testing.T) firstValueSampleContract {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(testRepoRoot(t), "scenarios", "axym", "first_value_sample", "contract.json"))
	if err != nil {
		t.Fatalf("read first-value sample contract: %v", err)
	}

	var contract firstValueSampleContract
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatalf("decode first-value sample contract: %v", err)
	}
	if contract.Frameworks == "" || contract.SamplePackPath == "" {
		t.Fatalf("first-value sample contract must declare frameworks and sample_pack_path: %+v", contract)
	}
	if contract.GovernanceEventCaptureCount <= 0 || contract.FinalChainRecordCount <= 0 {
		t.Fatalf("first-value sample contract must declare positive capture and chain counts: %+v", contract)
	}
	if contract.CoveredControlCount <= 0 || contract.ControlCount <= 0 {
		t.Fatalf("first-value sample contract must declare positive coverage counts: %+v", contract)
	}
	if contract.Grade == "" {
		t.Fatalf("first-value sample contract must declare a grade: %+v", contract)
	}
	if len(contract.RequiredArtifacts) == 0 {
		t.Fatalf("first-value sample contract must declare required artifacts: %+v", contract)
	}

	return contract
}

func loadLaunchNarrativeDocs(t *testing.T) []launchDocFixture {
	t.Helper()

	return []launchDocFixture{
		readLaunchDoc(t, "README.md"),
		readLaunchDoc(t, "docs/commands/axym.md"),
		readLaunchDoc(t, "docs/operator/quickstart.md"),
		readLaunchDoc(t, "docs-site/public/llm/axym.md"),
		readLaunchDoc(t, "docs-site/public/llms.txt"),
	}
}

func readLaunchDoc(t *testing.T, rel string) launchDocFixture {
	t.Helper()

	return launchDocFixture{
		path:    rel,
		content: readRepoFile(t, rel),
	}
}

func requireDocContainsAll(t *testing.T, doc launchDocFixture, snippets []string) {
	t.Helper()

	for _, snippet := range snippets {
		if !strings.Contains(doc.content, snippet) {
			t.Fatalf("%s missing snippet %q", doc.path, snippet)
		}
	}
}

func requireDocOmitsAll(t *testing.T, doc launchDocFixture, snippets []string) {
	t.Helper()

	for _, snippet := range snippets {
		if strings.Contains(doc.content, snippet) {
			t.Fatalf("%s contains stale snippet %q", doc.path, snippet)
		}
	}
}

func requireDocOrdering(t *testing.T, doc launchDocFixture, snippets []string) {
	t.Helper()

	offset := 0
	for _, snippet := range snippets {
		index := strings.Index(doc.content[offset:], snippet)
		if index < 0 {
			t.Fatalf("%s missing ordered snippet %q", doc.path, snippet)
		}
		offset += index + len(snippet)
	}
}

func launchInstallSurfaceSnippets() []string {
	return []string{
		"Homebrew:\n\n```bash\nbrew install Clyra-AI/tap/axym\naxym version --json\n```",
		"Source:\n\n```bash\ngo build ./cmd/axym\n./axym version --json\n```",
		"Release binary:\n\n```bash\n./axym version --json\n```",
		"If you installed via Homebrew, replace `./axym` with `axym` in the commands below.",
	}
}

func launchStoryBoundarySnippets() []string {
	return []string{
		"Smoke test",
		"Sample proof path",
		"Real integration path",
		"Built-in collectors",
		"Plugin collectors",
		"Manual record append",
		"Sibling ingest",
		"First value is evidence + ranked gaps + intact local verification, not full audit completeness.",
	}
}

func launchStoryCommandSnippets(contract firstValueSampleContract) []string {
	governancePath := strings.TrimSuffix(contract.SamplePackPath, "/") + "/governance/context_engineering.jsonl"
	approvalPath := strings.TrimSuffix(contract.SamplePackPath, "/") + "/records/approval.json"
	riskPath := strings.TrimSuffix(contract.SamplePackPath, "/") + "/records/risk_assessment.json"

	return []string{
		fmt.Sprintf("./axym init --sample-pack %s --json", contract.SamplePackPath),
		fmt.Sprintf("./axym collect --json --governance-event-file %s", governancePath),
		fmt.Sprintf("./axym record add --input %s --json", approvalPath),
		fmt.Sprintf("./axym record add --input %s --json", riskPath),
		fmt.Sprintf("./axym map --frameworks %s --json", contract.Frameworks),
		fmt.Sprintf("./axym gaps --frameworks %s --json", contract.Frameworks),
		fmt.Sprintf("./axym bundle --audit sample --frameworks %s --json", contract.Frameworks),
		"./axym verify --chain --json",
	}
}

func launchStoryOutcomeSnippets(contract firstValueSampleContract) []string {
	remainingGap := "the remaining sample gap"
	if len(contract.RemainingGapControlIDs) == 1 {
		remainingGap = fmt.Sprintf("SOC 2 `%s` as the remaining sample gap", contract.RemainingGapControlIDs[0])
	}

	return []string{
		"The sample pack is created locally with no network dependency and no repo fixture dependency.",
		fmt.Sprintf("`collect` captures `%d` governance events from the bundled sample pack.", contract.GovernanceEventCaptureCount),
		fmt.Sprintf("The local chain ends with `%d` total records after the manual approval and risk assessment append.", contract.FinalChainRecordCount),
		fmt.Sprintf("`map` reports `%d` covered controls out of `%d` across `%s`.", contract.CoveredControlCount, contract.ControlCount, contract.Frameworks),
		fmt.Sprintf("`gaps` reports grade `%s`, leaving %s.", contract.Grade, remainingGap),
		fmt.Sprintf("`bundle` emits identity-governance artifacts, keeps compliance incomplete (`complete=%t`), and leaves `weak_record_count=%d`.", contract.BundleComplete, contract.WeakRecordCount),
		fmt.Sprintf("`verify --chain --json` reports an intact `%d`-record chain.", contract.FinalChainRecordCount),
	}
}

func staleLaunchStorySnippets() []string {
	return []string{
		"appends 3 governance-event-derived records",
		"3 governance-event decisions",
		"5 total records to the local chain",
		"contains 5 records",
		"intact 5-record chain",
	}
}

func launchStoryOrderedSnippets(contract firstValueSampleContract) []string {
	governancePath := strings.TrimSuffix(contract.SamplePackPath, "/") + "/governance/context_engineering.jsonl"
	approvalPath := strings.TrimSuffix(contract.SamplePackPath, "/") + "/records/approval.json"
	riskPath := strings.TrimSuffix(contract.SamplePackPath, "/") + "/records/risk_assessment.json"

	return []string{
		"Homebrew:",
		"brew install Clyra-AI/tap/axym",
		"axym version --json",
		"Source:",
		"go build ./cmd/axym",
		"./axym version --json",
		"Release binary:",
		"Smoke test",
		"./axym init --json",
		"./axym collect --dry-run --json",
		"Sample proof path",
		fmt.Sprintf("./axym init --sample-pack %s --json", contract.SamplePackPath),
		fmt.Sprintf("./axym collect --json --governance-event-file %s", governancePath),
		fmt.Sprintf("./axym record add --input %s --json", approvalPath),
		fmt.Sprintf("./axym record add --input %s --json", riskPath),
		fmt.Sprintf("./axym map --frameworks %s --json", contract.Frameworks),
		fmt.Sprintf("./axym gaps --frameworks %s --json", contract.Frameworks),
		fmt.Sprintf("./axym bundle --audit sample --frameworks %s --json", contract.Frameworks),
		"./axym verify --chain --json",
		"Real integration path",
		"Built-in collectors",
	}
}
