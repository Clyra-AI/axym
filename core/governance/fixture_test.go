package governance

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestPinnedGovernanceFixtureManifestAndReversedDeterminism(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "governance", "v1", "fixture-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		FixtureVersion string `json:"fixture_version"`
		Sources        map[string]struct {
			Product       string `json:"product"`
			Version       string `json:"version"`
			Commit        string `json:"commit"`
			SchemaID      string `json:"schema_id"`
			SchemaVersion string `json:"schema_version"`
		} `json:"sources"`
		Scenarios []struct{ ID, Path, SHA256, Expected string } `json:"scenarios"`
	}
	if json.Unmarshal(raw, &m) != nil || m.FixtureVersion != "1" || len(m.Sources) != 3 || len(m.Scenarios) < 8 {
		t.Fatalf("invalid fixture manifest")
	}
	for _, s := range m.Sources {
		if s.Product == "" || s.Version == "" || s.Commit == "" || s.SchemaID == "" || s.SchemaVersion == "" {
			t.Fatalf("unpinned source: %+v", s)
		}
	}
	for _, s := range m.Scenarios {
		if s.ID == "" || s.Path == "" || s.SHA256 == "" || s.Expected == "" {
			t.Fatalf("incomplete scenario %+v", s)
		}
		b, e := os.ReadFile(filepath.Join("..", "..", s.Path))
		if e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(b)
		if "sha256:"+hex.EncodeToString(sum[:]) != s.SHA256 {
			t.Fatalf("digest mismatch %s", s.ID)
		}
	}
	base := CompletenessInput{Readiness: true, Preconditions: true, ProposalSeen: true, ActivationSeen: true, ExecutionOutcome: "succeeded", EffectValidated: true, Containment: "completed", AuthorityLineage: true, Fresh: true, CorrelationRefs: 2, CorrelationAuthoritative: true}
	if EvaluateCompleteness(CompletenessInput{Readiness: true, Preconditions: true, ProposalSeen: true, ActivationSeen: true, ExecutionOutcome: "failed", EffectValidated: true, Containment: "completed", CompensationRequired: true, Compensation: "completed", AuthorityLineage: true, Fresh: true, CorrelationRefs: 2, CorrelationAuthoritative: true}).Status != Complete {
		t.Fatal("failed+compensated fixture is not complete")
	}
	if EvaluateCompleteness(CompletenessInput{AuthorityLineage: true, CorrelationRefs: 1, CorrelationAuthoritative: false}).CorrelationConfidence != Low {
		t.Fatal("identifier-only correlation became authoritative")
	}
	if EvaluateCompleteness(CompletenessInput{JudgeOnly: true}).Status != Unverifiable {
		t.Fatal("Judge-only fixture became authoritative")
	}
	a := []Event{{ID: "z", ContractRef: Ref{ID: "c"}, Kind: "execution_started", OccurredAt: "2026-01-03T00:00:00Z"}, {ID: "a", ContractRef: Ref{ID: "c"}, Kind: "execution_succeeded", OccurredAt: "2026-01-04T00:00:00Z"}, {ID: "p", ContractRef: Ref{ID: "c"}, Kind: "proposed", OccurredAt: "2026-01-01T00:00:00Z"}, {ID: "v", ContractRef: Ref{ID: "c"}, Kind: "activated", OccurredAt: "2026-01-02T00:00:00Z"}}
	b := append([]Event(nil), a...)
	sort.Slice(b, func(i, j int) bool { return b[i].ID > b[j].ID })
	sa, ea := Reduce("c", a)
	sb, eb := Reduce("c", b)
	if ea != nil || eb != nil || sa.Status != sb.Status || EvaluateCompleteness(base).Status != Complete {
		t.Fatalf("reversed evaluation drift")
	}
}

func TestGaitGateFixtureAuthorityInputsRemainQuarantined(t *testing.T) {
	for _, name := range []string{"approval-exact.json", "approval-expired.json", "delegation-root.json", "delegation-child-tightened.json", "invalid/action-expansion.json", "invalid/wrong-parent-digest.json", "invalid/revoked-ancestor.json"} {
		b, e := os.ReadFile(filepath.Join("..", "..", "testdata", "governance", "v1", "gait-gate", name))
		if e != nil {
			t.Fatal(e)
		}
		var v map[string]any
		if json.Unmarshal(b, &v) != nil {
			t.Fatalf("malformed gate artifact %s", name)
		}
		if v["execution_authority"] == true || v["authoritative"] == true {
			t.Fatalf("gate artifact escalates authority: %s", name)
		}
	}
}

