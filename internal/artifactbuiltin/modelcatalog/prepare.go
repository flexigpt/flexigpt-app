package modelcatalog

import (
	"fmt"
	"sort"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/llmsupport"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	modelv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/contract/v1"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/domain"
	modelproviderv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/modelprovider/contract/v1"
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
	document modelproviderv1.ProviderDocument,
) (PreparedPackage, error) {
	support, err := llmsupport.Model()
	if err != nil {
		return PreparedPackage{}, err
	}

	definitionValue, err := modelproviderv1.DefinitionForDocument(
		document,
	)
	if err != nil {
		return PreparedPackage{}, err
	}
	raw := append([]byte(nil), definitionValue.Body...)

	address, err := support.ProviderPackage.Address(spec.LogicalName(document.Name), "")
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
		DocumentFile:        support.ProviderPackage.Document.Locator,
		PackageFiles: []managedpackageModel.ManagedPackageFile{{
			Locator: support.ProviderPackage.Document.Locator,
			Content: raw,
		}},
		ExpectedKind:           modelDomain.ModelProviderArtifactKind,
		ExpectedLogicalName:    definitionValue.LogicalName,
		ExpectedLogicalVersion: definitionValue.LogicalVersion,
		ExpectedDefinition:     definitionValue.Digest,
	}, nil
}

func PrepareModelPackage(
	document modelv1.ModelDocument,
) (PreparedPackage, error) {
	support, err := llmsupport.Model()
	if err != nil {
		return PreparedPackage{}, err
	}

	definitionValue, err := modelv1.DefinitionForDocument(document)
	if err != nil {
		return PreparedPackage{}, err
	}
	raw := append([]byte(nil), definitionValue.Body...)

	address, err := support.ModelPackage.Address(spec.LogicalName(document.Name), "")
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
		DocumentFile:        support.ModelPackage.Document.Locator,
		PackageFiles: []managedpackageModel.ManagedPackageFile{{
			Locator: support.ModelPackage.Document.Locator,
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
