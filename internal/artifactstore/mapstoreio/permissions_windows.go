//go:build windows

package mapstoreio

import "io/fs"

// Windows access control is governed by the directory DACL inherited from
// the application data root. POSIX mode bits cannot express that policy.
func applyPrivateMode(_ string, _ fs.FileMode) error {
	return nil
}
