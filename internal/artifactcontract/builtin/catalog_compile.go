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
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/install"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/install/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// Expectation is the declaration admission result expected from one source
// document or subresource.
type Expectation struct {
	Locator          model.Locator
	Subresource      model.SubresourceLocator
	Kind             artifact.ArtifactKind
	LogicalName      model.LogicalName
	LogicalVersion   model.LogicalVersion
	DefinitionDigest cryptoutil.Digest
}

// PackageInput is one prepared managed package. Package preparation remains
// domain-owned. This package only performs generic Artifact Store admission.
type PackageInput struct {
	EmbeddedRoot model.Locator
	Address      source.ManagedPackageAddress
	DocumentFile model.Locator
	Files        []source.ManagedPackageFile
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
) (topology.CompiledPackageSet, error) {
	if ctx == nil {
		return topology.CompiledPackageSet{}, fmt.Errorf(
			"%w: built-in catalog compilation context is nil",
			model.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return topology.CompiledPackageSet{}, err
	}
	if temporaryDirectory == "" {
		return topology.CompiledPackageSet{}, fmt.Errorf(
			"%w: built-in catalog compilation directory is empty",
			model.ErrInvalid,
		)
	}
	if config.SetName == "" {
		return topology.CompiledPackageSet{}, fmt.Errorf(
			"%w: built-in catalog set name is empty",
			model.ErrInvalid,
		)
	}
	if config.SchemaVersion == "" {
		return topology.CompiledPackageSet{}, fmt.Errorf(
			"%w: built-in catalog schema version is empty",
			model.ErrInvalid,
		)
	}
	if err := topology.ValidateHydrationInstallerName(
		config.InstallerName,
	); err != nil {
		return topology.CompiledPackageSet{}, fmt.Errorf(
			"built-in catalog installer name: %w",
			err,
		)
	}
	if len(config.Packages) == 0 {
		return topology.CompiledPackageSet{}, fmt.Errorf(
			"%w: built-in catalog set %q has no packages",
			model.ErrInvalid,
			config.SetName,
		)
	}

	canonicalProvider, err := providercanonical.New()
	if err != nil {
		return topology.CompiledPackageSet{}, err
	}

	providers := make(
		[]provider.Provider,
		0,
		1+len(config.AdditionalProviders),
	)
	providers = append(providers, canonicalProvider)
	for index, provider := range config.AdditionalProviders {
		if provider == nil {
			return topology.CompiledPackageSet{}, fmt.Errorf(
				"%w: built-in catalog provider %d is nil",
				model.ErrInvalid,
				index,
			)
		}
		providers = append(providers, provider)
	}

	validation, err := validationFingerprint(providers)
	if err != nil {
		return topology.CompiledPackageSet{}, err
	}

	declaration := documentTopology.BuiltinTopologyDeclaration()
	hydrationFingerprint, err := generatedHydrationFingerprint(
		config.SchemaVersion,
		config.SetName,
		declaration,
	)
	if err != nil {
		return topology.CompiledPackageSet{}, err
	}

	output := topology.CompiledPackageSet{
		Format: topology.CompiledPackageSetFormat,
		Name:   config.SetName,
		Hydration: topology.Hydration{
			InstallerName: config.InstallerName,
			RootID:        declaration.Root.ID,
			SourceID:      documentTopology.BuiltinPackageSourceID(),
			Fingerprint:   hydrationFingerprint,
		},
		Packages: make(
			[]topology.CompiledPackage,
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
		return topology.CompiledPackageSet{}, err
	}
	defer store.Close()

	ctx = install.WithPrivilege(ctx)
	if _, err := store.Topology.EnsureProtectedTopology(
		ctx,
		declaration,
	); err != nil {
		return topology.CompiledPackageSet{}, err
	}

	sourceID := documentTopology.BuiltinPackageSourceID()
	for _, input := range inputs {
		rootExpectation, err := input.rootExpectation()
		if err != nil {
			return topology.CompiledPackageSet{}, err
		}

		locator, err := input.Address.FileLocator(input.DocumentFile)
		if err != nil {
			return topology.CompiledPackageSet{}, err
		}

		if _, err := store.ManagedArtifacts.Publish(
			ctx,
			artifact.PublishArtifactRequest{
				RootID: declaration.Root.ID,
				Binding: artifact.SourceBinding{
					SourceID: sourceID,
					Locator:  locator,
				},
				ExpectedKind:        rootExpectation.Kind,
				ExpectedLogicalName: rootExpectation.LogicalName,
				ExpectedDefinition:  rootExpectation.DefinitionDigest,
				Package: source.ManagedPackagePublication{
					Address: input.Address,
					Files:   input.Files,
				},
				AllowProtected: true,
			},
		); err != nil {
			return topology.CompiledPackageSet{}, err
		}
	}

	entries, err := store.Artifacts.ListBySource(
		ctx,
		declaration.Root.ID,
		sourceID,
		catalog.ListOptions{},
	)
	if err != nil {
		return topology.CompiledPackageSet{}, err
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
			return topology.CompiledPackageSet{}, err
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
	files []source.ManagedPackageFile,
) []source.ManagedPackageFile {
	normalized := append([]source.ManagedPackageFile(nil), files...)
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
		model.ErrInvalid,
		p.EmbeddedRoot,
	)
}

func compilePackage(
	ctx context.Context,
	store *local.Store,
	input PackageInput,
	entries []catalog.Entry,
	validation cryptoutil.Digest,
) (topology.CompiledPackage, error) {
	scope, err := input.Address.Directory()
	if err != nil {
		return topology.CompiledPackage{}, err
	}

	output := topology.CompiledPackage{
		EmbeddedRoot: input.EmbeddedRoot,
		Address:      input.Address,
		Files:        make([]topology.CompiledFile, 0, len(input.Files)),
		Documents:    make([]topology.CompiledDocument, 0),
	}

	fileDigests := make(map[model.Locator]cryptoutil.Digest)
	for _, file := range input.Files {
		digest := cryptoutil.DigestBytes(file.Content)
		fileDigests[file.Locator] = digest
		output.Files = append(output.Files, topology.CompiledFile{
			Locator: file.Locator,
			Size:    int64(len(file.Content)),
			Digest:  digest,
			Content: append([]byte(nil), file.Content...),
		})
	}

	type origin struct {
		locator     model.Locator
		subresource model.SubresourceLocator
		kind        artifact.ArtifactKind
	}

	expected := make(map[origin]Expectation, len(input.Expectations))
	for _, value := range input.Expectations {
		expected[origin{
			locator:     value.Locator,
			subresource: value.Subresource,
			kind:        value.Kind,
		}] = value
	}

	documents := make(map[model.Locator]*topology.CompiledDocument)
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
			return topology.CompiledPackage{}, fmt.Errorf(
				"%w: package %q emitted undeclared Artifact %q",
				model.ErrInvalid,
				scope,
				entry.Ref().ArtifactID,
			)
		}

		if entry.State != artifact.StateAvailable ||
			entry.Definition == nil ||
			entry.LogicalName != wanted.LogicalName ||
			entry.LogicalVersion != wanted.LogicalVersion ||
			entry.Definition.Digest != wanted.DefinitionDigest {
			return topology.CompiledPackage{}, fmt.Errorf(
				"%w: admitted Artifact differs from package expectation",
				model.ErrDigestMismatch,
			)
		}

		record, err := store.Artifacts.Get(ctx, entry.Ref())
		if err != nil {
			return topology.CompiledPackage{}, err
		}
		if record.SourceContentDigest == nil ||
			*record.SourceContentDigest != fileDigests[relative] {
			return topology.CompiledPackage{}, fmt.Errorf(
				"%w: source digest differs from embedded package file",
				model.ErrDigestMismatch,
			)
		}

		definitionValue, err := store.Artifacts.GetDefinition(
			ctx,
			entry.Ref(),
		)
		if err != nil {
			return topology.CompiledPackage{}, err
		}

		document := documents[relative]
		if document == nil {
			document = &topology.CompiledDocument{
				Locator: relative,
				Digest:  fileDigests[relative],
			}
			documents[relative] = document
		}

		document.Artifacts = append(
			document.Artifacts,
			topology.CompiledArtifact{
				Subresource: entry.Binding.SubresourceLocator,
				Definition:  definitionValue,
				Diagnostics: record.Diagnostics,
			},
		)
		delete(expected, key)
	}

	if len(expected) != 0 {
		return topology.CompiledPackage{}, fmt.Errorf(
			"%w: package %q has unfulfilled Artifact expectations",
			model.ErrReferenceUnresolved,
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
		Validation cryptoutil.Digest        `json:"validation"`
		Package    topology.CompiledPackage `json:"package"`
	}{
		Validation: validation,
		Package:    output,
	})
	return output, err
}

func cutPackageRelativeLocator(
	scope model.Locator,
	locator model.Locator,
) (model.Locator, bool) {
	prefix := string(scope) + "/"
	raw, found := strings.CutPrefix(string(locator), prefix)
	if !found || raw == "" {
		return "", false
	}
	return model.Locator(raw), true
}

func generatedHydrationFingerprint(
	schemaVersion string,
	setName string,
	declaration topology.Declaration,
) (cryptoutil.Digest, error) {
	// Schema/decoder changes belong to package admission fingerprints.
	// They must not destroy the shared Root and its local Artifact state.
	return cryptoutil.CanonicalDigest(struct {
		SchemaVersion string               `json:"schemaVersion"`
		SetName       string               `json:"setName"`
		Topology      topology.Declaration `json:"topology"`
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
		ID       model.DecoderID `json:"id"`
		Revision string          `json:"revision"`
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
