package artifactstore

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/managedartifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

// ConsumerAPI is the complete Artifact Store capability granted to
// application-level consumers.
//
// It deliberately exposes Store-owned commands and immutable Store results.
// It does not expose repositories, source snapshots, source configuration,
// discovery plans, artifact reconciliation, provider registries, protected
// topology reset, or system composition.
type ConsumerAPI interface {
	ResourceResolver

	CreateSource(
		ctx context.Context,
		rootID basespec.RootID,
		draft source.Draft,
	) (source.Summary, error)

	CreateSourceWithStatus(
		ctx context.Context,
		rootID basespec.RootID,
		draft source.Draft,
	) (source.Summary, bool, error)

	DiscardSource(
		ctx context.Context,
		rootID basespec.RootID,
		sourceID basespec.SourceID,
		expectedRevision uint64,
	) error

	GetSource(
		ctx context.Context,
		rootID basespec.RootID,
		sourceID basespec.SourceID,
	) (source.Summary, error)

	CreateCollection(
		ctx context.Context,
		rootID basespec.RootID,
		draft collection.Draft,
		attachments []collection.AttachmentDraft,
	) (collection.Collection, []collection.Attachment, error)

	GetCollection(
		ctx context.Context,
		ref collection.CollectionRef,
	) (collection.Collection, error)

	GetRetiredCollection(
		ctx context.Context,
		ref collection.CollectionRef,
	) (collection.Collection, error)

	ListCollections(
		ctx context.Context,
		rootID basespec.RootID,
	) ([]collection.Collection, error)

	UpdateCollection(
		ctx context.Context,
		ref collection.CollectionRef,
		update collection.Update,
	) (collection.Collection, error)

	RetireCollection(
		ctx context.Context,
		ref collection.CollectionRef,
		expectedRevision uint64,
	) (collection.Collection, error)

	PurgeCollection(
		ctx context.Context,
		ref collection.CollectionRef,
		expectedRevision uint64,
	) error

	AttachCollectionSource(
		ctx context.Context,
		ref collection.CollectionRef,
		expectedCollectionRevision uint64,
		draft collection.AttachmentDraft,
	) (collection.Collection, collection.Attachment, error)

	GetCollectionAttachment(
		ctx context.Context,
		ref collection.CollectionRef,
		sourceID basespec.SourceID,
	) (collection.Attachment, error)

	ListCollectionAttachments(
		ctx context.Context,
		ref collection.CollectionRef,
	) ([]collection.Attachment, error)

	UpdateCollectionAttachment(
		ctx context.Context,
		ref collection.CollectionRef,
		sourceID basespec.SourceID,
		update collection.AttachmentUpdate,
	) (collection.Collection, collection.Attachment, error)

	DetachCollectionSource(
		ctx context.Context,
		ref collection.CollectionRef,
		sourceID basespec.SourceID,
		expectedCollectionRevision uint64,
		expectedAttachmentRevision uint64,
	) (collection.Collection, error)

	ReplaceCollectionAttachment(
		ctx context.Context,
		ref collection.CollectionRef,
		replacement collection.AttachmentReplacement,
	) (collection.Collection, collection.Attachment, error)

	GetArtifact(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (artifact.Artifact, error)

	ListCollectionArtifacts(
		ctx context.Context,
		ref collection.CollectionRef,
	) ([]artifact.Artifact, error)

	AdoptArtifact(
		ctx context.Context,
		request artifact.AdoptRequest,
	) (artifact.Artifact, error)

	PinArtifact(
		ctx context.Context,
		request artifact.PinRequest,
	) (artifact.Artifact, error)

	SetArtifactEnabled(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
		enabled bool,
	) (artifact.Artifact, error)

	SetArtifactName(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
		name string,
	) (artifact.Artifact, error)

	UpdateArtifactData(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
		data json.RawMessage,
	) (artifact.Artifact, error)

	UnadoptArtifact(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
		suppress bool,
	) error

	PurgeArtifact(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
	) error

	PurgeAndSuppressArtifact(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
	) error

	ListCollectionSuppressions(
		ctx context.Context,
		ref collection.CollectionRef,
	) ([]artifact.Suppression, error)

	SuppressBinding(
		ctx context.Context,
		request artifact.SuppressRequest,
	) (artifact.Suppression, error)

	UnsuppressBinding(
		ctx context.Context,
		ref collection.CollectionRef,
		binding artifact.SourceBinding,
		expectedRevision uint64,
	) error

	RefreshCollection(
		ctx context.Context,
		ref collection.CollectionRef,
	) (refresh.Result, error)

	CurrentCollectionCatalog(
		ctx context.Context,
		ref collection.CollectionRef,
	) (catalog.Snapshot, error)

	InspectCollectionCatalog(
		ctx context.Context,
		ref collection.CollectionRef,
	) (refresh.CatalogInspection, error)

	CanonicalizeExpectedDocument(
		ctx context.Context,
		expected providerapi.SchemaKey,
		raw []byte,
	) (providerapi.ParsedDocument, error)

	PublishManagedArtifact(
		ctx context.Context,
		request managedartifact.PublishRequest,
	) (managedartifact.PublishResult, error)

	PublishManagedCollection(
		ctx context.Context,
		request managedartifact.PublishCollectionRequest,
	) (managedartifact.PublishCollectionResult, error)

	RemoveManagedArtifact(
		ctx context.Context,
		request managedartifact.RemoveRequest,
	) error
}

func (a *API) CreateCollection(
	ctx context.Context,
	rootID basespec.RootID,
	draft collection.Draft,
	attachments []collection.AttachmentDraft,
) (collection.Collection, []collection.Attachment, error) {
	if err := a.requireStore(ctx); err != nil {
		return collection.Collection{}, nil, err
	}
	return a.components.Collections.Create(
		ctx,
		rootID,
		draft,
		cloneAttachmentDrafts(attachments),
	)
}

func (a *API) GetCollection(
	ctx context.Context,
	ref collection.CollectionRef,
) (collection.Collection, error) {
	if err := a.requireStore(ctx); err != nil {
		return collection.Collection{}, err
	}
	return a.components.Collections.Get(ctx, ref)
}

func (a *API) GetRetiredCollection(
	ctx context.Context,
	ref collection.CollectionRef,
) (collection.Collection, error) {
	if err := a.requireStore(ctx); err != nil {
		return collection.Collection{}, err
	}
	return a.components.Collections.GetRetired(ctx, ref)
}

func (a *API) ListCollections(
	ctx context.Context,
	rootID basespec.RootID,
) ([]collection.Collection, error) {
	if err := a.requireStore(ctx); err != nil {
		return nil, err
	}
	return a.components.Collections.ListByRoot(ctx, rootID)
}

func (a *API) UpdateCollection(
	ctx context.Context,
	ref collection.CollectionRef,
	update collection.Update,
) (collection.Collection, error) {
	if err := a.requireStore(ctx); err != nil {
		return collection.Collection{}, err
	}
	update.Data = append(json.RawMessage(nil), update.Data...)
	return a.components.Collections.Update(ctx, ref, update)
}

func (a *API) RetireCollection(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
) (collection.Collection, error) {
	if err := a.requireStore(ctx); err != nil {
		return collection.Collection{}, err
	}
	return a.components.Collections.Retire(
		ctx,
		ref,
		expectedRevision,
	)
}

func (a *API) PurgeCollection(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
) error {
	if err := a.requireStore(ctx); err != nil {
		return err
	}
	return a.components.Collections.Purge(
		ctx,
		ref,
		expectedRevision,
	)
}

func (a *API) AttachCollectionSource(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedCollectionRevision uint64,
	draft collection.AttachmentDraft,
) (collection.Collection, collection.Attachment, error) {
	if err := a.requireStore(ctx); err != nil {
		return collection.Collection{}, collection.Attachment{}, err
	}
	draft.Data = append(json.RawMessage(nil), draft.Data...)
	return a.components.Collections.Attach(
		ctx,
		ref,
		expectedCollectionRevision,
		draft,
	)
}

func (a *API) GetCollectionAttachment(
	ctx context.Context,
	ref collection.CollectionRef,
	sourceID basespec.SourceID,
) (collection.Attachment, error) {
	if err := a.requireStore(ctx); err != nil {
		return collection.Attachment{}, err
	}
	return a.components.Collections.GetAttachment(
		ctx,
		ref,
		sourceID,
	)
}

func (a *API) ListCollectionAttachments(
	ctx context.Context,
	ref collection.CollectionRef,
) ([]collection.Attachment, error) {
	if err := a.requireStore(ctx); err != nil {
		return nil, err
	}
	return a.components.Collections.ListAttachments(ctx, ref)
}

func (a *API) UpdateCollectionAttachment(
	ctx context.Context,
	ref collection.CollectionRef,
	sourceID basespec.SourceID,
	update collection.AttachmentUpdate,
) (collection.Collection, collection.Attachment, error) {
	if err := a.requireStore(ctx); err != nil {
		return collection.Collection{}, collection.Attachment{}, err
	}
	update.Data = append(json.RawMessage(nil), update.Data...)
	return a.components.Collections.UpdateAttachment(
		ctx,
		ref,
		sourceID,
		update,
	)
}

func (a *API) DetachCollectionSource(
	ctx context.Context,
	ref collection.CollectionRef,
	sourceID basespec.SourceID,
	expectedCollectionRevision uint64,
	expectedAttachmentRevision uint64,
) (collection.Collection, error) {
	if err := a.requireStore(ctx); err != nil {
		return collection.Collection{}, err
	}
	return a.components.Collections.Detach(
		ctx,
		ref,
		sourceID,
		expectedCollectionRevision,
		expectedAttachmentRevision,
	)
}

func (a *API) ReplaceCollectionAttachment(
	ctx context.Context,
	ref collection.CollectionRef,
	replacement collection.AttachmentReplacement,
) (collection.Collection, collection.Attachment, error) {
	if err := a.requireStore(ctx); err != nil {
		return collection.Collection{}, collection.Attachment{}, err
	}
	replacement.Replacement.Data = append(
		json.RawMessage(nil),
		replacement.Replacement.Data...,
	)
	return a.components.Collections.ReplaceAttachment(
		ctx,
		ref,
		replacement,
	)
}

func (a *API) GetArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (artifact.Artifact, error) {
	if err := a.requireStore(ctx); err != nil {
		return artifact.Artifact{}, err
	}
	return a.components.Artifacts.Get(ctx, ref)
}

func (a *API) ListCollectionArtifacts(
	ctx context.Context,
	ref collection.CollectionRef,
) ([]artifact.Artifact, error) {
	if err := a.requireStore(ctx); err != nil {
		return nil, err
	}
	return a.components.Artifacts.ListByCollection(ctx, ref)
}

func (a *API) AdoptArtifact(
	ctx context.Context,
	request artifact.AdoptRequest,
) (artifact.Artifact, error) {
	if err := a.requireStore(ctx); err != nil {
		return artifact.Artifact{}, err
	}
	request.Data = append(json.RawMessage(nil), request.Data...)
	return a.components.Artifacts.Adopt(ctx, request)
}

func (a *API) PinArtifact(
	ctx context.Context,
	request artifact.PinRequest,
) (artifact.Artifact, error) {
	if err := a.requireStore(ctx); err != nil {
		return artifact.Artifact{}, err
	}
	request.Data = append(json.RawMessage(nil), request.Data...)
	return a.components.Artifacts.Pin(ctx, request)
}

func (a *API) SetArtifactEnabled(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifact.Artifact, error) {
	if err := a.requireStore(ctx); err != nil {
		return artifact.Artifact{}, err
	}
	return a.components.Artifacts.SetEnabled(
		ctx,
		ref,
		expectedRevision,
		enabled,
	)
}

func (a *API) SetArtifactName(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	name string,
) (artifact.Artifact, error) {
	if err := a.requireStore(ctx); err != nil {
		return artifact.Artifact{}, err
	}
	return a.components.Artifacts.SetName(
		ctx,
		ref,
		expectedRevision,
		name,
	)
}

func (a *API) UpdateArtifactData(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	data json.RawMessage,
) (artifact.Artifact, error) {
	if err := a.requireStore(ctx); err != nil {
		return artifact.Artifact{}, err
	}
	return a.components.Artifacts.UpdateData(
		ctx,
		ref,
		expectedRevision,
		append(json.RawMessage(nil), data...),
	)
}

func (a *API) UnadoptArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	suppress bool,
) error {
	if err := a.requireStore(ctx); err != nil {
		return err
	}
	return a.components.Artifacts.Unadopt(
		ctx,
		ref,
		expectedRevision,
		suppress,
	)
}

