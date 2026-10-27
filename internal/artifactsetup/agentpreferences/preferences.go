// Package agentpreferences persists the application-level explicit default
// Agent override. Effective Agent fallback selection remains in the Agent
// family service.
package agentpreferences

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const schemaVersion = "v1"

const preferencesNamespace overlayModel.Namespace = "agent.preferences"

// StoreNamespaces returns the Store-scoped namespaces owned by this package.
func StoreNamespaces() []overlayModel.Namespace {
	return []overlayModel.Namespace{preferencesNamespace}
}

type payload struct {
	Agent *artifactModel.ArtifactRef `json:"agent,omitempty"`
}

// Store owns application preference persistence. Agent-family fallback and
// availability selection remain in llmartifactory-go/agent.
type Store struct {
	overlays overlay.StoreAPI
}

func New(
	overlays overlay.StoreAPI,
) (*Store, error) {
	if overlays == nil {
		return nil, fmt.Errorf(
			"%w: Agent preference overlay store is required",
			storeSpec.ErrInvalid,
		)
	}
	return &Store{overlays: overlays}, nil
}

// GetDefaultAgent returns the saved explicit Agent override. A nil result means
// that callers must use the application built-in fallback selection.
func (s *Store) GetDefaultAgent(
	ctx context.Context,
) (*artifactModel.ArtifactRef, error) {
	record, found, err := s.overlays.GetStoreOverlay(
		ctx,
		preferencesNamespace,
	)
	if err != nil || !found {
		return nil, err
	}

	var value payload
	if err := jsonutil.DecodeCanonicalObjectExactInto(
		record.Payload,
		&value,
		storeSpec.MaxLocalDataBytes,
	); err != nil {
		return nil, fmt.Errorf(
			"%w: decode Agent default preference: %w",
			storeSpec.ErrInvalid,
			err,
		)
	}
	if value.Agent == nil {
		//nolint:nilnil // Ok.
		return nil, nil
	}
	if err := value.Agent.Validate(); err != nil {
		return nil, err
	}

	output := *value.Agent
	return &output, nil
}

// SetDefaultAgent saves an explicit Agent override. A nil value clears the
// override and makes effective selection return to the built-in fallback.
func (s *Store) SetDefaultAgent(
	ctx context.Context,
	agent *artifactModel.ArtifactRef,
) error {
	if agent != nil {
		if err := agent.Validate(); err != nil {
			return err
		}
	}

	current, found, err := s.overlays.GetStoreOverlay(
		ctx,
		preferencesNamespace,
	)
	if err != nil {
		return err
	}

	if agent == nil {
		if !found {
			return nil
		}
		return s.overlays.DeleteStoreOverlay(
			ctx,
			preferencesNamespace,
			current.Revision,
		)
	}

	value := payload{Agent: agent}
	raw, err := jsonutil.MarshalCanonicalObject(
		value,
		storeSpec.MaxLocalDataBytes,
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
			Namespace:        preferencesNamespace,
			SchemaVersion:    schemaVersion,
			Payload:          raw,
			ExpectedRevision: expectedRevision,
		},
	)
	return err
}
