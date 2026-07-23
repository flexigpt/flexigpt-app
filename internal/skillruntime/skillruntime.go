package skillruntime

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/flexigpt/agentskills-go"
	"github.com/flexigpt/agentskills-go/fsskillprovider"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/skillstore"
	"github.com/flexigpt/flexigpt-app/internal/workspace/skilladapter"
)

const (
	runtimeResyncTimeout             = 30 * time.Second
	runtimeForegroundValidateTimeout = 15 * time.Second
)

// SkillRuntime owns the in-memory Agent Skills catalog, provider lifecycle,
// sessions, prompt generation, rendering, and tool invocation.
type SkillRuntime struct {
	store             *skillstore.SkillStore
	workspaceSkills   *skilladapter.Adapter
	runtime           *agentskills.Runtime
	runScriptsEnabled bool

	rtResyncMu sync.Mutex

	managedInstalled  runtimeDesiredView
	managedWorkspaces map[artifactstore.RootID]runtimeDesiredView
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

	value := &SkillRuntime{
		store:             store,
		workspaceSkills:   options.workspaceSkills,
		runtime:           options.runtime,
		runScriptsEnabled: options.runScriptsEnabled,
		managedInstalled: runtimeDesiredView{
			definitions: map[agentskillsSpec.SkillDef]string{},
		},
		managedWorkspaces: map[artifactstore.RootID]runtimeDesiredView{},
		managedRuntime:    map[agentskillsSpec.SkillDef]string{},
	}
	value.bestEffortInstalledResync(context.Background(), "init")
	return value, nil
}

func (s *SkillRuntime) Store() *skillstore.SkillStore {
	if s == nil {
		return nil
	}
	return s.store
}

func (s *SkillRuntime) AgentSkillsRuntime() *agentskills.Runtime {
	if s == nil {
		return nil
	}
	return s.runtime
}

// RunScriptsEnabled reports the effective shared filesystem-provider policy.
// Inference composition uses this value only to decide whether
// skills-runscript is advertised to the model.
func (s *SkillRuntime) RunScriptsEnabled() bool {
	if s == nil {
		return false
	}
	return s.runScriptsEnabled
}

func (s *SkillRuntime) ensureConfigured() error {
	if s == nil || s.store == nil || s.runtime == nil {
		return errors.New("Skill runtime is not configured")
	}
	return nil
}
