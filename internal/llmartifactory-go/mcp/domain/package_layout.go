package domain

import (
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
)

func ManagedPackageAddressForMCP(
	layout support.PackageLayout,
	name storeSpec.LogicalName,
	version storeSpec.LogicalVersion,
) (managedpackageModel.ManagedPackageAddress, error) {
	return layout.Address(name, version)
}

func ManagedPackageLocatorForMCP(
	layout support.PackageLayout,
	address managedpackageModel.ManagedPackageAddress,
) (storeSpec.Locator, error) {
	return layout.Locator(address)
}

func ManagedPackageAddressFromMCPLocator(
	layout support.PackageLayout,
	locator storeSpec.Locator,
) (managedpackageModel.ManagedPackageAddress, error) {
	return layout.AddressFromLocator(locator)
}

func ManagedPackageAddressFromMCPPolicyLocator(
	layout support.PackageLayout,
	locator storeSpec.Locator,
) (managedpackageModel.ManagedPackageAddress, error) {
	return layout.AddressFromLocator(locator)
}
