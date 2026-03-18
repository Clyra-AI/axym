package store

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Clyra-AI/axym/core/store/dedupe"
	"github.com/Clyra-AI/proof"
)

const (
	defaultStoreDir   = ".axym"
	defaultChainFile  = "chain.json"
	defaultDedupeFile = "dedupe.json"
	defaultKeyFile    = "signing-key.json"
	defaultChainID    = "axym-local"
	defaultDedupeTTL  = 24 * time.Hour
	defaultDedupeMax  = 10000
	defaultDirPerm    = 0o700
)

type AtomicWriter func(path string, data []byte, fsync bool) error

type Option func(*Store)

type Config struct {
	RootDir        string
	ChainID        string
	ComplianceMode bool
	DedupeTTL      time.Duration
	DedupeMaxSize  int
}

type AppendResult struct {
	Appended    bool
	Deduped     bool
	HeadHash    string
	RecordCount int
	RecordID    string
}

type Store struct {
	cfg         Config
	mu          sync.Mutex
	now         func() time.Time
	atomicWrite AtomicWriter
}

type keyFile struct {
	KeyID   string `json:"key_id"`
	Public  string `json:"public"`
	Private string `json:"private"`
}

type dedupeState struct {
	Entries map[string]dedupe.Entry `json:"entries"`
}

func New(cfg Config, opts ...Option) (*Store, error) {
	cfg = normalizeConfig(cfg)
	if err := os.MkdirAll(cfg.RootDir, defaultDirPerm); err != nil {
		return nil, fmt.Errorf("create store root: %w", err)
	}
	s := &Store{
		cfg:         cfg,
		now:         func() time.Time { return time.Now().UTC().Truncate(time.Second) },
		atomicWrite: WriteJSONAtomic,
	}
	for _, opt := range opts {
		opt(s)
	}
	if err := s.ensureSigningKey(); err != nil {
		return nil, err
	}
	return s, nil
}

func WithClock(now func() time.Time) Option {
	return func(s *Store) {
		if now != nil {
			s.now = now
		}
	}
}

func WithAtomicWriter(writer AtomicWriter) Option {
	return func(s *Store) {
		if writer != nil {
			s.atomicWrite = writer
		}
	}
}

func (s *Store) RootDir() string {
	return s.cfg.RootDir
}

func (s *Store) ChainPath() string {
	return filepath.Join(s.cfg.RootDir, defaultChainFile)
}

func (s *Store) Append(record *proof.Record, dedupeKey string) (AppendResult, error) {
	if record == nil {
		return AppendResult{}, errors.New("record is nil")
	}
	if strings.TrimSpace(record.RecordID) == "" {
		return AppendResult{}, errors.New("record_id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC().Truncate(time.Second)
	chain, err := s.loadChain()
	if err != nil {
		return AppendResult{}, err
	}
	idx, err := s.loadDedupeIndex()
	if err != nil {
		return AppendResult{}, err
	}

	if dedupeKey != "" {
		if idx.Seen(dedupeKey, now) || chainHasKey(chain, dedupeKey) {
			return AppendResult{
				Appended:    false,
				Deduped:     true,
				HeadHash:    chain.HeadHash,
				RecordCount: len(chain.Records),
				RecordID:    record.RecordID,
			}, nil
		}
	}

	linked := proof.Record(*record)
	linked.Integrity.PreviousRecordHash = chain.HeadHash
	linked.Integrity.RecordHash = ""
	linked.Integrity.Signature = ""
	linked.Integrity.SigningKeyID = ""

	signingKey, err := s.loadSigningKey()
	if err != nil {
		return AppendResult{}, err
	}
	if _, err := proof.Sign(&linked, signingKey); err != nil {
		return AppendResult{}, fmt.Errorf("sign record: %w", err)
	}
	if err := proof.AppendToChain(chain, &linked); err != nil {
		return AppendResult{}, fmt.Errorf("append to chain: %w", err)
	}
	if dedupeKey != "" {
		idx.Mark(dedupeKey, now)
	}
	if err := s.persistChain(chain); err != nil {
		return AppendResult{}, err
	}
	if err := s.persistDedupeIndex(idx); err != nil {
		return AppendResult{}, err
	}

	return AppendResult{
		Appended:    true,
		Deduped:     false,
		HeadHash:    chain.HeadHash,
		RecordCount: len(chain.Records),
		RecordID:    linked.RecordID,
	}, nil
}

func (s *Store) LoadChain() (*proof.Chain, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadChain()
}

func (s *Store) loadChain() (*proof.Chain, error) {
	path := s.ChainPath()
	// #nosec G304 -- chain path is derived from the managed local store root.
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return proof.NewChain(s.cfg.ChainID), nil
		}
		return nil, fmt.Errorf("read chain: %w", err)
	}
	var chain proof.Chain
	if err := json.Unmarshal(raw, &chain); err != nil {
		return nil, fmt.Errorf("decode chain: %w", err)
	}
	return &chain, nil
}

