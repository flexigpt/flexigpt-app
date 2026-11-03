package internal

import (
	"context"
	"testing"

	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type hydrationRepositoryStub struct {
	hydrations map[string]installModel.Hydration
	purged     []rootModel.RootID
}

func (s *hydrationRepositoryStub) GetTopologyHydration(
	_ context.Context,
	installerName string,
) (installModel.Hydration, bool, error) {
	value, found := s.hydrations[installerName]
	return value, found, nil
}

func (s *hydrationRepositoryStub) PutTopologyHydration(
	_ context.Context,
	value installModel.Hydration,
) error {
	if s.hydrations == nil {
		s.hydrations = make(map[string]installModel.Hydration)
	}
	s.hydrations[value.InstallerName] = value
	return nil
}

func (*hydrationRepositoryStub) ListTopologyPackageHydrations(
	context.Context,
) ([]installModel.PackageHydration, error) {
	return []installModel.PackageHydration{}, nil
}

func (*hydrationRepositoryStub) PutTopologyPackageHydration(
	context.Context,
	installModel.PackageHydration,
) error {
	return nil
}

func (*hydrationRepositoryStub) DeleteTopologyPackageHydration(
	context.Context,
	installModel.PackageHydrationKey,
) error {
	return nil
}

func (s *hydrationRepositoryStub) PurgeTopologyRoot(
	_ context.Context,
	rootID rootModel.RootID,
) error {
	s.purged = append(s.purged, rootID)
	return nil
}

func TestPrepareTopologyHydrationsKeepsSameEndpointForFingerprintDrift(
	t *testing.T,
) {
	rootID := rootModel.RootID(
		"0192c4c0-0000-7000-8000-000000000001",
	)
	sourceID := sourceModel.SourceID(
		"0192c4c0-0001-7000-8000-000000000001",
	)
	previous := installModel.Hydration{
		InstallerName: "catalog",
		RootID:        rootID,
		SourceID:      sourceID,
		Fingerprint:   cryptoutil.DigestBytes([]byte("before")),
	}
	desired := previous
	desired.Fingerprint = cryptoutil.DigestBytes([]byte("after"))

	policy, err := root.NewSetRootPolicy(
		[]rootModel.RootID{rootID},
		nil,
	)
	if err != nil {
		t.Fatalf("create protected-root policy: %v", err)
	}

	repository := &hydrationRepositoryStub{
		hydrations: map[string]installModel.Hydration{
			previous.InstallerName: previous,
		},
	}
	service := &Service{
		metadata:           repository,
		rootMutationPolicy: policy,
	}

	current, err := service.PrepareTopologyHydrations(
		root.WithInstallerPrivilege(t.Context()),
		[]installModel.Hydration{desired},
	)
	if err != nil {
		t.Fatalf("prepare topology hydration: %v", err)
	}

	if value, found := current[desired.InstallerName]; !found || value {
		t.Fatalf(
			"fingerprint-drift installer current state = %v, found = %v; want false, true",
			value,
			found,
		)
	}
	if len(repository.purged) != 0 {
		t.Fatalf(
			"fingerprint-only drift purged Roots %v; want no destructive reset",
			repository.purged,
		)
	}
}

func TestTopologyResetRequiredOnlyForEndpointMigration(t *testing.T) {
	rootA := rootModel.RootID(
		"0192c4c0-0000-7000-8000-000000000001",
	)
	rootB := rootModel.RootID(
		"0192c4c0-0000-7000-8000-000000000002",
	)
	sourceA := sourceModel.SourceID(
		"0192c4c0-0001-7000-8000-000000000001",
	)
	sourceB := sourceModel.SourceID(
		"0192c4c0-0001-7000-8000-000000000002",
	)
	base := installModel.Hydration{
		InstallerName: "catalog",
		RootID:        rootA,
		SourceID:      sourceA,
		Fingerprint:   cryptoutil.DigestBytes([]byte("base")),
	}

	tests := []struct {
		name string
		next installModel.Hydration
		want bool
	}{
		{
			name: "fingerprint only",
			next: installModel.Hydration{
				InstallerName: base.InstallerName,
				RootID:        rootA,
				SourceID:      sourceA,
				Fingerprint:   cryptoutil.DigestBytes([]byte("changed")),
			},
			want: false,
		},
		{
			name: "source identity",
			next: installModel.Hydration{
				InstallerName: base.InstallerName,
				RootID:        rootA,
				SourceID:      sourceB,
				Fingerprint:   base.Fingerprint,
			},
			want: true,
		},
		{
			name: "root identity",
			next: installModel.Hydration{
				InstallerName: base.InstallerName,
				RootID:        rootB,
				SourceID:      sourceA,
				Fingerprint:   base.Fingerprint,
			},
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := topologyResetRequired(base, test.next); got != test.want {
				t.Fatalf(
					"topologyResetRequired() = %v, want %v",
					got,
					test.want,
				)
			}
		})
	}
}
