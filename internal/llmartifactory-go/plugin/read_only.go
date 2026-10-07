package plugin

import (
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func (p Profile) validateAuthoringConfiguration() error {
	if p.ReadOnly {
		if p.Source != nil {
			return fmt.Errorf("%w: read-only Plugin profile cannot define a managed Source", spec.ErrInvalid)
		}
		return nil
	}
	if p.Source == nil {
		return fmt.Errorf("%w: managed Plugin profile requires a Source profile", spec.ErrInvalid)
	}
	if err := p.Source.Validate(); err != nil {
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
	if a.domain != nil && a.domain.ReadOnly {
		return fmt.Errorf(
			"%w: %s Plugin declarations are read-only",
			spec.ErrUnsupported,
			a.domain.Name,
		)
	}
	return nil
}

// A read-only family has no mutable managed Source. Its package origin is
// therefore part of family visibility, but the package layout remains
// application-supplied through Profile.Package.
func (a *API) readOnlyDomainOrigin(
	record artifactModel.Artifact,
) bool {
	if !a.domain.ReadOnly ||
		record.Binding.SubresourceLocator != "" {
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
