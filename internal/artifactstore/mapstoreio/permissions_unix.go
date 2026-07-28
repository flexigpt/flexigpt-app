//go:build !windows

package mapstoreio

import (
	"io/fs"
	"os"
)

func applyPrivateMode(location string, mode fs.FileMode) error {
	return os.Chmod(location, mode)
}
