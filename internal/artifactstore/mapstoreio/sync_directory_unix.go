//go:build !windows

package mapstoreio

import (
	"errors"
	"os"
	"syscall"
)

func SyncDirectory(directory string) error {
	value, err := os.Open(directory)
	if err != nil {
		return err
	}
	syncErr := value.Sync()
	closeErr := value.Close()
	if errors.Is(syncErr, syscall.EINVAL) ||
		errors.Is(syncErr, syscall.ENOTSUP) ||
		errors.Is(syncErr, syscall.EOPNOTSUPP) {
		// Some macOS and network-backed filesystems do not support directory
		// fsync. Publication remains valid; durability is best-effort there.
		syncErr = nil
	}
	return errors.Join(syncErr, closeErr)
}
