package locator

import (
	"context"
	"encoding/json"
	"fmt"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// Factory creates a declaration-locator resolver bound to one narrow Artifact
// catalog runtime.
//
// This contract belongs to artifactcontract because locator syntax, declaration
// entry selection, subresource conventions, and artifact-family-specific
// resolution behavior do not belong to generic Artifact Store.
type Factory interface {
	LocatorKind() string
	ArtifactKinds() []artifactModel.ArtifactKind
	Revision() string

	Bind(
		runtime Runtime,
	) (Resolver, error)
}

// Resolver resolves one declaration locator to one source-backed Artifact.
type Resolver interface {
	Resolve(
		ctx context.Context,
		request Request,
	) (artifactModel.ArtifactRef, error)
}

// Runtime is deliberately narrow, artifact/catalog.API satisfies this interface directly.
type Runtime interface {
	ListBySource(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
		options catalogModel.ListOptions,
	) ([]catalogModel.Entry, error)
}

// Request preserves declaration-owned JSON without requiring generic
// Artifact Store to know declaration vocabulary.
type Request struct {
	RootID rootModel.RootID
	From   *artifactModel.Artifact

	LocatorJSON json.RawMessage
	EntryJSON   json.RawMessage

	ExpectedKind        artifactModel.ArtifactKind
	ExpectedLogicalName spec.LogicalName
}

func (r Request) Validate() error {
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
				spec.ErrInvalid,
			)
		}
	}

	if len(r.LocatorJSON) == 0 ||
		len(r.LocatorJSON) > spec.MaxDefinitionBodyBytes ||
		!json.Valid(r.LocatorJSON) {
		return fmt.Errorf(
			"%w: locator request has invalid locator JSON",
			spec.ErrInvalid,
		)
	}

	if len(r.EntryJSON) != 0 &&
		(len(r.EntryJSON) > spec.MaxDefinitionBodyBytes ||
			!json.Valid(r.EntryJSON)) {
		return fmt.Errorf(
			"%w: locator request has invalid declaration JSON",
			spec.ErrInvalid,
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

// FactoryKey identifies one locator-kind/artifact-kind registration slot.
type FactoryKey struct {
	LocatorKind  string
	ArtifactKind artifactModel.ArtifactKind
}

func (k FactoryKey) Validate() error {
	if err := spec.ValidateIdentifier(
		"locator resolver kind",
		k.LocatorKind,
		spec.MaxKindBytes,
	); err != nil {
		return err
	}
	return k.ArtifactKind.Validate()
}

func ValidateFactory(
	value Factory,
) error {
	if value == nil {
		return fmt.Errorf(
			"%w: declaration locator factory is nil",
			spec.ErrInvalid,
		)
	}

	if err := spec.ValidateIdentifier(
		"locator resolver kind",
		value.LocatorKind(),
		spec.MaxKindBytes,
	); err != nil {
		return err
	}

	if err := spec.ValidateRequiredText(
		"locator resolver revision",
		value.Revision(),
		spec.MaxVersionBytes,
	); err != nil {
		return err
	}

	kinds := value.ArtifactKinds()
	if len(kinds) == 0 {
		return fmt.Errorf(
			"%w: locator resolver %q has no Artifact kinds",
			spec.ErrInvalid,
			value.LocatorKind(),
		)
	}

	seen := make(map[artifactModel.ArtifactKind]struct{}, len(kinds))
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
				spec.ErrConflict,
				value.LocatorKind(),
				kind,
			)
		}
		seen[kind] = struct{}{}
	}

	return nil
}
