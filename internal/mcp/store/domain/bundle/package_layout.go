package bundle

import (
	"fmt"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
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
	if address.Kind != artifactbuiltin.MCPBundlePackageKind {
		return "", fmt.Errorf(
			"%w: MCP Bundle package kind must be %q",
			basespec.ErrInvalid,
			artifactbuiltin.MCPBundlePackageKind,
		)
	}
	return address.FileLocator(artifactbuiltin.MCPBundleDocumentFileName)
}

func IsBundleDocumentLocator(value basespec.Locator) bool {
	if value.ValidatePortable(false) != nil ||
		path.Base(string(value)) != string(artifactbuiltin.MCPBundleDocumentFileName) ||
		path.Dir(string(value)) == "." {
		return false
	}
	return true
}

func validatePackageAddress(
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
