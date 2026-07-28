package fsdir

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

const (
	directoryReadBatchSize                          = 256
	Kind                   artifactstore.SourceKind = "fs-directory"
)

type Config struct {
	RootPath string `json:"rootPath"`
}

type Adapter struct {
	traversalPolicy normalizedTraversalPolicy
}

func New() *Adapter {
	adapter, err := NewWithTraversalPolicy(nil)
	if err != nil {
		panic(err)
	}
	return adapter
}

func NewWithTraversalPolicy(policy *TraversalPolicy) (*Adapter, error) {
	normalized, err := normalizeTraversalPolicy(policy)
	if err != nil {
		return nil, err
	}
	return &Adapter{traversalPolicy: normalized}, nil
}

func (a *Adapter) Kind() artifactstore.SourceKind {
	return Kind
}

func (a *Adapter) NormalizeConfig(
	ctx context.Context,
	raw json.RawMessage,
) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	canonical, err := jsoncanon.CanonicalizeObject(raw, artifactstore.MaxConfigBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: filesystem source config: %w", artifactstore.ErrInvalid, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("%w: decode filesystem source config: %w", artifactstore.ErrInvalid, err)
	}
	root, err := normalizeFilesystemRoot(config.RootPath)
	if err != nil {
		return nil, err
	}
	config.RootPath = root

	encoded, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	encoded, err = jsoncanon.CanonicalizeObject(encoded, artifactstore.MaxConfigBytes)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(encoded), nil
}

func (a *Adapter) Open(
	ctx context.Context,
	value source.Source,
) (source.Snapshot, error) {
	if value.Kind != Kind {
		return nil, fmt.Errorf(
			"%w: filesystem adapter received source kind %q",
			artifactstore.ErrInvalid,
			value.Kind,
		)
	}
	config, err := decodeConfig(value.Config)
	if err != nil {
		return nil, err
	}
	generation, err := fingerprint(ctx, config.RootPath, a.traversalPolicy)
	if err != nil {
		return nil, err
	}
	return &snapshot{
		root:            config.RootPath,
		generation:      generation,
		traversalPolicy: a.traversalPolicy,
	}, nil
}

// ResolveLocalPath returns the existing native path for a locator inside this
// filesystem source. It is used only by trusted runtime integrations after a
// consumer has approved a selected record.
//
// The returned path is never part of a portable definition or public API
// projection.
func (a *Adapter) ResolveLocalPath(
	ctx context.Context,
	value source.Source,
	locator artifactstore.Locator,
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if value.Kind != Kind {
		return "", fmt.Errorf(
			"%w: filesystem adapter received source kind %q",
			artifactstore.ErrInvalid,
			value.Kind,
		)
	}
	if err := artifactstore.ValidateLocator(locator, true); err != nil {
		return "", err
	}

	if a.traversalPolicy.excludesLocator(string(locator)) {
		return "", fmt.Errorf(
			"%w: source locator %q is excluded by traversal policy",
			artifactstore.ErrNotFound,
			locator,
		)
	}
	config, err := decodeConfig(value.Config)
	if err != nil {
		return "", err
	}
	return resolveNativePath(config.RootPath, locator)
}

func decodeConfig(raw json.RawMessage) (Config, error) {
	canonical, err := jsoncanon.CanonicalizeObject(raw, artifactstore.MaxConfigBytes)
	if err != nil {
		return Config{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return Config{}, err
	}
	if !filepath.IsAbs(config.RootPath) ||
		filepath.Clean(config.RootPath) != config.RootPath {
		return Config{}, fmt.Errorf(
			"%w: invalid normalized filesystem root",
			artifactstore.ErrInvalid,
		)
	}
	if err := validateFilesystemRoot(config.RootPath); err != nil {
		return Config{}, err
	}
	return config, nil
}

func normalizeFilesystemRoot(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf(
			"%w: filesystem root path is required",
			artifactstore.ErrInvalid,
		)
	}
	if !filepath.IsAbs(raw) {
		return "", fmt.Errorf(
			"%w: filesystem root path must be absolute",
			artifactstore.ErrInvalid,
		)
	}

	root := filepath.Clean(raw)
	info, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("stat filesystem source root: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf(
			"%w: filesystem source root is not a directory",
			artifactstore.ErrInvalid,
		)
	}
	return root, nil
}

