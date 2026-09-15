package builtin

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/collectionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	mcpConsumerAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/consumerapi"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	mcpDomainPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/policy"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

// PreparedPackage is one embedded canonical MCP Collection package ready for
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
// derives every expected Artifact from its canonical Collection declaration.
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

	roots, err := collectionPackageRoots(packages)
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

func collectionPackageRoots(
	packages fs.FS,
) ([]basespec.Locator, error) {
	entries, err := fs.ReadDir(packages, ".")
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf(
			"%w: embedded MCP package filesystem has no packages",
			basespec.ErrInvalid,
		)
	}

	output := make([]basespec.Locator, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			return nil, fmt.Errorf(
				"%w: embedded MCP package root contains non-directory %q",
				basespec.ErrInvalid,
				entry.Name(),
			)
		}
		root := basespec.Locator(entry.Name())
		if err := root.ValidatePortable(false); err != nil {
			return nil, err
		}
		output = append(output, root)
	}
	slices.Sort(output)
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

	document, found := packageDocument(files)
	if !found {
		return PreparedPackage{}, fmt.Errorf(
			"%w: embedded MCP package %q lacks %q",
			basespec.ErrInvalid,
			packageRoot,
			mcpDomain.MCPCollectionDocumentFile,
		)
	}
	expectations, err := canonicalCollectionExpectations(document)
	if err != nil {
		return PreparedPackage{}, fmt.Errorf(
			"decode embedded canonical MCP Collection %q: %w",
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
		DocumentFile:        mcpDomain.MCPCollectionDocumentFile,
		PackageFiles:        files,
		Expectations:        expectations,
	}, nil
}

func packageDocument(
	files []source.ManagedPackageFile,
) ([]byte, bool) {
	for _, file := range files {
		if file.Locator != mcpDomain.MCPCollectionDocumentFile {
			continue
		}
		return append([]byte(nil), file.Content...), true
	}
	return nil, false
}

func canonicalCollectionExpectations(
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
	if root.Header().Type != declaration.TypeCollection {
		return nil, fmt.Errorf(
			"%w: built-in MCP package root must be a Collection",
			basespec.ErrInvalid,
		)
	}
	if _, err := collectionv1.DecodeCollectionEntry(root); err != nil {
		return nil, err
	}
	if err := decoder.ValidateEntryTree(root); err != nil {
		return nil, err
	}

	entries, err := declaration.WalkNamedEntries(root)
	if err != nil {
		return nil, err
	}

	seen := make(
		map[basespec.SubresourceLocator]struct{},
		len(entries),
	)
	output := make(
		[]mcpConsumerAPI.BuiltInArtifactExpectation,
		0,
		len(entries),
	)
	for _, named := range entries {
		if _, duplicate := seen[named.SubresourceLocator]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate built-in MCP subresource %q",
				basespec.ErrInvalid,
				named.SubresourceLocator,
			)
		}
		seen[named.SubresourceLocator] = struct{}{}

		declarationType := named.Entry.Header().Type
		switch declarationType {
		case declaration.TypeCollection,
			declaration.TypeMCP,
			declaration.TypeMCPPolicy:
		default:
			return nil, fmt.Errorf(
				"%w: built-in MCP package emits unsupported declaration type %q",
				basespec.ErrInvalid,
				declarationType,
			)
		}

		definitionValue, err := decoder.DefinitionForNamedEntry(named)
		if err != nil {
			return nil, err
		}
		if definitionValue.Kind != artifact.ArtifactKind(declarationType) {
			return nil, fmt.Errorf(
				"%w: built-in MCP declaration type %q emitted kind %q",
				basespec.ErrInvalid,
				declarationType,
				definitionValue.Kind,
			)
		}

		switch declarationType {
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
		}

		output = append(output, mcpConsumerAPI.BuiltInArtifactExpectation{
			Subresource:      named.SubresourceLocator,
			Kind:             definitionValue.Kind,
			LogicalName:      definitionValue.LogicalName,
			DefinitionDigest: definitionValue.Digest,
			Enabled:          true,
		})
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Subresource != output[right].Subresource {
			return output[left].Subresource <
				output[right].Subresource
		}
		return output[left].Kind < output[right].Kind
	})
	if len(output) == 0 ||
		output[0].Subresource != "" ||
		output[0].Kind != artifact.ArtifactKind(
			declaration.TypeCollection,
		) {
		return nil, fmt.Errorf(
			"%w: built-in MCP package must emit a root Collection Artifact",
			basespec.ErrInvalid,
		)
	}
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
	sort.Slice(files, func(left, right int) bool {
		return files[left].Locator < files[right].Locator
	})
	return cryptoutil.CanonicalDigest(struct {
		PackageRoot  basespec.Locator                            `json:"packageRoot"`
		Address      source.ManagedPackageAddress                `json:"address"`
		DocumentFile basespec.Locator                            `json:"documentFile"`
		Expectations []mcpConsumerAPI.BuiltInArtifactExpectation `json:"expectations"`
		Files        []file                                      `json:"files"`
	}{
		PackageRoot:  value.EmbeddedPackageRoot,
		Address:      value.PackageAddress,
		DocumentFile: value.DocumentFile,
		Expectations: value.Expectations,
		Files:        files,
	})
}
