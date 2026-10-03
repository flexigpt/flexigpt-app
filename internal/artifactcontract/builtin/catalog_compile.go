package builtin

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/providercanonical"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/compose/local"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider"
	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// Expectation is the declaration admission result expected from one source
// document or subresource.
type Expectation struct {
	Locator          spec.Locator
	Subresource      spec.SubresourceLocator
	Kind             artifactModel.ArtifactKind
	LogicalName      spec.LogicalName
	LogicalVersion   spec.LogicalVersion
	DefinitionDigest cryptoutil.Digest
}

// PackageInput is one prepared managed package. Package preparation remains
// domain-owned. This package only performs generic Artifact Store admission.
type PackageInput struct {
	EmbeddedRoot spec.Locator
	Address      sourceModel.ManagedPackageAddress
	DocumentFile spec.Locator
	Files        []sourceModel.ManagedPackageFile
	Expectations []Expectation
}

// Config identifies one domain-owned generated built-in package set.
//
// AdditionalProviders excludes the canonical declaration provider. Compile
// always installs that provider because every built-in declaration requires it.
type Config struct {
	SetName             string
	SchemaVersion       string
	InstallerName       string
	AdditionalProviders []provider.Provider
	Packages            []PackageInput
}

// Compile admits one domain-owned built-in package set through the ordinary
// managed Source path and returns the generated runtime contract.
//
// It deliberately does not inspect domain package formats. Those rules remain
// in the owning domain's PreparePackages implementation.
func Compile(
	ctx context.Context,
	temporaryDirectory string,
	config Config,
) (installModel.CompiledPackageSet, error) {
	if ctx == nil {
		return installModel.CompiledPackageSet{}, fmt.Errorf(
			"%w: built-in catalog compilation context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return installModel.CompiledPackageSet{}, err
	}
	if temporaryDirectory == "" {
		return installModel.CompiledPackageSet{}, fmt.Errorf(
			"%w: built-in catalog compilation directory is empty",
			spec.ErrInvalid,
		)
	}
	if config.SetName == "" {
		return installModel.CompiledPackageSet{}, fmt.Errorf(
			"%w: built-in catalog set name is empty",
			spec.ErrInvalid,
		)
	}
	if config.SchemaVersion == "" {
		return installModel.CompiledPackageSet{}, fmt.Errorf(
			"%w: built-in catalog schema version is empty",
			spec.ErrInvalid,
		)
	}
	if err := installModel.ValidateHydrationInstallerName(
		config.InstallerName,
	); err != nil {
		return installModel.CompiledPackageSet{}, fmt.Errorf(
			"built-in catalog installer name: %w",
			err,
		)
	}
	if len(config.Packages) == 0 {
		return installModel.CompiledPackageSet{}, fmt.Errorf(
			"%w: built-in catalog set %q has no packages",
			spec.ErrInvalid,
			config.SetName,
		)
	}

	canonicalProvider, err := providercanonical.New()
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	providers := make(
		[]provider.Provider,
		0,
		1+len(config.AdditionalProviders),
	)
	providers = append(providers, canonicalProvider)
	for index, provider := range config.AdditionalProviders {
		if provider == nil {
			return installModel.CompiledPackageSet{}, fmt.Errorf(
				"%w: built-in catalog provider %d is nil",
				spec.ErrInvalid,
				index,
			)
		}
		providers = append(providers, provider)
	}

	validation, err := validationFingerprint(providers)
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	declaration := documentTopology.BuiltinTopologyDeclaration()
	hydrationFingerprint, err := generatedHydrationFingerprint(
		config.SchemaVersion,
		config.SetName,
		declaration,
	)
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	output := installModel.CompiledPackageSet{
		Format: installModel.CompiledPackageSetFormat,
		Name:   config.SetName,
		Hydration: installModel.Hydration{
			InstallerName: config.InstallerName,
			RootID:        declaration.Root.ID,
			SourceID:      documentTopology.BuiltinPackageSourceID(),
			Fingerprint:   hydrationFingerprint,
		},
		Packages: make(
			[]installModel.CompiledPackage,
			0,
			len(config.Packages),
		),
	}

	inputs := append([]PackageInput(nil), config.Packages...)
	for index := range inputs {
		inputs[index].Files = normalizePackageFileLineEndings(
			inputs[index].Files,
		)
	}
	sort.Slice(inputs, func(left, right int) bool {
		leftDirectory, _ := inputs[left].Address.Directory()
		rightDirectory, _ := inputs[right].Address.Directory()
		return leftDirectory < rightDirectory
	})

	store, err := local.Open(ctx, local.Config{
		BaseDirectory: filepath.Join(
			temporaryDirectory,
			"artifact-store",
		),
		Providers:        providers,
		ProtectedRootIDs: documentTopology.ProtectedRootIDs(),
	})
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}
	defer store.Close()

	ctx = installFlow.WithPrivilege(ctx)
	if _, err := store.Topology.EnsureProtectedTopology(
		ctx,
		declaration,
	); err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	sourceID := documentTopology.BuiltinPackageSourceID()
	for _, input := range inputs {
		rootExpectation, err := input.rootExpectation()
		if err != nil {
			return installModel.CompiledPackageSet{}, err
		}

		locator, err := input.Address.FileLocator(input.DocumentFile)
		if err != nil {
			return installModel.CompiledPackageSet{}, err
		}

		if _, err := store.ManagedPackages.Publish(
			ctx,
			artifactModel.PublishArtifactRequest{
				RootID: declaration.Root.ID,
				Binding: artifactModel.SourceBinding{
					SourceID: sourceID,
					Locator:  locator,
				},
				ExpectedKind:        rootExpectation.Kind,
				ExpectedLogicalName: rootExpectation.LogicalName,
				ExpectedDefinition:  rootExpectation.DefinitionDigest,
				Package: sourceModel.ManagedPackagePublication{
					Address: input.Address,
					Files:   input.Files,
				},
				AllowProtected: true,
			},
		); err != nil {
			return installModel.CompiledPackageSet{}, err
		}
	}

	entries, err := store.Catalog.ListBySource(
		ctx,
		declaration.Root.ID,
		sourceID,
		catalogModel.ListOptions{},
	)
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	for _, input := range inputs {
		value, err := compilePackage(
			ctx,
			store,
			input,
			entries,
			validation,
		)
		if err != nil {
			return installModel.CompiledPackageSet{}, err
		}
		output.Packages = append(output.Packages, value)
	}

	sort.Slice(output.Packages, func(left, right int) bool {
		leftDirectory, _ := output.Packages[left].Address.Directory()
		rightDirectory, _ := output.Packages[right].Address.Directory()
		return leftDirectory < rightDirectory
	})

	return output, nil
}

