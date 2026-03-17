package store

import (
	"encoding/base64"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

const renamed = "RENAMED"

func TestModelPresetStore_New_CreatesFiles_AndDefaultProviderFallback(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	st := newStoreAtDir(t, dir)
	ctx := t.Context()

	// User JSON file should exist.
	mustFileExists(t, filepath.Join(dir, spec.ModelPresetsFile))
	// Built-in overlay db should exist.
	mustFileExists(t, filepath.Join(dir, spec.ModelPresetsBuiltInOverlayDBFileName))

	// Default provider should be non-empty and should match builtin fallback logic.
	got, err := st.GetDefaultProvider(ctx, &spec.GetDefaultProviderRequest{})
	if err != nil {
		t.Fatalf("GetDefaultProvider: %v", err)
	}
	if got.Body.DefaultProvider == "" {
		t.Fatalf("expected non-empty default provider")
	}

	// Also ensure at least one provider exists (built-ins).
	resp, err := st.ListProviderPresets(ctx, &spec.ListProviderPresetsRequest{IncludeDisabled: true})
	if err != nil {
		t.Fatalf("ListProviderPresets: %v", err)
	}
	if len(resp.Body.Providers) == 0 {
		t.Fatalf("expected at least one provider (built-ins), got 0")
	}
}

func TestModelPresetStore_DefaultProvider_CRUD_AndPersistence(t *testing.T) {
	dir := t.TempDir()
	st := newStoreAtDir(t, dir)
	ctx := t.Context()

	// Create a user provider so PatchDefaultProvider can target user data too.
	userProvider := inferenceSpec.ProviderName("user-prov-default")
	postUserProvider(t, st, userProvider, true)

	builtinName, _ := anyBuiltInProviderFromStore(t, st)

	tests := []struct {
		name        string
		req         *spec.PatchDefaultProviderRequest
		wantErrIs   error
		wantErrText string
	}{
		{
			name:      "nil_request",
			req:       nil,
			wantErrIs: spec.ErrProviderNotFound,
		},
		{
			name: "nil_body",
			req: &spec.PatchDefaultProviderRequest{
				Body: nil,
			},
			wantErrIs: spec.ErrProviderNotFound,
		},
		{
			name: "empty_provider",
			req: &spec.PatchDefaultProviderRequest{
				Body: &spec.PatchDefaultProviderRequestBody{DefaultProvider: ""},
			},
			wantErrIs: spec.ErrProviderNotFound,
		},
		{
			name: "unknown_provider",
			req: &spec.PatchDefaultProviderRequest{
				Body: &spec.PatchDefaultProviderRequestBody{DefaultProvider: "ghost"},
			},
			wantErrIs: spec.ErrProviderNotFound,
		},
		{
			name: "set_to_builtin_provider",
			req: &spec.PatchDefaultProviderRequest{
				Body: &spec.PatchDefaultProviderRequestBody{DefaultProvider: builtinName},
			},
		},
		{
			name: "set_to_user_provider",
			req: &spec.PatchDefaultProviderRequest{
				Body: &spec.PatchDefaultProviderRequestBody{DefaultProvider: userProvider},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := st.PatchDefaultProvider(ctx, tt.req)
			if tt.wantErrIs != nil {
				wantErrIs(t, err, tt.wantErrIs)
				return
			}
			if tt.wantErrText != "" {
				wantErrContains(t, err, tt.wantErrText)
				return
			}
			if err != nil {
				t.Fatalf("unexpected: %v", err)
			}
		})
	}

	// Verify persisted to disk by reopening store in same dir.
	closeAndSleepOnWindows(t, st)
	st2 := newStoreAtDir(t, dir)

	got, err := st2.GetDefaultProvider(ctx, &spec.GetDefaultProviderRequest{})
	if err != nil {
		t.Fatalf("GetDefaultProvider(after reopen): %v", err)
	}
	if got.Body.DefaultProvider != userProvider {
		t.Fatalf("default provider not persisted: got=%q want=%q", got.Body.DefaultProvider, userProvider)
	}
}

func TestModelPresetStore_PostProviderPreset(t *testing.T) {
	st := newStore(t)
	ctx := t.Context()

	builtinName, _ := anyBuiltInProviderFromStore(t, st)

	okBody := &spec.PostProviderPresetRequestBody{
		DisplayName:              "OK",
		SDKType:                  inferenceSpec.ProviderSDKTypeOpenAIChatCompletions,
		IsEnabled:                true,
		Origin:                   "https://example.test",
		ChatCompletionPathPrefix: spec.DefaultOpenAIChatCompletionsPrefix,
		APIKeyHeaderKey:          spec.DefaultAuthorizationHeaderKey,
		DefaultHeaders:           map[string]string{"content-type": "application/json"},
	}

	tests := []struct {
		name        string
		req         *spec.PostProviderPresetRequest
		wantErrIs   error
		wantErrText string
		verify      func(t *testing.T)
	}{
		{
			name:      "nil_request",
			req:       nil,
			wantErrIs: spec.ErrInvalidDir,
		},
		{
			name: "nil_body",
			req: &spec.PostProviderPresetRequest{
				ProviderName: "user-prov-x",
				Body:         nil,
			},
			wantErrIs: spec.ErrInvalidDir,
		},
		{
			name: "empty_providerName",
			req: &spec.PostProviderPresetRequest{
				ProviderName: "",
				Body:         okBody,
			},
			wantErrIs: spec.ErrInvalidDir,
		},
		{
			name: "built_in_readonly",
			req: &spec.PostProviderPresetRequest{
				ProviderName: builtinName,
				Body:         okBody,
			},
			wantErrIs: spec.ErrBuiltInReadOnly,
		},
		{
			name: "validation_error_empty_displayName",
			req: &spec.PostProviderPresetRequest{
				ProviderName: "user-prov-bad1",
				Body: func() *spec.PostProviderPresetRequestBody {
					b := *okBody
					b.DisplayName = ""
					return &b
				}(),
			},
			wantErrText: "displayName is empty",
		},
		{
			name: "validation_error_empty_origin",
			req: &spec.PostProviderPresetRequest{
				ProviderName: "user-prov-bad2",
				Body: func() *spec.PostProviderPresetRequestBody {
					b := *okBody
					b.Origin = ""
					return &b
				}(),
			},
			wantErrText: "origin is empty",
		},
		{
			name: "validation_error_empty_chatCompletionPathPrefix",
			req: &spec.PostProviderPresetRequest{
				ProviderName: "user-prov-bad3",
				Body: func() *spec.PostProviderPresetRequestBody {
					b := *okBody
					b.ChatCompletionPathPrefix = ""
					return &b
				}(),
			},
			wantErrText: "chatCompletionPathPrefix is empty",
		},
		{
			name: "happy_create",
			req: &spec.PostProviderPresetRequest{
				ProviderName: "user-prov-ok",
				Body:         okBody,
			},
			verify: func(t *testing.T) {
				t.Helper()
				pp := getProviderByName(t, st, ctx, "user-prov-ok", true)
				if pp.IsBuiltIn {
					t.Fatalf("expected user provider (IsBuiltIn=false)")
				}
				if pp.CreatedAt.IsZero() || pp.ModifiedAt.IsZero() {
					t.Fatalf("timestamps not set")
				}
			},
		},
		{
			name: "already_exists",
			req: &spec.PostProviderPresetRequest{
				ProviderName: "user-prov-ok",
				Body:         okBody,
			},
			wantErrIs: spec.ErrProviderPresetAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := st.PostProviderPreset(ctx, tt.req)
			if tt.wantErrIs != nil {
				wantErrIs(t, err, tt.wantErrIs)
				return
			}
			if tt.wantErrText != "" {
				wantErrContains(t, err, tt.wantErrText)
				return
			}
			if err != nil {
				t.Fatalf("unexpected: %v", err)
			}
			if tt.verify != nil {
				tt.verify(t)
			}
		})
	}
}

