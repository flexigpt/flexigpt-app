package fsdir

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

const Kind artifactstore.SourceKind = "fs-directory"

type Config struct {
	RootPath string `json:"rootPath"`
}

type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (*Adapter) Kind() artifactstore.SourceKind {
	return Kind
}

func (*Adapter) NormalizeConfig(
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
	if strings.TrimSpace(config.RootPath) == "" {
		return nil, fmt.Errorf("%w: filesystem root path is required", artifactstore.ErrInvalid)
	}
	if !filepath.IsAbs(config.RootPath) {
		return nil, fmt.Errorf("%w: filesystem root path must be absolute", artifactstore.ErrInvalid)
	}
	config.RootPath = filepath.Clean(config.RootPath)

	info, err := os.Stat(config.RootPath)
	if err != nil {
		return nil, fmt.Errorf("stat filesystem source root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: filesystem source root is not a directory", artifactstore.ErrInvalid)
	}

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

func (*Adapter) Open(
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
	generation, err := fingerprint(ctx, config.RootPath)
	if err != nil {
		return nil, err
	}
	return &snapshot{
		root:       config.RootPath,
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
		return Config{}, err
	}
	if !filepath.IsAbs(config.RootPath) ||
		filepath.Clean(config.RootPath) != config.RootPath {
		return Config{}, fmt.Errorf(
			"%w: invalid normalized filesystem root",
			artifactstore.ErrInvalid,
		)
	}
	return config, nil
}

func fingerprint(ctx context.Context, root string) (string, error) {
	type item struct {
		relative string
		mode     os.FileMode
		size     int64
		modified time.Time
	}

	values := make([]item, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if path == root {
			return nil
		}
		if len(values) >= artifactstore.DefaultMaxEntries {
			return fmt.Errorf(
				"%w: source exceeds %d entries",
				artifactstore.ErrInvalid,
				artifactstore.DefaultMaxEntries,
			)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if strings.Count(filepath.ToSlash(relative), "/")+1 >
			artifactstore.DefaultMaxDepth {
			return fmt.Errorf(
				"%w: source exceeds traversal depth %d",
				artifactstore.ErrInvalid,
				artifactstore.DefaultMaxDepth,
			)
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf(
				"%w: source contains symbolic link %q",
				artifactstore.ErrInvalid,
				filepath.ToSlash(relative),
			)
		}
		values = append(values, item{
			relative: filepath.ToSlash(relative),
			mode:     info.Mode(),
			size:     info.Size(),
			modified: info.ModTime().UTC(),
		})
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Slice(values, func(left, right int) bool {
		return values[left].relative < values[right].relative
	})

	hash := sha256.New()
	for _, value := range values {
		_, _ = fmt.Fprintf(
			hash,
			"%s\x00%d\x00%d\x00%s\x00",
			value.relative,
			value.mode,
			value.size,
			value.modified.Format(time.RFC3339Nano),
		)
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}
