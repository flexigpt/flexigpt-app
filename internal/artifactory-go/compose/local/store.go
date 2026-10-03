package local

import (
	"context"
	"fmt"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/compose/local/internal/assembly"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	artifactcleanupFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/artifactcleanup"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	managepackageFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type Store struct {
	Roots     root.API
	Sources   source.API
	Refresh   refreshFlow.API
	Artifacts artifact.API
	Catalog   catalog.API

	Definitions definition.API
	Schemas     schema.API
	Resources   resourceFlow.API

	ManagedPackages managepackageFlow.API

	ProtectedOverlays overlay.API
	StoreOverlays     overlay.StoreAPI

	SecretBindings  secret.API
	SecretRuntime   secret.RuntimeAPI
	SecretLifecycle secret.LifecycleAPI

	ArtifactCleanup artifactcleanupFlow.API
	Protection      root.ProtectionAPI
	Topology        installFlow.API

	components *assembly.Components
	closeOnce  sync.Once
	closeErr   error
}

type protectionAPI struct {
	policy rootModel.RootPolicy
}

func (p protectionAPI) IsProtectedRoot(
	rootID rootModel.RootID,
) bool {
	return p.policy != nil &&
		p.policy.IsProtectedRoot(rootID)
}

func (p protectionAPI) RequirePrivilegedInstaller(
	ctx context.Context,
) error {
	return installFlow.RequirePrivileged(ctx)
}

func Open(
	ctx context.Context,
	config Config,
) (*Store, error) {
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Artifact Store composition context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	retainedRootIDs := make(
		[]rootModel.RootID,
		0,
		len(config.RetainedRoots),
	)
	for index, draft := range config.RetainedRoots {
		if err := draft.ID.Validate(); err != nil {
			return nil, fmt.Errorf(
				"retained Root declaration %d: %w",
				index,
				err,
			)
		}
		retainedRootIDs = append(retainedRootIDs, draft.ID)
	}
	rootPolicy, err := root.NewSetRootPolicy(
		append(
			[]rootModel.RootID(nil),
			config.ProtectedRootIDs...,
		),
		retainedRootIDs,
	)
	if err != nil {
		return nil, err
	}

	components, err := assembly.Open(
		ctx,
		assembly.Config{
			BaseDirectory:     config.BaseDirectory,
			EmbeddedProviders: config.EmbeddedProviders,
			SchemaCodecs: append(
				[]schema.Codec(nil),
				config.SchemaCodecs...,
			),
			Decoders: append(
				[]ingest.Decoder(nil),
				config.Decoders...,
			),
			RootMutationPolicy: rootPolicy,
			ProtectedOverlayNamespaces: append(
				[]overlayModel.Namespace(nil),
				config.ProtectedOverlayNamespaces...,
			),
			StoreOverlayNamespaces: append(
				[]overlayModel.Namespace(nil),
				config.StoreOverlayNamespaces...,
			),
			SecretValues: config.SecretValues,
		},
	)
	if err != nil {
		return nil, err
	}

	output := &Store{
		Roots:           components.Roots,
		Sources:         components.Sources,
		Refresh:         components.Refresh,
		Artifacts:       components.Artifacts,
		Catalog:         components.Catalog,
		Definitions:     components.Definitions,
		Schemas:         components.Schemas,
		Resources:       components.Resources,
		ManagedPackages: components.ManagedArtifacts,

		ProtectedOverlays: components.ProtectedOverlays,
		StoreOverlays:     components.StoreOverlays,

		SecretBindings:  components.SecretBindings,
		SecretRuntime:   components.SecretRuntime,
		SecretLifecycle: components.SecretLifecycle,

		ArtifactCleanup: components.ArtifactCleanup,

		Protection: protectionAPI{
			policy: rootPolicy,
		},
		Topology: components.Install,

		components: components,
	}

	for _, draft := range config.RetainedRoots {
		if _, err := output.Roots.Create(ctx, draft); err != nil {
			_ = output.Close()
			return nil, fmt.Errorf(
				"ensure retained application Root %q: %w",
				draft.ID,
				err,
			)
		}
	}
	return output, nil
}

func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	s.closeOnce.Do(func() {
		if s.components != nil {
			s.closeErr = s.components.Close()
		}
		s.components = nil
	})
	return s.closeErr
}
