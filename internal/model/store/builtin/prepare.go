package builtin

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/modelproviderv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/modelv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/model/store/domain"
)

const generatedCatalogName = "models"

// PreparedPackage is one generated independent Model Provider or Model package
// ready for generic Artifact Store built-in compilation.
type PreparedPackage struct {
	EmbeddedPackageRoot spec.Locator
	Address             managedpackageModel.ManagedPackageAddress
	DocumentFile        spec.Locator
	PackageFiles        []managedpackageModel.ManagedPackageFile

	ExpectedKind           artifactModel.ArtifactKind
	ExpectedLogicalName    spec.LogicalName
	ExpectedLogicalVersion spec.LogicalVersion
	ExpectedDefinition     cryptoutil.Digest
}

func PrepareProviderPackage(
	ctx context.Context,
	document modelproviderv1.ProviderDocument,
) (PreparedPackage, error) {
	if err := requirePreparationContext(ctx); err != nil {
		return PreparedPackage{}, err
	}
	if err := document.Validate(); err != nil {
		return PreparedPackage{}, err
	}

	raw, err := document.CanonicalJSON()
	if err != nil {
		return PreparedPackage{}, err
	}
	entry, err := declaration.NewEntry(document)
	if err != nil {
		return PreparedPackage{}, err
	}
	definitionValue, err := decoder.DefinitionForEntry(entry)
	if err != nil {
		return PreparedPackage{}, err
	}

	address, err := modelDomain.ModelProviderPackageAddress(
		spec.LogicalName(document.Name),
	)
	if err != nil {
		return PreparedPackage{}, err
	}
	root, err := address.Directory()
	if err != nil {
		return PreparedPackage{}, err
	}

	return PreparedPackage{
		EmbeddedPackageRoot: root,
		Address:             address,
		DocumentFile:        modelDomain.ModelProviderDocumentFile(),
		PackageFiles: []managedpackageModel.ManagedPackageFile{{
			Locator: modelDomain.ModelProviderDocumentFile(),
			Content: raw,
		}},
		ExpectedKind:           modelDomain.ModelProviderArtifactKind,
		ExpectedLogicalName:    definitionValue.LogicalName,
		ExpectedLogicalVersion: definitionValue.LogicalVersion,
		ExpectedDefinition:     definitionValue.Digest,
	}, nil
}

func PrepareModelPackage(
	ctx context.Context,
	document modelv1.ModelDocument,
) (PreparedPackage, error) {
	if err := requirePreparationContext(ctx); err != nil {
		return PreparedPackage{}, err
	}
	if err := document.Validate(); err != nil {
		return PreparedPackage{}, err
	}

	raw, err := document.CanonicalJSON()
	if err != nil {
		return PreparedPackage{}, err
	}
	entry, err := declaration.NewEntry(document)
	if err != nil {
		return PreparedPackage{}, err
	}
	definitionValue, err := decoder.DefinitionForEntry(entry)
	if err != nil {
		return PreparedPackage{}, err
	}

	address, err := modelDomain.ModelPackageAddress(
		spec.LogicalName(document.Name),
	)
	if err != nil {
		return PreparedPackage{}, err
	}
	root, err := address.Directory()
	if err != nil {
		return PreparedPackage{}, err
	}

	return PreparedPackage{
		EmbeddedPackageRoot: root,
		Address:             address,
		DocumentFile:        modelDomain.ModelDocumentFile(),
		PackageFiles: []managedpackageModel.ManagedPackageFile{{
			Locator: modelDomain.ModelDocumentFile(),
			Content: raw,
		}},
		ExpectedKind:           modelDomain.ModelArtifactKind,
		ExpectedLogicalName:    definitionValue.LogicalName,
		ExpectedLogicalVersion: definitionValue.LogicalVersion,
		ExpectedDefinition:     definitionValue.Digest,
	}, nil
}

func NormalizePreparedPackages(
	values []PreparedPackage,
) ([]PreparedPackage, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf(
			"%w: generated Model catalog has no packages",
			spec.ErrInvalid,
		)
	}

	seen := make(map[managedpackageModel.ManagedPackageAddress]struct{}, len(values))
	output := make([]PreparedPackage, len(values))
	for index, value := range values {
		if err := value.Validate(); err != nil {
			return nil, fmt.Errorf(
				"generated Model package %d: %w",
				index,
				err,
			)
		}
		if _, duplicate := seen[value.Address]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate generated Model package %q",
				spec.ErrConflict,
				value.Address,
			)
		}
		seen[value.Address] = struct{}{}
		output[index] = clonePreparedPackage(value)
	}

	sort.Slice(output, func(left, right int) bool {
		leftDirectory, _ := output[left].Address.Directory()
		rightDirectory, _ := output[right].Address.Directory()
		return leftDirectory < rightDirectory
	})
	return output, nil
}

func (p PreparedPackage) Validate() error {
	if err := p.EmbeddedPackageRoot.ValidatePortable(false); err != nil {
		return err
	}
	if err := p.Address.Validate(); err != nil {
		return err
	}
	if err := p.DocumentFile.ValidatePortable(false); err != nil {
		return err
	}
	if _, err := managedpackageModel.NormalizeManagedPackageFiles(
		p.PackageFiles,
	); err != nil {
		return err
	}
	if err := p.ExpectedKind.Validate(); err != nil {
		return err
	}
	if err := p.ExpectedLogicalName.Validate(); err != nil {
		return err
	}
	if err := p.ExpectedLogicalVersion.Validate(true); err != nil {
		return err
	}
	return cryptoutil.ValidateDigest(p.ExpectedDefinition)
}

func clonePreparedPackage(value PreparedPackage) PreparedPackage {
	output := value
	output.PackageFiles = make(
		[]managedpackageModel.ManagedPackageFile,
		len(value.PackageFiles),
	)
	for index, file := range value.PackageFiles {
		output.PackageFiles[index] = managedpackageModel.ManagedPackageFile{
			Locator: file.Locator,
			Content: append([]byte(nil), file.Content...),
		}
	}
	return output
}

func requirePreparationContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: generated Model package preparation context is nil",
			spec.ErrInvalid,
		)
	}
	return ctx.Err()
}
