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
	artifact "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	source "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

// PreparedPackage is one embedded canonical Skill Collection package ready for
// publication through the shared managed built-in Source.
type PreparedPackage struct {
	EmbeddedPackageRoot spec.Locator
	PackageAddress      source.ManagedPackageAddress
	DocumentFile        spec.Locator
	PackageFiles        []source.ManagedPackageFile
	Expectations        []ArtifactExpectation
}

// ArtifactExpectation is build-time package admission data. It belongs to the
// package compiler, not to the runtime Skill consumer API.
type ArtifactExpectation struct {
	Locator          spec.Locator
	Subresource      spec.SubresourceLocator
	Kind             artifact.ArtifactKind
	LogicalName      spec.LogicalName
	DefinitionDigest cryptoutil.Digest
}

// PreparePackages discovers direct embedded Skill Plugin package
// directories. The canonical collection.yaml document is the only registry:
// it declares Plugin identity and named Artifact membership.
func PreparePackages(
	ctx context.Context,
	packages fs.FS,
) ([]PreparedPackage, error) {
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: built-in Skill package preparation context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if packages == nil {
		return nil, fmt.Errorf(
			"%w: embedded Skill package filesystem is nil",
			spec.ErrInvalid,
		)
	}

	roots, err := builtin.DirectPackageRoots(packages)
	if err != nil {
		return nil, err
	}

	output := make([]PreparedPackage, 0, len(roots))
	for _, packageRoot := range roots {
		value, err := preparePackage(
			ctx,
			packages,
			packageRoot,
		)
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

	documentFile, document, found, err := source.PackageFileContentOneOf(
		files,
		documentTopology.CollectionDocumentFiles(),
	)
	if err != nil {
		return PreparedPackage{}, err
	}
	if !found {
		return PreparedPackage{}, fmt.Errorf(
			"%w: embedded Skill package %q lacks a supported Collection document",
			spec.ErrInvalid,
			packageRoot,
		)
	}

	collection, expectations, err := canonicalCollectionPackage(
		documentFile,
		document,
		files,
	)
	if err != nil {
		return PreparedPackage{}, fmt.Errorf(
			"decode embedded canonical Skill Plugin %q: %w",
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
	if collection.Name != string(packageName) {
		return PreparedPackage{}, fmt.Errorf(
			"%w: embedded Skill package directory %q does not match Plugin name %q",
			spec.ErrInvalid,
			packageRoot,
			collection.Name,
		)
	}

	address, err := source.NewManagedPackageAddress(
		skillDomain.BuiltinSkillCollectionPackageKind,
		packageName,
		documentTopology.UnversionedPackageVersion(),
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

func canonicalCollectionPackage(
	documentFile spec.Locator,
	document []byte,
	files []source.ManagedPackageFile,
) (
	pluginv1.PluginDocument,
	[]ArtifactExpectation,
	error,
) {
	raw, err := yamlutil.CanonicalObjectJSON(
		document,
		spec.MaxDefinitionBytes,
	)
	if err != nil {
		return pluginv1.PluginDocument{}, nil, err
	}
	root, err := declaration.DecodeCanonicalEntryJSON(raw)
	if err != nil {
		return pluginv1.PluginDocument{}, nil, err
	}
	if root.Header().Type != declaration.TypePlugin {
		return pluginv1.PluginDocument{}, nil, fmt.Errorf(
			"%w: built-in Skill package root must be a Plugin",
			spec.ErrInvalid,
		)
	}
	if err := decoder.ValidateEntryTree(root); err != nil {
		return pluginv1.PluginDocument{}, nil, err
	}

	collection, err := pluginv1.DecodePluginEntry(root)
	if err != nil {
		return pluginv1.PluginDocument{}, nil, err
	}

	rootDefinition, err := decoder.DefinitionForEntry(root)
	if err != nil {
		return pluginv1.PluginDocument{}, nil, err
	}

	filesByLocator := make(
		map[spec.Locator][]byte,
		len(files),
	)
	expectations := make(
		[]ArtifactExpectation,
		0,
		1+len(collection.Members),
	)
	expectations = append(
		expectations,
		ArtifactExpectation{
			Locator:          documentFile,
			Kind:             rootDefinition.Kind,
			LogicalName:      rootDefinition.LogicalName,
			DefinitionDigest: rootDefinition.Digest,
		},
	)
	for _, file := range files {
		filesByLocator[file.Locator] = append([]byte(nil), file.Content...)
	}

	seenDocuments := map[spec.Locator]struct{}{
		documentFile: {},
	}
	for index, member := range collection.Members {
		form, err := member.MemberForm()
		if err != nil {
			return pluginv1.PluginDocument{}, nil, err
		}
		if form != declaration.MemberNamed {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Skill Plugin member %d must be a named external reference",
				spec.ErrInvalid,
				index,
			)
		}

		header := member.Header()
		if header.Type != declaration.TypeSkill {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Skill Plugin member %d has type %q",
				spec.ErrInvalid,
				index,
				header.Type,
			)
		}
		if header.Locator == nil {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Skill %q requires a local package locator",
				spec.ErrInvalid,
				header.Name,
			)
		}

		documentLocator, err := skillDomain.SourceDocumentLocator(
			header.Locator,
			documentFile,
		)
		if err != nil {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"built-in Skill %q locator: %w",
				header.Name,
				err,
			)
		}
		content, found := filesByLocator[documentLocator]
		if !found {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Skill %q locator does not identify packaged %q",
				spec.ErrInvalid,
				header.Name,
				skillDomain.SkillDefinitionFileName(),
			)
		}
		if _, duplicate := seenDocuments[documentLocator]; duplicate {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Skill Plugin references document %q more than once",
				spec.ErrIdentityConflict,
				documentLocator,
			)
		}

		definitionValue, _, err := skillDomain.DecodeSkillDocument(
			content,
			header.Name,
		)
		if err != nil {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"validate built-in Skill %q: %w",
				header.Name,
				err,
			)
		}
		if definitionValue.LogicalName != spec.LogicalName(header.Name) {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Skill document name differs from Plugin member %q",
				spec.ErrInvalid,
				header.Name,
			)
		}
		seenDocuments[documentLocator] = struct{}{}
		expectations = append(
			expectations,
			ArtifactExpectation{
				Locator:          documentLocator,
				Kind:             definitionValue.Kind,
				LogicalName:      definitionValue.LogicalName,
				DefinitionDigest: definitionValue.Digest,
			},
		)
	}

	for locator := range filesByLocator {
		if !skillDomain.IsSkillDefinitionFile(locator) {
			continue
		}
		if _, found := seenDocuments[locator]; !found {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Skill document %q is not referenced by Plugin %q",
				spec.ErrInvalid,
				locator,
				collection.Name,
			)
		}
	}

	sortSkillExpectations(expectations)
	return collection, expectations, nil
}

