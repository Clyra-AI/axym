package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteDefaultAndLoad(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "axym-policy.yaml")
	policy, created, err := WriteDefault(path, false)
	if err != nil {
		t.Fatalf("write default policy: %v", err)
	}
	if !created {
		t.Fatal("expected created=true")
	}
	if policy.Version != VersionV1 {
		t.Fatalf("version mismatch: got %q", policy.Version)
	}
	if got := policy.ResolveStoreDir(""); got != ".axym" {
		t.Fatalf("store default mismatch: got %q", got)
	}
	frameworks := policy.ResolveFrameworks(nil)
	if len(frameworks) != 2 || frameworks[0] != "eu-ai-act" || frameworks[1] != "soc2" {
		t.Fatalf("framework defaults mismatch: %+v", frameworks)
	}
}

func TestLoadRejectsUnknownField(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "axym-policy.yaml")
	content := []byte("version: v1\ndefaults:\n  store_dir: .axym\n  frameworks: [eu-ai-act]\nunknown: true\n")
	if err := osWriteFile(path, content); err != nil {
		t.Fatalf("write policy: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected load error for unknown field")
	}
}

func TestDiscoverReturnsDefaultWhenFileMissing(t *testing.T) {
	t.Parallel()

	oldWD := mustGetwd(t)
	tmp := t.TempDir()
	mustChdir(t, tmp)
	t.Cleanup(func() { mustChdir(t, oldWD) })

	policy, found, err := Discover("")
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if found {
		t.Fatal("expected found=false")
	}
	if got := policy.ResolveStoreDir(""); got != ".axym" {
		t.Fatalf("store dir mismatch: %q", got)
	}
}

func TestResolveFrameworksPreservesPathValues(t *testing.T) {
	t.Parallel()

	absPath := filepath.Join(t.TempDir(), "Custom-Framework.yaml")
	policy := Policy{
		Version: VersionV1,
		Defaults: Defaults{
			StoreDir: ".axym",
			Frameworks: []string{
				"EU-AI-ACT",
				absPath,
				"./fixtures/frameworks/Regress-Minimal.yaml",
			},
		},
	}

	got := policy.ResolveFrameworks(nil)
	want := []string{
		"./fixtures/frameworks/Regress-Minimal.yaml",
		absPath,
		"eu-ai-act",
	}
	if len(got) != len(want) {
		t.Fatalf("framework count mismatch: got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("framework mismatch at %d: got=%q want=%q", i, got[i], want[i])
		}
	}
}

func osWriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return wd
}

func mustChdir(t *testing.T, path string) {
	t.Helper()
	if err := os.Chdir(path); err != nil {
		t.Fatalf("chdir %s: %v", path, err)
	}
}
