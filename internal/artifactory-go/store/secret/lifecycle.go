package secret

import "context"

// LifecycleAPI is trusted maintenance/lifecycle behavior, not normal binding
// CRUD.
type LifecycleAPI interface {
	DrainSecretGarbage(ctx context.Context) error
	RecoverPending(ctx context.Context) error
}
