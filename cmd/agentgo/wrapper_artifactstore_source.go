package main

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
)

// ArtifactSourceDraft is intentionally write-only. Source configuration can
// include local filesystem paths or provider credentials and is never returned
// from a public Artifact Store API.
type ArtifactSourceDraft struct {
	Kind        artifactstore.SourceKind `json:"kind"        required:"true"`
	DisplayName string                   `json:"displayName" required:"true"`
	Enabled     bool                     `json:"enabled"`
	Config      json.RawMessage          `json:"config"`
}

type CreateArtifactSourceRequest struct {
	RootID artifactstore.RootID `path:"rootID" required:"true"`
	Body   *ArtifactSourceDraft
}

type CreateArtifactSourceResponse struct {
	Body *source.Summary
}

func (w *ArtifactStoreWrapper) CreateArtifactSource(
	request *CreateArtifactSourceRequest,
) (*CreateArtifactSourceResponse, error) {
	return middleware.WithRecoveryResp(func() (*CreateArtifactSourceResponse, error) {
		if w == nil || w.components == nil ||
			request == nil || request.Body == nil {
			return nil, errors.New("invalid Artifact Source request")
		}
		value, err := w.components.Sources.Create(
			context.Background(),
			request.RootID,
			source.Draft{
				Kind:        request.Body.Kind,
				DisplayName: request.Body.DisplayName,
				Enabled:     request.Body.Enabled,
				Config:      append(json.RawMessage(nil), request.Body.Config...),
			},
		)
		if err != nil {
			return nil, err
		}
		return &CreateArtifactSourceResponse{Body: &value}, nil
	})
}

type GetArtifactSourceRequest struct {
	RootID   artifactstore.RootID   `path:"rootID"   required:"true"`
	SourceID artifactstore.SourceID `path:"sourceID" required:"true"`
}

type GetArtifactSourceResponse struct {
	Body *source.Summary
}

func (w *ArtifactStoreWrapper) GetArtifactSource(
	request *GetArtifactSourceRequest,
) (*GetArtifactSourceResponse, error) {
	return middleware.WithRecoveryResp(func() (*GetArtifactSourceResponse, error) {
		if w == nil || w.components == nil || request == nil {
			return nil, errors.New("invalid Artifact Source request")
		}
		value, err := w.components.Sources.Get(
			context.Background(),
			request.RootID,
			request.SourceID,
		)
		if err != nil {
			return nil, err
		}
		return &GetArtifactSourceResponse{Body: &value}, nil
	})
}

type ListArtifactSourcesRequest struct {
	RootID artifactstore.RootID `path:"rootID" required:"true"`
}

type ListArtifactSourcesResponseBody struct {
	Sources []source.Summary `json:"sources"`
}

type ListArtifactSourcesResponse struct {
	Body *ListArtifactSourcesResponseBody
}

func (w *ArtifactStoreWrapper) ListArtifactSources(
	request *ListArtifactSourcesRequest,
) (*ListArtifactSourcesResponse, error) {
	return middleware.WithRecoveryResp(func() (*ListArtifactSourcesResponse, error) {
		if w == nil || w.components == nil || request == nil {
			return nil, errors.New("invalid Artifact Source request")
		}
		values, err := w.components.Sources.List(
			context.Background(),
			request.RootID,
		)
		if err != nil {
			return nil, err
		}
		return &ListArtifactSourcesResponse{
			Body: &ListArtifactSourcesResponseBody{Sources: values},
		}, nil
	})
}

type UpdateArtifactSourceRequestBody struct {
	ExpectedRevision uint64          `json:"expectedRevision" required:"true"`
	DisplayName      string          `json:"displayName"      required:"true"`
	Enabled          bool            `json:"enabled"`
	Config           json.RawMessage `json:"config"`
}

type UpdateArtifactSourceRequest struct {
	RootID   artifactstore.RootID   `path:"rootID"   required:"true"`
	SourceID artifactstore.SourceID `path:"sourceID" required:"true"`
	Body     *UpdateArtifactSourceRequestBody
}

type UpdateArtifactSourceResponse struct {
	Body *source.Summary
}

