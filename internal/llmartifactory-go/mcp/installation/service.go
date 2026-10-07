// Package installation owns MCP installation-local state. It resolves server
// declarations, persists mutable or protected installation data, and performs
// post-write deterministic secret cleanup without owning MCP runtime process
// lifecycle or application transport.
package installation

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	resourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain"
	serverMCPDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain/server"
	mcpOverlay "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/overlay"
)

type Dependencies struct {
	Artifacts     artifact.API
	Resources     resourceFlow.API
	Protection    root.ProtectionAPI
	Overlays      mcpOverlay.OverlayRepository
	Declarations  *composition.Resolver
	SecretCleaner serverMCPDomain.SecretCleaner
}

type Service struct {
	artifacts     artifact.API
	resources     resourceFlow.API
	protection    root.ProtectionAPI
	overlays      mcpOverlay.OverlayRepository
	declarations  *composition.Resolver
	secretCleaner serverMCPDomain.SecretCleaner
}

type Material struct {
	Resource resourceModel.ResolvedArtifact
	Document serverMCPDomain.ServerDocument

	Installation              serverMCPDomain.ServerData
	InstallationRevision      uint64
	InstallationWriteRevision uint64
	BuiltIn                   bool
}

type SecretInput struct {
	Name        string
	Label       string
	Description string
	Required    bool
	Kind        serverMCPDomain.InputKind
	Configured  bool
}

func New(dependencies Dependencies) (*Service, error) {
	if dependencies.Artifacts == nil ||
		dependencies.Resources == nil ||
		dependencies.Protection == nil ||
		dependencies.Overlays == nil ||
		dependencies.Declarations == nil ||
		dependencies.SecretCleaner == nil {
		return nil, fmt.Errorf(
			"%w: MCP installation dependencies are incomplete",
			spec.ErrInvalid,
		)
	}
	return &Service{
		artifacts:     dependencies.Artifacts,
		resources:     dependencies.Resources,
		protection:    dependencies.Protection,
		overlays:      dependencies.Overlays,
		declarations:  dependencies.Declarations,
		secretCleaner: dependencies.SecretCleaner,
	}, nil
}

func (s *Service) Resolve(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (Material, error) {
	if err := ref.Validate(); err != nil {
		return Material{}, err
	}

	terminal, err := s.declarations.ResolveTerminalArtifact(ctx, ref)
	if err != nil {
		return Material{}, err
	}
	resolved, err := s.resources.ResolveArtifact(
		ctx,
		terminal,
		resourceModel.ResolveOptions{},
	)
	if err != nil {
		return Material{}, err
	}
	if resolved.Artifact.Kind != mcpDomain.MCPArtifactKind {
		return Material{}, fmt.Errorf(
			"%w: Artifact is not an MCP Server",
			spec.ErrReferenceUnresolved,
		)
	}

	document, err := serverMCPDomain.ServerDocumentFromDefinition(
		resolved.Definition,
	)
	if err != nil {
		return Material{}, err
	}
	data, effectiveRevision, writeRevision, builtIn, err := s.effective(
		ctx,
		resolved.Artifact,
		document,
	)
	if err != nil {
		return Material{}, err
	}

	return Material{
		Resource:                  resolved.Clone(),
		Document:                  document,
		Installation:              data,
		InstallationRevision:      effectiveRevision,
		InstallationWriteRevision: writeRevision,
		BuiltIn:                   builtIn,
	}, nil
}

func (s *Service) Save(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	data serverMCPDomain.ServerData,
) (artifactModel.Artifact, error) {
	if expectedRevision == 0 {
		return artifactModel.Artifact{}, fmt.Errorf(
			"%w: expected MCP installation revision is required",
			spec.ErrInvalid,
		)
	}

	material, err := s.Resolve(ctx, ref)
	if err != nil {
		return artifactModel.Artifact{}, err
	}
	if err := data.ValidateFor(
		material.Resource.Artifact.Ref(),
		material.Document,
	); err != nil {
		return artifactModel.Artifact{}, err
	}

	if material.BuiltIn {
		return s.saveProtected(ctx, material, expectedRevision, data)
	}
	return s.saveMutable(ctx, material, expectedRevision, data)
}

func (s *Service) SecretInputs(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) ([]SecretInput, error) {
	material, err := s.Resolve(ctx, ref)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0)
	for name, declaration := range material.Document.Configuration.Install.Inputs {
		switch declaration.Kind {
		case serverMCPDomain.InputSecret,
			serverMCPDomain.InputOAuthClientCredentials:
			names = append(names, name)
		default:
		}
	}
	sort.Strings(names)

	output := make([]SecretInput, 0, len(names))
	for _, name := range names {
		declaration := material.Document.Configuration.Install.Inputs[name]
		binding, found := material.Installation.Inputs[name]
		output = append(output, SecretInput{
			Name:        name,
			Label:       declaration.Label,
			Description: declaration.Description,
			Required:    declaration.Required,
			Kind:        declaration.Kind,
			Configured:  found && binding.SecretRef != "",
		})
	}
	return output, nil
}

