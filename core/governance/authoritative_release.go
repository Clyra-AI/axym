package governance

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	governanceschema "github.com/Clyra-AI/axym/schemas/v1/governance"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const AuthoritativeReleaseManifestName = "axym-authoritative-action-contract-manifest.json"

type AuthoritativeReleaseManifest struct {
	ManifestVersion  string `json:"manifest_version"`
	ReleaseTag       string `json:"release_tag"`
	PeeledCommit     string `json:"peeled_commit"`
	Authoritative    bool   `json:"authoritative"`
	FixtureOnly      bool   `json:"fixture_only"`
	NonAuthoritative bool   `json:"non_authoritative"`
	Quarantine       bool   `json:"quarantine"`
	Producer         struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Kind    string `json:"kind"`
	} `json:"producer"`
	Generator struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"generator"`
	Workflow struct {
		Name       string `json:"name"`
		Ref        string `json:"ref"`
		RunID      string `json:"run_id"`
		Repository string `json:"repository"`
	} `json:"workflow"`
	Signing struct {
		Algorithm       string `json:"algorithm"`
		PublicKeyPath   string `json:"public_key_path"`
		PublicKeySHA256 string `json:"public_key_sha256"`
		KeyOrigin       string `json:"key_origin"`
	} `json:"signing"`
	Artifacts map[string]string            `json:"artifacts"`
	Schemas   []AuthoritativeReleaseSchema `json:"schemas"`
	Files     []AuthoritativeReleaseFile   `json:"files"`
}

type AuthoritativeReleaseSchema struct {
	Path          string `json:"path"`
	SHA256        string `json:"sha256"`
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
}

type AuthoritativeReleaseFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

var commitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

func VerifyAuthoritativeRelease(root, expectedTag, expectedCommit string) error {
	return fmt.Errorf("external authoritative release trust anchor is required")
}

func VerifyAuthoritativeReleaseWithKeyDigest(root, expectedTag, expectedCommit, trustedKeyDigest string) error {
	if !validReleaseDigest(trustedKeyDigest) {
		return fmt.Errorf("external authoritative release trust anchor is invalid")
	}
	return verifyAuthoritativeRelease(root, expectedTag, expectedCommit, trustedKeyDigest)
}

func verifyAuthoritativeRelease(root, expectedTag, expectedCommit, trustedKeyDigest string) error {
	manifestRaw, err := readReleaseFile(root, AuthoritativeReleaseManifestName)
	if err != nil {
		return fmt.Errorf("read authoritative release manifest: %w", err)
	}
	var manifest AuthoritativeReleaseManifest
	if err := decodeStrictJSON(manifestRaw, &manifest); err != nil {
		return fmt.Errorf("decode authoritative release manifest: %w", err)
	}
	if manifest.ManifestVersion != "v1" || !manifest.Authoritative || manifest.FixtureOnly || manifest.NonAuthoritative || manifest.Quarantine {
		return fmt.Errorf("authoritative release manifest contains fixture or quarantine markers")
	}
	if strings.TrimSpace(manifest.ReleaseTag) == "" || (expectedTag != "" && manifest.ReleaseTag != expectedTag) || !commitPattern.MatchString(manifest.PeeledCommit) || (expectedCommit != "" && manifest.PeeledCommit != expectedCommit) {
		return fmt.Errorf("authoritative release identity is invalid")
	}
	if manifest.Producer.Name != "axym" || manifest.Producer.Version == "" || manifest.Producer.Kind == "" || manifest.Generator.Name == "" || manifest.Generator.Version == "" || manifest.Workflow.Name == "" || manifest.Workflow.Repository == "" {
		return fmt.Errorf("authoritative release ownership metadata is incomplete")
	}
	if manifest.Signing.Algorithm != "ed25519" || manifest.Signing.KeyOrigin != "release_time_generated" || manifest.Signing.PublicKeyPath == "" || !validReleaseDigest(manifest.Signing.PublicKeySHA256) {
		return fmt.Errorf("authoritative release signing metadata is invalid")
	}
	files := map[string]AuthoritativeReleaseFile{}
	for _, item := range manifest.Files {
		if item.Path == "" || !safeReleasePath(item.Path) || !validReleaseDigest(item.SHA256) || files[item.Path].Path != "" {
			return fmt.Errorf("authoritative release file manifest is invalid")
		}
		files[item.Path] = item
		raw, err := readReleaseFile(root, item.Path)
		if err != nil || releaseDigest(raw) != item.SHA256 {
			return fmt.Errorf("authoritative release file digest mismatch: %s", item.Path)
		}
	}
	keyRaw, err := readReleaseFile(root, manifest.Signing.PublicKeyPath)
	if err != nil {
		return fmt.Errorf("read authoritative release public key: %w", err)
	}
	if releaseDigest(keyRaw) != manifest.Signing.PublicKeySHA256 {
		return fmt.Errorf("authoritative release public key digest mismatch")
	}
	if files[manifest.Signing.PublicKeyPath].Path == "" || files[manifest.Signing.PublicKeyPath].SHA256 != manifest.Signing.PublicKeySHA256 {
		return fmt.Errorf("authoritative public key is not included in the file manifest")
	}
	if manifest.Signing.PublicKeySHA256 != trustedKeyDigest {
		return fmt.Errorf("authoritative release public key is not externally trusted")
	}
	public, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(keyRaw)))
	if err != nil || len(public) != ed25519.PublicKeySize {
		return fmt.Errorf("authoritative release public key is invalid")
	}
	var register Register
	registerPath := findReleaseArtifact(files, "register")
	if registerPath == "" {
		return fmt.Errorf("authoritative release register is missing")
	}
	registerRaw, _ := readReleaseFile(root, registerPath)
	if err := decodeStrictJSON(registerRaw, &register); err != nil {
		return fmt.Errorf("decode authoritative register: %w", err)
	}
	if err := VerifySignedRegister(register, ed25519.PublicKey(public)); err != nil {
		return fmt.Errorf("verify authoritative register: %w", err)
	}
	if err := governanceschema.ValidateRegister(registerRaw); err != nil {
		return fmt.Errorf("validate authoritative register schema: %w", err)
	}
	var packet Packet
	packetPath := findReleaseArtifact(files, "packet")
	if packetPath == "" {
		return fmt.Errorf("authoritative release packet is missing")
	}
	packetRaw, _ := readReleaseFile(root, packetPath)
	if err := decodeStrictJSON(packetRaw, &packet); err != nil {
		return fmt.Errorf("decode authoritative packet: %w", err)
	}
	if err := VerifySignedPacket(packet, ed25519.PublicKey(public)); err != nil {
		return fmt.Errorf("verify authoritative packet: %w", err)
	}
	if err := governanceschema.ValidatePacket(packetRaw); err != nil {
		return fmt.Errorf("validate authoritative packet schema: %w", err)
	}
	if len(register.Contracts) != 1 || packet.Signature == nil || register.Signature == nil || packet.Signature.KeyID != register.Signature.KeyID {
		return fmt.Errorf("authoritative register and packet relationship mismatch")
	}
	registeredContractDigest, _ := Digest(register.Contracts[0])
	packetContractDigest, _ := Digest(packet.Contract)
	if registeredContractDigest != packetContractDigest {
		return fmt.Errorf("authoritative register and packet contract drift")
	}
	if registerPath == "" || packetPath == "" || manifest.Artifacts["register"] != files[registerPath].SHA256 || manifest.Artifacts["packet"] != files[packetPath].SHA256 {
		return fmt.Errorf("authoritative artifact digest manifest mismatch")
	}
	bundlePath := findReleaseArtifact(files, "bundle")
	if bundlePath == "" || manifest.Artifacts["bundle"] != files[bundlePath].SHA256 {
		return fmt.Errorf("authoritative bundle digest manifest mismatch")
	}
	bundleRaw, err := readReleaseFile(root, bundlePath)
	if err != nil || !verifyReleaseBundle(bundleRaw, files) {
		return fmt.Errorf("authoritative compressed bundle contents mismatch")
	}
	for _, evidence := range packet.Evidence {
		if evidence.ContractRef != packet.Contract.CausalRef || evidence.Provenance.ID == "" || evidence.Provenance.Digest == "" {
			return fmt.Errorf("authoritative packet contains an inexact relationship reference")
		}
	}
	expectedSchemas := map[string]struct {
		id      string
		version string
	}{
		"axym-authoritative-action-contract-register.schema.json":        {id: RegisterSchemaID, version: SchemaVersion},
		"axym-authoritative-action-contract-evidence-packet.schema.json": {id: PacketSchemaID, version: SchemaVersion},
	}
	if len(manifest.Schemas) != len(expectedSchemas) {
		return fmt.Errorf("authoritative release must publish exactly both normative schemas")
	}
	seenSchemas := map[string]bool{}
	for _, schema := range manifest.Schemas {
		if schema.Path == "" || !safeReleasePath(schema.Path) || !validReleaseDigest(schema.SHA256) || schema.SchemaID == "" || schema.SchemaVersion == "" {
			return fmt.Errorf("authoritative schema manifest is invalid")
		}
		expected, ok := expectedSchemas[schema.Path]
		if !ok || seenSchemas[schema.Path] || schema.SchemaID != expected.id || schema.SchemaVersion != expected.version || files[schema.Path].Path == "" {
			return fmt.Errorf("authoritative schema manifest is incomplete or non-normative")
		}
		seenSchemas[schema.Path] = true
		raw, err := readReleaseFile(root, schema.Path)
		if err != nil || releaseDigest(raw) != schema.SHA256 {
			return fmt.Errorf("authoritative schema digest mismatch: %s", schema.Path)
		}
		var document map[string]any
		if err := json.Unmarshal(raw, &document); err != nil || document["$id"] != schema.SchemaID {
			return fmt.Errorf("authoritative schema identity mismatch: %s", schema.Path)
		}
		expectedSchema := governanceschema.RegisterSchema()
		if schema.SchemaID == PacketSchemaID {
			expectedSchema = governanceschema.PacketSchema()
		}
		if !bytes.Equal(raw, expectedSchema) {
			return fmt.Errorf("authoritative schema does not match the embedded normative schema: %s", schema.Path)
		}
	}
	return nil
}

func decodeStrictJSON(raw []byte, out any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("trailing JSON")
	}
	return nil
}

func findReleaseArtifact(files map[string]AuthoritativeReleaseFile, kind string) string {
	paths := make([]string, 0, len(files))
	for path := range files {
		if (strings.Contains(path, "register") && kind == "register") || (strings.Contains(path, "packet") && kind == "packet") || (strings.Contains(path, "bundle") && kind == "bundle") {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func safeReleasePath(path string) bool {
	clean := filepath.ToSlash(filepath.Clean(path))
	return clean == path && clean != "." && !strings.HasPrefix(clean, "../") && !strings.Contains(clean, "/../") && !filepath.IsAbs(path)
}

func readReleaseFile(root, path string) ([]byte, error) {
	if !safeReleasePath(path) {
		return nil, fmt.Errorf("unsafe release path: %s", path)
	}
	// #nosec G304 -- the path is constrained to a safe relative path beneath
	// the explicit release root before it is opened.
	return os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
}

func validReleaseDigest(value string) bool { return digestPattern.MatchString(value) }
func releaseDigest(raw []byte) string      { return rawDigest(raw) }

func verifyReleaseBundle(raw []byte, files map[string]AuthoritativeReleaseFile) (valid bool) {
	reader, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return false
	}
	defer func() {
		if err := reader.Close(); err != nil {
			valid = false
		}
	}()
	tarReader := tar.NewReader(reader)
	seen := map[string]bool{}
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil || header.Typeflag != tar.TypeReg || !strings.HasPrefix(header.Name, "authoritative-governance/") {
			return false
		}
		name := strings.TrimPrefix(header.Name, "authoritative-governance/")
		if name == bundleNameForRelease(files) || !safeReleasePath(name) || seen[name] {
			return false
		}
		item, ok := files[name]
		if !ok {
			return false
		}
		payload, err := io.ReadAll(tarReader)
		if err != nil || releaseDigest(payload) != item.SHA256 {
			return false
		}
		seen[name] = true
	}
	for name := range files {
		if name != bundleNameForRelease(files) && !seen[name] {
			return false
		}
	}
	return true
}

func bundleNameForRelease(files map[string]AuthoritativeReleaseFile) string {
	for name := range files {
		if strings.HasSuffix(name, "-bundle.tar.gz") {
			return name
		}
	}
	return ""
}
