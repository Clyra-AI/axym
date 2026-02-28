package snowflake

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func CanonicalQueryDigest(sql string) string {
	normalized := strings.ToLower(strings.Join(strings.Fields(sql), " "))
	sum := sha256.Sum256([]byte(normalized))
	return "sha256:" + hex.EncodeToString(sum[:])
}
