package fileprojectmemory

// File-backed storage for per-project memory. One JSON file per project at
// <dataDir>/projectmemory/<id>.json, mode 0600. The file holds the memory
// document, its enabled flag, and an update timestamp. Write path renames a
// temp file into place for atomic replacement, mirroring project secrets.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	serviceproject "github.com/futrx-com/remote.futrx.com/internal/service/project"
)

var _ serviceproject.MemoryRepository = (*Store)(nil)

type record struct {
	Content   string `json:"content"`
	Enabled   bool   `json:"enabled"`
	UpdatedAt int64  `json:"updatedAt"`
}

type Store struct {
	root string

	mu    sync.Mutex
	locks map[serviceproject.ID]*sync.Mutex
}

func New(dataDir string) (*Store, error) {
	dir := filepath.Join(dataDir, "projectmemory")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create projectmemory dir: %w", err)
	}
	_ = os.Chmod(dir, 0o700)
	return &Store{root: dir, locks: map[serviceproject.ID]*sync.Mutex{}}, nil
}

func (s *Store) path(id serviceproject.ID) string {
	return filepath.Join(s.root, string(id)+".json")
}

func (s *Store) lock(id serviceproject.ID) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.locks[id]; ok {
		return m
	}
	m := &sync.Mutex{}
	s.locks[id] = m
	return m
}

func (s *Store) Get(_ context.Context, id serviceproject.ID) (serviceproject.Memory, error) {
	lock := s.lock(id)
	lock.Lock()
	defer lock.Unlock()

	rec, err := s.loadLocked(id)
	if err != nil {
		return serviceproject.Memory{}, err
	}
	return serviceproject.Memory{
		Content:   rec.Content,
		Enabled:   rec.Enabled,
		UpdatedAt: rec.UpdatedAt,
	}, nil
}

func (s *Store) Set(_ context.Context, id serviceproject.ID, content string, enabled bool) (serviceproject.Memory, error) {
	lock := s.lock(id)
	lock.Lock()
	defer lock.Unlock()

	rec := record{
		Content:   content,
		Enabled:   enabled,
		UpdatedAt: time.Now().Unix(),
	}
	if err := s.writeLocked(id, rec); err != nil {
		return serviceproject.Memory{}, err
	}
	return serviceproject.Memory{
		Content:   rec.Content,
		Enabled:   rec.Enabled,
		UpdatedAt: rec.UpdatedAt,
	}, nil
}

func (s *Store) loadLocked(id serviceproject.ID) (record, error) {
	raw, err := os.ReadFile(s.path(id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return record{}, nil
		}
		return record{}, err
	}
	if len(raw) == 0 {
		return record{}, nil
	}
	var rec record
	if err := json.Unmarshal(raw, &rec); err != nil {
		return record{}, fmt.Errorf("decode project memory %s: %w", id, err)
	}
	return rec, nil
}

func (s *Store) writeLocked(id serviceproject.ID, rec record) error {
	path := s.path(id)
	raw, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("write project memory %s: %w", id, err)
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("chmod project memory %s: %w", id, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace project memory %s: %w", id, err)
	}
	return nil
}
