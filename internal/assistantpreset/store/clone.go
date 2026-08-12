package store

import (
	"github.com/flexigpt/flexigpt-app/internal/assistantpreset/spec"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

func cloneAllAssistantPresets(
	src map[bundleitemutils.BundleID]map[bundleitemutils.ItemID]spec.AssistantPreset,
) map[bundleitemutils.BundleID]map[bundleitemutils.ItemID]spec.AssistantPreset {
	dst := make(
		map[bundleitemutils.BundleID]map[bundleitemutils.ItemID]spec.AssistantPreset,
		len(src),
	)
	for bid, inner := range src {
		sub := make(map[bundleitemutils.ItemID]spec.AssistantPreset, len(inner))
		for pid, preset := range inner {
			sub[pid] = cloneAssistantPreset(preset)
		}
		dst[bid] = sub
	}
	return dst
}

func cloneAssistantPreset(in spec.AssistantPreset) spec.AssistantPreset {
	out := in

	out.StartingModelPresetRef = jsonutil.CloneJSONInValue(in.StartingModelPresetRef)
	if in.StartingIncludeModelSystemPrompt != nil {
		v := *in.StartingIncludeModelSystemPrompt
		out.StartingIncludeModelSystemPrompt = &v
	}
	out.StartingToolSelections = jsonutil.CloneJSONInValue(in.StartingToolSelections)
	out.StartingSkillSelections = jsonutil.CloneJSONInValue(in.StartingSkillSelections)
	out.StartingMCPContext = jsonutil.CloneJSONInValue(in.StartingMCPContext)

	return out
}
