package artifactstore

import "errors"

var (
	ErrClosed              = errors.New("artifact store: closed")
	ErrNotFound            = errors.New("artifact store: not found")
	ErrConflict            = errors.New("artifact store: conflict")
	ErrInvalid             = errors.New("artifact store: invalid")
	ErrUnsupported         = errors.New("artifact store: unsupported")
	ErrSourceUnavailable   = errors.New("artifact store: source unavailable")
	ErrDefinitionNotFound  = errors.New("artifact store: definition not found")
	ErrDigestMismatch      = errors.New("artifact store: digest mismatch")
	ErrCatalogUnavailable  = errors.New("artifact store: catalog unavailable")
	ErrCatalogStale        = errors.New("artifact store: catalog stale")
	ErrDecoderUnavailable  = errors.New("artifact store: decoder unavailable")
	ErrAmbiguousDecoder    = errors.New("artifact store: ambiguous decoder")
	ErrReferenceUnresolved = errors.New("artifact store: reference unresolved")
)
