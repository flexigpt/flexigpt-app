package store

import (
	"path/filepath"
	"strconv"
	"testing"
	"time"

	modelpresetSpec "github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	inferencegoSpec "github.com/flexigpt/inference-go/spec"
)

func TestModelPresetStore_New_CreatesFiles_AndDefaultProviderFallback(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	st := newStoreAtDir(t, dir)
	ctx := t.Context()

	// User JSON file should exist.
	mustFileExists(t, filepath.Join(dir, modelpresetSpec.ModelPresetsFile))
	// Built-in overlay db should exist.
	mustFileExists(t, filepath.Join(dir, modelpresetSpec.ModelPresetsBuiltInOverlayDBFileName))

	// Default provider should be non-empty and should match builtin fallback logic.
	got, err := st.GetDefaultProvider(ctx, &modelpresetSpec.GetDefaultProviderRequest{})
	if err != nil {
		t.Fatalf("GetDefaultProvider: %v", err)
	}
	if got.Body.DefaultProvider == "" {
		t.Fatalf("expected non-empty default provider")
	}

	// Also ensure at least one provider exists (built-ins).
	resp, err := st.ListProviderPresets(ctx, &modelpresetSpec.ListProviderPresetsRequest{IncludeDisabled: true})
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
	userProvider := inferencegoSpec.ProviderName("user-prov-default")
	putUserProvider(t, st, userProvider, true)

	builtinName, _ := anyBuiltInProviderFromStore(t, st)

	tests := []struct {
		name        string
		req         *modelpresetSpec.PatchDefaultProviderRequest
		wantErrIs   error
		wantErrText string
	}{
		{
			name:      "nil_request",
			req:       nil,
			wantErrIs: modelpresetSpec.ErrProviderNotFound,
		},
		{
			name: "nil_body",
			req: &modelpresetSpec.PatchDefaultProviderRequest{
				Body: nil,
			},
			wantErrIs: modelpresetSpec.ErrProviderNotFound,
		},
		{
			name: "empty_provider",
			req: &modelpresetSpec.PatchDefaultProviderRequest{
				Body: &modelpresetSpec.PatchDefaultProviderRequestBody{DefaultProvider: ""},
			},
			wantErrIs: modelpresetSpec.ErrProviderNotFound,
		},
		{
			name: "unknown_provider",
			req: &modelpresetSpec.PatchDefaultProviderRequest{
				Body: &modelpresetSpec.PatchDefaultProviderRequestBody{DefaultProvider: "ghost"},
			},
			wantErrIs: modelpresetSpec.ErrProviderNotFound,
		},
		{
			name: "set_to_builtin_provider",
			req: &modelpresetSpec.PatchDefaultProviderRequest{
				Body: &modelpresetSpec.PatchDefaultProviderRequestBody{DefaultProvider: builtinName},
			},
		},
		{
			name: "set_to_user_provider",
			req: &modelpresetSpec.PatchDefaultProviderRequest{
				Body: &modelpresetSpec.PatchDefaultProviderRequestBody{DefaultProvider: userProvider},
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

	got, err := st2.GetDefaultProvider(ctx, &modelpresetSpec.GetDefaultProviderRequest{})
	if err != nil {
		t.Fatalf("GetDefaultProvider(after reopen): %v", err)
	}
	if got.Body.DefaultProvider != userProvider {
		t.Fatalf("default provider not persisted: got=%q want=%q", got.Body.DefaultProvider, userProvider)
	}
}

func TestModelPresetStore_PutProviderPreset_TableDriven(t *testing.T) {
	st := newStore(t)
	ctx := t.Context()

	builtinName, _ := anyBuiltInProviderFromStore(t, st)

	okBody := &modelpresetSpec.PutProviderPresetRequestBody{
		DisplayName:              "OK",
		SDKType:                  inferencegoSpec.ProviderSDKTypeOpenAIChatCompletions,
		IsEnabled:                true,
		Origin:                   "https://example.test",
		ChatCompletionPathPrefix: modelpresetSpec.DefaultOpenAIChatCompletionsPrefix,
		APIKeyHeaderKey:          modelpresetSpec.DefaultAuthorizationHeaderKey,
		DefaultHeaders:           map[string]string{"content-type": "application/json"},
	}

	tests := []struct {
		name        string
		req         *modelpresetSpec.PutProviderPresetRequest
		wantErrIs   error
		wantErrText string
		verify      func(t *testing.T)
	}{
		{
			name:      "nil_request",
			req:       nil,
			wantErrIs: modelpresetSpec.ErrInvalidDir,
		},
		{
			name: "nil_body",
			req: &modelpresetSpec.PutProviderPresetRequest{
				ProviderName: "user-prov-x",
				Body:         nil,
			},
			wantErrIs: modelpresetSpec.ErrInvalidDir,
		},
		{
			name: "empty_providerName",
			req: &modelpresetSpec.PutProviderPresetRequest{
				ProviderName: "",
				Body:         okBody,
			},
			wantErrIs: modelpresetSpec.ErrInvalidDir,
		},
		{
			name: "built_in_readonly",
			req: &modelpresetSpec.PutProviderPresetRequest{
				ProviderName: builtinName,
				Body:         okBody,
			},
			wantErrIs: modelpresetSpec.ErrBuiltInReadOnly,
		},
		{
			name: "validation_error_empty_displayName",
			req: &modelpresetSpec.PutProviderPresetRequest{
				ProviderName: "user-prov-bad1",
				Body: func() *modelpresetSpec.PutProviderPresetRequestBody {
					b := *okBody
					b.DisplayName = ""
					return &b
				}(),
			},
			wantErrText: "displayName is empty",
		},
		{
			name: "validation_error_empty_origin",
			req: &modelpresetSpec.PutProviderPresetRequest{
				ProviderName: "user-prov-bad2",
				Body: func() *modelpresetSpec.PutProviderPresetRequestBody {
					b := *okBody
					b.Origin = ""
					return &b
				}(),
			},
			wantErrText: "origin is empty",
		},
		{
			name: "validation_error_empty_chatCompletionPathPrefix",
			req: &modelpresetSpec.PutProviderPresetRequest{
				ProviderName: "user-prov-bad3",
				Body: func() *modelpresetSpec.PutProviderPresetRequestBody {
					b := *okBody
					b.ChatCompletionPathPrefix = ""
					return &b
				}(),
			},
			wantErrText: "chatCompletionPathPrefix is empty",
		},
		{
			name: "happy_create",
			req: &modelpresetSpec.PutProviderPresetRequest{
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
			name: "overwrite_keeps_createdAt_updates_modifiedAt",
			req: &modelpresetSpec.PutProviderPresetRequest{
				ProviderName: "user-prov-ok",
				Body: func() *modelpresetSpec.PutProviderPresetRequestBody {
					b := *okBody
					b.DisplayName = "RENAMED"
					return &b
				}(),
			},
			verify: func(t *testing.T) {
				t.Helper()
				pp := getProviderByName(t, st, ctx, "user-prov-ok", true)
				if string(pp.DisplayName) != "RENAMED" {
					t.Fatalf("expected DisplayName to be updated, got %q", pp.DisplayName)
				}
			},
		},
	}

	// Seed initial provider for overwrite case.
	_, _ = st.PutProviderPreset(ctx, &modelpresetSpec.PutProviderPresetRequest{
		ProviderName: "user-prov-ok",
		Body:         okBody,
	})
	before := getProviderByName(t, st, ctx, "user-prov-ok", true)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := st.PutProviderPreset(ctx, tt.req)
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

	after := getProviderByName(t, st, ctx, "user-prov-ok", true)
	if !after.CreatedAt.Equal(before.CreatedAt) {
		t.Fatalf("CreatedAt should be preserved on overwrite: before=%v after=%v", before.CreatedAt, after.CreatedAt)
	}
	if !after.ModifiedAt.After(before.ModifiedAt) && !after.ModifiedAt.Equal(before.ModifiedAt) {
		// Time can be equal on very fast filesystems; accept equal but not older.
		t.Fatalf("ModifiedAt must not go backwards: before=%v after=%v", before.ModifiedAt, after.ModifiedAt)
	}
}

func TestModelPresetStore_PatchProviderPreset_UserProvider(t *testing.T) {
	st := newStore(t)
	ctx := t.Context()

	prov := inferencegoSpec.ProviderName("user-prov-patch")
	putUserProvider(t, st, prov, true)
	putUserModelPreset(t, ctx, st, prov, "m1", true)
	putUserModelPreset(t, ctx, st, prov, "m2", true)

	// Set default to m1.
	_, err := st.PatchProviderPreset(ctx, &modelpresetSpec.PatchProviderPresetRequest{
		ProviderName: prov,
		Body: &modelpresetSpec.PatchProviderPresetRequestBody{
			DefaultModelPresetID: mpidPtr("m1"),
		},
	})
	if err != nil {
		t.Fatalf("initial PatchProviderPreset(default=m1): %v", err)
	}

	tests := []struct {
		name        string
		req         *modelpresetSpec.PatchProviderPresetRequest
		wantErrIs   error
		wantErrText string
		verify      func(t *testing.T)
	}{
		{
			name:      "nil_request",
			req:       nil,
			wantErrIs: modelpresetSpec.ErrInvalidDir,
		},
		{
			name: "both_fields_nil",
			req: &modelpresetSpec.PatchProviderPresetRequest{
				ProviderName: prov,
				Body:         &modelpresetSpec.PatchProviderPresetRequestBody{},
			},
			wantErrIs: modelpresetSpec.ErrInvalidDir,
		},
		{
			name: "unknown_provider",
			req: &modelpresetSpec.PatchProviderPresetRequest{
				ProviderName: "ghost",
				Body:         &modelpresetSpec.PatchProviderPresetRequestBody{IsEnabled: boolPtr(false)},
			},
			wantErrIs: modelpresetSpec.ErrProviderNotFound,
		},
		{
			name: "invalid_default_model_id_tag",
			req: &modelpresetSpec.PatchProviderPresetRequest{
				ProviderName: prov,
				Body: &modelpresetSpec.PatchProviderPresetRequestBody{
					DefaultModelPresetID: mpidPtr("white space"),
				},
			},
			wantErrText: "invalid tag",
		},
		{
			name: "disable_provider",
			req: &modelpresetSpec.PatchProviderPresetRequest{
				ProviderName: prov,
				Body:         &modelpresetSpec.PatchProviderPresetRequestBody{IsEnabled: boolPtr(false)},
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
			req: &modelpresetSpec.PatchProviderPresetRequest{
				ProviderName: prov,
				Body: &modelpresetSpec.PatchProviderPresetRequestBody{
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
			req: &modelpresetSpec.PatchProviderPresetRequest{
				ProviderName: prov,
				Body: &modelpresetSpec.PatchProviderPresetRequestBody{
					DefaultModelPresetID: mpidPtr("missing"),
				},
			},
			wantErrIs: modelpresetSpec.ErrModelPresetNotFound,
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
		_, err := st.PatchProviderPreset(ctx, &modelpresetSpec.PatchProviderPresetRequest{
			ProviderName: pn,
			Body:         &modelpresetSpec.PatchProviderPresetRequestBody{IsEnabled: boolPtr(newEnabled)},
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

		_, err := st.PatchProviderPreset(ctx, &modelpresetSpec.PatchProviderPresetRequest{
			ProviderName: pn2,
			Body: &modelpresetSpec.PatchProviderPresetRequestBody{
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

func TestModelPresetStore_DeleteProviderPreset_TableDriven(t *testing.T) {
	st := newStore(t)
	ctx := t.Context()

	builtinName, _ := anyBuiltInProviderFromStore(t, st)

	userProv := inferencegoSpec.ProviderName("user-prov-del")
	putUserProvider(t, st, userProv, true)
	putUserModelPreset(t, ctx, st, userProv, "m1", true)

	tests := []struct {
		name        string
		req         *modelpresetSpec.DeleteProviderPresetRequest
		wantErrIs   error
		wantErrText string
		before      func()
	}{
		{
			name:      "nil_request",
			req:       nil,
			wantErrIs: modelpresetSpec.ErrInvalidDir,
		},
		{
			name: "built_in_readonly",
			req: &modelpresetSpec.DeleteProviderPresetRequest{
				ProviderName: builtinName,
			},
			wantErrIs: modelpresetSpec.ErrBuiltInReadOnly,
		},
		{
			name: "cannot_delete_non_empty_provider",
			req: &modelpresetSpec.DeleteProviderPresetRequest{
				ProviderName: userProv,
			},
			wantErrText: "not empty",
		},
		{
			name: "delete_after_removing_models",
			before: func() {
				_, _ = st.DeleteModelPreset(ctx, &modelpresetSpec.DeleteModelPresetRequest{
					ProviderName:  userProv,
					ModelPresetID: "m1",
				})
			},
			req: &modelpresetSpec.DeleteProviderPresetRequest{
				ProviderName: userProv,
			},
		},
		{
			name: "delete_again_not_found",
			req: &modelpresetSpec.DeleteProviderPresetRequest{
				ProviderName: userProv,
			},
			wantErrIs: modelpresetSpec.ErrProviderNotFound,
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

func TestModelPresetStore_ModelPreset_UserCRUD_TableDriven(t *testing.T) {
	st := newStore(t)
	ctx := t.Context()

	userProv := inferencegoSpec.ProviderName("user-prov-models")
	putUserProvider(t, st, userProv, true)

	temp := 0.1
	builtinName, _ := anyBuiltInProviderFromStore(t, st)

	t.Run("PutModelPreset_validation_and_errors", func(t *testing.T) {
		tests := []struct {
			name        string
			req         *modelpresetSpec.PutModelPresetRequest
			wantErrIs   error
			wantErrText string
		}{
			{
				name:      "nil_request",
				req:       nil,
				wantErrIs: modelpresetSpec.ErrInvalidDir,
			},
			{
				name: "nil_body",
				req: &modelpresetSpec.PutModelPresetRequest{
					ProviderName:  userProv,
					ModelPresetID: "m1",
					Body:          nil,
				},
				wantErrIs: modelpresetSpec.ErrInvalidDir,
			},
			{
				name: "invalid_modelPresetID_tag",
				req: &modelpresetSpec.PutModelPresetRequest{
					ProviderName:  userProv,
					ModelPresetID: "white space",
					Body: &modelpresetSpec.PutModelPresetRequestBody{
						Name:        "n",
						Slug:        "m1",
						DisplayName: "x",
						IsEnabled:   true,
						Temperature: &temp,
					},
				},
				wantErrText: "invalid tag",
			},
			{
				name: "invalid_slug_tag",
				req: &modelpresetSpec.PutModelPresetRequest{
					ProviderName:  userProv,
					ModelPresetID: "m1",
					Body: &modelpresetSpec.PutModelPresetRequestBody{
						Name:        "n",
						Slug:        "white space",
						DisplayName: "x",
						IsEnabled:   true,
						Temperature: &temp,
					},
				},
				wantErrText: "invalid tag",
			},
			{
				name: "missing_temp_and_reasoning",
				req: &modelpresetSpec.PutModelPresetRequest{
					ProviderName:  userProv,
					ModelPresetID: "m1",
					Body: &modelpresetSpec.PutModelPresetRequestBody{
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
				req: &modelpresetSpec.PutModelPresetRequest{
					ProviderName:  userProv,
					ModelPresetID: "m1",
					Body: &modelpresetSpec.PutModelPresetRequestBody{
						Name:        "n",
						Slug:        "m1",
						DisplayName: "x",
						IsEnabled:   true,
						Temperature: &temp,
						StopSequences: []string{
							"a", "b", "c", "d", "e",
						},
					},
				},
				wantErrText: "too many stop sequences",
			},
			{
				name: "unknown_provider",
				req: &modelpresetSpec.PutModelPresetRequest{
					ProviderName:  "ghost",
					ModelPresetID: "m1",
					Body: &modelpresetSpec.PutModelPresetRequestBody{
						Name:        "n",
						Slug:        "m1",
						DisplayName: "x",
						IsEnabled:   true,
						Temperature: &temp,
					},
				},
				wantErrIs: modelpresetSpec.ErrProviderNotFound,
			},
			{
				name: "built_in_readonly",
				req: &modelpresetSpec.PutModelPresetRequest{
					ProviderName:  builtinName,
					ModelPresetID: "m1",
					Body: &modelpresetSpec.PutModelPresetRequestBody{
						Name:        "n",
						Slug:        "m1",
						DisplayName: "x",
						IsEnabled:   true,
						Temperature: &temp,
					},
				},
				wantErrIs: modelpresetSpec.ErrBuiltInReadOnly,
			},
			{
				name: "happy_put",
				req: &modelpresetSpec.PutModelPresetRequest{
					ProviderName:  userProv,
					ModelPresetID: "m1",
					Body: &modelpresetSpec.PutModelPresetRequestBody{
						Name:        "model-one",
						Slug:        "m1",
						DisplayName: "Model One",
						IsEnabled:   true,
						Temperature: &temp,
					},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := st.PutModelPreset(ctx, tt.req)
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

	t.Run("OverwriteModel_keeps_createdAt", func(t *testing.T) {
		// Ensure baseline exists.
		putUserModelPreset(t, ctx, st, userProv, "m2", true)
		ppBefore := getProviderByName(t, st, ctx, userProv, true)
		before := ppBefore.ModelPresets["m2"]

		time.Sleep(2 * time.Millisecond)

		// Overwrite m2.
		_, err := st.PutModelPreset(ctx, &modelpresetSpec.PutModelPresetRequest{
			ProviderName:  userProv,
			ModelPresetID: "m2",
			Body: &modelpresetSpec.PutModelPresetRequestBody{
				Name:        "model-two",
				Slug:        "m2",
				DisplayName: "Model Two Renamed",
				IsEnabled:   false,
				Temperature: &temp,
			},
		})
		if err != nil {
			t.Fatalf("PutModelPreset(overwrite): %v", err)
		}

		ppAfter := getProviderByName(t, st, ctx, userProv, true)
		after := ppAfter.ModelPresets["m2"]

		if !after.CreatedAt.Equal(before.CreatedAt) {
			t.Fatalf("CreatedAt not preserved: before=%v after=%v", before.CreatedAt, after.CreatedAt)
		}
		if after.DisplayName != "Model Two Renamed" {
			t.Fatalf("DisplayName not updated: got=%q", after.DisplayName)
		}
	})

	t.Run("PatchModelPreset_user_branch", func(t *testing.T) {
		putUserModelPreset(t, ctx, st, userProv, "m3", true)

		// Unknown provider.
		_, err := st.PatchModelPreset(ctx, &modelpresetSpec.PatchModelPresetRequest{
			ProviderName:  "ghost",
			ModelPresetID: "m3",
			Body:          &modelpresetSpec.PatchModelPresetRequestBody{IsEnabled: false},
		})
		wantErrIs(t, err, modelpresetSpec.ErrProviderNotFound)

		// Unknown model.
		_, err = st.PatchModelPreset(ctx, &modelpresetSpec.PatchModelPresetRequest{
			ProviderName:  userProv,
			ModelPresetID: "ghost",
			Body:          &modelpresetSpec.PatchModelPresetRequestBody{IsEnabled: false},
		})
		wantErrIs(t, err, modelpresetSpec.ErrModelPresetNotFound)

		// Happy patch.
		_, err = st.PatchModelPreset(ctx, &modelpresetSpec.PatchModelPresetRequest{
			ProviderName:  userProv,
			ModelPresetID: "m3",
			Body:          &modelpresetSpec.PatchModelPresetRequestBody{IsEnabled: false},
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
		putUserModelPreset(t, ctx, st, userProv, "m-default", true)

		// Set provider default to m-default.
		_, err := st.PatchProviderPreset(ctx, &modelpresetSpec.PatchProviderPresetRequest{
			ProviderName: userProv,
			Body: &modelpresetSpec.PatchProviderPresetRequestBody{
				DefaultModelPresetID: mpidPtr("m-default"),
			},
		})
		if err != nil {
			t.Fatalf("PatchProviderPreset(set default): %v", err)
		}

		_, err = st.DeleteModelPreset(ctx, &modelpresetSpec.DeleteModelPresetRequest{
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

	_, err := st.PatchModelPreset(ctx, &modelpresetSpec.PatchModelPresetRequest{
		ProviderName:  pn,
		ModelPresetID: mid,
		Body:          &modelpresetSpec.PatchModelPresetRequestBody{IsEnabled: !mp.IsEnabled},
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

	p1 := inferencegoSpec.ProviderName("user-p1")
	p2 := inferencegoSpec.ProviderName("user-p2")
	p3 := inferencegoSpec.ProviderName("user-p3")

	putUserProvider(t, st, p1, true)
	putUserProvider(t, st, p2, false)
	putUserProvider(t, st, p3, true)

	names := []inferencegoSpec.ProviderName{p1, p2, p3}

	t.Run("disabled_filtered_out_by_default", func(t *testing.T) {
		resp, err := st.ListProviderPresets(ctx, &modelpresetSpec.ListProviderPresetsRequest{
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
		resp, err := st.ListProviderPresets(ctx, &modelpresetSpec.ListProviderPresetsRequest{
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
		seen := map[inferencegoSpec.ProviderName]bool{}

		req := &modelpresetSpec.ListProviderPresetsRequest{
			Names:           names,
			IncludeDisabled: true,
			PageSize:        1,
		}

		var token *string
		for i := range 10 {
			if token != nil {
				req = &modelpresetSpec.ListProviderPresetsRequest{PageToken: *token}
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
		_, err := st.ListProviderPresets(ctx, &modelpresetSpec.ListProviderPresetsRequest{
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
	p1 := inferencegoSpec.ProviderName("user-tok-1")
	p2 := inferencegoSpec.ProviderName("user-tok-2") // disabled
	p3 := inferencegoSpec.ProviderName("user-tok-3")

	putUserProvider(t, st, p1, true)
	putUserProvider(t, st, p2, false)
	putUserProvider(t, st, p3, true)

	names := []inferencegoSpec.ProviderName{p1, p2, p3}

	// First call to get token.
	resp1, err := st.ListProviderPresets(ctx, &modelpresetSpec.ListProviderPresetsRequest{
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

	total := modelpresetSpec.DefaultPageSize + 1
	names := make([]inferencegoSpec.ProviderName, 0, total)

	for i := range total {
		pn := inferencegoSpec.ProviderName("user-many-" + strconv.Itoa(i))
		putUserProvider(t, st, pn, true)
		names = append(names, pn)
	}

	// Ask for an invalid page size (> DefaultPageSize), expecting clamp back to DefaultPageSize.
	resp, err := st.ListProviderPresets(ctx, &modelpresetSpec.ListProviderPresetsRequest{
		Names:           names,
		IncludeDisabled: true,
		PageSize:        modelpresetSpec.MaxPageSize + 1000,
	})
	if err != nil {
		t.Fatalf("ListProviderPresets: %v", err)
	}
	if len(resp.Body.Providers) != modelpresetSpec.DefaultPageSize {
		t.Fatalf("expected clamped page size %d, got %d", modelpresetSpec.DefaultPageSize, len(resp.Body.Providers))
	}
	if resp.Body.NextPageToken == nil {
		t.Fatalf("expected next token for %d providers", total)
	}

	tok := decodeProviderPageToken(t, *resp.Body.NextPageToken)
	if tok.PageSize != modelpresetSpec.DefaultPageSize {
		t.Fatalf("expected token page size clamped to %d, got %d", modelpresetSpec.DefaultPageSize, tok.PageSize)
	}
}