func validateFilesystemRoot(root string) error {
	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf(
			"%w: filesystem source root is unavailable: %w",
			artifactstore.ErrSourceUnavailable,
			err,
		)
	}
	if !info.IsDir() {
		return fmt.Errorf(
			"%w: filesystem source root is no longer a directory",
			artifactstore.ErrSourceUnavailable,
		)
	}
	return nil
}

func fingerprint(ctx context.Context, root string, policy normalizedTraversalPolicy) (string, error) {
	type entry struct {
		relative string
		mode     os.FileMode
		size     int64
		modified time.Time
	}

	values := make([]entry, 0)
	visited := 0

	var walk func(location string, depth int) error
	walk = func(location string, depth int) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := os.Stat(location)
		if err != nil {
			return err
		}
		if location == root && !info.IsDir() {
			return fmt.Errorf(
				"%w: filesystem source root is no longer a directory",
				artifactstore.ErrSourceUnavailable,
			)
		}

		if location != root {
			visited++
			if visited > artifactstore.DefaultMaxEntries {
				return fmt.Errorf(
					"%w: source exceeds %d entries",
					artifactstore.ErrInvalid,
					artifactstore.DefaultMaxEntries,
				)
			}
			relative, err := filepath.Rel(root, location)
			if err != nil {
				return err
			}
			if depth > artifactstore.DefaultMaxDepth {
				return fmt.Errorf(
					"%w: source exceeds traversal depth %d",
					artifactstore.ErrInvalid,
					artifactstore.DefaultMaxDepth,
				)
			}
			if info.IsDir() &&
				(policy.shouldSkipDirectory(info.Name()) ||
					policy.isGitSubmoduleDirectory(location)) {
				return nil
			}
			if !info.IsDir() && !info.Mode().IsRegular() {
				return nil
			}
			values = append(values, entry{
				relative: filepath.ToSlash(relative),
				mode:     info.Mode(),
				size:     info.Size(),
				modified: info.ModTime().UTC(),
			})
		}

		if !info.IsDir() {
			return nil
		}

		directory, err := os.Open(location)
		if err != nil {
			return err
		}
		for {
			children, readErr := directory.ReadDir(directoryReadBatchSize)
			for _, child := range children {
				if err := walk(
					filepath.Join(location, child.Name()),
					depth+1,
				); err != nil {
					closeErr := directory.Close()
					return errors.Join(err, closeErr)
				}
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				closeErr := directory.Close()
				return errors.Join(readErr, closeErr)
			}
		}
		return directory.Close()
	}

	if err := walk(root, 0); err != nil {
		return "", err
	}
	sort.Slice(values, func(left, right int) bool {
		return values[left].relative < values[right].relative
	})

	hash := sha256.New()
	var totalBytes int64

	for _, value := range values {
		_, _ = fmt.Fprintf(
			hash,
			"%s\x00%d\x00%d\x00%s\x00",
			value.relative,
			value.mode,
			value.size,
			value.modified.Format(time.RFC3339Nano),
		)
		if !value.mode.IsRegular() {
			continue
		}
		if value.size < 0 ||
			value.size > artifactstore.MaxScanBytes-totalBytes {
			return "", fmt.Errorf(
				"%w: source exceeds byte limit",
				artifactstore.ErrInvalid,
			)
		}

		path := filepath.Join(root, filepath.FromSlash(value.relative))
		file, err := os.Open(path)
		if err != nil {
			return "", err
		}
		info, statErr := file.Stat()
		if statErr != nil {
			_ = file.Close()
			return "", statErr
		}
		if !info.Mode().IsRegular() || info.Size() != value.size {
			_ = file.Close()
			return "", fmt.Errorf(
				"%w: source entry %q changed during fingerprinting",
				artifactstore.ErrConflict,
				value.relative,
			)
		}

		_, _ = io.WriteString(hash, "content\x00")
		written, copyErr := io.Copy(
			hash,
			io.LimitReader(file, value.size+1),
		)
		closeErr := file.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		if written != value.size {
			return "", fmt.Errorf(
				"%w: source entry %q changed during fingerprinting",
				artifactstore.ErrConflict,
				value.relative,
			)
		}
		_, _ = hash.Write([]byte{0})
		totalBytes += written
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}
