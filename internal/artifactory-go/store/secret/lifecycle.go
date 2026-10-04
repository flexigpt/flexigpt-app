package secret

import "context"

// LifecycleAPI is trusted maintenance/lifecycle behavior, not normal binding
// CRUD. CleanupAPI is a still narrower post-commit cleanup capability.
type LifecycleAPI interface {
	DrainSecretGarbage(ctx context.Context) error
	RecoverPending(ctx context.Context) error
}
