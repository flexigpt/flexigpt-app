package agentcatalog

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	agentv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/contract/v1"
	agentDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/domain"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	pluginv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/contract/v1"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

type PreparedPackage struct {
	EmbeddedPackageRoot spec.Locator

	PackageAddress managedpackageModel.ManagedPackageAddress

	PluginDocumentFile spec.Locator

	PackageFiles []managedpackageModel.ManagedPackageFile

	Expectations []ArtifactExpectation
}

// ArtifactExpectation is build-time package admission data. Runtime Agent
// consumers do not own embedded package validation.
type ArtifactExpectation struct {
	Locator          spec.Locator
	Subresource      spec.SubresourceLocator
	Kind             artifactModel.ArtifactKind
	LogicalName      spec.LogicalName
	LogicalVersion   spec.LogicalVersion
	DefinitionDigest cryptoutil.Digest
}

// PreparePackages reads direct embedded Agent Plugin package directories.
// Every package contains one canonical Plugin document and each direct Plugin
// member identifies one concrete package-local Agent declaration.
func PreparePackages(
	ctx context.Context,
	packages fs.FS,
	interpretations *coreinterpretation.Registry,
) ([]PreparedPackage, error) {
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: built-in Agent package preparation context is nil",
			spec.ErrInvalid,
		)
	}
	if interpretations == nil {
		return nil, fmt.Errorf("%w: Agent package interpretation registry is nil", spec.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if packages == nil {
		return nil, fmt.Errorf(
			"%w: embedded Agent package filesystem is nil",
			spec.ErrInvalid,
		)
	}

	roots, err := managedpackage.DirectPackageRoots(packages)
	if err != nil {
		return nil, err
	}

	output := make([]PreparedPackage, 0, len(roots))
	for _, packageRoot := range roots {
		value, err := preparePackage(ctx, packages, packageRoot, interpretations)
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
	interpretations *coreinterpretation.Registry,
) (PreparedPackage, error) {
	files, err := managedpackage.ReadPackageFiles(ctx, packages, packageRoot)
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
			"%w: embedded Agent package %q lacks a supported Plugin document",
			spec.ErrInvalid,
			packageRoot,
		)
	}

	_, expectations, err := canonicalPluginPackage(
		documentFile,
		document,
		files,
		interpretations,
	)
	if err != nil {
		return PreparedPackage{}, fmt.Errorf(
			"decode embedded canonical Agent Plugin %q: %w",
			packageRoot,
			err,
		)
	}

	packageName := spec.LogicalName(path.Base(string(packageRoot)))
	if err := packageName.Validate(); err != nil {
		return PreparedPackage{}, err
	}
	address, err := managedpackageModel.NewManagedPackageAddress(
		agentDomain.BuiltinAgentPluginPackageKind,
		packageName,
		topology.UnversionedPackageVersion(),
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

func canonicalPluginPackage(
	documentFile spec.Locator,
	document []byte,
	files []managedpackageModel.ManagedPackageFile,
	interpretations *coreinterpretation.Registry,
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

	r, err := declaration.DecodeCanonicalEntryJSON(raw)
	if err != nil {
		return pluginv1.PluginDocument{}, nil, err
	}
	if r.Header().Type != declaration.TypePlugin {
		return pluginv1.PluginDocument{}, nil, fmt.Errorf(
			"%w: built-in Agent package root must be a Plugin",
			spec.ErrInvalid,
		)
	}
	if err := interpretations.ValidateTree(r); err != nil {
		return pluginv1.PluginDocument{}, nil, err
	}

	plugin, err := pluginv1.DecodePluginEntry(r)
	if err != nil {
		return pluginv1.PluginDocument{}, nil, err
	}

	expectations, err := expectationsForDocument(
		documentFile,
		r,
		interpretations,
	)
	if err != nil {
		return pluginv1.PluginDocument{}, nil, err
	}

	filesByLocator := make(map[spec.Locator][]byte, len(files))
	for _, file := range files {
		filesByLocator[file.Locator] = append([]byte(nil), file.Content...)
	}

	seenNames := make(map[spec.LogicalName]spec.Locator)
	seenDocuments := make(map[spec.Locator]struct{})

	for index, member := range plugin.Members {
		form, err := member.MemberForm()
		if err != nil {
			return pluginv1.PluginDocument{}, nil, err
		}
		if form != declaration.MemberNamed {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent Plugin member %d must be a named external Agent reference",
				spec.ErrInvalid,
				index,
			)
		}

		header := member.Header()
		if header.Type != declaration.TypeAgent {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent Plugin member %d has type %q",
				spec.ErrInvalid,
				index,
				header.Type,
			)
		}
		if header.Locator == nil {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent %q requires a local package locator",
				spec.ErrInvalid,
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
				spec.ErrInvalid,
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
				spec.ErrInvalid,
				header.Name,
				documentName,
			)
		}

		name := spec.LogicalName(header.Name)
		if previous, duplicate := seenNames[name]; duplicate {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent Plugin references Agent %q at both %q and %q",
				spec.ErrIdentityConflict,
				name,
				previous,
				documentLocator,
			)
		}
		seenNames[name] = documentLocator

		if _, duplicate := seenDocuments[documentLocator]; duplicate {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent Plugin references document %q more than once",
				spec.ErrIdentityConflict,
				documentLocator,
			)
		}

		content, found := filesByLocator[documentLocator]
		if !found {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent %q locator does not identify a configured packaged Agent document",
				spec.ErrInvalid,
				header.Name,
			)
		}

		agentRoot, agentDocument, err := canonicalAgentDocument(content, interpretations)
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
				spec.ErrInvalid,
				header.Name,
			)
		}
		if agentDocument.Name != header.Name {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent document name differs from Plugin member %q",
				spec.ErrInvalid,
				header.Name,
			)
		}

		agentExpectations, err := expectationsForDocument(
			documentLocator,
			agentRoot,
			interpretations,
		)
		if err != nil {
			return pluginv1.PluginDocument{}, nil, err
		}

		seenDocuments[documentLocator] = struct{}{}
		expectations = append(expectations, agentExpectations...)
	}

	for locator := range filesByLocator {
		documentFile := spec.Locator(path.Base(string(locator)))
		if !agentDomain.IsAgentDeclarationDocument(documentFile) {
			continue
		}
		if _, err := packageAgentDocumentName(locator); err != nil {
			return pluginv1.PluginDocument{}, nil, err
		}
		if _, found := seenDocuments[locator]; !found {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent document %q is not referenced by Plugin %q",
				spec.ErrInvalid,
				locator,
				plugin.Name,
			)
		}
	}

	sortExpectations(expectations)
	return plugin, expectations, nil
}

