package skillruntime

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/flexigpt/agentskills-go"
	"github.com/flexigpt/agentskills-go/fsskillprovider"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/skillstore"
	"github.com/flexigpt/flexigpt-app/internal/workspace/skilladapter"
)

const (
	runtimeResyncTimeout             = 30 * time.Second
	runtimeForegroundValidateTimeout = 15 * time.Second
	workspaceReconcileMaxAttempts    = 5
	workspaceReconcileInitialDelay   = 250 * time.Millisecond
	workspaceReconcileMaximumDelay   = 4 * time.Second
)

// SkillRuntime owns the in-memory Agent Skills catalog, provider lifecycle,
// sessions, prompt generation, rendering, and tool invocation.
type SkillRuntime struct {
	store             *skillstore.SkillStore
	workspaceSkills   *skilladapter.Adapter
	runtime           *agentskills.Runtime
	runScriptsEnabled bool

	rtResyncMu sync.Mutex

	lifecycleMu       sync.Mutex
	closed            bool
	backgroundContext context.Context
	backgroundCancel  context.CancelFunc
	backgroundWG      sync.WaitGroup

	installedResyncMu         sync.Mutex
	installedResyncPending    bool
	installedResyncGeneration uint64

	workspaceRequestMu sync.Mutex
	workspaceRequests  map[basespec.CollectionRef]workspaceReconcileRequest

	managedInstalled  runtimeDesiredView
	managedWorkspaces map[basespec.CollectionRef]runtimeDesiredView
	managedRuntime    map[agentskillsSpec.SkillDef]string
}

type skillRuntimeOptions struct {
	runtime              *agentskills.Runtime
	workspaceSkills      *skilladapter.Adapter
	runScriptsEnabled    bool
	runScriptsConfigured bool
}

type SkillRuntimeOption func(*skillRuntimeOptions) error

func WithRuntime(value *agentskills.Runtime) SkillRuntimeOption {
	return func(options *skillRuntimeOptions) error {
		if value == nil {
			return errors.New("Skill runtime is nil")
		}
		options.runtime = value
		return nil
	}
}

func WithWorkspaceSkillAdapter(
	value *skilladapter.Adapter,
) SkillRuntimeOption {
	return func(options *skillRuntimeOptions) error {
		if value == nil {
			return errors.New("Workspace Skill adapter is nil")
		}
		options.workspaceSkills = value
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

func NewSkillRuntime(
	store *skillstore.SkillStore,
	opts ...SkillRuntimeOption,
) (*SkillRuntime, error) {
	if store == nil {
		return nil, errors.New("Skill Store is nil")
	}

	options := skillRuntimeOptions{}
	for _, option := range opts {
		if option == nil {
			continue
		}
		if err := option(&options); err != nil {
			return nil, err
		}
	}
	//nolint:gocritic // Dont want a switch.
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
	} else if !slices.Contains(
		options.runtime.ProviderTypes(),
		fsskillprovider.Type,
	) {
		return nil, errors.New(
			"custom Agent Skills runtime has no filesystem Skill provider",
		)
	} else if !options.runScriptsConfigured {
		// A custom runtime may use a different filesystem-provider policy. Do
		// not advertise script execution unless the composing application says
		// that it is enabled.
		options.runScriptsEnabled = false
	}

	backgroundContext, backgroundCancel := context.WithCancel(context.Background())

	value := &SkillRuntime{
		store:             store,
		workspaceSkills:   options.workspaceSkills,
		runtime:           options.runtime,
		runScriptsEnabled: options.runScriptsEnabled,
		managedInstalled: runtimeDesiredView{
			definitions: map[agentskillsSpec.SkillDef]string{},
		},
		managedWorkspaces: map[basespec.CollectionRef]runtimeDesiredView{},
		managedRuntime:    map[agentskillsSpec.SkillDef]string{},
		backgroundContext: backgroundContext,
		backgroundCancel:  backgroundCancel,
		workspaceRequests: map[basespec.CollectionRef]workspaceReconcileRequest{},
	}
	value.bestEffortInstalledResync(context.Background(), "init")
	return value, nil
}

func (s *SkillRuntime) Store() *skillstore.SkillStore {
	if s == nil || s.isClosed() {
		return nil
	}
	return s.store
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

// ManagedWorkspaceRefs returns the process-local Workspace partitions that
// currently have derived Agent Skills runtime state. Application composition
// uses this to remove a partition after a Workspace was retired or purged
// through a generic Artifact Store lifecycle operation.
func (s *SkillRuntime) ManagedWorkspaceRefs() []basespec.CollectionRef {
	if s == nil {
		return nil
	}
	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()

	output := make([]basespec.CollectionRef, 0, len(s.managedWorkspaces))
	for ref := range s.managedWorkspaces {
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

// RequestInstalledResync schedules one coalesced reconciliation of the legacy
// standalone Skill Store partition. It intentionally does not make a durable
// Skill Store mutation fail after persistence has committed.
func (s *SkillRuntime) RequestInstalledResync() {
	if s == nil {
		return
	}
	parent, started := s.beginBackground()
	if !started {
		return
	}

	s.installedResyncMu.Lock()
	s.installedResyncGeneration++
	if s.installedResyncPending {
		s.installedResyncMu.Unlock()
		s.endBackground()
		return
	}
	s.installedResyncPending = true
	s.installedResyncMu.Unlock()

	go func() {
		defer s.endBackground()
		defer func() {
			s.installedResyncMu.Lock()
			s.installedResyncPending = false
			s.installedResyncMu.Unlock()
		}()

		for {
			if parent.Err() != nil {
				return
			}
			s.installedResyncMu.Lock()
			generation := s.installedResyncGeneration
			s.installedResyncMu.Unlock()

			s.bestEffortInstalledResync(
				parent,
				"installed-store mutation",
			)

			s.installedResyncMu.Lock()
			current := s.installedResyncGeneration
			s.installedResyncMu.Unlock()
			if current == generation {
				return
			}
		}
	}()
}

// Close stops derived-state reconciliation before the owning Skill Store is
// closed. Runtime registration is process-local, so no durable mutation occurs
// during shutdown.
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
	s.managedInstalled = newRuntimeDesiredView()
	s.managedWorkspaces = map[basespec.CollectionRef]runtimeDesiredView{}
	s.managedRuntime = map[agentskillsSpec.SkillDef]string{}
	s.rtResyncMu.Unlock()

	s.workspaceRequestMu.Lock()
	s.workspaceRequests = nil
	s.workspaceRequestMu.Unlock()
}

func (s *SkillRuntime) ensureConfigured() error {
	if s == nil {
		return errors.New("Skill runtime is not configured")
	}
	s.lifecycleMu.Lock()
	closed := s.closed
	configured := s.store != nil && s.runtime != nil
	s.lifecycleMu.Unlock()
	if closed || !configured {
		return errors.New("Skill runtime is not configured")
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
