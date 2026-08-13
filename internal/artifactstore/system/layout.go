package system

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	builtinSchema "github.com/flexigpt/flexigpt-app/internal/builtin/schema"
)

type storeManifest struct {
	Format        string `json:"format"`
	ContentLayout string `json:"contentLayout"`
}

func ensureStoreLayout(base string) error {
	if err := os.MkdirAll(
		base,
		os.FileMode(builtinSchema.ArtifactStoreDirectoryMode),
	); err != nil {
		return err
	}

	manifestPath := filepath.Join(
		base,
		builtinSchema.ArtifactStoreManifestFileName,
	)
	raw, err := os.ReadFile(manifestPath)
	switch {
	case errors.Is(err, os.ErrNotExist):
		if err := removeStaleManifestTemporaryFiles(base); err != nil {
			return err
		}
		entries, err := os.ReadDir(base)
		if err != nil {
			return err
		}
		if len(entries) != 0 {
			return fmt.Errorf(
				"%w: Artifact Store base directory is non-empty but has no %s",
				basespec.ErrUnsupported,
				builtinSchema.ArtifactStoreManifestFileName,
			)
		}

		raw, err = json.Marshal(storeManifest{
			Format:        builtinSchema.ArtifactStoreFormat,
			ContentLayout: builtinSchema.ArtifactStoreContentLayout,
		})
		if err != nil {
			return err
		}
		raw = append(raw, '\n')
		if err := writeNewStoreManifest(manifestPath, raw); err != nil {
			return err
		}

	case err != nil:
		return err
	}

	manifest, err := decodeStoreManifest(raw)
	if err != nil {
		return err
	}
	if manifest.Format != builtinSchema.ArtifactStoreFormat ||
		manifest.ContentLayout != builtinSchema.ArtifactStoreContentLayout {
		return fmt.Errorf(
			"%w: unsupported Artifact Store layout %q/%q",
			basespec.ErrUnsupported,
			manifest.Format,
			manifest.ContentLayout,
		)
	}

	for _, directory := range []string{
		builtinSchema.ArtifactStoreContentDirectoryName,
		builtinSchema.ArtifactStoreStagingDirectoryName,
	} {
		if err := os.MkdirAll(
			filepath.Join(base, directory),
			os.FileMode(builtinSchema.ArtifactStoreDirectoryMode),
		); err != nil {
			return err
		}
	}
	return nil
}

func removeStaleManifestTemporaryFiles(base string) error {
	entries, err := os.ReadDir(base)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !strings.HasPrefix(
			entry.Name(),
			builtinSchema.ArtifactStoreManifestTemporaryName,
		) {
			continue
		}
		if entry.IsDir() {
			return fmt.Errorf(
				"%w: invalid Artifact Store manifest temporary directory %q",
				basespec.ErrInvalid,
				entry.Name(),
			)
		}
		if err := os.Remove(filepath.Join(base, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func writeNewStoreManifest(
	manifestPath string,
	raw []byte,
) error {
	base := filepath.Dir(manifestPath)
	temporary, err := os.CreateTemp(
		base,
		builtinSchema.ArtifactStoreManifestTemporaryName,
	)
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()

	cleanup := func(cause error) error {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
		return cause
	}

	if err := temporary.Chmod(
		os.FileMode(builtinSchema.ArtifactStoreManifestMode),
	); err != nil {
		return cleanup(err)
	}
	if _, err := temporary.Write(raw); err != nil {
		return cleanup(err)
	}
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporaryPath)
		return err
	}
	if err := os.Rename(temporaryPath, manifestPath); err != nil {
		_ = os.Remove(temporaryPath)
		return err
	}
	return nil
}

func decodeStoreManifest(raw []byte) (storeManifest, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()

	var manifest storeManifest
	if err := decoder.Decode(&manifest); err != nil {
		return storeManifest{}, fmt.Errorf(
			"%w: decode artifact store layout manifest: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New(
				"artifact store layout manifest has trailing JSON",
			)
		}
		return storeManifest{}, fmt.Errorf(
			"%w: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	return manifest, nil
}
