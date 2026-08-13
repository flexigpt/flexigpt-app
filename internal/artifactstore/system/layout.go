package system

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

const (
	StoreManifestFileName     = "store.json"
	StoreMetadataFileName     = "app.sqlite"
	StoreContentDirectoryName = "content"
	storeFormat               = "flexigpt-artifactstore/v1"
	contentLayout             = "semantic-packages/v1"
	storeBaseDirectoryMode    = 0o750
	storeManifestFileMode     = 0o600
)

type storeManifest struct {
	Format        string `json:"format"`
	ContentLayout string `json:"contentLayout"`
}

func ensureStoreLayout(base string) error {
	if err := os.MkdirAll(base, storeBaseDirectoryMode); err != nil {
		return err
	}

	manifestPath := filepath.Join(base, StoreManifestFileName)
	raw, err := os.ReadFile(manifestPath)
	switch {
	case errors.Is(err, os.ErrNotExist):
		entries, readErr := os.ReadDir(base)
		if readErr != nil {
			return readErr
		}
		if len(entries) != 0 {
			return fmt.Errorf(
				"%w: Artifact Store base directory is non-empty but has no %s",
				basespec.ErrUnsupported,
				StoreManifestFileName,
			)
		}

		raw, err = json.Marshal(storeManifest{
			Format:        storeFormat,
			ContentLayout: contentLayout,
		})
		if err != nil {
			return err
		}
		raw = append(raw, '\n')

		temporary, err := os.CreateTemp(base, ".store.json-*")
		if err != nil {
			return err
		}
		temporaryName := temporary.Name()

		if err := temporary.Chmod(storeManifestFileMode); err != nil {
			_ = temporary.Close()
			_ = os.Remove(temporaryName)
			return err
		}
		if _, err := temporary.Write(raw); err != nil {
			_ = temporary.Close()
			_ = os.Remove(temporaryName)
			return err
		}
		if err := temporary.Close(); err != nil {
			_ = os.Remove(temporaryName)
			return err
		}
		if err := os.Rename(temporaryName, manifestPath); err != nil {
			_ = os.Remove(temporaryName)
			return err
		}

	case err != nil:
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()

	var manifest storeManifest
	if err := decoder.Decode(&manifest); err != nil {
		return fmt.Errorf(
			"%w: decode Artifact Store layout manifest: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("artifact store layout manifest has trailing JSON")
		}
		return fmt.Errorf("%w: %w", basespec.ErrInvalid, err)
	}

	if manifest.Format != storeFormat ||
		manifest.ContentLayout != contentLayout {
		return fmt.Errorf(
			"%w: unsupported Artifact Store layout %q/%q",
			basespec.ErrUnsupported,
			manifest.Format,
			manifest.ContentLayout,
		)
	}

	return os.MkdirAll(
		filepath.Join(base, StoreContentDirectoryName),
		storeBaseDirectoryMode,
	)
}
