package consumerapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/model/store/domain"
)

const (
	providerDefaultModelDataNamespace = "flexigpt.site/model-provider-default-model-v1"
	providerDefaultModelDataVersion   = "v1"
)

type providerDefaultModelData struct {
	SchemaVersion string                            `json:"schemaVersion"`
	DefaultModel  declaration.ArtifactNameReference `json:"defaultModel"`
}

func (a *API) SetMutableProviderDefaultModel(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
	defaultModel declaration.ArtifactNameReference,
) (ProviderView, error) {
	if err := a.ready(ctx); err != nil {
		return ProviderView{}, err
	}
	if err := validateExpectedArtifactRevision(
		expectedArtifactRevision,
	); err != nil {
		return ProviderView{}, err
	}
	if err := defaultModel.Validate(); err != nil {
		return ProviderView{}, err
	}

	provider, err := a.loadProvider(ctx, ref)
	if err != nil {
		return ProviderView{}, err
	}
	if provider.Artifact.Revision != expectedArtifactRevision {
		return ProviderView{}, basespec.ErrConflict
	}
	if a.protection.IsProtectedRoot(provider.Artifact.RootID) {
		return ProviderView{}, fmt.Errorf(
			"%w: protected Model Provider default belongs in a runtime overlay",
			basespec.ErrProtected,
		)
	}

	current, found, err := readMutableProviderDefaultModel(
		provider.Artifact.Data,
	)
	if err != nil {
		return ProviderView{}, err
	}
	if found && current == defaultModel {
		return a.providerView(provider)
	}

	fields, err := artifact.DecodeDataObject(provider.Artifact.Data)
	if err != nil {
		return ProviderView{}, err
	}
	payload, err := jsonutil.MarshalCanonicalObject(
		providerDefaultModelData{
			SchemaVersion: providerDefaultModelDataVersion,
			DefaultModel:  defaultModel.Clone(),
		},
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return ProviderView{}, err
	}
	fields[providerDefaultModelDataNamespace] = payload

	data, err := artifact.EncodeDataObject(fields)
	if err != nil {
		return ProviderView{}, err
	}
	updated, err := a.artifacts.UpdateData(
		ctx,
		ref,
		expectedArtifactRevision,
		data,
	)
	if err != nil {
		return ProviderView{}, err
	}

	updatedProvider, err := modelDomain.DecodeProvider(
		updated,
		provider.Definition,
	)
	if err != nil {
		return ProviderView{}, err
	}
	return a.providerView(updatedProvider)
}

func (a *API) ClearMutableProviderDefaultModel(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
) (ProviderView, error) {
	if err := a.ready(ctx); err != nil {
		return ProviderView{}, err
	}
	if err := validateExpectedArtifactRevision(
		expectedArtifactRevision,
	); err != nil {
		return ProviderView{}, err
	}

	provider, err := a.loadProvider(ctx, ref)
	if err != nil {
		return ProviderView{}, err
	}
	if provider.Artifact.Revision != expectedArtifactRevision {
		return ProviderView{}, basespec.ErrConflict
	}
	if a.protection.IsProtectedRoot(provider.Artifact.RootID) {
		return ProviderView{}, fmt.Errorf(
			"%w: protected Model Provider default belongs in a runtime overlay",
			basespec.ErrProtected,
		)
	}

	fields, err := artifact.DecodeDataObject(provider.Artifact.Data)
	if err != nil {
		return ProviderView{}, err
	}
	if _, found := fields[providerDefaultModelDataNamespace]; !found {
		return a.providerView(provider)
	}
	delete(fields, providerDefaultModelDataNamespace)

	data, err := artifact.EncodeDataObject(fields)
	if err != nil {
		return ProviderView{}, err
	}
	updated, err := a.artifacts.UpdateData(
		ctx,
		ref,
		expectedArtifactRevision,
		data,
	)
	if err != nil {
		return ProviderView{}, err
	}

	updatedProvider, err := modelDomain.DecodeProvider(
		updated,
		provider.Definition,
	)
	if err != nil {
		return ProviderView{}, err
	}
	return a.providerView(updatedProvider)
}

func readMutableProviderDefaultModel(
	raw json.RawMessage,
) (declaration.ArtifactNameReference, bool, error) {
	fields, err := artifact.DecodeDataObject(raw)
	if err != nil {
		return declaration.ArtifactNameReference{}, false, err
	}
	payload, found := fields[providerDefaultModelDataNamespace]
	if !found {
		return declaration.ArtifactNameReference{}, false, nil
	}

	var value providerDefaultModelData
	if err := jsonutil.DecodeCanonicalObjectExactInto(
		payload,
		&value,
		basespec.MaxLocalDataBytes,
	); err != nil {
		return declaration.ArtifactNameReference{}, false, fmt.Errorf(
			"%w: decode Model Provider default Model metadata: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if value.SchemaVersion != providerDefaultModelDataVersion {
		return declaration.ArtifactNameReference{}, false, fmt.Errorf(
			"%w: unsupported Model Provider default Model metadata schema %q",
			basespec.ErrInvalid,
			value.SchemaVersion,
		)
	}
	if err := value.DefaultModel.Validate(); err != nil {
		return declaration.ArtifactNameReference{}, false, err
	}
	return value.DefaultModel.Clone(), true, nil
}
