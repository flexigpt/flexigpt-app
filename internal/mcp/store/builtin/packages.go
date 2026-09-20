package builtin

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/pluginv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	mcpConsumerAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/consumerapi"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	mcpDomainPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/policy"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

// PreparedPackage is one embedded canonical MCP Plugin package ready for
// managed Source publication.
//
// Artifact IDs are intentionally absent. Artifact Store assigns them while
// refreshing the managed built-in Source.
type PreparedPackage struct {
	EmbeddedPackageRoot basespec.Locator
	PackageAddress      source.ManagedPackageAddress
	DocumentFile        basespec.Locator
	PackageFiles        []source.ManagedPackageFile
	Expectations        []mcpConsumerAPI.BuiltInArtifactExpectation
}

// PreparePackages discovers every direct embedded MCP package directory and
// derives every expected Artifact from its canonical Plugin declaration.
//
// There is deliberately no built-in MCP registry. The canonical declaration
// files are the sole source of package membership and Artifact identity.
func PreparePackages(
	ctx context.Context,
	packages fs.FS,
) ([]PreparedPackage, error) {
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: built-in MCP package preparation context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if packages == nil {
		return nil, fmt.Errorf(
			"%w: embedded MCP package filesystem is nil",
			basespec.ErrInvalid,
		)
	}

	roots, err := builtin.DirectPackageRoots(packages)
	if err != nil {
		return nil, err
	}

	output := make([]PreparedPackage, 0, len(roots))
	for _, packageRoot := range roots {
		value, err := preparePackage(ctx, packages, packageRoot)
		if err != nil {
			return nil, err
		}
		output = append(output, value)
	}

	if err := validatePreparedPackageIdentities(output); err != nil {
		return nil, err
	}
	return output, nil
}

func preparePackage(
	ctx context.Context,
	packages fs.FS,
	packageRoot basespec.Locator,
) (PreparedPackage, error) {
	files, err := topology.ReadPackageFiles(
		ctx,
		packages,
		packageRoot,
	)
	if err != nil {
		return PreparedPackage{}, err
	}
	files, err = source.NormalizeManagedPackageFiles(files)
	if err != nil {
		return PreparedPackage{}, err
	}

	documentFile, document, found, err := builtin.PackageFileContentOneOf(
		files,
		documentTopology.CollectionDocumentFiles(),
	)
	if err != nil {
		return PreparedPackage{}, err
	}
	if !found {
		return PreparedPackage{}, fmt.Errorf(
			"%w: embedded MCP package %q lacks a supported Collection document",
			basespec.ErrInvalid,
			packageRoot,
		)
	}
	expectations, err := canonicalCollectionExpectations(
		documentFile,
		document,
	)
	if err != nil {
		return PreparedPackage{}, fmt.Errorf(
			"decode embedded canonical MCP Plugin %q: %w",
			packageRoot,
			err,
		)
	}

	packageName := basespec.LogicalName(
		path.Base(string(packageRoot)),
	)
	if err := packageName.Validate(); err != nil {
		return PreparedPackage{}, err
	}
	address, err := source.NewManagedPackageAddress(
		mcpDomain.MCPCollectionPackageKind,
		packageName,
		builtin.UnversionedPackageVersion,
	)
	if err != nil {
		return PreparedPackage{}, err
	}

	return PreparedPackage{
		EmbeddedPackageRoot: packageRoot,
		PackageAddress:      address,
		DocumentFile:        documentFile,
		PackageFiles:        files,
		Expectations:        expectations,
	}, nil
}

