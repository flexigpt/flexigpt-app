package skillcatalog

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"sort"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	pluginv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/contract/v1"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/domain"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

// PreparedPackage is one embedded canonical Skill Plugin package ready for
// publication through the shared managed built-in Source.
type PreparedPackage struct {
	EmbeddedPackageRoot spec.Locator
	PackageAddress      managedpackageModel.ManagedPackageAddress
	DocumentFile        spec.Locator
	PackageFiles        []managedpackageModel.ManagedPackageFile
	Expectations        []ArtifactExpectation
}

// ArtifactExpectation is build-time package admission data. It belongs to the
// package compiler, not to the runtime Skill consumer API.
type ArtifactExpectation struct {
	Locator          spec.Locator
	Subresource      spec.SubresourceLocator
	Kind             artifactModel.ArtifactKind
	LogicalName      spec.LogicalName
	DefinitionDigest cryptoutil.Digest
}

// PreparePackages discovers direct embedded Skill Plugin package
// directories. The canonical plugin.yaml document is the only registry:
// it declares Plugin identity and named Artifact membership.
func PreparePackages(
	ctx context.Context,
	packages fs.FS,
	registry *interpretation.Registry,
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
	if registry == nil {
		return nil, fmt.Errorf(
			"%w: built-in Skill package interpretation registry is nil",
			spec.ErrInvalid,
		)
	}
	if packages == nil {
		return nil, fmt.Errorf(
			"%w: embedded Skill package filesystem is nil",
			spec.ErrInvalid,
		)
	}

	roots, err := managedpackage.DirectPackageRoots(packages)
	if err != nil {
		return nil, err
	}

	output := make([]PreparedPackage, 0, len(roots))
	for _, packageRoot := range roots {
		value, err := preparePackage(
			ctx,
			packages,
			packageRoot,
			registry,
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
	registry *interpretation.Registry,
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
			"%w: embedded Skill package %q lacks a supported Plugin document",
			spec.ErrInvalid,
			packageRoot,
		)
	}

	plugin, expectations, err := canonicalPluginPackage(
		documentFile,
		document,
		files,
		registry,
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
	if plugin.Name != string(packageName) {
		return PreparedPackage{}, fmt.Errorf(
			"%w: embedded Skill package directory %q does not match Plugin name %q",
			spec.ErrInvalid,
			packageRoot,
			plugin.Name,
		)
	}

	address, err := managedpackageModel.NewManagedPackageAddress(
		skillDomain.BuiltinSkillPluginPackageKind,
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

func canonicalPluginPackage(
	documentFile spec.Locator,
	document []byte,
	files []managedpackageModel.ManagedPackageFile,
	registry *interpretation.Registry,
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
	if err := registry.ValidateTree(root); err != nil {
		return pluginv1.PluginDocument{}, nil, err
	}

	plugin, err := pluginv1.DecodePluginEntry(root)
	if err != nil {
		return pluginv1.PluginDocument{}, nil, err
	}

	rootDefinition, err := registry.DefinitionForEntry(root)
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
		1+len(plugin.Members),
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
	for index, member := range plugin.Members {
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
				plugin.Name,
			)
		}
	}

	sortSkillExpectations(expectations)
	return plugin, expectations, nil
}

type preparedArtifactIdentity struct {
	kind artifactModel.ArtifactKind
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
	if !topology.IsPluginDocumentFile(value.DocumentFile) {
		return "", fmt.Errorf(
			"%w: built-in Skill Plugin document is not declared in topology",
			spec.ErrInvalid,
		)
	}
	expectations := append(
		[]ArtifactExpectation(nil),
		value.Expectations...,
	)
	sortSkillExpectations(expectations)

	return installModel.PackageFingerprint(
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
