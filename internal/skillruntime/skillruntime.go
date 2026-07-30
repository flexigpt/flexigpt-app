package skillruntime

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/flexigpt/agentskills-go"
	"github.com/flexigpt/agentskills-go/fsskillprovider"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
)

const (
	runtimeResyncTimeout             = 30 * time.Second
	runtimeForegroundValidateTimeout = 15 * time.Second
	collectionReconcileMaxAttempts   = 5
	collectionReconcileInitialDelay  = 250 * time.Millisecond
	collectionReconcileMaximumDelay  = 4 * time.Second
)

// SkillRuntime owns the in-memory Agent Skills catalog, provider lifecycle,
// sessions, prompt generation, rendering, and tool invocation.
type SkillRuntime struct {
	resolver          *ArtifactRouter
	runtime           *agentskills.Runtime
	runScriptsEnabled bool

	rtResyncMu sync.Mutex

	lifecycleMu       sync.Mutex
	closed            bool
	backgroundContext context.Context
	backgroundCancel  context.CancelFunc
	backgroundWG      sync.WaitGroup

	collectionRequestMu sync.Mutex
	collectionRequests  map[collection.CollectionRef]collectionReconcileRequest

	managedCollections map[collection.CollectionRef]runtimeDesiredView
	managedRuntime     map[agentskillsSpec.SkillDef]string
}

type skillRuntimeOptions struct {
	runtime              *agentskills.Runtime
	resolver             *ArtifactRouter
	runScriptsEnabled    bool
	runScriptsConfigured bool
}

type SkillRuntimeOption func(*skillRuntimeOptions) error

func WithRuntime(value *agentskills.Runtime) SkillRuntimeOption {
	return func(options *skillRuntimeOptions) error {
		if value == nil {
			return errors.New("skill runtime is nil")
		}
		options.runtime = value
		return nil
	}
}

func WithArtifactResolver(
	value *ArtifactRouter,
) SkillRuntimeOption {
	return func(options *skillRuntimeOptions) error {
		if value == nil {
			return errors.New("artifact skill resolver is nil")
		}
		options.resolver = value
		return nil
	}
}

// WithRunScripts configures the shared filesystem-provider execution policy.
// Workspace filesystem skills and installed filesystem skills always use this
// same provider and therefore this same policy.
func WithRunScripts(enabled bool) SkillRuntimeOption {
	return func(options *skillRuntimeOptions) error {
		options.runScriptsEnabled = enabled
		options.runScriptsConfigured = true
		return nil
	}
}

// NewSkillRuntime creates an Artifact-backed Agent Skills runtime. Durable
// Skill identity is always artifact.ArtifactRef; no standalone Skill Store,
// bundle slug, legacy Skill ID, or source location enters this package.
func NewSkillRuntime(
	opts ...SkillRuntimeOption,
) (*SkillRuntime, error) {
	options := skillRuntimeOptions{}
	for _, option := range opts {
		if option != nil {
			if err := option(&options); err != nil {
				return nil, err
			}
		}
	}
	if options.resolver == nil {
		return nil, errors.New("artifact skill resolver is required")
	}
	if options.runtime == nil {
		runScriptsEnabled := true
		if options.runScriptsConfigured {
			runScriptsEnabled = options.runScriptsEnabled
		}
		filesystemProvider, err := fsskillprovider.New(
			fsskillprovider.WithRunScripts(runScriptsEnabled),
		)
		if err != nil {
			return nil, err
		}
		options.runtime, err = agentskills.New(
			agentskills.WithProvider(filesystemProvider),
			agentskills.WithLogger(slog.Default()),
		)
		if err != nil {
			return nil, err
		}
		options.runScriptsEnabled = runScriptsEnabled
	} else if !options.runScriptsConfigured {
		options.runScriptsEnabled = false
	}

	backgroundContext, backgroundCancel := context.WithCancel(context.Background())

	value := &SkillRuntime{
		resolver:           options.resolver,
		runtime:            options.runtime,
		runScriptsEnabled:  options.runScriptsEnabled,
		managedCollections: map[collection.CollectionRef]runtimeDesiredView{},
		managedRuntime:     map[agentskillsSpec.SkillDef]string{},
		backgroundContext:  backgroundContext,
		backgroundCancel:   backgroundCancel,
		collectionRequests: map[collection.CollectionRef]collectionReconcileRequest{},
	}
	return value, nil
}

func (s *SkillRuntime) AgentSkillsRuntime() *agentskills.Runtime {
	if s == nil || s.isClosed() {
		return nil
	}
	return s.runtime
}

// RunScriptsEnabled reports the effective shared filesystem-provider policy.
// Inference composition uses this value only to decide whether
// skills-runscript is advertised to the model.
func (s *SkillRuntime) RunScriptsEnabled() bool {
	if s == nil || s.isClosed() {
		return false
	}
	return s.runScriptsEnabled
}

// ManagedCollectionRefs returns derived runtime partitions. Collection kind is
// deliberately not encoded in a Skill reference; application feature
// synchronizers use their own typed Collection discovery to decide removal.
func (s *SkillRuntime) ManagedCollectionRefs() []collection.CollectionRef {
	if s == nil {
		return nil
	}
	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()

	output := make([]collection.CollectionRef, 0, len(s.managedCollections))
	for ref := range s.managedCollections {
		output = append(output, ref)
	}
	sort.Slice(output, func(left, right int) bool {
		if output[left].RootID != output[right].RootID {
			return output[left].RootID < output[right].RootID
		}
		return output[left].CollectionID < output[right].CollectionID
	})
	return output
}

func (s *SkillRuntime) Close() {
	if s == nil {
		return
	}

	s.lifecycleMu.Lock()
	if s.closed {
		s.lifecycleMu.Unlock()
		return
	}
	s.closed = true
	cancel := s.backgroundCancel
	s.lifecycleMu.Unlock()

	if cancel != nil {
		cancel()
	}
	s.backgroundWG.Wait()

	s.rtResyncMu.Lock()
	if s.runtime != nil && len(s.managedRuntime) != 0 {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			runtimeResyncTimeout,
		)
		remaining, err := s.runtimeApplyDesired(
			ctx,
			s.managedRuntime,
			newRuntimeDesiredView(),
			runtimeApplyBestEffort,
		)
		cancel()
		if err != nil || len(remaining) != 0 {
			slog.Error(
				"remove managed Skill runtime registrations during close",
				"remaining",
				len(remaining),
				"error",
				err,
			)
		}
	}
	s.managedCollections = map[collection.CollectionRef]runtimeDesiredView{}
	s.managedRuntime = map[agentskillsSpec.SkillDef]string{}
	s.rtResyncMu.Unlock()

	s.collectionRequestMu.Lock()
	s.collectionRequests = nil
	s.collectionRequestMu.Unlock()
}

func (s *SkillRuntime) ensureConfigured() error {
	if s == nil {
		return errors.New("skill runtime is not configured")
	}
	s.lifecycleMu.Lock()
	closed := s.closed
	configured := s.resolver != nil && s.runtime != nil
	s.lifecycleMu.Unlock()
	if closed || !configured {
		return errors.New("skill runtime is not configured")
	}
	return nil
}

func (s *SkillRuntime) beginBackground() (context.Context, bool) {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.closed || s.backgroundContext == nil {
		return nil, false
	}
	s.backgroundWG.Add(1)
	return s.backgroundContext, true
}

func (s *SkillRuntime) endBackground() {
	s.backgroundWG.Done()
}

func (s *SkillRuntime) isClosed() bool {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	return s.closed
}
