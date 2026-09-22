package cryptoutil

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
)

const HMACSHA256KeyBytes = 32

func NewHMACSHA256Key() ([]byte, error) {
	key := make([]byte, HMACSHA256KeyBytes)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

func HMACSHA256(
	key []byte,
	value []byte,
) ([]byte, error) {
	if len(key) < HMACSHA256KeyBytes {
		return nil, errors.New("HMAC key is too short")
	}

	mac := hmac.New(sha256.New, key)
	if _, err := mac.Write(value); err != nil {
		return nil, err
	}
	return mac.Sum(nil), nil
}

func VerifyHMACSHA256(
	key []byte,
	value []byte,
	signature []byte,
) (bool, error) {
	expected, err := HMACSHA256(key, value)
	if err != nil {
		return false, err
	}
	return hmac.Equal(expected, signature), nil
}
