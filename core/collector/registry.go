package collector

import (
	"fmt"
	"sort"
	"strings"
)

type Registry struct {
	collectors map[string]Collector
}

func NewRegistry() *Registry {
	return &Registry{collectors: map[string]Collector{}}
}

func (r *Registry) Register(c Collector) error {
	if c == nil {
		return fmt.Errorf("collector is required")
	}
	name := strings.TrimSpace(c.Name())
	if name == "" {
		return fmt.Errorf("collector name is required")
	}
	if _, exists := r.collectors[name]; exists {
		return fmt.Errorf("collector %q already registered", name)
	}
	r.collectors[name] = c
	return nil
}

func (r *Registry) Ordered() []Collector {
	names := make([]string, 0, len(r.collectors))
	for name := range r.collectors {
		names = append(names, name)
	}
	sort.Strings(names)
	ordered := make([]Collector, 0, len(names))
	for _, name := range names {
		ordered = append(ordered, r.collectors[name])
	}
	return ordered
}