func (s *Service) Purge(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) error {
	return s.overlays.PurgeServerLocalState(ctx, ref)
}

func (s *Service) saveMutable(
	ctx context.Context,
	material Material,
	expectedRevision uint64,
	data serverMCPDomain.ServerData,
) (artifactModel.Artifact, error) {
	record := material.Resource.Artifact
	if record.Revision != expectedRevision {
		return artifactModel.Artifact{}, spec.ErrConflict
	}

	encoded, err := serverMCPDomain.MergeServerData(record.Data, data)
	if err != nil {
		return artifactModel.Artifact{}, err
	}
	if jsonutil.Equal(record.Data, encoded) {
		return record.Clone(), nil
	}

	updated, err := s.artifacts.UpdateData(
		ctx,
		record.Ref(),
		expectedRevision,
		encoded,
	)
	if err != nil {
		return artifactModel.Artifact{}, err
	}
	if err := serverMCPDomain.CleanupUnboundServerSecrets(
		ctx,
		updated.Ref(),
		material.Document,
		data,
		s.secretCleaner,
	); err != nil {
		return updated, fmt.Errorf(
			"MCP server secret cleanup remains pending: %w",
			err,
		)
	}
	return updated, nil
}

func (s *Service) saveProtected(
	ctx context.Context,
	material Material,
	expectedRevision uint64,
	data serverMCPDomain.ServerData,
) (artifactModel.Artifact, error) {
	record := material.Resource.Artifact
	if !s.protection.IsProtectedRoot(record.RootID) || !material.BuiltIn {
		return artifactModel.Artifact{}, fmt.Errorf(
			"%w: MCP Server is not in a protected Root",
			spec.ErrProtected,
		)
	}

	current, found, err := s.overlays.GetServerOverlay(
		ctx,
		record.Ref(),
	)
	if err != nil {
		return artifactModel.Artifact{}, err
	}
	if found && current.Revision != expectedRevision {
		return artifactModel.Artifact{}, spec.ErrConflict
	}
	if !found && expectedRevision != 0 {
		return artifactModel.Artifact{}, spec.ErrConflict
	}

	nextRevision := uint64(1)
	if found {
		nextRevision = current.Revision + 1
	}
	next := mcpOverlay.ServerOverlay{
		SchemaVersion: mcpDomain.InstallationDataSchemaVersion,
		Revision:      nextRevision,
		ServerData:    data,
	}
	if err := s.overlays.PutServerOverlay(
		ctx,
		record.Ref(),
		record.Revision,
		expectedRevision,
		next,
	); err != nil {
		return artifactModel.Artifact{}, err
	}
	if err := serverMCPDomain.CleanupUnboundServerSecrets(
		ctx,
		record.Ref(),
		material.Document,
		data,
		s.secretCleaner,
	); err != nil {
		return record.Clone(), fmt.Errorf(
			"MCP protected server secret cleanup remains pending: %w",
			err,
		)
	}
	return record.Clone(), nil
}

func (s *Service) effective(
	ctx context.Context,
	record artifactModel.Artifact,
	document serverMCPDomain.ServerDocument,
) (
	data serverMCPDomain.ServerData,
	effectiveRevision uint64,
	writeRevision uint64,
	builtIn bool,
	err error,
) {
	builtIn = s.protection.IsProtectedRoot(record.RootID)
	if !builtIn {
		data, err := serverMCPDomain.DecodeServerData(record.Data)
		if err != nil {
			return serverMCPDomain.ServerData{}, 0, 0, false, err
		}
		if err := data.ValidateFor(record.Ref(), document); err != nil {
			return serverMCPDomain.ServerData{}, 0, 0, false, err
		}
		return data, record.Revision, record.Revision, false, nil
	}

	overlay, found, err := s.overlays.GetServerOverlay(ctx, record.Ref())
	if err != nil {
		return serverMCPDomain.ServerData{}, 0, 0, true, err
	}
	if !found {
		return serverMCPDomain.DefaultServerData(), 1, 0, true, nil
	}
	if err := overlay.ServerData.ValidateFor(record.Ref(), document); err != nil {
		return serverMCPDomain.ServerData{}, 0, 0, true, err
	}
	return overlay.ServerData, overlay.Revision, overlay.Revision, true, nil
}