func (s *Store) persistChain(chain *proof.Chain) error {
	raw, err := json.MarshalIndent(chain, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal chain: %w", err)
	}
	if err := s.atomicWrite(s.ChainPath(), raw, s.cfg.ComplianceMode); err != nil {
		return fmt.Errorf("persist chain: %w", err)
	}
	return nil
}

func (s *Store) loadDedupeIndex() (*dedupe.Index, error) {
	path := filepath.Join(s.cfg.RootDir, defaultDedupeFile)
	idx := dedupe.New(s.cfg.DedupeTTL, s.cfg.DedupeMaxSize)
	// #nosec G304 -- dedupe index path is derived from the managed local store root.
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return idx, nil
		}
		return nil, fmt.Errorf("read dedupe index: %w", err)
	}
	var state dedupeState
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, fmt.Errorf("decode dedupe index: %w", err)
	}
	if state.Entries != nil {
		idx.Entries = state.Entries
	}
	return idx, nil
}

func (s *Store) persistDedupeIndex(idx *dedupe.Index) error {
	state := dedupeState{Entries: idx.Entries}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal dedupe index: %w", err)
	}
	path := filepath.Join(s.cfg.RootDir, defaultDedupeFile)
	if err := s.atomicWrite(path, raw, s.cfg.ComplianceMode); err != nil {
		return fmt.Errorf("persist dedupe index: %w", err)
	}
	return nil
}

func (s *Store) keyPath() string {
	return filepath.Join(s.cfg.RootDir, defaultKeyFile)
}

func (s *Store) ensureSigningKey() error {
	_, err := s.loadSigningKey()
	if err == nil {
		return nil
	}
	key, err := proof.GenerateSigningKey()
	if err != nil {
		return fmt.Errorf("generate signing key: %w", err)
	}
	serialized := keyFile{
		KeyID:   key.KeyID,
		Public:  base64.StdEncoding.EncodeToString(key.Public),
		Private: base64.StdEncoding.EncodeToString(key.Private),
	}
	raw, err := json.MarshalIndent(serialized, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal signing key: %w", err)
	}
	if err := s.atomicWrite(s.keyPath(), raw, s.cfg.ComplianceMode); err != nil {
		return fmt.Errorf("persist signing key: %w", err)
	}
	return nil
}

func (s *Store) loadSigningKey() (proof.SigningKey, error) {
	// #nosec G304 -- signing key path is derived from the managed local store root.
	raw, err := os.ReadFile(s.keyPath())
	if err != nil {
		return proof.SigningKey{}, err
	}
	var serialized keyFile
	if err := json.Unmarshal(raw, &serialized); err != nil {
		return proof.SigningKey{}, fmt.Errorf("decode signing key: %w", err)
	}
	publicKey, err := base64.StdEncoding.DecodeString(serialized.Public)
	if err != nil {
		return proof.SigningKey{}, fmt.Errorf("decode public key: %w", err)
	}
	privateKey, err := base64.StdEncoding.DecodeString(serialized.Private)
	if err != nil {
		return proof.SigningKey{}, fmt.Errorf("decode private key: %w", err)
	}
	return proof.SigningKey{
		KeyID:   serialized.KeyID,
		Public:  publicKey,
		Private: privateKey,
	}, nil
}

func normalizeConfig(cfg Config) Config {
	if cfg.RootDir == "" {
		cfg.RootDir = defaultStoreDir
	}
	if cfg.ChainID == "" {
		cfg.ChainID = defaultChainID
	}
	if cfg.DedupeTTL <= 0 {
		cfg.DedupeTTL = defaultDedupeTTL
	}
	if cfg.DedupeMaxSize <= 0 {
		cfg.DedupeMaxSize = defaultDedupeMax
	}
	return cfg
}

func chainHasKey(chain *proof.Chain, key string) bool {
	for i := range chain.Records {
		record := chain.Records[i]
		candidate, err := dedupe.BuildKey(record.SourceProduct, record.RecordType, record.Event)
		if err != nil {
			continue
		}
		if candidate == key {
			return true
		}
	}
	return false
}
