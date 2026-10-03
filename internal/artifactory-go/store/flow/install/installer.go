package install

import installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"

// API is the privileged application-composition capability for protected
// Artifact Store topology installation and hydration.
//
// It is intentionally separate from ordinary entity APIs and domain facades.
type API interface {
	installModel.Ensurer
	installModel.HydrationCoordinator
	installModel.PackageHydrationCoordinator
	installModel.CompiledHydrationCoordinator
}
