package domain

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/modelproviderv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/modelv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/definition"
)

// Provider is immutable decoded source material for one model.provider
// Artifact. It deliberately contains no credentials, local runtime overlay,
// effective capability profile, or inference-go value.
type Provider struct {
	Artifact   artifact.Artifact
	Definition definition.Definition
	Document   modelproviderv1.ProviderDocument
}

// Model is immutable decoded source material for one model Artifact. It
// deliberately contains no resolved provider, credentials, local runtime
// overlay, effective capability profile, or inference-go value.
type Model struct {
	Artifact   artifact.Artifact
	Definition definition.Definition
	Document   modelv1.ModelDocument
}

func DecodeProvider(
	record artifact.Artifact,
	value definition.Definition,
) (Provider, error) {
	if record.Kind != ModelProviderArtifactKind {
		return Provider{}, fmt.Errorf(
			"%w: Artifact %q is not a Model Provider",
			model.ErrUnsupported,
			record.ID,
		)
	}
	if record.State != artifact.StateAvailable {
		return Provider{}, fmt.Errorf(
			"%w: Model Provider Artifact %q is unavailable",
			model.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if err := value.Validate(); err != nil {
		return Provider{}, err
	}
	if value.Kind != ModelProviderArtifactKind ||
		value.SchemaID != modelproviderv1.ModelProviderSchemaKey.SchemaID ||
		value.SchemaVersion != modelproviderv1.ModelProviderSchemaKey.SchemaVersion {
		return Provider{}, fmt.Errorf(
			"%w: Model Provider Artifact %q has an unsupported schema",
			model.ErrUnsupported,
			record.ID,
		)
	}
	if record.ResolvedDefinition == nil ||
		*record.ResolvedDefinition != value.Digest {
		return Provider{}, fmt.Errorf(
			"%w: Model Provider definition changed during read",
			model.ErrRefreshRequired,
		)
	}

	document, err := modelproviderv1.DecodeModelProviderJSON(value.Body)
	if err != nil {
		return Provider{}, err
	}
	if record.LogicalName != model.LogicalName(document.Name) {
		return Provider{}, fmt.Errorf(
			"%w: Model Provider identity differs from its declaration",
			model.ErrDigestMismatch,
		)
	}

	return Provider{
		Artifact:   record.Clone(),
		Definition: value.Clone(),
		Document:   document,
	}, nil
}

func DecodeModel(
	record artifact.Artifact,
	value definition.Definition,
) (Model, error) {
	if record.Kind != ModelArtifactKind {
		return Model{}, fmt.Errorf(
			"%w: Artifact %q is not a Model",
			model.ErrUnsupported,
			record.ID,
		)
	}
	if record.State != artifact.StateAvailable {
		return Model{}, fmt.Errorf(
			"%w: Model Artifact %q is unavailable",
			model.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if err := value.Validate(); err != nil {
		return Model{}, err
	}
	if value.Kind != ModelArtifactKind ||
		value.SchemaID != modelv1.ModelSchemaKey.SchemaID ||
		value.SchemaVersion != modelv1.ModelSchemaKey.SchemaVersion {
		return Model{}, fmt.Errorf(
			"%w: Model Artifact %q has an unsupported schema",
			model.ErrUnsupported,
			record.ID,
		)
	}
	if record.ResolvedDefinition == nil ||
		*record.ResolvedDefinition != value.Digest {
		return Model{}, fmt.Errorf(
			"%w: Model definition changed during read",
			model.ErrRefreshRequired,
		)
	}

	document, err := modelv1.DecodeModelJSON(value.Body)
	if err != nil {
		return Model{}, err
	}
	if record.LogicalName != model.LogicalName(document.Name) {
		return Model{}, fmt.Errorf(
			"%w: Model identity differs from its declaration",
			model.ErrDigestMismatch,
		)
	}

	return Model{
		Artifact:   record.Clone(),
		Definition: value.Clone(),
		Document:   document,
	}, nil
}
