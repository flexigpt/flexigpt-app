package store

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

type mcpCollectionBehavior struct{}

func NewCollectionBehavior() providerapi.CollectionBehavior {
	return mcpCollectionBehavior{}
}

func (mcpCollectionBehavior) CollectionKind() basespec.CollectionKind {
	return artifactbuiltin.BundleKind
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

	data, err := DecodeCollectionData(collectionValue.Data)
	if err != nil {
		return providerapi.Plan{}, err
	}
	if len(attachments) != 1 {
		return providerapi.Plan{}, fmt.Errorf(
			"%w: MCP Bundle must have exactly one Source Attachment",
			basespec.ErrInvalid,
		)
	}

	attachment := attachments[0]
	if attachment.RootID != collectionValue.RootID ||
		attachment.CollectionID != collectionValue.ID {
		return providerapi.Plan{}, fmt.Errorf(
			"%w: MCP Bundle attachment belongs to another collection",
			basespec.ErrInvalid,
		)
	}
	if attachment.Role != artifactbuiltin.ManagedAttachmentRole &&
		attachment.Role != artifactbuiltin.BuiltInAttachmentRole {
		return providerapi.Plan{}, fmt.Errorf(
			"%w: unsupported MCP Attachment role %q",
			basespec.ErrInvalid,
			attachment.Role,
		)
	}

	attachmentData, err := DecodeAttachmentData(attachment.Data)
	if err != nil {
		return providerapi.Plan{}, err
	}
	if err := validateBundlePackageAddress(
		attachmentData.PackageAddress,
	); err != nil {
		return providerapi.Plan{}, err
	}
	documentLocator, err := attachmentData.DocumentLocator()
	if err != nil {
		return providerapi.Plan{}, err
	}

	sourcesByID := make(
		map[basespec.SourceID]providerapi.Source,
		len(sources),
	)
	for index, sourceValue := range sources {
		if sourceValue.RootID != collectionValue.RootID {
			return providerapi.Plan{}, fmt.Errorf(
				"%w: MCP provider source %d belongs to another root",
				basespec.ErrInvalid,
				index,
			)
		}
		if _, duplicate := sourcesByID[sourceValue.ID]; duplicate {
			return providerapi.Plan{}, fmt.Errorf(
				"%w: MCP provider received duplicate source %q",
				basespec.ErrInvalid,
				sourceValue.ID,
			)
		}
		sourcesByID[sourceValue.ID] = sourceValue
	}

	sourceValue, found := sourcesByID[attachment.SourceID]
	if !found {
		return providerapi.Plan{}, fmt.Errorf(
			"%w: MCP Bundle Source %q is unavailable",
			basespec.ErrAttachmentNotFound,
			attachment.SourceID,
		)
	}
	if sourceValue.Kind != basespec.SourceKindManagedDirectory {
		return providerapi.Plan{}, fmt.Errorf(
			"%w: MCP Bundle requires a managed Source",
			basespec.ErrInvalid,
		)
	}
	if data.ManagedSourceID != "" &&
		data.ManagedSourceID != sourceValue.ID {
		return providerapi.Plan{}, fmt.Errorf(
			"%w: MCP Bundle managed Source ownership mismatch",
			basespec.ErrInvalid,
		)
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
		Sources:  []providerapi.SourcePlan{p.Normalized()},
	}
	if err := plan.Validate(); err != nil {
		return providerapi.Plan{}, err
	}
	return plan.Normalized(), nil
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
