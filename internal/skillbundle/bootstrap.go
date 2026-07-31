package skillbundle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/managed"
)

func (a *API) BootstrapBuiltInBundle(
	ctx context.Context,
	request BootstrapBundleRequest,
) (Bundle, error) {
	if err := a.Ready(); err != nil {
		return Bundle{}, err
	}

	if err := basespec.ValidateIdentifier(
		"built-in skill bundle bootstrap key",
		request.BootstrapKey,
		basespec.MaxKindBytes,
	); err != nil {
		return Bundle{}, err
	}
	if err := basespec.ValidateLogicalName(request.LogicalName); err != nil {
		return Bundle{}, err
	}
	if err := basespec.ValidateLogicalVersion(
		request.LogicalVersion,
		true,
	); err != nil {
		return Bundle{}, err
	}

	existing, err := a.findBundleByBootstrapKey(
		ctx,
		request.RootID,
		request.BootstrapKey,
	)
	if err != nil {
		return Bundle{}, err
	}
	if existing.Collection.ID != "" &&
		(existing.Data.LogicalName != request.LogicalName ||
			existing.Data.LogicalVersion != request.LogicalVersion ||
			!maps.Equal(existing.Data.Labels, request.Labels)) {
		return Bundle{}, fmt.Errorf(
			"%w: built-in bundle bootstrap key %q has incompatible portable metadata",
			basespec.ErrConflict,
			request.BootstrapKey,
		)
	}
	if existing.Collection.ID == "" {
		managedSource, err := a.dependencies.Sources.Create(
			ctx,
			request.RootID,
			source.Draft{
				Kind:        managed.Kind,
				DisplayName: request.DisplayName + " built-in source",
				Enabled:     true,
				Config:      json.RawMessage(`{}`),
			},
		)
		if err != nil {
			return Bundle{}, err
		}

		created, createErr := a.CreateBundle(ctx, CreateBundleRequest{
			RootID:         request.RootID,
			LogicalName:    request.LogicalName,
			LogicalVersion: request.LogicalVersion,
			Labels:         maps.Clone(request.Labels),
			DisplayName:    request.DisplayName,
			Description:    request.Description,
			Enabled:        true,
			BootstrapKey:   request.BootstrapKey,
			Attachments: []AttachmentDraft{{
				SourceID: managedSource.ID,
				Role:     RoleBuiltIn,
				Enabled:  true,
			}},
		})
		if createErr == nil {
			existing = created
		} else {
			discardErr := a.dependencies.Sources.Discard(
				context.WithoutCancel(ctx),
				request.RootID,
				managedSource.ID,
				managedSource.Revision,
			)
			if !errors.Is(createErr, basespec.ErrConflict) ||
				discardErr != nil {
				return Bundle{}, errors.Join(createErr, discardErr)
			}

			// The unique durable idempotency key may have been claimed by a
			// concurrent bootstrapper. Its Collection is the winner; do not
			// create a second bundle or retain this provisional Source.
			existing, err = a.findBundleByBootstrapKey(
				ctx,
				request.RootID,
				request.BootstrapKey,
			)
			if err != nil {
				return Bundle{}, errors.Join(createErr, err)
			}
			if existing.Collection.ID == "" {
				return Bundle{}, createErr
			}
		}
	}

	for _, skill := range request.Skills {
		operationKey := "builtin-" + request.BootstrapKey + "-" + skill.Name
		existingSkill, err := a.findManagedSkillByOperation(
			ctx,
			existing.Collection.Ref(),
			operationKey,
		)
		if err != nil {
			return Bundle{}, err
		}
		if existingSkill != nil && existingSkill.Enabled != skill.Enabled {
			if _, err := a.dependencies.Artifacts.SetEnabled(
				ctx,
				existingSkill.Ref(),
				existingSkill.Revision,
				skill.Enabled,
			); err != nil {
				return Bundle{}, err
			}
		}
		if _, err := a.createManagedSkill(ctx, CreateManagedSkillRequest{
			Bundle:                     existing.Collection.Ref(),
			ExpectedCollectionRevision: existing.Collection.Revision,
			SkillName:                  skill.Name,
			OperationKey:               operationKey,
			SKILLMD:                    skill.SKILLMD,
			Files:                      skill.Files,
			Enabled:                    skill.Enabled,
		}, true); err != nil {
			return Bundle{}, err
		}
	}

	return a.GetBundle(ctx, existing.Collection.Ref())
}

func (a *API) findBundleByBootstrapKey(
	ctx context.Context,
	rootID basespec.RootID,
	key string,
) (Bundle, error) {
	values, err := a.dependencies.Collections.ListByRoot(ctx, rootID)
	if err != nil {
		return Bundle{}, err
	}

	var found *Bundle
	for _, value := range values {
		if value.Kind != CollectionKind {
			continue
		}
		if value.IdempotencyKey != key {
			continue
		}
		bundle, err := a.GetBundle(ctx, value.Ref())
		if err != nil {
			return Bundle{}, err
		}
		if found != nil {
			return Bundle{}, fmt.Errorf(
				"%w: multiple skill bundles use bootstrap key %q",
				basespec.ErrConflict,
				key,
			)
		}
		found = &bundle
	}
	if found != nil {
		return *found, nil
	}
	return Bundle{}, nil
}
