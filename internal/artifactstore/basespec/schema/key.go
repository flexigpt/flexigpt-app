package schema

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
)

type (
	// Kind is the entity-neutral kind portion of a schema key.
	//
	// The Entity field determines the validation domain for Kind. The v3
	// Store currently registers Artifact schemas only.
	Kind string

	EntityType string
	SchemaID   string
)

func (v SchemaID) Validate() error {
	return basespec.ValidateIdentifier("schema ID", string(v), basespec.MaxSchemaIDBytes)
}

const (
	EntityArtifact EntityType = "artifact"
)

type Key struct {
	Entity        EntityType `json:"entity"`
	Kind          Kind       `json:"kind"`
	SchemaID      SchemaID   `json:"schemaID"`
	SchemaVersion string     `json:"schemaVersion"`
}

func (k Key) Validate() error {
	switch k.Entity {
	case EntityArtifact:
		if err := artifact.ArtifactKind(k.Kind).Validate(); err != nil {
			return err
		}

	default:
		return fmt.Errorf(
			"%w: unsupported schema entity %q",
			basespec.ErrInvalid,
			k.Entity,
		)
	}

	if err := k.SchemaID.Validate(); err != nil {
		return err
	}
	return basespec.ValidateRequiredText(
		"schema version",
		k.SchemaVersion,
		basespec.MaxVersionBytes,
	)
}

func ArtifactKey(
	kind artifact.ArtifactKind,
	schemaID SchemaID,
	schemaVersion string,
) Key {
	return Key{
		Entity:        EntityArtifact,
		Kind:          Kind(kind),
		SchemaID:      schemaID,
		SchemaVersion: schemaVersion,
	}
}
