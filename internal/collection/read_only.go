package collection

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
)

func (p DomainPolicy) validateAuthoringConfiguration() error {
	if p.ReadOnly {
		return nil
	}
	if err := p.SourceStorageKey.Validate(); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"Collection domain Source display name",
		p.SourceDisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := p.BaselineName.Validate(); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"Collection baseline display name",
		p.BaselineDisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	return basespec.ValidateRequiredText(
		"Collection baseline description",
		p.BaselineDescription,
		basespec.MaxDescriptionBytes,
	)
}

func (a *API) requireDeclarationAuthoring() error {
	if a == nil {
		return basespec.ErrClosed
	}
	if a.domain != nil && a.domain.ReadOnly {
		return fmt.Errorf(
			"%w: %s Collection declarations are read-only",
			basespec.ErrUnsupported,
			a.domain.Name,
		)
	}
	return nil
}

// A read-only domain has no managed user Source or baseline through which
// an empty Collection can be classified. Its declared package origin is
// therefore part of domain visibility.
func (a *API) readOnlyDomainOrigin(record artifact.Artifact) bool {
	if a == nil || a.domain == nil || !a.domain.ReadOnly {
		return false
	}
	if record.Binding.SubresourceLocator != "" {
		return false
	}

	actual, err := a.managedCollectionAddressFromLocator(
		record.Binding.Locator,
	)
	if err != nil {
		return false
	}
	expected, err := a.managedCollectionAddress(record.LogicalName)
	if err != nil {
		return false
	}
	return actual == expected
}
