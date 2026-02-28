package collect

import (
	"sort"
	"strings"
)

func uniqueSorted(in []string) []string {
	set := map[string]struct{}{}
	for _, value := range in {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
