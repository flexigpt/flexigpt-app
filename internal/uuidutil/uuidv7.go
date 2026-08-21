package uuidutil

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"
	"uuid"
)

var errInvalid = errors.New("invalid UUIDv7")

func NewUUIDv7() string {
	return uuid.NewV7().String()
}

func ValidateUUIDv7(value string) error {
	id, err := uuid.Parse(value)
	if err != nil {
		return fmt.Errorf("%w: %w", errInvalid, err)
	}

	return isUUIDv7(id)
}

func GetUUIDv7UnixTime(id string) (t time.Time, err error) {
	sec, nsec, err := getv7Time(id)
	if err != nil {
		return t, err
	}
	return time.Unix(sec, nsec).UTC(), nil
}

func getv7Time(idStr string) (sec, nsec int64, err error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: %w", errInvalid, err)
	}

	err = isUUIDv7(id)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: %w", errInvalid, err)
	}

	t := binary.BigEndian.Uint64(id[:8])
	sec = int64(t>>16) * 10000
	nsec = (sec % 10000000) * 100
	sec /= 10000000
	return sec, nsec, nil
}

func isUUIDv7(id uuid.UUID) error {
	v := id[6] >> 4
	if v != 7 {
		return fmt.Errorf("bad uuid version: %d", v)
	}
	return nil
}
