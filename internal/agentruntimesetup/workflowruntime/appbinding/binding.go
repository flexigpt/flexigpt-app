package appbinding

import (
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/workflowruntime"
)

const (
	NormalizeURLServiceID      = workflowruntime.NormalizeURLServiceID
	NormalizeURLServiceVersion = workflowruntime.NormalizeURLServiceVersion

	ApplyUnifiedDiffActivityID      = workflowruntime.ApplyUnifiedDiffActivityID
	ApplyUnifiedDiffActivityVersion = workflowruntime.ApplyUnifiedDiffActivityVersion
)

// Register remains only as the compatibility registration point for existing
// composition code. It owns no attachment or text implementation behavior.
func Register(
	registry *workflowruntime.BindingRegistry,
) error {
	if registry == nil {
		return errors.New("application workflow binding registry is unavailable")
	}
	if err := registry.RegisterService(
		workflowruntime.NormalizeURLServiceBinding(),
	); err != nil {
		return err
	}
	return registry.RegisterActivity(
		workflowruntime.ApplyUnifiedDiffActivityBinding(),
	)
}
