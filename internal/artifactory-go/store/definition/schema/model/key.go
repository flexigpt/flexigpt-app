package model

import (
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type (
	// Kind is the entity-neutral kind portion of a schema key.
	//
	// The Entity field determines the validation domain for Kind. The store currently registers Artifact schemas only.
	Kind string

	EntityType string
	SchemaID   string
)

func (v SchemaID) Validate() error {
	return spec.ValidateIdentifier("schema ID", string(v), spec.MaxSchemaIDBytes)
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
		if err := artifactModel.ArtifactKind(k.Kind).Validate(); err != nil {
			return err
		}

	default:
		return fmt.Errorf(
			"%w: unsupported schema entity %q",
			spec.ErrInvalid,
			k.Entity,
		)
	}

	if err := k.SchemaID.Validate(); err != nil {
		return err
	}
	return spec.ValidateRequiredText(
		"schema version",
		k.SchemaVersion,
		spec.MaxVersionBytes,
	)
}

func ArtifactKey(
	kind artifactModel.ArtifactKind,
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
