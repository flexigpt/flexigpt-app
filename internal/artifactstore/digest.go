package artifactstore

import (
	"crypto/sha256"
	"encoding/hex"
)

// DigestBytes returns the canonical SHA-256 digest representation used by the
// artifact store for arbitrary immutable content.
func DigestBytes(content []byte) Digest {
	sum := sha256.Sum256(content)
	return Digest(DigestSHA256Prefix + hex.EncodeToString(sum[:]))
}
