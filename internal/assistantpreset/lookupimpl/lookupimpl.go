package lookupimpl

import (
	"context"
	"errors"
	"fmt"

	assistantpresetStore "github.com/flexigpt/flexigpt-app/internal/assistantpreset/store"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	modelpresetSpec "github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	modelpresetStore "github.com/flexigpt/flexigpt-app/internal/modelpreset/store"
	promptSpec "github.com/flexigpt/flexigpt-app/internal/prompt/spec"
	promptStore "github.com/flexigpt/flexigpt-app/internal/prompt/store"
	skillSpec "github.com/flexigpt/flexigpt-app/internal/skill/spec"
	skillStore "github.com/flexigpt/flexigpt-app/internal/skill/store"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
	toolStore "github.com/flexigpt/flexigpt-app/internal/tool/store"
)

func NewAssistantPresetReferenceLookups(
	modelPresetSt *modelpresetStore.ModelPresetStore,
	promptTemplateSt *promptStore.PromptTemplateStore,
	toolSt *toolStore.ToolStore,
	skillSt *skillStore.SkillStore,
) assistantpresetStore.ReferenceLookups {
	return assistantpresetStore.ReferenceLookups{
		ModelPresets: &modelPresetLookupAdapter{
			store: modelPresetSt,
		},
		PromptTemplates: &promptTemplateLookupAdapter{
			store: promptTemplateSt,
		},
		ToolSelections: &toolSelectionLookupAdapter{
			store: toolSt,
		},
		Skills: &skillLookupAdapter{
			store: skillSt,
		},
	}
}

type modelPresetLookupAdapter struct {
	store *modelpresetStore.ModelPresetStore
}

func (a *modelPresetLookupAdapter) GetModelPresetSummary(
	ctx context.Context,
	ref modelpresetSpec.ModelPresetRef,
) (assistantpresetStore.ModelPresetSummary, error) {
	if a == nil || a.store == nil {
		return assistantpresetStore.ModelPresetSummary{}, errors.New("model preset lookup adapter is not configured")
	}

	if ref.IsZero() {
		return assistantpresetStore.ModelPresetSummary{}, errors.New("model preset ref is zero")
	}

	resp, err := a.store.GetModelPreset(ctx, &modelpresetSpec.GetModelPresetRequest{
		ProviderName:    ref.ProviderName,
		ModelPresetID:   ref.ModelPresetID,
		IncludeDisabled: true,
	})
	if err != nil {
		return assistantpresetStore.ModelPresetSummary{}, err
	}
	if resp == nil || resp.Body == nil {
		return assistantpresetStore.ModelPresetSummary{}, errors.New("empty model preset response")
	}

	return assistantpresetStore.ModelPresetSummary{
		IsEnabled: resp.Body.Provider.IsEnabled && resp.Body.Model.IsEnabled,
	}, nil
}

type promptTemplateLookupAdapter struct {
	store *promptStore.PromptTemplateStore
}

func (a *promptTemplateLookupAdapter) GetPromptTemplateSummary(
	ctx context.Context,
	ref promptSpec.PromptTemplateRef,
) (assistantpresetStore.PromptTemplateSummary, error) {
	if a == nil || a.store == nil {
		return assistantpresetStore.PromptTemplateSummary{}, errors.New(
			"prompt template lookup adapter is not configured",
		)
	}

	if ref.BundleID == "" || ref.TemplateSlug == "" || ref.TemplateVersion == "" {
		return assistantpresetStore.PromptTemplateSummary{}, errors.New("prompt template ref is incomplete")
	}

	bundleEnabled, err := getPromptBundleEnabled(ctx, a.store, ref.BundleID)
	if err != nil {
		return assistantpresetStore.PromptTemplateSummary{}, err
	}

	resp, err := a.store.GetPromptTemplate(ctx, &promptSpec.GetPromptTemplateRequest{
		BundleID:     ref.BundleID,
		TemplateSlug: ref.TemplateSlug,
		Version:      ref.TemplateVersion,
	})
	if err != nil {
		return assistantpresetStore.PromptTemplateSummary{}, err
	}
	if resp == nil || resp.Body == nil {
		return assistantpresetStore.PromptTemplateSummary{}, errors.New("empty prompt template response")
	}

	return assistantpresetStore.PromptTemplateSummary{
		IsEnabled:  bundleEnabled && resp.Body.IsEnabled,
		Kind:       resp.Body.Kind,
		IsResolved: resp.Body.IsResolved,
	}, nil
}

type toolSelectionLookupAdapter struct {
	store *toolStore.ToolStore
}

