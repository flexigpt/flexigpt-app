package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	settingSpec "github.com/flexigpt/flexigpt-app/internal/setting/spec"
)

const (
	modelPreferenceNamespace = "model-preferences-v1"
	defaultProviderKey       = "default-provider"
)

type modelAuthKeyStore interface {
	GetAuthKey(
		ctx context.Context,
		req *settingSpec.GetAuthKeyRequest,
	) (*settingSpec.GetAuthKeyResponse, error)

	SetAuthKey(
		ctx context.Context,
		req *settingSpec.SetAuthKeyRequest,
	) (*settingSpec.SetAuthKeyResponse, error)
}

type storedDefaultProvider struct {
	Provider *artifact.ArtifactRef `json:"provider,omitempty"`
}

// modelDefaultProviderPreferences is temporary Settings-backed preference
// storage. It stores no credential, overlay, or Model runtime secret.
type modelDefaultProviderPreferences struct {
	store modelAuthKeyStore
}

func newModelDefaultProviderPreferences(
	store modelAuthKeyStore,
) (*modelDefaultProviderPreferences, error) {
	if store == nil {
		return nil, fmt.Errorf(
			"%w: Model default-provider preference store is required",
			basespec.ErrInvalid,
		)
	}
	return &modelDefaultProviderPreferences{
		store: store,
	}, nil
}

func (s *modelDefaultProviderPreferences) GetDefaultProvider(
	ctx context.Context,
) (*artifact.ArtifactRef, error) {
	if s == nil || s.store == nil {
		return nil, basespec.ErrClosed
	}

	response, err := s.store.GetAuthKey(
		ctx,
		&settingSpec.GetAuthKeyRequest{
			Type:    settingSpec.AuthKeyTypeProvider,
			KeyName: settingSpec.AuthKeyName(preferenceStorageKey(defaultProviderKey)),
		},
	)
	if errors.Is(err, settingSpec.ErrAuthKeyNotFound) {
		//nolint:nilnil // Ok.
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if response == nil ||
		response.Body == nil ||
		!response.Body.NonEmpty {
		//nolint:nilnil // Ok.
		return nil, nil
	}

	var value storedDefaultProvider
	if err := json.Unmarshal(
		[]byte(response.Body.Secret),
		&value,
	); err != nil {
		return nil, fmt.Errorf(
			"%w: decode stored default Model Provider preference: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if value.Provider != nil {
		if err := value.Provider.Validate(); err != nil {
			return nil, err
		}
		copyValue := *value.Provider
		return &copyValue, nil
	}
	//nolint:nilnil // Ok.
	return nil, nil
}

func (s *modelDefaultProviderPreferences) SetDefaultProvider(
	ctx context.Context,
	provider *artifact.ArtifactRef,
) error {
	if s == nil || s.store == nil {
		return basespec.ErrClosed
	}
	if provider != nil {
		if err := provider.Validate(); err != nil {
			return err
		}
	}

	raw, err := json.Marshal(storedDefaultProvider{
		Provider: provider,
	})
	if err != nil {
		return err
	}

	_, err = s.store.SetAuthKey(
		ctx,
		&settingSpec.SetAuthKeyRequest{
			Type:    settingSpec.AuthKeyTypeProvider,
			KeyName: settingSpec.AuthKeyName(preferenceStorageKey(defaultProviderKey)),
			Body: &settingSpec.SetAuthKeyRequestBody{
				Secret: string(raw),
			},
		},
	)
	return err
}

func preferenceStorageKey(
	logical string,
) string {
	sum := sha256.Sum256(
		[]byte(modelPreferenceNamespace + ":" + logical),
	)
	return modelPreferenceNamespace + ":" + hex.EncodeToString(sum[:])
}
