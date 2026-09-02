package runtime

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/flexigpt/llmtools-go"

	"github.com/flexigpt/agentskills-go/provider"
	"github.com/flexigpt/agentskills-go/provider/fs"
	agentskillsRuntime "github.com/flexigpt/agentskills-go/runtime"
	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"
)

// Service is an application-facing wrapper over agentskills-go/runtime.
// It deliberately contains no Artifact Store identity or persistence type.
type Service struct {
	agentRuntime  *agentskillsRuntime.Runtime
	catalogSource CatalogSource

	mu         sync.Mutex
	closed     bool
	catalogs   map[CatalogID]catalogView
	registered map[provider.SkillDef]string
	generation map[CatalogID]uint64
}

type options struct {
	agentRuntime      *agentskillsRuntime.Runtime
	catalogSource     CatalogSource
	runScriptsEnabled bool
}

type Option func(*options) error

func WithCatalogSource(value CatalogSource) Option {
	return func(options *options) error {
		if value == nil {
			return errors.New("skill catalog source is nil")
		}
		options.catalogSource = value
		return nil
	}
}

func WithAgentRuntime(value *agentskillsRuntime.Runtime) Option {
	return func(options *options) error {
		if value == nil {
			return errors.New("agent skills runtime is nil")
		}
		options.agentRuntime = value
		return nil
	}
}

func WithRunScripts(enabled bool) Option {
	return func(options *options) error {
		options.runScriptsEnabled = enabled
		return nil
	}
}

func New(opts ...Option) (*Service, error) {
	cfg := options{}
	for _, option := range opts {
		if option == nil {
			continue
		}
		if err := option(&cfg); err != nil {
			return nil, err
		}
	}

	if cfg.agentRuntime == nil {
		filesystemProvider, err := fs.New(
			fs.WithRunScripts(cfg.runScriptsEnabled),
		)
		if err != nil {
			return nil, err
		}
		cfg.agentRuntime, err = agentskillsRuntime.New(
			agentskillsRuntime.WithProvider(filesystemProvider),
		)
		if err != nil {
			return nil, err
		}
	}

	return &Service{
		agentRuntime:  cfg.agentRuntime,
		catalogSource: cfg.catalogSource,
		catalogs:      map[CatalogID]catalogView{},
		registered:    map[provider.SkillDef]string{},
		generation:    map[CatalogID]uint64{},
	}, nil
}

func (s *Service) SupportsRunScript() bool {
	value, err := s.readyAgentRuntime()
	return err == nil && value.SupportsRunScript()
}

func (s *Service) NewSession(
	ctx context.Context,
	opts ...agentskillsRuntime.SessionOption,
) (agentskillsRuntimeSpec.SessionID, []provider.SkillDef, error) {
	value, err := s.readyAgentRuntime()
	if err != nil {
		return "", nil, err
	}
	return value.NewSession(ctx, opts...)
}

func (s *Service) CloseSession(
	ctx context.Context,
	sessionID agentskillsRuntimeSpec.SessionID,
) error {
	value, err := s.readyAgentRuntime()
	if err != nil {
		return err
	}
	return value.CloseSession(ctx, sessionID)
}

func (s *Service) NewSessionRegistry(
	ctx context.Context,
	sessionID agentskillsRuntimeSpec.SessionID,
	opts ...llmtools.RegistryOption,
) (*llmtools.Registry, error) {
	value, err := s.readyAgentRuntime()
	if err != nil {
		return nil, err
	}
	return value.NewSessionRegistry(ctx, sessionID, opts...)
}

func (s *Service) SkillsPrompt(
	ctx context.Context,
	filter *agentskillsRuntime.SkillFilter,
) (string, error) {
	value, err := s.readyAgentRuntime()
	if err != nil {
		return "", err
	}
	return value.SkillsPrompt(ctx, filter)
}

func (s *Service) ListAgentSkills(
	ctx context.Context,
	filter *agentskillsRuntime.SkillListFilter,
) ([]agentskillsRuntimeSpec.SkillRecord, error) {
	value, err := s.readyAgentRuntime()
	if err != nil {
		return nil, err
	}
	return value.ListSkills(ctx, filter)
}

func (s *Service) RenderAgentSkill(
	ctx context.Context,
	params agentskillsRuntime.RenderSkillParams,
) (agentskillsRuntime.RenderSkillOut, error) {
	value, err := s.readyAgentRuntime()
	if err != nil {
		return agentskillsRuntime.RenderSkillOut{}, err
	}
	return value.RenderSkill(ctx, params)
}

func (s *Service) readyAgentRuntime() (*agentskillsRuntime.Runtime, error) {
	if s == nil {
		return nil, fmt.Errorf("%w: Skill runtime is nil", ErrClosed)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.agentRuntime == nil {
		return nil, ErrClosed
	}
	return s.agentRuntime, nil
}

func (s *Service) beginCatalogSync(
	id CatalogID,
) (CatalogSource, uint64, error) {
	if s == nil {
		return nil, 0, fmt.Errorf("%w: Skill runtime is nil", ErrClosed)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.agentRuntime == nil {
		return nil, 0, ErrClosed
	}
	if s.catalogSource == nil {
		return nil, 0, fmt.Errorf(
			"%w: Skill catalog source is not configured",
			ErrInvalidRequest,
		)
	}
	return s.catalogSource, s.nextGenerationLocked(id), nil
}

func (s *Service) beginCatalogRemoval(
	id CatalogID,
) (uint64, error) {
	if s == nil {
		return 0, fmt.Errorf("%w: Skill runtime is nil", ErrClosed)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.agentRuntime == nil {
		return 0, ErrClosed
	}
	return s.nextGenerationLocked(id), nil
}

func (s *Service) nextGenerationLocked(id CatalogID) uint64 {
	if s.generation == nil {
		s.generation = map[CatalogID]uint64{}
	}
	s.generation[id]++
	return s.generation[id]
}
