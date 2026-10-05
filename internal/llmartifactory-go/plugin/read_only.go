package plugin

import (
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func (p Profile) validateAuthoringConfiguration() error {
	if p.ReadOnly {
		return nil
	}
	if err := p.SourceStorageKey.Validate(); err != nil {
		return err
	}
	if err := spec.ValidateRequiredText(
		"Plugin domain Source display name",
		p.SourceDisplayName,
		spec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := p.BaselineName.Validate(); err != nil {
		return err
	}
	if err := spec.ValidateOptionalText(
		"Plugin baseline display name",
		p.BaselineDisplayName,
		spec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	return spec.ValidateRequiredText(
		"Plugin baseline description",
		p.BaselineDescription,
		spec.MaxDescriptionBytes,
	)
}

func (a *API) requireDeclarationAuthoring() error {
	if a == nil {
		return spec.ErrClosed
	}
	if a.domain != nil && a.domain.ReadOnly {
		return fmt.Errorf(
			"%w: %s Plugin declarations are read-only",
			spec.ErrUnsupported,
			a.domain.Name,
		)
	}
	return nil
}

// A read-only domain has no managed user Source or baseline through which
// an empty Plugin can be classified. Its declared package origin is
// therefore part of domain visibility.
func (a *API) readOnlyDomainOrigin(record artifactModel.Artifact) bool {
	if a == nil || a.domain == nil || !a.domain.ReadOnly {
		return false
	}
	if record.Binding.SubresourceLocator != "" {
		return false
	}

	actual, err := a.managedPluginAddressFromLocator(
		record.Binding.Locator,
	)
	if err != nil {
		return false
	}
	expected, err := a.managedPluginAddress(record.LogicalName)
	if err != nil {
		return false
	}
	return actual == expected
}
