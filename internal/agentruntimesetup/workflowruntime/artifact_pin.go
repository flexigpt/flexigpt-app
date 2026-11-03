package workflowruntime

import (
	"context"
	"errors"
	"fmt"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
)

// ArtifactPinResolverAdapter converts a verified current Artifact into the
// durable pin used by Workflow admission. It does not expose Artifact storage
// to sessionv3Runtime.
type ArtifactPinResolverAdapter struct {
	artifacts artifact.API
}

func NewArtifactPinResolver(
	artifacts artifact.API,
) (*ArtifactPinResolverAdapter, error) {
	if artifacts == nil {
		return nil, errors.New("workflow artifact pin resolver requires Artifact API")
	}
	return &ArtifactPinResolverAdapter{
		artifacts: artifacts,
	}, nil
}

func (r *ArtifactPinResolverAdapter) PinArtifact(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (sessionv3Runtime.ArtifactPin, error) {
	if r == nil || r.artifacts == nil {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"workflow artifact pin resolver is unavailable",
		)
	}
	if err := ref.Validate(); err != nil {
		return sessionv3Runtime.ArtifactPin{}, err
	}

	values, err := r.artifacts.GetMany(
		ctx,
		[]artifactModel.ArtifactRef{ref},
	)
	if err != nil {
		return sessionv3Runtime.ArtifactPin{}, err
	}
	if len(values) != 1 || values[0].Ref() != ref {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"workflow artifact pin resolver returned an unexpected artifact",
		)
	}

	value := values[0]
	if value.State != artifactModel.StateAvailable || !value.Enabled {
		return sessionv3Runtime.ArtifactPin{}, fmt.Errorf(
			"workflow artifact %q is not available and enabled",
			ref.ArtifactID,
		)
	}
	if value.ResolvedDefinition == nil {
		return sessionv3Runtime.ArtifactPin{}, fmt.Errorf(
			"workflow artifact %q has no resolved Definition",
			ref.ArtifactID,
		)
	}

	pin := sessionv3Runtime.ArtifactPin{
		RootID:     string(value.RootID),
		ArtifactID: string(value.ID),
		Revision:   value.Revision,
		Digest:     string(*value.ResolvedDefinition),
	}

	return pin, nil
}
