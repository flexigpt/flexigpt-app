package spec

import (
	"errors"
)

var (
	ErrClosed                    = errors.New("artifactstore: closed")
	ErrNotFound                  = errors.New("artifactstore: not found")
	ErrConflict                  = errors.New("artifactstore: conflict")
	ErrInvalidRequest            = errors.New("artifactstore: invalid request")
	ErrUnsupported               = errors.New("artifactstore: unsupported")
	ErrSourceNotAttached         = errors.New("artifactstore: source is not attached to root")
	ErrContentNotFound           = errors.New("artifactstore: portable content not found")
	ErrDigestMismatch            = errors.New("artifactstore: digest mismatch")
	ErrDriverUnavailable         = errors.New("artifactstore: source driver unavailable")
	ErrMaterializerUnavailable   = errors.New("artifactstore: source materializer unavailable")
	ErrFrontendUnavailable       = errors.New("artifactstore: artifact frontend unavailable")
	ErrVersionMatcherUnavailable = errors.New("artifactstore: version matcher unavailable")
)