func TestGaitGateArtifactsVerifyWithAxymIntegrityBoundary(t *testing.T) {
	keyRaw, e := os.ReadFile(filepath.Join("..", "..", "testdata", "governance", "v1", "gait-gate", "fixture-signing-key.public.b64"))
	if e != nil {
		t.Fatal(e)
	}
	pubBytes, e := base64.StdEncoding.DecodeString(string(keyRaw))
	if e != nil {
		t.Fatal(e)
	}
	pub := ed25519.PublicKey(pubBytes)
	rootRaw, _ := os.ReadFile(filepath.Join("..", "..", "testdata", "governance", "v1", "gait-gate", "delegation-root.json"))
	var root map[string]any
	_ = json.Unmarshal(rootRaw, &root)
	for _, name := range []string{"approval-exact.json", "approval-expired.json", "delegation-root.json", "delegation-child-tightened.json", "invalid/action-expansion.json", "invalid/data-expansion.json", "invalid/environment-expansion.json", "invalid/max-depth-expansion.json", "invalid/max-ops-expansion.json", "invalid/max-targets-expansion.json", "invalid/network-expansion.json", "invalid/revoked-ancestor.json", "invalid/target-expansion.json", "invalid/ttl-expansion.json", "invalid/wrong-origin-authority.json", "invalid/wrong-parent-digest.json"} {
		raw, e := os.ReadFile(filepath.Join("..", "..", "testdata", "governance", "v1", "gait-gate", name))
		if e != nil {
			t.Fatal(e)
		}
		parent := map[string]any(nil)
		if strings.HasPrefix(name, "invalid/") || name == "delegation-child-tightened.json" {
			parent = root
		}
		_, e = VerifyGateArtifact(raw, pub, time.Date(2026, 8, 1, 12, 30, 0, 0, time.UTC), parent)
		if name == "approval-expired.json" {
			if e == nil || e.Error() != ReasonGateExpired {
				t.Fatalf("expired classification: %v", e)
			}
		} else if name == "approval-exact.json" && e != nil {
			t.Fatalf("exact gate rejected: %v", e)
		} else if name == "delegation-child-tightened.json" && e != nil {
			t.Fatalf("tightened child rejected: %v", e)
		} else if strings.HasPrefix(name, "invalid/") && e == nil {
			t.Fatalf("invalid gate accepted: %s", name)
		}
	}
}

func TestGaitControlFixtureManifestIsPinnedAndQuarantined(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "governance", "v1", "gait-control", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		SyntheticExtension bool `json:"synthetic_extension"`
		Quarantine         bool `json:"quarantine"`
		Authoritative      bool `json:"authoritative"`
		Scenarios          []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"scenarios"`
	}
	if json.Unmarshal(raw, &m) != nil || !m.SyntheticExtension || !m.Quarantine || m.Authoritative || len(m.Scenarios) != 7 {
		t.Fatalf("invalid control fixture manifest")
	}
	for _, s := range m.Scenarios {
		b, e := os.ReadFile(filepath.Join("..", "..", "testdata", "governance", "v1", "gait-control", s.Path))
		if e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(b)
		if "sha256:"+hex.EncodeToString(sum[:]) != s.SHA256 {
			t.Fatalf("control fixture digest drift: %s", s.Path)
		}
	}
}

func TestGaitGateFixtureImportIsPinnedAndQuarantined(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "governance", "v1", "gait-gate", "upstream-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Quarantine    bool `json:"quarantine"`
		Authoritative bool `json:"authoritative"`
		Files         []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"files"`
	}
	if json.Unmarshal(raw, &m) != nil || !m.Quarantine || m.Authoritative || len(m.Files) < 16 {
		t.Fatalf("invalid gate manifest")
	}
	for _, f := range m.Files {
		b, e := os.ReadFile(filepath.Join("..", "..", "testdata", "governance", "v1", "gait-gate", f.Path))
		if e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(b)
		if "sha256:"+hex.EncodeToString(sum[:]) != f.SHA256 {
			t.Fatalf("gate digest drift: %s", f.Path)
		}
	}
}
