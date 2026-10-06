package modelpreferences

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	modelOverlay "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/overlay"
)

const schemaVersion = "v1"

type payload struct {
	Provider *artifactModel.ArtifactRef `json:"provider,omitempty"`
}

// Store owns application preference persistence. Model-family preference
// selection remains in llmartifactory-go/model.
type Store struct {
	overlays overlay.StoreAPI
}

func New(
	overlays overlay.StoreAPI,
) (*Store, error) {
	if overlays == nil {
		return nil, fmt.Errorf(
			"%w: Model preference overlay store is required",
			spec.ErrInvalid,
		)
	}
	return &Store{overlays: overlays}, nil
}

func (s *Store) GetDefaultProvider(
	ctx context.Context,
) (*artifactModel.ArtifactRef, error) {
	record, found, err := s.overlays.GetStoreOverlay(
		ctx,
		modelOverlay.PreferencesNamespace,
	)
	if err != nil || !found {
		return nil, err
	}

	var value payload
	if err := jsonutil.DecodeCanonicalObjectExactInto(
		record.Payload,
		&value,
		spec.MaxLocalDataBytes,
	); err != nil {
		return nil, fmt.Errorf(
			"%w: decode Model default-provider preference: %w",
			spec.ErrInvalid,
			err,
		)
	}
	if value.Provider == nil {
		//nolint:nilnil // Ok.
		return nil, nil
	}
	if err := value.Provider.Validate(); err != nil {
		return nil, err
	}

	output := *value.Provider
	return &output, nil
}

func (s *Store) SetDefaultProvider(
	ctx context.Context,
	provider *artifactModel.ArtifactRef,
) error {
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

	raw, err := jsonutil.MarshalCanonicalObject(
		payload{Provider: provider},
		spec.MaxLocalDataBytes,
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
		overlayModel.StorePutRequest{
			Namespace:        modelOverlay.PreferencesNamespace,
			SchemaVersion:    schemaVersion,
			Payload:          raw,
			ExpectedRevision: expectedRevision,
		},
	)
	return err
}
