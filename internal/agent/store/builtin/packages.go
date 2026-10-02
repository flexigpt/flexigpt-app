package builtin

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	agentDomain "github.com/flexigpt/flexigpt-app/internal/agent/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/pluginv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/install/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

type PreparedPackage struct {
	EmbeddedPackageRoot model.Locator

	PackageAddress source.ManagedPackageAddress

	PluginDocumentFile model.Locator

	PackageFiles []source.ManagedPackageFile

	Expectations []ArtifactExpectation
}

// ArtifactExpectation is build-time package admission data. Runtime Agent
// consumers do not own embedded package validation.
type ArtifactExpectation struct {
	Locator          model.Locator
	Subresource      model.SubresourceLocator
	Kind             artifact.ArtifactKind
	LogicalName      model.LogicalName
	LogicalVersion   model.LogicalVersion
	DefinitionDigest cryptoutil.Digest
}

// PreparePackages reads direct embedded Agent Collection package directories.
// Every package contains one canonical Plugin document and each direct Plugin
// member identifies one concrete package-local Agent declaration.
func PreparePackages(
	ctx context.Context,
	packages fs.FS,
) ([]PreparedPackage, error) {
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: built-in Agent package preparation context is nil",
			model.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if packages == nil {
		return nil, fmt.Errorf(
			"%w: embedded Agent package filesystem is nil",
			model.ErrInvalid,
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
	packageRoot model.Locator,
) (PreparedPackage, error) {
	files, err := topology.ReadPackageFiles(ctx, packages, packageRoot)
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
			"%w: embedded Agent package %q lacks a supported Collection document",
			model.ErrInvalid,
			packageRoot,
		)
	}

	_, expectations, err := canonicalCollectionPackage(
		documentFile,
		document,
		files,
	)
	if err != nil {
		return PreparedPackage{}, fmt.Errorf(
			"decode embedded canonical Agent Plugin %q: %w",
			packageRoot,
			err,
		)
	}

	packageName := model.LogicalName(path.Base(string(packageRoot)))
	if err := packageName.Validate(); err != nil {
		return PreparedPackage{}, err
	}
	address, err := source.NewManagedPackageAddress(
		agentDomain.BuiltinAgentCollectionPackageKind,
		packageName,
		documentTopology.UnversionedPackageVersion(),
	)
	if err != nil {
		return PreparedPackage{}, err
	}

	return PreparedPackage{
		EmbeddedPackageRoot: packageRoot,
		PackageAddress:      address,
		PluginDocumentFile:  documentFile,
		PackageFiles:        files,
		Expectations:        expectations,
	}, nil
}

