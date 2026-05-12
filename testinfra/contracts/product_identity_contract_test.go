package contracts

import (
	"strings"
	"testing"
)

func TestProductIdentityContract(t *testing.T) {
	t.Parallel()

	prd := readRepoFile(t, "product/axym.md")
	required := []string{
		"portable proof of identity-governed action in software delivery",
		"which non-human identity acted, through which delegated chain, against which target, under which policy and approval",
		"`actor_identity`",
		"`downstream_identity`",
		"`delegation_chain`",
		"`policy_digest`",
		"`approval_token_ref`",
		"`owner_identity`",
		"identity-chain summary",
		"ownership/approver register",
		"privilege-drift report",
		"delegated-chain exceptions",
		"not an IAM, PAM, or IGA replacement",
		"Not wider than software delivery.",
		"Truth-in-scope note",
		"`collect --governance-event-file`",
		"`record add`",
	}
	for _, snippet := range required {
		if !strings.Contains(prd, snippet) {
			t.Fatalf("product/axym.md missing identity contract snippet %q", snippet)
		}
	}
}

func TestPublicDocsPreserveSourceOfTruthBoundary(t *testing.T) {
	t.Parallel()

	required := []string{
		"Gait enforces",
		"Wrkr inventories",
		"Proof verifies",
		"Axym correlates",
		"does not block, freeze, broker, or sandbox execution",
	}
	for _, path := range []string{
		"README.md",
		"docs/commands/axym.md",
		"docs/operator/integration-model.md",
		"docs-site/public/llm/axym.md",
	} {
		doc := readRepoFile(t, path)
		for _, snippet := range required {
			if !strings.Contains(doc, snippet) {
				t.Fatalf("%s missing source-of-truth snippet %q", path, snippet)
			}
		}
	}
}
