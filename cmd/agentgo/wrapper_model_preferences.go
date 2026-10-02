package main

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/composition/local/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	artifactOverlay "github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/overlay"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	modelOverlay "github.com/flexigpt/flexigpt-app/internal/model/store/overlay"
)

const modelPreferenceSchemaVersion = "v1"

type modelDefaultProviderPreferencePayload struct {
	Provider *artifact.ArtifactRef `json:"provider,omitempty"`
}

type artifactModelDefaultProviderPreferences struct {
	overlays compositionapi.StoreOverlayAPI
}

func newArtifactModelDefaultProviderPreferences(
	overlays compositionapi.StoreOverlayAPI,
) (*artifactModelDefaultProviderPreferences, error) {
	if overlays == nil {
		return nil, fmt.Errorf(
			"%w: Model preference overlay store is required",
			model.ErrInvalid,
		)
	}
	return &artifactModelDefaultProviderPreferences{
		overlays: overlays,
	}, nil
}

func (s *artifactModelDefaultProviderPreferences) GetDefaultProvider(
	ctx context.Context,
) (*artifact.ArtifactRef, error) {
	if s == nil || s.overlays == nil {
		return nil, model.ErrClosed
	}

	record, found, err := s.overlays.GetStoreOverlay(
		ctx,
		modelOverlay.PreferencesNamespace,
	)
	if err != nil || !found {
		return nil, err
	}

	var payload modelDefaultProviderPreferencePayload
	if err := jsonutil.DecodeCanonicalObjectExactInto(
		record.Payload,
		&payload,
		model.MaxLocalDataBytes,
	); err != nil {
		return nil, fmt.Errorf(
			"%w: decode Model default-provider preference: %w",
			model.ErrInvalid,
			err,
		)
	}
	if payload.Provider == nil {
		//nolint:nilnil // Ok.
		return nil, nil
	}
	if err := payload.Provider.Validate(); err != nil {
		return nil, err
	}

	value := *payload.Provider
	return &value, nil
}

func (s *artifactModelDefaultProviderPreferences) SetDefaultProvider(
	ctx context.Context,
	provider *artifact.ArtifactRef,
) error {
	if s == nil || s.overlays == nil {
		return model.ErrClosed
	}
	if provider != nil {
		if err := provider.Validate(); err != nil {
			return err
		}
	}

	current, found, err := s.overlays.GetStoreOverlay(
		ctx,
		modelOverlay.PreferencesNamespace,
	)
	if err != nil {
		return err
	}

	if provider == nil {
		if !found {
			return nil
		}
		return s.overlays.DeleteStoreOverlay(
			ctx,
			modelOverlay.PreferencesNamespace,
			current.Revision,
		)
	}

	payload, err := jsonutil.MarshalCanonicalObject(
		modelDefaultProviderPreferencePayload{
			Provider: provider,
		},
		model.MaxLocalDataBytes,
	)
	if err != nil {
		return err
	}

	expectedRevision := uint64(0)
	if found {
		expectedRevision = current.Revision
	}

	_, err = s.overlays.PutStoreOverlay(
		ctx,
		artifactOverlay.StorePutRequest{
			Namespace:        modelOverlay.PreferencesNamespace,
			SchemaVersion:    modelPreferenceSchemaVersion,
			Payload:          payload,
			ExpectedRevision: expectedRevision,
		},
	)
	return err
}
