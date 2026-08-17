package project

import (
	"context"
	"errors"
	"strings"
)

// Memory is a per-project, human-readable context document shared across all
// chats in the project. Agents receive it as prompt context at the start of
// every run, so project decisions, conventions, and state survive across
// chats and agent sessions.
type Memory struct {
	Content   string `json:"content"`
	Enabled   bool   `json:"enabled"`
	UpdatedAt int64  `json:"updatedAt"`
}

// MemoryRepository is the storage port for project memory. Implementations
// must persist atomically; concurrent callers from different goroutines are
// serialized per project.
type MemoryRepository interface {
	Get(ctx context.Context, projectID ID) (Memory, error)
	Set(ctx context.Context, projectID ID, content string, enabled bool) (Memory, error)
}

var (
	// ErrMemoryUnavailable reports that no memory store was wired (tests or
	// minimal installs): memory reads come back empty and writes fail.
	ErrMemoryUnavailable = errors.New("project memory is unavailable")
	// ErrMemoryTooLarge bounds the memory document. It rides along with every
	// prompt, so it must stay small enough not to drown the agent's task.
	ErrMemoryTooLarge = errors.New("project memory is too large")
)

// MaxMemoryBytes is the upper bound for a memory document.
const MaxMemoryBytes = 32 << 10

// ValidMemoryContent trims and validates a memory document before storage.
func ValidMemoryContent(content string) bool {
	return len(strings.TrimSpace(content)) <= MaxMemoryBytes
}

// GetMemory returns the project's shared memory document.
func (s *Service) GetMemory(ctx context.Context, id ID) (Memory, error) {
	if s.memory == nil {
		return Memory{}, nil
	}
	return s.memory.Get(ctx, id)
}

// SetMemory replaces the project's shared memory document.
func (s *Service) SetMemory(ctx context.Context, id ID, content string, enabled bool) (Memory, error) {
	if s.memory == nil {
		return Memory{}, ErrMemoryUnavailable
	}
	if !ValidMemoryContent(content) {
		return Memory{}, ErrMemoryTooLarge
	}
	return s.memory.Set(ctx, id, strings.TrimSpace(content), enabled)
}
