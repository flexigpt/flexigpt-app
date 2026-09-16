package installerapi

import "github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"

// API is the privileged application-composition capability for protected
// Artifact Store topology installation and hydration.
//
// It is intentionally separate from ordinary entity APIs and domain facades.
type API interface {
	topology.Ensurer
	topology.HydrationCoordinator
	topology.PackageHydrationCoordinator
}