func TestModelPresetStore_PatchProviderPreset_MetadataPreservesModelsAndDefault(t *testing.T) {
	st := newStore(t)
	ctx := t.Context()

	prov := inferenceSpec.ProviderName("user-prov-preserve")
	postUserProvider(t, st, prov, true)
	postUserModelPreset(t, ctx, st, prov, "m1", true)
	_, err := st.PatchProviderPreset(ctx, &spec.PatchProviderPresetRequest{
		ProviderName: prov,
		Body:         &spec.PatchProviderPresetRequestBody{DefaultModelPresetID: mpidPtr("m1")},
	})
	if err != nil {
		t.Fatalf("PatchProviderPreset(set default): %v", err)
	}

	newDisplay := spec.ProviderDisplayName(renamed)
	newOrigin := "https://changed.example.test"
	_, err = st.PatchProviderPreset(ctx, &spec.PatchProviderPresetRequest{
		ProviderName: prov,
		Body: &spec.PatchProviderPresetRequestBody{
			DisplayName: &newDisplay,
			Origin:      &newOrigin,
		},
	})
	if err != nil {
		t.Fatalf("PatchProviderPreset(metadata): %v", err)
	}

	pp := getProviderByName(t, st, ctx, prov, true)
	if string(pp.DisplayName) != renamed {
		t.Fatalf("displayName not updated: got=%q", pp.DisplayName)
	}
	if pp.Origin != newOrigin {
		t.Fatalf("origin not updated: got=%q", pp.Origin)
	}
	if pp.DefaultModelPresetID != "m1" {
		t.Fatalf("default model should be preserved: got=%q want=m1", pp.DefaultModelPresetID)
	}
	if _, ok := pp.ModelPresets["m1"]; !ok {
		t.Fatalf("model presets should be preserved when patching provider metadata")
	}
}

func TestModelPresetStore_PatchProviderPreset_UserProvider(t *testing.T) {
	st := newStore(t)
	ctx := t.Context()

	prov := inferenceSpec.ProviderName("user-prov-patch")
	postUserProvider(t, st, prov, true)
	postUserModelPreset(t, ctx, st, prov, "m1", true)
	postUserModelPreset(t, ctx, st, prov, "m2", true)

	// Set default to m1.
	_, err := st.PatchProviderPreset(ctx, &spec.PatchProviderPresetRequest{
		ProviderName: prov,
		Body: &spec.PatchProviderPresetRequestBody{
			DefaultModelPresetID: mpidPtr("m1"),
		},
	})
	if err != nil {
		t.Fatalf("initial PatchProviderPreset(default=m1): %v", err)
	}

	tests := []struct {
		name        string
		req         *spec.PatchProviderPresetRequest
		wantErrIs   error
		wantErrText string
		verify      func(t *testing.T)
	}{
		{
			name:      "nil_request",
			req:       nil,
			wantErrIs: spec.ErrInvalidDir,
		},
		{
			name: "both_fields_nil",
			req: &spec.PatchProviderPresetRequest{
				ProviderName: prov,
				Body:         &spec.PatchProviderPresetRequestBody{},
			},
			wantErrIs: spec.ErrInvalidDir,
		},
		{
			name: "unknown_provider",
			req: &spec.PatchProviderPresetRequest{
				ProviderName: "ghost",
				Body:         &spec.PatchProviderPresetRequestBody{IsEnabled: boolPtr(false)},
			},
			wantErrIs: spec.ErrProviderNotFound,
		},
		{
			name: "invalid_default_model_id_tag",
			req: &spec.PatchProviderPresetRequest{
				ProviderName: prov,
				Body: &spec.PatchProviderPresetRequestBody{
					DefaultModelPresetID: mpidPtr("white space"),
				},
			},
			wantErrText: "invalid tag",
		},
		{
			name: "change_metadata",
			req: &spec.PatchProviderPresetRequest{
				ProviderName: prov,
				Body: &spec.PatchProviderPresetRequestBody{
					DisplayName: providerDisplayNamePtr("Patched Name"),
					Origin:      stringPtr("https://patched.example.test"),
				},
			},
			verify: func(t *testing.T) {
				t.Helper()
				pp := getProviderByName(t, st, ctx, prov, true)
				if pp.DisplayName != "Patched Name" || pp.Origin != "https://patched.example.test" {
					t.Fatalf("provider metadata not patched")
				}
			},
		},
		{
			name: "disable_provider",
			req: &spec.PatchProviderPresetRequest{
				ProviderName: prov,
				Body:         &spec.PatchProviderPresetRequestBody{IsEnabled: boolPtr(false)},
			},
			verify: func(t *testing.T) {
				t.Helper()
				pp := getProviderByName(t, st, ctx, prov, true)
				if pp.IsEnabled {
					t.Fatalf("expected provider disabled")
				}
			},
		},
		{
			name: "change_default_model",
			req: &spec.PatchProviderPresetRequest{
				ProviderName: prov,
				Body: &spec.PatchProviderPresetRequestBody{
					DefaultModelPresetID: mpidPtr("m2"),
				},
			},
			verify: func(t *testing.T) {
				t.Helper()
				pp := getProviderByName(t, st, ctx, prov, true)
				if pp.DefaultModelPresetID != "m2" {
					t.Fatalf("default model not updated: got=%q want=m2", pp.DefaultModelPresetID)
				}
			},
		},
		{
			name: "default_model_not_found",
			req: &spec.PatchProviderPresetRequest{
				ProviderName: prov,
				Body: &spec.PatchProviderPresetRequestBody{
					DefaultModelPresetID: mpidPtr("missing"),
				},
			},
			wantErrIs: spec.ErrModelPresetNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := st.PatchProviderPreset(ctx, tt.req)
			if tt.wantErrIs != nil {
				wantErrIs(t, err, tt.wantErrIs)
				return
			}
			if tt.wantErrText != "" {
				wantErrContains(t, err, tt.wantErrText)
				return
			}
			if err != nil {
				t.Fatalf("unexpected: %v", err)
			}
			if tt.verify != nil {
				tt.verify(t)
			}
		})
	}
}

func TestModelPresetStore_PatchProviderPreset_BuiltInProvider_ToggleAndDefaultModel(t *testing.T) {
	st := newStore(t)
	ctx := t.Context()

	pn, pp := anyBuiltInProviderFromStore(t, st)

	t.Run("toggle_isEnabled", func(t *testing.T) {
		newEnabled := !pp.IsEnabled
		_, err := st.PatchProviderPreset(ctx, &spec.PatchProviderPresetRequest{
			ProviderName: pn,
			Body:         &spec.PatchProviderPresetRequestBody{IsEnabled: boolPtr(newEnabled)},
		})
		if err != nil {
			t.Fatalf("PatchProviderPreset(builtin isEnabled): %v", err)
		}

		got := getProviderByName(t, st, ctx, pn, true)
		if got.IsEnabled != newEnabled {
			t.Fatalf("builtin provider enable mismatch: got=%v want=%v", got.IsEnabled, newEnabled)
		}
	})

	t.Run("change_default_model_if_possible", func(t *testing.T) {
		// Need a builtin provider with >=2 models.
		pn2, pp2 := builtInProviderWithAtLeastNModelsFromStore(t, ctx, st, 2)
		oldDefault := pp2.DefaultModelPresetID

		newID := anotherModelID(pp2, oldDefault)
		if newID == "" {
			t.Skip("no alternate model id found")
		}

		_, err := st.PatchProviderPreset(ctx, &spec.PatchProviderPresetRequest{
			ProviderName: pn2,
			Body: &spec.PatchProviderPresetRequestBody{
				DefaultModelPresetID: mpidPtr(newID),
			},
		})
		if err != nil {
			t.Fatalf("PatchProviderPreset(builtin defaultModelPresetID): %v", err)
		}

		got := getProviderByName(t, st, ctx, pn2, true)
		if got.DefaultModelPresetID != newID {
			t.Fatalf("defaultModelPresetID not updated: got=%q want=%q", got.DefaultModelPresetID, newID)
		}
	})
}

