package maprepo

import (
	"errors"
	"runtime"
	"sync"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const maprepoTestRootID basespec.RootID = "019d3150-6a48-7a6b-a34e-d9032342bc31"

func maprepoTestDefinition() definition.Definition {
	return definition.Definition{
		Kind:          "test.artifact",
		SchemaID:      "test.schema",
		SchemaVersion: "v1",
		LogicalName:   "maprepo-example",
		Body:          []byte(`{"z":2,"a":1}`),
	}
}

func TestRepositoryPersistsCanonicalDefinitionsAndSupportsConcurrentReads(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non win test")
	}
	repository, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer repository.Close()

	stored, err := repository.Put(t.Context(), maprepoTestRootID, maprepoTestDefinition())
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if string(stored.Body) != `{"a":1,"z":2}` {
		t.Fatalf("stored body=%s", stored.Body)
	}
	read, err := repository.Get(t.Context(), maprepoTestRootID, stored.Digest)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if read.Digest != stored.Digest || string(read.Body) != string(stored.Body) {
		t.Fatalf("read=%#v stored=%#v", read, stored)
	}
	if _, err := repository.Get(
		t.Context(),
		maprepoTestRootID,
		cryptoutil.DigestBytes([]byte("missing")),
	); !errors.Is(
		err,
		basespec.ErrDefinitionNotFound,
	) {
		t.Fatalf("missing definition error=%v", err)
	}
	if _, err := repository.Get(
		t.Context(),
		"019d3150-6a49-7a6b-a34e-d9032342bc31",
		stored.Digest,
	); !errors.Is(
		err,
		basespec.ErrDefinitionNotFound,
	) {
		t.Fatalf("other root error=%v", err)
	}

	const workers = 24
	var group sync.WaitGroup
	errorsSeen := make(chan error, workers)
	for range workers {
		group.Go(func() {
			value, err := repository.Get(t.Context(), maprepoTestRootID, stored.Digest)
			if err != nil {
				errorsSeen <- err
				return
			}
			if value.Digest != stored.Digest {
				errorsSeen <- errors.New("concurrent read returned another digest")
			}
		})
	}
	group.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		t.Fatalf("concurrent Get: %v", err)
	}

	if err := repository.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := repository.Get(
		t.Context(),
		maprepoTestRootID,
		stored.Digest,
	); !errors.Is(
		err,
		basespec.ErrClosed,
	) {
		t.Fatalf("Get after Close error=%v, want ErrClosed", err)
	}
}

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
	if _, err := definitionFileKey("not-a-uuid", digest); !errors.Is(err, basespec.ErrInvalid) {
		t.Fatalf("invalid root error=%v", err)
	}
}
