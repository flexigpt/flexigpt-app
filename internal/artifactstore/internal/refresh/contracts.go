package refreshimpl

import (
	"context"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	artifactimpl "github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/artifact"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type RefreshStateReader interface {
	GetRefreshState(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
	) (source.RefreshState, error)
}

type ArtifactReader interface {
	ListBySource(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
	) ([]artifact.Artifact, error)
}

type Publication struct {
	RootID root.RootID

	SourceID                source.SourceID
	ExpectedSourceRevision  uint64
	ExpectedRefreshRevision uint64

	SourceGeneration     string
	DiscoveryFingerprint cryptoutil.Digest
	DecoderFingerprint   cryptoutil.Digest

	Definitions     []definition.Definition
	ArtifactCreates []artifact.Artifact
	ArtifactUpdates []artifactimpl.SourceStateUpdate
	Diagnostics     []diagnostic.Diagnostic
	RefreshedAt     time.Time
}

func (p Publication) Validate() error {
	if err := p.RootID.Validate(); err != nil {
		return err
	}
	if err := p.SourceID.Validate(); err != nil {
		return err
	}
	if p.ExpectedSourceRevision == 0 {
		return fmt.Errorf(
			"%w: expected Source revision is required",
			basespec.ErrInvalid,
		)
	}
	if err := basespec.ValidateSourceGeneration(
		p.SourceGeneration,
	); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(
		p.DiscoveryFingerprint,
	); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(
		p.DecoderFingerprint,
	); err != nil {
		return err
	}
	if p.RefreshedAt.IsZero() {
		return fmt.Errorf(
			"%w: Source refresh publication time is required",
			basespec.ErrInvalid,
		)
	}
	if err := diagnostic.Validate(p.Diagnostics); err != nil {
		return err
	}

	seenDefinitions := make(
		map[cryptoutil.Digest]struct{},
		len(p.Definitions),
	)
	for index, value := range p.Definitions {
		canonical, err := definition.Canonicalize(value)
		if err != nil {
			return fmt.Errorf("definition %d: %w", index, err)
		}
		if canonical.Digest != value.Digest {
			return fmt.Errorf(
				"%w: publication Definition %d is not canonical",
				basespec.ErrInvalid,
				index,
			)
		}
		if _, duplicate := seenDefinitions[value.Digest]; duplicate {
			return fmt.Errorf(
				"%w: publication repeats Definition %q",
				basespec.ErrInvalid,
				value.Digest,
			)
		}
		seenDefinitions[value.Digest] = struct{}{}
	}

	seenArtifacts := make(map[artifact.ArtifactID]struct{})
	for index, value := range p.ArtifactCreates {
		if err := value.Validate(); err != nil {
			return fmt.Errorf("artifact create %d: %w", index, err)
		}
		if value.RootID != p.RootID ||
			value.Binding.SourceID != p.SourceID ||
			value.Revision != 1 ||
			value.State != artifact.StateAvailable {
			return fmt.Errorf(
				"%w: invalid source-created Artifact",
				basespec.ErrInvalid,
			)
		}
		if _, duplicate := seenArtifacts[value.ID]; duplicate {
			return fmt.Errorf(
				"%w: publication repeats Artifact %q",
				basespec.ErrInvalid,
				value.ID,
			)
		}
		seenArtifacts[value.ID] = struct{}{}
	}
	for index, value := range p.ArtifactUpdates {
		if err := value.Validate(); err != nil {
			return fmt.Errorf("artifact update %d: %w", index, err)
		}
		if value.RootID != p.RootID ||
			value.Binding.SourceID != p.SourceID {
			return fmt.Errorf(
				"%w: source-derived Artifact update belongs to another Source",
				basespec.ErrInvalid,
			)
		}
		if _, duplicate := seenArtifacts[value.ArtifactID]; duplicate {
			return fmt.Errorf(
				"%w: publication repeats Artifact %q",
				basespec.ErrInvalid,
				value.ArtifactID,
			)
		}
		seenArtifacts[value.ArtifactID] = struct{}{}
	}
	return nil
}

type Publisher interface {
	Publish(
		ctx context.Context,
		publication Publication,
	) (source.RefreshState, error)
}
