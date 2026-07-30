package skillbundle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/managed"
	"github.com/flexigpt/flexigpt-app/internal/skillartifact"
)

func (a *API) BootstrapBuiltInBundle(
	ctx context.Context,
	request BootstrapBundleRequest,
) (Bundle, error) {
	if err := a.Ready(); err != nil {
		return Bundle{}, err
	}

	a.bootstrapMu.Lock()
	defer a.bootstrapMu.Unlock()
	if err := basespec.ValidateIdentifier(
		"built-in skill bundle bootstrap key",
		request.BootstrapKey,
		basespec.MaxKindBytes,
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
			RootID:       request.RootID,
			DisplayName:  request.DisplayName,
			Description:  request.Description,
			Enabled:      true,
			BootstrapKey: request.BootstrapKey,
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
		existingSkill, err := a.findSkillByName(
			ctx,
			existing.Collection.Ref(),
			skill.Name,
		)
		if err != nil {
			return Bundle{}, err
		}
		if existingSkill != nil {
			if existingSkill.Enabled != skill.Enabled {
				if _, err := a.dependencies.Artifacts.SetEnabled(
					ctx,
					existingSkill.Ref(),
					existingSkill.Revision,
					skill.Enabled,
				); err != nil {
					return Bundle{}, err
				}
			}
			continue
		}
		if _, err := a.createManagedSkill(ctx, CreateManagedSkillRequest{
			Bundle:                     existing.Collection.Ref(),
			ExpectedCollectionRevision: existing.Collection.Revision,
			SkillName:                  skill.Name,
			OperationKey:               "builtin-" + request.BootstrapKey + "-" + skill.Name,
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
		matches := value.IdempotencyKey == key
		if !matches {
			// Compatibility for bundles created before migration 6. New
			// bundle writes use Collection.IdempotencyKey and do not duplicate
			// the key in opaque Collection data.
			data, err := DecodeCollectionData(value.Data)
			if err != nil {
				return Bundle{}, err
			}
			matches = data.BootstrapKey == key
		}
		if !matches {
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

func (a *API) findSkillByName(
	ctx context.Context,
	bundle collection.CollectionRef,
	name string,
) (*artifact.Artifact, error) {
	records, err := a.ListSkills(ctx, bundle)
	if err != nil {
		return nil, err
	}
	for index := range records {
		record := records[index]
		if record.ResolvedDefinition == nil {
			continue
		}
		value, err := definition.ReadCanonical(
			ctx,
			a.dependencies.Definitions,
			record.RootID,
			*record.ResolvedDefinition,
		)
		if err != nil {
			return nil, err
		}
		if value.Kind != skillartifact.Kind ||
			value.LogicalName != basespec.LogicalName(name) {
			continue
		}
		return &record, nil
	}
	//nolint:nilnil // Explicit.
	return nil, nil
}
