package install

import (
	"context"
	"fmt"
	"sort"
	"strings"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	managepackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type Expectation struct {
	Locator          spec.Locator
	Subresource      spec.SubresourceLocator
	Kind             artifactModel.ArtifactKind
	LogicalName      spec.LogicalName
	LogicalVersion   spec.LogicalVersion
	DefinitionDigest cryptoutil.Digest
}

// PackageInput contains final family-prepared bytes and expectations.
//
// All text normalization and declaration adaptation must already be complete.
// Compilation never rewrites package content after expectations are fixed.
type PackageInput struct {
	EmbeddedRoot spec.Locator
	Address      managedpackageModel.ManagedPackageAddress
	DocumentFile spec.Locator
	Files        []managedpackageModel.ManagedPackageFile
	Expectations []Expectation
}

type CompileConfig struct {
	SetName       string
	SchemaVersion string
	InstallerName string

	Declaration installModel.Declaration
	SourceID    sourceModel.SourceID

	// These must be the explicit registrations used to assemble the supplied
	// compilation Store. Their historical raw-schema fingerprint is retained.
	SchemaCodecs []schema.Codec
	Decoders     []ingest.Decoder
	Packages     []PackageInput
}

type CompilationPublisher interface {
	Publish(
		ctx context.Context,
		request managepackageModel.PublishRequest,
	) (managepackageModel.PublishResult, error)
}

type CompilationCatalog interface {
	ListBySource(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
		options catalogModel.ListOptions,
	) ([]catalogModel.Entry, error)
}

type CompilationArtifacts interface {
	Get(ctx context.Context, ref artifactModel.ArtifactRef) (artifactModel.Artifact, error)
	GetDefinition(ctx context.Context, ref artifactModel.ArtifactRef) (definitionModel.Definition, error)
}

// CompilePackageSet uses ordinary Store publication and admission.
//
// It borrows all capabilities. Deployment opens and closes the compilation
// Store; this function never selects SQLite, drivers, schema providers,
// application Root IDs, or declaration registrations.
func CompilePackageSet(
	ctx context.Context,
	topology installModel.Ensurer,
	publisher CompilationPublisher,
	catalog CompilationCatalog,
	artifacts CompilationArtifacts,
	config CompileConfig,
) (installModel.CompiledPackageSet, error) {
	if topology == nil || publisher == nil || catalog == nil || artifacts == nil {
		return installModel.CompiledPackageSet{}, fmt.Errorf(
			"%w: package compilation dependencies are incomplete",
			spec.ErrInvalid,
		)
	}
	if ctx == nil {
		return installModel.CompiledPackageSet{}, fmt.Errorf("%w: compilation context is nil", spec.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	inputs, validation, hydration, err := prepareCompilation(config)
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}
	output := installModel.CompiledPackageSet{
		Format:    installModel.CompiledPackageSetFormat,
		Name:      config.SetName,
		Hydration: hydration,
		Packages:  make([]installModel.CompiledPackage, 0, len(inputs)),
	}

	if _, err := topology.EnsureProtectedTopology(ctx, config.Declaration); err != nil {
		return installModel.CompiledPackageSet{}, err
	}
	for _, input := range inputs {
		expected, err := input.rootExpectation()
		if err != nil {
			return installModel.CompiledPackageSet{}, err
		}
		locator, err := input.Address.FileLocator(input.DocumentFile)
		if err != nil {
			return installModel.CompiledPackageSet{}, err
		}
		if _, err := publisher.Publish(ctx, managepackageModel.PublishRequest{
			RootID: hydration.RootID,
			Binding: artifactModel.SourceBinding{
				SourceID: hydration.SourceID,
				Locator:  locator,
			},
			ExpectedKind:        expected.Kind,
			ExpectedLogicalName: expected.LogicalName,
			ExpectedDefinition:  expected.DefinitionDigest,
			Package: managedpackageModel.ManagedPackagePublication{
				Address: input.Address,
				Files:   input.Files,
			},
		}); err != nil {
			return installModel.CompiledPackageSet{}, err
		}
	}

	entries, err := catalog.ListBySource(
		ctx,
		hydration.RootID,
		hydration.SourceID,
		catalogModel.ListOptions{},
	)
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}
	for _, input := range inputs {
		value, err := compilePackage(ctx, artifacts, input, entries, validation)
		if err != nil {
			return installModel.CompiledPackageSet{}, err
		}
		output.Packages = append(output.Packages, value)
	}
	return output, nil
}

func prepareCompilation(
	config CompileConfig,
) ([]PackageInput, cryptoutil.Digest, installModel.Hydration, error) {
	fail := func(err error) ([]PackageInput, cryptoutil.Digest, installModel.Hydration, error) {
		return nil, "", installModel.Hydration{}, err
	}
	if err := spec.ValidateRequiredText("compiled set name", config.SetName, spec.MaxKindBytes); err != nil {
		return fail(err)
	}
	if err := spec.ValidateRequiredText(
		"compiled set schema version",
		config.SchemaVersion,
		spec.MaxVersionBytes,
	); err != nil {
		return fail(err)
	}
	if err := installModel.ValidateHydrationInstallerName(config.InstallerName); err != nil {
		return fail(err)
	}
	if err := config.Declaration.Validate(); err != nil {
		return fail(err)
	}
	if err := config.SourceID.Validate(); err != nil {
		return fail(err)
	}
	sourceFound := false
	for _, draft := range config.Declaration.Sources {
		if draft.ID == config.SourceID {
			sourceFound = draft.Enabled && !draft.Discovery.Empty()
			break
		}
	}
	if !sourceFound {
		return fail(fmt.Errorf("%w: compilation requires a declared enabled discovery Source", spec.ErrInvalid))
	}
	if len(config.Packages) == 0 || len(config.Packages) > spec.MaxDiscoveryCandidates {
		return fail(fmt.Errorf("%w: compiled set package count is invalid", spec.ErrInvalid))
	}

	codecs, err := schema.NormalizeCodecs(config.SchemaCodecs)
	if err != nil {
		return fail(err)
	}
	decoders, err := ingest.NormalizeDecoders(config.Decoders)
	if err != nil {
		return fail(err)
	}
	validation, err := compilationValidationFingerprint(codecs, decoders)
	if err != nil {
		return fail(err)
	}

	inputs := make([]PackageInput, len(config.Packages))
	scopes := make(map[spec.Locator]struct{}, len(inputs))
	for index, input := range config.Packages {
		if err := input.EmbeddedRoot.ValidatePortable(false); err != nil {
			return fail(err)
		}
		scope, err := input.Address.Directory()
		if err != nil {
			return fail(err)
		}
		if _, duplicate := scopes[scope]; duplicate {
			return fail(fmt.Errorf("%w: duplicate compilation package %q", spec.ErrIdentityConflict, scope))
		}
		scopes[scope] = struct{}{}

		input.Files, err = managedpackageModel.NormalizeManagedPackageFiles(input.Files)
		if err != nil {
			return fail(err)
		}
		input.Expectations = append([]Expectation(nil), input.Expectations...)
		files := make(map[spec.Locator]struct{}, len(input.Files))
		for _, file := range input.Files {
			files[file.Locator] = struct{}{}
		}
		if _, found := files[input.DocumentFile]; !found {
			return fail(fmt.Errorf("%w: package primary document is absent", spec.ErrInvalid))
		}
		expected := make(map[compilationOrigin]struct{}, len(input.Expectations))
		for _, value := range input.Expectations {
			if err := value.Locator.ValidatePortable(false); err != nil {
				return fail(err)
			}
			if err := value.Subresource.Validate(); err != nil {
				return fail(err)
			}
			if err := value.Kind.Validate(); err != nil {
				return fail(err)
			}
			if err := value.LogicalName.Validate(); err != nil {
				return fail(err)
			}
			if err := value.LogicalVersion.Validate(true); err != nil {
				return fail(err)
			}
			if err := cryptoutil.ValidateDigest(value.DefinitionDigest); err != nil {
				return fail(err)
			}
			if _, found := files[value.Locator]; !found {
				return fail(fmt.Errorf("%w: expected declaration file is absent", spec.ErrInvalid))
			}
			key := compilationOrigin{value.Locator, value.Subresource, value.Kind}
			if _, duplicate := expected[key]; duplicate {
				return fail(fmt.Errorf("%w: duplicate compilation expectation", spec.ErrIdentityConflict))
			}
			expected[key] = struct{}{}
		}
		if _, err := input.rootExpectation(); err != nil {
			return fail(err)
		}
		inputs[index] = input
	}
	sort.Slice(inputs, func(left, right int) bool {
		a, _ := inputs[left].Address.Directory()
		b, _ := inputs[right].Address.Directory()
		return a < b
	})

	fingerprint, err := cryptoutil.CanonicalDigest(struct {
		SchemaVersion string                   `json:"schemaVersion"`
		SetName       string                   `json:"setName"`
		Topology      installModel.Declaration `json:"topology"`
	}{
		SchemaVersion: config.SchemaVersion,
		SetName:       config.SetName,
		Topology:      config.Declaration,
	})
	if err != nil {
		return fail(err)
	}
	return inputs, validation, installModel.Hydration{
		InstallerName: config.InstallerName,
		RootID:        config.Declaration.Root.ID,
		SourceID:      config.SourceID,
		Fingerprint:   fingerprint,
	}, nil
}

func (p PackageInput) rootExpectation() (Expectation, error) {
	for _, value := range p.Expectations {
		if value.Locator == p.DocumentFile && value.Subresource == "" {
			return value, nil
		}
	}
	return Expectation{}, fmt.Errorf(
		"%w: package %q has no primary Artifact expectation",
		spec.ErrInvalid,
		p.EmbeddedRoot,
	)
}

type compilationOrigin struct {
	locator     spec.Locator
	subresource spec.SubresourceLocator
	kind        artifactModel.ArtifactKind
}

func compilePackage(
	ctx context.Context,
	artifacts CompilationArtifacts,
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
	fileDigests := make(map[spec.Locator]cryptoutil.Digest, len(input.Files))
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

	expected := make(map[compilationOrigin]Expectation, len(input.Expectations))
	for _, value := range input.Expectations {
		expected[compilationOrigin{value.Locator, value.Subresource, value.Kind}] = value
	}

	documents := make(map[spec.Locator]*installModel.CompiledDocument)
	for _, entry := range entries {
		relative, inside := strings.CutPrefix(string(entry.Binding.Locator), string(scope)+"/")
		if !inside || relative == "" {
			continue
		}
		locator := spec.Locator(relative)
		key := compilationOrigin{locator, entry.Binding.SubresourceLocator, entry.Kind}
		wanted, found := expected[key]
		if !found {
			return installModel.CompiledPackage{}, fmt.Errorf(
				"%w: package %q emitted an undeclared typed Artifact origin",
				spec.ErrInvalid,
				scope,
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

		record, err := artifacts.Get(ctx, entry.Ref())
		if err != nil {
			return installModel.CompiledPackage{}, err
		}
		if record.SourceContentDigest == nil ||
			*record.SourceContentDigest != fileDigests[locator] {
			return installModel.CompiledPackage{}, fmt.Errorf(
				"%w: Artifact source digest differs from package bytes",
				spec.ErrDigestMismatch,
			)
		}
		definition, err := artifacts.GetDefinition(ctx, entry.Ref())
		if err != nil {
			return installModel.CompiledPackage{}, err
		}
		if definition.Digest != wanted.DefinitionDigest {
			return installModel.CompiledPackage{}, fmt.Errorf(
				"%w: Artifact Definition changed during compilation",
				spec.ErrDigestMismatch,
			)
		}

		document := documents[locator]
		if document == nil {
			document = &installModel.CompiledDocument{
				Locator: locator,
				Digest:  fileDigests[locator],
			}
			documents[locator] = document
		}
		document.Artifacts = append(document.Artifacts, installModel.CompiledArtifact{
			Subresource: entry.Binding.SubresourceLocator,
			Definition:  definition,
			Diagnostics: record.Diagnostics,
		})
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
			a, b := document.Artifacts[left], document.Artifacts[right]
			if a.Subresource != b.Subresource {
				return a.Subresource < b.Subresource
			}
			return a.Definition.Kind < b.Definition.Kind
		})
		output.Documents = append(output.Documents, document.Clone())
	}
	sort.Slice(output.Files, func(left, right int) bool {
		return output.Files[left].Locator < output.Files[right].Locator
	})
	sort.Slice(output.Documents, func(left, right int) bool {
		return output.Documents[left].Locator < output.Documents[right].Locator
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

// This deliberately preserves the compiled-package fingerprint domain.
// It is not the decoder registry's canonical-schema fingerprint.
func compilationValidationFingerprint(
	codecs []schema.Codec,
	decoders []ingest.Decoder,
) (cryptoutil.Digest, error) {
	type schemaValue struct {
		Identity string            `json:"identity"`
		Digest   cryptoutil.Digest `json:"digest"`
	}
	type decoderValue struct {
		ID       spec.DecoderID `json:"id"`
		Revision string         `json:"revision"`
	}
	schemas := make([]schemaValue, 0, len(codecs))
	for _, codec := range codecs {
		key := codec.Key()
		schemas = append(schemas, schemaValue{
			Identity: string(key.Entity) + "/" + string(key.Kind) + "/" +
				string(key.SchemaID) + "/" + key.SchemaVersion,
			Digest: cryptoutil.DigestBytes(codec.JSONSchema()),
		})
	}
	values := make([]decoderValue, 0, len(decoders))
	for _, decoder := range decoders {
		values = append(values, decoderValue{ID: decoder.ID(), Revision: decoder.Revision()})
	}
	sort.Slice(schemas, func(left, right int) bool {
		return schemas[left].Identity < schemas[right].Identity
	})
	sort.Slice(values, func(left, right int) bool {
		return values[left].ID < values[right].ID
	})
	return cryptoutil.CanonicalDigest(struct {
		Schemas  []schemaValue  `json:"schemas"`
		Decoders []decoderValue `json:"decoders"`
	}{
		Schemas:  schemas,
		Decoders: values,
	})
}
