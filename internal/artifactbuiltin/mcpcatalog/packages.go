package mcpcatalog

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"sort"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	pluginv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/contract/v1"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain"
	policyMCPDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain/policy"
	serverMCPDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain/server"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

// PreparedPackage is one embedded canonical MCP Plugin package ready for
// managed Source publication.
//
// Artifact IDs are intentionally absent. Artifact Store assigns them while
// refreshing the managed built-in Source.
type PreparedPackage struct {
	EmbeddedPackageRoot spec.Locator
	PackageAddress      managedpackageModel.ManagedPackageAddress
	DocumentFile        spec.Locator
	PackageFiles        []managedpackageModel.ManagedPackageFile
	Expectations        []ArtifactExpectation
}

type ArtifactExpectation struct {
	Locator          spec.Locator
	Subresource      spec.SubresourceLocator
	Kind             artifactModel.ArtifactKind
	LogicalName      spec.LogicalName
	LogicalVersion   spec.LogicalVersion
	DefinitionDigest cryptoutil.Digest
}

// PreparePackages discovers every direct embedded MCP package directory and
// derives every expected Artifact from its canonical Plugin declaration.
//
// There is deliberately no built-in MCP registry. The canonical declaration
// files are the sole source of package membership and Artifact identity.
func PreparePackages(
	ctx context.Context,
	packages fs.FS,
	registry *coreinterpretation.Registry,
) ([]PreparedPackage, error) {
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: built-in MCP package preparation context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if registry == nil {
		return nil, fmt.Errorf(
			"%w: built-in MCP package interpretation registry is nil",
			spec.ErrInvalid,
		)
	}
	if packages == nil {
		return nil, fmt.Errorf(
			"%w: embedded MCP package filesystem is nil",
			spec.ErrInvalid,
		)
	}

	roots, err := managedpackage.DirectPackageRoots(packages)
	if err != nil {
		return nil, err
	}

	output := make([]PreparedPackage, 0, len(roots))
	for _, packageRoot := range roots {
		value, err := preparePackage(ctx, packages, packageRoot, registry)
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
	packageRoot spec.Locator,
	registry *coreinterpretation.Registry,
) (PreparedPackage, error) {
	files, err := managedpackage.ReadPackageFiles(
		ctx,
		packages,
		packageRoot,
	)
	if err != nil {
		return PreparedPackage{}, err
	}
	files, err = managedpackageModel.NormalizeManagedPackageFiles(files)
	if err != nil {
		return PreparedPackage{}, err
	}

	documentFile, document, found, err := managedpackageModel.PackageFileContentOneOf(
		files,
		topology.PluginDocumentFiles(),
	)
	if err != nil {
		return PreparedPackage{}, err
	}
	if !found {
		return PreparedPackage{}, fmt.Errorf(
			"%w: embedded MCP package %q lacks a supported Plugin document",
			spec.ErrInvalid,
			packageRoot,
		)
	}
	expectations, err := canonicalPluginExpectations(
		documentFile,
		document,
		registry,
	)
	if err != nil {
		return PreparedPackage{}, fmt.Errorf(
			"decode embedded canonical MCP Plugin %q: %w",
			packageRoot,
			err,
		)
	}

	packageName := spec.LogicalName(
		path.Base(string(packageRoot)),
	)
	if err := packageName.Validate(); err != nil {
		return PreparedPackage{}, err
	}
	address, err := managedpackageModel.NewManagedPackageAddress(
		mcpDomain.MCPPluginPackageKind,
		packageName,
		topology.UnversionedPackageVersion(),
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

func canonicalPluginExpectations(
	documentFile spec.Locator,
	document []byte,
	registry *coreinterpretation.Registry,
) ([]ArtifactExpectation, error) {
	raw, err := yamlutil.CanonicalObjectJSON(
		document,
		spec.MaxDefinitionBytes,
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
			spec.ErrInvalid,
		)
	}
	if err := registry.ValidateTree(root); err != nil {
		return nil, err
	}
	plugin, err := pluginv1.DecodePluginEntry(root)
	if err != nil {
		return nil, err
	}
	for index, member := range plugin.Members {
		switch member.Header().Type {
		case declaration.TypeMCP, declaration.TypeMCPPolicy:
		default:
			return nil, fmt.Errorf(
				"%w: built-in MCP Plugin member %d has incompatible type %q",
				spec.ErrInvalid,
				index,
				member.Header().Type,
			)
		}
	}
	named, err := registry.WalkNamedEntries(root)
	if err != nil {
		return nil, err
	}

	output := make([]ArtifactExpectation, 0, len(named))
	for _, value := range named {
		definitionValue, err := registry.DefinitionForEntry(
			value.Entry,
		)
		if err != nil {
			return nil, err
		}
		switch declaration.Type(definitionValue.Kind) {
		case declaration.TypePlugin:
			if value.SubresourceLocator != "" {
				continue
			}
		case declaration.TypeMCP:
			if _, err := serverMCPDomain.ServerDocumentFromDefinition(
				definitionValue,
			); err != nil {
				return nil, err
			}
		case declaration.TypeMCPPolicy:
			if _, err := policyMCPDomain.BodyFromDefinition(
				definitionValue,
			); err != nil {
				return nil, err
			}
		default:
			continue
		}
		output = append(
			output,
			ArtifactExpectation{
				Locator:          documentFile,
				Subresource:      value.SubresourceLocator,
				Kind:             definitionValue.Kind,
				LogicalName:      definitionValue.LogicalName,
				LogicalVersion:   definitionValue.LogicalVersion,
				DefinitionDigest: definitionValue.Digest,
			},
		)
	}

	sortMCPExpectations(output)
	return output, nil
}

type preparedArtifactIdentity struct {
	kind artifactModel.ArtifactKind
	name spec.LogicalName
}

func validatePreparedPackageIdentities(
	packages []PreparedPackage,
) error {
	seen := make(map[preparedArtifactIdentity]spec.Locator)
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
					spec.ErrConflict,
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
	if !topology.IsPluginDocumentFile(value.DocumentFile) {
		return "", fmt.Errorf(
			"%w: built-in MCP Plugin document is not declared in topology",
			spec.ErrInvalid,
		)
	}

	expectations := append(
		[]ArtifactExpectation(nil),
		value.Expectations...,
	)
	sortMCPExpectations(expectations)

	return installModel.PackageFingerprint(
		value.EmbeddedPackageRoot,
		value.PackageAddress,
		value.DocumentFile,
		expectations,
		value.PackageFiles,
	)
}

func sortMCPExpectations(
	values []ArtifactExpectation,
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
