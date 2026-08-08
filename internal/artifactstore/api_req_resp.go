package artifactstore

import (
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

type CreateArtifactRootRequest struct {
	Body *root.RootDraft
}

type CreateArtifactRootResponse struct {
	Body *root.Root
}

type GetArtifactRootRequest struct {
	RootID basespec.RootID `path:"rootID" required:"true"`
}

type GetArtifactRootResponse struct {
	Body *root.Root
}

type ListArtifactRootsRequest struct{}

type ListArtifactRootsResponseBody struct {
	Roots []root.Root `json:"roots"`
}

type ListArtifactRootsResponse struct {
	Body *ListArtifactRootsResponseBody
}

type UpdateArtifactRootRequest struct {
	RootID basespec.RootID `path:"rootID" required:"true"`
	Body   *root.RootUpdate
}

type UpdateArtifactRootResponse struct {
	Body *root.Root
}

type RetireArtifactRootRequest struct {
	RootID           basespec.RootID `path:"rootID" required:"true"`
	ExpectedRevision uint64          `              required:"true" json:"expectedRevision"`
}

type RetireArtifactRootResponse struct {
	Body *root.Root
}

type PurgeArtifactRootRequest struct {
	RootID           basespec.RootID `path:"rootID" required:"true"`
	ExpectedRevision uint64          `              required:"true" json:"expectedRevision"`
}

type PurgeArtifactRootResponse struct {
	RootID basespec.RootID `json:"rootID"`
}

type GetShareableCollectionDocumentRequest struct {
	RootID       basespec.RootID       `path:"rootID"       required:"true"`
	CollectionID basespec.CollectionID `path:"collectionID" required:"true"`
}

type GetShareableCollectionDocumentResponse struct {
	Body *shareable.CollectionDocument
}

type StoreShareableCollectionDocumentRequestBody struct {
	Document json.RawMessage `json:"document" required:"true"`
}

type StoreShareableCollectionDocumentRequest struct {
	RootID       basespec.RootID       `path:"rootID"       required:"true"`
	CollectionID basespec.CollectionID `path:"collectionID" required:"true"`
	Body         *StoreShareableCollectionDocumentRequestBody
}

type StoreShareableCollectionDocumentResponse struct {
	Body *shareable.CollectionDocument
}

// ArtifactSourceDraft is write-only. Source configuration can contain local
// filesystem paths or provider credentials and is not returned by the API.
type ArtifactSourceDraft struct {
	ID          basespec.SourceID   `json:"id"          required:"true"`
	Kind        basespec.SourceKind `json:"kind"        required:"true"`
	DisplayName string              `json:"displayName" required:"true"`
	Enabled     bool                `json:"enabled"`
	Config      json.RawMessage     `json:"config"`
}

type CreateArtifactSourceRequest struct {
	RootID basespec.RootID `path:"rootID" required:"true"`
	Body   *ArtifactSourceDraft
}

type CreateArtifactSourceResponse struct {
	Body *source.Summary
}

type GetArtifactSourceRequest struct {
	RootID   basespec.RootID   `path:"rootID"   required:"true"`
	SourceID basespec.SourceID `path:"sourceID" required:"true"`
}

type GetArtifactSourceResponse struct {
	Body *source.Summary
}

type ListArtifactSourcesRequest struct {
	RootID basespec.RootID `path:"rootID" required:"true"`
}

type ListArtifactSourcesResponseBody struct {
	Sources []source.Summary `json:"sources"`
}

type ListArtifactSourcesResponse struct {
	Body *ListArtifactSourcesResponseBody
}

type UpdateArtifactSourceRequestBody struct {
	ExpectedRevision uint64          `json:"expectedRevision" required:"true"`
	DisplayName      string          `json:"displayName"      required:"true"`
	Enabled          bool            `json:"enabled"`
	Config           json.RawMessage `json:"config,omitempty"`
}

type UpdateArtifactSourceRequest struct {
	RootID   basespec.RootID   `path:"rootID"   required:"true"`
	SourceID basespec.SourceID `path:"sourceID" required:"true"`
	Body     *UpdateArtifactSourceRequestBody
}

type UpdateArtifactSourceResponse struct {
	Body *source.Summary
}

type RetireArtifactSourceRequest struct {
	RootID           basespec.RootID   `path:"rootID"   required:"true"`
	SourceID         basespec.SourceID `path:"sourceID" required:"true"`
	ExpectedRevision uint64            `                required:"true" json:"expectedRevision"`
}

type RetireArtifactSourceResponse struct {
	Body *source.Summary
}

type PurgeArtifactSourceRequest struct {
	RootID           basespec.RootID   `path:"rootID"   required:"true"`
	SourceID         basespec.SourceID `path:"sourceID" required:"true"`
	ExpectedRevision uint64            `                required:"true" json:"expectedRevision"`
}

type PurgeArtifactSourceResponse struct {
	RootID   basespec.RootID   `json:"rootID"`
	SourceID basespec.SourceID `json:"sourceID"`
}

type ListArtifactSourceKindsRequest struct{}

type ListArtifactSourceKindsResponseBody struct {
	Kinds []basespec.SourceKind `json:"kinds"`
}

type ListArtifactSourceKindsResponse struct {
	Body *ListArtifactSourceKindsResponseBody
}

type GetManagedSourceStateRequest struct {
	RootID   basespec.RootID   `json:"rootID"   required:"true"`
	SourceID basespec.SourceID `json:"sourceID" required:"true"`
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
	Directory              basespec.Locator            `json:"directory"                    required:"true"`
	ExpectedGeneration     string                      `json:"expectedGeneration,omitempty"`
	Files                  []source.ManagedPackageFile `json:"files"                        required:"true"`
}

type PublishManagedSourcePackageRequest struct {
	RootID   basespec.RootID   `json:"rootID"   required:"true"`
	SourceID basespec.SourceID `json:"sourceID" required:"true"`
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
	RootID                 basespec.RootID   `json:"rootID"                 required:"true"`
	SourceID               basespec.SourceID `json:"sourceID"               required:"true"`
	ExpectedSourceRevision uint64            `json:"expectedSourceRevision" required:"true"`
	Directory              basespec.Locator  `json:"directory"              required:"true"`
	ExpectedGeneration     string            `json:"expectedGeneration"     required:"true"`
}

type RemoveManagedSourcePackageResponseBody struct {
	Generation string         `json:"generation"`
	Source     source.Summary `json:"source"`
}

type RemoveManagedSourcePackageResponse struct {
	Body *RemoveManagedSourcePackageResponseBody
}