func canonicalAgentDocument(
	content []byte,
	interpretations *coreinterpretation.Registry,
) (declaration.Entry, agentv1.AgentDocument, error) {
	raw, err := yamlutil.CanonicalObjectJSON(
		content,
		spec.MaxDefinitionBytes,
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
			spec.ErrInvalid,
			declaration.TypeAgent,
		)
	}
	if err := interpretations.ValidateTree(r); err != nil {
		return declaration.Entry{}, agentv1.AgentDocument{}, err
	}
	document, err := agentv1.DecodeAgentEntry(r)
	if err != nil {
		return declaration.Entry{}, agentv1.AgentDocument{}, err
	}
	return r, document, nil
}

func expectationsForDocument(
	locator spec.Locator,
	entry declaration.Entry,
	interpretations *coreinterpretation.Registry,
) ([]ArtifactExpectation, error) {
	namedEntries, err := interpretations.WalkNamedEntries(entry)
	if err != nil {
		return nil, err
	}

	output := make(
		[]ArtifactExpectation,
		0,
		len(namedEntries),
	)
	for _, named := range namedEntries {
		value, err := interpretations.DefinitionForEntry(named.Entry)
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
	locator spec.Locator,
) (spec.LogicalName, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return "", err
	}

	segments := strings.Split(string(locator), "/")
	if len(segments) != 2 ||
		!agentDomain.IsAgentDeclarationDocument(
			spec.Locator(segments[1]),
		) {
		return "", fmt.Errorf(
			"%w: built-in Agent document %q must use <name>/<configured-agent-document>",
			spec.ErrInvalid,
			locator,
		)
	}

	name := spec.LogicalName(segments[0])
	if err := name.Validate(); err != nil {
		return "", err
	}
	return name, nil
}

type preparedArtifactIdentity struct {
	kind    artifactModel.ArtifactKind
	name    spec.LogicalName
	version spec.LogicalVersion
}

func validatePreparedPackageIdentities(
	packages []PreparedPackage,
) error {
	seen := make(map[preparedArtifactIdentity]spec.Locator)

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
					spec.ErrConflict,
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
	if !topology.IsPluginDocumentFile(value.PluginDocumentFile) {
		return "", fmt.Errorf(
			"%w: built-in Agent Plugin document is not declared in topology",
			spec.ErrInvalid,
		)
	}

	expectations := append(
		[]ArtifactExpectation(nil),
		value.Expectations...,
	)
	sortExpectations(expectations)

	return installModel.PackageFingerprint(
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
