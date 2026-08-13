package schemaadapter

import (
	"fmt"
	"path"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/bundle"
)

type ArtifactRegistration struct {
	ID          basespec.ArtifactID         `json:"id"`
	Subresource basespec.SubresourceLocator `json:"subresource"`
	Kind        basespec.ArtifactKind       `json:"kind"`
	Enabled     bool                        `json:"enabled"`
}

type BundleRegistration struct {
	CollectionID            basespec.CollectionID  `json:"collectionID"`
	EmbeddedPackageRoot     basespec.Locator       `json:"embeddedPackageRoot"`
	EmbeddedDocumentLocator basespec.Locator       `json:"embeddedDocumentLocator"`
	Artifacts               []ArtifactRegistration `json:"artifacts"`
}

type Registry struct {
	SchemaVersion string               `json:"schemaVersion"`
	Bundles       []BundleRegistration `json:"bundles"`
}

func (r Registry) Validate() error {
	if r.SchemaVersion != artifactbuiltin.MCPSchemaVersion {
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
			registered.EmbeddedPackageRoot,
			false,
		); err != nil {
			return fmt.Errorf("bundles[%d]: %w", index, err)
		}
		if err := bundle.ValidateDocumentLocator(
			registered.EmbeddedDocumentLocator,
		); err != nil {
			return fmt.Errorf("bundles[%d]: %w", index, err)
		}
		if path.Dir(string(registered.EmbeddedDocumentLocator)) !=
			string(registered.EmbeddedPackageRoot) {
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
			var expectedParent string
			switch value.Kind {
			case artifactbuiltin.ServerKind:
				expectedParent = string(artifactbuiltin.MCPServerSubresourceDirectory)
			case artifactbuiltin.PolicyKind:
				expectedParent = string(artifactbuiltin.MCPPolicySubresourceDirectory)
			default:
				return fmt.Errorf(
					"%w: unsupported MCP built-in Artifact kind %q",
					basespec.ErrInvalid,
					value.Kind,
				)
			}
			if path.Dir(string(value.Subresource)) != expectedParent {
				return fmt.Errorf(
					"%w: MCP built-in subresource %q must be directly below %q",
					basespec.ErrInvalid,
					value.Subresource,
					expectedParent,
				)
			}
			if err := basespec.ValidatePortableName(
				"MCP built-in subresource name",
				path.Base(string(value.Subresource)),
			); err != nil {
				return fmt.Errorf(
					"bundles[%d].artifacts[%d]: %w",
					index,
					artifactIndex,
					err,
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
		return output[left].EmbeddedPackageRoot <
			output[right].EmbeddedPackageRoot
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
