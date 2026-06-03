package sdkclient

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/flexigpt/flexigpt-app/internal/mcp/apps"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

func TestEnvMapAndDisplayHelpers(t *testing.T) {
	t.Run("envMapToList sorts keys and skips blanks", func(t *testing.T) {
		got := envMapToList(map[string]string{
			"beta":  "2",
			"":      "skip-me",
			"alpha": "1",
		})

		want := []string{"alpha=1", "beta=2"}
		if !slices.Equal(got, want) {
			t.Fatalf("envMapToList = %#v, want %#v", got, want)
		}
	})

	t.Run("displayName helpers choose first non-empty value", func(t *testing.T) {
		if got := displayNameFirstNonEmpty("  ", "", "fallback"); got != "fallback" {
			t.Fatalf("displayNameFirstNonEmpty = %q, want %q", got, "fallback")
		}

		tool := &mcpSDK.Tool{
			Name:  "tool-name",
			Title: "tool-title",
		}
		if got := displayNameForTool(tool); got != "tool-title" {
			t.Fatalf("displayNameForTool(title) = %q, want %q", got, "tool-title")
		}

		tool.Title = ""
		tool.Annotations = &mcpSDK.ToolAnnotations{Title: "annotation-title"}
		if got := displayNameForTool(tool); got != "annotation-title" {
			t.Fatalf("displayNameForTool(annotation title) = %q, want %q", got, "annotation-title")
		}

		tool.Annotations.Title = ""
		if got := displayNameForTool(tool); got != "tool-name" {
			t.Fatalf("displayNameForTool(name fallback) = %q, want %q", got, "tool-name")
		}
	})
}

func TestSchemaMapAndDigestHelpers(t *testing.T) {
	type sample struct {
		A int    `json:"a"`
		B string `json:"b"`
	}

	t.Run("schemaToMap handles structs and nil fallback", func(t *testing.T) {
		got := schemaToMap(sample{A: 1, B: "x"})
		if got["a"] != float64(1) {
			t.Fatalf("schemaToMap(a) = %#v, want 1", got["a"])
		}
		if got["b"] != "x" {
			t.Fatalf("schemaToMap(b) = %#v, want %q", got["b"], "x")
		}

		fallback := schemaToMap(nil)
		wantFallback := getEmptySchema()
		if !reflect.DeepEqual(fallback, wantFallback) {
			t.Fatalf("schemaToMap(nil) = %#v, want %#v", fallback, wantFallback)
		}
	})

	t.Run("optionalSchemaToMap returns nil for nil input", func(t *testing.T) {
		if got := optionalSchemaToMap(nil); got != nil {
			t.Fatalf("optionalSchemaToMap(nil) = %#v, want nil", got)
		}
	})

	t.Run("anyToMap clones maps", func(t *testing.T) {
		orig := map[string]any{
			"a": 1,
			"b": "two",
		}
		got := anyToMap(orig)
		if got == nil {
			t.Fatalf("anyToMap returned nil")
		}
		orig["a"] = 99
		if got["a"] != 1 {
			t.Fatalf("clone was mutated: got[%q]=%#v", "a", got["a"])
		}
	})

	t.Run("digestAny is stable for equivalent maps", func(t *testing.T) {
		a := map[string]any{"b": 2, "a": 1}
		b := map[string]any{"a": 1, "b": 2}
		if gotA, gotB := digestAny(a), digestAny(b); gotA == "" || gotA != gotB {
			t.Fatalf("digestAny(a) = %q, digestAny(b) = %q", gotA, gotB)
		}
	})
}