func canonicalCollectionPackage(
	documentFile model.Locator,
	document []byte,
	files []source.ManagedPackageFile,
) (
	pluginv1.PluginDocument,
	[]ArtifactExpectation,
	error,
) {
	raw, err := yamlutil.CanonicalObjectJSON(
		document,
		model.MaxDefinitionBytes,
	)
	if err != nil {
		return pluginv1.PluginDocument{}, nil, err
	}

	r, err := declaration.DecodeCanonicalEntryJSON(raw)
	if err != nil {
		return pluginv1.PluginDocument{}, nil, err
	}
	if r.Header().Type != declaration.TypePlugin {
		return pluginv1.PluginDocument{}, nil, fmt.Errorf(
			"%w: built-in Agent package root must be a Plugin",
			model.ErrInvalid,
		)
	}
	if err := decoder.ValidateEntryTree(r); err != nil {
		return pluginv1.PluginDocument{}, nil, err
	}

	collection, err := pluginv1.DecodePluginEntry(r)
	if err != nil {
		return pluginv1.PluginDocument{}, nil, err
	}

	expectations, err := expectationsForDocument(
		documentFile,
		r,
	)
	if err != nil {
		return pluginv1.PluginDocument{}, nil, err
	}

	filesByLocator := make(map[model.Locator][]byte, len(files))
	for _, file := range files {
		filesByLocator[file.Locator] = append([]byte(nil), file.Content...)
	}

	seenNames := make(map[model.LogicalName]model.Locator)
	seenDocuments := make(map[model.Locator]struct{})

	for index, member := range collection.Members {
		form, err := member.MemberForm()
		if err != nil {
			return pluginv1.PluginDocument{}, nil, err
		}
		if form != declaration.MemberNamed {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent Plugin member %d must be a named external Agent reference",
				model.ErrInvalid,
				index,
			)
		}

		header := member.Header()
		if header.Type != declaration.TypeAgent {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent Plugin member %d has type %q",
				model.ErrInvalid,
				index,
				header.Type,
			)
		}
		if header.Locator == nil {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent %q requires a local package locator",
				model.ErrInvalid,
				header.Name,
			)
		}

		relationship, err := member.Relationship()
		if err != nil {
			return pluginv1.PluginDocument{}, nil, err
		}
		if relationship.Scope != "" {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent %q cannot use lookup scope",
				model.ErrInvalid,
				header.Name,
			)
		}

		documentLocator, err := declaration.ResolveSourceRelativePathLocator(
			*header.Locator,
			documentFile,
		)
		if err != nil {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"built-in Agent %q locator: %w",
				header.Name,
				err,
			)
		}

		documentName, err := packageAgentDocumentName(documentLocator)
		if err != nil {
			return pluginv1.PluginDocument{}, nil, err
		}
		if header.Name != string(documentName) {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent member name %q does not match packaged document name %q",
				model.ErrInvalid,
				header.Name,
				documentName,
			)
		}

		name := model.LogicalName(header.Name)
		if previous, duplicate := seenNames[name]; duplicate {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent Plugin references Agent %q at both %q and %q",
				model.ErrIdentityConflict,
				name,
				previous,
				documentLocator,
			)
		}
		seenNames[name] = documentLocator

		if _, duplicate := seenDocuments[documentLocator]; duplicate {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent Plugin references document %q more than once",
				model.ErrIdentityConflict,
				documentLocator,
			)
		}

		content, found := filesByLocator[documentLocator]
		if !found {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent %q locator does not identify a configured packaged Agent document",
				model.ErrInvalid,
				header.Name,
			)
		}

		agentRoot, agentDocument, err := canonicalAgentDocument(content)
		if err != nil {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"validate built-in Agent %q: %w",
				header.Name,
				err,
			)
		}
		if agentDocument.Locator != nil {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent %q cannot be a source-selected alias",
				model.ErrInvalid,
				header.Name,
			)
		}
		if agentDocument.Name != header.Name {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent document name differs from Plugin member %q",
				model.ErrInvalid,
				header.Name,
			)
		}

		agentExpectations, err := expectationsForDocument(
			documentLocator,
			agentRoot,
		)
		if err != nil {
			return pluginv1.PluginDocument{}, nil, err
		}

		seenDocuments[documentLocator] = struct{}{}
		expectations = append(expectations, agentExpectations...)
	}

	for locator := range filesByLocator {
		documentFile := model.Locator(path.Base(string(locator)))
		if !agentDomain.IsAgentDeclarationDocument(documentFile) {
			continue
		}
		if _, err := packageAgentDocumentName(locator); err != nil {
			return pluginv1.PluginDocument{}, nil, err
		}
		if _, found := seenDocuments[locator]; !found {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent document %q is not referenced by Plugin %q",
				model.ErrInvalid,
				locator,
				collection.Name,
			)
		}
	}

	sortExpectations(expectations)
	return collection, expectations, nil
}

