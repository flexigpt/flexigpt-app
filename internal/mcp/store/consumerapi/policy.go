package consumerapi

import (
	"context"
	"fmt"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	mcpDomainPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/policy"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

func (a *API) UpsertManagedMCPPolicy(
	ctx context.Context,
	request ManagedMCPPolicyUpsertRequest,
) (ManagedMCPPolicyUpsertResult, error) {
	if a == nil {
		return ManagedMCPPolicyUpsertResult{}, basespec.ErrClosed
	}
	if err := request.RootID.Validate(); err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}
	if err := request.SourceID.Validate(); err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}
	if err := request.Name.Validate(); err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}
	if a.protection.IsProtectedRoot(request.RootID) {
		return ManagedMCPPolicyUpsertResult{}, fmt.Errorf(
			"%w: managed MCP Policy publication is not allowed in a protected Root",
			basespec.ErrProtected,
		)
	}

	body := request.Body
	document := mcppolicyv1.MCPPolicyDocument{
		APIVersion:  mcppolicyv1.MCPPolicySchemaVersion,
		Type:        mcppolicyv1.MCPPolicyType,
		Name:        string(request.Name),
		Description: request.Description,
		Body:        &body,
	}
	if err := document.Validate(); err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}
	definitionValue, err := mcpDomainPolicy.DefinitionForDocument(document)
	if err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}
	raw, err := document.CanonicalJSON()
	if err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}

	address, err := source.NewManagedPackageAddress(
		mcpDomain.ManagedMCPPolicyPackageKind,
		request.Name,
		builtin.UnversionedPackageVersion,
	)
	if err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}
	locator, err := address.FileLocator(
		mcpDomain.ManagedMCPPolicyDocumentFile,
	)
	if err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}

	sourceValue, err := a.ensureManagedPolicyDiscovery(
		ctx,
		request.RootID,
		request.SourceID,
		locator,
	)
	if err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}
	if sourceValue.Kind != source.SourceKindManagedDirectory {
		return ManagedMCPPolicyUpsertResult{}, fmt.Errorf(
			"%w: managed MCP Policy Source must have kind %q",
			basespec.ErrInvalid,
			source.SourceKindManagedDirectory,
		)
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		artifact.PublishArtifactRequest{
			RootID: request.RootID,
			Binding: artifact.SourceBinding{
				SourceID: request.SourceID,
				Locator:  locator,
			},
			ExpectedKind:        mcpDomain.MCPPolicyArtifactKind,
			ExpectedLogicalName: request.Name,
			ExpectedDefinition:  definitionValue.Digest,
			Package: source.ManagedPackagePublication{
				Address: address,
				Files: []source.ManagedPackageFile{{
					Locator: mcpDomain.ManagedMCPPolicyDocumentFile,
					Content: raw,
				}},
			},
		},
	)
	if err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}

	record := published.Artifact
	if record.Enabled != request.Enabled {
		record, err = a.artifacts.SetEnabled(
			ctx,
			record.Ref(),
			record.Revision,
			request.Enabled,
		)
		if err != nil {
			return ManagedMCPPolicyUpsertResult{}, err
		}
	}
	return ManagedMCPPolicyUpsertResult{
		Artifact: record,
		Address:  record.Address(),
	}, nil
}

