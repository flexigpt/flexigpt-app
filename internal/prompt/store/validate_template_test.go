package store

import (
	"strings"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/prompt/spec"
)

func TestSlugVersionValidation(t *testing.T) {
	cases := []struct {
		slug  bundleitemutils.ItemSlug
		ver   bundleitemutils.ItemVersion
		valid bool
	}{
		{"abc", "v1", true},
		{"abc-def", "v1", true},
		{"", "v1", false},
		{"abc", "", false},
		{"bad.slug", "v1", false},
		{"abc", "v 1", false},
	}

	for _, c := range cases {
		errSlug := bundleitemutils.ValidateItemSlug(c.slug)
		errVer := bundleitemutils.ValidateItemVersion(c.ver)
		if c.valid && (errSlug != nil || errVer != nil) {
			t.Fatalf("expected valid slug/version (%s/%s) got errors %v %v",
				c.slug, c.ver, errSlug, errVer)
		}
		if !c.valid && errSlug == nil && errVer == nil {
			t.Fatalf("expected invalid for (%s/%s)", c.slug, c.ver)
		}
	}
}

func TestValidateTemplate_KindAndResolvedRules(t *testing.T) {
	strptr := func(s string) *string { return &s }

	base := func() spec.PromptTemplate {
		return spec.PromptTemplate{
			SchemaVersion: spec.SchemaVersion,
			ID:            bundleitemutils.ItemID("item-1"),
			Slug:          bundleitemutils.ItemSlug("tpl"),
			Version:       bundleitemutils.ItemVersion("v1"),
			DisplayName:   "Template",
			IsEnabled:     true,
			CreatedAt:     time.Now().UTC(),
			ModifiedAt:    time.Now().UTC(),
			IsBuiltIn:     false,
		}
	}

	tests := []struct {
		name       string
		tpl        spec.PromptTemplate
		wantErrSub string
	}{
		{
			name: "instructions_only_allows_system_and_developer",
			tpl: func() spec.PromptTemplate {
				tpl := base()
				tpl.Kind = spec.PromptTemplateKindInstructionsOnly
				tpl.IsResolved = true
				tpl.Blocks = []spec.MessageBlock{
					{ID: "b1", Role: spec.System, Content: "sys"},
					{ID: "b2", Role: spec.Developer, Content: "dev"},
				}
				return tpl
			}(),
		},
		{
			name: "instructions_only_rejects_user_role",
			tpl: func() spec.PromptTemplate {
				tpl := base()
				tpl.Kind = spec.PromptTemplateKindInstructionsOnly
				tpl.IsResolved = true
				tpl.Blocks = []spec.MessageBlock{
					{ID: "b1", Role: spec.User, Content: "user"},
				}
				return tpl
			}(),
			wantErrSub: "not allowed for kind",
		},
		{
			name: "generic_allows_mixed_roles",
			tpl: func() spec.PromptTemplate {
				tpl := base()
				tpl.Kind = spec.PromptTemplateKindGeneric
				tpl.IsResolved = true
				tpl.Blocks = []spec.MessageBlock{
					{ID: "b1", Role: spec.System, Content: "sys"},
					{ID: "b2", Role: spec.User, Content: "usr"},
				}
				return tpl
			}(),
		},
		{
			name: "missing_kind_rejected",
			tpl: func() spec.PromptTemplate {
				tpl := base()
				tpl.Kind = ""
				tpl.IsResolved = true
				tpl.Blocks = []spec.MessageBlock{
					{ID: "b1", Role: spec.System, Content: "sys"},
				}
				return tpl
			}(),
			wantErrSub: "invalid kind",
		},
		{
			name: "resolved_true_when_user_var_has_default",
			tpl: func() spec.PromptTemplate {
				tpl := base()
				tpl.Kind = spec.PromptTemplateKindGeneric
				tpl.IsResolved = true
				tpl.Blocks = []spec.MessageBlock{
					{ID: "b1", Role: spec.User, Content: "Hello {{name}}"},
				}
				tpl.Variables = []spec.PromptVariable{
					{
						Name:     "name",
						Type:     spec.VarString,
						Source:   spec.SourceUser,
						Required: true,
						Default:  strptr("Alice"),
					},
				}
				return tpl
			}(),
		},
		{
			name: "resolved_mismatch_rejected_for_unresolved_user_var",
			tpl: func() spec.PromptTemplate {
				tpl := base()
				tpl.Kind = spec.PromptTemplateKindGeneric
				tpl.IsResolved = true
				tpl.Blocks = []spec.MessageBlock{
					{ID: "b1", Role: spec.User, Content: "Hello {{name}}"},
				}
				tpl.Variables = []spec.PromptVariable{
					{
						Name:     "name",
						Type:     spec.VarString,
						Source:   spec.SourceUser,
						Required: true,
					},
				}
				return tpl
			}(),
			wantErrSub: "isResolved mismatched",
		},
		{
			name: "resolved_false_allowed_for_unresolved_user_var",
			tpl: func() spec.PromptTemplate {
				tpl := base()
				tpl.Kind = spec.PromptTemplateKindGeneric
				tpl.IsResolved = false
				tpl.Blocks = []spec.MessageBlock{
					{ID: "b1", Role: spec.User, Content: "Hello {{name}}"},
				}
				tpl.Variables = []spec.PromptVariable{
					{
						Name:     "name",
						Type:     spec.VarString,
						Source:   spec.SourceUser,
						Required: true,
					},
				}
				return tpl
			}(),
		},
		{
			name: "enum_default_must_be_in_enum_values",
			tpl: func() spec.PromptTemplate {
				tpl := base()
				tpl.Kind = spec.PromptTemplateKindGeneric
				tpl.IsResolved = true
				tpl.Blocks = []spec.MessageBlock{
					{ID: "b1", Role: spec.User, Content: "Mode {{mode}}"},
				}
				tpl.Variables = []spec.PromptVariable{
					{
						Name:       "mode",
						Type:       spec.VarEnum,
						Source:     spec.SourceUser,
						Required:   true,
						EnumValues: []string{"fast", "safe"},
						Default:    strptr("bad"),
					},
				}
				return tpl
			}(),
			wantErrSub: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTemplate(&tt.tpl)
			if tt.wantErrSub == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErrSub)
			}
			if !strings.Contains(err.Error(), tt.wantErrSub) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErrSub)
			}
		})
	}
}

