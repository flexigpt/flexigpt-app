package builtin

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/providercanonical"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/compose/local"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// Config selects one application built-in declaration package set.
// Packages must already contain final family-normalized bytes.
type Config struct {
	SetName       string
	SchemaVersion string
	InstallerName string

	AdditionalSchemaCodecs []schema.Codec
	AdditionalDecoders     []ingest.Decoder
	Packages               []install.PackageInput
}

// Compile supplies this application's declaration registrations and topology
// to Install's provider-independent ordinary-path compiler.
func Compile(
	ctx context.Context,
	temporaryDirectory string,
	config Config,
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

	registration, err := providercanonical.NewRegistration()
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}
	codecs := append(registration.SchemaCodecs(), config.AdditionalSchemaCodecs...)
	decoders := append(registration.Decoders(), config.AdditionalDecoders...)

	store, err := local.Open(ctx, local.Config{
		BaseDirectory:    filepath.Join(temporaryDirectory, "artifact-store"),
		SchemaCodecs:     codecs,
		Decoders:         decoders,
		ProtectedRootIDs: documentTopology.ProtectedRootIDs(),
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
			Declaration:   documentTopology.BuiltinTopologyDeclaration(),
			SourceID:      documentTopology.BuiltinPackageSourceID(),
			SchemaCodecs:  codecs,
			Decoders:      decoders,
			Packages:      config.Packages,
		},
	)
}
