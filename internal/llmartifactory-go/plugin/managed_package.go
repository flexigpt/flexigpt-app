package plugin

import (
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func (a *API) managedPluginAddress(
	name storeSpec.LogicalName,
) (managedpackageModel.ManagedPackageAddress, error) {
	return a.domain.Package.Address(name, "")
}

func (a *API) managedPluginAddressFromLocator(
	locator storeSpec.Locator,
) (managedpackageModel.ManagedPackageAddress, error) {
	return a.domain.Package.AddressFromLocator(locator)
}

func (a *API) managedPluginDocumentFile() storeSpec.Locator {
	return a.domain.Package.Document.Locator
}

func (a *API) managedPluginDecoderID() (storeSpec.DecoderID, error) {
	return a.domain.Package.Document.DecoderID, nil
}
