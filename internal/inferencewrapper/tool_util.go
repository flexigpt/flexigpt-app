package inferencewrapper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
	toolStore "github.com/flexigpt/flexigpt-app/internal/tool/store"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

func buildToolChoices(
	ctx context.Context,
	ts *toolStore.ToolStore,
	toolStoreChoices []toolSpec.ToolStoreChoice,
) ([]inferenceSpec.ToolChoice, error) {
	out := make([]inferenceSpec.ToolChoice, 0)
	if len(toolStoreChoices) == 0 {
		return nil, nil
	}

	if ts == nil {
		return nil, errors.New("tool store not configured for provider set")
	}

	for _, sc := range toolStoreChoices {
		if sc.ChoiceID == "" || sc.BundleID == "" || sc.ToolSlug == "" || strings.TrimSpace(sc.ToolVersion) == "" {
			return nil, fmt.Errorf(
				"invalid tool store choice: choiceID/bundleID/toolSlug/toolVersion required: %+v",
				sc,
			)
		}
		tc, err := hydrateToolChoice(ctx, ts, sc)
		if err != nil {
			return nil, err
		}
		out = append(out, *tc)
	}

	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// hydrateToolChoice loads the Tool definition from tool-store and converts it
// into an inference-go ToolChoice. This is only called when we don't already
// have a ToolChoice persisted in the conversation for the same tool.
func hydrateToolChoice(
	ctx context.Context,
	ts *toolStore.ToolStore,
	sc toolSpec.ToolStoreChoice,
) (toolChoice *inferenceSpec.ToolChoice, err error) {
	if sc.ChoiceID == "" {
		return nil, errors.New("invalid choiceID for tool store choice")
	}
	if ts == nil {
		return nil, errors.New("tool store not configured for provider set")
	}
	req := &toolSpec.GetToolRequest{
		BundleID: sc.BundleID,
		ToolSlug: sc.ToolSlug,
		Version:  bundleitemutils.ItemVersion(sc.ToolVersion),
	}
	resp, err := ts.GetTool(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to load tool %s/%s@%s: %w",
			sc.BundleID,
			sc.ToolSlug,
			sc.ToolVersion,
			err,
		)
	}
	if resp == nil || resp.Body == nil {
		return nil, fmt.Errorf(
			"tool %s/%s@%s not found",
			sc.BundleID,
			sc.ToolSlug,
			sc.ToolVersion,
		)
	}
	tool := resp.Body
	if !tool.IsEnabled {
		return nil, fmt.Errorf(
			"tool %s/%s@%s is disabled",
			sc.BundleID,
			sc.ToolSlug,
			sc.ToolVersion,
		)
	}
	if !tool.LLMCallable {
		return nil, fmt.Errorf(
			"tool %s/%s@%s is not LLM-callable",
			sc.BundleID, sc.ToolSlug, sc.ToolVersion,
		)
	}
	name := string(sc.ToolSlug)
	desc := tool.Description
	if desc == "" {
		desc = sc.Description
	}

	tc := &inferenceSpec.ToolChoice{
		Type:        inferenceSpec.ToolType(sc.ToolType),
		ID:          sc.ChoiceID,
		Name:        name,
		Description: desc,
	}

	switch tool.Type {
	case toolSpec.ToolTypeGo, toolSpec.ToolTypeHTTP:
		argSchema, err := decodeToolArgSchema(string(tool.ArgSchema))
		if err != nil {
			return nil, fmt.Errorf(
				"invalid argSchema for %s/%s@%s: %w",
				sc.BundleID,
				sc.ToolSlug,
				sc.ToolVersion,
				err,
			)
		}
		tc.Arguments = argSchema

	case toolSpec.ToolTypeSDK:
		// SDK-backed server tools. Semantics come from sc.ToolType
		// (e.g., "webSearch"), while implementation is described by
		// tool.SDK and user configuration by tool.UserArgSchema plus
		// sc.Config.
		switch sc.ToolType {
		case toolSpec.ToolStoreChoiceTypeWebSearch:
			// Decode per-choice config (if any) and map to the
			// inference-go WebSearchToolChoiceItem.
			var cfg inferenceSpec.WebSearchToolChoiceItem
			rawCfg := strings.TrimSpace(sc.UserArgSchemaInstance)
			if rawCfg != "" {
				if err := json.Unmarshal([]byte(rawCfg), &cfg); err != nil {
					return nil, fmt.Errorf(
						"invalid config for webSearch tool %s/%s@%s: %w",
						sc.BundleID, sc.ToolSlug, sc.ToolVersion, err,
					)
				}
			}
			tc.Type = inferenceSpec.ToolTypeWebSearch
			tc.WebSearchArguments = &cfg

		default:
			// Future SDK-backed tool kinds (function/custom) could be added here.
			// For now, we treat anything other than webSearch as unsupported.
			return nil, fmt.Errorf(
				"unsupported ToolType %q for sdk tool %s/%s@%s",
				sc.ToolType, sc.BundleID, sc.ToolSlug, sc.ToolVersion,
			)
		}

	default:
		return nil, fmt.Errorf("unsupported tool impl type %q", tool.Type)
	}

	return tc, nil
}

func decodeToolArgSchema(raw toolSpec.JSONRawString) (map[string]any, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return getEmptySchema(), nil
	}
	var schema map[string]any
	if err := json.Unmarshal([]byte(s), &schema); err != nil {
		return nil, err
	}
	if len(schema) == 0 {
		schema = getEmptySchema()
	}
	return schema, nil
}
