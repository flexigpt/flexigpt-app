// Package text owns Text Artifact materialization for application-facing
// consumers. It resolves Text through the generic verified-resource boundary
// and does not inherit Agent, Workspace, or runtime ownership.
package text

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text/materialize"
)

type Materialization struct {
	Artifact         artifactModel.ArtifactRef `json:"artifact"`
	ArtifactRevision uint64                    `json:"artifactRevision"`
	DefinitionDigest cryptoutil.Digest         `json:"definitionDigest"`

	Name      spec.LogicalName         `json:"name"`
	Insert    declaration.InsertTarget `json:"insert"`
	MediaType string                   `json:"mediaType,omitempty"`
	Content   string                   `json:"content"`
	Locator   spec.Locator             `json:"locator"`

	BuiltIn bool `json:"builtIn"`
}

type Service struct {
	materializer *materialize.Adapter
	protection   root.ProtectionAPI
}

func New(
	resources resourceFlow.API,
	protection root.ProtectionAPI,
) (*Service, error) {
	if resources == nil || protection == nil {
		return nil, fmt.Errorf(
			"%w: Text service dependencies are incomplete",
			spec.ErrInvalid,
		)
	}

	value, err := materialize.NewAdapter(resources)
	if err != nil {
		return nil, err
	}
	return &Service{
		materializer: value,
		protection:   protection,
	}, nil
}

func (s *Service) Materialize(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (Materialization, error) {
	value, err := s.materializer.Resolve(ctx, ref)
	if err != nil {
		return Materialization{}, err
	}
	return Materialization{
		Artifact:         value.Artifact,
		ArtifactRevision: value.ArtifactRevision,
		DefinitionDigest: value.DefinitionDigest,
		Name:             spec.LogicalName(value.Name),
		Insert:           value.Insert,
		MediaType:        value.MediaType,
		Content:          value.Content,
		Locator:          value.Locator,
		BuiltIn:          s.protection.IsProtectedRoot(value.Artifact.RootID),
	}, nil
}
