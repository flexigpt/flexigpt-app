package builtin

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpConsumerAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/consumerapi"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/sourceformat"
)

// ArtifactRegistration is retained only as application-owned physical package
// metadata. ID is deliberately ignored. Artifact IDs are Store-owned.
type ArtifactRegistration struct {
	ID          string                      `json:"id,omitempty"`
	Subresource basespec.SubresourceLocator `json:"subresource"`
	Kind        artifact.ArtifactKind       `json:"kind"`
	Enabled     bool                        `json:"enabled"`
}

// PackageRegistration describes one physical embedded package. The JSON field
// remains `bundles` so existing embedded package indexes remain readable, but
// this no longer represents a Store Collection or an mcp.bundle Artifact.
type PackageRegistration struct {
	EmbeddedPackageRoot     basespec.Locator       `json:"embeddedPackageRoot"`
	EmbeddedDocumentLocator basespec.Locator       `json:"embeddedDocumentLocator"`
	Artifacts               []ArtifactRegistration `json:"artifacts"`
}

type Registry struct {
	SchemaVersion string                `json:"schemaVersion"`
	Packages      []PackageRegistration `json:"bundles"`
}

type PreparedPackage struct {
	Registration   PackageRegistration
	PackageAddress source.ManagedPackageAddress
	DocumentFile   basespec.Locator
	PackageFiles   []source.ManagedPackageFile
	Expectations   []mcpConsumerAPI.BuiltInArtifactExpectation
}

func LoadEmbeddedRegistry() (Registry, fs.FS, error) {
	packages, err := artifactbuiltin.EmbeddedMCPPackages()
	if err != nil {
		return Registry{}, nil, err
	}
	raw, err := artifactbuiltin.ReadEmbeddedMCPRegistry()
	if err != nil {
		return Registry{}, nil, err
	}
	registry, err := jsonutil.DecodeCanonicalObject[Registry](
		raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return Registry{}, nil, fmt.Errorf(
			"decode embedded MCP registry: %w",
			err,
		)
	}
	if err := registry.Validate(); err != nil {
		return Registry{}, nil, err
	}
	return registry, packages, nil
}

func (r Registry) Validate() error {
	if r.SchemaVersion != "v1" {
		return fmt.Errorf(
			"%w: unsupported embedded MCP registry schema %q",
			basespec.ErrInvalid,
			r.SchemaVersion,
		)
	}
	if len(r.Packages) == 0 {
		return fmt.Errorf(
			"%w: embedded MCP registry has no package registrations",
			basespec.ErrInvalid,
		)
	}

	roots := make(map[basespec.Locator]struct{}, len(r.Packages))
	for index, value := range r.Packages {
		if err := value.EmbeddedPackageRoot.ValidatePortable(false); err != nil {
			return fmt.Errorf("packages[%d]: %w", index, err)
		}
		if err := value.EmbeddedDocumentLocator.ValidatePortable(false); err != nil {
			return fmt.Errorf("packages[%d]: %w", index, err)
		}
		if path.Dir(string(value.EmbeddedDocumentLocator)) !=
			string(value.EmbeddedPackageRoot) {
			return fmt.Errorf(
				"%w: embedded MCP document must belong to package root",
				basespec.ErrInvalid,
			)
		}
		if _, duplicate := roots[value.EmbeddedPackageRoot]; duplicate {
			return fmt.Errorf(
				"%w: duplicate embedded MCP package root %q",
				basespec.ErrConflict,
				value.EmbeddedPackageRoot,
			)
		}
		roots[value.EmbeddedPackageRoot] = struct{}{}
		if len(value.Artifacts) == 0 {
			return fmt.Errorf(
				"%w: MCP package %q has no Artifact registrations",
				basespec.ErrInvalid,
				value.EmbeddedPackageRoot,
			)
		}

		subresources := make(
			map[basespec.SubresourceLocator]struct{},
			len(value.Artifacts),
		)
		for artifactIndex, registration := range value.Artifacts {
			if err := registration.Subresource.Validate(); err != nil {
				return fmt.Errorf(
					"packages[%d].artifacts[%d]: %w",
					index,
					artifactIndex,
					err,
				)
			}
			switch registration.Kind {
			case mcpDomain.MCPArtifactKind,
				mcpDomain.MCPPolicyArtifactKind:
			default:
				return fmt.Errorf(
					"%w: unsupported built-in MCP Artifact kind %q",
					basespec.ErrInvalid,
					registration.Kind,
				)
			}
			if _, duplicate := subresources[registration.Subresource]; duplicate {
				return fmt.Errorf(
					"%w: duplicate built-in MCP subresource %q",
					basespec.ErrConflict,
					registration.Subresource,
				)
			}
			subresources[registration.Subresource] = struct{}{}
		}
	}
	return nil
}

func (r Registry) OrderedPackages() []PackageRegistration {
	output := append([]PackageRegistration(nil), r.Packages...)
	sort.Slice(output, func(left, right int) bool {
		return output[left].EmbeddedPackageRoot <
			output[right].EmbeddedPackageRoot
	})
	return output
}

