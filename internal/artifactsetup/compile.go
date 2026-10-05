package artifactsetup

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/compose/local"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/registration"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
)

// CompileConfig selects one application-owned built-in package set. All
// package files must already contain family-normalized final bytes.
type CompileConfig struct {
	SetName       string
	SchemaVersion string
	InstallerName string

	AdditionalSchemaCodecs []schema.Codec
	AdditionalDecoders     []ingest.Decoder
	Packages               []install.PackageInput
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

	interpretations, err := registration.NewLLMInterpretationRegistry()
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}
	codecs, err := registration.LLMDeclarationSchemaCodecs()
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
	canonicalDecoders, err := registration.LLMCanonicalDeclarationDecoders(
		interpretations,
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

	return install.CompilePackageSet(
		root.WithInstallerPrivilege(ctx),
		store.Topology,
		store.ManagedPackages,
		store.Catalog,
		store.Artifacts,
		install.CompileConfig{
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
