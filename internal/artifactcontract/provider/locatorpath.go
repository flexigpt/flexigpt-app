package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

// Package locatorpath implements portable source-relative path declaration
// loading. It intentionally does not resolve URL, Git, package, archive, or
// other non-path locators.

const locatorKindPath = "path"

type locatorpathFactory struct {
	artifactKinds []artifact.ArtifactKind
}

func newLocatorpathFactory() *locatorpathFactory {
	declarationTypes := declaration.Types()
	artifactKinds := make(
		[]artifact.ArtifactKind,
		0,
		len(declarationTypes),
	)
	for _, declarationType := range declarationTypes {
		artifactKinds = append(
			artifactKinds,
			artifact.ArtifactKind(declarationType),
		)
	}
	return &locatorpathFactory{
		artifactKinds: artifactKinds,
	}
}

func (*locatorpathFactory) LocatorKind() string {
	return locatorKindPath
}

func (f *locatorpathFactory) ArtifactKinds() []artifact.ArtifactKind {
	if f == nil {
		return nil
	}
	return append([]artifact.ArtifactKind(nil), f.artifactKinds...)
}

func (*locatorpathFactory) Revision() string {
	return "artifact-declaration-path/v1"
}

func (f *locatorpathFactory) BindLocatorRuntime(
	runtime providerapi.LocatorRuntime,
) (providerapi.BoundLocatorResolver, error) {
	if f == nil || runtime == nil {
		return nil, fmt.Errorf(
			"%w: path locator resolver dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return &boundResolver{
		artifactKinds: f.ArtifactKinds(),

		runtime: runtime,
	}, nil
}

type boundResolver struct {
	artifactKinds []artifact.ArtifactKind

	runtime providerapi.LocatorRuntime
}

func (r *boundResolver) ResolveLocator(
	ctx context.Context,
	request providerapi.LocatorResolutionRequest,
) (artifact.ArtifactRef, error) {
	if r == nil || r.runtime == nil {
		return artifact.ArtifactRef{}, basespec.ErrClosed
	}
	if ctx == nil {
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: path locator resolution context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return artifact.ArtifactRef{}, err
	}
	if err := request.Validate(); err != nil {
		return artifact.ArtifactRef{}, err
	}
	if request.From == nil {
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: path declaration locator requires a source-backed origin Artifact",
			basespec.ErrLocatorUnresolved,
		)
	}
	if !slices.Contains(r.artifactKinds, request.ExpectedKind) {
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: path locator resolver does not support Artifact kind %q",
			basespec.ErrUnsupported,
			request.ExpectedKind,
		)
	}

	var locator declaration.Locator
	if err := json.Unmarshal(request.LocatorJSON, &locator); err != nil {
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: decode path locator: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	target, err := declaration.ResolveSourceRelativePathLocator(
		locator,
		request.From.Binding.Locator,
	)
	if err != nil {
		return artifact.ArtifactRef{}, err
	}

	candidates, err := declarationCandidateLocators(request.ExpectedKind, target)
	if err != nil {
		return artifact.ArtifactRef{}, err
	}
	return r.selectArtifact(ctx, request, candidates)
}

func declarationCandidateLocators(
	expectedKind artifact.ArtifactKind,
	target basespec.Locator,
) ([]basespec.Locator, error) {
	output := make([]basespec.Locator, 0, 2)
	output = append(output, target)
	if expectedKind != artifact.ArtifactKind(declaration.TypeSkill) ||
		path.Base(string(target)) == "SKILL.md" {
		return output, nil
	}

	// A portable Skill locator conventionally identifies the package
	// directory. The independently declared Artifact originates at SKILL.md.
	document := basespec.Locator(path.Join(
		string(target),
		"SKILL.md",
	))
	if err := document.Validate(false); err != nil {
		return nil, err
	}
	return append(output, document), nil
}

func (r *boundResolver) selectArtifact(
	ctx context.Context,
	request providerapi.LocatorResolutionRequest,
	locators []basespec.Locator,
) (artifact.ArtifactRef, error) {
	records, err := r.runtime.ListArtifactsBySource(
		ctx,
		request.RootID,
		request.From.Binding.SourceID,
	)
	if err != nil {
		return artifact.ArtifactRef{}, err
	}

	serverSubresource, hasServerSelector, err := requestedMCPServerSubresource(
		request,
	)
	if err != nil {
		return artifact.ArtifactRef{}, err
	}

	expectedTextVersion, hasTextVersion, err := requestedTextLogicalVersion(request)
	if err != nil {
		return artifact.ArtifactRef{}, err
	}

	selectedLocators := make(map[basespec.Locator]struct{}, len(locators))
	for _, locator := range locators {
		selectedLocators[locator] = struct{}{}
	}

	candidates := make([]artifact.Artifact, 0)
	for _, record := range records {
		if record.Kind != request.ExpectedKind ||
			record.State != artifact.StateAvailable {
			continue
		}
		if _, found := selectedLocators[record.Binding.Locator]; !found {
			continue
		}
		if request.ExpectedLogicalName != "" &&
			record.LogicalName != request.ExpectedLogicalName {
			continue
		}
		if hasTextVersion && record.LogicalVersion != expectedTextVersion {
			continue
		}
		if hasServerSelector &&
			record.Binding.SubresourceLocator != serverSubresource {
			continue
		}
		candidates = append(candidates, record)
	}

	switch len(candidates) {
	case 0:
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: path locator did not produce %s/%s",
			basespec.ErrReferenceUnresolved,
			request.ExpectedKind,
			request.ExpectedLogicalName,
		)
	case 1:
		return candidates[0].Ref(), nil
	default:
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: path locator produced %d %s Artifacts",
			basespec.ErrIdentityConflict,
			len(candidates),
			request.ExpectedKind,
		)
	}
}

func requestedTextLogicalVersion(
	request providerapi.LocatorResolutionRequest,
) (basespec.LogicalVersion, bool, error) {
	if request.ExpectedKind != artifact.ArtifactKind(declaration.TypeText) ||
		len(request.EntryJSON) == 0 {
		return "", false, nil
	}

	entry, err := declaration.DecodeEntryJSON(request.EntryJSON)
	if err != nil {
		return "", false, err
	}
	insert, err := entry.TextInsert()
	if err != nil {
		return "", false, err
	}
	return basespec.LogicalVersion(insert), true, nil
}

func requestedMCPServerSubresource(
	request providerapi.LocatorResolutionRequest,
) (basespec.SubresourceLocator, bool, error) {
	if request.ExpectedKind != artifact.ArtifactKind(declaration.TypeMCP) ||
		len(request.EntryJSON) == 0 {
		return "", false, nil
	}

	entry, err := declaration.DecodeEntryJSON(request.EntryJSON)
	if err != nil {
		return "", false, err
	}
	raw, err := entry.CanonicalJSON()
	if err != nil {
		return "", false, err
	}
	var selector struct {
		Server string `json:"server"`
	}
	if err := json.Unmarshal(raw, &selector); err != nil {
		return "", false, err
	}
	if selector.Server == "" {
		return "", false, nil
	}
	if err := basespec.LogicalName(selector.Server).Validate(); err != nil {
		return "", false, err
	}

	value := basespec.SubresourceLocator(
		"mcpServers/" + selector.Server,
	)
	if err := value.Validate(); err != nil {
		return "", false, err
	}
	return value, true, nil
}