// normalizePackageFileLineEndings makes generated package bytes independent
// of CRLF checkout conversion before publication and content hashing.
func normalizePackageFileLineEndings(
	files []sourceModel.ManagedPackageFile,
) []sourceModel.ManagedPackageFile {
	normalized := append([]sourceModel.ManagedPackageFile(nil), files...)
	for index := range normalized {
		content := normalized[index].Content
		if !bytes.Contains(content, []byte("\r\n")) {
			continue
		}
		normalized[index].Content = bytes.ReplaceAll(
			content,
			[]byte("\r\n"),
			[]byte("\n"),
		)
	}
	return normalized
}

func (p PackageInput) rootExpectation() (Expectation, error) {
	for _, value := range p.Expectations {
		if value.Locator == p.DocumentFile &&
			value.Subresource == "" {
			return value, nil
		}
	}

	return Expectation{}, fmt.Errorf(
		"%w: package %q has no root Artifact expectation",
		spec.ErrInvalid,
		p.EmbeddedRoot,
	)
}

func compilePackage(
	ctx context.Context,
	store *local.Store,
	input PackageInput,
	entries []catalogModel.Entry,
	validation cryptoutil.Digest,
) (installModel.CompiledPackage, error) {
	scope, err := input.Address.Directory()
	if err != nil {
		return installModel.CompiledPackage{}, err
	}

	output := installModel.CompiledPackage{
		EmbeddedRoot: input.EmbeddedRoot,
		Address:      input.Address,
		Files:        make([]installModel.CompiledFile, 0, len(input.Files)),
		Documents:    make([]installModel.CompiledDocument, 0),
	}

	fileDigests := make(map[spec.Locator]cryptoutil.Digest)
	for _, file := range input.Files {
		digest := cryptoutil.DigestBytes(file.Content)
		fileDigests[file.Locator] = digest
		output.Files = append(output.Files, installModel.CompiledFile{
			Locator: file.Locator,
			Size:    int64(len(file.Content)),
			Digest:  digest,
			Content: append([]byte(nil), file.Content...),
		})
	}

	type origin struct {
		locator     spec.Locator
		subresource spec.SubresourceLocator
		kind        artifactModel.ArtifactKind
	}

	expected := make(map[origin]Expectation, len(input.Expectations))
	for _, value := range input.Expectations {
		expected[origin{
			locator:     value.Locator,
			subresource: value.Subresource,
			kind:        value.Kind,
		}] = value
	}

	documents := make(map[spec.Locator]*installModel.CompiledDocument)
	for _, entry := range entries {
		relative, found := cutPackageRelativeLocator(
			scope,
			entry.Binding.Locator,
		)
		if !found {
			continue
		}

		key := origin{
			locator:     relative,
			subresource: entry.Binding.SubresourceLocator,
			kind:        entry.Kind,
		}
		wanted, found := expected[key]
		if !found {
			return installModel.CompiledPackage{}, fmt.Errorf(
				"%w: package %q emitted undeclared Artifact %q",
				spec.ErrInvalid,
				scope,
				entry.Ref().ArtifactID,
			)
		}

		if entry.State != artifactModel.StateAvailable ||
			entry.Definition == nil ||
			entry.LogicalName != wanted.LogicalName ||
			entry.LogicalVersion != wanted.LogicalVersion ||
			entry.Definition.Digest != wanted.DefinitionDigest {
			return installModel.CompiledPackage{}, fmt.Errorf(
				"%w: admitted Artifact differs from package expectation",
				spec.ErrDigestMismatch,
			)
		}

		record, err := store.Artifacts.Get(ctx, entry.Ref())
		if err != nil {
			return installModel.CompiledPackage{}, err
		}
		if record.SourceContentDigest == nil ||
			*record.SourceContentDigest != fileDigests[relative] {
			return installModel.CompiledPackage{}, fmt.Errorf(
				"%w: source digest differs from embedded package file",
				spec.ErrDigestMismatch,
			)
		}

		definitionValue, err := store.Artifacts.GetDefinition(
			ctx,
			entry.Ref(),
		)
		if err != nil {
			return installModel.CompiledPackage{}, err
		}

		document := documents[relative]
		if document == nil {
			document = &installModel.CompiledDocument{
				Locator: relative,
				Digest:  fileDigests[relative],
			}
			documents[relative] = document
		}

		document.Artifacts = append(
			document.Artifacts,
			installModel.CompiledArtifact{
				Subresource: entry.Binding.SubresourceLocator,
				Definition:  definitionValue,
				Diagnostics: record.Diagnostics,
			},
		)
		delete(expected, key)
	}

	if len(expected) != 0 {
		return installModel.CompiledPackage{}, fmt.Errorf(
			"%w: package %q has unfulfilled Artifact expectations",
			spec.ErrReferenceUnresolved,
			scope,
		)
	}

	for _, document := range documents {
		sort.Slice(document.Artifacts, func(left, right int) bool {
			if document.Artifacts[left].Subresource !=
				document.Artifacts[right].Subresource {
				return document.Artifacts[left].Subresource <
					document.Artifacts[right].Subresource
			}
			return document.Artifacts[left].Definition.Kind <
				document.Artifacts[right].Definition.Kind
		})
		output.Documents = append(output.Documents, document.Clone())
	}

	sort.Slice(output.Files, func(left, right int) bool {
		return output.Files[left].Locator < output.Files[right].Locator
	})
	sort.Slice(output.Documents, func(left, right int) bool {
		return output.Documents[left].Locator <
			output.Documents[right].Locator
	})

	output.Fingerprint, err = cryptoutil.CanonicalDigest(struct {
		Validation cryptoutil.Digest            `json:"validation"`
		Package    installModel.CompiledPackage `json:"package"`
	}{
		Validation: validation,
		Package:    output,
	})
	return output, err
}

