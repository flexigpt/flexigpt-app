package main

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
	"github.com/flexigpt/flexigpt-app/internal/skill/store"
)

// SkillStoreWrapper exposes SkillStore APIs to Wails bindings (same pattern as other stores).
type SkillStoreWrapper struct {
	store *store.SkillStore
}

func InitSkillStoreWrapper(s *SkillStoreWrapper, skillsDir string) error {
	st, err := store.NewSkillStore(skillsDir)
	if err != nil {
		return err
	}
	s.store = st
	return nil
}

func (s *SkillStoreWrapper) Close() {
	if s == nil || s.store == nil {
		return
	}
	s.store.Close()
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
