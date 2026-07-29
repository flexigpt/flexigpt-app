package discovery

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

func TestDescriptorLoaderMissingAndValidDescriptor(t *testing.T) {
	t.Parallel()

	workspace := plannerTestWorkspace(t)
	sourceValue := descriptorTestSource(workspace)

	missing := &engineTestSnapshot{generation: "generation-missing"}
	loader, err := NewDescriptorLoader(engineTestRuntime{
		getFn: func(context.Context, basespec.RootID, basespec.SourceID) (source.Source, error) {
			return sourceValue, nil
		},
		openFn: func(context.Context, source.Source) (source.Snapshot, error) {
			return missing, nil
		},
	})
	if err != nil {
		t.Fatalf("NewDescriptorLoader: %v", err)
	}
	observation, err := loader.Load(t.Context(), workspace)
	if err != nil {
		t.Fatalf("Load missing descriptor: %v", err)
	}
	if observation.SourceID != sourceValue.ID || observation.Generation != "generation-missing" ||
		len(observation.Preferences.AdditionalLocators) != 0 || missing.confirmed != 1 || missing.closed != 1 {
		t.Fatalf("missing observation=%#v confirmed=%d closed=%d", observation, missing.confirmed, missing.closed)
	}

	content := descriptorTestDocument(t)
	valid := &engineTestSnapshot{
		generation: "generation-valid",
		entries: map[basespec.Locator]source.Entry{
			spec.DescriptorLocator: {
				Locator:   spec.DescriptorLocator,
				Name:      spec.WorkspaceDescriptorFileName,
				SizeBytes: int64(len(content)),
				IsRegular: true,
			},
		},
		contents: map[basespec.Locator][]byte{spec.DescriptorLocator: content},
	}
	loader, err = NewDescriptorLoader(engineTestRuntime{
		getFn: func(context.Context, basespec.RootID, basespec.SourceID) (source.Source, error) {
			return sourceValue, nil
		},
		openFn: func(context.Context, source.Source) (source.Snapshot, error) {
			return valid, nil
		},
	})
	if err != nil {
		t.Fatalf("NewDescriptorLoader valid: %v", err)
	}
	observation, err = loader.Load(t.Context(), workspace)
	if err != nil {
		t.Fatalf("Load descriptor: %v", err)
	}
	if observation.SourceID != sourceValue.ID || observation.Generation != "generation-valid" ||
		!observation.Preferences.IncludeReadme || len(observation.Preferences.AdditionalLocators) != 2 ||
		observation.Preferences.AdditionalLocators[0] != ".flexigpt/docs/guide.md" ||
		observation.Preferences.AdditionalLocators[1] != ".flexigpt/member.md" ||
		len(observation.Preferences.AdditionalRoots) != 1 ||
		observation.Preferences.AdditionalRoots[0].Root != ".flexigpt/docs" ||
		observation.ExpectedContentDigests[".flexigpt/member.md"] == "" ||
		valid.confirmed != 1 || valid.closed != 1 {
		t.Fatalf("valid observation=%#v confirmed=%d closed=%d", observation, valid.confirmed, valid.closed)
	}
}

func TestDescriptorLoaderRejectsBadObservationsAndClosesSnapshots(t *testing.T) {
	t.Parallel()

	workspace := plannerTestWorkspace(t)
	sourceValue := descriptorTestSource(workspace)
	if loader, err := NewDescriptorLoader(nil); !errors.Is(err, spec.ErrInvalidWorkspace) || loader != nil {
		t.Fatalf("NewDescriptorLoader(nil) loader=%#v err=%v", loader, err)
	}

	content := []byte(`[]`)
	snapshot := &engineTestSnapshot{
		generation: "generation-invalid",
		entries: map[basespec.Locator]source.Entry{
			spec.DescriptorLocator: {
				Locator:   spec.DescriptorLocator,
				Name:      spec.WorkspaceDescriptorFileName,
				SizeBytes: int64(len(content)),
				IsRegular: true,
			},
		},
		contents: map[basespec.Locator][]byte{spec.DescriptorLocator: content},
	}
	loader, err := NewDescriptorLoader(engineTestRuntime{
		getFn: func(context.Context, basespec.RootID, basespec.SourceID) (source.Source, error) {
			return sourceValue, nil
		},
		openFn: func(context.Context, source.Source) (source.Snapshot, error) { return snapshot, nil },
	})
	if err != nil {
		t.Fatalf("NewDescriptorLoader: %v", err)
	}
	if _, err := loader.Load(t.Context(), workspace); !errors.Is(err, spec.ErrWorkspaceDefinitionInvalid) {
		t.Fatalf("invalid descriptor error=%v, want ErrWorkspaceDefinitionInvalid", err)
	}
	if snapshot.closed != 1 {
		t.Fatalf("invalid snapshot closed=%d, want 1", snapshot.closed)
	}

	confirmation := errors.New("confirmation failed")
	missing := &engineTestSnapshot{generation: "generation-confirm", confirmErr: confirmation}
	loader, err = NewDescriptorLoader(engineTestRuntime{
		getFn: func(context.Context, basespec.RootID, basespec.SourceID) (source.Source, error) {
			return sourceValue, nil
		},
		openFn: func(context.Context, source.Source) (source.Snapshot, error) { return missing, nil },
	})
	if err != nil {
		t.Fatalf("NewDescriptorLoader confirmation fixture: %v", err)
	}
	if _, err := loader.Load(t.Context(), workspace); !errors.Is(err, confirmation) {
		t.Fatalf("confirmation error=%v", err)
	}
	if missing.closed != 1 {
		t.Fatalf("confirmation snapshot closed=%d", missing.closed)
	}
}

