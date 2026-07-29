package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

func TestServiceGetAndLifecycleGuardrails(t *testing.T) {
	t.Parallel()

	if _, err := NewService(nil, nil, "policy.v1"); !errors.Is(err, ErrInvalidWorkspace) {
		t.Fatalf("NewService(nil,nil) error=%v", err)
	}

	workspace := plannerTestWorkspace(t)
	store := &engineTestCollectionStore{
		getFn: func(_ context.Context, ref collection.CollectionRef) (collection.Collection, error) {
			if ref != workspace.Collection.Ref() {
				return collection.Collection{}, basespec.ErrCollectionNotFound
			}
			return workspace.Collection, nil
		},
		listAttachmentsFn: func(_ context.Context, ref collection.CollectionRef) ([]collection.Attachment, error) {
			if ref != workspace.Collection.Ref() {
				return nil, basespec.ErrCollectionNotFound
			}
			return []collection.Attachment{workspace.Attachments[1], workspace.Attachments[0]}, nil
		},
	}
	sources := engineTestSources{
		getFn: func(_ context.Context, rootID basespec.RootID, sourceID basespec.SourceID) (source.Summary, error) {
			if rootID != workspace.Collection.RootID {
				return source.Summary{}, basespec.ErrSourceNotFound
			}
			for _, value := range workspace.Sources {
				if value.ID == sourceID {
					return value, nil
				}
			}
			return source.Summary{}, basespec.ErrSourceNotFound
		},
	}
	service, err := NewService(store, sources, "policy.v1")
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	got, err := service.Get(t.Context(), workspace.Collection.Ref())
	if err != nil || got.Mode != ModeFilesystem || got.PrimarySourceID != workspace.PrimarySourceID ||
		got.Attachments[0].SourceID > got.Attachments[1].SourceID || got.Sources[0].ID > got.Sources[1].ID {
		t.Fatalf("Get workspace=%#v err=%v", got, err)
	}
	if _, err := service.SetPrimary(
		t.Context(),
		SetPrimaryRequest{Workspace: workspace.Collection.Ref()},
	); !errors.Is(
		err,
		ErrInvalidWorkspace,
	) {
		t.Fatalf("SetPrimary missing revision error=%v", err)
	}
	if _, err := service.SetPrimary(
		t.Context(),
		SetPrimaryRequest{
			Workspace:                  workspace.Collection.Ref(),
			ExpectedCollectionRevision: 1,
			Clear:                      true,
			SourceID:                   workspace.PrimarySourceID,
		},
	); !errors.Is(
		err,
		ErrInvalidWorkspace,
	) {
		t.Fatalf("SetPrimary incompatible clear/source error=%v", err)
	}
	if _, err := service.Detach(
		t.Context(),
		workspace.Collection.Ref(),
		workspace.PrimarySourceID,
		0,
		0,
	); !errors.Is(
		err,
		ErrInvalidWorkspace,
	) {
		t.Fatalf("Detach missing revisions error=%v", err)
	}

	disabledSource := workspace.Sources[0]
	disabledSource.Enabled = false
	disabledService, err := NewService(
		store,
		engineTestSources{
			getFn: func(context.Context, basespec.RootID, basespec.SourceID) (source.Summary, error) {
				return disabledSource, nil
			},
		},
		"policy.v1",
	)
	if err != nil {
		t.Fatalf("NewService disabled fixture: %v", err)
	}
	if _, err := disabledService.CreateFilesystem(
		t.Context(),
		FilesystemWorkspaceRequest{RootID: workspace.Collection.RootID, PrimarySourceID: disabledSource.ID},
	); !errors.Is(
		err,
		ErrInvalidWorkspace,
	) {
		t.Fatalf("CreateFilesystem disabled source error=%v", err)
	}
	wrongKind := workspace.Sources[0]
	wrongKind.Kind = "other.source"
	wrongKind.Enabled = true
	wrongKind.RootID = workspace.Collection.RootID
	wrongKind.ID = workspace.PrimarySourceID
	wrongKind.DisplayName = "Other"
	wrongKind.Revision = 1
	wrongKind.CreatedAt = workspace.Collection.CreatedAt
	wrongKind.ModifiedAt = workspace.Collection.ModifiedAt
	kindService, err := NewService(
		store,
		engineTestSources{
			getFn: func(context.Context, basespec.RootID, basespec.SourceID) (source.Summary, error) {
				return wrongKind, nil
			},
		},
		"policy.v1",
	)
	if err != nil {
		t.Fatalf("NewService wrong-kind fixture: %v", err)
	}
	if _, err := kindService.CreateFilesystem(
		t.Context(),
		FilesystemWorkspaceRequest{RootID: workspace.Collection.RootID, PrimarySourceID: wrongKind.ID},
	); !errors.Is(
		err,
		ErrInvalidWorkspace,
	) {
		t.Fatalf("CreateFilesystem wrong kind error=%v", err)
	}
}

func TestServicePurgeChecksRetiredWorkspaceIdentity(t *testing.T) {
	t.Parallel()

	workspace := plannerTestWorkspace(t)
	retired := workspace.Collection
	retired.Enabled = false
	retired.Revision = 3
	store := &engineTestCollectionStore{
		getRetiredFn: func(context.Context, collection.CollectionRef) (collection.Collection, error) { return retired, nil },
		purgeFn:      func(context.Context, collection.CollectionRef, uint64) error { return nil },
	}
	service, err := NewService(store, engineTestSources{}, "policy.v1")
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if err := service.Purge(t.Context(), workspace.Collection.Ref(), 0); !errors.Is(err, ErrInvalidWorkspace) {
		t.Fatalf("Purge missing revision error=%v", err)
	}
	wrongKind := retired
	wrongKind.Kind = "other.collection"
	store.getRetiredFn = func(context.Context, collection.CollectionRef) (collection.Collection, error) {
		return wrongKind, nil
	}
	if err := service.Purge(
		t.Context(),
		workspace.Collection.Ref(),
		retired.Revision,
	); !errors.Is(
		err,
		ErrNotWorkspace,
	) {
		t.Fatalf("Purge wrong kind error=%v", err)
	}
	store.getRetiredFn = func(context.Context, collection.CollectionRef) (collection.Collection, error) { return retired, nil }
	if err := service.Purge(
		t.Context(),
		workspace.Collection.Ref(),
		retired.Revision-1,
	); !errors.Is(
		err,
		basespec.ErrConflict,
	) {
		t.Fatalf("Purge stale revision error=%v", err)
	}
	if err := service.Purge(t.Context(), workspace.Collection.Ref(), retired.Revision); err != nil {
		t.Fatalf("Purge valid retired workspace: %v", err)
	}
}
