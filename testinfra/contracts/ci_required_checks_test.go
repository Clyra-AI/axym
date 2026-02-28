package contracts

import (
	"strings"
	"testing"
)

func TestPRWorkflowHasConcurrencyCancelInProgress(t *testing.T) {
	t.Parallel()

	pr := readRepoFile(t, ".github/workflows/pr.yml")
	if !strings.Contains(pr, "concurrency:") || !strings.Contains(pr, "cancel-in-progress: true") {
		t.Fatal("pr workflow must enforce concurrency cancel-in-progress")
	}
}

func TestRequiredChecksMapToPRTriggeredWorkflow(t *testing.T) {
	t.Parallel()

	pr := readRepoFile(t, ".github/workflows/pr.yml")
	main := readRepoFile(t, ".github/workflows/main.yml")

	if !strings.Contains(pr, "pull_request:") {
		t.Fatal("required checks must be emitted by pull_request workflows")
	}

	requiredJobs := []string{"lint-fast:", "test-fast:", "test-contracts:"}
	for _, job := range requiredJobs {
		if !strings.Contains(pr, job) {
			t.Fatalf("pr workflow missing required check job: %s", job)
		}
	}

	// Prevent accidental coupling to push-only workflow status names.
	if strings.Contains(main, "name: required-checks") {
		t.Fatal("main workflow should not define pull-request required-check aliases")
	}
}

func TestReleaseWorkflowContainsIntegrityGates(t *testing.T) {
	t.Parallel()

	release := readRepoFile(t, ".github/workflows/release.yml")
	requiredSteps := []string{
		"Verify checksums",
		"Generate SBOM",
		"Vulnerability scan",
		"Sign",
		"Provenance",
		"Verify release integrity",
	}
	for _, step := range requiredSteps {
		if !strings.Contains(release, step) {
			t.Fatalf("release workflow missing integrity gate: %s", step)
		}
	}
}
