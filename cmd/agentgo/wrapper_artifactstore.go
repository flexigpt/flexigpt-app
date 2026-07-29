package main

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/workspace"
)

type ArtifactStoreWrapper struct {
	components *system.Components

	mutationMu     sync.RWMutex
	onRootMutation func(basespec.RootID)
}

func InitArtifactStoreWrapper(
	wrapper *ArtifactStoreWrapper,
	baseDirectory string,
) error {
	if wrapper == nil {
		return errors.New("artifact store wrapper is nil")
	}
	components, err := system.Open(context.Background(), system.Config{
		BaseDirectory: baseDirectory,
		Decoders:      workspace.BuiltinDecoders(),
	})
	if err != nil {
		return err
	}
	wrapper.components = components
	return nil
}

type CreateArtifactRootRequest struct {
	Body *root.RootDraft
}

type CreateArtifactRootResponse struct {
	Body *root.Root
}

func (w *ArtifactStoreWrapper) CreateArtifactRoot(
	request *CreateArtifactRootRequest,
) (*CreateArtifactRootResponse, error) {
	return middleware.WithRecoveryResp(func() (*CreateArtifactRootResponse, error) {
		if w == nil || w.components == nil ||
			request == nil || request.Body == nil {
			return nil, errors.New("invalid Artifact Root request")
		}
		value, err := w.components.Roots.Create(
			context.Background(),
			*request.Body,
		)
		if err != nil {
			return nil, err
		}
		return &CreateArtifactRootResponse{Body: &value}, nil
	})
}

type GetArtifactRootRequest struct {
	RootID basespec.RootID `path:"rootID" required:"true"`
}

type GetArtifactRootResponse struct {
	Body *root.Root
}

func (w *ArtifactStoreWrapper) GetArtifactRoot(
	request *GetArtifactRootRequest,
) (*GetArtifactRootResponse, error) {
	return middleware.WithRecoveryResp(func() (*GetArtifactRootResponse, error) {
		if w == nil || w.components == nil || request == nil {
			return nil, errors.New("invalid Artifact Root request")
		}
		value, err := w.components.Roots.Get(
			context.Background(),
			request.RootID,
		)
		if err != nil {
			return nil, err
		}
		return &GetArtifactRootResponse{Body: &value}, nil
	})
}

type ListArtifactRootsRequest struct{}

type ListArtifactRootsResponseBody struct {
	Roots []root.Root `json:"roots"`
}

type ListArtifactRootsResponse struct {
	Body *ListArtifactRootsResponseBody
}

func (w *ArtifactStoreWrapper) ListArtifactRoots(
	_ *ListArtifactRootsRequest,
) (*ListArtifactRootsResponse, error) {
	return middleware.WithRecoveryResp(func() (*ListArtifactRootsResponse, error) {
		if w == nil || w.components == nil {
			return nil, errors.New("artifact store is not initialized")
		}
		values, err := w.components.Roots.List(context.Background())
		if err != nil {
			return nil, err
		}
		return &ListArtifactRootsResponse{
			Body: &ListArtifactRootsResponseBody{Roots: values},
		}, nil
	})
}

type UpdateArtifactRootRequest struct {
	RootID basespec.RootID `path:"rootID" required:"true"`
	Body   *root.RootUpdate
}

type UpdateArtifactRootResponse struct {
	Body *root.Root
}

func (w *ArtifactStoreWrapper) UpdateArtifactRoot(
	request *UpdateArtifactRootRequest,
) (*UpdateArtifactRootResponse, error) {
	return middleware.WithRecoveryResp(func() (*UpdateArtifactRootResponse, error) {
		if w == nil || w.components == nil ||
			request == nil || request.Body == nil {
			return nil, errors.New("invalid Artifact Root request")
		}
		value, err := w.components.Roots.Update(
			context.Background(),
			request.RootID,
			*request.Body,
		)
		if err != nil {
			return nil, err
		}
		return &UpdateArtifactRootResponse{Body: &value}, nil
	})
}

type RetireArtifactRootRequest struct {
	RootID           basespec.RootID `path:"rootID" required:"true"`
	ExpectedRevision uint64          `              required:"true" json:"expectedRevision"`
}

type RetireArtifactRootResponse struct {
	Body *root.Root
}

func (w *ArtifactStoreWrapper) RetireArtifactRoot(
	request *RetireArtifactRootRequest,
) (*RetireArtifactRootResponse, error) {
	return middleware.WithRecoveryResp(func() (*RetireArtifactRootResponse, error) {
		if w == nil || w.components == nil || request == nil {
			return nil, errors.New("invalid Artifact Root request")
		}
		value, err := w.components.Roots.Retire(
			context.Background(),
			request.RootID,
			request.ExpectedRevision,
		)
		if err != nil {
			return nil, err
		}
		w.notifyRootMutation(request.RootID)
		return &RetireArtifactRootResponse{Body: &value}, nil
	})
}

type PurgeArtifactRootRequest struct {
	RootID           basespec.RootID `path:"rootID" required:"true"`
	ExpectedRevision uint64          `              required:"true" json:"expectedRevision"`
}

type PurgeArtifactRootResponse struct {
	RootID basespec.RootID `json:"rootID"`
}

func (w *ArtifactStoreWrapper) PurgeArtifactRoot(
	request *PurgeArtifactRootRequest,
) (*PurgeArtifactRootResponse, error) {
	return middleware.WithRecoveryResp(func() (*PurgeArtifactRootResponse, error) {
		if w == nil || w.components == nil || request == nil {
			return nil, errors.New("invalid Artifact Root request")
		}
		if err := w.components.Roots.Purge(
			context.Background(),
			request.RootID,
			request.ExpectedRevision,
		); err != nil {
			return nil, err
		}
		w.notifyRootMutation(request.RootID)
		return &PurgeArtifactRootResponse{RootID: request.RootID}, nil
	})
}

func (w *ArtifactStoreWrapper) close() {
	if w == nil || w.components == nil {
		return
	}
	w.mutationMu.Lock()
	w.onRootMutation = nil
	w.mutationMu.Unlock()
	if err := w.components.Close(); err != nil {
		slog.Error("close artifact store", "error", err)
	}
	w.components = nil
}

func (w *ArtifactStoreWrapper) setRootMutationObserver(
	observer func(basespec.RootID),
) {
	if w == nil {
		return
	}
	w.mutationMu.Lock()
	w.onRootMutation = observer
	w.mutationMu.Unlock()
}

func (w *ArtifactStoreWrapper) notifyRootMutation(
	rootID basespec.RootID,
) {
	if w == nil {
		return
	}
	w.mutationMu.RLock()
	observer := w.onRootMutation
	w.mutationMu.RUnlock()
	if observer != nil {
		observer(rootID)
	}
}
