package builtin

import (
	"context"
	"fmt"
	"io/fs"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/pluginv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/toolv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/api/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/tool/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

type PreparedPackage struct {
	EmbeddedPackageRoot    model.Locator
	Address                source.ManagedPackageAddress
	DocumentFile           model.Locator
	PackageFiles           []source.ManagedPackageFile
	ExpectedKind           artifact.ArtifactKind
	ExpectedLogicalName    model.LogicalName
	ExpectedLogicalVersion model.LogicalVersion
	ExpectedDefinition     cryptoutil.Digest
}

func PreparePackages(
	ctx context.Context,
	packages fs.FS,
	goTools toolDomain.GoToolLocator,
) ([]PreparedPackage, error) {
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Tool package preparation context is nil",
			model.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if packages == nil || goTools == nil {
		return nil, fmt.Errorf(
			"%w: Tool package preparation dependencies are incomplete",
			model.ErrInvalid,
		)
	}

	roots, err := builtin.DirectPackageRoots(packages)
	if err != nil {
		return nil, err
	}

	seenTools := map[model.LogicalName]model.Locator{}
	output := make([]PreparedPackage, 0)

	for _, root := range roots {
		values, err := prepareCollectionDirectory(
			ctx,
			packages,
			root,
			goTools,
			seenTools,
		)
		if err != nil {
			return nil, err
		}
		output = append(output, values...)
	}

	return normalizePreparedPackages(output)
}

func prepareCollectionDirectory(
	ctx context.Context,
	packages fs.FS,
	collectionRoot model.Locator,
	goTools toolDomain.GoToolLocator,
	seenTools map[model.LogicalName]model.Locator,
) ([]PreparedPackage, error) {
	pluginLocation := string(collectionRoot) + "/" +
		string(toolDomain.ToolCollectionDocumentFile())
	pluginBytes, err := fs.ReadFile(packages, pluginLocation)
	if err != nil {
		return nil, fmt.Errorf(
			"read Tool Collection %q: %w",
			collectionRoot,
			err,
		)
	}

	pluginRaw, err := yamlutil.CanonicalObjectJSON(
		pluginBytes,
		model.MaxDefinitionBytes,
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
	toolNames, err := toolDomain.ValidateToolCollectionDocument(
		pluginDocument,
	)
	if err != nil {
		return nil, err
	}

	collectionPackage, err := prepareCollectionPackage(
		collectionRoot,
		pluginDocument,
	)
	if err != nil {
		return nil, err
	}

	staticTools, err := readStaticSDKTools(
		ctx,
		packages,
		collectionRoot,
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
	output = append(output, collectionPackage)

	for _, name := range toolNames {
		if previous, duplicate := seenTools[name]; duplicate {
			return nil, fmt.Errorf(
				"%w: Tool %q belongs to both Collection directories %q and %q",
				model.ErrIdentityConflict,
				name,
				previous,
				collectionRoot,
			)
		}
		seenTools[name] = collectionRoot

		document, found := staticTools[name]
		if !found {
			goDescriptor, err := goTools.LookupGoToolByName(ctx, name)
			if err != nil {
				return nil, fmt.Errorf(
					"tool collection %q references Tool %q that is neither embedded SDK Tool nor registered Go Tool: %w",
					pluginDocument.Name,
					name,
					err,
				)
			}
			document = toolDocumentFromGoDescriptor(goDescriptor)
		}

		prepared, err := prepareToolPackage(
			collectionRoot,
			document,
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
	collectionRoot model.Locator,
	declared []model.LogicalName,
) (map[model.LogicalName]toolv1.ToolDocument, error) {
	allowed := make(map[model.LogicalName]struct{}, len(declared))
	for _, name := range declared {
		allowed[name] = struct{}{}
	}

	entries, err := fs.ReadDir(packages, string(collectionRoot))
	if err != nil {
		return nil, err
	}

	output := make(map[model.LogicalName]toolv1.ToolDocument)
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if entry.Name() == string(toolDomain.ToolCollectionDocumentFile()) {
			continue
		}
		if !entry.IsDir() {
			return nil, fmt.Errorf(
				"%w: Tool Collection directory %q contains unexpected file %q",
				model.ErrInvalid,
				collectionRoot,
				entry.Name(),
			)
		}

		directory := string(collectionRoot) + "/" + entry.Name()
		files, err := fs.ReadDir(packages, directory)
		if err != nil {
			return nil, err
		}
		if len(files) != 1 ||
			files[0].IsDir() ||
			files[0].Name() != string(toolDomain.ToolDocumentFile()) {
			return nil, fmt.Errorf(
				"%w: embedded SDK Tool directory %q must contain only its Tool document",
				model.ErrInvalid,
				directory,
			)
		}

		location := string(collectionRoot) + "/" + entry.Name() +
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
			model.MaxDefinitionBytes,
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
				model.ErrInvalid,
				document.Name,
			)
		}

		name := model.LogicalName(document.Name)
		if _, found := allowed[name]; !found {
			return nil, fmt.Errorf(
				"%w: embedded SDK Tool %q is not declared by Collection %q",
				model.ErrInvalid,
				name,
				collectionRoot,
			)
		}
		if _, duplicate := output[name]; duplicate {
			return nil, fmt.Errorf(
				"%w: Collection %q repeats embedded SDK Tool %q",
				model.ErrIdentityConflict,
				collectionRoot,
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

func prepareCollectionPackage(
	packageRoot model.Locator,
	document pluginv1.PluginDocument,
) (PreparedPackage, error) {
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
	address, err := toolDomain.ToolCollectionPackageAddress(
		model.LogicalName(document.Name),
	)
	if err != nil {
		return PreparedPackage{}, err
	}

	return PreparedPackage{
		EmbeddedPackageRoot: packageRoot,
		Address:             address,
		DocumentFile:        toolDomain.ToolCollectionDocumentFile(),
		PackageFiles: []source.ManagedPackageFile{{
			Locator: toolDomain.ToolCollectionDocumentFile(),
			Content: raw,
		}},
		ExpectedKind: artifact.ArtifactKind(
			pluginv1.PluginType,
		),
		ExpectedLogicalName:    model.LogicalName(document.Name),
		ExpectedLogicalVersion: definitionValue.LogicalVersion,
		ExpectedDefinition:     definitionValue.Digest,
	}, nil
}

func prepareToolPackage(
	collectionRoot model.Locator,
	document toolv1.ToolDocument,
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
	definitionValue, err := decoder.DefinitionForEntry(entry)
	if err != nil {
		return PreparedPackage{}, err
	}
	address, err := toolDomain.ToolPackageAddress(
		model.LogicalName(document.Name),
		document.Version,
	)
	if err != nil {
		return PreparedPackage{}, err
	}

	return PreparedPackage{
		EmbeddedPackageRoot: collectionRoot,
		Address:             address,
		DocumentFile:        toolDomain.ToolDocumentFile(),
		PackageFiles: []source.ManagedPackageFile{{
			Locator: toolDomain.ToolDocumentFile(),
			Content: raw,
		}},
		ExpectedKind:           toolDomain.ToolArtifactKind,
		ExpectedLogicalName:    model.LogicalName(document.Name),
		ExpectedLogicalVersion: definitionValue.LogicalVersion,
		ExpectedDefinition:     definitionValue.Digest,
	}, nil
}

func normalizePreparedPackages(
	values []PreparedPackage,
) ([]PreparedPackage, error) {
	seen := make(map[source.ManagedPackageAddress]struct{}, len(values))
	output := make([]PreparedPackage, len(values))

	for index, value := range values {
		if err := value.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := seen[value.Address]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate Tool package %q",
				model.ErrConflict,
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
	if _, err := source.NormalizeManagedPackageFiles(
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
	return topology.PackageFingerprint(
		p.EmbeddedPackageRoot,
		p.Address,
		p.DocumentFile,
		struct {
			Kind       artifact.ArtifactKind `json:"kind"`
			Name       model.LogicalName     `json:"name"`
			Version    model.LogicalVersion  `json:"version"`
			Definition cryptoutil.Digest     `json:"definition"`
		}{
			Kind:       p.ExpectedKind,
			Name:       p.ExpectedLogicalName,
			Version:    p.ExpectedLogicalVersion,
			Definition: p.ExpectedDefinition,
		},
		p.PackageFiles,
	)
}
