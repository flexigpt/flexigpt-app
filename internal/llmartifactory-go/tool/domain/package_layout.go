package domain

import (
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
)

func ToolPackageAddress(
	layout support.PackageLayout,
	name spec.LogicalName,
	version spec.LogicalVersion,
) (managedpackageModel.ManagedPackageAddress, error) {
	return layout.Address(name, version)
}

func ToolPluginPackageAddress(
	layout support.PackageLayout,
	name spec.LogicalName,
) (managedpackageModel.ManagedPackageAddress, error) {
	return layout.Address(name, "")
}

func ToolPackageAddressFromLocator(
	layout support.PackageLayout,
	locator spec.Locator,
) (managedpackageModel.ManagedPackageAddress, error) {
	return layout.AddressFromLocator(locator)
}

func ToolPluginPackageAddressFromLocator(
	layout support.PackageLayout,
	locator spec.Locator,
) (managedpackageModel.ManagedPackageAddress, error) {
	return layout.AddressFromLocator(locator)
}
