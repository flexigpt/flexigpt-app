package providerapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

// LocatorResolverFactory declares one owner-provided portable declaration
// locator resolver.
//
// Artifact Store validates and distributes these factories but does not invoke
// them itself. Artifact Resolver or owner APIs bind a factory to narrow Store
// lifecycle operations when they need to resolve a declaration locator.
//
// Current built-in support is limited to `path`. URL, Git, package, archive,
// and registry resolvers are intentionally future provider extensions.
type LocatorResolverFactory interface {
	LocatorKind() string
	ArtifactKinds() []artifact.ArtifactKind
	Revision() string

	BindLocatorRuntime(
		runtime LocatorRuntime,
	) (BoundLocatorResolver, error)
}

// BoundLocatorResolver is a per-owner resolver instance. Binding keeps Store
// lifecycle operations explicit and prevents providers from retaining global
// Store service references.
type BoundLocatorResolver interface {
	ResolveLocator(
		ctx context.Context,
		request LocatorResolutionRequest,
	) (artifact.ArtifactRef, error)
}

// LocatorRuntime is the narrow generic Store lifecycle capability available
// to a bound declaration locator resolver.
//
// It deliberately contains no concrete declaration contracts, no consumer
// runtime behavior, and no Source adapter configuration access.
type LocatorRuntime interface {
	GetSource(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
	) (source.Summary, error)

	UpdateSource(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
		update source.Update,
	) (source.Summary, error)

	RefreshSource(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
	) (refresh.RefreshSourceResult, error)

	ListArtifactsBySource(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
	) ([]artifact.Artifact, error)
}

// LocatorResolutionRequest is generic provider input. LocatorJSON and
// EntryJSON preserve the portable declaration data without making
// artifactstore/providerapi import artifact declaration contracts.
type LocatorResolutionRequest struct {
	RootID root.RootID
	From   *artifact.Artifact

	LocatorJSON json.RawMessage
	EntryJSON   json.RawMessage

	ExpectedKind        artifact.ArtifactKind
	ExpectedLogicalName basespec.LogicalName
}

func (r LocatorResolutionRequest) Validate() error {
	if err := r.RootID.Validate(); err != nil {
		return err
	}
	if r.From != nil {
		if err := r.From.Validate(); err != nil {
			return err
		}
		if r.From.RootID != r.RootID {
			return fmt.Errorf(
				"%w: locator origin Artifact belongs to another Root",
				basespec.ErrInvalid,
			)
		}
	}
	if len(r.LocatorJSON) == 0 ||
		len(r.LocatorJSON) > basespec.MaxDefinitionBodyBytes ||
		!json.Valid(r.LocatorJSON) {
		return fmt.Errorf(
			"%w: locator resolver request has invalid locator JSON",
			basespec.ErrInvalid,
		)
	}
	if len(r.EntryJSON) != 0 &&
		(len(r.EntryJSON) > basespec.MaxDefinitionBodyBytes ||
			!json.Valid(r.EntryJSON)) {
		return fmt.Errorf(
			"%w: locator resolver request has invalid declaration JSON",
			basespec.ErrInvalid,
		)
	}
	if err := r.ExpectedKind.Validate(); err != nil {
		return err
	}
	if r.ExpectedLogicalName != "" {
		if err := r.ExpectedLogicalName.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// LocatorResolverKey identifies one provider-owned declaration locator
// resolver slot.
type LocatorResolverKey struct {
	LocatorKind  string
	ArtifactKind artifact.ArtifactKind
}

func (k LocatorResolverKey) Validate() error {
	if err := basespec.ValidateIdentifier(
		"locator resolver locator kind",
		k.LocatorKind,
		basespec.MaxKindBytes,
	); err != nil {
		return err
	}
	return k.ArtifactKind.Validate()
}

func ValidateLocatorResolverFactory(
	value LocatorResolverFactory,
) error {
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
	if err := basespec.ValidateRequiredText(
		"locator resolver revision",
		value.Revision(),
		basespec.MaxVersionBytes,
	); err != nil {
		return err
	}

	kinds := value.ArtifactKinds()
	if len(kinds) == 0 {
		return fmt.Errorf(
			"%w: locator resolver %q has no Artifact kinds",
			basespec.ErrInvalid,
			value.LocatorKind(),
		)
	}

	seen := make(map[artifact.ArtifactKind]struct{}, len(kinds))
	for index, kind := range kinds {
		if err := kind.Validate(); err != nil {
			return fmt.Errorf(
				"locator resolver %q Artifact kind %d: %w",
				value.LocatorKind(),
				index,
				err,
			)
		}
		if _, duplicate := seen[kind]; duplicate {
			return fmt.Errorf(
				"%w: locator resolver %q repeats Artifact kind %q",
				basespec.ErrConflict,
				value.LocatorKind(),
				kind,
			)
		}
		seen[kind] = struct{}{}
	}
	return nil
}
