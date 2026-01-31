package goregistry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/builtin"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/tool/spec"
	"github.com/flexigpt/flexigpt-app/internal/tool/storehelper"

	"github.com/flexigpt/llmtools-go"
	llmtoolsgoSpec "github.com/flexigpt/llmtools-go/spec"
)

type llmtoolsGoToolMeta struct {
	BundleID     bundleitemutils.BundleID
	AutoExecReco bool
}

// llmtoolsGoBuiltinCatalog is the SINGLE source of truth for how llmtools-go builtins
// are represented in this app.
//
// Key: exact Go FuncID string from llmtools-go (stable in your environment).
// Value: bundle routing + auto-exec recommendation.
//
// When llmtools-go returns a tool that isn't mapped here, injectLLMToolsGoBuiltins fails
// with a clear error telling you to add it.
var llmtoolsGoBuiltinCatalog = map[string]llmtoolsGoToolMeta{
	// FStool.
	"github.com/flexigpt/llmtools-go/fstool/readfile.ReadFile": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsFS),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/fstool/searchfiles.SearchFiles": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsFS),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/fstool/writefile.WriteFile": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsFS),
		AutoExecReco: false,
	},
	"github.com/flexigpt/llmtools-go/fstool/deletefile.DeleteFile": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsFS),
		AutoExecReco: false,
	},
	"github.com/flexigpt/llmtools-go/fstool/listdirectory.ListDirectory": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsFS),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/fstool/statpath.StatPath": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsFS),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/fstool/mimeforpath.MIMEForPath": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsFS),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/fstool/mimeforextension.MIMEForExtension": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsFS),
		AutoExecReco: true,
	},

	// Imagetool.
	"github.com/flexigpt/llmtools-go/imagetool/readimage.ReadImage": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsImage),
		AutoExecReco: true,
	},

	// Shelltool (dangerous by default).
	"github.com/flexigpt/llmtools-go/shelltool/shell.ShellCommand": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsShell),
		AutoExecReco: false,
	},

	// Texttool.
	"github.com/flexigpt/llmtools-go/texttool/readtextrange.ReadTextRange": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsText),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/texttool/findtext.FindText": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsText),
		AutoExecReco: true,
	},
	"github.com/flexigpt/llmtools-go/texttool/inserttextlines.InsertTextLines": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsText),
		AutoExecReco: false,
	},
	"github.com/flexigpt/llmtools-go/texttool/replacetextlines.ReplaceTextLines": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsText),
		AutoExecReco: false,
	},
	"github.com/flexigpt/llmtools-go/texttool/deletetextlines.DeleteTextLines": {
		BundleID:     bundleitemutils.BundleID(builtin.BuiltinBundleIDLLMToolsText),
		AutoExecReco: false,
	},
}

func InjectLLMToolsGo(
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

func metaForLLMToolsGo(t llmtoolsgoSpec.Tool) (llmtoolsGoToolMeta, error) {
	fid := strings.TrimSpace(string(t.GoImpl.FuncID))
	if fid == "" {
		return llmtoolsGoToolMeta{}, errors.New("llmtools-go tool has empty funcID")
	}

	meta, ok := llmtoolsGoBuiltinCatalog[fid]
	if !ok {
		return llmtoolsGoToolMeta{}, fmt.Errorf(
			"llmtools-go tool funcID=%q is not mapped in llmtoolsGoBuiltinCatalog; add it there",
			fid,
		)
	}

	if meta.BundleID == "" {
		return llmtoolsGoToolMeta{}, fmt.Errorf(
			"llmtools-go tool funcID=%q has empty bundleID mapping; fix llmtoolsGoBuiltinCatalog",
			fid,
		)
	}

	return meta, nil
}

func toAppToolFromLLMToolsGo(t llmtoolsgoSpec.Tool, meta llmtoolsGoToolMeta) (spec.Tool, error) {
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
