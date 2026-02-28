package collect

import (
	"context"
	"testing"

	"github.com/Clyra-AI/axym/core/collector"
)

type stubCollector struct{ name string }

func (s stubCollector) Name() string { return s.name }
func (s stubCollector) Collect(_ context.Context, _ collector.Request) (collector.Result, error) {
	return collector.Result{}, nil
}

func TestRegistryDeterministicOrdering(t *testing.T) {
	t.Parallel()

	registry := collector.NewRegistry()
	items := []collector.Collector{
		stubCollector{name: "webhook"},
		stubCollector{name: "mcp"},
		stubCollector{name: "llmapi"},
	}
	for _, item := range items {
		if err := registry.Register(item); err != nil {
			t.Fatalf("register collector: %v", err)
		}
	}

	ordered := registry.Ordered()
	got := []string{ordered[0].Name(), ordered[1].Name(), ordered[2].Name()}
	want := []string{"llmapi", "mcp", "webhook"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order mismatch at %d: got %s want %s", i, got[i], want[i])
		}
	}
}
