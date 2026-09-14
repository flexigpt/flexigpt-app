package providerapi

import "github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"

// LocatorResolver is registration metadata for a contract-owned portable
// locator resolver.
//
// Artifact Store records and verifies source-backed declarations. It does not
// resolve portable path, URL, Git, package, or command locators itself.
// Resolver and Workspace-loader integrations can inspect provider descriptors
// and select their own registered locator implementation.
type LocatorResolver interface {
	LocatorKind() string
	Revision() string
}

func ValidateLocatorResolver(value LocatorResolver) error {
	if value == nil {
		return basespec.ErrInvalid
	}
	if err := basespec.ValidateIdentifier(
		"locator resolver kind",
		value.LocatorKind(),
		basespec.MaxKindBytes,
	); err != nil {
		return err
	}
	return basespec.ValidateRequiredText(
		"locator resolver revision",
		value.Revision(),
		basespec.MaxVersionBytes,
	)
}
