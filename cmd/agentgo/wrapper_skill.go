package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/skillruntime"
	skillruntimeSpec "github.com/flexigpt/flexigpt-app/internal/skillruntime/spec"
	"github.com/flexigpt/flexigpt-app/internal/skillstore"
	"github.com/flexigpt/flexigpt-app/internal/skillstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/workspace/skilladapter"
)

// SkillStoreWrapper exposes SkillStore APIs to Wails bindings (same pattern as other stores).
type SkillStoreWrapper struct {
	store             *skillstore.SkillStore
	runtime           *skillruntime.SkillRuntime
	installedProvider skillruntime.Provider
	provider          skillruntime.Provider
}

func InitSkillStoreWrapper(
	s *SkillStoreWrapper,
	skillsDir string,
	workspaceSkills *skilladapter.Adapter,
) error {
	if s == nil {
		return errors.New("skill store wrapper is nil")
	}
	st, err := skillstore.NewSkillStore(skillsDir)
	if err != nil {
		return err
	}
	runtimeOptions := []skillruntime.SkillRuntimeOption{}
	if workspaceSkills != nil {
		runtimeOptions = append(
			runtimeOptions,
			skillruntime.WithWorkspaceSkillAdapter(workspaceSkills),
		)
	}
	rt, err := skillruntime.NewSkillRuntime(st, runtimeOptions...)
	if err != nil {
		st.Close()
		return err
	}
	installed, err := skillruntime.NewInstalled(st, rt)
	if err != nil {
		st.Close()
		return err
	}
	s.store = st
	s.runtime = rt
	s.installedProvider = installed
	s.provider = installed
	return nil
}

func InitAggregateSkillProvider(
	s *SkillStoreWrapper,
) error {
	if s == nil || s.runtime == nil || s.installedProvider == nil {
		return errors.New("skill wrapper is not initialized")
	}
	workspaceProvider, err := skillruntime.NewWorkspace(s.runtime)
	if err != nil {
		return err
	}
	aggregate, err := newAggregateSkillProvider(
		s.installedProvider,
		workspaceProvider,
	)
	if err != nil {
		return err
	}
	s.provider = aggregate
	return nil
}

func mutateInstalledSkill[T any](
	ctx context.Context,
	wrapper *SkillStoreWrapper,
	mutation func() (T, error),
) (T, error) {
	var zero T
	if wrapper == nil || wrapper.store == nil || wrapper.runtime == nil {
		return zero, errors.New("skill wrapper is not initialized")
	}
	response, err := mutation()
	if err != nil {
		return zero, err
	}
	if err := wrapper.runtime.ResyncInstalled(ctx); err != nil {
		return zero, fmt.Errorf("sync installed Skills: %w", err)
	}
	return response, nil
}

func (s *SkillStoreWrapper) PutSkillBundle(
	req *spec.PutSkillBundleRequest,
) (*spec.PutSkillBundleResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PutSkillBundleResponse, error) {
		ctx := context.Background()
		return mutateInstalledSkill(ctx, s, func() (*spec.PutSkillBundleResponse, error) {
			return s.store.PutSkillBundle(ctx, req)
		})
	})
}

func (s *SkillStoreWrapper) PatchSkillBundle(
	req *spec.PatchSkillBundleRequest,
) (*spec.PatchSkillBundleResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PatchSkillBundleResponse, error) {
		ctx := context.Background()
		return mutateInstalledSkill(ctx, s, func() (*spec.PatchSkillBundleResponse, error) {
			return s.store.PatchSkillBundle(ctx, req)
		})
	})
}

func (s *SkillStoreWrapper) DeleteSkillBundle(
	req *spec.DeleteSkillBundleRequest,
) (*spec.DeleteSkillBundleResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.DeleteSkillBundleResponse, error) {
		ctx := context.Background()
		return mutateInstalledSkill(ctx, s, func() (*spec.DeleteSkillBundleResponse, error) {
			return s.store.DeleteSkillBundle(ctx, req)
		})
	})
}

func (s *SkillStoreWrapper) ListSkillBundles(
	req *spec.ListSkillBundlesRequest,
) (*spec.ListSkillBundlesResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ListSkillBundlesResponse, error) {
		return s.store.ListSkillBundles(context.Background(), req)
	})
}

func (s *SkillStoreWrapper) PutSkill(req *spec.PutSkillRequest) (*spec.PutSkillResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PutSkillResponse, error) {
		ctx := context.Background()
		return mutateInstalledSkill(ctx, s, func() (*spec.PutSkillResponse, error) {
			return s.store.PutSkill(ctx, req)
		})
	})
}

func (s *SkillStoreWrapper) PutSkillArtifact(
	req *spec.PutSkillArtifactRequest,
) (*spec.PutSkillArtifactResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PutSkillArtifactResponse, error) {
		ctx := context.Background()
		return mutateInstalledSkill(ctx, s, func() (*spec.PutSkillArtifactResponse, error) {
			return s.store.PutSkillArtifact(ctx, req)
		})
	})
}

func (s *SkillStoreWrapper) PatchSkill(req *spec.PatchSkillRequest) (*spec.PatchSkillResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PatchSkillResponse, error) {
		ctx := context.Background()
		return mutateInstalledSkill(ctx, s, func() (*spec.PatchSkillResponse, error) {
			return s.store.PatchSkill(ctx, req)
		})
	})
}

