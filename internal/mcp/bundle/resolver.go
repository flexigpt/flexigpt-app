package bundle

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpArtifact "github.com/flexigpt/flexigpt-app/internal/mcp/artifact"
	"github.com/flexigpt/flexigpt-app/internal/mcp/installation"
	"github.com/flexigpt/flexigpt-app/internal/mcp/policy"
	"github.com/flexigpt/flexigpt-app/internal/mcp/schema"
	"github.com/flexigpt/flexigpt-app/internal/mcp/server"
)

func (a *API) ResolveMCPServer(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (server.Resolved, error) {
	return a.resolveMCPServer(ctx, ref, true)
}

// InspectMCPServer establishes Artifact, Collection, Catalog, Definition,
// installation, and policy validity without opening a Source snapshot or
// resolving a secret. It is deliberately for read-only status and setup
// projections. Connection and explicit runtime refresh must use
// ResolveMCPServer, which verifies exact current source bytes.
func (a *API) InspectMCPServer(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (server.Resolved, error) {
	return a.resolveMCPServer(ctx, ref, false)
}

func (a *API) resolveMCPServer(
	ctx context.Context,
	ref artifact.ArtifactRef,
	verifySource bool,
) (server.Resolved, error) {
	if a == nil {
		return server.Resolved{}, basespec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return server.Resolved{}, err
	}

	record, err := a.dependencies.Artifacts.Get(ctx, ref)
	if err != nil {
		return server.Resolved{}, err
	}
	if record.Kind != schema.ServerKind ||
		record.State != artifact.StateAvailable ||
		record.ResolvedDefinition == nil {
		return server.Resolved{}, fmt.Errorf(
			"%w: MCP Server Artifact is not available",
			basespec.ErrReferenceUnresolved,
		)
	}

	bundle, err := a.Get(ctx, collection.CollectionRef{
		RootID:       record.RootID,
		CollectionID: record.CollectionID,
	})
	if err != nil {
		return server.Resolved{}, err
	}

	snapshot, err := a.currentCatalog(ctx, bundle)
	if err != nil {
		return server.Resolved{}, err
	}
	occurrence, err := currentServerOccurrence(snapshot, record)
	if err != nil {
		return server.Resolved{}, err
	}

	definitionValue, err := definition.ReadCanonical(
		ctx,
		a.dependencies.Definitions,
		record.RootID,
		*record.ResolvedDefinition,
	)
	if err != nil {
		return server.Resolved{}, err
	}
	body, err := mcpArtifact.ServerBodyFromDefinition(
		definitionValue,
	)
	if err != nil {
		return server.Resolved{}, err
	}
	document := schema.ServerDocument{
		Kind:           schema.ServerKind,
		SchemaID:       schema.ServerSchemaID,
		SchemaVersion:  schema.SchemaVersion,
		LogicalName:    definitionValue.LogicalName,
		LogicalVersion: definitionValue.LogicalVersion,
		DisplayName:    definitionValue.DisplayName,
		Description:    definitionValue.Description,
		Labels:         maps.Clone(definitionValue.Labels),
		MCPServer:      body.MCPServer,
		Extension:      body.Extension,
	}

	sourceRevision := snapshot.SourceRevisions[record.Binding.SourceID]
	sourceGeneration := snapshot.SourceGenerations[record.Binding.SourceID]
	if sourceRevision == 0 || sourceGeneration == "" ||
		occurrence.SourceContentDigest == nil {
		return server.Resolved{}, fmt.Errorf(
			"%w: MCP Server Source has no current Catalog state",
			basespec.ErrCatalogStale,
		)
	}
	if verifySource {
		sourceValue, err := a.dependencies.SourceRuntime.Get(
			ctx,
			record.RootID,
			record.Binding.SourceID,
		)
		if err != nil {
			return server.Resolved{}, err
		}
		if sourceValue.Revision != sourceRevision {
			return server.Resolved{}, fmt.Errorf(
				"%w: MCP Source changed after Catalog publication",
				basespec.ErrCatalogStale,
			)
		}
		if err := source.VerifySnapshotContentDigest(
			ctx,
			a.dependencies.SourceRuntime,
			sourceValue,
			record.Binding.Locator,
			sourceGeneration,
			*occurrence.SourceContentDigest,
			basespec.MaxCandidateBytes,
		); err != nil {
			return server.Resolved{}, err
		}
	}

	installationData, installationRevision, runtimeEnabled, err := a.effectiveInstallation(
		ctx,
		bundle,
		record,
		document,
	)
	if err != nil {
		return server.Resolved{}, err
	}

	policyValue, err := a.effectivePolicy(
		ctx,
		bundle,
		document,
		installationData.AdditionalPolicies,
	)
	if err != nil {
		return server.Resolved{}, err
	}

	version, err := resolvedVersion(struct {
		ServerRef            artifact.ArtifactRef     `json:"server"`
		CollectionRef        collection.CollectionRef `json:"collection"`
		ArtifactRevision     uint64                   `json:"artifactRevision"`
		CatalogRevision      uint64                   `json:"catalogRevision"`
		DefinitionDigest     cryptoutil.Digest        `json:"definitionDigest"`
		SourceContentDigest  cryptoutil.Digest        `json:"sourceContentDigest"`
		SourceGeneration     string                   `json:"sourceGeneration"`
		InstallationRevision uint64                   `json:"installationRevision"`
		PolicyDigest         cryptoutil.Digest        `json:"policyDigest"`
	}{
		ServerRef:            ref,
		CollectionRef:        bundle.Collection.Ref(),
		ArtifactRevision:     record.Revision,
		CatalogRevision:      snapshot.Revision,
		DefinitionDigest:     *record.ResolvedDefinition,
		SourceContentDigest:  *occurrence.SourceContentDigest,
		SourceGeneration:     sourceGeneration,
		InstallationRevision: installationRevision,
		PolicyDigest:         policyValue.Digest,
	})
	if err != nil {
		return server.Resolved{}, err
	}

	resolved := server.Resolved{
		Server:               ref,
		Collection:           bundle.Collection.Ref(),
		ArtifactRevision:     record.Revision,
		CatalogRevision:      snapshot.Revision,
		DefinitionDigest:     *record.ResolvedDefinition,
		SourceContentDigest:  *occurrence.SourceContentDigest,
		SourceGeneration:     sourceGeneration,
		Document:             document,
		Installation:         installationData,
		Policy:               policyValue,
		InstallationRevision: installationRevision,
		RuntimeEnabled:       runtimeEnabled,
		BuiltIn: a.dependencies.RootPolicy.IsProtectedRoot(
			ref.RootID,
		),
		Version: version,
	}
	if err := resolved.Validate(); err != nil {
		return server.Resolved{}, err
	}
	return resolved, nil
}

func (a *API) currentCatalog(
	ctx context.Context,
	bundle Bundle,
) (catalog.Snapshot, error) {
	snapshot, err := catalog.ReadCurrent(
		ctx,
		a.dependencies.Catalogs,
		bundle.Collection.Ref(),
	)
	if err != nil {
		return catalog.Snapshot{}, err
	}
	planFingerprint, err := a.discoveryPlan(bundle).Fingerprint()
	if err != nil {
		return catalog.Snapshot{}, err
	}
	decoderFingerprint, err := a.dependencies.DecoderFingerprint()
	if err != nil {
		return catalog.Snapshot{}, err
	}
	if snapshot.PlanFingerprint != planFingerprint ||
		snapshot.DecoderFingerprint != decoderFingerprint {
		return catalog.Snapshot{}, fmt.Errorf(
			"%w: MCP configuration Catalog inputs changed",
			basespec.ErrCatalogStale,
		)
	}
	return snapshot, nil
}

func currentServerOccurrence(
	snapshot catalog.Snapshot,
	record artifact.Artifact,
) (catalog.Occurrence, error) {
	key := catalog.OccurrenceKey{
		CollectionID:       record.CollectionID,
		SourceID:           record.Binding.SourceID,
		Locator:            record.Binding.Locator,
		SubresourceLocator: record.Binding.SubresourceLocator,
	}
	for _, occurrence := range snapshot.Occurrences {
		if occurrence.Key != key {
			continue
		}
		if occurrence.State != catalog.OccurrenceValid ||
			occurrence.Kind != schema.ServerKind ||
			occurrence.DefinitionDigest == nil ||
			occurrence.SourceContentDigest == nil ||
			record.ResolvedDefinition == nil ||
			*occurrence.DefinitionDigest != *record.ResolvedDefinition {
			break
		}
		return catalog.CloneOccurrence(occurrence), nil
	}
	return catalog.Occurrence{}, fmt.Errorf(
		"%w: MCP Server does not match its current Catalog occurrence",
		basespec.ErrCatalogStale,
	)
}

func (a *API) effectiveInstallation(
	ctx context.Context,
	bundle Bundle,
	record artifact.Artifact,
	document schema.ServerDocument,
) (
	installationData installation.ServerData,
	installationRevision uint64,
	runtimeEnabled bool,
	err error,
) {
	if !a.dependencies.RootPolicy.IsProtectedRoot(record.RootID) {
		data, err := installation.DecodeServerData(record.Data)
		if err != nil {
			return installation.ServerData{}, 0, false, err
		}
		if err := installation.ValidateServerDataForDocument(record.Ref(), document, data); err != nil {
			return installation.ServerData{}, 0, false, err
		}
		return data,
			record.Revision,
			bundle.Collection.Enabled && record.Enabled,
			nil
	}

	if a.dependencies.Overlays == nil {
		return installation.ServerData{},
			0,
			false,
			fmt.Errorf(
				"%w: protected MCP installation overlay store is unavailable",
				basespec.ErrReferenceUnresolved,
			)
	}

	serverOverlay, found, err := a.dependencies.Overlays.GetServerOverlay(
		ctx,
		record.Ref(),
	)
	if err != nil {
		return installation.ServerData{}, 0, false, err
	}
	if !found {
		return installation.DefaultServerData(), 0, false, nil
	}
	if err := installation.ValidateServerDataForDocument(
		record.Ref(),
		document,
		serverOverlay.ServerData,
	); err != nil {
		return installation.ServerData{}, 0, false, err
	}
	bundleOverlay, bundleFound, err := a.dependencies.Overlays.GetBundleOverlay(
		ctx,
		record.RootID,
		record.CollectionID,
	)
	if err != nil {
		return installation.ServerData{}, 0, false, err
	}
	if !bundleFound {
		return serverOverlay.ServerData,
			serverOverlay.Revision,
			false,
			nil
	}
	return serverOverlay.ServerData,
		serverOverlay.Revision,
		bundle.Collection.Enabled &&
			record.Enabled &&
			serverOverlay.RuntimeEnabled &&
			bundleOverlay.RuntimeEnabled,
		nil
}

func (a *API) effectivePolicy(
	ctx context.Context,
	bundle Bundle,
	serverDocument schema.ServerDocument,
	additional []artifact.ArtifactRef,
) (policy.Effective, error) {
	values := make([]policy.MCPPolicy, 0, 1+len(additional))
	hasPrimaryPolicy := false

	if reference := serverDocument.Extension.Policy; reference != nil {
		matches, err := a.policyBodiesByLogicalName(
			ctx,
			bundle.Collection.Ref(),
			reference.Ref,
		)
		if err != nil {
			return policy.Effective{}, err
		}
		switch len(matches) {
		case 0:
			if reference.Required {
				return policy.Effective{}, fmt.Errorf(
					"%w: required MCP policy %q is unavailable",
					basespec.ErrReferenceUnresolved,
					reference.Ref,
				)
			}
		case 1:
			values = append(values, matches[0])
			hasPrimaryPolicy = true
		default:
			return policy.Effective{}, fmt.Errorf(
				"%w: MCP policy %q is ambiguous",
				basespec.ErrConflict,
				reference.Ref,
			)
		}
	}

	for _, ref := range sortedArtifactRefs(additional) {
		if ref.RootID != bundle.Collection.RootID {
			return policy.Effective{}, fmt.Errorf(
				"%w: additional MCP policy belongs to another Root",
				basespec.ErrInvalid,
			)
		}
		record, err := a.dependencies.Artifacts.Get(ctx, ref)
		if err != nil {
			return policy.Effective{}, err
		}
		if record.Kind != schema.PolicyKind ||
			record.CollectionID != bundle.Collection.ID ||
			!record.Enabled ||
			record.State != artifact.StateAvailable ||
			record.ResolvedDefinition == nil {
			return policy.Effective{}, fmt.Errorf(
				"%w: additional MCP policy %q is unavailable",
				basespec.ErrReferenceUnresolved,
				ref.ArtifactID,
			)
		}
		definitionValue, err := definition.ReadCanonical(
			ctx,
			a.dependencies.Definitions,
			record.RootID,
			*record.ResolvedDefinition,
		)
		if err != nil {
			return policy.Effective{}, err
		}
		body, err := mcpArtifact.PolicyBodyFromDefinition(
			definitionValue,
		)
		if err != nil {
			return policy.Effective{}, err
		}
		values = append(values, body)
	}

	baseline := a.dependencies.BaselinePolicy
	if hasPrimaryPolicy {
		baseline = values[0]
		values = values[1:]
	}
	return policy.Compose(baseline, values...)
}

func (a *API) policyBodiesByLogicalName(
	ctx context.Context,
	ref collection.CollectionRef,
	name basespec.LogicalName,
) ([]policy.MCPPolicy, error) {
	records, err := a.dependencies.Artifacts.ListByCollection(ctx, ref)
	if err != nil {
		return nil, err
	}
	output := make([]policy.MCPPolicy, 0)
	for _, record := range records {
		if record.Kind != schema.PolicyKind ||
			!record.Enabled ||
			record.State != artifact.StateAvailable ||
			record.ResolvedDefinition == nil {
			continue
		}
		definitionValue, err := definition.ReadCanonical(
			ctx,
			a.dependencies.Definitions,
			record.RootID,
			*record.ResolvedDefinition,
		)
		if err != nil {
			return nil, err
		}
		if definitionValue.LogicalName != name {
			continue
		}
		body, err := mcpArtifact.PolicyBodyFromDefinition(
			definitionValue,
		)
		if err != nil {
			return nil, err
		}
		output = append(output, body)
	}
	return output, nil
}

func resolvedVersion(value any) (cryptoutil.Digest, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return "", err
	}
	return cryptoutil.DigestBytes(canonical), nil
}

func sortedArtifactRefs(
	values []artifact.ArtifactRef,
) []artifact.ArtifactRef {
	output := append([]artifact.ArtifactRef(nil), values...)
	sort.Slice(output, func(left, right int) bool {
		if output[left].RootID != output[right].RootID {
			return output[left].RootID < output[right].RootID
		}
		return output[left].ArtifactID < output[right].ArtifactID
	})
	return output
}
