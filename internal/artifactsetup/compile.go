package artifactsetup

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/compose/local"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/registration"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

// CompileConfig selects one application-owned built-in package set. All
// package files must already contain family-normalized final bytes.
type CompileConfig struct {
	SetName       string
	SchemaVersion string
	InstallerName string

	Interpretations *coreinterpretation.Registry

	AdditionalSchemaCodecs []schema.Codec
	AdditionalDecoders     []ingest.Decoder
	Packages               []installFlow.PackageInput
}

// schemaKeysForCompiledPackages derives the declaration admission keys from
// the Artifacts the package compiler has already declared it will emit.
//
// This deliberately prevents a schema or canonical-decoder change for an
// unrelated declaration family from changing another built-in package set's
// hydration identity.
func schemaKeysForCompiledPackages(
	interpretations *coreinterpretation.Registry,
	packages []installFlow.PackageInput,
) ([]schemaModel.Key, error) {
	if interpretations == nil {
		return nil, fmt.Errorf(
			"%w: built-in compilation interpretation registry is nil",
			spec.ErrInvalid,
		)
	}

	kinds := make([]artifactModel.ArtifactKind, 0)
	seen := make(map[artifactModel.ArtifactKind]struct{})
	for packageIndex, packageValue := range packages {
		for expectationIndex, expectation := range packageValue.Expectations {
			if err := expectation.Kind.Validate(); err != nil {
				return nil, fmt.Errorf(
					"built-in package %d expectation %d kind: %w",
					packageIndex,
					expectationIndex,
					err,
				)
			}
			if _, found := seen[expectation.Kind]; found {
				continue
			}
			seen[expectation.Kind] = struct{}{}
			kinds = append(kinds, expectation.Kind)
		}
	}
	if len(kinds) == 0 {
		return nil, fmt.Errorf(
			"%w: built-in package set has no expected Artifact kinds",
			spec.ErrInvalid,
		)
	}
	return interpretations.SchemaKeysForArtifactKinds(kinds)
}

// CompileBuiltInPackageSet compiles application-owned embedded content through
// the ordinary generic Store admission and publication path.
func CompileBuiltInPackageSet(
	ctx context.Context,
	temporaryDirectory string,
	config CompileConfig,
) (_ installModel.CompiledPackageSet, returnErr error) {
	if ctx == nil {
		return installModel.CompiledPackageSet{}, fmt.Errorf(
			"%w: built-in compilation context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return installModel.CompiledPackageSet{}, err
	}
	if temporaryDirectory == "" {
		return installModel.CompiledPackageSet{}, fmt.Errorf(
			"%w: built-in compilation directory is empty",
			spec.ErrInvalid,
		)
	}
	if config.Interpretations == nil {
		return installModel.CompiledPackageSet{}, fmt.Errorf(
			"%w: built-in compilation interpretation registry is nil",
			spec.ErrInvalid,
		)
	}

	schemaKeys, err := schemaKeysForCompiledPackages(
		config.Interpretations,
		config.Packages,
	)
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	codecs, err := registration.LLMDeclarationSchemaCodecsForSchemaKeys(
		schemaKeys,
	)
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}
	codecs = append(
		append([]schema.Codec(nil), codecs...),
		config.AdditionalSchemaCodecs...,
	)
	codecs, err = schema.NormalizeCodecs(codecs)
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	// Compilation intentionally receives only canonical declaration decoders
	// plus the explicit source-format adapters required by this package set.
	//
	// Runtime Store assembly uses the complete application source-format
	// selection. Reusing that runtime set here would make unrelated decoder
	// registrations alter compiled package fingerprints.
	canonicalDecoders, err := registration.LLMCanonicalDeclarationDecodersForSchemaKeys(
		config.Interpretations,
		schemaKeys,
	)
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	decoders := make([]ingest.Decoder, 0, len(canonicalDecoders)+len(config.AdditionalDecoders))
	decoders = append(
		decoders,
		canonicalDecoders...,
	)
	decoders = append(
		decoders,
		config.AdditionalDecoders...,
	)
	decoders, err = ingest.NormalizeDecoders(decoders)
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	store, err := local.Open(ctx, local.Config{
		BaseDirectory:    filepath.Join(temporaryDirectory, "artifact-store"),
		SchemaCodecs:     codecs,
		Decoders:         decoders,
		ProtectedRootIDs: topology.ProtectedRootIDs(),
	})
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}
	defer func() {
		returnErr = errors.Join(returnErr, store.Close())
	}()

	return installFlow.CompilePackageSet(
		root.WithInstallerPrivilege(ctx),
		store.Topology,
		store.ManagedPackages,
		store.Catalog,
		store.Artifacts,
		installFlow.CompileConfig{
			SetName:       config.SetName,
			SchemaVersion: config.SchemaVersion,
			InstallerName: config.InstallerName,
			Declaration:   topology.BuiltinTopologyDeclaration(),
			SourceID:      topology.BuiltinPackageSourceID(),
			SchemaCodecs:  codecs,
			Decoders:      decoders,
			Packages:      config.Packages,
		},
	)
}
