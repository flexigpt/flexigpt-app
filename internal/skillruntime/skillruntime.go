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
	"github.com/flexigpt/flexigpt-app/internal/skillstore"
	"github.com/flexigpt/flexigpt-app/internal/workspace/skilladapter"
)

const (
	runtimeResyncTimeout             = 30 * time.Second
	runtimeForegroundValidateTimeout = 15 * time.Second
	workspaceSkillProviderType       = "workspace"
)

// SkillRuntime owns the in-memory Agent Skills catalog, provider lifecycle,
// sessions, prompt generation, rendering, and tool invocation.
type SkillRuntime struct {
	store           *skillstore.SkillStore
	workspaceSkills *skilladapter.Adapter
	runtime         *agentskills.Runtime

	rtResyncMu sync.Mutex
}

type skillRuntimeOptions struct {
	runtime         *agentskills.Runtime
	workspaceSkills *skilladapter.Adapter
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
	if options.runtime == nil {
		filesystemProvider, err := fsskillprovider.New(
			fsskillprovider.WithRunScripts(true),
		)
		if err != nil {
			return nil, err
		}
		runtimeOptions := []agentskills.Option{
			agentskills.WithProvider(filesystemProvider),
			agentskills.WithLogger(slog.Default()),
		}
		if options.workspaceSkills != nil {
			runtimeOptions = append(
				runtimeOptions,
				agentskills.WithProvider(
					newWorkspaceAgentProvider(options.workspaceSkills),
				),
			)
		}
		options.runtime, err = agentskills.New(runtimeOptions...)
		if err != nil {
			return nil, err
		}
	} else if options.workspaceSkills != nil &&
		!slices.Contains(options.runtime.ProviderTypes(), workspaceSkillProviderType) {
		return nil, errors.New(
			"custom Agent Skills runtime has no Workspace Skill provider",
		)
	}

	value := &SkillRuntime{
		store:           store,
		workspaceSkills: options.workspaceSkills,
		runtime:         options.runtime,
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

func (s *SkillRuntime) ensureConfigured() error {
	if s == nil || s.store == nil || s.runtime == nil {
		return errors.New("Skill runtime is not configured")
	}
	return nil
}