func PreparePackages(
	ctx context.Context,
	registry Registry,
	packages fs.FS,
) ([]PreparedPackage, error) {
	if err := registry.Validate(); err != nil {
		return nil, err
	}
	if packages == nil {
		return nil, fmt.Errorf(
			"%w: embedded MCP package filesystem is nil",
			basespec.ErrInvalid,
		)
	}

	output := make([]PreparedPackage, 0, len(registry.Packages))
	for _, registration := range registry.OrderedPackages() {
		files, err := topology.ReadPackageFiles(
			ctx,
			packages,
			registration.EmbeddedPackageRoot,
		)
		if err != nil {
			return nil, err
		}
		documentFile := basespec.Locator(
			path.Base(string(registration.EmbeddedDocumentLocator)),
		)
		var document []byte
		for _, file := range files {
			if file.Locator == documentFile {
				document = append([]byte(nil), file.Content...)
				break
			}
		}
		if len(document) == 0 {
			return nil, fmt.Errorf(
				"%w: MCP package %q lacks document %q",
				basespec.ErrInvalid,
				registration.EmbeddedPackageRoot,
				documentFile,
			)
		}

		packageName := basespec.LogicalName(
			path.Base(string(registration.EmbeddedPackageRoot)),
		)
		if err := packageName.Validate(); err != nil {
			return nil, err
		}
		address, err := source.NewManagedPackageAddress(
			mcpDomain.LegacyMCPPackageKind,
			packageName,
			artifactbuiltin.UnversionedPackageVersion,
		)
		if err != nil {
			return nil, err
		}
		documentLocator, err := address.FileLocator(documentFile)
		if err != nil {
			return nil, err
		}
		collectionName, err := declaration.DeriveLogicalName(
			"mcp-bundle",
			documentLocator,
		)
		if err != nil {
			return nil, err
		}
		decoded, err := sourceformat.DecodeLegacyBundleWithCollection(
			document,
			collectionName,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"decode embedded MCP package %q: %w",
				registration.EmbeddedPackageRoot,
				err,
			)
		}
		expectations, err := expectedArtifacts(registration, decoded)
		if err != nil {
			return nil, err
		}
		normalized, err := source.NormalizeManagedPackageFiles(files)
		if err != nil {
			return nil, err
		}
		output = append(output, PreparedPackage{
			Registration:   registration,
			PackageAddress: address,
			DocumentFile:   documentFile,
			PackageFiles:   normalized,
			Expectations:   expectations,
		})
	}
	return output, nil
}

func expectedArtifacts(
	registration PackageRegistration,
	decoded []sourceformat.Decoded,
) ([]mcpConsumerAPI.BuiltInArtifactExpectation, error) {
	bySubresource := make(
		map[basespec.SubresourceLocator]definition.Definition,
		len(decoded),
	)
	for _, value := range decoded {
		if _, duplicate := bySubresource[value.SubresourceLocator]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate decoded MCP subresource %q",
				basespec.ErrInvalid,
				value.SubresourceLocator,
			)
		}
		bySubresource[value.SubresourceLocator] = value.Definition
	}

	output := make(
		[]mcpConsumerAPI.BuiltInArtifactExpectation,
		0,
		len(registration.Artifacts),
	)
	registered := make(
		map[basespec.SubresourceLocator]struct{},
		len(registration.Artifacts),
	)
	for _, item := range registration.Artifacts {
		registered[item.Subresource] = struct{}{}
		value, found := bySubresource[item.Subresource]
		if !found || value.Kind != item.Kind {
			return nil, fmt.Errorf(
				"%w: MCP package registration does not match %q",
				basespec.ErrInvalid,
				item.Subresource,
			)
		}
		output = append(output, mcpConsumerAPI.BuiltInArtifactExpectation{
			Subresource:      item.Subresource,
			Kind:             item.Kind,
			LogicalName:      value.LogicalName,
			DefinitionDigest: value.Digest,
			Enabled:          item.Enabled,
		})
	}
	for subresource, value := range bySubresource {
		if _, found := registered[subresource]; found {
			continue
		}
		if subresource != "collection" ||
			value.Kind != artifact.ArtifactKind(declaration.TypeCollection) {
			return nil, fmt.Errorf(
				"%w: MCP package emitted unexpected Artifact %q/%q",
				basespec.ErrInvalid,
				subresource,
				value.Kind,
			)
		}
		output = append(output, mcpConsumerAPI.BuiltInArtifactExpectation{
			Subresource:      subresource,
			Kind:             value.Kind,
			LogicalName:      value.LogicalName,
			DefinitionDigest: value.Digest,
			Enabled:          true,
		})
	}
	sort.Slice(output, func(left, right int) bool {
		return output[left].Subresource < output[right].Subresource
	})
	return output, nil
}

func PackageFingerprint(
	value PreparedPackage,
) (cryptoutil.Digest, error) {
	type file struct {
		Locator basespec.Locator  `json:"locator"`
		Digest  cryptoutil.Digest `json:"digest"`
		Size    int64             `json:"size"`
	}
	files := make([]file, 0, len(value.PackageFiles))
	for _, item := range value.PackageFiles {
		files = append(files, file{
			Locator: item.Locator,
			Digest:  cryptoutil.DigestBytes(item.Content),
			Size:    int64(len(item.Content)),
		})
	}
	return cryptoutil.CanonicalDigest(struct {
		Address      source.ManagedPackageAddress                `json:"address"`
		DocumentFile basespec.Locator                            `json:"documentFile"`
		Expectations []mcpConsumerAPI.BuiltInArtifactExpectation `json:"expectations"`
		Files        []file                                      `json:"files"`
	}{
		Address:      value.PackageAddress,
		DocumentFile: value.DocumentFile,
		Expectations: value.Expectations,
		Files:        files,
	})
}
