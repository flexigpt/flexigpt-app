package cryptoutil

import (
	"encoding/json"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const maxBytes = 16 << 20

func CanonicalDigest(value any) (Digest, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return "", err
	}
	if len(canonical) > maxBytes {
		return "", errors.New("invalid value: canonical value exceeds byte limit")
	}
	return DigestBytes(canonical), nil
}