func TestGetAndPatchPromptTemplate_ValidateStoredTemplate(t *testing.T) {
	tests := []struct {
		name        string
		corrupt     func(raw map[string]any)
		getErrSub   string
		patchErrSub string
	}{
		{
			name: "corrupt_kind_rejected",
			corrupt: func(raw map[string]any) {
				raw["kind"] = ""
			},
			getErrSub:   "invalid kind",
			patchErrSub: "invalid kind",
		},
		{
			name: "corrupt_isResolved_rejected",
			corrupt: func(raw map[string]any) {
				raw["isResolved"] = false
			},
			getErrSub:   "isResolved mismatched",
			patchErrSub: "isResolved mismatched",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, clean := newTestStore(t)
			defer clean()

			const (
				bid   = bundleitemutils.BundleID("b1")
				bslug = bundleitemutils.BundleSlug("slug1")
				tslug = bundleitemutils.ItemSlug("tpl")
				tver  = bundleitemutils.ItemVersion("v1")
			)

			mustPutBundle(t, s, bid, bslug, "Bundle", true)
			mustPutTemplate(t, s, bid, tslug, tver, "Display", true)

			dirInfo, err := bundleitemutils.BuildBundleDir(bid, bslug)
			if err != nil {
				t.Fatalf("BuildBundleDir: %v", err)
			}
			fileInfo, err := bundleitemutils.BuildItemFileInfo(tslug, tver)
			if err != nil {
				t.Fatalf("BuildItemFileInfo: %v", err)
			}
			key := bundleitemutils.GetBundlePartitionFileKey(fileInfo.FileName, dirInfo.DirName)

			raw, err := s.templateStore.GetFileData(key, false)
			if err != nil {
				t.Fatalf("GetFileData: %v", err)
			}
			tt.corrupt(raw)
			if err := s.templateStore.SetFileData(key, raw); err != nil {
				t.Fatalf("SetFileData: %v", err)
			}

			_, err = s.GetPromptTemplate(t.Context(), &spec.GetPromptTemplateRequest{
				BundleID:     bid,
				TemplateSlug: tslug,
				Version:      tver,
			})
			if err == nil || !strings.Contains(err.Error(), tt.getErrSub) {
				t.Fatalf("GetPromptTemplate error = %v, want substring %q", err, tt.getErrSub)
			}

			_, err = s.PatchPromptTemplate(t.Context(), &spec.PatchPromptTemplateRequest{
				BundleID:     bid,
				TemplateSlug: tslug,
				Version:      tver,
				Body: &spec.PatchPromptTemplateRequestBody{
					IsEnabled: false,
				},
			})
			if err == nil || !strings.Contains(err.Error(), tt.patchErrSub) {
				t.Fatalf("PatchPromptTemplate error = %v, want substring %q", err, tt.patchErrSub)
			}
		})
	}
}
