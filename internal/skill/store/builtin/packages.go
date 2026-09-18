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
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/consumerapi"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

// PreparedPackage is one embedded canonical Skill Collection package ready for
// publication through the shared managed built-in Source.
type PreparedPackage struct {
	EmbeddedPackageRoot basespec.Locator
	PackageAddress      source.ManagedPackageAddress
	DocumentFile        basespec.Locator
	PackageFiles        []source.ManagedPackageFile
	Expectations        []skillConsumerAPI.BuiltInSkillArtifactExpectation
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
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if packages == nil {
		return nil, fmt.Errorf(
			"%w: embedded Skill package filesystem is nil",
			basespec.ErrInvalid,
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

	document, found := builtin.PackageFileContent(
		files,
		skillDomain.BuiltinSkillPluginDocumentFile,
	)
	if !found {
		return PreparedPackage{}, fmt.Errorf(
			"%w: embedded Skill package %q lacks %q",
			basespec.ErrInvalid,
			packageRoot,
			skillDomain.BuiltinSkillPluginDocumentFile,
		)
	}

	collection, expectations, err := canonicalCollectionPackage(
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

	packageName := basespec.LogicalName(
		path.Base(string(packageRoot)),
	)
	if err := packageName.Validate(); err != nil {
		return PreparedPackage{}, err
	}
	if collection.Name != string(packageName) {
		return PreparedPackage{}, fmt.Errorf(
			"%w: embedded Skill package directory %q does not match Plugin name %q",
			basespec.ErrInvalid,
			packageRoot,
			collection.Name,
		)
	}

	address, err := source.NewManagedPackageAddress(
		skillDomain.BuiltinSkillCollectionPackageKind,
		packageName,
		builtin.UnversionedPackageVersion,
	)
	if err != nil {
		return PreparedPackage{}, err
	}

	return PreparedPackage{
		EmbeddedPackageRoot: packageRoot,
		PackageAddress:      address,
		DocumentFile: skillDomain.
			BuiltinSkillPluginDocumentFile,
		PackageFiles: files,
		Expectations: expectations,
	}, nil
}

func canonicalCollectionPackage(
	document []byte,
	files []source.ManagedPackageFile,
) (
	pluginv1.PluginDocument,
	[]skillConsumerAPI.BuiltInSkillArtifactExpectation,
	error,
) {
	raw, err := yamlutil.CanonicalObjectJSON(
		document,
		basespec.MaxDefinitionBytes,
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
			basespec.ErrInvalid,
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
		map[basespec.Locator][]byte,
		len(files),
	)
	expectations := make(
		[]skillConsumerAPI.BuiltInSkillArtifactExpectation,
		0,
		1+len(collection.Members),
	)
	expectations = append(
		expectations,
		skillConsumerAPI.BuiltInSkillArtifactExpectation{
			Locator:          skillDomain.BuiltinSkillPluginDocumentFile,
			Kind:             rootDefinition.Kind,
			LogicalName:      rootDefinition.LogicalName,
			DefinitionDigest: rootDefinition.Digest,
		},
	)
	for _, file := range files {
		filesByLocator[file.Locator] = append([]byte(nil), file.Content...)
	}

	seenDocuments := map[basespec.Locator]struct{}{
		skillDomain.BuiltinSkillPluginDocumentFile: {},
	}
	for index, member := range collection.Members {
		form, err := member.MemberForm()
		if err != nil {
			return pluginv1.PluginDocument{}, nil, err
		}
		if form != declaration.MemberNamed {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Skill Plugin member %d must be a named external reference",
				basespec.ErrInvalid,
				index,
			)
		}

		header := member.Header()
		if header.Type != declaration.TypeSkill {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Skill Plugin member %d has type %q",
				basespec.ErrInvalid,
				index,
				header.Type,
			)
		}
		if header.Locator == nil {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Skill %q requires a local package locator",
				basespec.ErrInvalid,
				header.Name,
			)
		}

		documentLocator, err := skillDomain.SourceDocumentLocator(
			header.Locator,
			skillDomain.BuiltinSkillPluginDocumentFile,
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
				basespec.ErrInvalid,
				header.Name,
				skillDomain.SkillDefinitionFileName,
			)
		}
		if _, duplicate := seenDocuments[documentLocator]; duplicate {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Skill Plugin references document %q more than once",
				basespec.ErrIdentityConflict,
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
		if definitionValue.LogicalName != basespec.LogicalName(header.Name) {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Skill document name differs from Plugin member %q",
				basespec.ErrInvalid,
				header.Name,
			)
		}
		seenDocuments[documentLocator] = struct{}{}
		expectations = append(
			expectations,
			skillConsumerAPI.BuiltInSkillArtifactExpectation{
				Locator:          documentLocator,
				Kind:             definitionValue.Kind,
				LogicalName:      definitionValue.LogicalName,
				DefinitionDigest: definitionValue.Digest,
			},
		)
	}

	for locator := range filesByLocator {
		if path.Base(string(locator)) !=
			string(skillDomain.SkillDefinitionFileName) {
			continue
		}
		if _, found := seenDocuments[locator]; !found {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Skill document %q is not referenced by Plugin %q",
				basespec.ErrInvalid,
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
	name basespec.LogicalName
}

func validatePreparedPackageIdentities(
	packages []PreparedPackage,
) error {
	seen := make(
		map[preparedArtifactIdentity]basespec.Locator,
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
	if err := value.PackageAddress.Validate(); err != nil {
		return "", err
	}
	if value.DocumentFile != skillDomain.BuiltinSkillPluginDocumentFile {
		return "", fmt.Errorf(
			"%w: built-in Skill Plugin document must be %q",
			basespec.ErrInvalid,
			skillDomain.BuiltinSkillPluginDocumentFile,
		)
	}
	expectations := append(
		[]skillConsumerAPI.BuiltInSkillArtifactExpectation(nil),
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
	values []skillConsumerAPI.BuiltInSkillArtifactExpectation,
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
