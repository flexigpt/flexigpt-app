package topology

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// HydrationFingerprint returns the stable generic topology fingerprint for one
// artifact-family installer.
func HydrationFingerprint(
	schemaVersion string,
	declaration Declaration,
) (cryptoutil.Digest, error) {
	if err := basespec.ValidateRequiredText(
		"topology hydration schema version",
		schemaVersion,
		basespec.MaxVersionBytes,
	); err != nil {
		return "", err
	}
	if err := declaration.Validate(); err != nil {
		return "", err
	}
	return cryptoutil.CanonicalDigest(struct {
		SchemaVersion string      `json:"schemaVersion"`
		Topology      Declaration `json:"topology"`
	}{
		SchemaVersion: schemaVersion,
		Topology:      declaration,
	})
}

// PackageFingerprint returns a stable package hydration fingerprint.
//
// Callers own validation and deterministic ordering of Expectations because
// Artifact Store topology intentionally does not know domain expectation types.
func PackageFingerprint(
	packageRoot basespec.Locator,
	address source.ManagedPackageAddress,
	documentFile basespec.Locator,
	expectations any,
	packageFiles []source.ManagedPackageFile,
) (cryptoutil.Digest, error) {
	if err := packageRoot.ValidatePortable(false); err != nil {
		return "", err
	}
	if err := address.Validate(); err != nil {
		return "", err
	}
	if err := documentFile.ValidatePortable(false); err != nil {
		return "", err
	}
	files, err := source.NormalizeManagedPackageFiles(packageFiles)
	if err != nil {
		return "", err
	}

	type file struct {
		Locator basespec.Locator  `json:"locator"`
		Digest  cryptoutil.Digest `json:"digest"`
		Size    int64             `json:"size"`
	}
	values := make([]file, 0, len(files))
	for _, item := range files {
		values = append(values, file{
			Locator: item.Locator,
			Digest:  cryptoutil.DigestBytes(item.Content),
			Size:    int64(len(item.Content)),
		})
	}

	return cryptoutil.CanonicalDigest(struct {
		PackageRoot  basespec.Locator             `json:"packageRoot"`
		Address      source.ManagedPackageAddress `json:"address"`
		DocumentFile basespec.Locator             `json:"documentFile"`
		Expectations any                          `json:"expectations"`
		Files        []file                       `json:"files"`
	}{
		PackageRoot:  packageRoot,
		Address:      address,
		DocumentFile: documentFile,
		Expectations: expectations,
		Files:        values,
	})
}