func (a *API) PurgeArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	if err := a.requireStore(ctx); err != nil {
		return err
	}
	return a.components.Artifacts.Purge(
		ctx,
		ref,
		expectedRevision,
	)
}

func (a *API) PurgeAndSuppressArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	if err := a.requireStore(ctx); err != nil {
		return err
	}
	return a.components.Artifacts.PurgeAndSuppress(
		ctx,
		ref,
		expectedRevision,
	)
}

func (a *API) ListCollectionSuppressions(
	ctx context.Context,
	ref collection.CollectionRef,
) ([]artifact.Suppression, error) {
	if err := a.requireStore(ctx); err != nil {
		return nil, err
	}
	return a.components.Artifacts.ListSuppressions(ctx, ref)
}

func (a *API) SuppressBinding(
	ctx context.Context,
	request artifact.SuppressRequest,
) (artifact.Suppression, error) {
	if err := a.requireStore(ctx); err != nil {
		return artifact.Suppression{}, err
	}
	return a.components.Artifacts.Suppress(ctx, request)
}

func (a *API) UnsuppressBinding(
	ctx context.Context,
	ref collection.CollectionRef,
	binding artifact.SourceBinding,
	expectedRevision uint64,
) error {
	if err := a.requireStore(ctx); err != nil {
		return err
	}
	return a.components.Artifacts.Unsuppress(
		ctx,
		ref,
		binding,
		expectedRevision,
	)
}

