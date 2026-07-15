package sourcedriver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

// EmbeddedFSProviderRegistrar is implemented by embedded source drivers that
// accept application-owned read-only filesystem providers.
type EmbeddedFSProviderRegistrar interface {
	RegisterProvider(providerKey string, provider fs.FS) error
}

// EmbeddedFSDirectoryDriver exposes application-registered read-only fs.FS
// providers through the generic SourceDriver contract.
type EmbeddedFSDirectoryDriver struct {
	mu        sync.RWMutex
	providers map[string]fs.FS
}

func NewEmbeddedFSDirectoryDriver() *EmbeddedFSDirectoryDriver {
	return &EmbeddedFSDirectoryDriver{providers: make(map[string]fs.FS)}
}

func (*EmbeddedFSDirectoryDriver) Kind() spec.SourceKind {
	return spec.SourceKindEmbeddedFSDirectory
}

func (d *EmbeddedFSDirectoryDriver) RegisterProvider(providerKey string, provider fs.FS) error {
	if d == nil || provider == nil {
		return fmt.Errorf("%w: embedded filesystem provider is nil", spec.ErrInvalidRequest)
	}
	config := spec.EmbeddedFSDirectorySourceConfig{
		ProviderKey: providerKey,
		RootLocator: ".",
	}
	if err := spec.ValidateEmbeddedFSDirectorySourceConfig(config); err != nil {
		return fmt.Errorf("%w: embedded filesystem provider key: %w", spec.ErrInvalidRequest, err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.providers[providerKey]; exists {
		return fmt.Errorf("%w: embedded filesystem provider %q", spec.ErrConflict, providerKey)
	}
	d.providers[providerKey] = provider
	return nil
}

func (*EmbeddedFSDirectoryDriver) ValidateConfig(
	_ context.Context,
	raw json.RawMessage,
) []spec.Diagnostic {
	if _, err := decodeEmbeddedFSConfig(raw); err != nil {
		return []spec.Diagnostic{{
			Severity: spec.DiagnosticSeverityError,
			Code:     "artifactstore.source.config.invalid",
			Message:  err.Error(),
		}}
	}
	return nil
}

func (d *EmbeddedFSDirectoryDriver) Snapshot(
	ctx context.Context,
	source spec.ArtifactSource,
) (spec.SourceGeneration, error) {
	entries := make([]spec.SourceEntry, 0)
	if err := d.Walk(ctx, source, ".", func(_ context.Context, entry spec.SourceEntry) error {
		entries = append(entries, entry)
		return nil
	}); err != nil {
		return "", err
	}
	sort.Slice(entries, func(left, right int) bool {
		return entries[left].Locator < entries[right].Locator
	})

	hash := sha256.New()
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		switch {
		case entry.IsDirectory:
			_, _ = io.WriteString(hash, "d\x00"+string(entry.Locator)+"\x00")
		case entry.IsRegular:
			reader, err := d.Open(ctx, source, entry.Locator)
			if err != nil {
				return "", err
			}
			_, _ = io.WriteString(hash, "f\x00"+string(entry.Locator)+"\x00")
			_, copyErr := io.Copy(hash, reader)
			closeErr := reader.Close()
			if copyErr != nil {
				return "", copyErr
			}
			if closeErr != nil {
				return "", closeErr
			}
			_, _ = hash.Write([]byte{0})
		default:
			_, _ = io.WriteString(hash, "o\x00"+string(entry.Locator)+"\x00")
		}
	}
	return spec.SourceGeneration("sha256:" + hex.EncodeToString(hash.Sum(nil))), nil
}

func (d *EmbeddedFSDirectoryDriver) Open(
	ctx context.Context,
	source spec.ArtifactSource,
	locator spec.SourceLocator,
) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	provider, err := d.providerFor(source)
	if err != nil {
		return nil, err
	}
	name, err := embeddedFSName(locator)
	if err != nil {
		return nil, err
	}
	info, err := fs.Stat(provider, name)
	if err != nil {
		return nil, mapEmbeddedFSError(locator, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: source locator %q is not a regular file", spec.ErrInvalidRequest, locator)
	}
	file, err := provider.Open(name)
	if err != nil {
		return nil, mapEmbeddedFSError(locator, err)
	}
	return file, nil
}

func (d *EmbeddedFSDirectoryDriver) Walk(
	ctx context.Context,
	source spec.ArtifactSource,
	root spec.SourceLocator,
	walk spec.WalkFunc,
) error {
	if walk == nil {
		return fmt.Errorf("%w: walk callback is nil", spec.ErrInvalidRequest)
	}
	rootEntry, err := d.Stat(ctx, source, root)
	if err != nil {
		return err
	}
	if !rootEntry.IsDirectory {
		return fmt.Errorf("%w: walk root %q is not a directory", spec.ErrInvalidRequest, root)
	}

	var visit func(spec.SourceLocator) error
	visit = func(directory spec.SourceLocator) error {
		entries, err := d.ReadDir(ctx, source, directory)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := walk(ctx, entry); err != nil {
				return err
			}
			if entry.IsDirectory {
				if err := visit(entry.Locator); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return visit(root)
}

func (d *EmbeddedFSDirectoryDriver) Stat(
	ctx context.Context,
	source spec.ArtifactSource,
	locator spec.SourceLocator,
) (spec.SourceEntry, error) {
	if err := ctx.Err(); err != nil {
		return spec.SourceEntry{}, err
	}
	provider, err := d.providerFor(source)
	if err != nil {
		return spec.SourceEntry{}, err
	}
	name, err := embeddedFSName(locator)
	if err != nil {
		return spec.SourceEntry{}, err
	}
	info, err := fs.Stat(provider, name)
	if err != nil {
		return spec.SourceEntry{}, mapEmbeddedFSError(locator, err)
	}
	return embeddedSourceEntry(locator, info), nil
}

func (d *EmbeddedFSDirectoryDriver) ReadDir(
	ctx context.Context,
	source spec.ArtifactSource,
	locator spec.SourceLocator,
) ([]spec.SourceEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	provider, err := d.providerFor(source)
	if err != nil {
		return nil, err
	}
	name, err := embeddedFSName(locator)
	if err != nil {
		return nil, err
	}
	dirEntries, err := fs.ReadDir(provider, name)
	if err != nil {
		return nil, mapEmbeddedFSError(locator, err)
	}
	entries := make([]spec.SourceEntry, 0, len(dirEntries))
	for _, dirEntry := range dirEntries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		childLocator, err := joinEmbeddedLocator(locator, dirEntry.Name())
		if err != nil {
			return nil, err
		}
		info, err := dirEntry.Info()
		if err != nil {
			return nil, mapEmbeddedFSError(childLocator, err)
		}
		entries = append(entries, embeddedSourceEntry(childLocator, info))
	}
	sort.Slice(entries, func(left, right int) bool {
		return entries[left].Locator < entries[right].Locator
	})
	return entries, nil
}

func (d *EmbeddedFSDirectoryDriver) providerFor(source spec.ArtifactSource) (fs.FS, error) {
	if d == nil {
		return nil, fmt.Errorf("%w: embedded filesystem driver is nil", spec.ErrDriverUnavailable)
	}
	if source.Kind != spec.SourceKindEmbeddedFSDirectory {
		return nil, fmt.Errorf(
			"%w: embedded filesystem driver received source kind %q",
			spec.ErrInvalidRequest,
			source.Kind,
		)
	}
	config, err := decodeEmbeddedFSConfig(source.Config)
	if err != nil {
		return nil, err
	}

	d.mu.RLock()
	provider, ok := d.providers[config.ProviderKey]
	d.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf(
			"%w: embedded filesystem provider %q",
			spec.ErrDriverUnavailable,
			config.ProviderKey,
		)
	}
	if config.RootLocator == "." {
		return provider, nil
	}
	sub, err := fs.Sub(provider, string(config.RootLocator))
	if err != nil {
		return nil, fmt.Errorf("resolve embedded filesystem root %q: %w", config.RootLocator, err)
	}
	return sub, nil
}

func decodeEmbeddedFSConfig(raw json.RawMessage) (spec.EmbeddedFSDirectorySourceConfig, error) {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	var config spec.EmbeddedFSDirectorySourceConfig
	if err := decoder.Decode(&config); err != nil {
		return spec.EmbeddedFSDirectorySourceConfig{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return spec.EmbeddedFSDirectorySourceConfig{}, errors.New("source config contains trailing JSON")
		}
		return spec.EmbeddedFSDirectorySourceConfig{}, err
	}
	if err := spec.ValidateEmbeddedFSDirectorySourceConfig(config); err != nil {
		return spec.EmbeddedFSDirectorySourceConfig{}, err
	}
	return config, nil
}

func embeddedFSName(locator spec.SourceLocator) (string, error) {
	value := string(locator)
	if value == "." {
		return ".", nil
	}
	if !fs.ValidPath(value) {
		return "", fmt.Errorf("%w: invalid embedded source locator %q", spec.ErrInvalidRequest, locator)
	}
	return value, nil
}

func joinEmbeddedLocator(parent spec.SourceLocator, name string) (spec.SourceLocator, error) {
	if name == "" || !fs.ValidPath(name) || strings.Contains(name, "/") {
		return "", fmt.Errorf("%w: invalid embedded source entry name %q", spec.ErrInvalidRequest, name)
	}
	if parent == "." {
		return spec.SourceLocator(name), nil
	}
	joined := path.Join(string(parent), name)
	if !fs.ValidPath(joined) {
		return "", fmt.Errorf("%w: invalid embedded source locator %q", spec.ErrInvalidRequest, joined)
	}
	return spec.SourceLocator(joined), nil
}

func embeddedSourceEntry(locator spec.SourceLocator, info fs.FileInfo) spec.SourceEntry {
	modified := info.ModTime()
	if !modified.IsZero() {
		modified = modified.UTC()
	}
	mode := info.Mode()
	return spec.SourceEntry{
		Locator:     locator,
		Name:        info.Name(),
		Mode:        uint32(mode),
		SizeBytes:   info.Size(),
		ModifiedAt:  modified,
		IsDirectory: mode.IsDir(),
		IsRegular:   mode.IsRegular(),
		IsSymlink:   mode&fs.ModeSymlink != 0,
	}
}

func mapEmbeddedFSError(locator spec.SourceLocator, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf(
			"%w: embedded source locator %q: %w",
			spec.ErrNotFound,
			locator,
			err,
		)
	}
	if errors.Is(err, fs.ErrPermission) {
		return fmt.Errorf("access embedded source locator %q: %w", locator, err)
	}
	return fmt.Errorf("access embedded source locator %q: %w", locator, err)
}

var (
	_ spec.SourceDriver           = (*EmbeddedFSDirectoryDriver)(nil)
	_ EmbeddedFSProviderRegistrar = (*EmbeddedFSDirectoryDriver)(nil)
)
