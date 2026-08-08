package shareable

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type FileRepository struct {
	root string

	mu     sync.RWMutex
	closed bool
}

func OpenFileRepository(root string) (*FileRepository, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf(
			"%w: shareable document repository root is empty",
			basespec.ErrInvalid,
		)
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	absolute = filepath.Clean(absolute)
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		return nil, err
	}
	return &FileRepository{root: absolute}, nil
}

func (r *FileRepository) Close() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	return nil
}

func (r *FileRepository) Put(
	ctx context.Context,
	rootID basespec.RootID,
	digest cryptoutil.Digest,
	raw json.RawMessage,
) error {
	if err := r.check(ctx, rootID, digest); err != nil {
		return err
	}
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return basespec.ErrClosed
	}

	location, err := r.location(rootID, digest)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(location), 0o700); err != nil {
		return err
	}
	return writeCanonicalDocument(location, digest, canonical)
}

func (r *FileRepository) Get(
	ctx context.Context,
	rootID basespec.RootID,
	digest cryptoutil.Digest,
) (json.RawMessage, error) {
	if err := r.check(ctx, rootID, digest); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		return nil, basespec.ErrClosed
	}

	location, err := r.location(rootID, digest)
	if err != nil {
		return nil, err
	}
	raw, err := readBoundedFile(
		location,
		int64(basespec.MaxDefinitionBytes),
	)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf(
			"%w: shareable document %q",
			basespec.ErrShareableDocumentNotFound,
			digest,
		)
	}
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func (r *FileRepository) RemoveRoot(
	ctx context.Context,
	rootID basespec.RootID,
) error {
	if err := basespec.ValidateRootID(rootID); err != nil {
		return err
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: shareable document removal context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if r == nil {
		return basespec.ErrClosed
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return basespec.ErrClosed
	}

	location := filepath.Join(r.root, string(rootID))
	relative, err := filepath.Rel(r.root, location)
	if err != nil {
		return err
	}
	if relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) ||
		filepath.IsAbs(relative) {
		return fmt.Errorf(
			"%w: shareable document root path escapes repository",
			basespec.ErrInvalid,
		)
	}
	return os.RemoveAll(location)
}

func (r *FileRepository) check(
	ctx context.Context,
	rootID basespec.RootID,
	digest cryptoutil.Digest,
) error {
	if r == nil {
		return basespec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: shareable document context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := basespec.ValidateRootID(rootID); err != nil {
		return err
	}
	return cryptoutil.ValidateDigest(digest)
}

func (r *FileRepository) location(
	rootID basespec.RootID,
	digest cryptoutil.Digest,
) (string, error) {
	hexDigest := strings.TrimPrefix(
		string(digest),
		cryptoutil.DigestSHA256Prefix,
	)
	if len(hexDigest) < 2 {
		return "", fmt.Errorf("%w: invalid shareable digest", basespec.ErrInvalid)
	}
	return filepath.Join(
		r.root,
		string(rootID),
		"sha256",
		hexDigest[:2],
		hexDigest+".json",
	), nil
}

func writeCanonicalDocument(
	location string,
	digest cryptoutil.Digest,
	canonical []byte,
) error {
	temporary, err := os.CreateTemp(
		filepath.Dir(location),
		".shareable-document-*",
	)
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer func() {
		_ = os.Remove(temporaryName)
	}()

	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(canonical); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}

	if err := os.Link(temporaryName, location); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrExist) {
		return err
	}

	existing, err := readBoundedFile(
		location,
		int64(basespec.MaxDefinitionBytes),
	)
	if err != nil {
		return err
	}
	if !bytes.Equal(existing, canonical) {
		return fmt.Errorf(
			"%w: shareable document digest %q has different canonical bytes",
			basespec.ErrDigestMismatch,
			digest,
		)
	}
	return nil
}

func readBoundedFile(location string, maximum int64) ([]byte, error) {
	file, err := os.Open(location)
	if err != nil {
		return nil, err
	}
	raw, readErr := io.ReadAll(io.LimitReader(file, maximum+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return nil, errors.Join(readErr, closeErr)
	}
	if int64(len(raw)) > maximum {
		return nil, fmt.Errorf("%w: shareable document exceeds size limit", basespec.ErrInvalid)
	}
	return raw, nil
}
