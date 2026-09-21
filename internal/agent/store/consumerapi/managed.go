package consumerapi

import (
	"context"
	"fmt"

	agentDomain "github.com/flexigpt/flexigpt-app/internal/agent/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/collection"
)

type editableManagedAgent struct {
	artifact   artifact.Artifact
	address    source.ManagedPackageAddress
	generation string
}

func (a *API) CreateManagedAgent(
	ctx context.Context,
	request ManagedAgentCreateRequest,
) (ManagedAgentCreateResult, error) {
	if a == nil || a.collections == nil {
		return ManagedAgentCreateResult{}, basespec.ErrClosed
	}
	if request.ExpectedCollectionRevision == 0 {
		return ManagedAgentCreateResult{}, fmt.Errorf(
			"%w: expected Agent Collection revision is required",
			basespec.ErrInvalid,
		)
	}
	if err := request.Collection.Validate(); err != nil {
		return ManagedAgentCreateResult{}, err
	}

	document, err := request.Document.Declaration()
	if err != nil {
		return ManagedAgentCreateResult{}, err
	}

	_, definitionValue, err := agentDomain.ManagedAgentDocumentPayload(
		document,
	)
	if err != nil {
		return ManagedAgentCreateResult{}, err
	}

	address, err := agentDomain.ManagedPackageAddressForAgent(
		definitionValue.LogicalName,
	)
	if err != nil {
		return ManagedAgentCreateResult{}, err
	}
	locator, err := agentDomain.ManagedPackageLocatorForAgent(address)
	if err != nil {
		return ManagedAgentCreateResult{}, err
	}

	membership, err := a.collections.EnsureMemberForCollectionSource(
		ctx,
		collection.EnsureMemberForCollectionSourceRequest{
			Collection:       request.Collection,
			ExpectedRevision: request.ExpectedCollectionRevision,
			Type:             declaration.TypeAgent,
			Name:             definitionValue.LogicalName,
			Locator:          locator,
		},
	)
	if err != nil {
		return ManagedAgentCreateResult{}, err
	}

	result := ManagedAgentCreateResult{
		Collection:        membership.Collection,
		MembershipCreated: membership.Created,
	}
	published, err := a.publishManagedAgent(
		ctx,
		membership.Collection.Artifact.RootID,
		membership.Collection.Artifact.Binding.SourceID,
		address,
		document,
		"",
		false,
	)
	if err != nil {
		return result, err
	}

	record := published
	if record.Enabled != request.Enabled {
		record, err = a.artifacts.SetEnabled(
			ctx,
			record.Ref(),
			record.Revision,
			request.Enabled,
		)
		if err != nil {
			return result, err
		}
	}

	collectionView, err := a.GetAgentCollection(
		ctx,
		membership.Collection.Artifact.Ref(),
	)
	if err != nil {
		return result, err
	}

	result.Agent = record.Clone()
	result.Address = record.Address()
	result.Collection = collectionView
	return result, nil
}

func (a *API) ReplaceManagedAgent(
	ctx context.Context,
	request ManagedAgentReplaceRequest,
) (ManagedAgentReplaceResult, error) {
	if a == nil {
		return ManagedAgentReplaceResult{}, basespec.ErrClosed
	}
	if request.ExpectedRevision == 0 {
		return ManagedAgentReplaceResult{}, fmt.Errorf(
			"%w: expected Agent Artifact revision is required",
			basespec.ErrInvalid,
		)
	}

	current, err := a.loadEditableManagedAgent(
		ctx,
		request.Agent,
		request.ExpectedRevision,
	)
	if err != nil {
		return ManagedAgentReplaceResult{}, err
	}

	document, err := request.Document.Declaration()
	if err != nil {
		return ManagedAgentReplaceResult{}, err
	}

	_, definitionValue, err := agentDomain.ManagedAgentDocumentPayload(
		document,
	)
	if err != nil {
		return ManagedAgentReplaceResult{}, err
	}
	if definitionValue.LogicalName != current.artifact.LogicalName {
		return ManagedAgentReplaceResult{}, fmt.Errorf(
			"%w: managed Agent replacement cannot rename %q to %q",
			basespec.ErrInvalid,
			current.artifact.LogicalName,
			definitionValue.LogicalName,
		)
	}

	record, err := a.publishManagedAgent(
		ctx,
		current.artifact.RootID,
		current.artifact.Binding.SourceID,
		current.address,
		document,
		current.generation,
		true,
	)
	if err != nil {
		return ManagedAgentReplaceResult{}, err
	}
	if record.Ref() != current.artifact.Ref() {
		return ManagedAgentReplaceResult{}, fmt.Errorf(
			"%w: managed Agent replacement produced another Artifact",
			basespec.ErrInvalid,
		)
	}

	return ManagedAgentReplaceResult{
		Agent:   record.Clone(),
		Address: record.Address(),
	}, nil
}

func (a *API) DeleteManagedAgent(
	ctx context.Context,
	request ManagedAgentDeleteRequest,
) error {
	if a == nil {
		return basespec.ErrClosed
	}
	if request.ExpectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected Agent Artifact revision is required",
			basespec.ErrInvalid,
		)
	}

	current, err := a.loadEditableManagedAgent(
		ctx,
		request.Agent,
		request.ExpectedRevision,
	)
	if err != nil {
		return err
	}

	locator := current.artifact.Binding.Locator
	if err := a.managedArtifacts.Remove(
		ctx,
		artifact.RemoveArtifactRequest{
			RootID:                current.artifact.RootID,
			SourceID:              current.artifact.Binding.SourceID,
			Package:               current.address,
			ExpectedGeneration:    current.generation,
			ExpectedArtifact:      &request.Agent,
			PruneDiscoveryLocator: &locator,
		},
	); err != nil {
		return err
	}

	missing, err := a.artifacts.Get(ctx, request.Agent)
	if err != nil {
		return err
	}
	if missing.State != artifact.StateMissing {
		return fmt.Errorf(
			"%w: removed managed Agent Artifact is not missing",
			basespec.ErrConflict,
		)
	}

	return a.artifacts.Purge(
		ctx,
		request.Agent,
		missing.Revision,
	)
}

