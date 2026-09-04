package cryptoutil

import (
	"errors"
	"fmt"
	"regexp"
)

const (
	DigestSHA256Prefix = "sha256:"
)

var (
	digestPattern = regexp.MustCompile(`^` + DigestSHA256Prefix + `[0-9a-f]{64}$`)
	errInvalid    = errors.New("invalid digest")
)

type Digest string

// DigestBytes returns the canonical sha256 digest representation used by the
// artifact store for arbitrary immutable content.
func DigestBytes(content []byte) Digest {
	writer := NewDigestWriter()
	_, _ = writer.Write(content)
	return writer.Digest()
}

func ValidateDigest(value Digest) error {
	if !digestPattern.MatchString(string(value)) {
		return fmt.Errorf(
			"%w: digest must be sha256:<64 lowercase hexadecimal characters>",
			errInvalid,
		)
	}
	return nil
}

func CloneDigest(value *Digest) *Digest {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func IsDigestEqual(
	left *Digest,
	right *Digest,
) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
