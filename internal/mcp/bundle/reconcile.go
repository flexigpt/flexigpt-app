package bundle

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/managedartifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpArtifact "github.com/flexigpt/flexigpt-app/internal/mcp/artifact"
	"github.com/flexigpt/flexigpt-app/internal/mcp/installation"
	"github.com/flexigpt/flexigpt-app/internal/mcp/schema"
)

type ReplaceDocumentRequest struct {
	Bundle                     collection.CollectionRef
	ExpectedCollectionRevision uint64
	Document                   schema.BundleDocument
	Registrations              []Registration
	AllowProtected             bool
}

func (a *API) ReplaceDocument(
	ctx context.Context,
	request ReplaceDocumentRequest,
) (Bundle, error) {
	plan, err := a.prepareDocumentReplace(ctx, request)
	if err != nil {
		return Bundle{}, err
	}
	if err := a.invalidateReplacePlan(ctx, plan); err != nil {
		return Bundle{}, err
	}

	if replaceCollectionMetadataNeeded(
		plan.bundle,
		plan.document,
		plan.collectionData,
	) {
		updated, err := a.dependencies.Collections.Update(
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
			current, err = a.pinRegisteredArtifact(
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
			current, err = a.updateRegisteredArtifact(
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

	if err := ValidateDocumentLocator(plan.bundle.DocumentLocator); err != nil {
		return Bundle{}, err
	}
	documentFile := basespec.Locator(
		path.Base(string(plan.bundle.DocumentLocator)),
	)

	if _, err := a.dependencies.ManagedArtifacts.PublishCollection(
		ctx,
		managedartifact.PublishCollectionRequest{
			Collection: plan.bundle.Collection.Ref(),
			SourceID:   plan.bundle.Source.ID,
			Package: source.ManagedPackagePublication{
				Directory: plan.bundle.PackageDirectory,
				Files: []source.ManagedPackageFile{{
					Locator: documentFile,
					Content: append([]byte(nil), plan.raw...),
				}},
			},
			Plan:           a.discoveryPlan(plan.bundle),
			RefreshPolicy:  noAutomaticAdoption{},
			AllowProtected: request.AllowProtected,
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
		resolved, err := a.dependencies.Artifacts.Get(
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
		current, err := a.dependencies.Artifacts.Get(ctx, current.Ref())
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
		if err := a.dependencies.Artifacts.Purge(
			ctx,
			current.Ref(),
			current.Revision,
		); err != nil {
			return Bundle{}, err
		}
	}

	return a.Get(ctx, plan.bundle.Collection.Ref())
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

	return a.dependencies.Artifacts.Pin(
		ctx,
		artifact.PinRequest{
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
		next, err = a.dependencies.Artifacts.SetName(
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
		next, err = a.dependencies.Artifacts.UpdateData(
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
		next, err = a.dependencies.Artifacts.SetEnabled(
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
	artifactIDs := make(map[basespec.ArtifactID]basespec.SubresourceLocator, len(values))
	for _, value := range values {
		if err := basespec.ValidateArtifactID(value.ArtifactID); err != nil {
			return nil, err
		}
		if err := basespec.ValidateSubresourceLocator(value.Subresource); err != nil {
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
		if !mcpArtifact.IsMCPKind(value.Kind) {
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
	case schema.ServerKind:
		return installation.EncodeServerData(
			installation.DefaultServerData(),
		)
	case schema.PolicyKind:
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
	if record.Kind != schema.ServerKind {
		return nil
	}
	if a.dependencies.Overlays == nil ||
		!a.dependencies.RootPolicy.IsProtectedRoot(record.RootID) {
		return nil
	}

	overlay, found, err := a.dependencies.Overlays.GetServerOverlay(
		ctx,
		record.Ref(),
	)
	if err != nil || !found {
		return err
	}
	return a.dependencies.Overlays.DeleteServerOverlay(
		ctx,
		record.Ref(),
		overlay.Revision,
	)
}

func (a *API) requireBundleMutation(
	ctx context.Context,
	rootID basespec.RootID,
	allowProtected bool,
) error {
	if err := basespec.ValidateRootID(rootID); err != nil {
		return err
	}
	if !a.dependencies.RootPolicy.IsProtectedRoot(rootID) {
		return protection.RequireMutableRoot(
			ctx,
			a.dependencies.RootPolicy,
			rootID,
		)
	}
	if !allowProtected {
		return fmt.Errorf(
			"%w: protected MCP Bundle mutation requires installer access",
			basespec.ErrProtected,
		)
	}
	return protection.RequirePrivilegedInstaller(ctx)
}

func (a *API) UpdateServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
	data installation.ServerData,
) (artifact.Artifact, error) {
	if a == nil {
		return artifact.Artifact{}, basespec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return artifact.Artifact{}, err
	}
	if expectedArtifactRevision == 0 {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: expected MCP Server Artifact revision is required",
			basespec.ErrInvalid,
		)
	}

	record, err := a.dependencies.Artifacts.Get(ctx, ref)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if a.dependencies.RootPolicy.IsProtectedRoot(record.RootID) {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: protected MCP Server installation data belongs in an overlay",
			basespec.ErrProtected,
		)
	}
	if record.Revision != expectedArtifactRevision ||
		record.Kind != schema.ServerKind ||
		record.ResolvedDefinition == nil {
		return artifact.Artifact{}, basespec.ErrConflict
	}

	definitionValue, err := definition.ReadCanonical(
		ctx,
		a.dependencies.Definitions,
		record.RootID,
		*record.ResolvedDefinition,
	)
	if err != nil {
		return artifact.Artifact{}, err
	}
	document, err := serverDocumentFromDefinition(definitionValue)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if err := installation.ValidateServerDataForDocument(
		ref,
		document,
		data,
	); err != nil {
		return artifact.Artifact{}, err
	}

	encoded, err := installation.EncodeServerData(data)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if jsonutil.Equal(record.Data, encoded) {
		return record, nil
	}
	before, err := installation.DecodeServerData(record.Data)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if err := a.dependencies.Runtime.Invalidate(ctx, ref); err != nil {
		return artifact.Artifact{}, err
	}
	updated, err := a.dependencies.Artifacts.UpdateData(
		ctx,
		ref,
		expectedArtifactRevision,
		encoded,
	)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if err := a.cleanupChangedServerInstallation(
		ctx,
		updated,
		before,
		data,
	); err != nil {
		return updated, err
	}
	return updated, nil
}

func (a *API) UpdateProtectedServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedOverlayRevision uint64,
	runtimeEnabled bool,
	data installation.ServerData,
) error {
	if a == nil {
		return basespec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return err
	}
	if !a.dependencies.RootPolicy.IsProtectedRoot(ref.RootID) {
		return fmt.Errorf(
			"%w: MCP Server is not in a protected Root",
			basespec.ErrProtected,
		)
	}
	if a.dependencies.Overlays == nil {
		return fmt.Errorf(
			"%w: protected MCP installation overlay store is unavailable",
			basespec.ErrReferenceUnresolved,
		)
	}

	record, err := a.dependencies.Artifacts.Get(ctx, ref)
	if err != nil {
		return err
	}
	if record.Kind != schema.ServerKind ||
		record.ResolvedDefinition == nil {
		return fmt.Errorf(
			"%w: Artifact is not an available MCP Server",
			basespec.ErrInvalid,
		)
	}

	definitionValue, err := definition.ReadCanonical(
		ctx,
		a.dependencies.Definitions,
		record.RootID,
		*record.ResolvedDefinition,
	)
	if err != nil {
		return err
	}
	document, err := serverDocumentFromDefinition(definitionValue)
	if err != nil {
		return err
	}
	if err := installation.ValidateServerDataForDocument(
		ref,
		document,
		data,
	); err != nil {
		return err
	}

	current, found, err := a.dependencies.Overlays.GetServerOverlay(
		ctx,
		ref,
	)
	if err != nil {
		return err
	}
	if found && current.Revision != expectedOverlayRevision {
		return basespec.ErrConflict
	}
	if !found && expectedOverlayRevision != 0 {
		return basespec.ErrConflict
	}

	before := installation.DefaultServerData()
	if found {
		before = current.ServerData
	}

	nextRevision := uint64(1)
	if found {
		nextRevision = current.Revision + 1
	}
	if err := a.dependencies.Runtime.Invalidate(ctx, ref); err != nil {
		return err
	}
	if err := a.dependencies.Overlays.PutServerOverlay(
		ctx,
		ref,
		expectedOverlayRevision,
		installation.ServerOverlay{
			SchemaVersion:  installation.SchemaVersion,
			Revision:       nextRevision,
			RuntimeEnabled: runtimeEnabled,
			ServerData:     data,
		},
	); err != nil {
		return err
	}

	beforeRaw, err := installation.EncodeServerData(before)
	if err != nil {
		return err
	}
	afterRaw, err := installation.EncodeServerData(data)
	if err != nil {
		return err
	}
	if jsonutil.Equal(beforeRaw, afterRaw) {
		return nil
	}

	if err := a.cleanupChangedServerInstallation(
		ctx,
		record,
		before,
		data,
	); err != nil {
		return fmt.Errorf(
			"MCP protected server installation cleanup remains pending: %w",
			err,
		)
	}
	return nil
}

func serverDocumentFromDefinition(
	value definition.Definition,
) (schema.ServerDocument, error) {
	body, err := mcpArtifact.ServerBodyFromDefinition(value)
	if err != nil {
		return schema.ServerDocument{}, err
	}
	return schema.ServerDocument{
		SchemaURL:      schema.ServerSchemaURL,
		Kind:           schema.ServerKind,
		SchemaID:       schema.ServerSchemaID,
		SchemaVersion:  schema.SchemaVersion,
		LogicalName:    value.LogicalName,
		LogicalVersion: value.LogicalVersion,
		DisplayName:    value.DisplayName,
		Description:    value.Description,
		Labels:         maps.Clone(value.Labels),
		MCPServer:      body.MCPServer,
		Extension:      body.Extension,
	}, nil
}