func TestConversionAndInferenceHelpers(t *testing.T) {
	t.Run("stringSliceFromAny filters blanks and preserves strings", func(t *testing.T) {
		got := stringSliceFromAny([]any{"alpha", "", 123, "beta"})
		want := []string{"alpha", "beta"}
		if !slices.Equal(got, want) {
			t.Fatalf("stringSliceFromAny = %#v, want %#v", got, want)
		}

		orig := []string{"x", "y"}
		got2 := stringSliceFromAny(orig)
		if !slices.Equal(got2, orig) {
			t.Fatalf("stringSliceFromAny([]string) = %#v, want %#v", got2, orig)
		}
		orig[0] = "mutated"
		if got2[0] != "x" {
			t.Fatalf("stringSliceFromAny did not clone slice: %#v", got2)
		}
	})

	t.Run("appInfoFromMeta and taskSupportFromMeta", func(t *testing.T) {
		info := appInfoFromMeta(mcpSDK.Meta{
			"ui": map[string]any{
				"resourceUri": "ui://demo",
				"visibility":  []any{apps.VisibilityModel, apps.VisibilityApp, ""},
			},
		})
		if info == nil {
			t.Fatalf("appInfoFromMeta returned nil")
		}
		if info.ResourceURI != "ui://demo" {
			t.Fatalf("ResourceURI = %q, want %q", info.ResourceURI, "ui://demo")
		}
		if !slices.Equal(info.Visibility, []string{apps.VisibilityModel, apps.VisibilityApp}) {
			t.Fatalf("Visibility = %#v, want [model app]", info.Visibility)
		}

		info2 := appInfoFromMeta(mcpSDK.Meta{
			"ui": map[string]any{
				"resourceUri": "ui://fallback",
			},
		})
		if !slices.Equal(info2.Visibility, []string{apps.VisibilityModel, apps.VisibilityApp}) {
			t.Fatalf("default Visibility = %#v, want [model app]", info2.Visibility)
		}

		if got := taskSupportFromMeta(mcpSDK.Meta{
			"execution": map[string]any{
				"taskSupport": string(spec.MCPTaskSupportRequired),
			},
		}); got != spec.MCPTaskSupportRequired {
			t.Fatalf("taskSupportFromMeta = %q, want %q", got, spec.MCPTaskSupportRequired)
		}

		if got := taskSupportFromMeta(nil); got != spec.MCPTaskSupportForbidden {
			t.Fatalf("taskSupportFromMeta(nil) = %q, want forbidden", got)
		}
	})

	t.Run("toolAnnotationsToSpec and inferRisk", func(t *testing.T) {
		destructive := true
		openWorld := true

		ann := &mcpSDK.ToolAnnotations{
			DestructiveHint: &destructive,
			IdempotentHint:  true,
			OpenWorldHint:   &openWorld,
			ReadOnlyHint:    true,
			Title:           "annotated",
		}
		gotAnn := toolAnnotationsToSpec(ann)
		if gotAnn == nil {
			t.Fatalf("toolAnnotationsToSpec returned nil")
		}
		if gotAnn.Title != "annotated" || !gotAnn.IdempotentHint || !gotAnn.ReadOnlyHint {
			t.Fatalf("toolAnnotationsToSpec = %#v", gotAnn)
		}

		if got := inferRisk(nil, spec.MCPTrustLevelUntrusted); got != spec.MCPToolRiskUnknown {
			t.Fatalf("inferRisk(nil) = %q, want unknown", got)
		}
		if got := inferRisk(
			&mcpSDK.ToolAnnotations{ReadOnlyHint: true},
			spec.MCPTrustLevelTrusted,
		); got != spec.MCPToolRiskRead {
			t.Fatalf("inferRisk(read-only, trusted) = %q, want read", got)
		}
		if got := inferRisk(
			&mcpSDK.ToolAnnotations{ReadOnlyHint: true},
			spec.MCPTrustLevelUntrusted,
		); got != spec.MCPToolRiskUnknown {
			t.Fatalf("inferRisk(read-only, untrusted) = %q, want unknown", got)
		}
		if got := inferRisk(
			&mcpSDK.ToolAnnotations{OpenWorldHint: &openWorld},
			spec.MCPTrustLevelUntrusted,
		); got != spec.MCPToolRiskOpenWorld {
			t.Fatalf("inferRisk(open-world) = %q, want openWorld", got)
		}
		if got := inferRisk(
			&mcpSDK.ToolAnnotations{DestructiveHint: &destructive},
			spec.MCPTrustLevelTrusted,
		); got != spec.MCPToolRiskDestructive {
			t.Fatalf("inferRisk(destructive) = %q, want destructive", got)
		}
		if got := inferRisk(
			&mcpSDK.ToolAnnotations{DestructiveHint: new(false)},
			spec.MCPTrustLevelTrusted,
		); got != spec.MCPToolRiskWrite {
			t.Fatalf("inferRisk(non-destructive, trusted) = %q, want write", got)
		}
	})

	t.Run("contentToSpec and resourceContentsToSpec", func(t *testing.T) {
		text := &mcpSDK.TextContent{
			Text: "hello",
			Meta: map[string]any{"kind": "text"},
		}
		gotText := contentToSpec(text)
		if gotText.Type != spec.MCPContentTypeText || gotText.Text != "hello" {
			t.Fatalf("text content = %#v", gotText)
		}
		if gotText.Meta["kind"] != "text" {
			t.Fatalf("text meta = %#v", gotText.Meta)
		}

		img := &mcpSDK.ImageContent{
			Data:     []byte{1, 2, 3},
			MIMEType: "image/png",
		}
		gotImg := contentToSpec(img)
		if gotImg.Type != spec.MCPContentTypeImage || gotImg.MIMEType != "image/png" {
			t.Fatalf("image content = %#v", gotImg)
		}
		img.Data[0] = 9
		if gotImg.Data[0] != 1 {
			t.Fatalf("image data was not cloned: %#v", gotImg.Data)
		}

		audio := &mcpSDK.AudioContent{
			Data:     []byte{4, 5},
			MIMEType: "audio/mpeg",
		}
		gotAudio := contentToSpec(audio)
		if gotAudio.Type != spec.MCPContentTypeAudio || gotAudio.MIMEType != "audio/mpeg" {
			t.Fatalf("audio content = %#v", gotAudio)
		}

		size := int64(7)
		link := &mcpSDK.ResourceLink{
			URI:         "https://example.test/resource",
			Name:        "demo",
			Title:       "Demo",
			Description: "demo link",
			MIMEType:    "text/plain",
			Size:        &size,
			Icons: []mcpSDK.Icon{
				{
					Source:   "https://example.test/icon.png",
					MIMEType: "image/png",
					Sizes:    []string{"16x16"},
					Theme:    "dark",
				},
			},
		}
		gotLink := contentToSpec(link)
		if gotLink.Type != spec.MCPContentTypeResourceLink || gotLink.URI != link.URI || gotLink.Name != "demo" {
			t.Fatalf("resource link = %#v", gotLink)
		}
		if len(gotLink.Icons) != 1 || gotLink.Icons[0].Source != "https://example.test/icon.png" {
			t.Fatalf("icons = %#v", gotLink.Icons)
		}

		rc := &mcpSDK.ResourceContents{
			URI:      "file:///demo",
			MIMEType: "text/plain",
			Text:     "resource-body",
			Blob:     []byte{9, 8, 7},
			Meta:     map[string]any{"x": "y"},
		}
		gotRC := resourceContentsToSpec(rc)
		if gotRC == nil || gotRC.URI != "file:///demo" || gotRC.Text != "resource-body" {
			t.Fatalf("resourceContentsToSpec = %#v", gotRC)
		}
		rc.Blob[0] = 0
		if gotRC.Blob[0] != 9 {
			t.Fatalf("resource blob was not cloned: %#v", gotRC.Blob)
		}

		embedded := &mcpSDK.EmbeddedResource{Resource: rc}
		gotEmbedded := contentToSpec(embedded)
		if gotEmbedded.Type != spec.MCPContentTypeResource || gotEmbedded.Resource == nil {
			t.Fatalf("embedded resource = %#v", gotEmbedded)
		}

		gotSlice := contentSliceToSpec([]mcpSDK.Content{nil, text, img})
		if len(gotSlice) != 2 {
			t.Fatalf("contentSliceToSpec len = %d, want 2", len(gotSlice))
		}
		if gotSlice[0].Type != spec.MCPContentTypeText || gotSlice[1].Type != spec.MCPContentTypeImage {
			t.Fatalf("contentSliceToSpec = %#v", gotSlice)
		}

		icons := iconsToSpec([]mcpSDK.Icon{
			{
				Source:   "https://example.test/icon.svg",
				MIMEType: "image/svg+xml",
				Sizes:    []string{"16x16", "32x32"},
				Theme:    "light",
			},
		})
		if len(icons) != 1 || icons[0].Source != "https://example.test/icon.svg" {
			t.Fatalf("iconsToSpec = %#v", icons)
		}
		if !slices.Equal(icons[0].Sizes, []string{"16x16", "32x32"}) {
			t.Fatalf("iconsToSpec sizes = %#v", icons[0].Sizes)
		}
	})

	t.Run("completionReference", func(t *testing.T) {
		tests := []struct {
			name            string
			req             spec.MCPCompleteArgumentRequestBody
			wantType        string
			wantName        string
			wantURI         string
			wantErrContains string
		}{
			{
				name: "prompt",
				req: spec.MCPCompleteArgumentRequestBody{
					RefType: " prompt ",
					Name:    "greet",
				},
				wantType: "ref/prompt",
				wantName: "greet",
			},

			{
				name: "missing prompt name",
				req: spec.MCPCompleteArgumentRequestBody{
					RefType: "prompt",
				},
				wantErrContains: "completion prompt name required",
			},
			{
				name: "missing resource uri",
				req: spec.MCPCompleteArgumentRequestBody{
					RefType: "resource",
				},
				wantErrContains: "completion resource uri required",
			},
			{
				name: "invalid ref type",
				req: spec.MCPCompleteArgumentRequestBody{
					RefType: "bogus",
				},
				wantErrContains: "invalid completion refType",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := completionReference(tt.req)
				if tt.wantErrContains != "" {
					if err == nil || !strings.Contains(err.Error(), tt.wantErrContains) {
						t.Fatalf("err = %v, want substring %q", err, tt.wantErrContains)
					}
					if got != nil {
						t.Fatalf("got = %#v, want nil", got)
					}
					return
				}

				if err != nil {
					t.Fatalf("completionReference: %v", err)
				}
				if got == nil {
					t.Fatalf("got is nil")
				}
				if got.Type != tt.wantType {
					t.Fatalf("Type = %q, want %q", got.Type, tt.wantType)
				}
				if tt.wantName != "" && got.Name != tt.wantName {
					t.Fatalf("Name = %q, want %q", got.Name, tt.wantName)
				}
				if tt.wantURI != "" && got.URI != tt.wantURI {
					t.Fatalf("URI = %q, want %q", got.URI, tt.wantURI)
				}
			})
		}
	})
}
