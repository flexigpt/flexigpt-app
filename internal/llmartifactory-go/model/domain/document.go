package domain

import (
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/contract/v1"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/modelprovider/contract/v1"
)

// Provider is immutable decoded source material for one model.provider
// Artifact. It deliberately contains no credentials, local runtime overlay,
// effective capability profile, or inference-go value.
type Provider struct {
	Artifact   artifactModel.Artifact
	Definition definitionModel.Definition
	Document   modelproviderv1.ProviderDocument
}

// Model is immutable decoded source material for one model Artifact. It
// deliberately contains no resolved provider, credentials, local runtime
// overlay, effective capability profile, or inference-go value.
type Model struct {
	Artifact   artifactModel.Artifact
	Definition definitionModel.Definition
	Document   modelv1.ModelDocument
}

func DecodeProvider(
	record artifactModel.Artifact,
	value definitionModel.Definition,
) (Provider, error) {
	if record.Kind != ModelProviderArtifactKind {
		return Provider{}, fmt.Errorf(
			"%w: Artifact %q is not a Model Provider",
			spec.ErrUnsupported,
			record.ID,
		)
	}
	if record.State != artifactModel.StateAvailable {
		return Provider{}, fmt.Errorf(
			"%w: Model Provider Artifact %q is unavailable",
			spec.ErrReferenceUnresolved,
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
			spec.ErrUnsupported,
			record.ID,
		)
	}
	if record.ResolvedDefinition == nil ||
		*record.ResolvedDefinition != value.Digest {
		return Provider{}, fmt.Errorf(
			"%w: Model Provider definition changed during read",
			spec.ErrRefreshRequired,
		)
	}

	document, err := modelproviderv1.DecodeModelProviderJSON(value.Body)
	if err != nil {
		return Provider{}, err
	}
	if record.LogicalName != spec.LogicalName(document.Name) {
		return Provider{}, fmt.Errorf(
			"%w: Model Provider identity differs from its declaration",
			spec.ErrDigestMismatch,
		)
	}

	return Provider{
		Artifact:   record.Clone(),
		Definition: value.Clone(),
		Document:   document,
	}, nil
}

func DecodeModel(
	record artifactModel.Artifact,
	value definitionModel.Definition,
) (Model, error) {
	if record.Kind != ModelArtifactKind {
		return Model{}, fmt.Errorf(
			"%w: Artifact %q is not a Model",
			spec.ErrUnsupported,
			record.ID,
		)
	}
	if record.State != artifactModel.StateAvailable {
		return Model{}, fmt.Errorf(
			"%w: Model Artifact %q is unavailable",
			spec.ErrReferenceUnresolved,
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
			spec.ErrUnsupported,
			record.ID,
		)
	}
	if record.ResolvedDefinition == nil ||
		*record.ResolvedDefinition != value.Digest {
		return Model{}, fmt.Errorf(
			"%w: Model definition changed during read",
			spec.ErrRefreshRequired,
		)
	}

	document, err := modelv1.DecodeModelJSON(value.Body)
	if err != nil {
		return Model{}, err
	}
	if record.LogicalName != spec.LogicalName(document.Name) {
		return Model{}, fmt.Errorf(
			"%w: Model identity differs from its declaration",
			spec.ErrDigestMismatch,
		)
	}

	return Model{
		Artifact:   record.Clone(),
		Definition: value.Clone(),
		Document:   document,
	}, nil
}
