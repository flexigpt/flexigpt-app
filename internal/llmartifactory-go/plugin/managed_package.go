package plugin

import (
	"fmt"
	"path"

	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
)

func (a *API) managedPluginAddress(
	name spec.LogicalName,
) (managedpackageModel.ManagedPackageAddress, error) {
	return managedPluginAddressFor(
		a.managedPluginPackageKind(),
		name,
	)
}

func managedPluginAddressFor(
	packageKind managedpackageModel.PackageKind,
	name spec.LogicalName,
) (managedpackageModel.ManagedPackageAddress, error) {
	return managedpackageModel.NewManagedPackageAddress(
		packageKind,
		name,
		topology.UnversionedPackageVersion(),
	)
}

func (a *API) managedPluginAddressFromLocator(
	locator spec.Locator,
) (managedpackageModel.ManagedPackageAddress, error) {
	return managedPluginAddressFromLocatorFor(
		a.managedPluginPackageKind(),
		a.managedPluginDocumentFile(),
		locator,
	)
}

func (a *API) managedPluginPackageKind() managedpackageModel.PackageKind {
	if a != nil && a.domain != nil && a.domain.PackageKind != "" {
		return a.domain.PackageKind
	}
	return ManagedPluginPackageKind
}

func (a *API) managedPluginDocumentUse() string {
	if a != nil && a.domain != nil && a.domain.DocumentUse != "" {
		return a.domain.DocumentUse
	}
	return topology.DocumentUseManagedPlugin
}

func (a *API) managedPluginDocumentFile() spec.Locator {
	return topology.MustDefaultDocumentFile(a.managedPluginDocumentUse())
}

func (a *API) managedPluginDecoderID() (spec.DecoderID, error) {
	return topology.DefaultDocumentDecoderID(a.managedPluginDocumentUse())
}

func managedPluginAddressFromLocator(
	locator spec.Locator,
) (managedpackageModel.ManagedPackageAddress, error) {
	return managedPluginAddressFromLocatorFor(
		ManagedPluginPackageKind,
		topology.MustDefaultDocumentFile(topology.DocumentUseManagedPlugin),
		locator,
	)
}

func managedPluginAddressFromLocatorFor(
	packageKind managedpackageModel.PackageKind,
	documentFile spec.Locator,
	locator spec.Locator,
) (managedpackageModel.ManagedPackageAddress, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return managedpackageModel.ManagedPackageAddress{}, err
	}
	if err := packageKind.Validate(); err != nil {
		return managedpackageModel.ManagedPackageAddress{}, err
	}
	if err := documentFile.ValidatePortable(false); err != nil {
		return managedpackageModel.ManagedPackageAddress{}, err
	}
	if path.Base(string(locator)) != string(documentFile) {
		return managedpackageModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: Plugin locator %q is not %q",
			spec.ErrUnsupported,
			locator,
			documentFile,
		)
	}

	address, err := managedpackageModel.ParseManagedPackageAddressDirectory(
		spec.Locator(path.Dir(string(locator))),
	)
	if err != nil {
		return managedpackageModel.ManagedPackageAddress{}, err
	}
	if address.Kind != packageKind {
		return managedpackageModel.ManagedPackageAddress{}, fmt.Errorf(
			"%w: managed Plugin package kind must be %q",
			spec.ErrUnsupported,
			packageKind,
		)
	}
	return address, nil
}
