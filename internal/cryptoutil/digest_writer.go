package cryptoutil

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
)

var errNilDigestWriter = errors.New("cryptoutil: nil digest writer")

// DigestWriter incrementally calculates a Digest.
//
// It implements io.Writer and is suitable for bounded streaming reads,
// directory fingerprints, and other inputs that should not be accumulated
// into one in-memory byte slice.
//
// DigestWriter is not safe for concurrent use.
type DigestWriter struct {
	state hash.Hash
}

func NewDigestWriter() *DigestWriter {
	return &DigestWriter{
		state: sha256.New(),
	}
}

func (w *DigestWriter) Write(value []byte) (int, error) {
	if w == nil || w.state == nil {
		return 0, errNilDigestWriter
	}
	return w.state.Write(value)
}

// Digest returns the current digest without changing writer state.
func (w *DigestWriter) Digest() Digest {
	if w == nil || w.state == nil {
		return ""
	}
	return Digest(
		DigestSHA256Prefix + hex.EncodeToString(w.state.Sum(nil)),
	)
}
