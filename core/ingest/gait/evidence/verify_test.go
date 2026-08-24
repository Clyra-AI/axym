package evidence

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReleasedFixtureScenarioMatrix(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "testdata", "gait-action-contract-evidence", "v1")
	keyRaw, err := os.ReadFile(filepath.Join(root, "fixture-signing-key.public.b64"))
	if err != nil {
		t.Fatal(err)
	}
	keyBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(keyRaw)))
	if err != nil || len(keyBytes) != ed25519.PublicKeySize {
		t.Fatalf("key: %v", err)
	}
	manifestRaw, err := os.ReadFile(filepath.Join(root, "fixture-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Scenarios []struct {
			ScenarioID, Path, ExpectedReason string
			ExpectedValid                    bool
			EvaluationTime                   string `json:"evaluation_time"`
		} `json:"scenarios"`
	}
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Scenarios) != 9 {
		t.Fatalf("fixture scenario count=%d", len(manifest.Scenarios))
	}
	for _, scenario := range manifest.Scenarios {
		t.Run(scenario.ScenarioID, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(scenario.Path)))
			if err != nil {
				t.Fatal(err)
			}
			pack, err := ParseLifecyclePack(raw)
			if err != nil {
				t.Fatal(err)
			}
			first := pack.Records[0]
			activation := first.ActivationRef
			if activation == nil {
				for _, record := range pack.Records {
					if record.ActivationRef != nil {
						activation = record.ActivationRef
						break
					}
				}
			}
			if activation == nil {
				t.Fatal("missing activation ref")
			}
			now := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
			if scenario.EvaluationTime != "" {
				now, err = time.Parse(time.RFC3339, scenario.EvaluationTime)
				if err != nil {
					t.Fatal(err)
				}
			}
			result := VerifyLifecyclePack(pack, VerificationOptions{
				TrustedPublicKey: ed25519.PublicKey(keyBytes), EvaluationTime: now, AllowFixtureOnly: true,
				ExpectedContract: first.ContractRef, ExpectedFamily: first.ContractFamilyID, ExpectedRevision: first.Revision,
				ExpectedActivation:  *activation,
				ActivationNotBefore: time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
				ActivationNotAfter:  time.Date(2027, 7, 19, 0, 0, 0, 0, time.UTC),
			})
			if result.Valid != scenario.ExpectedValid {
				t.Fatalf("valid=%v want=%v reasons=%v", result.Valid, scenario.ExpectedValid, result.ReasonCodes)
			}
			if !scenario.ExpectedValid && scenario.ExpectedReason != "" && !containsReason(result.ReasonCodes, scenario.ExpectedReason) {
				t.Fatalf("reasons=%v missing %q", result.ReasonCodes, scenario.ExpectedReason)
			}
			if scenario.ExpectedValid && !result.FixtureOnly || scenario.ExpectedValid && result.Authoritative {
				t.Fatalf("fixture authority leak: %+v", result)
			}
		})
	}
}

func TestAxymSourceManifestPinsReleasedFixtureBytes(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "testdata", "gait-action-contract-evidence", "v1")
	manifestRaw, err := os.ReadFile(filepath.Join(root, "SOURCE-MANIFEST.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		SourceTag             string            `json:"source_tag"`
		SourceCommit          string            `json:"source_commit"`
		FixtureManifestSHA256 string            `json:"fixture_manifest_sha256"`
		PublicKeySHA256       string            `json:"public_key_sha256"`
		Files                 map[string]string `json:"files"`
	}
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SourceTag != FixtureTag || manifest.SourceCommit != FixtureCommit || len(manifest.Files) != 11 {
		t.Fatalf("source manifest drift: %+v", manifest)
	}
	fixtureManifest, err := os.ReadFile(filepath.Join(root, "fixture-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if RawDigest(fixtureManifest) != manifest.FixtureManifestSHA256 {
		t.Fatal("fixture manifest digest drift")
	}
	publicKey, err := os.ReadFile(filepath.Join(root, "fixture-signing-key.public.b64"))
	if err != nil {
		t.Fatal(err)
	}
	if RawDigest(publicKey) != manifest.PublicKeySHA256 {
		t.Fatal("public key digest drift")
	}
	for path, digest := range manifest.Files {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		if RawDigest(raw) != digest {
			t.Fatalf("fixture digest drift: %s", path)
		}
	}
}

func TestReleasedFixtureRejectsWrongKeyAndTamper(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "testdata", "gait-action-contract-evidence", "v1")
	raw, err := os.ReadFile(filepath.Join(root, "successful-execution-effect-containment", "lifecycle.json"))
	if err != nil {
		t.Fatal(err)
	}
	pack, err := ParseLifecyclePack(raw)
	if err != nil {
		t.Fatal(err)
	}
	wrong := make(ed25519.PublicKey, ed25519.PublicKeySize)
	if result := VerifyLifecyclePack(pack, VerificationOptions{TrustedPublicKey: wrong, EvaluationTime: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), AllowFixtureOnly: true}); result.Valid || !containsReason(result.ReasonCodes, ReasonSignatureInvalid) {
		t.Fatalf("wrong key result=%+v", result)
	}
	pack.Records[5].Execution.Outcome = "failed"
	if result := VerifyLifecyclePack(pack, VerificationOptions{TrustedPublicKey: keyForTest(t, root), EvaluationTime: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), AllowFixtureOnly: true}); result.Valid {
		t.Fatalf("tampered evidence accepted: %+v", result)
	}
}

func keyForTest(t *testing.T, root string) ed25519.PublicKey {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "fixture-signing-key.public.b64"))
	if err != nil {
		t.Fatal(err)
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	return ed25519.PublicKey(key)
}

func containsReason(reasons []string, want string) bool {
	for _, reason := range reasons {
		if reason == want || strings.Contains(reason, want) {
			return true
		}
	}
	return false
}
