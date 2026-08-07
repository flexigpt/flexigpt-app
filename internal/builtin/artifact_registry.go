package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/topology"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const artifactBuiltinSchemaVersion = "v1"

// Registry describes application-owned protected topology only. Artifact
// families own their own embedded package registries and installation logic.
type Registry struct {
	SchemaVersion string `json:"schemaVersion"`
	Root          Root   `json:"root"`
	Source        Source `json:"source"`
}

type Root struct {
	ID          basespec.RootID `json:"id"`
	DisplayName string          `json:"displayName"`
	Description string          `json:"description,omitempty"`
}

type Source struct {
	ID          basespec.SourceID   `json:"id"`
	Kind        basespec.SourceKind `json:"kind"`
	DisplayName string              `json:"displayName"`
	Enabled     bool                `json:"enabled"`
}

func LoadRegistry() (Registry, error) {
	decoder := json.NewDecoder(bytes.NewReader(registryJSON))
	decoder.DisallowUnknownFields()

	var value Registry
	if err := decoder.Decode(&value); err != nil {
		return Registry{}, fmt.Errorf("decode built-in topology registry: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("built-in topology registry contains trailing JSON values")
		}
		return Registry{}, fmt.Errorf("%w: %w", basespec.ErrInvalid, err)
	}
	if err := value.Validate(); err != nil {
		return Registry{}, err
	}
	return value, nil
}

func (r Registry) Validate() error {
	if r.SchemaVersion != artifactBuiltinSchemaVersion {
		return fmt.Errorf(
			"%w: unsupported built-in topology registry schema %q",
			basespec.ErrInvalid,
			r.SchemaVersion,
		)
	}
	if err := basespec.ValidateRootID(r.Root.ID); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"built-in root display name",
		r.Root.DisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"built-in root description",
		r.Root.Description,
		basespec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateSourceID(r.Source.ID); err != nil {
		return err
	}
	if err := basespec.ValidateSourceKind(r.Source.Kind); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"built-in source display name",
		r.Source.DisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if r.Root.ID == basespec.RootID(r.Source.ID) {
		return fmt.Errorf(
			"%w: built-in Root and Source IDs must differ",
			basespec.ErrConflict,
		)
	}
	return nil
}

func (r Registry) TopologyDeclaration() (topology.Declaration, error) {
	if err := r.Validate(); err != nil {
		return topology.Declaration{}, err
	}
	return topology.Declaration{
		Root: root.RootDraft{
			ID:          r.Root.ID,
			DisplayName: r.Root.DisplayName,
			Description: r.Root.Description,
		},
		Sources: []source.Draft{{
			ID:          r.Source.ID,
			Kind:        r.Source.Kind,
			DisplayName: r.Source.DisplayName,
			Enabled:     r.Source.Enabled,
			Config:      json.RawMessage(jsonutil.EmptyObject),
		}},
	}, nil
}

// EnsureTopology establishes only shared protected topology. Typed artifact
// installers are run separately by builtin.BootstrapRegistry.
func (r Registry) EnsureTopology(
	ctx context.Context,
	ensurer topology.Ensurer,
) (topology.Installed, error) {
	if ctx == nil {
		return topology.Installed{}, fmt.Errorf(
			"%w: built-in topology context is nil",
			basespec.ErrInvalid,
		)
	}
	if ensurer == nil {
		return topology.Installed{}, fmt.Errorf(
			"%w: built-in topology ensurer is nil",
			basespec.ErrInvalid,
		)
	}
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return topology.Installed{}, err
	}
	declaration, err := r.TopologyDeclaration()
	if err != nil {
		return topology.Installed{}, err
	}
	return ensurer.EnsureProtectedTopology(ctx, declaration)
}
