package toolcatalog

import (
	"context"
	"fmt"
	"io/fs"
	"sort"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	pluginv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/contract/v1"
	toolv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/contract/v1"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/domain"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

type PreparedPackage struct {
	EmbeddedPackageRoot    spec.Locator
	Address                managedpackageModel.ManagedPackageAddress
	DocumentFile           spec.Locator
	PackageFiles           []managedpackageModel.ManagedPackageFile
	ExpectedKind           artifactModel.ArtifactKind
	ExpectedLogicalName    spec.LogicalName
	ExpectedLogicalVersion spec.LogicalVersion
	ExpectedDefinition     cryptoutil.Digest
}

func PreparePackages(
	ctx context.Context,
	packages fs.FS,
	goTools toolDomain.GoToolLocator,
	registry *interpretation.Registry,
) ([]PreparedPackage, error) {
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Tool package preparation context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if registry == nil {
		return nil, fmt.Errorf(
			"%w: Tool package interpretation registry is nil",
			spec.ErrInvalid,
		)
	}
	if packages == nil || goTools == nil {
		return nil, fmt.Errorf(
			"%w: Tool package preparation dependencies are incomplete",
			spec.ErrInvalid,
		)
	}

	roots, err := managedpackage.DirectPackageRoots(packages)
	if err != nil {
		return nil, err
	}

	seenTools := map[spec.LogicalName]spec.Locator{}
	output := make([]PreparedPackage, 0)

	for _, root := range roots {
		values, err := preparePluginDirectory(
			ctx,
			packages,
			root,
			goTools,
			seenTools,
			registry,
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
	pluginRoot spec.Locator,
	goTools toolDomain.GoToolLocator,
	seenTools map[spec.LogicalName]spec.Locator,
	registry *interpretation.Registry,
) ([]PreparedPackage, error) {
	pluginLocation := string(pluginRoot) + "/" +
		string(toolDomain.ToolPluginDocumentFile())
	pluginBytes, err := fs.ReadFile(packages, pluginLocation)
	if err != nil {
		return nil, fmt.Errorf(
			"read Tool Plugin %q: %w",
			pluginRoot,
			err,
		)
	}

	pluginRaw, err := yamlutil.CanonicalObjectJSON(
		pluginBytes,
		spec.MaxDefinitionBytes,
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
	)
	if err != nil {
		return nil, err
	}

	staticTools, err := readStaticSDKTools(
		ctx,
		packages,
		pluginRoot,
		toolNames,
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
				spec.ErrIdentityConflict,
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
		)
		if err != nil {
			return nil, err
		}
		output = append(output, prepared)
	}

	return output, nil
}

func readStaticSDKTools(
	ctx context.Context,
	packages fs.FS,
	pluginRoot spec.Locator,
	declared []spec.LogicalName,
) (map[spec.LogicalName]toolv1.ToolDocument, error) {
	allowed := make(map[spec.LogicalName]struct{}, len(declared))
	for _, name := range declared {
		allowed[name] = struct{}{}
	}

	entries, err := fs.ReadDir(packages, string(pluginRoot))
	if err != nil {
		return nil, err
	}

	output := make(map[spec.LogicalName]toolv1.ToolDocument)
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if entry.Name() == string(toolDomain.ToolPluginDocumentFile()) {
			continue
		}
		if !entry.IsDir() {
			return nil, fmt.Errorf(
				"%w: Tool Plugin directory %q contains unexpected file %q",
				spec.ErrInvalid,
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
			files[0].Name() != string(toolDomain.ToolDocumentFile()) {
			return nil, fmt.Errorf(
				"%w: embedded SDK Tool directory %q must contain only its Tool document",
				spec.ErrInvalid,
				directory,
			)
		}

		location := string(pluginRoot) + "/" + entry.Name() +
			"/" + string(toolDomain.ToolDocumentFile())
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
			spec.MaxDefinitionBytes,
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
				spec.ErrInvalid,
				document.Name,
			)
		}

		name := spec.LogicalName(document.Name)
		if _, found := allowed[name]; !found {
			return nil, fmt.Errorf(
				"%w: embedded SDK Tool %q is not declared by Plugin %q",
				spec.ErrInvalid,
				name,
				pluginRoot,
			)
		}
		if _, duplicate := output[name]; duplicate {
			return nil, fmt.Errorf(
				"%w: Plugin %q repeats embedded SDK Tool %q",
				spec.ErrIdentityConflict,
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
		Implementation: toolv1.ToolImplementation{
			Kind:     toolv1.ImplementationKindGo,
			Function: value.Function,
		},
	}
}

func preparePluginPackage(
	packageRoot spec.Locator,
	document pluginv1.PluginDocument,
	registry *interpretation.Registry,
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
	address, err := toolDomain.ToolPluginPackageAddress(
		spec.LogicalName(document.Name),
	)
	if err != nil {
		return PreparedPackage{}, err
	}

	return PreparedPackage{
		EmbeddedPackageRoot: packageRoot,
		Address:             address,
		DocumentFile:        toolDomain.ToolPluginDocumentFile(),
		PackageFiles: []managedpackageModel.ManagedPackageFile{{
			Locator: toolDomain.ToolPluginDocumentFile(),
			Content: raw,
		}},
		ExpectedKind: artifactModel.ArtifactKind(
			pluginv1.PluginType,
		),
		ExpectedLogicalName:    spec.LogicalName(document.Name),
		ExpectedLogicalVersion: definitionValue.LogicalVersion,
		ExpectedDefinition:     definitionValue.Digest,
	}, nil
}

func prepareToolPackage(
	pluginRoot spec.Locator,
	document toolv1.ToolDocument,
	registry *interpretation.Registry,
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
	address, err := toolDomain.ToolPackageAddress(
		spec.LogicalName(document.Name),
		document.Version,
	)
	if err != nil {
		return PreparedPackage{}, err
	}

	return PreparedPackage{
		EmbeddedPackageRoot: pluginRoot,
		Address:             address,
		DocumentFile:        toolDomain.ToolDocumentFile(),
		PackageFiles: []managedpackageModel.ManagedPackageFile{{
			Locator: toolDomain.ToolDocumentFile(),
			Content: raw,
		}},
		ExpectedKind:           toolDomain.ToolArtifactKind,
		ExpectedLogicalName:    spec.LogicalName(document.Name),
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
				spec.ErrConflict,
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
			Name       spec.LogicalName           `json:"name"`
			Version    spec.LogicalVersion        `json:"version"`
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
