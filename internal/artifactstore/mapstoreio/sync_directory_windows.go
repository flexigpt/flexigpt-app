//go:build windows

package mapstoreio

// Windows does not expose a portable directory fsync through os.File.
// Publication is still atomic at the writer boundary, but directory metadata
// durability is best-effort after a sudden power loss.
func SyncDirectory(_ string) error {
	return nil
}