func TestModelPresetStore_DeleteProviderPreset(t *testing.T) {
	st := newStore(t)
	ctx := t.Context()

	builtinName, _ := anyBuiltInProviderFromStore(t, st)

	userProv := inferenceSpec.ProviderName("user-prov-del")
	postUserProvider(t, st, userProv, true)
	postUserModelPreset(t, ctx, st, userProv, "m1", true)

	tests := []struct {
		name        string
		req         *spec.DeleteProviderPresetRequest
		wantErrIs   error
		wantErrText string
		before      func()
	}{
		{
			name:      "nil_request",
			req:       nil,
			wantErrIs: spec.ErrInvalidDir,
		},
		{
			name: "built_in_readonly",
			req: &spec.DeleteProviderPresetRequest{
				ProviderName: builtinName,
			},
			wantErrIs: spec.ErrBuiltInReadOnly,
		},
		{
			name: "cannot_delete_non_empty_provider",
			req: &spec.DeleteProviderPresetRequest{
				ProviderName: userProv,
			},
			wantErrText: "not empty",
		},
		{
			name: "delete_after_removing_models",
			before: func() {
				_, _ = st.DeleteModelPreset(ctx, &spec.DeleteModelPresetRequest{
					ProviderName:  userProv,
					ModelPresetID: "m1",
				})
			},
			req: &spec.DeleteProviderPresetRequest{
				ProviderName: userProv,
			},
		},
		{
			name: "delete_again_not_found",
			req: &spec.DeleteProviderPresetRequest{
				ProviderName: userProv,
			},
			wantErrIs: spec.ErrProviderNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.before != nil {
				tt.before()
			}
			_, err := st.DeleteProviderPreset(ctx, tt.req)
			if tt.wantErrIs != nil {
				wantErrIs(t, err, tt.wantErrIs)
				return
			}
			if tt.wantErrText != "" {
				wantErrContains(t, err, tt.wantErrText)
				return
			}
			if err != nil {
				t.Fatalf("unexpected: %v", err)
			}
		})
	}
}

