package consumerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcpserverv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpDomainBundle "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/bundle"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

type documentReplacePlan struct {
	bundle Bundle

	document mcpDomainBundle.BundleDocument
	raw      json.RawMessage

	collectionData json.RawMessage

	definitions   map[basespec.SubresourceLocator]definition.Definition
	registrations map[basespec.SubresourceLocator]Registration

	dataBySubresource   map[basespec.SubresourceLocator]json.RawMessage
	orderedSubresources []basespec.SubresourceLocator

	existingBySubresource map[basespec.SubresourceLocator]artifact.Artifact

	removed []artifact.Artifact
}

func (a *API) prepareDocumentReplace(
	ctx context.Context,
	request ReplaceDocumentRequest,
	document mcpDomainBundle.BundleDocument,
	raw json.RawMessage,
) (documentReplacePlan, error) {
	if a == nil {
		return documentReplacePlan{}, basespec.ErrClosed
	}
	if err := request.Bundle.Validate(); err != nil {
		return documentReplacePlan{}, err
	}
	if request.ExpectedCollectionRevision == 0 {
		return documentReplacePlan{}, fmt.Errorf(
			"%w: expected MCP Bundle revision is required",
			basespec.ErrInvalid,
		)
	}
	if err := a.requireBundleMutation(
		ctx,
		request.Bundle.RootID,
		request.AllowProtected,
	); err != nil {
		return documentReplacePlan{}, err
	}

	bundle, err := a.Get(ctx, request.Bundle)
	if err != nil {
		return documentReplacePlan{}, err
	}
	if bundle.Collection.Revision != request.ExpectedCollectionRevision {
		return documentReplacePlan{}, basespec.ErrConflict
	}

	if len(raw) == 0 {
		return documentReplacePlan{}, fmt.Errorf(
			"%w: canonical MCP Bundle document is required",
			basespec.ErrInvalid,
		)
	}
	raw = append(json.RawMessage(nil), raw...)
	if bundle.Data.LogicalName != document.LogicalName ||
		bundle.Data.LogicalVersion != document.LogicalVersion {
		return documentReplacePlan{}, fmt.Errorf(
			"%w: MCP document identity differs from Bundle identity",
			basespec.ErrConflict,
		)
	}

	collectionData, err := mcpDomainBundle.EncodeCollectionData(mcpDomainBundle.CollectionData{
		SchemaVersion:           artifactbuiltin.MCPSchemaVersion,
		DiscoveryPolicyRevision: artifactbuiltin.DecoderRevision,
		LogicalName:             document.LogicalName,
		LogicalVersion:          document.LogicalVersion,
		Labels:                  maps.Clone(document.Labels),
		ManagedSourceID:         bundle.Data.ManagedSourceID,
	})
	if err != nil {
		return documentReplacePlan{}, err
	}

	definitionsBySubresource, err := mcpDomainBundle.DefinitionsForDocument(document)
	if err != nil {
		return documentReplacePlan{}, err
	}
	registrations, err := registrationMap(
		request.Registrations,
		definitionsBySubresource,
	)
	if err != nil {
		return documentReplacePlan{}, err
	}

	existing, err := a.artifacts.ListByCollection(
		ctx,
		bundle.Collection.Ref(),
	)
	if err != nil {
		return documentReplacePlan{}, err
	}
	existingBySubresource, err := mcpArtifactsBySubresource(bundle, existing)
	if err != nil {
		return documentReplacePlan{}, err
	}
	existingByID := make(map[artifact.ArtifactID]artifact.Artifact, len(existingBySubresource))
	for _, record := range existingBySubresource {
		existingByID[record.ID] = record
	}
	for subresource, registration := range registrations {
		if previous, found := existingByID[registration.ArtifactID]; found &&
			previous.Binding.SubresourceLocator != subresource {
			return documentReplacePlan{}, fmt.Errorf(
				"%w: MCP Artifact %q cannot be rebound from %q to %q",
				basespec.ErrConflict,
				registration.ArtifactID,
				previous.Binding.SubresourceLocator,
				subresource,
			)
		}
	}
	orderedSubresources := make(
		[]basespec.SubresourceLocator,
		0,
		len(definitionsBySubresource),
	)
	for subresource := range definitionsBySubresource {
		orderedSubresources = append(orderedSubresources, subresource)
	}
	slices.Sort(orderedSubresources)

	dataBySubresource := make(
		map[basespec.SubresourceLocator]json.RawMessage,
		len(orderedSubresources),
	)
	for _, subresource := range orderedSubresources {
		definitionValue := definitionsBySubresource[subresource]
		registration := registrations[subresource]
		existing := existingBySubresource[subresource]

		data, err := preparedRegistrationData(
			bundle,
			registration,
			definitionValue,
			existing,
		)
		if err != nil {
			return documentReplacePlan{}, err
		}
		dataBySubresource[subresource] = data
	}

	removed := make([]artifact.Artifact, 0)
	for subresource, record := range existingBySubresource {
		if _, retained := definitionsBySubresource[subresource]; retained {
			continue
		}
		removed = append(removed, record)
	}
	sort.Slice(removed, func(left, right int) bool {
		return removed[left].ID < removed[right].ID
	})

	return documentReplacePlan{
		bundle:                bundle,
		document:              document,
		raw:                   append(json.RawMessage(nil), raw...),
		collectionData:        append(json.RawMessage(nil), collectionData...),
		definitions:           definitionsBySubresource,
		registrations:         registrations,
		dataBySubresource:     dataBySubresource,
		orderedSubresources:   orderedSubresources,
		existingBySubresource: existingBySubresource,
		removed:               removed,
	}, nil
}

func preparedRegistrationData(
	bundle Bundle,
	registration Registration,
	definitionValue definition.Definition,
	existing artifact.Artifact,
) (json.RawMessage, error) {
	var (
		data json.RawMessage
		err  error
	)
	if registration.Data == nil && existing.ID != "" {
		data = append(json.RawMessage(nil), existing.Data...)
	} else {
		data, err = registrationData(registration)
		if err != nil {
			return nil, err
		}
	}

	if registration.Kind != mcpserverv1.MCPServerKind {
		return data, nil
	}

	document, err := mcpDomainServer.ServerDocumentFromDefinition(definitionValue)
	if err != nil {
		return nil, err
	}
	serverRef := artifact.ArtifactRef{
		RootID:     bundle.Collection.RootID,
		ArtifactID: registration.ArtifactID,
	}
	serverData, err := mcpDomainServer.DecodeServerData(data)
	if err != nil {
		return nil, err
	}
	if err := serverData.ValidateFor(
		serverRef,
		document,
	); err != nil {
		return nil, err
	}
	return data, nil
}

func replaceCollectionMetadataNeeded(
	bundle Bundle,
	document mcpDomainBundle.BundleDocument,
	collectionData json.RawMessage,
) bool {
	return bundle.Collection.DisplayName != displayName(document) ||
		bundle.Collection.Description != document.Description ||
		!jsonutil.Equal(bundle.Collection.Data, collectionData)
}
