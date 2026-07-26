//go:build windows

package mapstoreio

// Windows does not provide the same portable directory fsync operation as
// Unix. MapStore's Windows replacement path uses MoveFileEx with
// MOVEFILE_WRITE_THROUGH for file publication.
func SyncDirectory(_ string) error {
	return nil
}
