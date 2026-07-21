package fsrepo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
)

const (
	directoryMode = 0o700
	fileMode      = 0o600
)

type Repository struct {
	root string

	mu     sync.RWMutex
	closed bool
}

func Open(root string) (*Repository, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf(
			"%w: definition repository root is empty",
			artifactstore.ErrInvalid,
		)
	}
	clean := filepath.Clean(root)
	if err := os.MkdirAll(clean, directoryMode); err != nil {
		return nil, fmt.Errorf("create definition repository: %w", err)
	}
	return &Repository{root: clean}, nil
}

func (r *Repository) Close() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	return nil
}

func (r *Repository) Put(
	ctx context.Context,
	value definition.Definition,
) (definition.Definition, error) {
	if err := ctx.Err(); err != nil {
		return definition.Definition{}, err
	}
	canonical, err := definition.Canonicalize(value)
	if err != nil {
		return definition.Definition{}, err
	}
	content, err := encodeFile(canonical)
	if err != nil {
		return definition.Definition{}, err
	}
	path, err := r.pathFor(canonical.Digest)
	if err != nil {
		return definition.Definition{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return definition.Definition{}, artifactstore.ErrClosed
	}

	if existing, readErr := os.ReadFile(path); readErr == nil {
		decoded, decodeErr := decodeFile(existing)
		if decodeErr != nil {
			return definition.Definition{}, fmt.Errorf(
				"decode existing definition %q: %w",
				canonical.Digest,
				decodeErr,
			)
		}
		if decoded.Digest != canonical.Digest {
			return definition.Definition{}, fmt.Errorf(
				"%w: existing definition file %q",
				artifactstore.ErrDigestMismatch,
				canonical.Digest,
			)
		}
		return decoded, nil
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return definition.Definition{}, fmt.Errorf(
			"read existing definition %q: %w",
			canonical.Digest,
			readErr,
		)
	}

	temp, err := os.CreateTemp(r.root, ".definition-*.tmp")
	if err != nil {
		return definition.Definition{}, fmt.Errorf("create definition temporary file: %w", err)
	}
	tempPath := temp.Name()
	committed := false
	defer func() {
		_ = temp.Close()
		if !committed {
			_ = os.Remove(tempPath)
		}
	}()

	if err := temp.Chmod(fileMode); err != nil {
		return definition.Definition{}, err
	}
	if _, err := temp.Write(content); err != nil {
		return definition.Definition{}, fmt.Errorf("write definition temporary file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return definition.Definition{}, fmt.Errorf("sync definition temporary file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return definition.Definition{}, fmt.Errorf("close definition temporary file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		if _, statErr := os.Stat(path); statErr == nil {
			committed = true
			return canonical, nil
		}
		return definition.Definition{}, fmt.Errorf("publish definition file: %w", err)
	}
	committed = true
	return canonical, nil
}

func (r *Repository) Get(
	ctx context.Context,
	digest artifactstore.Digest,
) (definition.Definition, error) {
	if err := ctx.Err(); err != nil {
		return definition.Definition{}, err
	}
	path, err := r.pathFor(digest)
	if err != nil {
		return definition.Definition{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		return definition.Definition{}, artifactstore.ErrClosed
	}

	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return definition.Definition{}, fmt.Errorf(
			"%w: definition %q",
			artifactstore.ErrDefinitionNotFound,
			digest,
		)
	}
	if err != nil {
		return definition.Definition{}, fmt.Errorf(
			"read definition %q: %w",
			digest,
			err,
		)
	}
	value, err := decodeFile(content)
	if err != nil {
		return definition.Definition{}, err
	}
	if value.Digest != digest {
		return definition.Definition{}, fmt.Errorf(
			"%w: requested %q, read %q",
			artifactstore.ErrDigestMismatch,
			digest,
			value.Digest,
		)
	}
	return value, nil
}

func (r *Repository) pathFor(
	digest artifactstore.Digest,
) (string, error) {
	if err := artifactstore.ValidateDigest(digest); err != nil {
		return "", err
	}
	value := strings.TrimPrefix(
		string(digest),
		artifactstore.DigestSHA256Prefix,
	)
	return filepath.Join(r.root, "definition-"+value+".json"), nil
}

var _ definition.Repository = (*Repository)(nil)