func (a *toolSelectionLookupAdapter) GetToolSummaryForSelection(
	ctx context.Context,
	selection toolSpec.ToolSelection,
) (assistantpresetStore.ToolSummary, error) {
	if a == nil || a.store == nil {
		return assistantpresetStore.ToolSummary{}, errors.New("tool selection lookup adapter is not configured")
	}
	if selection.ToolRef.BundleID == "" || selection.ToolRef.ToolSlug == "" || selection.ToolRef.ToolVersion == "" {
		return assistantpresetStore.ToolSummary{}, errors.New("tool selection toolRef is incomplete")
	}

	bundleEnabled, err := getToolBundleEnabled(ctx, a.store, selection.ToolRef.BundleID)
	if err != nil {
		return assistantpresetStore.ToolSummary{}, err
	}

	resp, err := a.store.GetTool(ctx, &toolSpec.GetToolRequest{
		BundleID: selection.ToolRef.BundleID,
		ToolSlug: selection.ToolRef.ToolSlug,
		Version:  selection.ToolRef.ToolVersion,
	})
	if err != nil {
		if errors.Is(err, toolSpec.ErrToolDisabled) || errors.Is(err, toolSpec.ErrBundleDisabled) {
			return assistantpresetStore.ToolSummary{IsEnabled: false}, nil
		}
		return assistantpresetStore.ToolSummary{}, err
	}
	if resp == nil || resp.Body == nil {
		return assistantpresetStore.ToolSummary{}, errors.New("empty tool response")
	}

	return assistantpresetStore.ToolSummary{
		IsEnabled: bundleEnabled && resp.Body.IsEnabled,
	}, nil
}

type skillLookupAdapter struct {
	store *skillStore.SkillStore
}

func (a *skillLookupAdapter) GetSkillSummary(
	ctx context.Context,
	ref skillSpec.SkillRef,
) (assistantpresetStore.SkillSummary, error) {
	if a == nil || a.store == nil {
		return assistantpresetStore.SkillSummary{}, errors.New("skill lookup adapter is not configured")
	}

	if ref.BundleID == "" || ref.SkillSlug == "" {
		return assistantpresetStore.SkillSummary{}, errors.New("skill ref is incomplete")
	}

	bundleEnabled, err := getSkillBundleEnabled(ctx, a.store, ref.BundleID)
	if err != nil {
		return assistantpresetStore.SkillSummary{}, err
	}
	resp, err := a.store.GetSkill(ctx, &skillSpec.GetSkillRequest{
		BundleID:        ref.BundleID,
		SkillSlug:       ref.SkillSlug,
		IncludeDisabled: true,
	})
	if err != nil {
		if errors.Is(err, skillSpec.ErrSkillDisabled) || errors.Is(err, skillSpec.ErrSkillBundleDisabled) {
			return assistantpresetStore.SkillSummary{IsEnabled: false}, nil
		}
		return assistantpresetStore.SkillSummary{}, err
	}
	if resp == nil || resp.Body == nil {
		return assistantpresetStore.SkillSummary{}, errors.New("empty skill response")
	}
	if ref.SkillID != "" && resp.Body.ID != ref.SkillID {
		return assistantpresetStore.SkillSummary{}, fmt.Errorf(
			"skill ref id mismatch: got %q, expected %q",
			resp.Body.ID,
			ref.SkillID,
		)
	}

	return assistantpresetStore.SkillSummary{
		IsEnabled: bundleEnabled && resp.Body.IsEnabled,
	}, nil
}

func getPromptBundleEnabled(
	ctx context.Context,
	store *promptStore.PromptTemplateStore,
	bundleID bundleitemutils.BundleID,
) (bool, error) {
	resp, err := store.ListPromptBundles(ctx, &promptSpec.ListPromptBundlesRequest{
		BundleIDs:       []bundleitemutils.BundleID{bundleID},
		IncludeDisabled: true,
		PageSize:        1,
	})
	if err != nil {
		return false, err
	}
	if resp == nil || resp.Body == nil || len(resp.Body.PromptBundles) == 0 {
		return false, promptSpec.ErrBundleNotFound
	}
	return resp.Body.PromptBundles[0].IsEnabled, nil
}

func getToolBundleEnabled(
	ctx context.Context,
	store *toolStore.ToolStore,
	bundleID bundleitemutils.BundleID,
) (bool, error) {
	resp, err := store.ListToolBundles(ctx, &toolSpec.ListToolBundlesRequest{
		BundleIDs:       []bundleitemutils.BundleID{bundleID},
		IncludeDisabled: true,
		PageSize:        1,
	})
	if err != nil {
		return false, err
	}
	if resp == nil || resp.Body == nil || len(resp.Body.ToolBundles) == 0 {
		return false, toolSpec.ErrBundleNotFound
	}
	return resp.Body.ToolBundles[0].IsEnabled, nil
}

func getSkillBundleEnabled(
	ctx context.Context,
	store *skillStore.SkillStore,
	bundleID bundleitemutils.BundleID,
) (bool, error) {
	resp, err := store.ListSkillBundles(ctx, &skillSpec.ListSkillBundlesRequest{
		BundleIDs:       []bundleitemutils.BundleID{bundleID},
		IncludeDisabled: true,
		PageSize:        1,
	})
	if err != nil {
		return false, err
	}
	if resp == nil || resp.Body == nil || len(resp.Body.SkillBundles) == 0 {
		return false, skillSpec.ErrSkillBundleNotFound
	}
	return resp.Body.SkillBundles[0].IsEnabled, nil
}
