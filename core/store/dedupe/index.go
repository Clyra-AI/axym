package dedupe

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Clyra-AI/proof"
)

type Entry struct {
	ExpiresAt time.Time `json:"expires_at"`
	LastSeen  time.Time `json:"last_seen"`
}

type Index struct {
	Entries map[string]Entry `json:"entries"`
	TTL     time.Duration    `json:"-"`
	MaxSize int              `json:"-"`
}

func New(ttl time.Duration, maxSize int) *Index {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	if maxSize <= 0 {
		maxSize = 10000
	}
	return &Index{Entries: map[string]Entry{}, TTL: ttl, MaxSize: maxSize}
}

func BuildKey(sourceProduct string, recordType string, event map[string]any) (string, error) {
	raw, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("marshal event: %w", err)
	}
	canonical, err := proof.Canonicalize(raw, proof.DomainJSON)
	if err != nil {
		return "", fmt.Errorf("canonicalize event: %w", err)
	}
	sum := sha256.Sum256(canonical)
	eventHash := "sha256:" + hex.EncodeToString(sum[:])
	return strings.Join([]string{strings.TrimSpace(sourceProduct), strings.TrimSpace(recordType), eventHash}, "|"), nil
}

func (i *Index) Seen(key string, now time.Time) bool {
	i.Purge(now)
	entry, ok := i.Entries[key]
	if !ok {
		return false
	}
	if !entry.ExpiresAt.After(now) {
		delete(i.Entries, key)
		return false
	}
	entry.LastSeen = now
	i.Entries[key] = entry
	return true
}

func (i *Index) Mark(key string, now time.Time) {
	if i.Entries == nil {
		i.Entries = map[string]Entry{}
	}
	i.Entries[key] = Entry{ExpiresAt: now.Add(i.TTL), LastSeen: now}
	i.Purge(now)
	if len(i.Entries) <= i.MaxSize {
		return
	}
	type pair struct {
		Key  string
		Seen time.Time
	}
	pairs := make([]pair, 0, len(i.Entries))
	for k, v := range i.Entries {
		pairs = append(pairs, pair{Key: k, Seen: v.LastSeen})
	}
	sort.Slice(pairs, func(a, b int) bool {
		return pairs[a].Seen.Before(pairs[b].Seen)
	})
	for len(i.Entries) > i.MaxSize && len(pairs) > 0 {
		delete(i.Entries, pairs[0].Key)
		pairs = pairs[1:]
	}
}

func (i *Index) Purge(now time.Time) {
	for key, entry := range i.Entries {
		if !entry.ExpiresAt.After(now) {
			delete(i.Entries, key)
		}
	}
}
