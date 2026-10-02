package local

import (
	"context"
	"fmt"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/compose/local/internal/assembly"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/topology"
	overlay "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	rootimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/impl"
	root "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type Store struct {
	Roots             RootAPI
	Sources           SourceAPI
	Discovery         DiscoveryAPI
	Artifacts         ArtifactAPI
	Resources         ResourceAPI
	Schemas           SchemaAPI
	ManagedArtifacts  ManagedArtifactAPI
	ProtectedOverlays ProtectedOverlayAPI
	SecretBindings    SecretBindingAPI
	SecretRuntime     SecretRuntimeAPI
	LocalState        LocalStateMaintenanceAPI
	StoreOverlays     StoreOverlayAPI
	Protection        ProtectionAPI
	Topology          install.API
	LocatorResolvers  []provider.LocatorResolverFactory

	components *assembly.Components
	closeOnce  sync.Once
	closeErr   error
}

type protectionAPI struct {
	policy root.RootPolicy
}

func (p protectionAPI) IsProtectedRoot(
	rootID root.RootID,
) bool {
	return p.policy != nil &&
		p.policy.IsProtectedRoot(rootID)
}

func (p protectionAPI) RequirePrivilegedInstaller(
	ctx context.Context,
) error {
	return install.RequirePrivileged(ctx)
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
		[]root.RootID,
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
	rootPolicy, err := rootimpl.NewSetRootPolicy(
		append(
			[]root.RootID(nil),
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
			ArtifactProviders: append(
				[]provider.Provider(nil),
				config.Providers...,
			),
			RootMutationPolicy: rootPolicy,
			ProtectedOverlayNamespaces: append(
				[]overlay.Namespace(nil),
				config.ProtectedOverlayNamespaces...,
			),
			StoreOverlayNamespaces: append(
				[]overlay.Namespace(nil),
				config.StoreOverlayNamespaces...,
			),
			SecretValues: config.SecretValues,
		},
	)
	if err != nil {
		return nil, err
	}

	output := &Store{
		Roots:             components.Roots,
		Sources:           components.Sources,
		Discovery:         components.Refresh,
		Artifacts:         components.Artifacts,
		Resources:         components.Resources,
		Schemas:           components.ShareableSchemas,
		ManagedArtifacts:  components.ManagedArtifacts,
		ProtectedOverlays: components.LocalState,
		SecretBindings:    components.LocalState,
		SecretRuntime:     components.LocalState,
		LocalState:        components.LocalState,
		StoreOverlays:     components.LocalState,
		Protection: protectionAPI{
			policy: rootPolicy,
		},
		LocatorResolvers: append(
			[]provider.LocatorResolverFactory(nil),
			components.LocatorResolvers...,
		),
		components: components,
	}
	output.Topology = output

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

func (s *Store) EnsureProtectedTopology(
	ctx context.Context,
	declaration topology.Declaration,
) (topology.Installed, error) {
	if s == nil || s.components == nil {
		return topology.Installed{}, spec.ErrClosed
	}
	return s.components.EnsureProtectedTopology(ctx, declaration)
}

func (s *Store) PrepareTopologyHydrations(
	ctx context.Context,
	desired []topology.Hydration,
) (map[string]bool, error) {
	if s == nil || s.components == nil {
		return nil, spec.ErrClosed
	}
	return s.components.PrepareTopologyHydrations(ctx, desired)
}

func (s *Store) CommitTopologyHydration(
	ctx context.Context,
	desired topology.Hydration,
) error {
	if s == nil || s.components == nil {
		return spec.ErrClosed
	}
	return s.components.CommitTopologyHydration(ctx, desired)
}

func (s *Store) PrepareTopologyPackageHydrations(
	ctx context.Context,
	installerNames []string,
	desired []topology.PackageHydration,
) (topology.PackageHydrationPreparation, error) {
	if s == nil || s.components == nil {
		return topology.PackageHydrationPreparation{}, spec.ErrClosed
	}
	return s.components.PrepareTopologyPackageHydrations(
		ctx,
		installerNames,
		desired,
	)
}

func (s *Store) CommitTopologyPackageHydration(
	ctx context.Context,
	desired topology.PackageHydration,
) error {
	if s == nil || s.components == nil {
		return spec.ErrClosed
	}
	return s.components.CommitTopologyPackageHydration(ctx, desired)
}

func (s *Store) DeleteTopologyPackageHydration(
	ctx context.Context,
	value topology.PackageHydration,
) error {
	if s == nil || s.components == nil {
		return spec.ErrClosed
	}
	return s.components.DeleteTopologyPackageHydration(ctx, value)
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
