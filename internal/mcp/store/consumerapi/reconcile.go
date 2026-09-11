package consumerapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcpbundlev1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcpserverv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpDomainBundle "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/bundle"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
	mcpOverlay "github.com/flexigpt/flexigpt-app/internal/mcp/store/overlay"
)

type ReplaceDocumentRequest struct {
	Bundle                     collection.CollectionRef
	ExpectedCollectionRevision uint64
	Document                   json.RawMessage
	Registrations              []Registration
	AllowProtected             bool
}

// ReplaceDocument publishes the canonical document-only package used by
// user-managed MCP Bundles.
func (a *API) ReplaceDocument(
	ctx context.Context,
	request ReplaceDocumentRequest,
) (Bundle, error) {
	commit, err := a.PrepareReplaceDocument(ctx, request)
	if err != nil {
		return Bundle{}, err
	}
	return commit(ctx)
}

// PrepareReplaceDocument validates and materializes all request-derived
// document state before runtime invalidation. The returned commit function
// still relies on Artifact Store compare-and-swap checks for concurrency.
func (a *API) PrepareReplaceDocument(
	ctx context.Context,
	request ReplaceDocumentRequest,
) (BundleMutationCommit, error) {
	if a == nil {
		return nil, basespec.ErrClosed
	}
	document, parsed, err := a.canonicalizeBundleBytes(
		ctx,
		request.Document,
	)
	if err != nil {
		return nil, err
	}
	return a.prepareCanonicalDocumentReplace(
		ctx,
		request,
		document,
		parsed.Raw,
		nil,
	)
}

// replaceCanonicalDocument is shared by user-managed Bundle updates and
// protected hydration. The caller supplies a typed Bundle document and its
// corresponding canonical bytes after expected-schema canonicalization.
func (a *API) replaceCanonicalDocument(
	ctx context.Context,
	request ReplaceDocumentRequest,
	document mcpDomainBundle.BundleDocument,
	raw json.RawMessage,
	suppliedFiles []source.ManagedPackageFile,
) (Bundle, error) {
	commit, err := a.prepareCanonicalDocumentReplace(
		ctx,
		request,
		document,
		raw,
		suppliedFiles,
	)
	if err != nil {
		return Bundle{}, err
	}
	return commit(ctx)
}

func (a *API) prepareCanonicalDocumentReplace(
	ctx context.Context,
	request ReplaceDocumentRequest,
	document mcpDomainBundle.BundleDocument,
	raw json.RawMessage,
	suppliedFiles []source.ManagedPackageFile,
) (BundleMutationCommit, error) {
	plan, err := a.prepareDocumentReplace(
		ctx,
		request,
		document,
		raw,
	)
	if err != nil {
		return nil, err
	}
	packageAddress, err := mcpDomainBundle.PackageAddressForBundle(
		plan.document.LogicalName,
		plan.document.LogicalVersion,
	)
	if err != nil {
		return nil, err
	}
	if packageAddress != plan.bundle.PackageAddress {
		return nil, fmt.Errorf(
			"%w: MCP Bundle document identity would move package content",
			basespec.ErrConflict,
		)
	}
	files, err := documentPackageFiles(
		packageAddress,
		plan.raw,
		suppliedFiles,
	)
	if err != nil {
		return nil, err
	}

	return func(commitCtx context.Context) (Bundle, error) {
		return a.commitDocumentReplace(
			commitCtx,
			plan,
			packageAddress,
			files,
			request.AllowProtected,
		)
	}, nil
}

