package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func sum(b []byte) string { s := sha256.Sum256(b); return "sha256:" + hex.EncodeToString(s[:]) }
func main() {
	src := flag.String("source", "/Users/tr/Clyra/gait-doc-completion/testdata/action-contract-evidence/v1", "Gait fixture source")
	dst := flag.String("dest", "testdata/governance/v1/gait-control", "Axym fixture destination")
	update := flag.Bool("update", false, "import fixtures")
	flag.Parse()
	if !*update {
		if err := check(*dst); err != nil {
			panic(err)
		}
		return
	}
	names := []string{"out-of-scope-containment", "stop-requested-acknowledged", "stop-requested-failed", "external-revocation-acknowledged", "external-revocation-failed", "capability-invalidation", "descendant-invalidation"}
	upstreamRaw, err := os.ReadFile(filepath.Join(*src, "fixture-manifest.json"))
	if err != nil {
		panic(err)
	}
	var upstream map[string]any
	if err = json.Unmarshal(upstreamRaw, &upstream); err != nil {
		panic(err)
	}
	meta := map[string]map[string]any{}
	if arr, ok := upstream["scenarios"].([]any); ok {
		for _, v := range arr {
			if x, ok := v.(map[string]any); ok {
				if p, ok := x["path"].(string); ok {
					meta[p] = x
				}
			}
		}
	}
	if err := os.MkdirAll(*dst, 0700); err != nil {
		panic(err)
	}
	entries := []map[string]any{}
	for _, n := range names {
		// #nosec G304 -- scenario names are fixed by the importer and source is explicit operator input.
		b, e := os.ReadFile(filepath.Join(*src, n, "lifecycle.json"))
		if e != nil {
			panic(e)
		}
		p := filepath.Join(*dst, n, "lifecycle.json")
		if e = os.MkdirAll(filepath.Dir(p), 0700); e != nil {
			panic(e)
		}
		// #nosec G304 -- destination is explicit fixture output and scenario path is fixed.
		if e = os.WriteFile(p, b, 0600); e != nil {
			panic(e)
		}
		pname := filepath.ToSlash(filepath.Join(n, "lifecycle.json"))
		entries = append(entries, map[string]any{"scenario_id": n, "path": pname, "sha256": sum(b), "producer_version": "unreleased-control-extension", "generator_sha256": meta[pname]["generator_sha256"], "schema_sha256": meta[pname]["schema_sha256"], "synthetic_extension": true, "quarantine": true, "authoritative": false, "base_commit": "eb4c599a5c1a24dbb270c39a5a513d78f253506d"})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i]["scenario_id"].(string) < entries[j]["scenario_id"].(string) })
	m := map[string]any{"fixture_version": "1", "producer": "gait", "compatibility_base_version": "v1.5.0", "producer_version": "unreleased-control-extension", "source_commit": "eb4c599a5c1a24dbb270c39a5a513d78f253506d", "schema_id": "https://gait.dev/schemas/v1/runtime-lifecycle-record.schema.json", "schema_version": "1", "synthetic_extension": true, "quarantine": true, "authoritative": false, "base_commit": upstream["base_commit"], "scenarios": entries}
	raw, _ := json.MarshalIndent(m, "", "  ")
	if e := os.WriteFile(filepath.Join(*dst, "manifest.json"), append(raw, '\n'), 0600); e != nil {
		panic(e)
	}
	// #nosec G304 -- source is explicit public fixture input.
	k, e := os.ReadFile(filepath.Join(*src, "fixture-signing-key.public.b64"))
	if e != nil {
		panic(e)
	}
	// #nosec G304 -- destination is explicit fixture output.
	if e = os.WriteFile(filepath.Join(*dst, "fixture-signing-key.public.b64"), k, 0600); e != nil {
		panic(e)
	}
}
func check(dst string) error {
	// #nosec G304 -- destination is explicit managed fixture output.
	raw, e := os.ReadFile(filepath.Join(dst, "manifest.json"))
	if e != nil {
		return e
	}
	var m struct {
		Scenarios []struct {
			ScenarioID string `json:"scenario_id"`
			Path       string `json:"path"`
			SHA256     string `json:"sha256"`
		} `json:"scenarios"`
	}
	if e = json.Unmarshal(raw, &m); e != nil {
		return e
	}
	for _, s := range m.Scenarios {
		// #nosec G304 -- paths are read from the checked-in manifest under explicit fixture root.
		b, e := os.ReadFile(filepath.Join(dst, s.Path))
		if e != nil {
			return e
		}
		if sum(b) != s.SHA256 {
			return fmt.Errorf("digest drift: %s", s.ScenarioID)
		}
	}
	return nil
}
