//go:build gatefixture

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func sum(b []byte) string { s := sha256.Sum256(b); return "sha256:" + hex.EncodeToString(s[:]) }
func main() {
	src := flag.String("source", "/Users/tr/Clyra/gait-doc-completion/testdata/action-contract-gate/v1", "source")
	dst := flag.String("dest", "testdata/governance/v1/gait-gate", "dest")
	update := flag.Bool("update", false, "update")
	flag.Parse()
	if *update {
		if e := importAll(*src, *dst); e != nil {
			panic(e)
		}
	} else if e := check(*dst); e != nil {
		panic(e)
	}
}
func importAll(src, dst string) error {
	raw, e := os.ReadFile(filepath.Join(src, "fixture-manifest.json"))
	if e != nil {
		return e
	}
	var m map[string]any
	if e = json.Unmarshal(raw, &m); e != nil {
		return e
	}
	files := m["files"].([]any)
	if e = os.MkdirAll(dst, 0700); e != nil {
		return e
	}
	if e = os.WriteFile(filepath.Join(dst, "upstream-manifest.json"), raw, 0600); e != nil {
		return e
	}
	for _, v := range files {
		p := v.(map[string]any)["path"].(string)
		b, e := os.ReadFile(filepath.Join(src, p))
		if e != nil {
			return e
		}
		d := filepath.Join(dst, p)
		if e = os.MkdirAll(filepath.Dir(d), 0700); e != nil {
			return e
		}
		if e = os.WriteFile(d, b, 0600); e != nil {
			return e
		}
	}
	k, e := os.ReadFile(filepath.Join(src, "fixture-signing-key.public.b64"))
	if e != nil {
		return e
	}
	return os.WriteFile(filepath.Join(dst, "fixture-signing-key.public.b64"), k, 0600)
}
func check(dst string) error {
	raw, e := os.ReadFile(filepath.Join(dst, "upstream-manifest.json"))
	if e != nil {
		return e
	}
	var m struct {
		Files []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"files"`
	}
	if e = json.Unmarshal(raw, &m); e != nil {
		return e
	}
	for _, f := range m.Files {
		b, e := os.ReadFile(filepath.Join(dst, f.Path))
		if e != nil {
			return e
		}
		if sum(b) != f.SHA256 {
			return fmt.Errorf("digest drift: %s", f.Path)
		}
	}
	return nil
}