func cutPackageRelativeLocator(
	scope spec.Locator,
	locator spec.Locator,
) (spec.Locator, bool) {
	prefix := string(scope) + "/"
	raw, found := strings.CutPrefix(string(locator), prefix)
	if !found || raw == "" {
		return "", false
	}
	return spec.Locator(raw), true
}

func generatedHydrationFingerprint(
	schemaVersion string,
	setName string,
	declaration installModel.Declaration,
) (cryptoutil.Digest, error) {
	// Schema/decoder changes belong to package admission fingerprints.
	// They must not destroy the shared Root and its local Artifact state.
	return cryptoutil.CanonicalDigest(struct {
		SchemaVersion string                   `json:"schemaVersion"`
		SetName       string                   `json:"setName"`
		Topology      installModel.Declaration `json:"topology"`
	}{
		SchemaVersion: schemaVersion,
		SetName:       setName,
		Topology:      declaration,
	})
}

func validationFingerprint(
	providers []provider.Provider,
) (cryptoutil.Digest, error) {
	type schemaValue struct {
		Identity string            `json:"identity"`
		Digest   cryptoutil.Digest `json:"digest"`
	}
	type decoderValue struct {
		ID       spec.DecoderID `json:"id"`
		Revision string         `json:"revision"`
	}

	schemas := make([]schemaValue, 0)
	decoders := make([]decoderValue, 0)

	for _, provider := range providers {
		descriptor := provider.Descriptor()

		for _, codec := range descriptor.Schemas {
			key := codec.Key()
			schemas = append(schemas, schemaValue{
				Identity: string(key.Entity) + "/" +
					string(key.Kind) + "/" +
					string(key.SchemaID) + "/" +
					key.SchemaVersion,
				Digest: cryptoutil.DigestBytes(codec.JSONSchema()),
			})
		}

		for _, decoder := range descriptor.Decoders {
			decoders = append(decoders, decoderValue{
				ID:       decoder.ID(),
				Revision: decoder.Revision(),
			})
		}
	}

	sort.Slice(schemas, func(left, right int) bool {
		return schemas[left].Identity < schemas[right].Identity
	})
	sort.Slice(decoders, func(left, right int) bool {
		return decoders[left].ID < decoders[right].ID
	})

	return cryptoutil.CanonicalDigest(struct {
		Schemas  []schemaValue  `json:"schemas"`
		Decoders []decoderValue `json:"decoders"`
	}{
		Schemas:  schemas,
		Decoders: decoders,
	})
}
