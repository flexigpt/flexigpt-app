package embedded

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	sourceimpl "github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type Config struct {
	ProviderKey string           `json:"providerKey"`
	Root        basespec.Locator `json:"root"`
}

type Adapter struct {
	providers map[string]fs.FS
}

func New(providers map[string]fs.FS) (*Adapter, error) {
	output := make(map[string]fs.FS, len(providers))
	for key, provider := range providers {
		if err := basespec.ValidateIdentifier(
			"embedded provider key",
			key,
			basespec.MaxKindBytes,
		); err != nil {
			return nil, err
		}
		if provider == nil {
			return nil, fmt.Errorf(
				"%w: embedded provider %q is nil",
				basespec.ErrInvalid,
				key,
			)
		}
		output[key] = provider
	}
	return &Adapter{providers: output}, nil
}

func (*Adapter) Kind() basespec.SourceKind {
	return basespec.SourceKindEmbeddedDirectory
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
			basespec.ErrSourceUnavailable,
			config.ProviderKey,
		)
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	canonical, err := jsonutil.CanonicalizeObject(
		encoded,
		basespec.MaxConfigBytes,
	)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(canonical), nil
}

func (a *Adapter) Open(
	ctx context.Context,
	value source.Source,
) (sourceimpl.Snapshot, error) {
	if value.Kind != basespec.SourceKindEmbeddedDirectory {
		return nil, fmt.Errorf(
			"%w: embedded adapter received source kind %q",
			basespec.ErrInvalid,
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
			basespec.ErrSourceUnavailable,
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

func decodeConfig(raw json.RawMessage) (Config, error) {
	canonical, err := jsonutil.CanonicalizeObject(raw, basespec.MaxConfigBytes)
	if err != nil {
		return Config{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()

	var config Config
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf(
			"%w: decode embedded source config: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if err := basespec.ValidateIdentifier(
		"embedded provider key",
		config.ProviderKey,
		basespec.MaxKindBytes,
	); err != nil {
		return Config{}, err
	}
	if config.Root == "" {
		config.Root = "."
	}
	if err := basespec.ValidateLocator(config.Root, true); err != nil {
		return Config{}, err
	}
	return config, nil
}

func fingerprint(ctx context.Context, provider fs.FS) (string, error) {
	hash := cryptoutil.NewDigestWriter()
	entries := 0
	var totalBytes int64

	err := fs.WalkDir(provider, ".", func(name string, _ fs.DirEntry, walkErr error) error {
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
		if entries > basespec.DefaultMaxEntries {
			return fmt.Errorf(
				"%w: embedded source exceeds %d entries",
				basespec.ErrInvalid,
				basespec.DefaultMaxEntries,
			)
		}
		if strings.Count(name, "/")+1 > basespec.DefaultMaxDepth {
			return fmt.Errorf(
				"%w: embedded source exceeds depth %d",
				basespec.ErrInvalid,
				basespec.DefaultMaxDepth,
			)
		}
		info, err := fs.Stat(provider, name)
		if err != nil {
			return err
		}
		if info.IsDir() {
			_, _ = io.WriteString(hash, "d\x00"+name+"\x00")
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if info.Size() < 0 || info.Size() > basespec.MaxScanBytes-totalBytes {
			return fmt.Errorf(
				"%w: embedded source exceeds byte limit",
				basespec.ErrInvalid,
			)
		}
		totalBytes += info.Size()

		file, err := provider.Open(name)
		if err != nil {
			return err
		}
		_, _ = io.WriteString(hash, "f\x00"+name+"\x00")
		written, copyErr := io.Copy(hash, io.LimitReader(file, info.Size()+1))
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if written != info.Size() {
			return fmt.Errorf(
				"%w: embedded source entry %q changed during fingerprinting",
				basespec.ErrConflict,
				name,
			)
		}
		_, _ = hash.Write([]byte{0})
		return nil
	})
	if err != nil {
		return "", err
	}
	return string(hash.Digest()), nil
}
