package refresh

import (
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type Publication struct {
	RootID rootModel.RootID

	SourceID                sourceModel.SourceID
	ExpectedSourceRevision  uint64
	ExpectedRefreshRevision uint64

	SourceGeneration     string
	DiscoveryFingerprint cryptoutil.Digest
	DecoderFingerprint   cryptoutil.Digest

	Definitions     []definitionModel.Definition
	ArtifactCreates []artifactModel.Artifact
	ArtifactUpdates []artifact.SourceStateUpdate
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
			spec.ErrInvalid,
		)
	}
	if err := spec.ValidateSourceGeneration(p.SourceGeneration); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(p.DiscoveryFingerprint); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(p.DecoderFingerprint); err != nil {
		return err
	}
	if p.RefreshedAt.IsZero() {
		return fmt.Errorf(
			"%w: Source refresh publication time is required",
			spec.ErrInvalid,
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
		if err := cryptoutil.ValidateDigest(value.Digest); err != nil {
			return fmt.Errorf("definition %d: %w", index, err)
		}
		if _, duplicate := seenDefinitions[value.Digest]; duplicate {
			return fmt.Errorf(
				"%w: publication repeats Definition %q",
				spec.ErrInvalid,
				value.Digest,
			)
		}
		seenDefinitions[value.Digest] = struct{}{}
	}

	seenArtifacts := make(map[artifactModel.ArtifactID]struct{})
	for index, value := range p.ArtifactCreates {
		if err := value.Validate(); err != nil {
			return fmt.Errorf("artifact create %d: %w", index, err)
		}
		if value.RootID != p.RootID ||
			value.Binding.SourceID != p.SourceID ||
			value.Revision != 1 ||
			value.State != artifactModel.StateAvailable {
			return fmt.Errorf(
				"%w: invalid source-created Artifact",
				spec.ErrInvalid,
			)
		}
		if _, duplicate := seenArtifacts[value.ID]; duplicate {
			return fmt.Errorf(
				"%w: publication repeats Artifact %q",
				spec.ErrInvalid,
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
				spec.ErrInvalid,
			)
		}
		if _, duplicate := seenArtifacts[value.ArtifactID]; duplicate {
			return fmt.Errorf(
				"%w: publication repeats Artifact %q",
				spec.ErrInvalid,
				value.ArtifactID,
			)
		}
		seenArtifacts[value.ArtifactID] = struct{}{}
	}

	return nil
}
