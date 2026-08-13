package bundle

import (
	"fmt"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	builtinSchema "github.com/flexigpt/flexigpt-app/internal/builtin/schema"
)

const (
	ManagedPackageKind      = builtinSchema.MCPBundlePackageKind
	ManagedDocumentFileName = builtinSchema.MCPBundleDocumentFileName
)

func PackageAddressForBundle(
	logicalName basespec.LogicalName,
	logicalVersion basespec.LogicalVersion,
) (source.ManagedPackageAddress, error) {
	if logicalVersion == "" {
		logicalVersion = builtinSchema.UnversionedPackageVersion
	}
	return source.NewManagedPackageAddress(
		ManagedPackageKind,
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
	return address.FileLocator(ManagedDocumentFileName)
}

func ValidateDocumentLocator(value basespec.Locator) error {
	if err := basespec.ValidatePortableLocator(value, false); err != nil {
		return err
	}
	if path.Base(string(value)) != string(ManagedDocumentFileName) ||
		path.Dir(string(value)) == "." {
		return fmt.Errorf(
			"%w: MCP Bundle document locator must be nested and named %q",
			basespec.ErrInvalid,
			ManagedDocumentFileName,
		)
	}
	return nil
}

func IsBundleDocumentLocator(value basespec.Locator) bool {
	return path.Base(string(value)) == string(ManagedDocumentFileName)
}

func validateBundlePackageAddress(
	address source.ManagedPackageAddress,
) error {
	if err := address.Validate(); err != nil {
		return err
	}
	if address.Kind != ManagedPackageKind {
		return fmt.Errorf(
			"%w: MCP Bundle package kind must be %q",
			basespec.ErrInvalid,
			ManagedPackageKind,
		)
	}
	return nil
}
