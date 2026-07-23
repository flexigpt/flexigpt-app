package embedded

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
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
			return nil
		}
		if entry.IsDir() {
			_, _ = io.WriteString(hash, "d\x00"+name+"\x00")
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
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
