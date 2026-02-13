package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/flexigpt/llmtools-go"
	llmtoolsgoSpec "github.com/flexigpt/llmtools-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	"github.com/flexigpt/flexigpt-app/internal/tool/spec"
	"github.com/flexigpt/flexigpt-app/internal/tool/storehelper"
)

func injectLLMToolsGo(
	ctx context.Context,
	bundles map[bundleitemutils.BundleID]spec.ToolBundle,
	tools map[bundleitemutils.BundleID]map[bundleitemutils.ItemID]spec.Tool,
) error {
	r, err := llmtools.NewBuiltinRegistry()
	if err != nil {
		return fmt.Errorf("llmtools-go builtin registry: %w", err)
	}

	for _, ext := range r.Tools() {
		meta, err := metaForLLMToolsGo(ext)
		if err != nil {
			return err
		}

		if _, ok := bundles[meta.BundleID]; !ok {
			return fmt.Errorf(
				"llmtools-go tool %q mapped to bundle %q but bundle is missing in tools.bundles.json",
				ext.GoImpl.FuncID,
				meta.BundleID,
			)
		}

		appTool, err := toAppToolFromLLMToolsGo(ext, meta)
		if err != nil {
			return fmt.Errorf("convert llmtools-go tool %q: %w", ext.GoImpl.FuncID, err)
		}

		if tools[meta.BundleID] == nil {
			tools[meta.BundleID] = make(map[bundleitemutils.ItemID]spec.Tool)
		}

		// Prevent duplicates by slug+version (important for GetBuiltInTool which searches by slug+version).
		for id, existing := range tools[meta.BundleID] {
			if existing.Slug == appTool.Slug && existing.Version == appTool.Version && id != appTool.ID {
				delete(tools[meta.BundleID], id)
			}
		}

		if _, exists := tools[meta.BundleID][appTool.ID]; exists {
			slog.Info(
				"overriding embedded builtin tool with llmtools-go definition",
				"bundleID", meta.BundleID,
				"toolID", appTool.ID,
				"slug", appTool.Slug,
				"version", appTool.Version,
				"func", appTool.GoImpl.Func,
			)
		}

		tools[meta.BundleID][appTool.ID] = appTool
	}

	return nil
}

func metaForLLMToolsGo(t llmtoolsgoSpec.Tool) (llmtoolsutil.LLMToolMeta, error) {
	fid := strings.TrimSpace(string(t.GoImpl.FuncID))
	if fid == "" {
		return llmtoolsutil.LLMToolMeta{}, errors.New("llmtools-go tool has empty funcID")
	}

	meta, ok := llmtoolsutil.LLMToolsGoBuiltinCatalog[fid]
	if !ok {
		return llmtoolsutil.LLMToolMeta{}, fmt.Errorf(
			"llmtools-go tool funcID=%q is not mapped in LLMToolsGoBuiltinCatalog; add it there",
			fid,
		)
	}

	if meta.BundleID == "" {
		return llmtoolsutil.LLMToolMeta{}, fmt.Errorf(
			"llmtools-go tool funcID=%q has empty bundleID mapping; fix LLMToolsGoBuiltinCatalog",
			fid,
		)
	}

	return meta, nil
}

func toAppToolFromLLMToolsGo(t llmtoolsgoSpec.Tool, meta llmtoolsutil.LLMToolMeta) (spec.Tool, error) {
	now := time.Now().UTC()
	created := t.CreatedAt
	if created.IsZero() {
		created = now
	}
	mod := t.ModifiedAt
	if mod.IsZero() {
		mod = created
	}

	out := spec.Tool{
		SchemaVersion: spec.SchemaVersion,

		ID:      bundleitemutils.ItemID(t.ID),
		Slug:    bundleitemutils.ItemSlug(t.Slug),
		Version: bundleitemutils.ItemVersion(t.Version),

		DisplayName: t.DisplayName,
		Description: t.Description,
		Tags:        slices.Clone(t.Tags),

		UserCallable: true,
		LLMCallable:  true,
		AutoExecReco: meta.AutoExecReco,

		ArgSchema: json.RawMessage(t.ArgSchema),

		LLMToolType: spec.ToolStoreChoiceTypeFunction,

		Type:   spec.ToolTypeGo,
		GoImpl: &spec.GoToolImpl{Func: string(t.GoImpl.FuncID)},

		IsEnabled: true,
		IsBuiltIn: true,

		CreatedAt:  created,
		ModifiedAt: mod,
	}

	if err := storehelper.ValidateTool(&out); err != nil {
		return spec.Tool{}, err
	}
	return out, nil
}
