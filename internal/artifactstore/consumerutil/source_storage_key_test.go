package consumerutil

import (
	"fmt"
	"strings"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func TestFilesystemSourceStorageKeyUsesIdentifierSafeHashSegment(
	t *testing.T,
) {
	const prefix = "skill-path"

	for index := range 128 {
		rootPath := fmt.Sprintf("/fixture/skills/%d", index)
		digest := strings.TrimPrefix(
			string(cryptoutil.DigestBytes([]byte(rootPath))),
			cryptoutil.DigestSHA256Prefix,
		)

		key := FilesystemSourceStorageKey(prefix, rootPath)
		if err := key.Validate(); err != nil {
			t.Fatalf(
				"storage key %q for root %q is invalid: %v",
				key,
				rootPath,
				err,
			)
		}

		want := basespec.StorageKey(
			prefix + "-hash" + digest[:24],
		)
		if key != want {
			t.Fatalf(
				"storage key=%q, want %q",
				key,
				want,
			)
		}

	}
}
