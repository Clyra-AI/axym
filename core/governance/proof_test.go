package governance

import (
	"github.com/Clyra-AI/axym/core/store"
	"testing"
	"time"
)

func TestAppendProjectionUsesChainDedupe(t *testing.T) {
	st, err := store.New(store.Config{RootDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	rec, err := ToProofRecord("test_result", "judge", "judge", "j", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), map[string]any{"verdict": "pass"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err := AppendProjection(st, rec)
	if err != nil || !a.Appended {
		t.Fatalf("append: %+v %v", a, err)
	}
	b, err := AppendProjection(st, rec)
	if err != nil || !b.Deduped || b.Appended {
		t.Fatalf("dedupe: %+v %v", b, err)
	}
}
