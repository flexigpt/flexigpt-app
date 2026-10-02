package system

import (
	"errors"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/api/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/api/installerapi/topology"
	rootimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/internal/engine/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func TestPrepareTopologyHydrationsAllowsFreshProtectedRoot(
	t *testing.T,
) {
	t.Parallel()

	rootID := root.RootID(
		"0192c4c0-0000-7000-8000-000000000001",
	)
	sourceID := source.SourceID(
		"0192c4c0-0001-7000-8000-000000000001",
	)
	policy, err := rootimpl.NewSetRootPolicy(
		[]root.RootID{rootID},
		nil,
	)
	if err != nil {
		t.Fatalf("NewSetRootPolicy: %v", err)
	}

	components, err := Open(
		t.Context(),
		Config{
			BaseDirectory:      t.TempDir(),
			RootMutationPolicy: policy,
		},
	)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := components.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})

	desired := topology.Hydration{
		InstallerName: "test.installer",
		RootID:        rootID,
		SourceID:      sourceID,
		Fingerprint:   cryptoutil.DigestBytes([]byte("fresh-install")),
	}
	ctx := installerapi.WithPrivilege(t.Context())

	current, err := components.PrepareTopologyHydrations(
		ctx,
		[]topology.Hydration{desired},
	)
	if err != nil {
		t.Fatalf("PrepareTopologyHydrations: %v", err)
	}
	if current[desired.InstallerName] {
		t.Fatal("fresh topology hydration was unexpectedly current")
	}

	if _, err := components.Roots.Get(ctx, rootID); !errors.Is(
		err,
		model.ErrRootNotFound,
	) {
		t.Fatalf("fresh protected root read error=%v, want ErrRootNotFound", err)
	}
}
