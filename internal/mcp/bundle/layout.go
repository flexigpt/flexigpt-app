package bundle

import (
	"fmt"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

func PackageAddressForBundle(
	logicalName basespec.LogicalName,
	logicalVersion basespec.LogicalVersion,
) (source.ManagedPackageAddress, error) {
	if logicalVersion == "" {
		logicalVersion = artifactbuiltin.UnversionedPackageVersion
	}
	return source.NewManagedPackageAddress(
		artifactbuiltin.MCPBundlePackageKind,
		logicalName,
		logicalVersion,
	)
}

func DocumentLocatorForPackage(
	address source.ManagedPackageAddress,
) (basespec.Locator, error) {
	if err := validateBundlePackageAddress(address); err != nil {
		return "", err
	}
	return address.FileLocator(artifactbuiltin.MCPBundleDocumentFileName)
}

func ValidateDocumentLocator(value basespec.Locator) error {
	if err := basespec.ValidatePortableLocator(value, false); err != nil {
		return err
	}
	if path.Base(string(value)) != string(artifactbuiltin.MCPBundleDocumentFileName) ||
		path.Dir(string(value)) == "." {
		return fmt.Errorf(
			"%w: MCP Bundle document locator must be nested and named %q",
			basespec.ErrInvalid,
			artifactbuiltin.MCPBundleDocumentFileName,
		)
	}
	return nil
}

func IsBundleDocumentLocator(value basespec.Locator) bool {
	return path.Base(string(value)) == string(artifactbuiltin.MCPBundleDocumentFileName)
}

func validateBundlePackageAddress(
	address source.ManagedPackageAddress,
) error {
	if err := address.Validate(); err != nil {
		return err
	}
	if address.Kind != artifactbuiltin.MCPBundlePackageKind {
		return fmt.Errorf(
			"%w: MCP Bundle package kind must be %q",
			basespec.ErrInvalid,
			artifactbuiltin.MCPBundlePackageKind,
		)
	}
	return nil
}
