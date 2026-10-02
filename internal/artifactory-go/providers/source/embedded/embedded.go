package embedded

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"strings"

	sourceimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/internal/engine/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type Config struct {
	ProviderKey string        `json:"providerKey"`
	Root        model.Locator `json:"root"`
}

type Adapter struct {
	providers map[string]fs.FS
}

func New(providers map[string]fs.FS) (*Adapter, error) {
	output := make(map[string]fs.FS, len(providers))
	for key, provider := range providers {
		if err := model.ValidateIdentifier(
			"embedded provider key",
			key,
			model.MaxKindBytes,
		); err != nil {
			return nil, err
		}
		if provider == nil {
			return nil, fmt.Errorf(
				"%w: embedded provider %q is nil",
				model.ErrInvalid,
				key,
			)
		}
		output[key] = provider
	}
	return &Adapter{providers: output}, nil
}

func (*Adapter) Kind() source.SourceKind {
	return source.SourceKindEmbeddedDirectory
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
			model.ErrSourceUnavailable,
			config.ProviderKey,
		)
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(encoded), nil
}

func (a *Adapter) Open(
	ctx context.Context,
	value source.Source,
) (sourceimpl.Snapshot, error) {
	if value.Kind != source.SourceKindEmbeddedDirectory {
		return nil, fmt.Errorf(
			"%w: embedded adapter received source kind %q",
			model.ErrInvalid,
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
			model.ErrSourceUnavailable,
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
	canonical, err := jsonutil.CanonicalizeObject(raw, model.MaxConfigBytes)
	if err != nil {
		return Config{}, err
	}

	var config Config
	if err := jsonutil.DecodeCanonicalObjectBytesInto(
		canonical,
		&config,
		model.MaxConfigBytes,
	); err != nil {
		return Config{}, fmt.Errorf(
			"%w: decode embedded source config: %w",
			model.ErrInvalid,
			err,
		)
	}
	if err := model.ValidateIdentifier(
		"embedded provider key",
		config.ProviderKey,
		model.MaxKindBytes,
	); err != nil {
		return Config{}, err
	}
	if config.Root == "" {
		config.Root = "."
	}
	if err := config.Root.Validate(true); err != nil {
		return Config{}, err
	}
	return config, nil
}

func fingerprint(ctx context.Context, provider fs.FS) (string, error) {
	hash := cryptoutil.NewDigestWriter()
	_, _ = io.WriteString(hash, "artifact-embedded-tree/v1\x00")
	writeHeader := func(kind byte, name string, size uint64) {
		var encoded [8]byte
		_, _ = hash.Write([]byte{kind})
		binary.BigEndian.PutUint64(encoded[:], uint64(len(name)))
		_, _ = hash.Write(encoded[:])
		_, _ = io.WriteString(hash, name)
		binary.BigEndian.PutUint64(encoded[:], size)
		_, _ = hash.Write(encoded[:])
	}

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
		if entries > model.DefaultMaxEntries {
			return fmt.Errorf(
				"%w: embedded source exceeds %d entries",
				model.ErrInvalid,
				model.DefaultMaxEntries,
			)
		}
		if strings.Count(name, "/")+1 > model.DefaultMaxDepth {
			return fmt.Errorf(
				"%w: embedded source exceeds depth %d",
				model.ErrInvalid,
				model.DefaultMaxDepth,
			)
		}
		info, err := fs.Stat(provider, name)
		if err != nil {
			return err
		}
		if info.IsDir() {
			writeHeader('d', name, 0)
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if info.Size() < 0 || info.Size() > model.MaxScanBytes-totalBytes {
			return fmt.Errorf(
				"%w: embedded source exceeds byte limit",
				model.ErrInvalid,
			)
		}
		totalBytes += info.Size()

		file, err := provider.Open(name)
		if err != nil {
			return err
		}
		//nolint:gosec // Ok.
		writeHeader('f', name, uint64(info.Size()))
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
				model.ErrConflict,
				name,
			)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return string(hash.Digest()), nil
}
