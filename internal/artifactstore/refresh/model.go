package refresh

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

// CatalogInspection reports every Store-owned catalog freshness dimension.
type CatalogInspection struct {
	Catalog         catalog.Snapshot
	MetadataChanged bool
	PlanChanged     bool
	DecoderChanged  bool
}

func (i CatalogInspection) IsCurrent() bool {
	return !i.MetadataChanged &&
		!i.PlanChanged &&
		!i.DecoderChanged
}

// Result is the consumer-facing result of one Store-owned Collection refresh.
type Result struct {
	Catalog          catalog.Snapshot
	CreatedArtifacts []basespec.ArtifactID
	UpdatedArtifacts []basespec.ArtifactID
	Diagnostics      []providerapi.Diagnostic
	Candidates       int
}
