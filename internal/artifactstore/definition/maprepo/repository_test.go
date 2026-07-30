package maprepo

import (
	"strings"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const maprepoTestRootID basespec.RootID = "019d3150-6a48-7a6b-a34e-d9032342bc31"

func TestDefinitionFileKeyRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	digest := cryptoutil.DigestBytes([]byte("content"))
	key, err := definitionFileKey(maprepoTestRootID, digest)
	if err != nil {
		t.Fatalf("definitionFileKey: %v", err)
	}
	provider := definitionPartitionProvider{}
	partition, err := provider.GetPartitionDir(key)
	if err != nil || partition == "" {
		t.Fatalf("GetPartitionDir partition=%q err=%v", partition, err)
	}
	if _, err := definitionFileKey("not-a-uuid", digest); !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("invalid root error=%v", err)
	}
}
