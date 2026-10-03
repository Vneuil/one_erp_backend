package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/divinecoid/one-backend/internal/foundation/config"
)

// NewWithLocalFallback returns R2 storage when configured, otherwise files on
// local disk under localDir when set, otherwise the disabled no-op storage.
func NewWithLocalFallback(cfg config.R2Config, localDir string) Storage {
	if s := NewStorage(cfg); s.Enabled() {
		return s
	}
	if strings.TrimSpace(localDir) == "" {
		return &noopStorage{}
	}
	return NewLocal(localDir)
}

type localStorage struct{ root string }

// NewLocal stores objects as files under root.
func NewLocal(root string) Storage { return &localStorage{root: root} }

func (s *localStorage) Enabled() bool { return true }

// path maps a key to a file inside root and refuses anything that would
// escape it (absolute paths, "..").
func (s *localStorage) path(key string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(key))
	if key == "" || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid storage key")
	}
	return filepath.Join(s.root, clean), nil
}

func (s *localStorage) Upload(_ context.Context, key string, content []byte, _ string) (string, error) {
	p, err := s.path(key)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return "", fmt.Errorf("failed to prepare storage: %w", err)
	}
	if err := os.WriteFile(p, content, 0o640); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	return key, nil
}

func (s *localStorage) Download(_ context.Context, key string) ([]byte, error) {
	p, err := s.path(key)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(p)
}

func (s *localStorage) Delete(_ context.Context, key string) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
