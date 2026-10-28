package toolcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
	pluginv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/contract/v1"
	toolAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool"
	toolv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/contract/v1"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/domain"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

type PreparedPackage struct {
	EmbeddedPackageRoot    storeSpec.Locator
	Address                managedpackageModel.ManagedPackageAddress
	DocumentFile           storeSpec.Locator
	PackageFiles           []managedpackageModel.ManagedPackageFile
	ExpectedKind           artifactModel.ArtifactKind
	ExpectedLogicalName    storeSpec.LogicalName
	ExpectedLogicalVersion storeSpec.LogicalVersion
	ExpectedDefinition     cryptoutil.Digest
}

// PreparePackages compiles both Tool implementation families.
//
// A Plugin member first resolves to a static SDK Tool declaration when one is
// present in the embedded Plugin directory. Otherwise it resolves to a Go
// Tool descriptor supplied by the application runtime adapter. Both forms
// become ordinary source-backed Tool Artifacts and share Plugin membership,
// enablement, catalog, and composition behavior.
func PreparePackages(
	ctx context.Context,
	packages fs.FS,
	goTools toolDomain.GoToolLocator,
	registry *coreinterpretation.Registry,
	s toolAPI.Support,
) ([]PreparedPackage, error) {
	if registry == nil {
		return nil, fmt.Errorf(
			"%w: Tool package interpretation registry is nil",
			storeSpec.ErrInvalid,
		)
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	if packages == nil || goTools == nil {
		return nil, fmt.Errorf(
			"%w: Tool package preparation dependencies are incomplete",
			storeSpec.ErrInvalid,
		)
	}

	roots, err := managedpackage.DirectPackageRoots(packages)
	if err != nil {
		return nil, err
	}

	seenTools := map[storeSpec.LogicalName]storeSpec.Locator{}
	output := make([]PreparedPackage, 0)

	for _, root := range roots {
		values, err := preparePluginDirectory(
			ctx,
			packages,
			root,
			goTools,
			seenTools,
			registry,
			s,
		)
		if err != nil {
			return nil, err
		}
		output = append(output, values...)
	}

	return normalizePreparedPackages(output)
}

func preparePluginDirectory(
	ctx context.Context,
	packages fs.FS,
	pluginRoot storeSpec.Locator,
	goTools toolDomain.GoToolLocator,
	seenTools map[storeSpec.LogicalName]storeSpec.Locator,
	registry *coreinterpretation.Registry,
	s toolAPI.Support,
) ([]PreparedPackage, error) {
	pluginDocumentFile, pluginBytes, err := readPluginDocument(
		packages,
		pluginRoot,
		s.PluginDocuments,
	)
	if err != nil {
		return nil, err
	}

	pluginRaw, err := yamlutil.CanonicalObjectJSON(
		pluginBytes,
		storeSpec.MaxDefinitionBytes,
	)
	if err != nil {
		return nil, err
	}
	pluginEntry, err := declaration.DecodeCanonicalEntryJSON(pluginRaw)
	if err != nil {
		return nil, err
	}
	pluginDocument, err := pluginv1.DecodePluginEntry(pluginEntry)
	if err != nil {
		return nil, err
	}
	toolNames, err := toolDomain.ValidateToolPluginDocument(
		pluginDocument,
	)
	if err != nil {
		return nil, err
	}

	pluginPackage, err := preparePluginPackage(
		pluginRoot,
		pluginDocument,
		registry,
		s.PluginProfile.Package,
	)
	if err != nil {
		return nil, err
	}

	staticTools, err := readStaticSDKTools(
		packages,
		pluginRoot,
		toolNames,
		pluginDocumentFile,
		s.Documents,
	)
	if err != nil {
		return nil, err
	}

	output := make(
		[]PreparedPackage,
		0,
		1+len(toolNames),
	)
	output = append(output, pluginPackage)

	for _, name := range toolNames {
		if previous, duplicate := seenTools[name]; duplicate {
			return nil, fmt.Errorf(
				"%w: Tool %q belongs to both Plugin directories %q and %q",
				storeSpec.ErrIdentityConflict,
				name,
				previous,
				pluginRoot,
			)
		}
		seenTools[name] = pluginRoot

		document, found := staticTools[name]
		if !found {
			goDescriptor, err := goTools.LookupGoToolByName(ctx, name)
			if err != nil {
				return nil, fmt.Errorf(
					"tool plugin %q references Tool %q that is neither embedded SDK Tool nor registered Go Tool: %w",
					pluginDocument.Name,
					name,
					err,
				)
			}
			document = toolDocumentFromGoDescriptor(goDescriptor)
		}

		prepared, err := prepareToolPackage(
			pluginRoot,
			document,
			registry,
			s.ToolPackage,
		)
		if err != nil {
			return nil, err
		}
		output = append(output, prepared)
	}

	return output, nil
}

func readPluginDocument(
	packages fs.FS,
	pluginRoot storeSpec.Locator,
	documents support.Documents,
) (storeSpec.Locator, []byte, error) {
	entries, err := fs.ReadDir(packages, string(pluginRoot))
	if err != nil {
		return "", nil, err
	}

	var documentFile storeSpec.Locator
	for _, entry := range entries {
		if entry.IsDir() || !documents.Matches(storeSpec.Locator(entry.Name())) {
			continue
		}
		if documentFile != "" {
			return "", nil, fmt.Errorf(
				"%w: Tool Plugin directory %q contains multiple configured Plugin documents",
				storeSpec.ErrIdentityConflict,
				pluginRoot,
			)
		}
		documentFile = storeSpec.Locator(entry.Name())
	}
	if documentFile == "" {
		return "", nil, fmt.Errorf(
			"%w: Tool Plugin directory %q lacks a configured Plugin document",
			storeSpec.ErrInvalid,
			pluginRoot,
		)
	}

	content, err := fs.ReadFile(
		packages,
		string(pluginRoot)+"/"+string(documentFile),
	)
	if err != nil {
		return "", nil, fmt.Errorf(
			"read Tool Plugin %q: %w",
			pluginRoot,
			err,
		)
	}
	return documentFile, content, nil
}

// readStaticSDKTools reads provider-native SDK Tool declarations packaged
// below one Tool Plugin directory. Go Tool declarations are generated from
// GoToolLocator and are intentionally not read from static files.
func readStaticSDKTools(
	packages fs.FS,
	pluginRoot storeSpec.Locator,
	declared []storeSpec.LogicalName,
	pluginDocumentFile storeSpec.Locator,
	documents support.Documents,
) (map[storeSpec.LogicalName]toolv1.ToolDocument, error) {
	allowed := make(map[storeSpec.LogicalName]struct{}, len(declared))
	for _, name := range declared {
		allowed[name] = struct{}{}
	}

	entries, err := fs.ReadDir(packages, string(pluginRoot))
	if err != nil {
		return nil, err
	}

	output := make(map[storeSpec.LogicalName]toolv1.ToolDocument)
	for _, entry := range entries {
		if entry.Name() == string(pluginDocumentFile) {
			continue
		}
		if !entry.IsDir() {
			return nil, fmt.Errorf(
				"%w: Tool Plugin directory %q contains unexpected file %q",
				storeSpec.ErrInvalid,
				pluginRoot,
				entry.Name(),
			)
		}

		directory := string(pluginRoot) + "/" + entry.Name()
		files, err := fs.ReadDir(packages, directory)
		if err != nil {
			return nil, err
		}
		if len(files) != 1 ||
			files[0].IsDir() ||
			!documents.Matches(storeSpec.Locator(files[0].Name())) {
			return nil, fmt.Errorf(
				"%w: embedded SDK Tool directory %q must contain only one configured Tool document",
				storeSpec.ErrInvalid,
				directory,
			)
		}

		location := string(pluginRoot) + "/" + entry.Name() +
			"/" + files[0].Name()
		rawDocument, err := fs.ReadFile(packages, location)
		if err != nil {
			return nil, fmt.Errorf(
				"read embedded SDK Tool %q: %w",
				location,
				err,
			)
		}
		raw, err := yamlutil.CanonicalObjectJSON(
			rawDocument,
			storeSpec.MaxDefinitionBytes,
		)
		if err != nil {
			return nil, err
		}
		entryValue, err := declaration.DecodeCanonicalEntryJSON(raw)
		if err != nil {
			return nil, err
		}
		document, err := toolv1.DecodeToolEntry(entryValue)
		if err != nil {
			return nil, err
		}
		if document.Implementation.Kind != toolv1.ImplementationKindSDK {
			return nil, fmt.Errorf(
				"%w: embedded Tool %q must use sdk implementation",
				storeSpec.ErrInvalid,
				document.Name,
			)
		}

		name := storeSpec.LogicalName(document.Name)
		if _, found := allowed[name]; !found {
			return nil, fmt.Errorf(
				"%w: embedded SDK Tool %q is not declared by Plugin %q",
				storeSpec.ErrInvalid,
				name,
				pluginRoot,
			)
		}
		if _, duplicate := output[name]; duplicate {
			return nil, fmt.Errorf(
				"%w: Plugin %q repeats embedded SDK Tool %q",
				storeSpec.ErrIdentityConflict,
				pluginRoot,
				name,
			)
		}
		output[name] = document
	}

	return output, nil
}

func toolDocumentFromGoDescriptor(
	value toolDomain.GoToolDescriptor,
) toolv1.ToolDocument {
	var outSchema json.RawMessage = append(
		[]byte(nil),
		value.OutputSchema...,
	)
	return toolv1.ToolDocument{
		Type:        toolv1.ToolType,
		Name:        string(value.Name),
		DisplayName: value.DisplayName,
		Description: value.Description,

		Version:     value.Version,
		Tags:        append([]string(nil), value.Tags...),
		AutoExecute: value.AutoExecute,
		InputSchema: append(
			[]byte(nil),
			value.InputSchema...,
		),
		OutputSchema: &outSchema,
		Implementation: toolv1.ToolImplementation{
			Kind:     toolv1.ImplementationKindGo,
			Function: value.Function,
		},
	}
}

func preparePluginPackage(
	packageRoot storeSpec.Locator,
	document pluginv1.PluginDocument,
	registry *coreinterpretation.Registry,
	layout support.PackageLayout,
) (PreparedPackage, error) {
	raw, err := document.CanonicalJSON()
	if err != nil {
		return PreparedPackage{}, err
	}
	entry, err := declaration.NewEntry(document)
	if err != nil {
		return PreparedPackage{}, err
	}
	definitionValue, err := registry.DefinitionForEntry(entry)
	if err != nil {
		return PreparedPackage{}, err
	}
	address, err := layout.Address(
		storeSpec.LogicalName(document.Name),
		"",
	)
	if err != nil {
		return PreparedPackage{}, err
	}

	return PreparedPackage{
		EmbeddedPackageRoot: packageRoot,
		Address:             address,
		DocumentFile:        layout.Document.Locator,
		PackageFiles: []managedpackageModel.ManagedPackageFile{{
			Locator: layout.Document.Locator,
			Content: raw,
		}},
		ExpectedKind: artifactModel.ArtifactKind(
			pluginv1.PluginType,
		),
		ExpectedLogicalName:    storeSpec.LogicalName(document.Name),
		ExpectedLogicalVersion: definitionValue.LogicalVersion,
		ExpectedDefinition:     definitionValue.Digest,
	}, nil
}

func prepareToolPackage(
	pluginRoot storeSpec.Locator,
	document toolv1.ToolDocument,
	registry *coreinterpretation.Registry,
	layout support.PackageLayout,
) (PreparedPackage, error) {
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
	definitionValue, err := registry.DefinitionForEntry(entry)
	if err != nil {
		return PreparedPackage{}, err
	}
	address, err := layout.Address(
		storeSpec.LogicalName(document.Name),
		document.Version,
	)
	if err != nil {
		return PreparedPackage{}, err
	}

	return PreparedPackage{
		EmbeddedPackageRoot: pluginRoot,
		Address:             address,
		DocumentFile:        layout.Document.Locator,
		PackageFiles: []managedpackageModel.ManagedPackageFile{{
			Locator: layout.Document.Locator,
			Content: raw,
		}},
		ExpectedKind:           toolDomain.ToolArtifactKind,
		ExpectedLogicalName:    storeSpec.LogicalName(document.Name),
		ExpectedLogicalVersion: definitionValue.LogicalVersion,
		ExpectedDefinition:     definitionValue.Digest,
	}, nil
}

func normalizePreparedPackages(
	values []PreparedPackage,
) ([]PreparedPackage, error) {
	seen := make(map[managedpackageModel.ManagedPackageAddress]struct{}, len(values))
	output := make([]PreparedPackage, len(values))

	for index, value := range values {
		if err := value.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := seen[value.Address]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate Tool package %q",
				storeSpec.ErrConflict,
				value.Address,
			)
		}
		seen[value.Address] = struct{}{}
		output[index] = value
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

func (p PreparedPackage) Fingerprint() (
	cryptoutil.Digest,
	error,
) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	return installModel.PackageFingerprint(
		p.EmbeddedPackageRoot,
		p.Address,
		p.DocumentFile,
		struct {
			Kind       artifactModel.ArtifactKind `json:"kind"`
			Name       storeSpec.LogicalName      `json:"name"`
			Version    storeSpec.LogicalVersion   `json:"version"`
			Definition cryptoutil.Digest          `json:"definition"`
		}{
			Kind:       p.ExpectedKind,
			Name:       p.ExpectedLogicalName,
			Version:    p.ExpectedLogicalVersion,
			Definition: p.ExpectedDefinition,
		},
		p.PackageFiles,
	)
}
