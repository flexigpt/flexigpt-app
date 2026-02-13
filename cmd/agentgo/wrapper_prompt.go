package main

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/prompt/spec"
	promptStore "github.com/flexigpt/flexigpt-app/internal/prompt/store"
)

type PromptTemplateStoreWrapper struct {
	store *promptStore.PromptTemplateStore
}

func InitPromptTemplateStoreWrapper(
	p *PromptTemplateStoreWrapper,
	promptDir string,
) error {
	promptStoreAPI, err := promptStore.NewPromptTemplateStore(
		promptDir,
	)
	if err != nil {
		return err
	}
	p.store = promptStoreAPI
	return nil
}

func (pbw *PromptTemplateStoreWrapper) PutPromptBundle(
	req *spec.PutPromptBundleRequest,
) (*spec.PutPromptBundleResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PutPromptBundleResponse, error) {
		return pbw.store.PutPromptBundle(context.Background(), req)
	})
}

func (pbw *PromptTemplateStoreWrapper) DeletePromptBundle(
	req *spec.DeletePromptBundleRequest,
) (*spec.DeletePromptBundleResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.DeletePromptBundleResponse, error) {
		return pbw.store.DeletePromptBundle(context.Background(), req)
	})
}

func (pbw *PromptTemplateStoreWrapper) PatchPromptBundle(
	req *spec.PatchPromptBundleRequest,
) (*spec.PatchPromptBundleResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PatchPromptBundleResponse, error) {
		return pbw.store.PatchPromptBundle(context.Background(), req)
	})
}

func (pbw *PromptTemplateStoreWrapper) ListPromptBundles(
	req *spec.ListPromptBundlesRequest,
) (*spec.ListPromptBundlesResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ListPromptBundlesResponse, error) {
		return pbw.store.ListPromptBundles(context.Background(), req)
	})
}

func (pbw *PromptTemplateStoreWrapper) PutPromptTemplate(
	req *spec.PutPromptTemplateRequest,
) (*spec.PutPromptTemplateResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PutPromptTemplateResponse, error) {
		return pbw.store.PutPromptTemplate(context.Background(), req)
	})
}

func (pbw *PromptTemplateStoreWrapper) DeletePromptTemplate(
	req *spec.DeletePromptTemplateRequest,
) (*spec.DeletePromptTemplateResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.DeletePromptTemplateResponse, error) {
		return pbw.store.DeletePromptTemplate(context.Background(), req)
	})
}

func (pbw *PromptTemplateStoreWrapper) PatchPromptTemplate(
	req *spec.PatchPromptTemplateRequest,
) (*spec.PatchPromptTemplateResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PatchPromptTemplateResponse, error) {
		return pbw.store.PatchPromptTemplate(context.Background(), req)
	})
}

func (pbw *PromptTemplateStoreWrapper) GetPromptTemplate(
	req *spec.GetPromptTemplateRequest,
) (*spec.GetPromptTemplateResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.GetPromptTemplateResponse, error) {
		return pbw.store.GetPromptTemplate(context.Background(), req)
	})
}

func (pbw *PromptTemplateStoreWrapper) ListPromptTemplates(
	req *spec.ListPromptTemplatesRequest,
) (*spec.ListPromptTemplatesResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ListPromptTemplatesResponse, error) {
		return pbw.store.ListPromptTemplates(context.Background(), req)
	})
}
