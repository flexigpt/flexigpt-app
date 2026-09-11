package providerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcpbundlev1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	mcpDomainBundle "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/bundle"
)

type mcpCollectionBehavior struct{}

func NewCollectionBehavior() providerapi.CollectionBehavior {
	return mcpCollectionBehavior{}
}

func (mcpCollectionBehavior) CollectionKind() collection.CollectionKind {
	return mcpbundlev1.MCPBundleKind
}

func (mcpCollectionBehavior) Revision() string {
	return artifactbuiltin.DecoderRevision
}

func (b mcpCollectionBehavior) BuildDiscoveryPlan(
	ctx context.Context,
	collectionValue providerapi.Collection,
	attachments []providerapi.Attachment,
	sources []providerapi.Source,
) (providerapi.Plan, error) {
	if ctx == nil {
		return providerapi.Plan{}, fmt.Errorf(
			"%w: MCP collection behavior context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return providerapi.Plan{}, err
	}
	if collectionValue.Kind != b.CollectionKind() {
		return providerapi.Plan{}, fmt.Errorf(
			"%w: MCP collection behavior received collection kind %q",
			basespec.ErrInvalid,
			collectionValue.Kind,
		)
	}

	data, err := mcpDomainBundle.DecodeCollectionData(collectionValue.Data)
	if err != nil {
		return providerapi.Plan{}, err
	}
	topology := mcpDomainBundle.StoreTopology{
		Collection: collection.CollectionRef{
			RootID:       collectionValue.RootID,
			CollectionID: collectionValue.ID,
		},
		Data:        data,
		Attachments: make([]mcpDomainBundle.StoreAttachment, 0, len(attachments)),
		Sources:     make([]mcpDomainBundle.StoreSource, 0, len(sources)),
	}
	for _, attachment := range attachments {
		topology.Attachments = append(
			topology.Attachments,
			mcpDomainBundle.StoreAttachment{
				RootID:       attachment.RootID,
				CollectionID: attachment.CollectionID,
				SourceID:     attachment.SourceID,
				Role:         attachment.Role,
			},
		)
	}
	for _, sourceValue := range sources {
		topology.Sources = append(
			topology.Sources,
			mcpDomainBundle.StoreSource{
				ID:     sourceValue.ID,
				RootID: sourceValue.RootID,
				Kind:   sourceValue.Kind,
			},
		)
	}

	if err := mcpDomainBundle.ValidateStoreTopology(topology); err != nil {
		return providerapi.Plan{}, err
	}
	attachment := attachments[0]
	attachmentData, err := mcpDomainBundle.DecodeAttachmentData(
		attachment.Data,
	)
	if err != nil {
		return providerapi.Plan{}, err
	}
	documentLocator, err := mcpDomainBundle.DocumentLocatorForPackage(
		attachmentData.PackageAddress,
	)
	if err != nil {
		return providerapi.Plan{}, err
	}
	p := providerapi.SourcePlan{
		SourceID: attachment.SourceID,
		ExplicitLocators: []basespec.Locator{
			documentLocator,
		},
		DecoderHints: []providerapi.DecoderHint{{
			Locator:   documentLocator,
			Recursive: false,
			DecoderIDs: []basespec.DecoderID{
				artifactbuiltin.DecoderID,
			},
		}},
		AllowedDecoderIDs: []basespec.DecoderID{
			artifactbuiltin.DecoderID,
		},
		Authoritative: true,
	}
	plan := providerapi.Plan{
		Revision: b.Revision(),
		Sources:  []providerapi.SourcePlan{p},
	}
	return plan, nil
}

func (mcpCollectionBehavior) DecideAutomaticAdoption(
	ctx context.Context,
	_ providerapi.AdoptionInput,
) (providerapi.AdoptionDecision, error) {
	if ctx == nil {
		return providerapi.AdoptionDecision{}, fmt.Errorf(
			"%w: MCP automatic adoption context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return providerapi.AdoptionDecision{}, err
	}

	// MCP artifacts are explicitly registered and pinned from the canonical
	// MCP Bundle document. They must never be automatically adopted.
	return providerapi.AdoptionDecision{}, nil
}