type preparedArtifactIdentity struct {
	kind artifact.ArtifactKind
	name spec.LogicalName
}

func validatePreparedPackageIdentities(
	packages []PreparedPackage,
) error {
	seen := make(
		map[preparedArtifactIdentity]spec.Locator,
	)
	for _, packageValue := range packages {
		if err := packageValue.PackageAddress.Validate(); err != nil {
			return err
		}
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
					"%w: embedded Skill packages %q and %q both provide %q/%q",
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
	if err := value.PackageAddress.Validate(); err != nil {
		return "", err
	}
	if !documentTopology.IsCollectionDocumentFile(value.DocumentFile) {
		return "", fmt.Errorf(
			"%w: built-in Skill Collection document is not declared in topology",
			spec.ErrInvalid,
		)
	}
	expectations := append(
		[]ArtifactExpectation(nil),
		value.Expectations...,
	)
	sortSkillExpectations(expectations)

	return topology.PackageFingerprint(
		value.EmbeddedPackageRoot,
		value.PackageAddress,
		value.DocumentFile,
		expectations,
		value.PackageFiles,
	)
}

func sortSkillExpectations(
	values []ArtifactExpectation,
) {
	sort.Slice(values, func(left, right int) bool {
		if values[left].Locator != values[right].Locator {
			return values[left].Locator < values[right].Locator
		}
		if values[left].Subresource != values[right].Subresource {
			return values[left].Subresource < values[right].Subresource
		}
		return values[left].Kind < values[right].Kind
	})
}
