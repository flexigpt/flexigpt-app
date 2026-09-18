package builtin

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	agentConsumerAPI "github.com/flexigpt/flexigpt-app/internal/agent/store/consumerapi"
	agentDomain "github.com/flexigpt/flexigpt-app/internal/agent/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/pluginv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

type PreparedPackage struct {
	EmbeddedPackageRoot basespec.Locator

	PackageAddress source.ManagedPackageAddress

	PluginDocumentFile basespec.Locator

	PackageFiles []source.ManagedPackageFile

	Expectations []agentConsumerAPI.BuiltInAgentArtifactExpectation
}

func (p PreparedPackage) installRequest(
	rootID root.RootID,
	sourceID source.SourceID,
) agentConsumerAPI.BuiltInAgentPackageInstallRequest {
	return agentConsumerAPI.BuiltInAgentPackageInstallRequest{
		RootID:             rootID,
		SourceID:           sourceID,
		PackageAddress:     p.PackageAddress,
		PluginDocumentFile: p.PluginDocumentFile,
		PackageFiles:       clonePackageFiles(p.PackageFiles),
		Expectations: append(
			[]agentConsumerAPI.BuiltInAgentArtifactExpectation(nil),
			p.Expectations...,
		),
	}
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
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if packages == nil {
		return nil, fmt.Errorf(
			"%w: embedded Agent package filesystem is nil",
			basespec.ErrInvalid,
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
	packageRoot basespec.Locator,
) (PreparedPackage, error) {
	files, err := topology.ReadPackageFiles(ctx, packages, packageRoot)
	if err != nil {
		return PreparedPackage{}, err
	}
	files, err = source.NormalizeManagedPackageFiles(files)
	if err != nil {
		return PreparedPackage{}, err
	}

	document, found := builtin.PackageFileContent(
		files,
		agentDomain.BuiltinAgentPluginDocumentFile,
	)
	if !found {
		return PreparedPackage{}, fmt.Errorf(
			"%w: embedded Agent package %q lacks %q",
			basespec.ErrInvalid,
			packageRoot,
			agentDomain.BuiltinAgentPluginDocumentFile,
		)
	}

	_, expectations, err := canonicalCollectionPackage(document, files)
	if err != nil {
		return PreparedPackage{}, fmt.Errorf(
			"decode embedded canonical Agent Plugin %q: %w",
			packageRoot,
			err,
		)
	}

	packageName := basespec.LogicalName(path.Base(string(packageRoot)))
	if err := packageName.Validate(); err != nil {
		return PreparedPackage{}, err
	}
	address, err := source.NewManagedPackageAddress(
		agentDomain.BuiltinAgentCollectionPackageKind,
		packageName,
		builtin.UnversionedPackageVersion,
	)
	if err != nil {
		return PreparedPackage{}, err
	}

	return PreparedPackage{
		EmbeddedPackageRoot: packageRoot,
		PackageAddress:      address,
		PluginDocumentFile:  agentDomain.BuiltinAgentPluginDocumentFile,
		PackageFiles:        files,
		Expectations:        expectations,
	}, nil
}

func canonicalCollectionPackage(
	document []byte,
	files []source.ManagedPackageFile,
) (
	pluginv1.PluginDocument,
	[]agentConsumerAPI.BuiltInAgentArtifactExpectation,
	error,
) {
	raw, err := yamlutil.CanonicalObjectJSON(
		document,
		basespec.MaxDefinitionBytes,
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
			basespec.ErrInvalid,
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
		agentDomain.BuiltinAgentPluginDocumentFile,
		r,
	)
	if err != nil {
		return pluginv1.PluginDocument{}, nil, err
	}

	filesByLocator := make(map[basespec.Locator][]byte, len(files))
	for _, file := range files {
		filesByLocator[file.Locator] = append([]byte(nil), file.Content...)
	}

	seenNames := make(map[basespec.LogicalName]basespec.Locator)
	seenDocuments := make(map[basespec.Locator]struct{})

	for index, member := range collection.Members {
		form, err := member.MemberForm()
		if err != nil {
			return pluginv1.PluginDocument{}, nil, err
		}
		if form != declaration.MemberNamed {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent Plugin member %d must be a named external Agent reference",
				basespec.ErrInvalid,
				index,
			)
		}

		header := member.Header()
		if header.Type != declaration.TypeAgent {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent Plugin member %d has type %q",
				basespec.ErrInvalid,
				index,
				header.Type,
			)
		}
		if header.Locator == nil {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent %q requires a local package locator",
				basespec.ErrInvalid,
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
				basespec.ErrInvalid,
				header.Name,
			)
		}

		documentLocator, err := declaration.ResolveSourceRelativePathLocator(
			*header.Locator,
			agentDomain.BuiltinAgentPluginDocumentFile,
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
				basespec.ErrInvalid,
				header.Name,
				documentName,
			)
		}

		name := basespec.LogicalName(header.Name)
		if previous, duplicate := seenNames[name]; duplicate {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent Plugin references Agent %q at both %q and %q",
				basespec.ErrIdentityConflict,
				name,
				previous,
				documentLocator,
			)
		}
		seenNames[name] = documentLocator

		if _, duplicate := seenDocuments[documentLocator]; duplicate {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent Plugin references document %q more than once",
				basespec.ErrIdentityConflict,
				documentLocator,
			)
		}

		content, found := filesByLocator[documentLocator]
		if !found {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent %q locator does not identify packaged %q",
				basespec.ErrInvalid,
				header.Name,
				agentDomain.BuiltinAgentDocumentFile,
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
				basespec.ErrInvalid,
				header.Name,
			)
		}
		if agentDocument.Name != header.Name {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent document name differs from Plugin member %q",
				basespec.ErrInvalid,
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
		if path.Base(string(locator)) != string(agentDomain.BuiltinAgentDocumentFile) {
			continue
		}
		if _, err := packageAgentDocumentName(locator); err != nil {
			return pluginv1.PluginDocument{}, nil, err
		}
		if _, found := seenDocuments[locator]; !found {
			return pluginv1.PluginDocument{}, nil, fmt.Errorf(
				"%w: built-in Agent document %q is not referenced by Plugin %q",
				basespec.ErrInvalid,
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
		basespec.MaxDefinitionBytes,
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
			basespec.ErrInvalid,
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
	locator basespec.Locator,
	entry declaration.Entry,
) ([]agentConsumerAPI.BuiltInAgentArtifactExpectation, error) {
	namedEntries, err := declaration.WalkNamedEntries(entry)
	if err != nil {
		return nil, err
	}

	output := make(
		[]agentConsumerAPI.BuiltInAgentArtifactExpectation,
		0,
		len(namedEntries),
	)
	for _, named := range namedEntries {
		value, err := decoder.DefinitionForNamedEntry(named)
		if err != nil {
			return nil, err
		}
		output = append(output, agentConsumerAPI.BuiltInAgentArtifactExpectation{
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
	locator basespec.Locator,
) (basespec.LogicalName, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return "", err
	}

	segments := strings.Split(string(locator), "/")
	if len(segments) != 3 ||
		segments[0] != "agents" ||
		segments[2] != string(agentDomain.BuiltinAgentDocumentFile) {
		return "", fmt.Errorf(
			"%w: built-in Agent document %q must use agents/<name>/%s",
			basespec.ErrInvalid,
			locator,
			agentDomain.BuiltinAgentDocumentFile,
		)
	}

	name := basespec.LogicalName(segments[1])
	if err := name.Validate(); err != nil {
		return "", err
	}
	return name, nil
}

type preparedArtifactIdentity struct {
	kind    artifact.ArtifactKind
	name    basespec.LogicalName
	version basespec.LogicalVersion
}

func validatePreparedPackageIdentities(
	packages []PreparedPackage,
) error {
	seen := make(map[preparedArtifactIdentity]basespec.Locator)

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
					basespec.ErrConflict,
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
	type file struct {
		Locator basespec.Locator  `json:"locator"`
		Digest  cryptoutil.Digest `json:"digest"`
		Size    int64             `json:"size"`
	}

	if err := value.PackageAddress.Validate(); err != nil {
		return "", err
	}
	if value.PluginDocumentFile != agentDomain.BuiltinAgentPluginDocumentFile {
		return "", fmt.Errorf(
			"%w: built-in Agent Plugin document must be %q",
			basespec.ErrInvalid,
			agentDomain.BuiltinAgentPluginDocumentFile,
		)
	}

	packageFiles, err := source.NormalizeManagedPackageFiles(
		value.PackageFiles,
	)
	if err != nil {
		return "", err
	}

	files := make([]file, 0, len(packageFiles))
	for _, item := range packageFiles {
		files = append(files, file{
			Locator: item.Locator,
			Digest:  cryptoutil.DigestBytes(item.Content),
			Size:    int64(len(item.Content)),
		})
	}
	sort.Slice(files, func(left, right int) bool {
		return files[left].Locator < files[right].Locator
	})

	expectations := append(
		[]agentConsumerAPI.BuiltInAgentArtifactExpectation(nil),
		value.Expectations...,
	)
	sortExpectations(expectations)

	return cryptoutil.CanonicalDigest(struct {
		PackageRoot        basespec.Locator                                   `json:"packageRoot"`
		Address            source.ManagedPackageAddress                       `json:"address"`
		PluginDocumentFile basespec.Locator                                   `json:"pluginDocumentFile"`
		Expectations       []agentConsumerAPI.BuiltInAgentArtifactExpectation `json:"expectations"`
		Files              []file                                             `json:"files"`
	}{
		PackageRoot:        value.EmbeddedPackageRoot,
		Address:            value.PackageAddress,
		PluginDocumentFile: value.PluginDocumentFile,
		Expectations:       expectations,
		Files:              files,
	})
}

func sortExpectations(
	values []agentConsumerAPI.BuiltInAgentArtifactExpectation,
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

func clonePackageFiles(
	values []source.ManagedPackageFile,
) []source.ManagedPackageFile {
	output := make([]source.ManagedPackageFile, len(values))
	for index, value := range values {
		output[index] = source.ManagedPackageFile{
			Locator: value.Locator,
			Content: append([]byte(nil), value.Content...),
		}
	}
	return output
}
