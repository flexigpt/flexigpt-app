// Package local opens Artifact Store for the local filesystem deployment.
//
// It owns local layout, SQLite, concrete source drivers, the JSON Schema
// provider, and successful-close ownership of locally assembled resources. It
// returns the single generic store/compose.Store handle rather than a local
// forwarding wrapper.
package local

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/jsonschema"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/sqlite"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/compose"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// Open opens the local Artifact Store deployment and returns the one generic
// assembled Store handle. On success, this handle owns closure of locally
// opened SQLite resources and the caller-supplied SecretValues backend. On
// failure, caller-supplied SecretValues remains caller-owned.
func Open(ctx context.Context, config Config) (_ *compose.Store, returnErr error) {
	if config.BaseDirectory == "" {
		return nil, fmt.Errorf("%w: Artifact Store local base directory is empty", spec.ErrInvalid)
	}

	policy, err := localRootPolicy(config)
	if err != nil {
		return nil, err
	}
	base, err := filepath.Abs(config.BaseDirectory)
	if err != nil {
		return nil, err
	}
	base = filepath.Clean(base)
	if err := ensureStoreLayout(base); err != nil {
		return nil, err
	}

	metadata, err := sqlite.Open(ctx, filepath.Join(base, storeMetadataFileName))
	if err != nil {
		return nil, err
	}
	resources := &deploymentResources{
		metadata:     metadata,
		secretValues: config.SecretValues,
	}
	defer func() {
		if returnErr != nil {
			returnErr = errors.Join(returnErr, resources.closeAfterFailure())
		}
	}()

	drivers, err := localSourceDrivers(ctx, config, base)
	if err != nil {
		return nil, err
	}

	rootRepository := metadata.Roots()
	sourceRepository := metadata.Sources()
	artifactRepository := metadata.Artifacts()
	secretBindingRepository := metadata.Secrets()
	secretRuntimeRepository := metadata.Secrets()
	secretLifecycleRepository := metadata.Secrets()

	assembled, err := compose.Open(ctx, compose.Config{
		RootRepository:             rootRepository,
		SourceRepository:           sourceRepository,
		ArtifactRepository:         artifactRepository,
		CatalogRepository:          metadata.Catalog(),
		DefinitionRepository:       metadata.Definitions(),
		RefreshStates:              metadata.RefreshStates(),
		RefreshPublisher:           metadata.Publisher(),
		OverlayRepository:          metadata.Overlays(),
		SecretBindingRepository:    secretBindingRepository,
		SecretRuntimeRepository:    secretRuntimeRepository,
		SecretLifecycleRepository:  secretLifecycleRepository,
		ArtifactCleanupRepository:  metadata.ArtifactCleanup(),
		InstallationRepository:     metadata,
		SourceDrivers:              drivers,
		SchemaFactory:              jsonschema.Factory{},
		SchemaCodecs:               append([]schema.Codec(nil), config.SchemaCodecs...),
		Decoders:                   append([]ingest.Decoder(nil), config.Decoders...),
		Clock:                      config.Clock,
		ArtifactIDProvider:         config.ArtifactIDProvider,
		RootPolicy:                 policy,
		ProtectedOverlayNamespaces: append([]overlayModel.Namespace(nil), config.ProtectedOverlayNamespaces...),
		StoreOverlayNamespaces:     append([]overlayModel.Namespace(nil), config.StoreOverlayNamespaces...),
		SecretValues:               config.SecretValues,
		Shutdown:                   resources.Close,
	})
	if err != nil {
		return nil, err
	}

	for _, draft := range config.RetainedRoots {
		if _, err := assembled.Roots.Create(ctx, draft); err != nil {
			return nil, errors.Join(
				fmt.Errorf("ensure retained application Root %q: %w", draft.ID, err),
				assembled.Close(),
			)
		}
	}
	// Retained Root initialization is part of local opening. SecretValues
	// remains caller-owned until the fully assembled deployment succeeds.
	resources.transferSecretOwnership()
	return assembled, nil
}

func localRootPolicy(config Config) (root.Policy, error) {
	retained := make([]rootModel.RootID, len(config.RetainedRoots))
	for index, draft := range config.RetainedRoots {
		retained[index] = draft.ID
	}
	return root.NewSetRootPolicy(
		append([]rootModel.RootID(nil), config.ProtectedRootIDs...),
		retained,
	)
}