func (s *SkillStoreWrapper) DeleteSkill(req *spec.DeleteSkillRequest) (*spec.DeleteSkillResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.DeleteSkillResponse, error) {
		ctx := context.Background()
		return mutateInstalledSkill(ctx, s, func() (*spec.DeleteSkillResponse, error) {
			return s.store.DeleteSkill(ctx, req)
		})
	})
}

func (s *SkillStoreWrapper) GetSkill(req *spec.GetSkillRequest) (*spec.GetSkillResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.GetSkillResponse, error) {
		return s.store.GetSkill(context.Background(), req)
	})
}

func (s *SkillStoreWrapper) ListSkills(req *spec.ListSkillsRequest) (*spec.ListSkillsResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ListSkillsResponse, error) {
		return s.store.ListSkills(context.Background(), req)
	})
}

func (s *SkillStoreWrapper) CreateSkillSession(
	req *skillruntimeSpec.CreateSkillSessionRequest,
) (*skillruntimeSpec.CreateSkillSessionResponse, error) {
	return middleware.WithRecoveryResp(func() (*skillruntimeSpec.CreateSkillSessionResponse, error) {
		return s.runtime.CreateSkillSession(context.Background(), req)
	})
}

func (s *SkillStoreWrapper) CloseSkillSession(
	req *skillruntimeSpec.CloseSkillSessionRequest,
) (*skillruntimeSpec.CloseSkillSessionResponse, error) {
	return middleware.WithRecoveryResp(func() (*skillruntimeSpec.CloseSkillSessionResponse, error) {
		return s.runtime.CloseSkillSession(context.Background(), req)
	})
}

func (s *SkillStoreWrapper) GetSkillsPrompt(
	req *skillruntimeSpec.GetSkillsPromptRequest,
) (*skillruntimeSpec.GetSkillsPromptResponse, error) {
	return middleware.WithRecoveryResp(func() (*skillruntimeSpec.GetSkillsPromptResponse, error) {
		return s.runtime.GetSkillsPrompt(context.Background(), req)
	})
}

func (s *SkillStoreWrapper) ListRuntimeSkills(
	req *skillruntimeSpec.ListRuntimeSkillsRequest,
) (*skillruntimeSpec.ListRuntimeSkillsResponse, error) {
	return middleware.WithRecoveryResp(func() (*skillruntimeSpec.ListRuntimeSkillsResponse, error) {
		return s.runtime.ListRuntimeSkills(context.Background(), req)
	})
}

func (s *SkillStoreWrapper) RenderSkill(
	req *skillruntimeSpec.RenderSkillRequest,
) (*skillruntimeSpec.RenderSkillResponse, error) {
	return middleware.WithRecoveryResp(func() (*skillruntimeSpec.RenderSkillResponse, error) {
		return s.runtime.RenderSkill(context.Background(), req)
	})
}

func (s *SkillStoreWrapper) InvokeSkillTool(
	req *skillruntimeSpec.InvokeSkillToolRequest,
) (*skillruntimeSpec.InvokeSkillToolResponse, error) {
	return middleware.WithRecoveryResp(func() (*skillruntimeSpec.InvokeSkillToolResponse, error) {
		return s.runtime.InvokeSkillTool(context.Background(), req)
	})
}

func (s *SkillStoreWrapper) ListProvidedSkills(
	req *skillruntime.ListProvidedSkillsRequest,
) (*skillruntime.ListProvidedSkillsResponse, error) {
	return middleware.WithRecoveryResp(func() (*skillruntime.ListProvidedSkillsResponse, error) {
		if s == nil || s.provider == nil {
			return nil, errors.New("invalid request")
		}
		scope := skillruntime.Scope{}
		if req != nil {
			scope.WorkspaceRootID = req.WorkspaceRootID
		}
		values, err := s.provider.List(context.Background(), scope)
		if err != nil {
			return nil, err
		}
		return &skillruntime.ListProvidedSkillsResponse{
			Body: &skillruntime.ListProvidedSkillsResponseBody{
				Skills: values,
			},
		}, nil
	})
}

func (s *SkillStoreWrapper) RenderProvidedSkill(
	req *skillruntime.RenderProvidedSkillRequest,
) (*skillruntime.RenderProvidedSkillResponse, error) {
	return middleware.WithRecoveryResp(func() (*skillruntime.RenderProvidedSkillResponse, error) {
		if s == nil || s.provider == nil ||
			req == nil || req.Body == nil {
			return nil, errors.New("invalid request")
		}
		value, err := s.provider.Render(
			context.Background(),
			skillruntime.RenderRequest{
				Scope: skillruntime.Scope{
					WorkspaceRootID: req.Body.WorkspaceRootID,
				},
				Identity:  req.Body.Identity,
				Arguments: req.Body.Arguments,
			},
		)
		if err != nil {
			return nil, err
		}
		return &skillruntime.RenderProvidedSkillResponse{Body: &value}, nil
	})
}

func (s *SkillStoreWrapper) close() {
	if s == nil || s.store == nil {
		return
	}
	s.store.Close()
	s.runtime = nil
	s.store = nil
}
