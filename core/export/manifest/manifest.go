package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Clyra-AI/proof"
)

func Build(root string, relPaths []string) (proof.BundleManifest, error) {
	normalized := make([]string, 0, len(relPaths))
	for _, rel := range relPaths {
		if rel == "" {
			continue
		}
		normalized = append(normalized, filepath.ToSlash(filepath.Clean(rel)))
	}
	sort.Strings(normalized)

	files := make([]proof.BundleManifestEntry, 0, len(normalized))
	for _, rel := range normalized {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		// #nosec G304 -- manifest paths are built from bundle-root plus normalized manifest entries.
		data, err := os.ReadFile(abs)
		if err != nil {
			return proof.BundleManifest{}, fmt.Errorf("read bundle file %q: %w", rel, err)
		}
		sum := sha256.Sum256(data)
		files = append(files, proof.BundleManifestEntry{
			Path:   rel,
			SHA256: "sha256:" + hex.EncodeToString(sum[:]),
		})
	}

	return proof.BundleManifest{
		Files:  files,
		AlgoID: "sha256",
		SaltID: "",
	}, nil
}
