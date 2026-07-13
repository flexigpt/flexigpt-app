package store

import (
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/assistantpreset/spec"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
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

	out.StartingModelPresetRef = cloneJSONValue(in.StartingModelPresetRef)
	if in.StartingIncludeModelSystemPrompt != nil {
		v := *in.StartingIncludeModelSystemPrompt
		out.StartingIncludeModelSystemPrompt = &v
	}
	out.StartingToolSelections = cloneJSONValue(in.StartingToolSelections)
	out.StartingSkillSelections = cloneJSONValue(in.StartingSkillSelections)
	out.StartingMCPContext = cloneJSONValue(in.StartingMCPContext)

	return out
}

func cloneJSONValue[T any](in T) T {
	raw, err := json.Marshal(in)
	if err != nil {
		return in
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		return in
	}
	return out
}
