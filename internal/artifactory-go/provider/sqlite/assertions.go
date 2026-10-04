package sqlite

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	artifactcleanupFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/artifactcleanup"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
)

var (
	_ artifact.Repository            = (*ArtifactRepository)(nil)
	_ catalog.Repository             = (*CatalogRepository)(nil)
	_ definition.Repository          = (*DefinitionRepository)(nil)
	_ overlay.Repository             = (*OverlayRepository)(nil)
	_ secret.BindingRepository       = (*SecretRepository)(nil)
	_ secret.RuntimeRepository       = (*SecretRepository)(nil)
	_ secret.LifecycleRepository     = (*SecretRepository)(nil)
	_ artifactcleanupFlow.Repository = (*ArtifactCleanupRepository)(nil)
)
