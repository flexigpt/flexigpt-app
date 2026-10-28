package internal

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	refreshModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

var errUnexpectedVerificationSnapshotOperation = errors.New(
	"unexpected verification snapshot operation",
)

type verificationSessionTestSnapshot struct {
	generation string
}

func (s verificationSessionTestSnapshot) Generation() string {
	return s.generation
}

func (verificationSessionTestSnapshot) Stat(
	context.Context,
	storeSpec.Locator,
) (sourceModel.Entry, error) {
	return sourceModel.Entry{}, errUnexpectedVerificationSnapshotOperation
}

func (verificationSessionTestSnapshot) ReadDir(
	context.Context,
	storeSpec.Locator,
) ([]sourceModel.Entry, error) {
	return nil, errUnexpectedVerificationSnapshotOperation
}

func (verificationSessionTestSnapshot) Open(
	context.Context,
	storeSpec.Locator,
) (io.ReadCloser, error) {
	return nil, errUnexpectedVerificationSnapshotOperation
}

func (verificationSessionTestSnapshot) Confirm(context.Context) error {
	return nil
}

func (verificationSessionTestSnapshot) Close() error {
	return nil
}

type verificationSessionTestInspector struct {
	inspection refreshModel.Inspection
}

func (i verificationSessionTestInspector) InspectSourceMetadata(
	context.Context,
	sourceModel.Source,
) (refreshModel.Inspection, error) {
	return i.inspection.Clone(), nil
}

func TestVerificationSessionSetupAllowsAdmissionMetadataDrift(
	t *testing.T,
) {
	session := &verificationSession{
		service: &Service{
			refresh: verificationSessionTestInspector{
				inspection: refreshModel.Inspection{
					State: refreshModel.State{
						SourceGeneration: "generation-a",
					},
					SourceRevisionChanged: true,
					DiscoveryChanged:      true,
					DecoderChanged:        true,
				},
			},
		},
	}

	_, err := session.ensureRefreshCurrentLocked(
		t.Context(),
		&verificationSessionSource{
			source: sourceModel.Source{
				ID: "source-a",
			},
			snapshot: verificationSessionTestSnapshot{
				generation: "generation-a",
			},
		},
	)
	if err != nil {
		t.Fatalf(
			"setup rejected unchanged source content: %v",
			err,
		)
	}
}

func TestVerificationSessionSetupRejectsGenerationDrift(
	t *testing.T,
) {
	session := &verificationSession{
		service: &Service{
			refresh: verificationSessionTestInspector{
				inspection: refreshModel.Inspection{
					State: refreshModel.State{
						SourceGeneration: "generation-a",
					},
				},
			},
		},
	}

	_, err := session.ensureRefreshCurrentLocked(
		t.Context(),
		&verificationSessionSource{
			source: sourceModel.Source{
				ID: "source-a",
			},
			snapshot: verificationSessionTestSnapshot{
				generation: "generation-b",
			},
		},
	)
	if !errors.Is(err, storeSpec.ErrRefreshRequired) {
		t.Fatalf(
			"setup error = %v, want ErrRefreshRequired",
			err,
		)
	}
}

func TestSameSourceContentIdentityIgnoresAdmissionMetadata(
	t *testing.T,
) {
	observed := sourceModel.Source{
		ID:             "source-a",
		RootID:         "root-a",
		RootStorageKey: "root-storage",
		StorageKey:     "source-storage",
		Kind:           "fs-directory",
		Config:         json.RawMessage(`{"rootPath":"/workspace"}`),
	}
	current := observed.Clone()
	current.Revision = 2
	current.DisplayName = "Renamed source"
	current.Discovery = sourceModel.DiscoverySpec{
		ExplicitLocators: []storeSpec.Locator{"workspace.yaml"},
	}

	if !sameSourceContentIdentity(observed, current) {
		t.Fatal("metadata-only source change changed content identity")
	}

	current.Config = json.RawMessage(`{"rootPath":"/other-workspace"}`)
	if sameSourceContentIdentity(observed, current) {
		t.Fatal("source configuration change did not change content identity")
	}
}