func TestModelPresetStore_ModelPreset_UserCRUD(t *testing.T) {
	st := newStore(t)
	ctx := t.Context()

	userProv := inferenceSpec.ProviderName("user-prov-models")
	postUserProvider(t, st, userProv, true)

	temp := 0.1
	builtinName, _ := anyBuiltInProviderFromStore(t, st)

	t.Run("PostModelPreset_validation_and_errors", func(t *testing.T) {
		tests := []struct {
			name        string
			req         *spec.PostModelPresetRequest
			wantErrIs   error
			wantErrText string
		}{
			{
				name:      "nil_request",
				req:       nil,
				wantErrIs: spec.ErrInvalidDir,
			},
			{
				name: "nil_body",
				req: &spec.PostModelPresetRequest{
					ProviderName:  userProv,
					ModelPresetID: "m1",
					Body:          nil,
				},
				wantErrIs: spec.ErrInvalidDir,
			},
			{
				name: "invalid_modelPresetID_tag",
				req: &spec.PostModelPresetRequest{
					ProviderName:  userProv,
					ModelPresetID: "white space",
					Body: &spec.PostModelPresetRequestBody{
						Name:        "n",
						Slug:        "m1",
						DisplayName: "x",
						IsEnabled:   true,
						ModelPresetPatch: spec.ModelPresetPatch{
							Temperature: &temp,
						},
					},
				},
				wantErrText: "invalid tag",
			},
			{
				name: "invalid_slug_tag",
				req: &spec.PostModelPresetRequest{
					ProviderName:  userProv,
					ModelPresetID: "m1",
					Body: &spec.PostModelPresetRequestBody{
						Name:        "n",
						Slug:        "white space",
						DisplayName: "x",
						IsEnabled:   true,
						ModelPresetPatch: spec.ModelPresetPatch{
							Temperature: &temp,
						},
					},
				},
				wantErrText: "invalid tag",
			},
			{
				name: "missing_temp_and_reasoning",
				req: &spec.PostModelPresetRequest{
					ProviderName:  userProv,
					ModelPresetID: "m1",
					Body: &spec.PostModelPresetRequestBody{
						Name:        "n",
						Slug:        "m1",
						DisplayName: "x",
						IsEnabled:   true,
						// Neither Temperature nor Reasoning => validateModelPreset error.
					},
				},
				wantErrText: "either reasoning or temperature must be set",
			},
			{
				name: "too_many_stop_sequences",
				req: &spec.PostModelPresetRequest{
					ProviderName:  userProv,
					ModelPresetID: "m1",
					Body: &spec.PostModelPresetRequestBody{
						Name:        "n",
						Slug:        "m1",
						DisplayName: "x",
						IsEnabled:   true,
						ModelPresetPatch: spec.ModelPresetPatch{
							Temperature: &temp,
							StopSequences: []string{
								"a", "b", "c", "d", "e",
							},
						},
					},
				},
				wantErrText: "too many stop sequences",
			},
			{
				name: "unknown_provider",
				req: &spec.PostModelPresetRequest{
					ProviderName:  "ghost",
					ModelPresetID: "m1",
					Body: &spec.PostModelPresetRequestBody{
						Name:        "n",
						Slug:        "m1",
						DisplayName: "x",
						IsEnabled:   true,
						ModelPresetPatch: spec.ModelPresetPatch{
							Temperature: &temp,
						},
					},
				},
				wantErrIs: spec.ErrProviderNotFound,
			},
			{
				name: "built_in_readonly",
				req: &spec.PostModelPresetRequest{
					ProviderName:  builtinName,
					ModelPresetID: "m1",
					Body: &spec.PostModelPresetRequestBody{
						Name:        "n",
						Slug:        "m1",
						DisplayName: "x",
						IsEnabled:   true,
						ModelPresetPatch: spec.ModelPresetPatch{
							Temperature: &temp,
						},
					},
				},
				wantErrIs: spec.ErrBuiltInReadOnly,
			},
			{
				name: "happy_post",
				req: &spec.PostModelPresetRequest{
					ProviderName:  userProv,
					ModelPresetID: "m1",
					Body: &spec.PostModelPresetRequestBody{
						Name:        "model-one",
						Slug:        "m1",
						DisplayName: "Model One",
						IsEnabled:   true,
						ModelPresetPatch: spec.ModelPresetPatch{
							Temperature: &temp,
						},
					},
				},
			},
			{
				name: "already_exists",
				req: &spec.PostModelPresetRequest{
					ProviderName:  userProv,
					ModelPresetID: "m1",
					Body: &spec.PostModelPresetRequestBody{
						Name:             "model-one",
						Slug:             "m1",
						DisplayName:      "Model One",
						IsEnabled:        true,
						ModelPresetPatch: spec.ModelPresetPatch{Temperature: &temp},
					},
				},
				wantErrIs: spec.ErrModelPresetAlreadyExists,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := st.PostModelPreset(ctx, tt.req)
				if tt.wantErrIs != nil {
					wantErrIs(t, err, tt.wantErrIs)
					return
				}
				if tt.wantErrText != "" {
					wantErrContains(t, err, tt.wantErrText)
					return
				}
				if err != nil {
					t.Fatalf("unexpected: %v", err)
				}
			})
		}
	})

	t.Run("PatchModelPreset_user_branch_updates_and_clears_optional_fields", func(t *testing.T) {
		temp2 := 0.2
		sysPrompt := "system"
		rawJSON := `{"x":1}`

		_, err := st.PostModelPreset(ctx, &spec.PostModelPresetRequest{
			ProviderName:  userProv,
			ModelPresetID: "m4",
			Body: &spec.PostModelPresetRequestBody{
				Name:        "model-four",
				Slug:        "m4",
				DisplayName: "Model Four",
				IsEnabled:   true,
				ModelPresetPatch: spec.ModelPresetPatch{
					Temperature:                 &temp2,
					SystemPrompt:                &sysPrompt,
					StopSequences:               []string{"END"},
					AdditionalParametersRawJSON: &rawJSON,
				},
			},
		})
		if err != nil {
			t.Fatalf("PostModelPreset(seed m4): %v", err)
		}

		reasoning := inferenceSpec.ReasoningParam{
			Type:   inferenceSpec.ReasoningTypeHybridWithTokens,
			Tokens: 8,
		}
		_, err = st.PatchModelPreset(ctx, &spec.PatchModelPresetRequest{
			ProviderName:  userProv,
			ModelPresetID: "m4",
			Body: &spec.PatchModelPresetRequestBody{
				ModelPresetPatch: spec.ModelPresetPatch{
					Reasoning:                   &reasoning,
					Temperature:                 floatPtr(0),
					SystemPrompt:                stringPtr(""),
					AdditionalParametersRawJSON: stringPtr(""),
				},
				ClearStopSequences: true,
			},
		})
		if err != nil {
			t.Fatalf("PatchModelPreset(clear optional fields): %v", err)
		}

		pp := getProviderByName(t, st, ctx, userProv, true)
		got := pp.ModelPresets["m4"]
		if got.Temperature != nil && (*got.Temperature != 0) {
			t.Fatalf("optional field patch/clear did not apply correctly: temp, %v", *got.Temperature)
		}

		if got.SystemPrompt != nil && *got.SystemPrompt != "" {
			t.Fatalf("optional field patch/clear did not apply correctly: sysPrompt")
		}

		if got.StopSequences != nil {
			t.Fatalf("optional field patch/clear did not apply correctly: stopSequences")
		}

		if got.AdditionalParametersRawJSON != nil && *got.AdditionalParametersRawJSON != "" {
			t.Fatalf("optional field patch/clear did not apply correctly: additionalParams")
		}

		if got.Reasoning == nil {
			t.Fatalf("optional field patch/clear did not apply correctly: reasoning")
		}
	})

	t.Run("PatchModelPreset_user_branch", func(t *testing.T) {
		postUserModelPreset(t, ctx, st, userProv, "m3", true)

		// Unknown provider.
		_, err := st.PatchModelPreset(ctx, &spec.PatchModelPresetRequest{
			ProviderName:  "ghost",
			ModelPresetID: "m3",
			Body:          &spec.PatchModelPresetRequestBody{IsEnabled: boolPtr(false)},
		})
		wantErrIs(t, err, spec.ErrProviderNotFound)

		// Unknown model.
		_, err = st.PatchModelPreset(ctx, &spec.PatchModelPresetRequest{
			ProviderName:  userProv,
			ModelPresetID: "ghost",
			Body:          &spec.PatchModelPresetRequestBody{IsEnabled: boolPtr(false)},
		})
		wantErrIs(t, err, spec.ErrModelPresetNotFound)

		// Happy patch.
		_, err = st.PatchModelPreset(ctx, &spec.PatchModelPresetRequest{
			ProviderName:  userProv,
			ModelPresetID: "m3",
			Body:          &spec.PatchModelPresetRequestBody{IsEnabled: boolPtr(false)},
		})
		if err != nil {
			t.Fatalf("PatchModelPreset: %v", err)
		}
		pp := getProviderByName(t, st, ctx, userProv, true)
		if pp.ModelPresets["m3"].IsEnabled {
			t.Fatalf("expected model disabled")
		}
	})

	t.Run("DeleteModelPreset_resets_default_if_pointing_to_deleted", func(t *testing.T) {
		postUserModelPreset(t, ctx, st, userProv, "m-default", true)

		// Set provider default to m-default.
		_, err := st.PatchProviderPreset(ctx, &spec.PatchProviderPresetRequest{
			ProviderName: userProv,
			Body: &spec.PatchProviderPresetRequestBody{
				DefaultModelPresetID: mpidPtr("m-default"),
			},
		})
		if err != nil {
			t.Fatalf("PatchProviderPreset(set default): %v", err)
		}

		_, err = st.DeleteModelPreset(ctx, &spec.DeleteModelPresetRequest{
			ProviderName:  userProv,
			ModelPresetID: "m-default",
		})
		if err != nil {
			t.Fatalf("DeleteModelPreset: %v", err)
		}

		pp := getProviderByName(t, st, ctx, userProv, true)
		if pp.DefaultModelPresetID != "" {
			t.Fatalf("expected default model reset to empty after delete, got %q", pp.DefaultModelPresetID)
		}
	})
}

func TestModelPresetStore_ModelPreset_BuiltInToggle_ViaPatchModelPreset(t *testing.T) {
	t.Parallel()

	st := newStore(t)
	ctx := t.Context()

	pn, pp := anyBuiltInProviderFromStore(t, st)
	mid, mp := anyModelID(pp)
	if mid == "" {
		t.Skip("built-in provider has no models")
	}

	_, err := st.PatchModelPreset(ctx, &spec.PatchModelPresetRequest{
		ProviderName:  pn,
		ModelPresetID: mid,
		Body:          &spec.PatchModelPresetRequestBody{IsEnabled: boolPtr(!(mp.IsEnabled))},
	})
	if err != nil {
		t.Fatalf("PatchModelPreset(builtin): %v", err)
	}

	got := getProviderByName(t, st, ctx, pn, true)
	if got.ModelPresets[mid].IsEnabled == mp.IsEnabled {
		t.Fatalf("expected builtin model enabled toggled from %v", mp.IsEnabled)
	}
}