func canonicalCollectionExpectations(
	documentFile basespec.Locator,
	document []byte,
) ([]mcpConsumerAPI.BuiltInArtifactExpectation, error) {
	raw, err := yamlutil.CanonicalObjectJSON(
		document,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return nil, err
	}
	root, err := declaration.DecodeCanonicalEntryJSON(raw)
	if err != nil {
		return nil, err
	}
	if root.Header().Type != declaration.TypePlugin {
		return nil, fmt.Errorf(
			"%w: built-in MCP package root must be a Plugin",
			basespec.ErrInvalid,
		)
	}
	if err := decoder.ValidateEntryTree(root); err != nil {
		return nil, err
	}
	collection, err := pluginv1.DecodePluginEntry(root)
	if err != nil {
		return nil, err
	}
	for index, member := range collection.Members {
		switch member.Header().Type {
		case declaration.TypeMCP, declaration.TypeMCPPolicy:
		default:
			return nil, fmt.Errorf(
				"%w: built-in MCP Plugin member %d has incompatible type %q",
				basespec.ErrInvalid,
				index,
				member.Header().Type,
			)
		}
	}
	named, err := declaration.WalkNamedEntries(root)
	if err != nil {
		return nil, err
	}

	output := make([]mcpConsumerAPI.BuiltInArtifactExpectation, 0, len(named))
	for _, value := range named {
		var definitionValue definition.Definition
		if value.SubresourceLocator == "" {
			definitionValue, err = decoder.DefinitionForEntry(value.Entry)
		} else {
			definitionValue, err = decoder.DefinitionForNamedEntry(value)
		}
		if err != nil {
			return nil, err
		}
		switch declaration.Type(definitionValue.Kind) {
		case declaration.TypePlugin:
			if value.SubresourceLocator != "" {
				continue
			}
		case declaration.TypeMCP:
			if _, err := mcpDomainServer.ServerDocumentFromDefinition(
				definitionValue,
			); err != nil {
				return nil, err
			}
		case declaration.TypeMCPPolicy:
			if _, err := mcpDomainPolicy.BodyFromDefinition(
				definitionValue,
			); err != nil {
				return nil, err
			}
		default:
			continue
		}
		output = append(
			output,
			mcpConsumerAPI.BuiltInArtifactExpectation{
				Locator:          documentFile,
				Subresource:      value.SubresourceLocator,
				Kind:             definitionValue.Kind,
				LogicalName:      definitionValue.LogicalName,
				DefinitionDigest: definitionValue.Digest,
			},
		)
	}

	sortMCPExpectations(output)
	return output, nil
}

type preparedArtifactIdentity struct {
	kind artifact.ArtifactKind
	name basespec.LogicalName
}

func validatePreparedPackageIdentities(
	packages []PreparedPackage,
) error {
	seen := make(map[preparedArtifactIdentity]basespec.Locator)
	for _, packageValue := range packages {
		for _, expected := range packageValue.Expectations {
			if err := expected.Locator.ValidatePortable(false); err != nil {
				return err
			}
			if err := expected.Subresource.Validate(); err != nil {
				return err
			}
			if err := expected.Kind.Validate(); err != nil {
				return err
			}
			if err := expected.LogicalName.Validate(); err != nil {
				return err
			}
			if err := cryptoutil.ValidateDigest(
				expected.DefinitionDigest,
			); err != nil {
				return err
			}

			identity := preparedArtifactIdentity{
				kind: expected.Kind,
				name: expected.LogicalName,
			}
			if previous, duplicate := seen[identity]; duplicate {
				return fmt.Errorf(
					"%w: embedded MCP packages %q and %q both provide %q/%q",
					basespec.ErrConflict,
					previous,
					packageValue.EmbeddedPackageRoot,
					expected.Kind,
					expected.LogicalName,
				)
			}
			seen[identity] = packageValue.EmbeddedPackageRoot
		}
	}
	return nil
}

func PackageFingerprint(
	value PreparedPackage,
) (cryptoutil.Digest, error) {
	if !documentTopology.IsCollectionDocumentFile(value.DocumentFile) {
		return "", fmt.Errorf(
			"%w: built-in MCP Collection document is not declared in topology",
			basespec.ErrInvalid,
		)
	}

	expectations := append(
		[]mcpConsumerAPI.BuiltInArtifactExpectation(nil),
		value.Expectations...,
	)
	sortMCPExpectations(expectations)

	return topology.PackageFingerprint(
		value.EmbeddedPackageRoot,
		value.PackageAddress,
		value.DocumentFile,
		expectations,
		value.PackageFiles,
	)
}

func sortMCPExpectations(
	values []mcpConsumerAPI.BuiltInArtifactExpectation,
) {
	sort.Slice(values, func(left, right int) bool {
		if values[left].Locator != values[right].Locator {
			return values[left].Locator < values[right].Locator
		}
		if values[left].Subresource != values[right].Subresource {
			return values[left].Subresource < values[right].Subresource
		}
		if values[left].Kind != values[right].Kind {
			return values[left].Kind < values[right].Kind
		}
		return values[left].LogicalName < values[right].LogicalName
	})
}