func (a *API) commitDocumentReplace(
	ctx context.Context,
	plan documentReplacePlan,
	packageAddress source.ManagedPackageAddress,
	files []source.ManagedPackageFile,
	allowProtected bool,
) (Bundle, error) {
	if replaceCollectionMetadataNeeded(
		plan.bundle,
		plan.document,
		plan.collectionData,
	) {
		updated, err := a.collections.Update(
			ctx,
			plan.bundle.Collection.Ref(),
			collection.Update{
				ExpectedRevision: plan.bundle.Collection.Revision,
				DisplayName:      displayName(plan.document),
				Description:      plan.document.Description,
				Enabled:          plan.bundle.Collection.Enabled,
				Data:             plan.collectionData,
			},
		)
		if err != nil {
			return Bundle{}, err
		}
		plan.bundle, err = a.Get(ctx, updated.Ref())
		if err != nil {
			return Bundle{}, err
		}
	}

	for _, subresource := range plan.orderedSubresources {
		expectedDefinition := plan.definitions[subresource]
		registration := plan.registrations[subresource]
		current, found := plan.existingBySubresource[subresource]
		if !found {
			data := plan.dataBySubresource[subresource]
			_, err := a.pinRegisteredArtifact(
				ctx,
				plan.bundle,
				registration,
				expectedDefinition,
				data,
			)
			if err != nil {
				return Bundle{}, err
			}
		} else {
			data := plan.dataBySubresource[subresource]
			_, err := a.updateRegisteredArtifact(
				ctx,
				plan.bundle,
				current,
				registration,
				expectedDefinition,
				data,
			)
			if err != nil {
				return Bundle{}, err
			}
		}

		plan.existingBySubresource[subresource] = current
	}

	if _, err := a.managedArtifacts.PublishCollection(
		ctx,
		collection.PublishCollectionRequest{
			Collection: plan.bundle.Collection.Ref(),
			SourceID:   plan.bundle.Source.ID,
			Package: source.ManagedPackagePublication{
				Address: packageAddress,
				Files:   files,
			},
			AllowProtected: allowProtected,
			ForceRefresh:   true,
		},
	); err != nil {
		return Bundle{}, fmt.Errorf(
			"MCP document publication remains pending; retry with current revisions: %w",
			err,
		)
	}

	for _, subresource := range plan.orderedSubresources {
		registration := plan.registrations[subresource]
		expectedDefinition := plan.definitions[subresource]
		resolved, err := a.artifacts.Get(
			ctx,
			artifact.ArtifactRef{
				RootID:     plan.bundle.Collection.RootID,
				ArtifactID: registration.ArtifactID,
			},
		)
		if err != nil {
			return Bundle{}, err
		}
		if resolved.State != artifact.StateAvailable ||
			resolved.ResolvedDefinition == nil ||
			*resolved.ResolvedDefinition != expectedDefinition.Digest {
			return Bundle{}, fmt.Errorf(
				"%w: MCP Artifact %q did not resolve to the published Definition",
				basespec.ErrReferenceUnresolved,
				registration.ArtifactID,
			)
		}
	}

	for _, current := range plan.removed {
		current, err := a.artifacts.Get(ctx, current.Ref())
		if err != nil {
			return Bundle{}, err
		}
		if current.State != artifact.StateMissing {
			return Bundle{}, fmt.Errorf(
				"%w: removed MCP Artifact %q did not become missing",
				basespec.ErrConflict,
				current.ID,
			)
		}

		if err := a.cleanupRemovedServerInstallation(ctx, current); err != nil {
			return Bundle{}, err
		}
		if err := a.deleteProtectedOverlayIfPresent(ctx, current); err != nil {
			return Bundle{}, err
		}
		if err := a.artifacts.Purge(
			ctx,
			current.Ref(),
			current.Revision,
		); err != nil {
			return Bundle{}, err
		}
	}

	return a.Get(ctx, plan.bundle.Collection.Ref())
}

func documentPackageFiles(
	address source.ManagedPackageAddress,
	canonicalDocument json.RawMessage,
	supplied []source.ManagedPackageFile,
) ([]source.ManagedPackageFile, error) {
	if len(supplied) == 0 {
		supplied = []source.ManagedPackageFile{{
			Locator: artifactbuiltin.MCPBundleDocumentFileName,
			Content: append([]byte(nil), canonicalDocument...),
		}}
	}

	files := make([]source.ManagedPackageFile, len(supplied))
	foundDocument := false
	for index, file := range supplied {
		files[index] = source.ManagedPackageFile{
			Locator: file.Locator,
			Content: append([]byte(nil), file.Content...),
		}
		if file.Locator == artifactbuiltin.MCPBundleDocumentFileName {
			files[index].Content = append(
				[]byte(nil),
				canonicalDocument...,
			)
			foundDocument = true
		}
	}
	if !foundDocument {
		return nil, fmt.Errorf(
			"%w: MCP package does not contain canonical document %q",
			basespec.ErrInvalid,
			artifactbuiltin.MCPBundleDocumentFileName,
		)
	}

	publication, err := source.NormalizeManagedPackagePublication(
		source.ManagedPackagePublication{
			Address: address,
			Files:   files,
		},
	)
	if err != nil {
		return nil, err
	}
	return publication.Files, nil
}

