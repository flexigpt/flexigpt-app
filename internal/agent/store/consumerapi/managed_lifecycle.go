package consumerapi

import (
	"context"
	"fmt"

	agentDomain "github.com/flexigpt/flexigpt-app/internal/agent/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

type editableManagedAgent struct {
	artifact   artifact.Artifact
	address    source.ManagedPackageAddress
	generation string
}

// DeleteManagedAgent removes the managed Agent package and purges every
// Artifact emitted from its package document. It deliberately does not detach
// Collection relationships, which become unavailable until an exact Agent
// occurrence is restored.
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

	return a.purgeRemovedManagedAgentArtifacts(
		ctx,
		current.artifact,
		request.Agent,
	)
}

func (a *API) purgeRemovedManagedAgentArtifacts(
	ctx context.Context,
	current artifact.Artifact,
	rootRef artifact.ArtifactRef,
) error {
	if a == nil || a.artifacts == nil {
		return basespec.ErrClosed
	}

	records, err := a.artifacts.ListBySource(
		ctx,
		current.RootID,
		current.Binding.SourceID,
	)
	if err != nil {
		return fmt.Errorf(
			"list removed managed Agent package Artifacts: %w",
			err,
		)
	}

	packageRecords := make([]artifact.Artifact, 0)
	rootFound := false
	for _, record := range records {
		if record.Binding.Locator != current.Binding.Locator {
			continue
		}
		if record.State != artifact.StateMissing {
			return fmt.Errorf(
				"%w: removed managed Agent package Artifact %q is not missing",
				basespec.ErrConflict,
				record.ID,
			)
		}
		if record.Ref() == rootRef {
			rootFound = true
		}
		packageRecords = append(packageRecords, record)
	}

	if !rootFound {
		return fmt.Errorf(
			"%w: removed managed Agent Artifact %q was not found",
			basespec.ErrConflict,
			rootRef.ArtifactID,
		)
	}

	for _, record := range packageRecords {
		if err := a.artifacts.Purge(
			ctx,
			record.Ref(),
			record.Revision,
		); err != nil {
			return fmt.Errorf(
				"purge removed managed Agent package Artifact %q: %w",
				record.ID,
				err,
			)
		}
	}
	return nil
}

func (a *API) loadEditableManagedAgent(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) (editableManagedAgent, error) {
	return a.loadManagedAgent(ctx, ref, expectedRevision, true)
}

func (a *API) loadManagedAgent(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	requireCurrentSource bool,
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
			"%w: contained Agent declarations are not exportable managed Agents",
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
	if requireCurrentSource && !sourceValue.Enabled {
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

	generation := ""
	if requireCurrentSource {
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
		generation = inspection.State.SourceGeneration
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
		generation: generation,
	}, nil
}
