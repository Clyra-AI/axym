package framework

import "testing"

func TestLoadManyDeterministicAndStrict(t *testing.T) {
	t.Parallel()

	frameworks, err := LoadMany([]string{"soc2", "eu-ai-act", "soc2"})
	if err != nil {
		t.Fatalf("LoadMany: %v", err)
	}
	if len(frameworks) != 2 {
		t.Fatalf("framework count mismatch: %d", len(frameworks))
	}
	if frameworks[0].ID != "eu-ai-act" || frameworks[1].ID != "soc2" {
		t.Fatalf("framework order mismatch: %+v", frameworks)
	}
	if len(frameworks[0].Controls) == 0 {
		t.Fatalf("expected controls for %s", frameworks[0].ID)
	}
}

func TestLoadManyMissingFramework(t *testing.T) {
	t.Parallel()

	_, err := LoadMany([]string{"does-not-exist"})
	if err == nil {
		t.Fatalf("expected error")
	}
	loadErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("unexpected error type: %T", err)
	}
	if loadErr.ReasonCode != ReasonFrameworkLoad {
		t.Fatalf("reason mismatch: %s", loadErr.ReasonCode)
	}
}
