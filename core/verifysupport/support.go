package verifysupport

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Clyra-AI/proof"
)

const (
	storeSigningKeyFile     = "signing-key.json"
	BundlePublicKeyArtifact = "record-signing-key.json"
	BundlePublicKeyVersion  = "v1"
	BundlePublicKeyAlg      = "ed25519"
)

type storeSigningKey struct {
	KeyID   string `json:"key_id"`
	Public  string `json:"public"`
	Private string `json:"private"`
}

type BundlePublicKey struct {
	Version   string `json:"version"`
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"key_id"`
	Public    string `json:"public"`
}

func LoadStoreSigningKey(storeDir string) (proof.SigningKey, error) {
	path := filepath.Join(strings.TrimSpace(storeDir), storeSigningKeyFile)
	// #nosec G304 -- signing key path is derived from the explicit store directory contract.
	raw, err := os.ReadFile(path)
	if err != nil {
		return proof.SigningKey{}, err
	}
	var payload storeSigningKey
	if err := json.Unmarshal(raw, &payload); err != nil {
		return proof.SigningKey{}, err
	}
	pub, err := base64.StdEncoding.DecodeString(payload.Public)
	if err != nil {
		return proof.SigningKey{}, fmt.Errorf("decode public key: %w", err)
	}
	priv, err := base64.StdEncoding.DecodeString(payload.Private)
	if err != nil {
		return proof.SigningKey{}, fmt.Errorf("decode private key: %w", err)
	}
	return proof.SigningKey{
		KeyID:   strings.TrimSpace(payload.KeyID),
		Public:  pub,
		Private: priv,
	}, nil
}

func LoadStorePublicKey(storeDir string) (proof.PublicKey, error) {
	signingKey, err := LoadStoreSigningKey(storeDir)
	if err != nil {
		return proof.PublicKey{}, err
	}
	return proof.PublicKey{
		KeyID:  signingKey.KeyID,
		Public: signingKey.Public,
	}, nil
}

func MarshalBundlePublicKey(publicKey proof.PublicKey) ([]byte, error) {
	payload := BundlePublicKey{
		Version:   BundlePublicKeyVersion,
		Algorithm: BundlePublicKeyAlg,
		KeyID:     strings.TrimSpace(publicKey.KeyID),
		Public:    base64.StdEncoding.EncodeToString(publicKey.Public),
	}
	return json.MarshalIndent(payload, "", "  ")
}

func UnmarshalBundlePublicKey(raw []byte) (proof.PublicKey, error) {
	var payload BundlePublicKey
	if err := json.Unmarshal(raw, &payload); err != nil {
		return proof.PublicKey{}, err
	}
	pub, err := base64.StdEncoding.DecodeString(payload.Public)
	if err != nil {
		return proof.PublicKey{}, fmt.Errorf("decode bundle public key: %w", err)
	}
	return proof.PublicKey{
		KeyID:  strings.TrimSpace(payload.KeyID),
		Public: pub,
	}, nil
}

func VerifyRecords(records []proof.Record, publicKey proof.PublicKey) (int, string, error) {
	for i := range records {
		record := records[i]
		if err := proof.Verify(&record, publicKey); err != nil {
			return i, record.RecordID, err
		}
	}
	return -1, "", nil
}
