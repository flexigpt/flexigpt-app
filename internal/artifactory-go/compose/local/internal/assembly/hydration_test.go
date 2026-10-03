package assembly

import (
	"errors"
	"testing"

	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	rootimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/impl"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func TestPrepareTopologyHydrationsAllowsFreshProtectedRoot(
	t *testing.T,
) {
	t.Parallel()

	rootID := rootModel.RootID(
		"0192c4c0-0000-7000-8000-000000000001",
	)
	sourceID := sourceModel.SourceID(
		"0192c4c0-0001-7000-8000-000000000001",
	)
	policy, err := rootimpl.NewSetRootPolicy(
		[]rootModel.RootID{rootID},
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

	desired := installModel.Hydration{
		InstallerName: "test.installer",
		RootID:        rootID,
		SourceID:      sourceID,
		Fingerprint:   cryptoutil.DigestBytes([]byte("fresh-install")),
	}
	ctx := installFlow.WithPrivilege(t.Context())

	current, err := components.PrepareTopologyHydrations(
		ctx,
		[]installModel.Hydration{desired},
	)
	if err != nil {
		t.Fatalf("PrepareTopologyHydrations: %v", err)
	}
	if current[desired.InstallerName] {
		t.Fatal("fresh topology hydration was unexpectedly current")
	}

	if _, err := components.Roots.Get(ctx, rootID); !errors.Is(
		err,
		spec.ErrRootNotFound,
	) {
		t.Fatalf("fresh protected root read error=%v, want ErrRootNotFound", err)
	}
}
