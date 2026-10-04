package secret

// Narrow role conformance: no one concrete value is exported as the combined
// binding, plaintext, and lifecycle capability.
var (
	_ API          = (*BindingService)(nil)
	_ RuntimeAPI   = (*RuntimeService)(nil)
	_ LifecycleAPI = (*LifecycleService)(nil)
	_ CleanupAPI   = (*LifecycleService)(nil)
)
