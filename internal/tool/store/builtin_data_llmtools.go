package store

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/builtin"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/tool/spec"

	"github.com/flexigpt/llmtools-go"
	llmtoolsgoSpec "github.com/flexigpt/llmtools-go/spec"
)

func injectLLMToolsGoBuiltins(
	ctx context.Context,
	bundles map[bundleitemutils.BundleID]spec.ToolBundle,
	tools map[bundleitemutils.BundleID]map[bundleitemutils.ItemID]spec.Tool,
) error {
	r, err := llmtools.NewBuiltinRegistry()
	if err != nil {
		return fmt.Errorf("llmtools-go builtin registry: %w", err)
	}

	for _, ext := range r.Tools() {
		bundleID, err := bundleIDForLLMToolsGo(ext)
		if err != nil {
			return err
		}
		if _, ok := bundles[bundleID]; !ok {
			return fmt.Errorf(
				"llmtools-go tool %q mapped to bundle %q but bundle is missing in tools.bundles.json",
				ext.GoImpl.FuncID,
				bundleID,
			)
		}
		appTool, err := toAppToolFromLLMToolsGo(ext)
		if err != nil {
			return fmt.Errorf("convert llmtools-go tool %q: %w", ext.GoImpl.FuncID, err)
		}

		if tools[bundleID] == nil {
			tools[bundleID] = make(map[bundleitemutils.ItemID]spec.Tool)
		}

		// Prevent duplicates by slug+version (important for GetBuiltInTool which searches by slug+version).
		for id, existing := range tools[bundleID] {
			if existing.Slug == appTool.Slug && existing.Version == appTool.Version && id != appTool.ID {
				delete(tools[bundleID], id)
			}
		}

		if _, exists := tools[bundleID][appTool.ID]; exists {
			slog.Info(
				"overriding embedded builtin tool with llmtools-go definition",
				"bundleID", bundleID,
				"toolID", appTool.ID,
				"slug", appTool.Slug,
				"version", appTool.Version,
				"func", appTool.GoImpl.Func,
			)
		}
		tools[bundleID][appTool.ID] = appTool
	}

	return nil
}

func bundleIDForLLMToolsGo(t llmtoolsgoSpec.Tool) (bundleitemutils.BundleID, error) {
	fid := string(t.GoImpl.FuncID)

	// Infer from funcID.
	switch {
	case strings.Contains(fid, "/fstool/"):
		return bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsFS), nil
	case strings.Contains(fid, "/imagetool/"):
		return bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsImage), nil
	case strings.Contains(fid, "/shelltool/"):
		return bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsShell), nil
	case strings.Contains(fid, "/texttool/"):
		return bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsText), nil
	default:
		return "", fmt.Errorf("no bundle mapping for llmtools-go tool funcID=%q tags=%v", fid, t.Tags)
	}
}

func toAppToolFromLLMToolsGo(t llmtoolsgoSpec.Tool) (spec.Tool, error) {
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

		ArgSchema: json.RawMessage(t.ArgSchema),

		LLMToolType: spec.ToolStoreChoiceTypeFunction,

		Type:   spec.ToolTypeGo,
		GoImpl: &spec.GoToolImpl{Func: string(t.GoImpl.FuncID)},

		IsEnabled: true,
		IsBuiltIn: true,

		CreatedAt:  created,
		ModifiedAt: mod,
	}

	if err := validateTool(&out); err != nil {
		return spec.Tool{}, err
	}
	return out, nil
}
