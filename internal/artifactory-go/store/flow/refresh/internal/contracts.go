package internal

import refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"

type (
	RefreshStateReader = refreshFlow.StateReader
	ArtifactReader     = refreshFlow.ArtifactReader
	Publication        = refreshFlow.Publication
	Publisher          = refreshFlow.Repository
)
