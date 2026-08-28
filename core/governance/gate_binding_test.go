package governance

import (
	"testing"
	"time"
)

func TestValidateGateContractBindingChecksDigestAndLifecycleWindow(t *testing.T) {
	base := map[string]any{
		"contract_id":        "contract-1",
		"contract_family_id": "family-1",
		"contract_revision":  float64(2),
		"contract_digest":    "sha256:" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"created_at":         "2026-08-01T12:00:00Z",
		"expires_at":         "2026-08-01T14:00:00Z",
	}
	validAt := time.Date(2026, 8, 1, 13, 0, 0, 0, time.UTC)
	if err := ValidateGateContractBinding(base, "contract-1", "family-1", 2, "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", validAt); err != nil {
		t.Fatalf("valid gate rejected: %v", err)
	}
	badDigest := cloneGateMap(base)
	badDigest["contract_digest"] = "sha256:" + "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	if err := ValidateGateContractBinding(badDigest, "contract-1", "family-1", 2, "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", validAt); err == nil {
		t.Fatal("mismatched contract digest accepted")
	}
	if err := ValidateGateContractBinding(base, "contract-1", "family-1", 2, "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", time.Date(2026, 8, 1, 11, 59, 59, 0, time.UTC)); err == nil {
		t.Fatal("gate accepted before creation")
	}
}