func descriptorTestSource(workspace spec.Workspace) source.Source {
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	return source.Source{
		ID:          workspace.PrimarySourceID,
		RootID:      workspace.Collection.RootID,
		Kind:        fsdir.Kind,
		DisplayName: "Primary",
		Enabled:     true,
		Config:      []byte(`{}`),
		Revision:    1,
		CreatedAt:   now,
		ModifiedAt:  now,
	}
}

func descriptorTestDocument(t *testing.T) []byte {
	t.Helper()
	digest := cryptoutil.DigestBytes([]byte("member"))
	return fmt.Appendf(nil, `{
		"kind":"workspace.collection",
		"schemaID":"workspace.collection.v1",
		"schemaVersion":"v1",
		"logicalName":"workspace",
		"body":{"discovery":{"additionalLocators":["docs/guide.md"],"additionalRoots":[{"root":"docs","recursive":true,"includePatterns":["*.md"]}],"includeReadme":true}},
		"members":[{"locator":"member.md","digest":%q}]
	}`, digest)
}

type engineTestSnapshot struct {
	generation string
	entries    map[basespec.Locator]source.Entry
	contents   map[basespec.Locator][]byte
	statErrors map[basespec.Locator]error
	openErrors map[basespec.Locator]error
	confirmErr error
	closeErr   error
	confirmed  int
	closed     int
}

func (s *engineTestSnapshot) Generation() string { return s.generation }

func (s *engineTestSnapshot) Stat(_ context.Context, locator basespec.Locator) (source.Entry, error) {
	if err := s.statErrors[locator]; err != nil {
		return source.Entry{}, err
	}
	value, found := s.entries[locator]
	if !found {
		return source.Entry{}, basespec.ErrNotFound
	}
	return value, nil
}

func (*engineTestSnapshot) ReadDir(context.Context, basespec.Locator) ([]source.Entry, error) {
	return nil, basespec.ErrNotFound
}

func (s *engineTestSnapshot) Open(_ context.Context, locator basespec.Locator) (io.ReadCloser, error) {
	if err := s.openErrors[locator]; err != nil {
		return nil, err
	}
	value, found := s.contents[locator]
	if !found {
		return nil, basespec.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(append([]byte(nil), value...))), nil
}

func (s *engineTestSnapshot) Confirm(context.Context) error {
	s.confirmed++
	return s.confirmErr
}

func (s *engineTestSnapshot) Close() error {
	s.closed++
	return s.closeErr
}

var errEngineTestUnexpected = errors.New("unexpected engine test dependency call")

type engineTestRuntime struct {
	getFn  func(context.Context, basespec.RootID, basespec.SourceID) (source.Source, error)
	openFn func(context.Context, source.Source) (source.Snapshot, error)
}

func (r engineTestRuntime) Get(
	ctx context.Context,
	rootID basespec.RootID,
	id basespec.SourceID,
) (source.Source, error) {
	if r.getFn == nil {
		return source.Source{}, errEngineTestUnexpected
	}
	return r.getFn(ctx, rootID, id)
}

func (r engineTestRuntime) Open(ctx context.Context, value source.Source) (source.Snapshot, error) {
	if r.openFn == nil {
		return nil, errEngineTestUnexpected
	}
	return r.openFn(ctx, value)
}
