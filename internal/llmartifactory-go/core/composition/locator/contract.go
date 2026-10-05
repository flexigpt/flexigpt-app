package locator

import (
	"context"
	"fmt"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

type Factory interface {
	LocatorKind() string
	ArtifactKinds() []artifactModel.ArtifactKind
	Revision() string
	Bind(runtime Runtime) (Resolver, error)
}

// Resolver consumes committed or verified state. It must not create Sources,
// expand discovery, refresh, or publish packages.
type Resolver interface {
	Resolve(ctx context.Context, request Request) (artifactModel.ArtifactRef, error)
}

// Runtime exposes committed catalog queries only.
type Runtime interface {
	ListBySource(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
		options catalogModel.ListOptions,
	) ([]catalogModel.Entry, error)
}

type Request struct {
	RootID rootModel.RootID
	From   *artifactModel.Artifact

	Locator declaration.Locator
	Entry   *declaration.Entry

	ExpectedKind        artifactModel.ArtifactKind
	ExpectedLogicalName spec.LogicalName
}

func (r Request) Validate() error {
	if err := r.RootID.Validate(); err != nil {
		return err
	}
	if r.From != nil {
		if err := r.From.ValidateRead(); err != nil {
			return err
		}
		if r.From.RootID != r.RootID {
			return fmt.Errorf("%w: locator origin belongs to another Root", spec.ErrInvalid)
		}
	}
	if err := r.Locator.Validate(); err != nil {
		return err
	}
	if err := r.ExpectedKind.Validate(); err != nil {
		return err
	}
	if r.ExpectedLogicalName != "" {
		if err := r.ExpectedLogicalName.Validate(); err != nil {
			return err
		}
	}
	if r.Entry != nil {
		if err := r.Entry.Validate(); err != nil {
			return err
		}
		header := r.Entry.Header()
		if artifactModel.ArtifactKind(header.Type) != r.ExpectedKind {
			return fmt.Errorf("%w: locator entry type differs from expected kind", spec.ErrInvalid)
		}
		if r.ExpectedLogicalName != "" && header.Name != string(r.ExpectedLogicalName) {
			return fmt.Errorf("%w: locator entry name differs from expected identity", spec.ErrInvalid)
		}
	}
	return nil
}

func (r Request) Clone() Request {
	output := r
	if r.From != nil {
		value := r.From.Clone()
		output.From = &value
	}
	if r.Entry != nil {
		value := r.Entry.Clone()
		output.Entry = &value
	}
	output.Locator = r.Locator.Clone()
	return output
}

type FactoryKey struct {
	LocatorKind  string
	ArtifactKind artifactModel.ArtifactKind
}

func (k FactoryKey) Validate() error {
	if err := spec.ValidateIdentifier("locator resolver kind", k.LocatorKind, spec.MaxKindBytes); err != nil {
		return err
	}
	return k.ArtifactKind.Validate()
}

func ValidateFactory(value Factory) error {
	if value == nil {
		return fmt.Errorf("%w: declaration locator factory is nil", spec.ErrInvalid)
	}
	if err := spec.ValidateIdentifier("locator resolver kind", value.LocatorKind(), spec.MaxKindBytes); err != nil {
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
		return fmt.Errorf("%w: locator resolver has no Artifact kinds", spec.ErrInvalid)
	}
	seen := make(map[artifactModel.ArtifactKind]struct{}, len(kinds))
	for _, kind := range kinds {
		if err := kind.Validate(); err != nil {
			return err
		}
		if _, duplicate := seen[kind]; duplicate {
			return fmt.Errorf("%w: locator resolver repeats Artifact kind %q", spec.ErrConflict, kind)
		}
		seen[kind] = struct{}{}
	}
	return nil
}
