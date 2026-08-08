package bundle

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
	"github.com/flexigpt/flexigpt-app/internal/builtin/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type skillCollectionCodec struct{}

func NewShareableCodec() shareable.Codec {
	return skillCollectionCodec{}
}

func (skillCollectionCodec) Key() shareable.SchemaKey {
	return shareable.SchemaKey{
		Entity:        shareable.EntityCollection,
		Kind:          CollectionKind,
		SchemaID:      PortableBundleSchemaID,
		SchemaVersion: PortableBundleSchemaVersion,
	}
}

func (skillCollectionCodec) JSONSchema() []byte {
	return schema.SkillCollectionV1JSONSchema()
}

func (c skillCollectionCodec) Canonicalize(
	_ context.Context,
	raw []byte,
) (shareable.ParsedDocument, error) {
	wire, err := schema.ParseSkillCollectionV1(raw)
	if err != nil {
		return shareable.ParsedDocument{}, err
	}
	value, err := collectionDefinitionFromWire(wire)
	if err != nil {
		return shareable.ParsedDocument{}, err
	}
	canonical, err := CanonicalizePortableBundleDefinition(value)
	if err != nil {
		return shareable.ParsedDocument{}, err
	}
	encoded, err := marshalCollectionDefinitionWire(canonical)
	if err != nil {
		return shareable.ParsedDocument{}, err
	}
	return shareable.ParsedDocument{
		Key:    c.Key(),
		Digest: canonical.Digest,
		Raw:    json.RawMessage(encoded),
	}, nil
}

func collectionDefinitionFromWire(
	input schema.SkillCollectionV1,
) (definition.CollectionDefinition, error) {
	output := definition.CollectionDefinition{
		Kind:           basespec.CollectionKind(input.Kind),
		SchemaID:       basespec.SchemaID(input.SchemaID),
		SchemaVersion:  input.SchemaVersion,
		LogicalName:    basespec.LogicalName(input.LogicalName),
		LogicalVersion: basespec.LogicalVersion(input.LogicalVersion),
		DisplayName:    input.DisplayName,
		Description:    input.Description,
		Labels:         input.Labels,
		Body:           append(json.RawMessage(nil), input.Body...),
		Members:        make([]definition.ContentRef, 0, len(input.Members)),
	}
	if input.Digest != nil {
		output.Digest = cryptoutil.Digest(*input.Digest)
	}
	for index, member := range input.Members {
		value, err := contentRefFromWire(member)
		if err != nil {
			return definition.CollectionDefinition{}, fmt.Errorf(
				"members[%d]: %w",
				index,
				err,
			)
		}
		output.Members = append(output.Members, value)
	}
	return output, nil
}

func contentRefFromWire(
	input schema.ContentRef,
) (definition.ContentRef, error) {
	output := definition.ContentRef{
		Locator:            basespec.Locator(input.Locator),
		URI:                input.URI,
		SubresourceLocator: basespec.SubresourceLocator(input.SubresourceLocator),
		MediaType:          input.MediaType,
		Role:               input.Role,
	}
	if input.Digest != nil {
		digest := cryptoutil.Digest(*input.Digest)
		output.Digest = &digest
	}
	if err := output.Validate(); err != nil {
		return definition.ContentRef{}, err
	}
	return output, nil
}

func marshalCollectionDefinitionWire(
	input definition.CollectionDefinition,
) ([]byte, error) {
	wire := schema.SkillCollectionV1{
		Kind:           string(input.Kind),
		SchemaID:       string(input.SchemaID),
		SchemaVersion:  input.SchemaVersion,
		LogicalName:    string(input.LogicalName),
		LogicalVersion: string(input.LogicalVersion),
		DisplayName:    input.DisplayName,
		Description:    input.Description,
		Labels:         input.Labels,
		Body:           append(json.RawMessage(nil), input.Body...),
		Members:        make([]schema.ContentRef, 0, len(input.Members)),
	}
	if input.Digest != "" {
		digest := string(input.Digest)
		wire.Digest = &digest
	}
	for _, member := range input.Members {
		value := schema.ContentRef{
			Locator:            string(member.Locator),
			URI:                member.URI,
			SubresourceLocator: string(member.SubresourceLocator),
			MediaType:          member.MediaType,
			Role:               member.Role,
		}
		if member.Digest != nil {
			digest := string(*member.Digest)
			value.Digest = &digest
		}
		wire.Members = append(wire.Members, value)
	}
	return schema.MarshalSkillCollectionV1(wire)
}
