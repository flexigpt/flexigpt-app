package spec

import "context"

// PreparedConnection is the runtime-owned result of authorization
// preparation. OAuthHandler is intentionally opaque to the core runtime.
// A concrete transport adapter, such as sdkclient, may interpret it.
type PreparedConnection struct {
	Env     map[string]string
	Headers map[string]string

	SensitiveValues []string
	OAuthHandler    any
}

// ConnectionAuthorizer is a narrow runtime port. The runtime session manager
// knows nothing about settings, secret persistence, artifact storage, OAuth
// loopback listeners, or concrete SDK types.
type ConnectionAuthorizer interface {
	PrepareConnection(
		ctx context.Context,
		config RuntimeConfig,
	) (PreparedConnection, error)

	ConnectionSucceeded(
		ctx context.Context,
		config RuntimeConfig,
	)

	ConnectionFailed(
		ctx context.Context,
		config RuntimeConfig,
		err error,
	)
}
