package iofs

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/driver"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const Kind sourceModel.SourceKind = "embedded-directory"

type Config struct {
	ProviderKey string       `json:"providerKey"`
	Root        spec.Locator `json:"root"`
}

type Adapter struct {
	providers map[string]fs.FS
	immutable map[string]immutableProvider
}

func New(ctx context.Context, providers map[string]fs.FS) (*Adapter, error) {
	registrations := make(
		map[string]ProviderRegistration,
		len(providers),
	)
	for key, provider := range providers {
		registrations[key] = ProviderRegistration{
			Filesystem: provider,
		}
	}
	return NewWithRegistrations(ctx, registrations)
}

func (*Adapter) Kind() sourceModel.SourceKind {
	return Kind
}

func (a *Adapter) NormalizeConfig(
	ctx context.Context,
	raw json.RawMessage,
) (json.RawMessage, error) {
	config, err := decodeConfig(raw)
	if err != nil {
		return nil, err
	}
	if _, exists := a.providers[config.ProviderKey]; !exists {
		return nil, fmt.Errorf(
			"%w: embedded provider %q",
			spec.ErrSourceUnavailable,
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
	value sourceModel.Source,
) (driver.Snapshot, error) {
	config, err := decodeConfig(value.Config)
	if err != nil {
		return nil, err
	}
	provider, exists := a.providers[config.ProviderKey]
	if !exists {
		return nil, fmt.Errorf(
			"%w: embedded provider %q",
			spec.ErrSourceUnavailable,
			config.ProviderKey,
		)
	}
	if config.Root != "." {
		provider, err = fs.Sub(provider, string(config.Root))
		if err != nil {
			return nil, fmt.Errorf("open embedded root %q: %w", config.Root, err)
		}
	}
	var generation string
	immutable := false
	if evidence, found := a.immutable[config.ProviderKey]; found {
		generation, err = evidence.generation(
			config.ProviderKey,
			config.Root,
		)
		immutable = true
	} else {
		generation, err = fingerprint(ctx, provider)
	}
	if err != nil {
		return nil, err
	}
	return &snapshot{
		provider:   provider,
		generation: generation,
		immutable:  immutable,
	}, nil
}

func decodeConfig(raw json.RawMessage) (Config, error) {
	canonical, err := jsonutil.CanonicalizeObject(raw, spec.MaxConfigBytes)
	if err != nil {
		return Config{}, err
	}

	var config Config
	if err := jsonutil.DecodeCanonicalObjectBytesInto(
		canonical,
		&config,
		spec.MaxConfigBytes,
	); err != nil {
		return Config{}, fmt.Errorf(
			"%w: decode embedded source config: %w",
			spec.ErrInvalid,
			err,
		)
	}
	if err := spec.ValidateIdentifier(
		"embedded provider key",
		config.ProviderKey,
		spec.MaxKindBytes,
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
		if entries > spec.DefaultMaxEntries {
			return fmt.Errorf(
				"%w: embedded source exceeds %d entries",
				spec.ErrInvalid,
				spec.DefaultMaxEntries,
			)
		}
		if strings.Count(name, "/")+1 > spec.DefaultMaxDepth {
			return fmt.Errorf(
				"%w: embedded source exceeds depth %d",
				spec.ErrInvalid,
				spec.DefaultMaxDepth,
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
		if info.Size() < 0 || info.Size() > spec.MaxScanBytes-totalBytes {
			return fmt.Errorf(
				"%w: embedded source exceeds byte limit",
				spec.ErrInvalid,
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
				spec.ErrConflict,
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