func TestModelPresetStore_ListProviderPresets_FilterAndPaging(t *testing.T) {
	st := newStore(t)
	ctx := t.Context()

	p1 := inferenceSpec.ProviderName("user-p1")
	p2 := inferenceSpec.ProviderName("user-p2")
	p3 := inferenceSpec.ProviderName("user-p3")

	postUserProvider(t, st, p1, true)
	postUserProvider(t, st, p2, false)
	postUserProvider(t, st, p3, true)

	names := []inferenceSpec.ProviderName{p1, p2, p3}

	t.Run("disabled_filtered_out_by_default", func(t *testing.T) {
		resp, err := st.ListProviderPresets(ctx, &spec.ListProviderPresetsRequest{
			Names: names,
		})
		if err != nil {
			t.Fatalf("ListProviderPresets: %v", err)
		}
		if len(resp.Body.Providers) != 2 {
			t.Fatalf("expected 2 enabled providers, got %d", len(resp.Body.Providers))
		}
		for _, p := range resp.Body.Providers {
			if p.Name == p2 {
				t.Fatalf("did not expect disabled provider to be returned")
			}
		}
	})

	t.Run("include_disabled", func(t *testing.T) {
		resp, err := st.ListProviderPresets(ctx, &spec.ListProviderPresetsRequest{
			Names:           names,
			IncludeDisabled: true,
		})
		if err != nil {
			t.Fatalf("ListProviderPresets: %v", err)
		}
		if len(resp.Body.Providers) != 3 {
			t.Fatalf("expected 3 providers, got %d", len(resp.Body.Providers))
		}
	})

	t.Run("paging_pageSize_1_and_token_roundtrip", func(t *testing.T) {
		seen := map[inferenceSpec.ProviderName]bool{}

		req := &spec.ListProviderPresetsRequest{
			Names:           names,
			IncludeDisabled: true,
			PageSize:        1,
		}

		var token *string
		for i := range 10 {
			if token != nil {
				req = &spec.ListProviderPresetsRequest{PageToken: *token}
			}
			resp, err := st.ListProviderPresets(ctx, req)
			if err != nil {
				t.Fatalf("ListProviderPresets(page %d): %v", i, err)
			}
			for _, p := range resp.Body.Providers {
				seen[p.Name] = true
			}
			token = resp.Body.NextPageToken
			if token == nil {
				break
			}
		}

		for _, n := range names {
			if !seen[n] {
				t.Fatalf("paging missed provider %q", n)
			}
		}
	})

	t.Run("invalid_page_token_is_ignored_no_error", func(t *testing.T) {
		_, err := st.ListProviderPresets(ctx, &spec.ListProviderPresetsRequest{
			PageToken: "not-base64!!!",
		})
		if err != nil {
			t.Fatalf("expected no error for invalid token, got %v", err)
		}
	})
}

func TestModelPresetStore_ListProviderPresets_TokenPreservesFilters(t *testing.T) {
	t.Parallel()

	st := newStore(t)
	ctx := t.Context()

	// Ensure >1 providers and include a disabled one.
	p1 := inferenceSpec.ProviderName("user-tok-1")
	p2 := inferenceSpec.ProviderName("user-tok-2") // disabled
	p3 := inferenceSpec.ProviderName("user-tok-3")

	postUserProvider(t, st, p1, true)
	postUserProvider(t, st, p2, false)
	postUserProvider(t, st, p3, true)

	names := []inferenceSpec.ProviderName{p1, p2, p3}

	// First call to get token.
	resp1, err := st.ListProviderPresets(ctx, &spec.ListProviderPresetsRequest{
		Names:           names,
		IncludeDisabled: true,
		PageSize:        1,
	})
	if err != nil {
		t.Fatalf("ListProviderPresets: %v", err)
	}
	if len(resp1.Body.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(resp1.Body.Providers))
	}
	if resp1.Body.NextPageToken == nil {
		t.Fatalf("expected next page token")
	}

	// Decode returned token and assert it preserved our filters.
	tok := decodeProviderPageToken(t, *resp1.Body.NextPageToken)
	if tok.PageSize != 1 {
		t.Fatalf("token PageSize mismatch: got=%d want=1", tok.PageSize)
	}
	if !tok.IncludeDisabled {
		t.Fatalf("token IncludeDisabled mismatch: got=false want=true")
	}
	if len(tok.Names) != len(names) {
		t.Fatalf("token Names length mismatch: got=%d want=%d", len(tok.Names), len(names))
	}
}

func TestModelPresetStore_ListProviderPresets_PageSizeClamping_Heavy(t *testing.T) {
	// This test intentionally creates DefaultPageSize+1 user providers to verify clamp behavior.
	// It can be skipped in -short runs.
	if testing.Short() {
		t.Skip("skipping heavy paging test in short mode")
	}
	t.Parallel()

	dir := t.TempDir()
	st := newStoreAtDir(t, dir)
	ctx := t.Context()

	total := spec.DefaultPageSize + 1
	names := make([]inferenceSpec.ProviderName, 0, total)

	for i := range total {
		pn := inferenceSpec.ProviderName("user-many-" + strconv.Itoa(i))
		postUserProvider(t, st, pn, true)
		names = append(names, pn)
	}

	// Ask for an invalid page size (> DefaultPageSize), expecting clamp back to DefaultPageSize.
	resp, err := st.ListProviderPresets(ctx, &spec.ListProviderPresetsRequest{
		Names:           names,
		IncludeDisabled: true,
		PageSize:        spec.MaxPageSize + 1000,
	})
	if err != nil {
		t.Fatalf("ListProviderPresets: %v", err)
	}
	if len(resp.Body.Providers) != spec.DefaultPageSize {
		t.Fatalf("expected clamped page size %d, got %d", spec.DefaultPageSize, len(resp.Body.Providers))
	}
	if resp.Body.NextPageToken == nil {
		t.Fatalf("expected next token for %d providers", total)
	}

	tok := decodeProviderPageToken(t, *resp.Body.NextPageToken)
	if tok.PageSize != spec.DefaultPageSize {
		t.Fatalf("expected token page size clamped to %d, got %d", spec.DefaultPageSize, tok.PageSize)
	}
}

