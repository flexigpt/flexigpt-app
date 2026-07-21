package embedded

import (
	"bytes"
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

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

const Kind artifactstore.SourceKind = "embedded-directory"

type Config struct {
	ProviderKey string                `json:"providerKey"`
	Root        artifactstore.Locator `json:"root"`
}

type Adapter struct {
	providers map[string]fs.FS
}

func New(providers map[string]fs.FS) (*Adapter, error) {
	output := make(map[string]fs.FS, len(providers))
	for key, provider := range providers {
		if err := artifactstore.ValidateIdentifier(
			"embedded provider key",
			key,
			artifactstore.MaxKindBytes,
		); err != nil {
			return nil, err
		}
		if provider == nil {
			return nil, fmt.Errorf(
				"%w: embedded provider %q is nil",
				artifactstore.ErrInvalid,
				key,
			)
		}
		output[key] = provider
	}
	return &Adapter{providers: output}, nil
}

func (*Adapter) Kind() artifactstore.SourceKind {
	return Kind
}

func (a *Adapter) NormalizeConfig(
	ctx context.Context,
	raw json.RawMessage,
) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	config, err := decodeConfig(raw)
	if err != nil {
		return nil, err
	}
	if _, exists := a.providers[config.ProviderKey]; !exists {
		return nil, fmt.Errorf(
			"%w: embedded provider %q",
			artifactstore.ErrSourceUnavailable,
			config.ProviderKey,
		)
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	canonical, err := jsoncanon.CanonicalizeObject(
		encoded,
		artifactstore.MaxConfigBytes,
	)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(canonical), nil
}

func (a *Adapter) Open(
	ctx context.Context,
	value source.Source,
) (source.Snapshot, error) {
	if value.Kind != Kind {
		return nil, fmt.Errorf(
			"%w: embedded adapter received source kind %q",
			artifactstore.ErrInvalid,
			value.Kind,
		)
	}
	config, err := decodeConfig(value.Config)
	if err != nil {
		return nil, err
	}
	provider, exists := a.providers[config.ProviderKey]
	if !exists {
		return nil, fmt.Errorf(
			"%w: embedded provider %q",
			artifactstore.ErrSourceUnavailable,
			config.ProviderKey,
		)
	}
	if config.Root != "." {
		provider, err = fs.Sub(provider, string(config.Root))
		if err != nil {
			return nil, fmt.Errorf("open embedded root %q: %w", config.Root, err)
		}
	}
	generation, err := fingerprint(ctx, provider)
	if err != nil {
		return nil, err
	}
	return &snapshot{
		provider:   provider,
		generation: generation,
	}, nil
}

type snapshot struct {
	provider   fs.FS
	generation string
	closed     bool
}

func (s *snapshot) Generation() string {
	return s.generation
}

func (s *snapshot) Stat(
	ctx context.Context,
	locator artifactstore.Locator,
) (source.Entry, error) {
	if err := s.ensureOpen(ctx); err != nil {
		return source.Entry{}, err
	}
	name, err := fsName(locator)
	if err != nil {
		return source.Entry{}, err
	}
	info, err := fs.Stat(s.provider, name)
	if errors.Is(err, fs.ErrNotExist) {
		return source.Entry{}, fmt.Errorf(
			"%w: embedded locator %q",
			artifactstore.ErrNotFound,
			locator,
		)
	}
	if err != nil {
		return source.Entry{}, err
	}
	return entryFromInfo(locator, info), nil
}

func (s *snapshot) ReadDir(
	ctx context.Context,
	locator artifactstore.Locator,
) ([]source.Entry, error) {
	if err := s.ensureOpen(ctx); err != nil {
		return nil, err
	}
	name, err := fsName(locator)
	if err != nil {
		return nil, err
	}
	values, err := fs.ReadDir(s.provider, name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf(
			"%w: embedded directory %q",
			artifactstore.ErrNotFound,
			locator,
		)
	}
	if err != nil {
		return nil, err
	}

	output := make([]source.Entry, 0, len(values))
	for _, value := range values {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		child, err := joinLocator(locator, value.Name())
		if err != nil {
			return nil, err
		}
		info, err := value.Info()
		if err != nil {
			return nil, err
		}
		output = append(output, entryFromInfo(child, info))
	}
	sort.Slice(output, func(left, right int) bool {
		return output[left].Locator < output[right].Locator
	})
	return output, nil
}

func (s *snapshot) Open(
	ctx context.Context,
	locator artifactstore.Locator,
) (io.ReadCloser, error) {
	if err := s.ensureOpen(ctx); err != nil {
		return nil, err
	}
	name, err := fsName(locator)
	if err != nil {
		return nil, err
	}
	info, err := fs.Stat(s.provider, name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf(
			"%w: embedded locator %q is not a regular file",
			artifactstore.ErrInvalid,
			locator,
		)
	}
	file, err := s.provider.Open(name)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (s *snapshot) Confirm(ctx context.Context) error {
	if err := s.ensureOpen(ctx); err != nil {
		return err
	}
	current, err := fingerprint(ctx, s.provider)
	if err != nil {
		return err
	}
	if current != s.generation {
		return fmt.Errorf(
			"%w: embedded source changed during discovery",
			artifactstore.ErrConflict,
		)
	}
	return nil
}

func (s *snapshot) Close() error {
	s.closed = true
	return nil
}

func (s *snapshot) ensureOpen(ctx context.Context) error {
	if s == nil || s.closed {
		return artifactstore.ErrClosed
	}
	return ctx.Err()
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
		return Config{}, fmt.Errorf(
			"%w: decode embedded source config: %w",
			artifactstore.ErrInvalid,
			err,
		)
	}
	if err := artifactstore.ValidateIdentifier(
		"embedded provider key",
		config.ProviderKey,
		artifactstore.MaxKindBytes,
	); err != nil {
		return Config{}, err
	}
	if config.Root == "" {
		config.Root = "."
	}
	if err := artifactstore.ValidateLocator(config.Root, true); err != nil {
		return Config{}, err
	}
	return config, nil
}

func fingerprint(ctx context.Context, provider fs.FS) (string, error) {
	hash := sha256.New()
	entries := 0
	var totalBytes int64

	err := fs.WalkDir(provider, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if name == "." {
			return nil
		}
		entries++
		if entries > artifactstore.DefaultMaxEntries {
			return fmt.Errorf(
				"%w: embedded source exceeds %d entries",
				artifactstore.ErrInvalid,
				artifactstore.DefaultMaxEntries,
			)
		}
		if strings.Count(name, "/")+1 > artifactstore.DefaultMaxDepth {
			return fmt.Errorf(
				"%w: embedded source exceeds depth %d",
				artifactstore.ErrInvalid,
				artifactstore.DefaultMaxDepth,
			)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf(
				"%w: embedded source contains symbolic link %q",
				artifactstore.ErrInvalid,
				name,
			)
		}
		if entry.IsDir() {
			_, _ = io.WriteString(hash, "d\x00"+name+"\x00")
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf(
				"%w: embedded source contains unsupported entry %q",
				artifactstore.ErrInvalid,
				name,
			)
		}
		if info.Size() < 0 || info.Size() > artifactstore.MaxScanBytes-totalBytes {
			return fmt.Errorf(
				"%w: embedded source exceeds byte limit",
				artifactstore.ErrInvalid,
			)
		}
		totalBytes += info.Size()

		file, err := provider.Open(name)
		if err != nil {
			return err
		}
		_, _ = io.WriteString(hash, "f\x00"+name+"\x00")
		_, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		_, _ = hash.Write([]byte{0})
		return nil
	})
	if err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func fsName(locator artifactstore.Locator) (string, error) {
	if err := artifactstore.ValidateLocator(locator, true); err != nil {
		return "", err
	}
	if locator == "." {
		return ".", nil
	}
	if !fs.ValidPath(string(locator)) {
		return "", fmt.Errorf(
			"%w: invalid embedded locator %q",
			artifactstore.ErrInvalid,
			locator,
		)
	}
	return string(locator), nil
}

func joinLocator(
	parent artifactstore.Locator,
	name string,
) (artifactstore.Locator, error) {
	if name == "" || strings.Contains(name, "/") || !fs.ValidPath(name) {
		return "", fmt.Errorf(
			"%w: invalid embedded entry name %q",
			artifactstore.ErrInvalid,
			name,
		)
	}
	if parent == "." {
		return artifactstore.Locator(name), nil
	}
	return artifactstore.Locator(path.Join(string(parent), name)), nil
}

func entryFromInfo(
	locator artifactstore.Locator,
	info fs.FileInfo,
) source.Entry {
	return source.Entry{
		Locator:     locator,
		Name:        info.Name(),
		SizeBytes:   info.Size(),
		Mode:        uint32(info.Mode()),
		ModifiedAt:  info.ModTime().UTC(),
		IsDirectory: info.IsDir(),
		IsRegular:   info.Mode().IsRegular(),
		IsSymlink:   info.Mode()&fs.ModeSymlink != 0,
	}
}

var _ source.Adapter = (*Adapter)(nil)
