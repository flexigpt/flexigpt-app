package skillruntime

import (
	"context"
	"fmt"
	"sync"

	skillRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/skill"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// Resolver synchronizes selected source-backed Skills and verifies that their
// current definitions are registered in Skill Runtime.
type Resolver struct {
	source  *CatalogAdapter
	runtime *skillRuntime.Service

	syncMu  sync.Mutex
	syncing map[rootModel.RootID]*rootCatalogSync
}

type rootCatalogSync struct {
	done chan struct{}
	err  error
}

func NewResolver(
	source *CatalogAdapter,
	runtime *skillRuntime.Service,
) (*Resolver, error) {
	if source == nil || runtime == nil {
		return nil, fmt.Errorf(
			"%w: Skill resolver dependencies are incomplete",
			storeSpec.ErrInvalid,
		)
	}

	return &Resolver{
		source:  source,
		runtime: runtime,
		syncing: make(map[rootModel.RootID]*rootCatalogSync),
	}, nil
}

func (a *Resolver) RunScriptsEnabled() bool {
	return a != nil && a.runtime != nil && a.runtime.SupportsRunScript()
}

// ResolveSkills synchronizes each selected Root once, then verifies selected
// current revisions against registrations installed in Skill Runtime.
func (a *Resolver) ResolveSkills(
	ctx context.Context,
	refs []artifactModel.ArtifactRef,
) (ResolvedSkills, error) {
	if a == nil || a.source == nil || a.runtime == nil {
		return ResolvedSkills{}, storeSpec.ErrClosed
	}

	roots, err := selectionRoots(refs)
	if err != nil {
		return ResolvedSkills{}, err
	}
	for _, rootID := range roots {
		if err := ctx.Err(); err != nil {
			return ResolvedSkills{}, err
		}
		if err := a.syncRoot(ctx, rootID); err != nil {
			return ResolvedSkills{}, fmt.Errorf(
				"synchronize Skill Root %q: %w",
				rootID,
				err,
			)
		}
	}

	records, err := a.source.readRecords(ctx, refs)
	if err != nil {
		return ResolvedSkills{}, fmt.Errorf(
			"%w: read selected Skill Artifacts: %w",
			storeSpec.ErrReferenceUnresolved,
			err,
		)
	}
	values, err := a.source.materializeRecords(ctx, records)
	if err != nil {
		return ResolvedSkills{}, err
	}

	for _, value := range values {
		if !a.runtime.IsRegistered(skillRuntime.SkillRegistration{
			Definition: value.Definition,
			Revision:   value.Version,
		}) {
			return ResolvedSkills{}, fmt.Errorf(
				"%w: Skill %s/%s revision %q is not registered",
				storeSpec.ErrReferenceUnresolved,
				value.Artifact.RootID,
				value.Artifact.ArtifactID,
				value.Version,
			)
		}
	}
	return selectionFromValues(values)
}

func (a *Resolver) SyncRootCatalog(
	ctx context.Context,
	rootID rootModel.RootID,
) error {
	if a == nil || a.source == nil || a.runtime == nil {
		return storeSpec.ErrClosed
	}
	if err := rootID.Validate(); err != nil {
		return err
	}
	return a.syncRoot(ctx, rootID)
}

// syncRoot coalesces overlapping observations but intentionally does not cache
// successful observations across requests. Runtime owns reconciliation state.
func (a *Resolver) syncRoot(
	ctx context.Context,
	rootID rootModel.RootID,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	a.syncMu.Lock()
	if current, found := a.syncing[rootID]; found {
		a.syncMu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-current.done:
			return current.err
		}
	}

	current := &rootCatalogSync{done: make(chan struct{})}
	a.syncing[rootID] = current
	a.syncMu.Unlock()

	err := a.runtime.SyncCatalog(
		ctx,
		skillRuntime.CatalogID(artifactRootCatalogPrefix+string(rootID)),
	)

	a.syncMu.Lock()
	current.err = err
	delete(a.syncing, rootID)
	close(current.done)
	a.syncMu.Unlock()
	return err
}