func TestModelPresetStore_PatchProviderPreset_User_BothFields_NoOp_AtomicOnError(t *testing.T) {
	type setupOut struct {
		prov inferenceSpec.ProviderName
	}

	setup := func(t *testing.T) (*ModelPresetStore, setupOut) {
		t.Helper()
		st := newStore(t)
		ctx := t.Context()

		prov := inferenceSpec.ProviderName("user-prov-patch-both")
		postUserProvider(t, st, prov, true)
		postUserModelPreset(t, ctx, st, prov, "m1", true)
		postUserModelPreset(t, ctx, st, prov, "m2", true)

		// Set default to m1.
		_, err := st.PatchProviderPreset(ctx, &spec.PatchProviderPresetRequest{
			ProviderName: prov,
			Body: &spec.PatchProviderPresetRequestBody{
				DefaultModelPresetID: mpidPtr("m1"),
			},
		})
		if err != nil {
			t.Fatalf("seed PatchProviderPreset(default=m1): %v", err)
		}
		return st, setupOut{prov: prov}
	}

	tests := []struct {
		name        string
		req         func(out setupOut) *spec.PatchProviderPresetRequest
		wantErrIs   error
		wantErrText string
		verify      func(t *testing.T, st *ModelPresetStore, out setupOut, before spec.ProviderPreset)
	}{
		{
			name: "both_fields_applied",
			req: func(out setupOut) *spec.PatchProviderPresetRequest {
				return &spec.PatchProviderPresetRequest{
					ProviderName: out.prov,
					Body: &spec.PatchProviderPresetRequestBody{
						IsEnabled:            boolPtr(false),
						DefaultModelPresetID: mpidPtr("m2"),
					},
				}
			},
			verify: func(t *testing.T, st *ModelPresetStore, out setupOut, _ spec.ProviderPreset) {
				t.Helper()
				got := getProviderByName(t, st, t.Context(), out.prov, true)
				if got.IsEnabled != false {
					t.Fatalf("isEnabled not applied: got=%v want=false", got.IsEnabled)
				}
				if got.DefaultModelPresetID != "m2" {
					t.Fatalf("default model not applied: got=%q want=m2", got.DefaultModelPresetID)
				}
			},
		},
		{
			name: "no_op_does_not_bump_modifiedAt",
			req: func(out setupOut) *spec.PatchProviderPresetRequest {
				return &spec.PatchProviderPresetRequest{
					ProviderName: out.prov,
					Body: &spec.PatchProviderPresetRequestBody{
						IsEnabled:            boolPtr(true),
						DefaultModelPresetID: mpidPtr("m1"),
					},
				}
			},
			verify: func(t *testing.T, st *ModelPresetStore, out setupOut, before spec.ProviderPreset) {
				t.Helper()
				time.Sleep(2 * time.Millisecond) // make bump observable if it happens
				_ = out
				after := getProviderByName(t, st, t.Context(), before.Name, true)
				if !after.ModifiedAt.Equal(before.ModifiedAt) {
					t.Fatalf("ModifiedAt should not change on no-op: before=%v after=%v",
						before.ModifiedAt, after.ModifiedAt)
				}
			},
		},
		{
			name: "atomic_on_error_default_model_missing_does_not_persist_isEnabled",
			req: func(out setupOut) *spec.PatchProviderPresetRequest {
				return &spec.PatchProviderPresetRequest{
					ProviderName: out.prov,
					Body: &spec.PatchProviderPresetRequestBody{
						IsEnabled:            boolPtr(false),
						DefaultModelPresetID: mpidPtr("missing"),
					},
				}
			},
			wantErrIs: spec.ErrModelPresetNotFound,
			verify: func(t *testing.T, st *ModelPresetStore, out setupOut, _ spec.ProviderPreset) {
				t.Helper()
				got := getProviderByName(t, st, t.Context(), out.prov, true)
				if got.IsEnabled != true {
					t.Fatalf("isEnabled must not be persisted when request fails: got=%v want=true", got.IsEnabled)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, out := setup(t)
			ctx := t.Context()

			before := getProviderByName(t, st, ctx, out.prov, true)

			_, err := st.PatchProviderPreset(ctx, tt.req(out))
			if tt.wantErrIs != nil {
				wantErrIs(t, err, tt.wantErrIs)
				if tt.verify != nil {
					tt.verify(t, st, out, before)
				}
				return
			}
			if tt.wantErrText != "" {
				wantErrContains(t, err, tt.wantErrText)
				return
			}
			if err != nil {
				t.Fatalf("unexpected: %v", err)
			}
			if tt.verify != nil {
				tt.verify(t, st, out, before)
			}
		})
	}
}

func TestModelPresetStore_PatchProviderPreset_BuiltIn_BothFieldsAndErrors(t *testing.T) {
	st := newStore(t)
	ctx := t.Context()

	pn2, pp2 := builtInProviderWithAtLeastNModelsFromStore(t, ctx, st, 2)
	alt := anotherModelID(pp2, pp2.DefaultModelPresetID)
	if alt == "" {
		t.Skip("no alternate model found")
	}

	tests := []struct {
		name        string
		req         *spec.PatchProviderPresetRequest
		wantErrIs   error
		wantErrText string
		verify      func(t *testing.T)
	}{
		{
			name: "invalid_default_model_id_tag_rejected",
			req: &spec.PatchProviderPresetRequest{
				ProviderName: pn2,
				Body: &spec.PatchProviderPresetRequestBody{
					DefaultModelPresetID: mpidPtr("white space"),
				},
			},
			wantErrText: "invalid tag",
		},
		{
			name: "unknown_default_model_returns_not_found",
			req: &spec.PatchProviderPresetRequest{
				ProviderName: pn2,
				Body: &spec.PatchProviderPresetRequestBody{
					DefaultModelPresetID: mpidPtr("ghost"),
				},
			},
			wantErrIs: spec.ErrModelPresetNotFound,
		},
		{
			name: "both_fields_applied",
			req: &spec.PatchProviderPresetRequest{
				ProviderName: pn2,
				Body: &spec.PatchProviderPresetRequestBody{
					IsEnabled:            boolPtr(!pp2.IsEnabled),
					DefaultModelPresetID: mpidPtr(alt),
				},
			},
			verify: func(t *testing.T) {
				t.Helper()
				got := getProviderByName(t, st, ctx, pn2, true)
				if got.IsEnabled == pp2.IsEnabled {
					t.Fatalf("expected isEnabled to change")
				}
				if got.DefaultModelPresetID != alt {
					t.Fatalf("defaultModelPresetID not applied: got=%q want=%q", got.DefaultModelPresetID, alt)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := st.PatchProviderPreset(ctx, tt.req)
			if tt.wantErrIs != nil {
				wantErrIs(t, err, tt.wantErrIs)
				return
			}
			if tt.wantErrText != "" {
				wantErrContains(t, err, tt.wantErrText)
				return
			}
			if err != nil {
				t.Fatalf("unexpected: %v", err)
			}
			if tt.verify != nil {
				tt.verify(t)
			}
		})
	}
}

func TestModelPresetStore_PatchModelPreset_AdditionalErrors(t *testing.T) {
	st := newStore(t)
	ctx := t.Context()

	bpn, _ := anyBuiltInProviderFromStore(t, st)

	tests := []struct {
		name      string
		req       *spec.PatchModelPresetRequest
		wantErrIs error
	}{
		{
			name:      "nil_request",
			req:       nil,
			wantErrIs: spec.ErrInvalidDir,
		},
		{
			name: "nil_body",
			req: &spec.PatchModelPresetRequest{
				ProviderName:  "p",
				ModelPresetID: "m",
				Body:          nil,
			},
			wantErrIs: spec.ErrInvalidDir,
		},
		{
			name: "empty_providerName",
			req: &spec.PatchModelPresetRequest{
				ProviderName:  "",
				ModelPresetID: "m",
				Body:          &spec.PatchModelPresetRequestBody{IsEnabled: boolPtr(false)},
			},
			wantErrIs: spec.ErrInvalidDir,
		},
		{
			name: "empty_modelPresetID",
			req: &spec.PatchModelPresetRequest{
				ProviderName:  "p",
				ModelPresetID: "",
				Body:          &spec.PatchModelPresetRequestBody{IsEnabled: boolPtr(false)},
			},
			wantErrIs: spec.ErrInvalidDir,
		},
		{
			name: "builtin_provider_unknown_model",
			req: &spec.PatchModelPresetRequest{
				ProviderName:  bpn,
				ModelPresetID: "ghost",
				Body:          &spec.PatchModelPresetRequestBody{IsEnabled: boolPtr(false)},
			},
			wantErrIs: spec.ErrModelPresetNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := st.PatchModelPreset(ctx, tt.req)
			wantErrIs(t, err, tt.wantErrIs)
		})
	}
}

func TestModelPresetStore_DeleteModelPreset_AdditionalCases(t *testing.T) {
	st := newStore(t)
	ctx := t.Context()

	userProv := inferenceSpec.ProviderName("user-del-model")
	postUserProvider(t, st, userProv, true)
	postUserModelPreset(t, ctx, st, userProv, "m1", true)
	postUserModelPreset(t, ctx, st, userProv, "m2", true)

	// Set default to m1.
	_, err := st.PatchProviderPreset(ctx, &spec.PatchProviderPresetRequest{
		ProviderName: userProv,
		Body: &spec.PatchProviderPresetRequestBody{
			DefaultModelPresetID: mpidPtr("m1"),
		},
	})
	if err != nil {
		t.Fatalf("PatchProviderPreset(set default): %v", err)
	}

	builtinName, _ := anyBuiltInProviderFromStore(t, st)

	tests := []struct {
		name        string
		req         *spec.DeleteModelPresetRequest
		wantErrIs   error
		wantErrText string
		verify      func(t *testing.T)
	}{
		{
			name:      "nil_request",
			req:       nil,
			wantErrIs: spec.ErrInvalidDir,
		},
		{
			name: "empty_providerName",
			req: &spec.DeleteModelPresetRequest{
				ProviderName:  "",
				ModelPresetID: "m",
			},
			wantErrIs: spec.ErrInvalidDir,
		},
		{
			name: "empty_modelPresetID",
			req: &spec.DeleteModelPresetRequest{
				ProviderName:  "p",
				ModelPresetID: "",
			},
			wantErrIs: spec.ErrInvalidDir,
		},
		{
			name: "builtin_readonly",
			req: &spec.DeleteModelPresetRequest{
				ProviderName:  builtinName,
				ModelPresetID: "anything",
			},
			wantErrIs: spec.ErrBuiltInReadOnly,
		},
		{
			name: "unknown_provider",
			req: &spec.DeleteModelPresetRequest{
				ProviderName:  "ghost",
				ModelPresetID: "m1",
			},
			wantErrIs: spec.ErrProviderNotFound,
		},
		{
			name: "unknown_model",
			req: &spec.DeleteModelPresetRequest{
				ProviderName:  userProv,
				ModelPresetID: "ghost",
			},
			wantErrIs: spec.ErrModelPresetNotFound,
		},
		{
			name: "delete_non_default_does_not_reset_default",
			req: &spec.DeleteModelPresetRequest{
				ProviderName:  userProv,
				ModelPresetID: "m2",
			},
			verify: func(t *testing.T) {
				t.Helper()
				pp := getProviderByName(t, st, ctx, userProv, true)
				if pp.DefaultModelPresetID != "m1" {
					t.Fatalf("default must remain m1: got=%q", pp.DefaultModelPresetID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := st.DeleteModelPreset(ctx, tt.req)
			if tt.wantErrIs != nil {
				wantErrIs(t, err, tt.wantErrIs)
				return
			}
			if tt.wantErrText != "" {
				wantErrContains(t, err, tt.wantErrText)
				return
			}
			if err != nil {
				t.Fatalf("unexpected: %v", err)
			}
			if tt.verify != nil {
				tt.verify(t)
			}
		})
	}
}

func TestModelPresetStore_PostModelPreset_AdditionalValidation(t *testing.T) {
	st := newStore(t)
	ctx := t.Context()

	userProv := inferenceSpec.ProviderName("user-model-validate-more")
	postUserProvider(t, st, userProv, true)

	temp := 0.1
	intPtr := func(v int) *int { return &v }

	baseReq := func() *spec.PostModelPresetRequest {
		return &spec.PostModelPresetRequest{
			ProviderName:  userProv,
			ModelPresetID: "m1",
			Body: &spec.PostModelPresetRequestBody{
				Name:        "n1",
				Slug:        "m1",
				DisplayName: "Model 1",
				IsEnabled:   true,
				ModelPresetPatch: spec.ModelPresetPatch{
					Temperature: &temp,
				},
			},
		}
	}

	tests := []struct {
		name        string
		req         func() *spec.PostModelPresetRequest
		wantErrText string
	}{
		{
			name: "empty_model_name",
			req: func() *spec.PostModelPresetRequest {
				r := baseReq()
				r.Body.Name = ""
				return r
			},
			wantErrText: "name is empty",
		},
		{
			name: "empty_display_name",
			req: func() *spec.PostModelPresetRequest {
				r := baseReq()
				r.Body.DisplayName = ""
				return r
			},
			wantErrText: "displayName is empty",
		},
		{
			name: "negative_max_prompt_length",
			req: func() *spec.PostModelPresetRequest {
				r := baseReq()
				r.Body.MaxPromptLength = intPtr(-1)
				return r
			},
			wantErrText: "maxPromptLength must be >= 0",
		},
		{
			name: "negative_max_output_length",
			req: func() *spec.PostModelPresetRequest {
				r := baseReq()
				r.Body.MaxOutputLength = intPtr(-1)
				return r
			},
			wantErrText: "maxOutputLength must be >= 0",
		},
		{
			name: "negative_timeout",
			req: func() *spec.PostModelPresetRequest {
				r := baseReq()
				r.Body.Timeout = intPtr(-1)
				return r
			},
			wantErrText: "timeout must be >= 0",
		},
		{
			name: "stop_sequence_empty_string",
			req: func() *spec.PostModelPresetRequest {
				r := baseReq()
				r.Body.StopSequences = []string{""}
				return r
			},
			wantErrText: "stopSequences[0] is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := st.PostModelPreset(ctx, tt.req())
			if tt.wantErrText != "" {
				wantErrContains(t, err, tt.wantErrText)
				return
			}
			if err != nil {
				t.Fatalf("unexpected: %v", err)
			}
		})
	}
}

func TestModelPresetStore_ListProviderPresets_PageTokenOverridesRequestParams(t *testing.T) {
	st := newStore(t)
	ctx := t.Context()

	p2 := inferenceSpec.ProviderName("user-override-2") // disabled, older
	p1 := inferenceSpec.ProviderName("user-override-1") // enabled, newer

	postUserProvider(t, st, p2, false)
	time.Sleep(2 * time.Millisecond)
	postUserProvider(t, st, p1, true)

	resp1, err := st.ListProviderPresets(ctx, &spec.ListProviderPresetsRequest{
		Names:           []inferenceSpec.ProviderName{p1, p2},
		IncludeDisabled: true,
		PageSize:        1,
	})
	if err != nil {
		t.Fatalf("ListProviderPresets(page1): %v", err)
	}
	if len(resp1.Body.Providers) != 1 {
		t.Fatalf("expected 1 provider on page1, got %d", len(resp1.Body.Providers))
	}
	if resp1.Body.Providers[0].Name != p1 {
		t.Fatalf("expected page1 to return %q first, got %q", p1, resp1.Body.Providers[0].Name)
	}
	if resp1.Body.NextPageToken == nil {
		t.Fatalf("expected next page token")
	}

	// Intentionally provide conflicting params; token must win.
	resp2, err := st.ListProviderPresets(ctx, &spec.ListProviderPresetsRequest{
		PageToken:       *resp1.Body.NextPageToken,
		Names:           []inferenceSpec.ProviderName{p1}, // would exclude p2 if not overridden
		IncludeDisabled: false,                            // would exclude disabled if not overridden
		PageSize:        999,
	})
	if err != nil {
		t.Fatalf("ListProviderPresets(page2): %v", err)
	}
	if len(resp2.Body.Providers) != 1 {
		t.Fatalf("expected 1 provider on page2, got %d", len(resp2.Body.Providers))
	}
	if resp2.Body.Providers[0].Name != p2 {
		t.Fatalf("expected page2 to return %q due to token override, got %q", p2, resp2.Body.Providers[0].Name)
	}
	if resp2.Body.Providers[0].IsEnabled {
		t.Fatalf("expected returned provider to be disabled")
	}
}

func TestModelPresetStore_ListProviderPresets_Base64ButInvalidJSONToken_Ignored(t *testing.T) {
	st := newStore(t)
	ctx := t.Context()

	disabled := inferenceSpec.ProviderName("user-disabled-token-test")
	postUserProvider(t, st, disabled, false)

	// Base64-valid, but NOT JSON.
	token := base64.StdEncoding.EncodeToString([]byte("not-json"))

	resp, err := st.ListProviderPresets(ctx, &spec.ListProviderPresetsRequest{
		PageToken: token,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for _, p := range resp.Body.Providers {
		if p.Name == disabled {
			t.Fatalf("expected disabled provider to be filtered out when token is ignored")
		}
	}
}

func TestModelPresetStore_UserData_PersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	st := newStoreAtDir(t, dir)
	ctx := t.Context()

	prov := inferenceSpec.ProviderName("user-persist-prov")
	postUserProvider(t, st, prov, true)
	postUserModelPreset(t, ctx, st, prov, "m1", true)

	_, err := st.PatchModelPreset(ctx, &spec.PatchModelPresetRequest{
		ProviderName:  prov,
		ModelPresetID: "m1",
		Body:          &spec.PatchModelPresetRequestBody{IsEnabled: boolPtr(false)},
	})
	if err != nil {
		t.Fatalf("PatchModelPreset: %v", err)
	}

	_, err = st.PatchProviderPreset(ctx, &spec.PatchProviderPresetRequest{
		ProviderName: prov,
		Body: &spec.PatchProviderPresetRequestBody{
			DefaultModelPresetID: mpidPtr("m1"),
		},
	})
	if err != nil {
		t.Fatalf("PatchProviderPreset: %v", err)
	}

	closeAndSleepOnWindows(t, st)

	st2 := newStoreAtDir(t, dir)

	pp := getProviderByName(t, st2, ctx, prov, true)
	if pp.DefaultModelPresetID != "m1" {
		t.Fatalf("defaultModelPresetID not persisted: got=%q want=m1", pp.DefaultModelPresetID)
	}
	m1 := pp.ModelPresets["m1"]
	if m1.ID != "m1" {
		t.Fatalf("model not persisted")
	}
	if m1.IsEnabled != false {
		t.Fatalf("model enabled flag not persisted: got=%v want=false", m1.IsEnabled)
	}
}

func TestModelPresetStore_BuiltinOverlay_PersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	st := newStoreAtDir(t, dir)
	ctx := t.Context()

	pn, pp := anyBuiltInProviderFromStore(t, st)
	mid, mp := anyModelID(pp)
	if mid == "" {
		t.Skip("builtin provider has no models")
	}

	_, err := st.PatchModelPreset(ctx, &spec.PatchModelPresetRequest{
		ProviderName:  pn,
		ModelPresetID: mid,
		Body:          &spec.PatchModelPresetRequestBody{IsEnabled: boolPtr(!mp.IsEnabled)},
	})
	if err != nil {
		t.Fatalf("PatchModelPreset(builtin): %v", err)
	}

	closeAndSleepOnWindows(t, st)

	st2 := newStoreAtDir(t, dir)
	got := getProviderByName(t, st2, ctx, pn, true)

	if got.ModelPresets[mid].IsEnabled == mp.IsEnabled {
		t.Fatalf("builtin overlay did not persist across reopen")
	}
}

func TestModelPresetStore_PostModelPreset_InvalidOutputParamAndReasoningErrors(t *testing.T) {
	st := newStore(t)
	ctx := t.Context()

	userProv := inferenceSpec.ProviderName("user-model-validate-op-reason")
	postUserProvider(t, st, userProv, true)

	// Helpers for pointers to foreign types.
	verbPtr := func(v inferenceSpec.OutputVerbosity) *inferenceSpec.OutputVerbosity { return &v }
	kindPtr := func(k inferenceSpec.OutputFormatKind) *inferenceSpec.OutputFormatKind { return &k }
	summaryPtr := func(s inferenceSpec.ReasoningSummaryStyle) *inferenceSpec.ReasoningSummaryStyle { return &s }

	base := func() *spec.PostModelPresetRequest {
		temp := 0.1
		return &spec.PostModelPresetRequest{
			ProviderName:  userProv,
			ModelPresetID: "m1",
			Body: &spec.PostModelPresetRequestBody{
				Name:        "n",
				Slug:        "m1",
				DisplayName: "x",
				IsEnabled:   true,
				ModelPresetPatch: spec.ModelPresetPatch{
					Temperature: &temp,
				},
			},
		}
	}

	tests := []struct {
		name        string
		req         func() *spec.PostModelPresetRequest
		wantErrText string
	}{
		{
			name: "outputParam_unknown_verbosity",
			req: func() *spec.PostModelPresetRequest {
				r := base()
				bad := inferenceSpec.OutputVerbosity("weird")
				r.Body.OutputParam = &inferenceSpec.OutputParam{
					Verbosity: verbPtr(bad),
				}
				return r
			},
			wantErrText: "unknown verbosity",
		},
		{
			name: "outputFormat_text_with_json_schema_param_is_invalid",
			req: func() *spec.PostModelPresetRequest {
				r := base()
				r.Body.OutputParam = &inferenceSpec.OutputParam{
					Format: &inferenceSpec.OutputFormat{
						Kind: inferenceSpec.OutputFormatKindText,
						JSONSchemaParam: &inferenceSpec.JSONSchemaParam{
							Name: "x",
							// Schema nil => still should fail because text kind forbids any JSONSchemaParam.
						},
					},
				}
				return r
			},
			wantErrText: "jsonSchemaParam must be nil when format.kind is text",
		},
		{
			name: "outputFormat_jsonSchema_requires_jsonSchemaParam",
			req: func() *spec.PostModelPresetRequest {
				r := base()
				r.Body.OutputParam = &inferenceSpec.OutputParam{
					Format: &inferenceSpec.OutputFormat{
						Kind: inferenceSpec.OutputFormatKindJSONSchema,
						// JSONSchemaParam nil => invalid.
					},
				}
				return r
			},
			wantErrText: "jsonSchemaParam is required when format.kind is jsonSchema",
		},
		{
			name: "outputFormat_unknown_kind",
			req: func() *spec.PostModelPresetRequest {
				r := base()
				badKind := inferenceSpec.OutputFormatKind("wat")
				_ = kindPtr(badKind)
				r.Body.OutputParam = &inferenceSpec.OutputParam{
					Format: &inferenceSpec.OutputFormat{
						Kind: badKind,
					},
				}
				return r
			},
			wantErrText: "unknown format.kind",
		},
		{
			name: "reasoning_unknown_type",
			req: func() *spec.PostModelPresetRequest {
				r := base()
				r.Body.Temperature = nil
				r.Body.Reasoning = &inferenceSpec.ReasoningParam{
					Type: inferenceSpec.ReasoningType("ghost"),
				}
				return r
			},
			wantErrText: "unknown type",
		},
		{
			name: "reasoning_hybrid_tokens_must_be_positive",
			req: func() *spec.PostModelPresetRequest {
				r := base()
				r.Body.Temperature = nil
				r.Body.Reasoning = &inferenceSpec.ReasoningParam{
					Type:   inferenceSpec.ReasoningTypeHybridWithTokens,
					Tokens: 0,
				}
				return r
			},
			wantErrText: "tokens must be >0",
		},
		{
			name: "reasoning_unknown_summary_style",
			req: func() *spec.PostModelPresetRequest {
				r := base()
				r.Body.Temperature = nil
				bad := inferenceSpec.ReasoningSummaryStyle("nope")
				r.Body.Reasoning = &inferenceSpec.ReasoningParam{
					Type:         inferenceSpec.ReasoningTypeHybridWithTokens,
					Tokens:       1,
					SummaryStyle: summaryPtr(bad),
				}
				return r
			},
			wantErrText: "unknown summaryStyle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := st.PostModelPreset(ctx, tt.req())
			if tt.wantErrText != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrText) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErrText, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected: %v", err)
			}
		})
	}
}
