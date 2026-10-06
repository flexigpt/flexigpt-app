package main

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	textAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text"
)

type TextStoreWrapper struct {
	service *textAPI.Service
}

func InitTextStoreWrapper(
	wrapper *TextStoreWrapper,
	resources resourceFlow.API,
	protection root.ProtectionAPI,
) error {
	if wrapper == nil {
		return spec.ErrClosed
	}
	service, err := textAPI.New(resources, protection)
	if err != nil {
		return err
	}
	wrapper.service = service
	return nil
}

func (w *TextStoreWrapper) MaterializeText(
	ref artifactModel.ArtifactRef,
) (textAPI.Materialization, error) {
	return withRecoveryResp(func() (textAPI.Materialization, error) {
		if w == nil || w.service == nil {
			return textAPI.Materialization{}, spec.ErrClosed
		}
		return w.service.Materialize(context.Background(), ref)
	})
}

func (w *TextStoreWrapper) close() {
	if w == nil {
		return
	}
	w.service = nil
}
