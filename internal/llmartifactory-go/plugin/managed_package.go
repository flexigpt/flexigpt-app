package plugin

import (
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func (a *API) managedPluginAddress(
	name spec.LogicalName,
) (managedpackageModel.ManagedPackageAddress, error) {
	return a.domain.Package.Address(name, "")
}

func (a *API) managedPluginAddressFromLocator(
	locator spec.Locator,
) (managedpackageModel.ManagedPackageAddress, error) {
	return a.domain.Package.AddressFromLocator(locator)
}

func (a *API) managedPluginDocumentFile() spec.Locator {
	return a.domain.Package.Document.Locator
}

func (a *API) managedPluginDecoderID() (spec.DecoderID, error) {
	return a.domain.Package.Document.DecoderID, nil
}