func (a *API) pinRegisteredArtifact(
	ctx context.Context,
	bundle Bundle,
	registration Registration,
	expected definition.Definition,
	data json.RawMessage,
) (artifact.Artifact, error) {
	name := expected.DisplayName
	if name == "" {
		name = string(expected.LogicalName)
	}

	return a.artifacts.Pin(
		ctx,
		catalog.PinRequest{
			ArtifactID:                 registration.ArtifactID,
			Collection:                 bundle.Collection.Ref(),
			ExpectedCollectionRevision: bundle.Collection.Revision,
			Binding: artifact.SourceBinding{
				SourceID:           bundle.Source.ID,
				Locator:            bundle.DocumentLocator,
				SubresourceLocator: registration.Subresource,
				ExpectedKind:       registration.Kind,
			},
			Name:    name,
			Enabled: registration.Enabled,
			Data:    data,
		},
	)
}

func (a *API) updateRegisteredArtifact(
	ctx context.Context,
	bundle Bundle,
	current artifact.Artifact,
	registration Registration,
	expected definition.Definition,
	data json.RawMessage,
) (artifact.Artifact, error) {
	expectedBinding := artifact.SourceBinding{
		SourceID:           bundle.Source.ID,
		Locator:            bundle.DocumentLocator,
		SubresourceLocator: registration.Subresource,
		ExpectedKind:       registration.Kind,
	}
	if current.ID != registration.ArtifactID ||
		current.RootID != bundle.Collection.RootID ||
		current.CollectionID != bundle.Collection.ID ||
		current.Adoption != artifact.AdoptionPinned ||
		current.Kind != registration.Kind ||
		current.Binding != expectedBinding {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: MCP Artifact registration %q conflicts with existing local identity",
			basespec.ErrConflict,
			registration.ArtifactID,
		)
	}

	name := expected.DisplayName
	if name == "" {
		name = string(expected.LogicalName)
	}

	next := current
	var err error
	if next.Name != name {
		next, err = a.artifacts.SetName(
			ctx,
			next.Ref(),
			next.Revision,
			name,
		)
		if err != nil {
			return artifact.Artifact{}, err
		}
	}
	if !jsonutil.Equal(next.Data, data) {
		next, err = a.artifacts.UpdateData(
			ctx,
			next.Ref(),
			next.Revision,
			data,
		)
		if err != nil {
			return artifact.Artifact{}, err
		}
	}
	if next.Enabled != registration.Enabled {
		next, err = a.artifacts.SetEnabled(
			ctx,
			next.Ref(),
			next.Revision,
			registration.Enabled,
		)
		if err != nil {
			return artifact.Artifact{}, err
		}
	}
	return next, nil
}

func registrationMap(
	values []Registration,
	expected map[basespec.SubresourceLocator]definition.Definition,
) (map[basespec.SubresourceLocator]Registration, error) {
	if len(values) != len(expected) {
		return nil, fmt.Errorf(
			"%w: MCP registrations must cover every server and policy subresource",
			basespec.ErrInvalid,
		)
	}

	output := make(
		map[basespec.SubresourceLocator]Registration,
		len(values),
	)
	artifactIDs := make(map[artifact.ArtifactID]basespec.SubresourceLocator, len(values))
	for _, value := range values {
		if err := value.ArtifactID.Validate(); err != nil {
			return nil, err
		}
		if err := value.Subresource.Validate(); err != nil {
			return nil, err
		}
		if previous, duplicate := artifactIDs[value.ArtifactID]; duplicate {
			return nil, fmt.Errorf(
				"%w: MCP Artifact %q is registered for both %q and %q",
				basespec.ErrInvalid,
				value.ArtifactID,
				previous,
				value.Subresource,
			)
		}
		artifactIDs[value.ArtifactID] = value.Subresource
		expectedDefinition, found := expected[value.Subresource]
		if !found || expectedDefinition.Kind != value.Kind {
			return nil, fmt.Errorf(
				"%w: invalid MCP Artifact registration for subresource %q",
				basespec.ErrInvalid,
				value.Subresource,
			)
		}
		if _, duplicate := output[value.Subresource]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate MCP Artifact registration for subresource %q",
				basespec.ErrInvalid,
				value.Subresource,
			)
		}
		output[value.Subresource] = value
	}
	return output, nil
}