func (a *API) policyBodyForResolvedArtifact(
	ctx context.Context,
	resolved resource.ResolvedArtifact,
) (mcpPolicy.MCPPolicy, error) {
	if resolved.Definition.Kind != mcpDomain.MCPPolicyArtifactKind ||
		resolved.Definition.SchemaID !=
			mcppolicyv1.MCPPolicySchemaKey.SchemaID ||
		resolved.Definition.SchemaVersion !=
			mcppolicyv1.MCPPolicySchemaKey.SchemaVersion {
		return mcpPolicy.MCPPolicy{}, fmt.Errorf(
			"%w: Definition is not an MCP Policy",
			basespec.ErrInvalid,
		)
	}

	document, err := mcppolicyv1.DecodeMCPPolicyJSON(
		resolved.Definition.Body,
	)
	if err != nil {
		return mcpPolicy.MCPPolicy{}, err
	}

	expectedName := resolved.Definition.LogicalName
	declarationLocator := resolved.Artifact.Binding.Locator
	seen := map[basespec.Locator]struct{}{
		declarationLocator: {},
	}

	for range basespec.MaxDiscoveryDepth {
		if document.Name != string(expectedName) {
			return mcpPolicy.MCPPolicy{}, fmt.Errorf(
				"%w: located MCP Policy name %q differs from %q",
				basespec.ErrReferenceUnresolved,
				document.Name,
				expectedName,
			)
		}
		if document.Body != nil {
			return mcpDomainPolicy.BodyFromDocument(document)
		}
		if document.Locator == nil {
			return mcpPolicy.MCPPolicy{}, fmt.Errorf(
				"%w: MCP Policy %q has no body or locator",
				basespec.ErrReferenceUnresolved,
				expectedName,
			)
		}

		target, err := declaration.ResolveSourceRelativePathLocator(
			*document.Locator,
			declarationLocator,
		)
		if err != nil {
			return mcpPolicy.MCPPolicy{}, err
		}
		if _, duplicate := seen[target]; duplicate {
			return mcpPolicy.MCPPolicy{}, fmt.Errorf(
				"%w: MCP Policy locator cycle at %q",
				basespec.ErrReferenceUnresolved,
				target,
			)
		}
		seen[target] = struct{}{}

		entry, err := a.resources.ReadSourceEntry(
			ctx,
			resolved.Artifact.RootID,
			resolved.Artifact.Binding.SourceID,
			target,
			basespec.MaxCandidateBytes,
		)
		if err != nil {
			return mcpPolicy.MCPPolicy{}, err
		}
		if entry.SourceRevision != resolved.RefreshState.SourceRevision ||
			entry.SourceGeneration != resolved.RefreshState.SourceGeneration {
			return mcpPolicy.MCPPolicy{}, fmt.Errorf(
				"%w: located MCP Policy source changed during resolution",
				basespec.ErrRefreshRequired,
			)
		}

		raw, err := yamlutil.CanonicalObjectJSON(
			entry.Content,
			basespec.MaxDefinitionBytes,
		)
		if err != nil {
			return mcpPolicy.MCPPolicy{}, err
		}
		document, err = mcppolicyv1.DecodeMCPPolicyJSON(raw)
		if err != nil {
			return mcpPolicy.MCPPolicy{}, err
		}
		declarationLocator = target
	}

	return mcpPolicy.MCPPolicy{}, fmt.Errorf(
		"%w: MCP Policy locator chain exceeds depth %d",
		basespec.ErrLocatorLimitExceeded,
		basespec.MaxDiscoveryDepth,
	)
}

func (a *API) ensureManagedPolicyDiscovery(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	locator basespec.Locator,
) (source.Summary, error) {
	value, err := a.sources.Get(ctx, rootID, sourceID)
	if err != nil {
		return source.Summary{}, err
	}
	if !value.Enabled {
		return source.Summary{}, fmt.Errorf(
			"%w: managed MCP Policy Source is disabled",
			basespec.ErrConflict,
		)
	}

	next := value.Discovery.Clone()
	inScope, err := next.InScope(locator)
	if err != nil {
		return source.Summary{}, err
	}
	if !inScope {
		next.ExplicitLocators = append(next.ExplicitLocators, locator)
	}
	if len(next.AllowedDecoderIDs) != 0 &&
		!slices.Contains(next.AllowedDecoderIDs, decoder.JSONDecoderID) {
		next.AllowedDecoderIDs = append(
			next.AllowedDecoderIDs,
			decoder.JSONDecoderID,
		)
	}
	next = next.Normalized()
	if err := next.Validate(); err != nil {
		return source.Summary{}, err
	}
	if value.Discovery.Equal(next) {
		return value, nil
	}
	return a.sources.Update(
		ctx,
		rootID,
		sourceID,
		source.Update{
			ExpectedRevision: value.Revision,
			DisplayName:      value.DisplayName,
			Enabled:          value.Enabled,
			Discovery:        &next,
		},
	)
}
