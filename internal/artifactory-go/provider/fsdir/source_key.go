package fsdir

import (
	"fmt"
	"path/filepath"
	"strings"

	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func NormalizeFilesystemSourceRoot(
	raw string,
	label string,
) (string, error) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return "", fmt.Errorf("%w: %s is required", storeSpec.ErrInvalid, label)
	}
	absolute, err := filepath.Abs(raw)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absolute), nil
}

func FilesystemSourceStorageKey(
	prefix string,
	rootPath string,
) storeSpec.StorageKey {
	digest := strings.TrimPrefix(
		string(cryptoutil.DigestBytes([]byte(rootPath))),
		cryptoutil.DigestSHA256Prefix,
	)

	return storeSpec.StorageKey(prefix + "-hash" + digest[:24])
}
