package system

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type storeManifest struct {
	Format        string `json:"format"`
	ContentLayout string `json:"contentLayout"`
}

func ensureStoreLayout(base string) error {
	if err := os.MkdirAll(
		base,
		os.FileMode(model.ArtifactStoreDirectoryMode),
	); err != nil {
		return err
	}

	manifestPath := filepath.Join(
		base,
		model.ArtifactStoreManifestFileName,
	)
	raw, err := os.ReadFile(manifestPath)
	switch {
	case errors.Is(err, os.ErrNotExist):
		if err := removeStaleManifestTemporaryFiles(base); err != nil {
			return err
		}

		raw, err = json.Marshal(storeManifest{
			Format:        model.ArtifactStoreFormat,
			ContentLayout: model.ArtifactStoreContentLayout,
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
	if manifest.Format != model.ArtifactStoreFormat ||
		manifest.ContentLayout != model.ArtifactStoreContentLayout {
		return fmt.Errorf(
			"%w: unsupported Artifact Store layout %q/%q",
			model.ErrUnsupported,
			manifest.Format,
			manifest.ContentLayout,
		)
	}

	for _, directory := range []string{
		model.ArtifactStoreContentDirectoryName,
		model.ArtifactStoreStagingDirectoryName,
	} {
		if err := os.MkdirAll(
			filepath.Join(base, directory),
			os.FileMode(model.ArtifactStoreDirectoryMode),
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
			model.ArtifactStoreManifestTemporaryName,
		) {
			continue
		}
		if entry.IsDir() {
			return fmt.Errorf(
				"%w: invalid Artifact Store manifest temporary directory %q",
				model.ErrInvalid,
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
		model.ArtifactStoreManifestTemporaryName,
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
		os.FileMode(model.ArtifactStoreManifestMode),
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
	manifest, err := jsonutil.DecodeCanonicalObject[storeManifest](
		raw,
		model.MaxConfigBytes,
	)
	if err != nil {
		return storeManifest{}, fmt.Errorf(
			"%w: decode artifact store layout manifest: %w",
			model.ErrInvalid,
			err,
		)
	}
	return manifest, nil
}
