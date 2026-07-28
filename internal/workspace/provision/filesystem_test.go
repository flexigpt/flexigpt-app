package provision

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
)

func TestNewServiceValidatesDependencies(t *testing.T) {
	t.Parallel()

	if _, err := NewService(nil, nil); !errors.Is(err, engine.ErrInvalidWorkspace) {
		t.Fatalf("NewService(nil,nil) error=%v, want ErrInvalidWorkspace", err)
	}
	if _, err := NewService(provisionTestSources{}, provisionTestWorkspaces{}); err != nil {
		t.Fatalf("NewService valid dependencies: %v", err)
	}
}

func TestCreateFilesystemCreatesSourceAndWorkspace(t *testing.T) {
	t.Parallel()

	var createdDraft source.Draft
	var workspaceRequest engine.FilesystemWorkspaceRequest
	sources := provisionTestSources{
		create: func(_ context.Context, rootID artifactstore.RootID, draft source.Draft) (source.Summary, error) {
			if rootID != provisionTestRootID {
				t.Fatalf("source root=%q", rootID)
			}
			createdDraft = draft
			return source.Summary{ID: provisionTestSourceID, Revision: 7}, nil
		},
	}
	workspaces := provisionTestWorkspaces{
		create: func(_ context.Context, request engine.FilesystemWorkspaceRequest) (engine.Workspace, error) {
			workspaceRequest = request
			return engine.Workspace{}, nil
		},
	}
	service, err := NewService(sources, workspaces)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	_, err = service.CreateFilesystem(t.Context(), Request{
		RootID:      provisionTestRootID,
		DisplayName: "Project",
		Description: "Description",
		RootPath:    "/portable-test-path",
		Discovery: engine.DiscoveryPreferences{
			IncludeReadme: true,
		},
	})
	if err != nil {
		t.Fatalf("CreateFilesystem: %v", err)
	}
	if createdDraft.Kind != fsdir.Kind || createdDraft.DisplayName != "Project" || !createdDraft.Enabled {
		t.Fatalf("source draft=%#v", createdDraft)
	}
	var config fsdir.Config
	if err := json.Unmarshal(createdDraft.Config, &config); err != nil {
		t.Fatalf("decode filesystem config: %v", err)
	}
	if config.RootPath != "/portable-test-path" {
		t.Fatalf("filesystem config=%#v", config)
	}
	if workspaceRequest.RootID != provisionTestRootID || workspaceRequest.PrimarySourceID != provisionTestSourceID ||
		workspaceRequest.DisplayName != "Project" || workspaceRequest.Description != "Description" ||
		!workspaceRequest.Discovery.IncludeReadme {
		t.Fatalf("workspace request=%#v", workspaceRequest)
	}
}

func TestCreateFilesystemCompensatesWorkspaceFailure(t *testing.T) {
	t.Parallel()

	workspaceErr := errors.New("workspace create failed")
	discardErr := errors.New("source discard failed")
	var discardCalled bool
	sources := provisionTestSources{
		create: func(context.Context, artifactstore.RootID, source.Draft) (source.Summary, error) {
			return source.Summary{ID: provisionTestSourceID, Revision: 3}, nil
		},
		discard: func(ctx context.Context, rootID artifactstore.RootID, id artifactstore.SourceID, revision uint64) error {
			discardCalled = true
			if ctx.Err() != nil || rootID != provisionTestRootID || id != provisionTestSourceID || revision != 3 {
				t.Fatalf(
					"discard context/request invalid: err=%v root=%q id=%q revision=%d",
					ctx.Err(),
					rootID,
					id,
					revision,
				)
			}
			return discardErr
		},
	}
	workspaces := provisionTestWorkspaces{
		create: func(context.Context, engine.FilesystemWorkspaceRequest) (engine.Workspace, error) {
			return engine.Workspace{}, workspaceErr
		},
	}
	service, err := NewService(sources, workspaces)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = service.CreateFilesystem(
		ctx,
		Request{RootID: provisionTestRootID, DisplayName: "Project", RootPath: "/root"},
	)
	if !discardCalled || !errors.Is(err, workspaceErr) || !errors.Is(err, discardErr) {
		t.Fatalf("CreateFilesystem error=%v discardCalled=%t", err, discardCalled)
	}

	sourceErr := errors.New("source create failed")
	service, err = NewService(provisionTestSources{
		create: func(context.Context, artifactstore.RootID, source.Draft) (source.Summary, error) {
			return source.Summary{}, sourceErr
		},
	}, workspaces)
	if err != nil {
		t.Fatalf("NewService source failure fixture: %v", err)
	}
	if _, err := service.CreateFilesystem(
		t.Context(),
		Request{RootID: provisionTestRootID, DisplayName: "Project", RootPath: "/root"},
	); !errors.Is(
		err,
		sourceErr,
	) {
		t.Fatalf("source failure error=%v", err)
	}
}

const (
	provisionTestRootID   artifactstore.RootID   = "019d3150-6e01-7a6b-a34e-d9032342bc31"
	provisionTestSourceID artifactstore.SourceID = "019d3150-6e02-7a6b-a34e-d9032342bc31"
)

type provisionTestSources struct {
	create  func(context.Context, artifactstore.RootID, source.Draft) (source.Summary, error)
	discard func(context.Context, artifactstore.RootID, artifactstore.SourceID, uint64) error
}

func (s provisionTestSources) Create(
	ctx context.Context,
	rootID artifactstore.RootID,
	draft source.Draft,
) (source.Summary, error) {
	if s.create == nil {
		return source.Summary{}, errors.New("unexpected source create")
	}
	return s.create(ctx, rootID, draft)
}

func (s provisionTestSources) Discard(
	ctx context.Context,
	rootID artifactstore.RootID,
	id artifactstore.SourceID,
	revision uint64,
) error {
	if s.discard == nil {
		return nil
	}
	return s.discard(ctx, rootID, id, revision)
}

type provisionTestWorkspaces struct {
	create func(context.Context, engine.FilesystemWorkspaceRequest) (engine.Workspace, error)
}

func (w provisionTestWorkspaces) CreateFilesystem(
	ctx context.Context,
	request engine.FilesystemWorkspaceRequest,
) (engine.Workspace, error) {
	if w.create == nil {
		return engine.Workspace{}, errors.New("unexpected workspace create")
	}
	return w.create(ctx, request)
}