func mcpArtifactsBySubresource(
	bundle Bundle,
	values []artifact.Artifact,
) (map[basespec.SubresourceLocator]artifact.Artifact, error) {
	output := make(map[basespec.SubresourceLocator]artifact.Artifact)
	for _, value := range values {
		if !isMCPKind(value.Kind) {
			return nil, fmt.Errorf(
				"%w: non-MCP Artifact %q exists in MCP Bundle %q",
				basespec.ErrConflict,
				value.ID,
				bundle.Collection.ID,
			)
		}
		if value.Binding.SourceID != bundle.Source.ID ||
			value.Binding.Locator != bundle.DocumentLocator {
			return nil, fmt.Errorf(
				"%w: MCP Artifact %q has an unsupported source binding",
				basespec.ErrConflict,
				value.ID,
			)
		}
		if _, duplicate := output[value.Binding.SubresourceLocator]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate MCP Artifact subresource %q",
				basespec.ErrConflict,
				value.Binding.SubresourceLocator,
			)
		}
		output[value.Binding.SubresourceLocator] = value
	}
	return output, nil
}

func registrationData(value Registration) (json.RawMessage, error) {
	if value.Data != nil {
		canonical, err := jsonutil.CanonicalizeObject(
			value.Data,
			basespec.MaxLocalDataBytes,
		)
		if err != nil {
			return nil, err
		}
		return json.RawMessage(canonical), nil
	}

	switch value.Kind {
	case mcpserverv1.MCPServerKind:
		return mcpDomainServer.EncodeServerData(
			mcpDomainServer.DefaultServerData(),
		)
	case mcppolicyv1.MCPPolicyKind:
		return json.RawMessage(jsonutil.EmptyObject), nil
	default:
		return nil, fmt.Errorf(
			"%w: unsupported MCP registration kind %q",
			basespec.ErrInvalid,
			value.Kind,
		)
	}
}

func (a *API) deleteProtectedOverlayIfPresent(
	ctx context.Context,
	record artifact.Artifact,
) error {
	if record.Kind != mcpserverv1.MCPServerKind {
		return nil
	}
	if a.overlays == nil ||
		!a.protection.IsProtectedRoot(record.RootID) {
		return nil
	}

	ovr, found, err := a.overlays.GetServerOverlay(
		ctx,
		record.Ref(),
	)
	if err != nil || !found {
		return err
	}
	return a.overlays.DeleteServerOverlay(
		ctx,
		record.Ref(),
		ovr.Revision,
	)
}

func (a *API) requireBundleMutation(
	ctx context.Context,
	rootID root.RootID,
	allowProtected bool,
) error {
	if err := rootID.Validate(); err != nil {
		return err
	}
	if !a.protection.IsProtectedRoot(rootID) {
		return nil
	}
	if !allowProtected {
		return fmt.Errorf(
			"%w: protected MCP Bundle mutation requires installer access",
			basespec.ErrProtected,
		)
	}
	return a.protection.RequirePrivilegedInstaller(ctx)
}

func (a *API) UpdateServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
	data mcpDomainServer.ServerData,
) (artifact.Artifact, error) {
	commit, err := a.PrepareUpdateServerInstallation(
		ctx,
		ref,
		expectedArtifactRevision,
		data,
	)
	if err != nil {
		return artifact.Artifact{}, err
	}
	return commit(ctx)
}

