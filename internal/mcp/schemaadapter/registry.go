package schemaadapter

import (
	"fmt"
	"path"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/bundle"
)

const RegistrySchemaVersion = "v1"

type ArtifactRegistration struct {
	ID          basespec.ArtifactID         `json:"id"`
	Subresource basespec.SubresourceLocator `json:"subresource"`
	Kind        basespec.ArtifactKind       `json:"kind"`
	Enabled     bool                        `json:"enabled"`
}

type BundleRegistration struct {
	CollectionID     basespec.CollectionID  `json:"collectionID"`
	PackageDirectory basespec.Locator       `json:"packageDirectory"`
	DocumentLocator  basespec.Locator       `json:"documentLocator"`
	Artifacts        []ArtifactRegistration `json:"artifacts"`
}

type Registry struct {
	SchemaVersion string               `json:"schemaVersion"`
	Bundles       []BundleRegistration `json:"bundles"`
}

func (r Registry) Validate() error {
	if r.SchemaVersion != RegistrySchemaVersion {
		return fmt.Errorf(
			"%w: unsupported MCP built-in registry schema %q",
			basespec.ErrInvalid,
			r.SchemaVersion,
		)
	}
	if len(r.Bundles) == 0 {
		return fmt.Errorf(
			"%w: MCP built-in registry has no Bundle registrations",
			basespec.ErrInvalid,
		)
	}

	collections := make(map[basespec.CollectionID]struct{}, len(r.Bundles))
	artifacts := make(map[basespec.ArtifactID]struct{})

	for index, registered := range r.Bundles {
		if err := basespec.ValidateCollectionID(registered.CollectionID); err != nil {
			return fmt.Errorf("bundles[%d]: %w", index, err)
		}
		if err := basespec.ValidatePortableLocator(
			registered.PackageDirectory,
			false,
		); err != nil {
			return fmt.Errorf("bundles[%d]: %w", index, err)
		}
		if err := bundle.ValidateDocumentLocator(
			registered.DocumentLocator,
		); err != nil {
			return fmt.Errorf("bundles[%d]: %w", index, err)
		}
		if path.Dir(string(registered.DocumentLocator)) !=
			string(registered.PackageDirectory) {
			return fmt.Errorf(
				"%w: MCP built-in document locator must belong to its package directory",
				basespec.ErrInvalid,
			)
		}
		if _, duplicate := collections[registered.CollectionID]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP built-in Collection ID %q",
				basespec.ErrConflict,
				registered.CollectionID,
			)
		}
		collections[registered.CollectionID] = struct{}{}

		if len(registered.Artifacts) == 0 {
			return fmt.Errorf(
				"%w: MCP built-in Bundle %q has no static Artifacts",
				basespec.ErrInvalid,
				registered.CollectionID,
			)
		}

		subresources := make(
			map[basespec.SubresourceLocator]struct{},
			len(registered.Artifacts),
		)
		for artifactIndex, value := range registered.Artifacts {
			if err := basespec.ValidateArtifactID(value.ID); err != nil {
				return fmt.Errorf(
					"bundles[%d].artifacts[%d]: %w",
					index,
					artifactIndex,
					err,
				)
			}
			if err := basespec.ValidateSubresourceLocator(
				value.Subresource,
			); err != nil {
				return fmt.Errorf(
					"bundles[%d].artifacts[%d]: %w",
					index,
					artifactIndex,
					err,
				)
			}
			if value.Kind != artifactbuiltin.ServerKind &&
				value.Kind != artifactbuiltin.PolicyKind {
				return fmt.Errorf(
					"%w: unsupported MCP built-in Artifact kind %q",
					basespec.ErrInvalid,
					value.Kind,
				)
			}
			if _, duplicate := artifacts[value.ID]; duplicate {
				return fmt.Errorf(
					"%w: duplicate MCP built-in Artifact ID %q",
					basespec.ErrConflict,
					value.ID,
				)
			}
			if _, duplicate := subresources[value.Subresource]; duplicate {
				return fmt.Errorf(
					"%w: duplicate MCP built-in subresource %q",
					basespec.ErrConflict,
					value.Subresource,
				)
			}
			artifacts[value.ID] = struct{}{}
			subresources[value.Subresource] = struct{}{}
		}
	}
	return nil
}

func (r Registry) OrderedBundles() []BundleRegistration {
	output := append([]BundleRegistration(nil), r.Bundles...)
	sort.Slice(output, func(left, right int) bool {
		return output[left].PackageDirectory <
			output[right].PackageDirectory
	})
	return output
}

func (r BundleRegistration) ToBundleRegistrations() []bundle.Registration {
	output := make([]bundle.Registration, 0, len(r.Artifacts))
	for _, value := range r.Artifacts {
		output = append(output, bundle.Registration{
			ArtifactID:  value.ID,
			Subresource: value.Subresource,
			Kind:        value.Kind,
			Enabled:     value.Enabled,
		})
	}
	return output
}

func (r BundleRegistration) HasExpectedSubresource(
	subresource basespec.SubresourceLocator,
) bool {
	for _, value := range r.Artifacts {
		if value.Subresource == subresource {
			return true
		}
	}
	return false
}
