package main

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/skill/provider"
	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
	"github.com/flexigpt/flexigpt-app/internal/skill/store"
)

// SkillStoreWrapper exposes SkillStore APIs to Wails bindings (same pattern as other stores).
type SkillStoreWrapper struct {
	store             *store.SkillStore
	installedProvider provider.Provider
	provider          provider.Provider
}

func InitSkillStoreWrapper(s *SkillStoreWrapper, skillsDir string) error {
	st, err := store.NewSkillStore(skillsDir)
	if err != nil {
		return err
	}
	s.store = st
	installed, err := provider.NewInstalled(st)
	if err != nil {
		st.Close()
		return err
	}
	s.installedProvider = installed
	s.provider = installed
	return nil
}

func InitAggregateSkillProvider(
	s *SkillStoreWrapper,
	workspaceProvider provider.Provider,
) error {
	aggregate, err := provider.NewAggregate(
		s.installedProvider,
		workspaceProvider,
	)
	if err != nil {
		return err
	}
	s.provider = aggregate
	return nil
}

func (s *SkillStoreWrapper) PutSkillBundle(
	req *spec.PutSkillBundleRequest,
) (*spec.PutSkillBundleResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PutSkillBundleResponse, error) {
		return s.store.PutSkillBundle(context.Background(), req)
	})
}

func (s *SkillStoreWrapper) PatchSkillBundle(
	req *spec.PatchSkillBundleRequest,
) (*spec.PatchSkillBundleResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PatchSkillBundleResponse, error) {
		return s.store.PatchSkillBundle(context.Background(), req)
	})
}

func (s *SkillStoreWrapper) DeleteSkillBundle(
	req *spec.DeleteSkillBundleRequest,
) (*spec.DeleteSkillBundleResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.DeleteSkillBundleResponse, error) {
		return s.store.DeleteSkillBundle(context.Background(), req)
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
		return s.store.PutSkill(context.Background(), req)
	})
}

func (s *SkillStoreWrapper) PutSkillArtifact(
	req *spec.PutSkillArtifactRequest,
) (*spec.PutSkillArtifactResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PutSkillArtifactResponse, error) {
		return s.store.PutSkillArtifact(context.Background(), req)
	})
}

func (s *SkillStoreWrapper) PatchSkill(req *spec.PatchSkillRequest) (*spec.PatchSkillResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PatchSkillResponse, error) {
		return s.store.PatchSkill(context.Background(), req)
	})
}

func (s *SkillStoreWrapper) DeleteSkill(req *spec.DeleteSkillRequest) (*spec.DeleteSkillResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.DeleteSkillResponse, error) {
		return s.store.DeleteSkill(context.Background(), req)
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
	req *spec.CreateSkillSessionRequest,
) (*spec.CreateSkillSessionResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.CreateSkillSessionResponse, error) {
		return s.store.CreateSkillSession(context.Background(), req)
	})
}

func (s *SkillStoreWrapper) CloseSkillSession(
	req *spec.CloseSkillSessionRequest,
) (*spec.CloseSkillSessionResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.CloseSkillSessionResponse, error) {
		return s.store.CloseSkillSession(context.Background(), req)
	})
}

func (s *SkillStoreWrapper) GetSkillsPrompt(
	req *spec.GetSkillsPromptRequest,
) (*spec.GetSkillsPromptResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.GetSkillsPromptResponse, error) {
		return s.store.GetSkillsPrompt(context.Background(), req)
	})
}

func (s *SkillStoreWrapper) ListRuntimeSkills(
	req *spec.ListRuntimeSkillsRequest,
) (*spec.ListRuntimeSkillsResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ListRuntimeSkillsResponse, error) {
		return s.store.ListRuntimeSkills(context.Background(), req)
	})
}

func (s *SkillStoreWrapper) RenderSkill(
	req *spec.RenderSkillRequest,
) (*spec.RenderSkillResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.RenderSkillResponse, error) {
		return s.store.RenderSkill(context.Background(), req)
	})
}

func (s *SkillStoreWrapper) InvokeSkillTool(
	req *spec.InvokeSkillToolRequest,
) (*spec.InvokeSkillToolResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.InvokeSkillToolResponse, error) {
		return s.store.InvokeSkillTool(context.Background(), req)
	})
}

func (s *SkillStoreWrapper) ListProvidedSkills(
	req *provider.ListProvidedSkillsRequest,
) (*provider.ListProvidedSkillsResponse, error) {
	return middleware.WithRecoveryResp(func() (*provider.ListProvidedSkillsResponse, error) {
		if s == nil || s.provider == nil {
			return nil, spec.ErrSkillInvalidRequest
		}
		scope := provider.Scope{}
		if req != nil {
			scope.WorkspaceRootID = req.WorkspaceRootID
		}
		values, err := s.provider.List(context.Background(), scope)
		if err != nil {
			return nil, err
		}
		return &provider.ListProvidedSkillsResponse{
			Body: &provider.ListProvidedSkillsResponseBody{
				Skills: values,
			},
		}, nil
	})
}

func (s *SkillStoreWrapper) RenderProvidedSkill(
	req *provider.RenderProvidedSkillRequest,
) (*provider.RenderProvidedSkillResponse, error) {
	return middleware.WithRecoveryResp(func() (*provider.RenderProvidedSkillResponse, error) {
		if s == nil || s.provider == nil ||
			req == nil || req.Body == nil {
			return nil, spec.ErrSkillInvalidRequest
		}
		value, err := s.provider.Render(
			context.Background(),
			provider.RenderRequest{
				Scope: provider.Scope{
					WorkspaceRootID: req.Body.WorkspaceRootID,
				},
				Identity:  req.Body.Identity,
				Arguments: req.Body.Arguments,
			},
		)
		if err != nil {
			return nil, err
		}
		return &provider.RenderProvidedSkillResponse{Body: &value}, nil
	})
}

func (s *SkillStoreWrapper) close() {
	if s == nil || s.store == nil {
		return
	}
	s.store.Close()
}