func (a *API) PrepareUpdateServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
	data mcpDomainServer.ServerData,
) (ArtifactMutationCommit, error) {
	if a == nil {
		return nil, basespec.ErrClosed
	}
	if expectedArtifactRevision == 0 {
		return nil, fmt.Errorf(
			"%w: expected MCP Server Artifact revision is required",
			basespec.ErrInvalid,
		)
	}

	resolvedResource, err := a.resources.ResolveArtifact(
		ctx,
		ref,
		resource.ResolveOptions{},
	)
	if err != nil {
		return nil, err
	}
	record := resolvedResource.Artifact
	if a.protection.IsProtectedRoot(record.RootID) {
		return nil, fmt.Errorf(
			"%w: protected MCP Server installation data belongs in an overlay",
			basespec.ErrProtected,
		)
	}
	if record.Revision != expectedArtifactRevision ||
		record.Kind != mcpserverv1.MCPServerKind {
		return nil, basespec.ErrConflict
	}
	document, err := mcpDomainServer.ServerDocumentFromDefinition(
		resolvedResource.Definition,
	)
	if err != nil {
		return nil, err
	}
	if err := data.ValidateFor(
		ref,
		document,
	); err != nil {
		return nil, err
	}

	encoded, err := mcpDomainServer.EncodeServerData(data)
	if err != nil {
		return nil, err
	}
	return func(commitCtx context.Context) (artifact.Artifact, error) {
		if jsonutil.Equal(record.Data, encoded) {
			return record, a.cleanupChangedServerInstallation(
				commitCtx,
				record,
				document,
				data,
			)
		}
		updated, err := a.artifacts.UpdateData(
			commitCtx,
			ref,
			expectedArtifactRevision,
			encoded,
		)
		if err != nil {
			return artifact.Artifact{}, err
		}
		if err := a.cleanupChangedServerInstallation(
			commitCtx,
			updated,
			document,
			data,
		); err != nil {
			return updated, err
		}
		return updated, nil
	}, nil
}

func (a *API) UpdateProtectedServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedOverlayRevision uint64,
	runtimeEnabled bool,
	data mcpDomainServer.ServerData,
) error {
	commit, err := a.PrepareUpdateProtectedServerInstallation(
		ctx,
		ref,
		expectedOverlayRevision,
		runtimeEnabled,
		data,
	)
	if err != nil {
		return err
	}
	return commit(ctx)
}

func (a *API) PrepareUpdateProtectedServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedOverlayRevision uint64,
	runtimeEnabled bool,
	data mcpDomainServer.ServerData,
) (MutationCommit, error) {
	if a == nil {
		return nil, basespec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return nil, err
	}

	if !a.protection.IsProtectedRoot(ref.RootID) {
		return nil, fmt.Errorf(
			"%w: MCP Server is not in a protected Root",
			basespec.ErrProtected,
		)
	}
	if a.overlays == nil {
		return nil, fmt.Errorf(
			"%w: protected MCP installation overlay store is unavailable",
			basespec.ErrReferenceUnresolved,
		)
	}

	resolvedResource, err := a.resources.ResolveArtifact(
		ctx,
		ref,
		resource.ResolveOptions{},
	)
	if err != nil {
		return nil, err
	}
	record := resolvedResource.Artifact
	if record.Kind != mcpserverv1.MCPServerKind ||
		resolvedResource.Collection.Kind != mcpbundlev1.MCPBundleKind {
		return nil, fmt.Errorf(
			"%w: Artifact is not an available MCP Server",
			basespec.ErrInvalid,
		)
	}
	document, err := mcpDomainServer.ServerDocumentFromDefinition(
		resolvedResource.Definition,
	)
	if err != nil {
		return nil, err
	}
	if err := data.ValidateFor(
		ref,
		document,
	); err != nil {
		return nil, err
	}

	current, found, err := a.overlays.GetServerOverlay(
		ctx,
		ref,
	)
	if err != nil {
		return nil, err
	}
	if found && current.Revision != expectedOverlayRevision {
		return nil, basespec.ErrConflict
	}
	if !found && expectedOverlayRevision != 0 {
		return nil, basespec.ErrConflict
	}

	nextRevision := uint64(1)
	if found {
		nextRevision = current.Revision + 1
	}
	next := mcpOverlay.ServerOverlay{
		SchemaVersion:  artifactbuiltin.MCPSchemaVersion,
		Revision:       nextRevision,
		RuntimeEnabled: runtimeEnabled,
		ServerData:     data,
	}
	return func(commitCtx context.Context) error {
		if err := a.overlays.PutServerOverlay(
			commitCtx,
			ref,
			expectedOverlayRevision,
			next,
		); err != nil {
			return err
		}
		if err := a.cleanupChangedServerInstallation(
			commitCtx,
			record,
			document,
			data,
		); err != nil {
			return fmt.Errorf(
				"MCP protected server installation cleanup remains pending: %w",
				err,
			)
		}
		return nil
	}, nil
}