func (w *ArtifactStoreWrapper) UpdateArtifactSource(
	request *UpdateArtifactSourceRequest,
) (*UpdateArtifactSourceResponse, error) {
	return middleware.WithRecoveryResp(func() (*UpdateArtifactSourceResponse, error) {
		if w == nil || w.components == nil ||
			request == nil || request.Body == nil {
			return nil, errors.New("invalid Artifact Source request")
		}
		value, err := w.components.Sources.Update(
			context.Background(),
			request.RootID,
			request.SourceID,
			source.Update{
				ExpectedRevision: request.Body.ExpectedRevision,
				DisplayName:      request.Body.DisplayName,
				Enabled:          request.Body.Enabled,
				Config:           append(json.RawMessage(nil), request.Body.Config...),
			},
		)
		if err != nil {
			return nil, err
		}
		w.notifyRootMutation(request.RootID)
		return &UpdateArtifactSourceResponse{Body: &value}, nil
	})
}

type RetireArtifactSourceRequest struct {
	RootID           artifactstore.RootID   `path:"rootID"   required:"true"`
	SourceID         artifactstore.SourceID `path:"sourceID" required:"true"`
	ExpectedRevision uint64                 `                required:"true" json:"expectedRevision"`
}

type RetireArtifactSourceResponse struct {
	Body *source.Summary
}

func (w *ArtifactStoreWrapper) RetireArtifactSource(
	request *RetireArtifactSourceRequest,
) (*RetireArtifactSourceResponse, error) {
	return middleware.WithRecoveryResp(func() (*RetireArtifactSourceResponse, error) {
		if w == nil || w.components == nil || request == nil {
			return nil, errors.New("invalid Artifact Source request")
		}
		value, err := w.components.Sources.Retire(
			context.Background(),
			request.RootID,
			request.SourceID,
			request.ExpectedRevision,
		)
		if err != nil {
			return nil, err
		}
		w.notifyRootMutation(request.RootID)
		return &RetireArtifactSourceResponse{Body: &value}, nil
	})
}

type PurgeArtifactSourceRequest struct {
	RootID           artifactstore.RootID   `path:"rootID"   required:"true"`
	SourceID         artifactstore.SourceID `path:"sourceID" required:"true"`
	ExpectedRevision uint64                 `                required:"true" json:"expectedRevision"`
}

type PurgeArtifactSourceResponse struct {
	RootID   artifactstore.RootID   `json:"rootID"`
	SourceID artifactstore.SourceID `json:"sourceID"`
}

func (w *ArtifactStoreWrapper) PurgeArtifactSource(
	request *PurgeArtifactSourceRequest,
) (*PurgeArtifactSourceResponse, error) {
	return middleware.WithRecoveryResp(func() (*PurgeArtifactSourceResponse, error) {
		if w == nil || w.components == nil || request == nil {
			return nil, errors.New("invalid Artifact Source request")
		}
		if err := w.components.Sources.Purge(
			context.Background(),
			request.RootID,
			request.SourceID,
			request.ExpectedRevision,
		); err != nil {
			return nil, err
		}
		w.notifyRootMutation(request.RootID)
		return &PurgeArtifactSourceResponse{
			RootID:   request.RootID,
			SourceID: request.SourceID,
		}, nil
	})
}

type ListArtifactSourceKindsRequest struct{}

type ListArtifactSourceKindsResponseBody struct {
	Kinds []artifactstore.SourceKind `json:"kinds"`
}

type ListArtifactSourceKindsResponse struct {
	Body *ListArtifactSourceKindsResponseBody
}

func (w *ArtifactStoreWrapper) ListArtifactSourceKinds(
	_ *ListArtifactSourceKindsRequest,
) (*ListArtifactSourceKindsResponse, error) {
	return middleware.WithRecoveryResp(func() (*ListArtifactSourceKindsResponse, error) {
		if w == nil || w.components == nil {
			return nil, errors.New("artifact store is not initialized")
		}
		return &ListArtifactSourceKindsResponse{
			Body: &ListArtifactSourceKindsResponseBody{
				Kinds: w.components.Sources.Kinds(),
			},
		}, nil
	})
}
