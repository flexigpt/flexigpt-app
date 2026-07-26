//go:build !windows

package mapstoreio

import (
	"errors"
	"os"
)

func SyncDirectory(directory string) error {
	value, err := os.Open(directory)
	if err != nil {
		return err
	}
	syncErr := value.Sync()
	closeErr := value.Close()
	return errors.Join(syncErr, closeErr)
}