func (a *API) RefreshCollection(
	ctx context.Context,
	ref collection.CollectionRef,
) (refresh.Result, error) {
	if err := a.requireStore(ctx); err != nil {
		return refresh.Result{}, err
	}
	return a.components.Refresh.RefreshCollection(ctx, ref)
}

func (a *API) CurrentCollectionCatalog(
	ctx context.Context,
	ref collection.CollectionRef,
) (catalog.Snapshot, error) {
	if err := a.requireStore(ctx); err != nil {
		return catalog.Snapshot{}, err
	}
	return a.components.Refresh.CurrentCatalog(ctx, ref)
}

func (a *API) InspectCollectionCatalog(
	ctx context.Context,
	ref collection.CollectionRef,
) (refresh.CatalogInspection, error) {
	if err := a.requireStore(ctx); err != nil {
		return refresh.CatalogInspection{}, err
	}
	return a.components.Refresh.InspectCollectionCatalog(ctx, ref)
}

func (a *API) CanonicalizeExpectedDocument(
	ctx context.Context,
	expected providerapi.SchemaKey,
	raw []byte,
) (providerapi.ParsedDocument, error) {
	if err := a.requireStore(ctx); err != nil {
		return providerapi.ParsedDocument{}, err
	}
	if a.components.ShareableSchemas == nil {
		return providerapi.ParsedDocument{}, basespec.ErrClosed
	}
	return a.components.ShareableSchemas.CanonicalizeExpected(
		ctx,
		expected,
		append([]byte(nil), raw...),
	)
}