func canonicalAgentDocument(
	content []byte,
) (declaration.Entry, agentv1.AgentDocument, error) {
	raw, err := yamlutil.CanonicalObjectJSON(
		content,
		model.MaxDefinitionBytes,
	)
	if err != nil {
		return declaration.Entry{}, agentv1.AgentDocument{}, err
	}
	r, err := declaration.DecodeCanonicalEntryJSON(raw)
	if err != nil {
		return declaration.Entry{}, agentv1.AgentDocument{}, err
	}
	if r.Header().Type != declaration.TypeAgent {
		return declaration.Entry{}, agentv1.AgentDocument{}, fmt.Errorf(
			"%w: package Agent document must have type %q",
			model.ErrInvalid,
			declaration.TypeAgent,
		)
	}
	if err := decoder.ValidateEntryTree(r); err != nil {
		return declaration.Entry{}, agentv1.AgentDocument{}, err
	}
	document, err := agentv1.DecodeAgentEntry(r)
	if err != nil {
		return declaration.Entry{}, agentv1.AgentDocument{}, err
	}
	return r, document, nil
}

func expectationsForDocument(
	locator model.Locator,
	entry declaration.Entry,
) ([]ArtifactExpectation, error) {
	namedEntries, err := declaration.WalkNamedEntries(entry)
	if err != nil {
		return nil, err
	}

	output := make(
		[]ArtifactExpectation,
		0,
		len(namedEntries),
	)
	for _, named := range namedEntries {
		value, err := decoder.DefinitionForNamedEntry(named)
		if err != nil {
			return nil, err
		}
		output = append(output, ArtifactExpectation{
			Locator:          locator,
			Subresource:      named.SubresourceLocator,
			Kind:             value.Kind,
			LogicalName:      value.LogicalName,
			LogicalVersion:   value.LogicalVersion,
			DefinitionDigest: value.Digest,
		})
	}
	return output, nil
}

func packageAgentDocumentName(
	locator model.Locator,
) (model.LogicalName, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return "", err
	}

	segments := strings.Split(string(locator), "/")
	if len(segments) != 2 ||
		!agentDomain.IsAgentDeclarationDocument(
			model.Locator(segments[1]),
		) {
		return "", fmt.Errorf(
			"%w: built-in Agent document %q must use <name>/<configured-agent-document>",
			model.ErrInvalid,
			locator,
		)
	}

	name := model.LogicalName(segments[0])
	if err := name.Validate(); err != nil {
		return "", err
	}
	return name, nil
}

type preparedArtifactIdentity struct {
	kind    artifact.ArtifactKind
	name    model.LogicalName
	version model.LogicalVersion
}

func validatePreparedPackageIdentities(
	packages []PreparedPackage,
) error {
	seen := make(map[preparedArtifactIdentity]model.Locator)

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
			if err := expected.LogicalVersion.Validate(true); err != nil {
				return err
			}
			if err := cryptoutil.ValidateDigest(
				expected.DefinitionDigest,
			); err != nil {
				return err
			}

			identity := preparedArtifactIdentity{
				kind:    expected.Kind,
				name:    expected.LogicalName,
				version: expected.LogicalVersion,
			}
			if previous, duplicate := seen[identity]; duplicate {
				return fmt.Errorf(
					"%w: embedded Agent packages %q and %q both provide %q/%q/%q",
					model.ErrConflict,
					previous,
					packageValue.EmbeddedPackageRoot,
					expected.Kind,
					expected.LogicalName,
					expected.LogicalVersion,
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
	if !documentTopology.IsCollectionDocumentFile(value.PluginDocumentFile) {
		return "", fmt.Errorf(
			"%w: built-in Agent Collection document is not declared in topology",
			model.ErrInvalid,
		)
	}

	expectations := append(
		[]ArtifactExpectation(nil),
		value.Expectations...,
	)
	sortExpectations(expectations)

	return topology.PackageFingerprint(
		value.EmbeddedPackageRoot,
		value.PackageAddress,
		value.PluginDocumentFile,
		expectations,
		value.PackageFiles,
	)
}

func sortExpectations(
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
		if values[left].LogicalName != values[right].LogicalName {
			return values[left].LogicalName < values[right].LogicalName
		}
		return values[left].LogicalVersion < values[right].LogicalVersion
	})
}
