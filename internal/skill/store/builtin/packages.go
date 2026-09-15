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

// PreparePackages discovers direct embedded Skill Collection package
// directories. The canonical collection.yaml document is the only registry:
// it declares collection identity and named nested Artifact membership.
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

	roots, err := collectionPackageRoots(packages)
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

func collectionPackageRoots(
	packages fs.FS,
) ([]basespec.Locator, error) {
	entries, err := fs.ReadDir(packages, ".")
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf(
			"%w: embedded Skill package filesystem has no packages",
			basespec.ErrInvalid,
		)
	}

	output := make([]basespec.Locator, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			return nil, fmt.Errorf(
				"%w: embedded Skill package root contains non-directory %q",
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
			"%w: embedded Skill package %q lacks %q",
			basespec.ErrInvalid,
			packageRoot,
			skillDomain.BuiltinSkillCollectionDocumentFile,
		)
	}

	entries, expectations, err := canonicalCollectionPackage(document)
	if err != nil {
		return PreparedPackage{}, fmt.Errorf(
			"decode embedded canonical Skill Collection %q: %w",
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
	if entries[0].Entry.Header().Name != string(packageName) {
		return PreparedPackage{}, fmt.Errorf(
			"%w: embedded Skill package directory %q does not match Collection name %q",
			basespec.ErrInvalid,
			packageRoot,
			entries[0].Entry.Header().Name,
		)
	}
	if err := validateCollectionSkillResources(entries, files); err != nil {
		return PreparedPackage{}, err
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
			BuiltinSkillCollectionDocumentFile,
		PackageFiles: files,
		Expectations: expectations,
	}, nil
}

func packageDocument(
	files []source.ManagedPackageFile,
) ([]byte, bool) {
	for _, file := range files {
		if file.Locator !=
			skillDomain.BuiltinSkillCollectionDocumentFile {
			continue
		}
		return append([]byte(nil), file.Content...), true
	}
	return nil, false
}

func canonicalCollectionPackage(
	document []byte,
) (
	[]declaration.NamedEntry,
	[]skillConsumerAPI.BuiltInSkillArtifactExpectation,
	error,
) {
	raw, err := yamlutil.CanonicalObjectJSON(
		document,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return nil, nil, err
	}
	root, err := declaration.DecodeCanonicalEntryJSON(raw)
	if err != nil {
		return nil, nil, err
	}
	if root.Header().Type != declaration.TypeCollection {
		return nil, nil, fmt.Errorf(
			"%w: built-in Skill package root must be a Collection",
			basespec.ErrInvalid,
		)
	}
	if err := decoder.ValidateEntryTree(root); err != nil {
		return nil, nil, err
	}

	entries, err := declaration.WalkNamedEntries(root)
	if err != nil {
		return nil, nil, err
	}
	if len(entries) == 0 ||
		entries[0].SubresourceLocator != "" ||
		entries[0].Entry.Header().Type != declaration.TypeCollection {
		return nil, nil, fmt.Errorf(
			"%w: built-in Skill package has no root Collection Artifact",
			basespec.ErrInvalid,
		)
	}

	seen := make(
		map[basespec.SubresourceLocator]struct{},
		len(entries),
	)
	expectations := make(
		[]skillConsumerAPI.BuiltInSkillArtifactExpectation,
		0,
		len(entries),
	)
	for _, named := range entries {
		if _, duplicate := seen[named.SubresourceLocator]; duplicate {
			return nil, nil, fmt.Errorf(
				"%w: duplicate built-in Skill subresource %q",
				basespec.ErrInvalid,
				named.SubresourceLocator,
			)
		}
		seen[named.SubresourceLocator] = struct{}{}

		definitionValue, err := decoder.DefinitionForNamedEntry(named)
		if err != nil {
			return nil, nil, err
		}
		expectations = append(
			expectations,
			skillConsumerAPI.BuiltInSkillArtifactExpectation{
				Subresource:      named.SubresourceLocator,
				Kind:             definitionValue.Kind,
				LogicalName:      definitionValue.LogicalName,
				DefinitionDigest: definitionValue.Digest,
				Enabled:          true,
			},
		)
	}

	sort.Slice(expectations, func(left, right int) bool {
		if expectations[left].Subresource !=
			expectations[right].Subresource {
			return expectations[left].Subresource <
				expectations[right].Subresource
		}
		return expectations[left].Kind <
			expectations[right].Kind
	})
	return entries, expectations, nil
}

func validateCollectionSkillResources(
	entries []declaration.NamedEntry,
	files []source.ManagedPackageFile,
) error {
	filesByLocator := make(
		map[basespec.Locator][]byte,
		len(files),
	)
	for _, file := range files {
		filesByLocator[file.Locator] = file.Content
	}

	for _, named := range entries {
		if named.SubresourceLocator == "" ||
			named.Entry.Header().Type != declaration.TypeSkill {
			continue
		}

		header := named.Entry.Header()
		if header.Locator == nil {
			return fmt.Errorf(
				"%w: built-in Skill %q requires a local path locator",
				basespec.ErrInvalid,
				header.Name,
			)
		}

		// Embedded built-in packages intentionally use only source-relative
		// paths. Git, URL, package, archive, and zip resolution remain
		// future locator-resolver work.
		documentLocator, err := skillDomain.SourceDocumentLocator(
			header.Locator,
			skillDomain.BuiltinSkillCollectionDocumentFile,
		)
		if err != nil {
			return fmt.Errorf(
				"built-in Skill %q locator: %w",
				header.Name,
				err,
			)
		}
		content, found := filesByLocator[documentLocator]
		if !found {
			return fmt.Errorf(
				"%w: built-in Skill %q locator does not identify packaged %q",
				basespec.ErrInvalid,
				header.Name,
				skillDomain.SkillDefinitionFileName,
			)
		}
		if _, _, err := skillDomain.ParseSkillDocument(
			content,
			header.Name,
		); err != nil {
			return fmt.Errorf(
				"validate built-in Skill %q: %w",
				header.Name,
				err,
			)
		}
	}
	return nil
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
	type file struct {
		Locator basespec.Locator  `json:"locator"`
		Digest  cryptoutil.Digest `json:"digest"`
		Size    int64             `json:"size"`
	}

	if err := value.PackageAddress.Validate(); err != nil {
		return "", err
	}
	if err := value.DocumentFile.ValidatePortable(false); err != nil {
		return "", err
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
		PackageRoot  basespec.Locator                                   `json:"packageRoot"`
		Address      source.ManagedPackageAddress                       `json:"address"`
		DocumentFile basespec.Locator                                   `json:"documentFile"`
		Expectations []skillConsumerAPI.BuiltInSkillArtifactExpectation `json:"expectations"`
		Files        []file                                             `json:"files"`
	}{
		PackageRoot:  value.EmbeddedPackageRoot,
		Address:      value.PackageAddress,
		DocumentFile: value.DocumentFile,
		Expectations: value.Expectations,
		Files:        files,
	})
}