func (a *API) publishManagedAgent(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	address source.ManagedPackageAddress,
	document agentv1.AgentDocument,
	expectedGeneration string,
	allowPackageReplacement bool,
) (artifact.Artifact, error) {
	raw, definitionValue, err := agentDomain.ManagedAgentDocumentPayload(
		document,
	)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if definitionValue.Kind != agentDomain.AgentArtifactKind {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: managed declaration is not an Agent",
			basespec.ErrInvalid,
		)
	}

	if err := agentDomain.ValidateManagedAgentPackageAddress(address); err != nil {
		return artifact.Artifact{}, err
	}
	locator, err := agentDomain.ManagedPackageLocatorForAgent(address)
	if err != nil {
		return artifact.Artifact{}, err
	}

	decoderID, err := documentTopology.DefaultDocumentDecoderID(
		documentTopology.DocumentUseManagedAgent,
	)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if _, err := a.collections.EnsureManagedDeclarationDiscovery(
		ctx,
		rootID,
		sourceID,
		locator,
		decoderID,
	); err != nil {
		return artifact.Artifact{}, err
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		artifact.PublishArtifactRequest{
			RootID: rootID,
			Binding: artifact.SourceBinding{
				SourceID: sourceID,
				Locator:  locator,
			},
			ExpectedKind:            agentDomain.AgentArtifactKind,
			ExpectedLogicalName:     definitionValue.LogicalName,
			ExpectedDefinition:      definitionValue.Digest,
			AllowPackageReplacement: allowPackageReplacement,
			Package: source.ManagedPackagePublication{
				Address:            address,
				ExpectedGeneration: expectedGeneration,
				Files: []source.ManagedPackageFile{{
					Locator: agentDomain.ManagedAgentDocumentFile(),
					Content: raw,
				}},
			},
		},
	)
	if err != nil {
		return artifact.Artifact{}, err
	}
	return published.Artifact.Clone(), nil
}

func (a *API) loadEditableManagedAgent(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) (editableManagedAgent, error) {
	if a == nil {
		return editableManagedAgent{}, basespec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return editableManagedAgent{}, err
	}

	record, err := a.GetAgent(ctx, ref)
	if err != nil {
		return editableManagedAgent{}, err
	}
	if expectedRevision != 0 && record.Revision != expectedRevision {
		return editableManagedAgent{}, basespec.ErrConflict
	}
	if record.State != artifact.StateAvailable {
		return editableManagedAgent{}, fmt.Errorf(
			"%w: Agent Artifact %q is unavailable",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if record.LogicalVersion != "" {
		return editableManagedAgent{}, fmt.Errorf(
			"%w: Agent Artifact has an unexpected logical version",
			basespec.ErrDigestMismatch,
		)
	}
	if record.Binding.SubresourceLocator != "" {
		return editableManagedAgent{}, fmt.Errorf(
			"%w: contained Agent declarations are not editable managed Agents",
			basespec.ErrUnsupported,
		)
	}
	if err := a.requireMutable(ctx, record.RootID, false); err != nil {
		return editableManagedAgent{}, err
	}

	sourceValue, err := a.sources.Get(
		ctx,
		record.RootID,
		record.Binding.SourceID,
	)
	if err != nil {
		return editableManagedAgent{}, err
	}
	if sourceValue.Kind != source.SourceKindManagedDirectory ||
		sourceValue.StorageKey != agentDomain.AgentManagedSourceStorageKey {
		return editableManagedAgent{}, fmt.Errorf(
			"%w: Agent is not backed by the managed Agent Source",
			basespec.ErrUnsupported,
		)
	}
	if !sourceValue.Enabled {
		return editableManagedAgent{}, fmt.Errorf(
			"%w: managed Agent Source is disabled",
			basespec.ErrReferenceUnresolved,
		)
	}

	address, err := agentDomain.ManagedPackageAddressFromAgentLocator(
		record.Binding.Locator,
	)
	if err != nil {
		return editableManagedAgent{}, err
	}
	if address.Name != record.LogicalName {
		return editableManagedAgent{}, fmt.Errorf(
			"%w: managed Agent package name differs from Artifact identity",
			basespec.ErrInvalid,
		)
	}

	inspection, err := a.discovery.InspectSource(
		ctx,
		record.RootID,
		record.Binding.SourceID,
	)
	if err != nil {
		return editableManagedAgent{}, err
	}
	if !inspection.IsCurrent() {
		return editableManagedAgent{}, fmt.Errorf(
			"%w: managed Agent Source requires refresh",
			basespec.ErrRefreshRequired,
		)
	}

	definitionValue, err := a.artifacts.GetDefinition(ctx, ref)
	if err != nil {
		return editableManagedAgent{}, err
	}
	document, err := agentv1.DecodeAgentJSON(definitionValue.Body)
	if err != nil {
		return editableManagedAgent{}, err
	}
	if document.Name != string(record.LogicalName) ||
		document.Locator != nil {
		return editableManagedAgent{}, fmt.Errorf(
			"%w: managed Agent declaration is not a concrete matching Agent",
			basespec.ErrInvalid,
		)
	}

	return editableManagedAgent{
		artifact:   record.Clone(),
		address:    address,
		generation: inspection.State.SourceGeneration,
	}, nil
}
