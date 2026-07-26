package main

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
)

type PurgeArtifactRequest struct {
	Artifact         artifactstore.ArtifactRef `json:"artifact"         required:"true"`
	ExpectedRevision uint64                    `json:"expectedRevision" required:"true"`
}

type PurgeArtifactResponse struct {
	Artifact artifactstore.ArtifactRef `json:"artifact"`
}

type GetManagedSourceStateRequest struct {
	RootID   artifactstore.RootID   `json:"rootID"   required:"true"`
	SourceID artifactstore.SourceID `json:"sourceID" required:"true"`
}

type GetManagedSourceStateResponseBody struct {
	Generation string         `json:"generation"`
	Source     source.Summary `json:"source"`
}

type GetManagedSourceStateResponse struct {
	Body *GetManagedSourceStateResponseBody
}

type PublishManagedSourcePackageRequestBody struct {
	ExpectedSourceRevision uint64                      `json:"expectedSourceRevision"       required:"true"`
	Directory              artifactstore.Locator       `json:"directory"                    required:"true"`
	ExpectedGeneration     string                      `json:"expectedGeneration,omitempty"`
	Files                  []source.ManagedPackageFile `json:"files"                        required:"true"`
}

type PublishManagedSourcePackageRequest struct {
	RootID   artifactstore.RootID   `json:"rootID"   required:"true"`
	SourceID artifactstore.SourceID `json:"sourceID" required:"true"`
	Body     *PublishManagedSourcePackageRequestBody
}

type PublishManagedSourcePackageResponseBody struct {
	Generation string         `json:"generation"`
	Source     source.Summary `json:"source"`
}

type PublishManagedSourcePackageResponse struct {
	Body *PublishManagedSourcePackageResponseBody
}

type RemoveManagedSourcePackageRequest struct {
	RootID                 artifactstore.RootID   `json:"rootID"                 required:"true"`
	SourceID               artifactstore.SourceID `json:"sourceID"               required:"true"`
	ExpectedSourceRevision uint64                 `json:"expectedSourceRevision" required:"true"`
	Directory              artifactstore.Locator  `json:"directory"              required:"true"`
	ExpectedGeneration     string                 `json:"expectedGeneration"     required:"true"`
}

type RemoveManagedSourcePackageResponseBody struct {
	Generation string         `json:"generation"`
	Source     source.Summary `json:"source"`
}

type RemoveManagedSourcePackageResponse struct {
	Body *RemoveManagedSourcePackageResponseBody
}

func (w *ArtifactStoreWrapper) GetManagedSourceState(
	request *GetManagedSourceStateRequest,
) (*GetManagedSourceStateResponse, error) {
	return middleware.WithRecoveryResp(func() (*GetManagedSourceStateResponse, error) {
		if w == nil || w.components == nil || request == nil {
			return nil, errors.New("artifact store wrapper is not initialized")
		}
		result, err := w.components.GetManagedSourceState(
			context.Background(),
			request.RootID,
			request.SourceID,
		)
		if err != nil {
			return nil, err
		}
		return &GetManagedSourceStateResponse{
			Body: &GetManagedSourceStateResponseBody{
				Generation: result.Generation,
				Source:     result.Source,
			},
		}, nil
	})
}

func (w *ArtifactStoreWrapper) PurgeArtifact(
	request *PurgeArtifactRequest,
) (*PurgeArtifactResponse, error) {
	return middleware.WithRecoveryResp(func() (*PurgeArtifactResponse, error) {
		if w == nil || w.components == nil || request == nil {
			return nil, errors.New("artifact store wrapper is not initialized")
		}
		if err := w.components.Artifacts.Purge(
			context.Background(),
			request.Artifact,
			request.ExpectedRevision,
		); err != nil {
			return nil, err
		}
		w.notifyRootMutation(request.Artifact.RootID)
		return &PurgeArtifactResponse{
			Artifact: request.Artifact,
		}, nil
	})
}

func (w *ArtifactStoreWrapper) PublishManagedSourcePackage(
	request *PublishManagedSourcePackageRequest,
) (*PublishManagedSourcePackageResponse, error) {
	return middleware.WithRecoveryResp(func() (*PublishManagedSourcePackageResponse, error) {
		if w == nil ||
			w.components == nil ||
			request == nil ||
			request.Body == nil {
			return nil, errors.New("artifact store wrapper is not initialized")
		}
		ctx := context.Background()
		result, err := w.components.PublishManagedPackage(
			ctx,
			request.RootID,
			request.SourceID,
			request.Body.ExpectedSourceRevision,
			source.ManagedPackagePublication{
				Directory:          request.Body.Directory,
				ExpectedGeneration: request.Body.ExpectedGeneration,
				Files:              request.Body.Files,
			},
		)
		if err != nil {
			return nil, err
		}
		w.notifyRootMutation(request.RootID)
		return &PublishManagedSourcePackageResponse{
			Body: &PublishManagedSourcePackageResponseBody{
				Generation: result.Generation,
				Source:     result.Source,
			},
		}, nil
	})
}

func (w *ArtifactStoreWrapper) RemoveManagedSourcePackage(
	request *RemoveManagedSourcePackageRequest,
) (*RemoveManagedSourcePackageResponse, error) {
	return middleware.WithRecoveryResp(func() (*RemoveManagedSourcePackageResponse, error) {
		if w == nil || w.components == nil || request == nil {
			return nil, errors.New("artifact store wrapper is not initialized")
		}
		ctx := context.Background()
		result, err := w.components.RemoveManagedPackage(
			ctx,
			request.RootID,
			request.SourceID,
			request.ExpectedSourceRevision,
			request.Directory,
			request.ExpectedGeneration,
		)
		if err != nil {
			return nil, err
		}
		w.notifyRootMutation(request.RootID)
		return &RemoveManagedSourcePackageResponse{
			Body: &RemoveManagedSourcePackageResponseBody{
				Generation: result.Generation,
				Source:     result.Source,
			},
		}, nil
	})
}
