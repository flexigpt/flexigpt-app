package basespec

import "errors"

var (
	ErrClosed               = errors.New("artifact store: closed")
	ErrNotFound             = errors.New("artifact store: not found")
	ErrRootNotFound         = errors.New("artifact store: root not found")
	ErrSourceNotFound       = errors.New("artifact store: source not found")
	ErrArtifactNotFound     = errors.New("artifact store: artifact not found")
	ErrDefinitionNotFound   = errors.New("artifact store: definition not found")
	ErrRefreshStateNotFound = errors.New("artifact store: refresh state not found")
	ErrConflict             = errors.New("artifact store: conflict")
	ErrIdentityConflict     = errors.New("artifact store: identity conflict")
	ErrInvalid              = errors.New("artifact store: invalid")
	ErrUnsupported          = errors.New("artifact store: unsupported")
	ErrSourceUnavailable    = errors.New("artifact store: source unavailable")
	ErrRefreshRequired      = errors.New("artifact store: source refresh required")
	ErrDigestMismatch       = errors.New("artifact store: digest mismatch")
	ErrDecoderUnavailable   = errors.New("artifact store: decoder unavailable")
	ErrAmbiguousDecoder     = errors.New("artifact store: ambiguous decoder")
	ErrReferenceUnresolved  = errors.New("artifact store: reference unresolved")
	ErrLocatorUnresolved    = errors.New("artifact store: locator unresolved")
	ErrLocatorLimitExceeded = errors.New("artifact store: locator limit exceeded")
	ErrProtected            = errors.New("artifact store: protected")
	ErrRetired              = errors.New("artifact store: retired")
)
