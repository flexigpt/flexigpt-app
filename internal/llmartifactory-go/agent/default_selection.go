package agent

import (
	"context"
	"errors"
	"fmt"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	agentDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/domain"
)

// SelectDefaultAgent resolves the effective compulsory default Agent.
//
// An explicit enabled Agent is preferred. Otherwise it selects the named
// built-in fallback Agent, then a deterministic enabled built-in Agent. A nil
// result means that no enabled built-in Agent is available.
func (a *Service) SelectDefaultAgent(
	ctx context.Context,
	preferred *artifactModel.ArtifactRef,
	baseName storeSpec.LogicalName,
) (*artifactModel.ArtifactRef, error) {
	if preferred != nil {
		record, err := a.artifacts.Get(ctx, *preferred)
		if err != nil {
			if !errors.Is(err, storeSpec.ErrArtifactNotFound) &&
				!errors.Is(err, storeSpec.ErrRootNotFound) &&
				!errors.Is(err, storeSpec.ErrRetired) {
				return nil, err
			}
		} else if record.Kind == agentDomain.AgentArtifactKind &&
			record.State == artifactModel.StateAvailable &&
			record.Enabled {
			ref := record.Ref()
			return &ref, nil
		}
	}

	enabled := true
	options := catalogModel.ListOptions{
		Kind:    agentDomain.AgentArtifactKind,
		Enabled: &enabled,
	}
	entries, err := a.cat.FindByIdentity(
		ctx,
		a.support.BuiltinRoot,
		agentDomain.AgentArtifactKind,
		baseName,
		options,
	)
	if err != nil {
		return nil, err
	}
	if ref := firstAvailableAgentRef(entries); ref != nil {
		return ref, nil
	}

	entries, err = a.cat.ListByRoot(
		ctx,
		a.support.BuiltinRoot,
		options,
	)
	if err != nil {
		return nil, err
	}
	return firstAvailableAgentRef(entries), nil
}

// RequireSettableDefaultAgent verifies the current state required for an
// explicit Agent default selection. It does not resolve Agent capabilities.
func (a *Service) RequireSettableDefaultAgent(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) error {
	if err := ref.Validate(); err != nil {
		return err
	}

	record, err := a.getAgentRecord(ctx, ref)
	if err != nil {
		return err
	}
	if record.State != artifactModel.StateAvailable || !record.Enabled {
		return fmt.Errorf(
			"%w: Agent %q must be enabled before it can be selected as default",
			storeSpec.ErrReferenceUnresolved,
			record.LogicalName,
		)
	}
	return nil
}

func firstAvailableAgentRef(
	entries []catalogModel.Entry,
) *artifactModel.ArtifactRef {
	var first *catalogModel.Entry
	for index := range entries {
		entry := &entries[index]
		if entry.Kind != agentDomain.AgentArtifactKind ||
			entry.State != artifactModel.StateAvailable ||
			!entry.Enabled {
			continue
		}
		if first == nil ||
			entry.LogicalName < first.LogicalName ||
			(entry.LogicalName == first.LogicalName &&
				entry.ID < first.ID) {
			first = entry
		}
	}
	if first == nil {
		return nil
	}

	ref := first.Ref()
	return &ref
}
