package domain

import (
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
)

func ManagedPackageAddressForMCP(
	layout support.PackageLayout,
	name spec.LogicalName,
	version spec.LogicalVersion,
) (managedpackageModel.ManagedPackageAddress, error) {
	return layout.Address(name, version)
}

func ManagedPackageLocatorForMCP(
	layout support.PackageLayout,
	address managedpackageModel.ManagedPackageAddress,
) (spec.Locator, error) {
	return layout.Locator(address)
}

func ManagedPackageAddressFromMCPLocator(
	layout support.PackageLayout,
	locator spec.Locator,
) (managedpackageModel.ManagedPackageAddress, error) {
	return layout.AddressFromLocator(locator)
}

func ManagedPackageAddressFromMCPPolicyLocator(
	layout support.PackageLayout,
	locator spec.Locator,
) (managedpackageModel.ManagedPackageAddress, error) {
	return layout.AddressFromLocator(locator)
}