func (a *API) PublishManagedArtifact(
	ctx context.Context,
	request managedartifact.PublishRequest,
) (managedartifact.PublishResult, error) {
	if err := a.requireStore(ctx); err != nil {
		return managedartifact.PublishResult{}, err
	}
	return a.components.ManagedArtifacts.Publish(ctx, request)
}

func (a *API) PublishManagedCollection(
	ctx context.Context,
	request managedartifact.PublishCollectionRequest,
) (managedartifact.PublishCollectionResult, error) {
	if err := a.requireStore(ctx); err != nil {
		return managedartifact.PublishCollectionResult{}, err
	}
	return a.components.ManagedArtifacts.PublishCollection(ctx, request)
}

func (a *API) RemoveManagedArtifact(
	ctx context.Context,
	request managedartifact.RemoveRequest,
) error {
	if err := a.requireStore(ctx); err != nil {
		return err
	}
	return a.components.ManagedArtifacts.Remove(ctx, request)
}

func (a *API) requireStore(ctx context.Context) error {
	if err := a.check(ctx); err != nil {
		return err
	}
	if a.components.Collections == nil ||
		a.components.Artifacts == nil ||
		a.components.Refresh == nil ||
		a.components.ManagedArtifacts == nil ||
		a.components.ShareableSchemas == nil {
		return fmt.Errorf(
			"%w: Artifact Store command services are unavailable",
			basespec.ErrClosed,
		)
	}
	return nil
}

func cloneAttachmentDrafts(
	values []collection.AttachmentDraft,
) []collection.AttachmentDraft {
	if values == nil {
		return nil
	}

	output := make(
		[]collection.AttachmentDraft,
		len(values),
	)
	for index, value := range values {
		output[index] = value
		output[index].Data = append(
			json.RawMessage(nil),
			value.Data...,
		)
	}
	return output
}
