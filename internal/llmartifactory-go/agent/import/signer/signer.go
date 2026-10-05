package signer

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const hmacSHA256SignatureBytes = 32

type Sealed struct {
	Envelope    string
	Fingerprint cryptoutil.Digest
}

// Signer authenticates client-carried prepared import payloads. It is
// process-local by design. Application restart invalidates pending envelopes.
type Signer struct {
	key []byte
}

func NewSigner(
	key []byte,
) (*Signer, error) {
	if key == nil {
		value, err := cryptoutil.NewHMACSHA256Key()
		if err != nil {
			return nil, err
		}
		key = value
	}
	if len(key) < cryptoutil.HMACSHA256KeyBytes {
		return nil, fmt.Errorf(
			"%w: prepared import signing key is too short",
			spec.ErrInvalid,
		)
	}
	return &Signer{
		key: append([]byte(nil), key...),
	}, nil
}

func (s *Signer) Seal(
	value any,
) (Sealed, error) {
	if s == nil || len(s.key) == 0 {
		return Sealed{}, spec.ErrClosed
	}

	payload, err := jsonutil.MarshalCanonicalObject(
		value,
		spec.MaxDefinitionBytes,
	)
	if err != nil {
		return Sealed{}, err
	}
	signature, err := cryptoutil.HMACSHA256(s.key, payload)
	if err != nil {
		return Sealed{}, err
	}

	return Sealed{
		Envelope: base64.RawURLEncoding.EncodeToString(payload) +
			"." +
			base64.RawURLEncoding.EncodeToString(signature),
		Fingerprint: cryptoutil.DigestBytes(payload),
	}, nil
}

func (s *Signer) Open(
	envelope string,
	expected cryptoutil.Digest,
	target any,
) (cryptoutil.Digest, error) {
	if s == nil || len(s.key) == 0 {
		return "", spec.ErrClosed
	}
	if target == nil {
		return "", fmt.Errorf(
			"%w: prepared import target is nil",
			spec.ErrInvalid,
		)
	}
	if err := cryptoutil.ValidateDigest(expected); err != nil {
		return "", err
	}

	payloadEncoded, signatureEncoded, found := strings.Cut(
		envelope,
		".",
	)
	if !found || payloadEncoded == "" || signatureEncoded == "" {
		return "", fmt.Errorf(
			"%w: prepared import envelope is malformed",
			spec.ErrInvalid,
		)
	}
	if base64.RawURLEncoding.DecodedLen(
		len(payloadEncoded),
	) > spec.MaxDefinitionBytes {
		return "", fmt.Errorf(
			"%w: prepared import payload exceeds maximum size",
			spec.ErrInvalid,
		)
	}
	if base64.RawURLEncoding.DecodedLen(
		len(signatureEncoded),
	) != hmacSHA256SignatureBytes {
		return "", fmt.Errorf(
			"%w: prepared import signature has invalid size",
			spec.ErrInvalid,
		)
	}

	payload, err := base64.RawURLEncoding.DecodeString(payloadEncoded)
	if err != nil {
		return "", fmt.Errorf(
			"%w: decode prepared import payload: %w",
			spec.ErrInvalid,
			err,
		)
	}
	signature, err := base64.RawURLEncoding.DecodeString(
		signatureEncoded,
	)
	if err != nil {
		return "", fmt.Errorf(
			"%w: decode prepared import signature: %w",
			spec.ErrInvalid,
			err,
		)
	}

	valid, err := cryptoutil.VerifyHMACSHA256(
		s.key,
		payload,
		signature,
	)
	if err != nil {
		return "", err
	}
	if !valid {
		return "", fmt.Errorf(
			"%w: prepared import signature is invalid",
			spec.ErrInvalid,
		)
	}

	fingerprint := cryptoutil.DigestBytes(payload)
	if fingerprint != expected {
		return "", fmt.Errorf(
			"%w: prepared import fingerprint differs from envelope",
			spec.ErrConflict,
		)
	}

	if err := jsonutil.DecodeCanonicalObjectExactInto(
		payload,
		target,
		spec.MaxDefinitionBytes,
	); err != nil {
		return "", err
	}
	return fingerprint, nil
}
