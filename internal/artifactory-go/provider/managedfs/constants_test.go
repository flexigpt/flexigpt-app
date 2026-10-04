package managedfs

import "testing"

func TestManagedFilesystemLayoutConstantsRemainProviderOwned(t *testing.T) {
	t.Parallel()

	if managedDirectoryMode != 0o750 {
		t.Fatalf("managedDirectoryMode=%#o", managedDirectoryMode)
	}
	for _, prefix := range []string{
		managedPackageTemporaryPrefix,
		managedPackagePreviousPrefix,
		managedPackageRemovalPrefix,
	} {
		if prefix == "" {
			t.Fatal("managed package staging prefix is empty")
		}
	}
}
