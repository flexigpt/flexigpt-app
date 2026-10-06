package agent

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/managedfs"
	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	managepackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	agentv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/contract/v1"
	agentDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/domain"
)

type editableManagedAgent struct {
	artifact   artifactModel.Artifact
	address    managedpackageModel.ManagedPackageAddress
	generation string
}

// DeleteManagedAgent removes the managed Agent package and purges every
// Artifact emitted from its package document. It deliberately does not detach
// Plugin relationships, which become unavailable until an exact Agent
// occurrence is restored.
func (a *Service) DeleteManagedAgent(
	ctx context.Context,
	request ManagedAgentDeleteRequest,
) error {
	if a == nil {
		return spec.ErrClosed
	}
	if request.ExpectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected Agent Artifact revision is required",
			spec.ErrInvalid,
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
		managepackageModel.RemoveRequest{
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

func (a *Service) purgeRemovedManagedAgentArtifacts(
	ctx context.Context,
	current artifactModel.Artifact,
	rootRef artifactModel.ArtifactRef,
) error {
	if a == nil || a.artifacts == nil {
		return spec.ErrClosed
	}

	records, err := a.cat.ListBySource(
		ctx,
		current.RootID,
		current.Binding.SourceID,
		catalogModel.ListOptions{},
	)
	if err != nil {
		return fmt.Errorf(
			"list removed managed Agent package Artifacts: %w",
			err,
		)
	}

	packageRecords := make([]catalogModel.Entry, 0)
	rootFound := false
	for _, record := range records {
		if record.Binding.Locator != current.Binding.Locator {
			continue
		}
		if record.State != artifactModel.StateMissing {
			return fmt.Errorf(
				"%w: removed managed Agent package Artifact %q is not missing",
				spec.ErrConflict,
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
			spec.ErrConflict,
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

func (a *Service) loadEditableManagedAgent(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) (editableManagedAgent, error) {
	return a.loadManagedAgent(ctx, ref, expectedRevision, true)
}

func (a *Service) loadManagedAgent(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	requireCurrentSource bool,
) (editableManagedAgent, error) {
	if a == nil {
		return editableManagedAgent{}, spec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return editableManagedAgent{}, err
	}

	record, err := a.getAgentRecord(ctx, ref)
	if err != nil {
		return editableManagedAgent{}, err
	}
	if expectedRevision != 0 && record.Revision != expectedRevision {
		return editableManagedAgent{}, spec.ErrConflict
	}
	if record.State != artifactModel.StateAvailable {
		return editableManagedAgent{}, fmt.Errorf(
			"%w: Agent Artifact %q is unavailable",
			spec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if record.LogicalVersion != "" {
		return editableManagedAgent{}, fmt.Errorf(
			"%w: Agent Artifact has an unexpected logical version",
			spec.ErrDigestMismatch,
		)
	}
	if record.Binding.SubresourceLocator != "" {
		return editableManagedAgent{}, fmt.Errorf(
			"%w: contained Agent declarations are not exportable managed Agents",
			spec.ErrUnsupported,
		)
	}
	if err := a.requireMutable(ctx, record.RootID); err != nil {
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
	if sourceValue.Kind != managedfs.Kind ||
		sourceValue.StorageKey != agentDomain.AgentManagedSourceStorageKey {
		return editableManagedAgent{}, fmt.Errorf(
			"%w: Agent is not backed by the managed Agent Source",
			spec.ErrUnsupported,
		)
	}
	if requireCurrentSource && !sourceValue.Enabled {
		return editableManagedAgent{}, fmt.Errorf(
			"%w: managed Agent Source is disabled",
			spec.ErrReferenceUnresolved,
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
			spec.ErrInvalid,
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
				spec.ErrRefreshRequired,
			)
		}
		generation = inspection.State.SourceGeneration
	}

	definitionValue, err := a.artifacts.GetDefinition(ctx, ref)
	if err != nil {
		return editableManagedAgent{}, err
	}
	document, err := agentv1.DecodeAdmittedAgentJSON(definitionValue.Body)
	if err != nil {
		return editableManagedAgent{}, err
	}
	if document.Name != string(record.LogicalName) ||
		document.Locator != nil {
		return editableManagedAgent{}, fmt.Errorf(
			"%w: managed Agent declaration is not a concrete matching Agent",
			spec.ErrInvalid,
		)
	}

	return editableManagedAgent{
		artifact:   record.Clone(),
		address:    address,
		generation: generation,
	}, nil
}
