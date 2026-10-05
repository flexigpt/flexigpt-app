package iofs

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// ImmutableContentEvidence is application-supplied evidence for a provider
// registered as immutable for the lifetime of one Adapter.
//
// ContentDigest is checked once at registration. Revision is application-owned
// immutable content identity and becomes part of generated Source evidence.
type ImmutableContentEvidence struct {
	Revision      string
	ContentDigest cryptoutil.Digest
}

// ProviderRegistration selects mutable or explicitly immutable behavior for
// one embedded provider key.
type ProviderRegistration struct {
	Filesystem fs.FS
	Immutable  *ImmutableContentEvidence
}

type immutableProvider struct {
	revision      string
	contentDigest cryptoutil.Digest
}

func NewWithRegistrations(
	ctx context.Context,
	registrations map[string]ProviderRegistration,
) (*Adapter, error) {
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: embedded provider registration context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	providers := make(map[string]fs.FS, len(registrations))
	immutable := make(map[string]immutableProvider)

	for key, registration := range registrations {
		if err := spec.ValidateIdentifier(
			"embedded provider key",
			key,
			spec.MaxKindBytes,
		); err != nil {
			return nil, err
		}
		if registration.Filesystem == nil {
			return nil, fmt.Errorf(
				"%w: embedded provider %q is nil",
				spec.ErrInvalid,
				key,
			)
		}
		providers[key] = registration.Filesystem

		if registration.Immutable == nil {
			continue
		}
		if err := spec.ValidateRequiredText(
			"immutable embedded provider revision",
			registration.Immutable.Revision,
			spec.MaxVersionBytes,
		); err != nil {
			return nil, err
		}
		if err := cryptoutil.ValidateDigest(
			registration.Immutable.ContentDigest,
		); err != nil {
			return nil, err
		}

		observed, err := fingerprint(ctx, registration.Filesystem)
		if err != nil {
			return nil, err
		}
		if cryptoutil.Digest(observed) != registration.Immutable.ContentDigest {
			return nil, fmt.Errorf(
				"%w: immutable embedded provider %q content evidence differs",
				spec.ErrDigestMismatch,
				key,
			)
		}

		immutable[key] = immutableProvider{
			revision:      registration.Immutable.Revision,
			contentDigest: registration.Immutable.ContentDigest,
		}
	}

	return &Adapter{
		providers: providers,
		immutable: immutable,
	}, nil
}

func (p immutableProvider) generation(
	providerKey string,
	root spec.Locator,
) (string, error) {
	value, err := cryptoutil.CanonicalDigest(struct {
		Format        string            `json:"format"`
		Provider      string            `json:"provider"`
		Root          spec.Locator      `json:"root"`
		Revision      string            `json:"revision"`
		ContentDigest cryptoutil.Digest `json:"contentDigest"`
	}{
		Format:        "artifact-embedded-immutable/v1",
		Provider:      providerKey,
		Root:          root,
		Revision:      p.revision,
		ContentDigest: p.contentDigest,
	})
	if err != nil {
		return "", err
	}
	return string(value), nil
}
