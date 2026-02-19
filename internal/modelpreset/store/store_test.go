package store_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	"github.com/flexigpt/flexigpt-app/internal/modelpreset/store"
	inferencegoSpec "github.com/flexigpt/inference-go/spec"
)

func TestPutProviderPreset(t *testing.T) {
	ctx := t.Context()
	s := newTestStore(t)

	// First successful insert - used by several scenarios later on.
	okBody := validProviderBody("openai2")

	tests := []struct {
		name        string
		req         *spec.PutProviderPresetRequest
		expectError error
		verify      func(t *testing.T)
	}{
		{
			name:        "NilRequest",
			req:         nil,
			expectError: spec.ErrInvalidDir,
		},
		{
			name: "NilBody",
			req: &spec.PutProviderPresetRequest{
				ProviderName: "openai2",
				Body:         nil,
			},
			expectError: spec.ErrInvalidDir,
		},

		{
			name: "HappyPath",
			req: &spec.PutProviderPresetRequest{
				ProviderName: "openai2",
				Body:         okBody,
			},
		},
		{
			name: "OverwriteProviderKeepsCreatedAtUpdatesModifiedAt",
			req: &spec.PutProviderPresetRequest{
				ProviderName: "openai2",
				Body: func() *spec.PutProviderPresetRequestBody {
					b := *okBody
					b.DisplayName = "OPEN-AI-TWO(renamed)"
					return &b
				}(),
			},
			verify: func(t *testing.T) {
				t.Helper()
				resp, err := s.ListProviderPresets(ctx, &spec.ListProviderPresetsRequest{
					Names: []inferencegoSpec.ProviderName{"openai2"},
				})
				if err != nil {
					t.Fatalf("list failed: %v", err)
				}
				got := resp.Body.Providers[0]
				if string(got.DisplayName) != "OPEN-AI-TWO(renamed)" {
					t.Errorf("DisplayName not updated")
				}
				if got.CreatedAt.After(got.ModifiedAt) {
					t.Errorf("CreatedAt must not be after ModifiedAt")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.PutProviderPreset(ctx, tc.req)
			if tc.expectError != nil {
				if err == nil || !errors.Is(err, tc.expectError) {
					t.Fatalf("expected %v, got %v", tc.expectError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.verify != nil {
				tc.verify(t)
			}
		})
	}
}

func TestPatchProviderPreset(t *testing.T) {
	ctx := t.Context()
	s := newTestStore(t)

	// Prepare provider with two model presets.
	createProvider(t, s, "provA", true)
	createModelPreset(t, s, "provA", "m1", true, "")
	createModelPreset(t, s, "provA", "m2", true, "")
	// Make m1 default to start with.
	_, _ = s.PatchProviderPreset(ctx, &spec.PatchProviderPresetRequest{
		ProviderName: "provA",
		Body: &spec.PatchProviderPresetRequestBody{
			DefaultModelPresetID: ptrMPID("m1"),
		},
	})

	tests := []struct {
		name        string
		req         *spec.PatchProviderPresetRequest
		expectError error
		verify      func(t *testing.T)
	}{
		{
			name:        "NilRequest",
			req:         nil,
			expectError: spec.ErrInvalidDir,
		},
		{
			name: "BothFieldsNil",
			req: &spec.PatchProviderPresetRequest{
				ProviderName: "provA",
				Body:         &spec.PatchProviderPresetRequestBody{},
			},
			expectError: spec.ErrInvalidDir,
		},
		{
			name: "UnknownProvider",
			req: &spec.PatchProviderPresetRequest{
				ProviderName: "ghost",
				Body:         &spec.PatchProviderPresetRequestBody{IsEnabled: boolPtr(false)},
			},
			expectError: spec.ErrProviderNotFound,
		},
		{
			name: "SetIsEnabledFalse",
			req: &spec.PatchProviderPresetRequest{
				ProviderName: "provA",
				Body:         &spec.PatchProviderPresetRequestBody{IsEnabled: boolPtr(false)},
			},
			verify: func(t *testing.T) {
				t.Helper()
				pp := fetchProvider(t, s, ctx, "provA")
				if pp.IsEnabled {
					t.Errorf("provider still enabled")
				}
			},
		},
		{
			name: "ChangeDefaultModelPresetID",
			req: &spec.PatchProviderPresetRequest{
				ProviderName: "provA",
				Body: &spec.PatchProviderPresetRequestBody{
					DefaultModelPresetID: ptrMPID("m2"),
				},
			},
			verify: func(t *testing.T) {
				t.Helper()
				pp := fetchProvider(t, s, ctx, "provA")
				if pp.DefaultModelPresetID != "m2" {
					t.Errorf("default not updated - got %q", pp.DefaultModelPresetID)
				}
			},
		},
		{
			name: "DefaultModelPresetNotFound",
			req: &spec.PatchProviderPresetRequest{
				ProviderName: "provA",
				Body: &spec.PatchProviderPresetRequestBody{
					DefaultModelPresetID: ptrMPID("missing"),
				},
			},
			expectError: spec.ErrModelPresetNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.PatchProviderPreset(ctx, tc.req)
			if tc.expectError != nil {
				if err == nil || !errors.Is(err, tc.expectError) {
					t.Fatalf("want %v got %v", tc.expectError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.verify != nil {
				tc.verify(t)
			}
		})
	}
}

func TestDeleteProviderPreset(t *testing.T) {
	ctx := t.Context()
	s := newTestStore(t)

	createProvider(t, s, "provDel", true)
	createModelPreset(t, s, "provDel", "mx", true, "")

	tests := []struct {
		name        string
		req         *spec.DeleteProviderPresetRequest
		expectError error
	}{
		{
			name:        "NilRequest",
			req:         nil,
			expectError: spec.ErrInvalidDir,
		},
		{
			name: "CannotDeleteNonEmptyProvider",
			req: &spec.DeleteProviderPresetRequest{
				ProviderName: "provDel",
			},
			expectError: nil, // function returns ordinary error string - sentinel not exported.
		},
		{
			name: "DeleteAfterRemovingModels",
			req: &spec.DeleteProviderPresetRequest{
				ProviderName: "provDel",
			},
		},
		{
			name:        "DeleteAgain",
			req:         &spec.DeleteProviderPresetRequest{ProviderName: "provDel"},
			expectError: spec.ErrProviderNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.DeleteProviderPreset(ctx, tc.req)
			if tc.expectError != nil {
				if err == nil || !errors.Is(err, tc.expectError) {
					t.Fatalf("want %v got %v", tc.expectError, err)
				}
				return
			}
			if tc.name == "CannotDeleteNonEmptyProvider" {
				if err == nil {
					t.Fatalf("expected error - provider still had model presets")
				}
				// Remove model before second delete scenario.
				_, _ = s.DeleteModelPreset(ctx, &spec.DeleteModelPresetRequest{
					ProviderName:  "provDel",
					ModelPresetID: "mx",
				})

				return

			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestPutModelPreset(t *testing.T) {
	ctx := t.Context()
	s := newTestStore(t)
	createProvider(t, s, "provM", true)
	temp := 0.1
	tests := []struct {
		name        string
		req         *spec.PutModelPresetRequest
		expectError error
	}{
		{
			name:        "NilRequest",
			req:         nil,
			expectError: spec.ErrInvalidDir,
		},
		{
			name: "NilBody",
			req: &spec.PutModelPresetRequest{
				ProviderName:  "provM",
				ModelPresetID: "m1",
				Body:          nil,
			},
			expectError: spec.ErrInvalidDir,
		},
		{
			name: "InvalidSlug",
			req: &spec.PutModelPresetRequest{
				ProviderName:  "provM",
				ModelPresetID: "m1",
				Body: &spec.PutModelPresetRequestBody{
					Name:        "model-one",
					DisplayName: "Model-1",
					Slug:        "white space",
					IsEnabled:   true,
					Temperature: &temp,
				},
			},
			expectError: errors.New("invalid tag"),
		},
		{
			name: "UnknownProvider",
			req: &spec.PutModelPresetRequest{
				ProviderName:  "ghost",
				ModelPresetID: "m1",
				Body: &spec.PutModelPresetRequestBody{
					Name:        "m-ghost",
					DisplayName: "Ghost",
					Slug:        "g",
					IsEnabled:   true,
					Temperature: &temp,
				},
			},
			expectError: spec.ErrProviderNotFound,
		},
		{
			name: "HappyPath",
			req: &spec.PutModelPresetRequest{
				ProviderName:  "provM",
				ModelPresetID: "m1",
				Body: &spec.PutModelPresetRequestBody{
					Name:        "model-one",
					DisplayName: "Model-1",
					Slug:        "m1",
					IsEnabled:   true,
					Temperature: &temp,
				},
			},
		},
		{
			name: "OverwriteExistingModel",
			req: &spec.PutModelPresetRequest{
				ProviderName:  "provM",
				ModelPresetID: "m1",
				Body: &spec.PutModelPresetRequestBody{
					Name:        "model-one",
					DisplayName: "Model-ONE-RENAMED",
					Slug:        "m1",
					IsEnabled:   false,
					Temperature: &temp,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.PutModelPreset(ctx, tc.req)
			if tc.expectError != nil {
				if err == nil || !strings.Contains(err.Error(), tc.expectError.Error()) {
					t.Fatalf("want %v got %v", tc.expectError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestPatchModelPreset(t *testing.T) {
	ctx := t.Context()
	s := newTestStore(t)
	createProvider(t, s, "provPM", true)
	createModelPreset(t, s, "provPM", "mA", true, "")

	tests := []struct {
		name        string
		req         *spec.PatchModelPresetRequest
		expectError error
		verify      func(t *testing.T)
	}{
		{
			name:        "NilRequest",
			req:         nil,
			expectError: spec.ErrInvalidDir,
		},
		{
			name: "UnknownProvider",
			req: &spec.PatchModelPresetRequest{
				ProviderName:  "ghost",
				ModelPresetID: "mA",
				Body:          &spec.PatchModelPresetRequestBody{IsEnabled: true},
			},
			expectError: spec.ErrProviderNotFound,
		},
		{
			name: "DisableModel",
			req: &spec.PatchModelPresetRequest{
				ProviderName:  "provPM",
				ModelPresetID: "mA",
				Body:          &spec.PatchModelPresetRequestBody{IsEnabled: false},
			},
			verify: func(t *testing.T) {
				t.Helper()
				pp := fetchProvider(t, s, ctx, "provPM")
				if pp.ModelPresets["mA"].IsEnabled {
					t.Errorf("model still enabled")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.PatchModelPreset(ctx, tc.req)
			if tc.expectError != nil {
				if err == nil || !errors.Is(err, tc.expectError) {
					t.Fatalf("want %v got %v", tc.expectError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.verify != nil {
				tc.verify(t)
			}
		})
	}
}

func TestDeleteModelPreset(t *testing.T) {
	ctx := t.Context()
	s := newTestStore(t)
	createProvider(t, s, "provDM", true)
	createModelPreset(t, s, "provDM", "toDie", true, "")

	tests := []struct {
		name        string
		req         *spec.DeleteModelPresetRequest
		expectError error
	}{
		{
			name:        "NilRequest",
			req:         nil,
			expectError: spec.ErrInvalidDir,
		},
		{
			name: "UnknownProvider",
			req: &spec.DeleteModelPresetRequest{
				ProviderName:  "ghost",
				ModelPresetID: "x",
			},
			expectError: spec.ErrProviderNotFound,
		},
		{
			name: "HappyPath",
			req: &spec.DeleteModelPresetRequest{
				ProviderName:  "provDM",
				ModelPresetID: "toDie",
			},
		},
		{
			name: "DeleteAgain",
			req: &spec.DeleteModelPresetRequest{
				ProviderName:  "provDM",
				ModelPresetID: "toDie",
			},
			expectError: spec.ErrModelPresetNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.DeleteModelPreset(ctx, tc.req)
			if tc.expectError != nil {
				if err == nil || !errors.Is(err, tc.expectError) {
					t.Fatalf("want %v got %v", tc.expectError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected: %v", err)
			}
		})
	}
}

func TestListProviderPresetsPagingAndFiltering(t *testing.T) {
	ctx := t.Context()
	s := newTestStore(t)

	createProvider(t, s, "p1", true)
	createProvider(t, s, "p2", false) // disabled
	createProvider(t, s, "p3", true)

	// Default request - disabled filtered out.
	resp, err := s.ListProviderPresets(ctx, &spec.ListProviderPresetsRequest{})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	providers1 := make([]spec.ProviderPreset, 0)
	for _, p := range resp.Body.Providers {
		if p.IsBuiltIn != true {
			providers1 = append(providers1, p)
		}
	}
	if len(providers1) != 2 {
		t.Fatalf("expected 2 enabled providers, got %d", len(providers1))
	}

	// Include disabled.
	resp, err = s.ListProviderPresets(ctx, &spec.ListProviderPresetsRequest{IncludeDisabled: true})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	providers2 := make([]spec.ProviderPreset, 0)
	for _, p := range resp.Body.Providers {
		if p.IsBuiltIn != true {
			providers2 = append(providers2, p)
		}
	}
	if len(providers2) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(providers2))
	}

	// Names filter.
	resp, err = s.ListProviderPresets(ctx, &spec.ListProviderPresetsRequest{
		Names: []inferencegoSpec.ProviderName{"p2"},
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(resp.Body.Providers) != 0 {
		t.Fatalf("disabled provider should be filtered when IncludeDisabled=false")
	}

	// Names + includeDisabled.
	resp, _ = s.ListProviderPresets(ctx, &spec.ListProviderPresetsRequest{
		Names:           []inferencegoSpec.ProviderName{"p2"},
		IncludeDisabled: true,
	})
	if len(resp.Body.Providers) != 1 || resp.Body.Providers[0].Name != "p2" {
		t.Fatalf("filter on names failed")
	}

	// Paging - pageSize 2.
	resp1, _ := s.ListProviderPresets(ctx, &spec.ListProviderPresetsRequest{
		PageSize:        2,
		IncludeDisabled: true,
	})
	if len(resp1.Body.Providers) != 2 || resp1.Body.NextPageToken == nil {
		t.Fatalf("expected first page size=2 with token")
	}
	// Second page using token.
	resp2, err := s.ListProviderPresets(ctx, &spec.ListProviderPresetsRequest{
		PageToken: *resp1.Body.NextPageToken,
	})
	if err != nil {
		t.Fatalf("second page error: %v", err)
	}

	providers3 := make([]spec.ProviderPreset, 0)
	for _, p := range resp2.Body.Providers {
		if p.IsBuiltIn != true {
			providers3 = append(providers3, p)
		}
	}

	if len(providers3) != 1 {
		t.Fatalf("expected remaining 1 provider")
	}

	// Sanity - decode token and ensure cursor matches.
	var tok spec.ProviderPageToken
	_ = jsonDecodeB64(*resp1.Body.NextPageToken, &tok)
	if tok.CursorSlug != resp1.Body.Providers[len(resp1.Body.Providers)-1].Name {
		t.Errorf("cursor mismatch")
	}
}

func TestBuiltInProviderReadOnlyAndToggle(t *testing.T) {
	ctx := t.Context()
	s := newTestStore(t) // Loads built-ins automatically.

	const (
		builtinProvider = inferencegoSpec.ProviderName("openai") // shipped built-in.
		builtinModelID  = spec.ModelPresetID("gpt4o")            // ditto model-preset.
	)

	// Sanity: the built-in provider must be present right after store creation.
	if !providerExists(t, s, ctx, builtinProvider) {
		t.Fatalf("built-in provider %q not found - fixture / embedding broken", builtinProvider)
	}

	t.Run("PutProviderPreset_disallowed", func(t *testing.T) {
		_, err := s.PutProviderPreset(ctx, &spec.PutProviderPresetRequest{
			ProviderName: builtinProvider,
			Body:         validProviderBody(string(builtinProvider)),
		})
		if !errors.Is(err, spec.ErrBuiltInReadOnly) {
			t.Fatalf("expected ErrBuiltInReadOnly, got %v", err)
		}
	})

	t.Run("DeleteProviderPreset_disallowed", func(t *testing.T) {
		_, err := s.DeleteProviderPreset(ctx, &spec.DeleteProviderPresetRequest{
			ProviderName: builtinProvider,
		})
		if !errors.Is(err, spec.ErrBuiltInReadOnly) {
			t.Fatalf("expected ErrBuiltInReadOnly, got %v", err)
		}
	})

	t.Run("PutModelPreset_disallowed", func(t *testing.T) {
		_, err := s.PutModelPreset(ctx, &spec.PutModelPresetRequest{
			ProviderName:  builtinProvider,
			ModelPresetID: "whatever",
			Body: &spec.PutModelPresetRequestBody{
				Name:        "foo",
				DisplayName: "Foo",
				Slug:        "foo",
				IsEnabled:   true,
			},
		})
		if !errors.Is(err, spec.ErrBuiltInReadOnly) {
			t.Fatalf("expected ErrBuiltInReadOnly, got %v", err)
		}
	})

	t.Run("Disable_and_enable_provider", func(t *testing.T) {
		// Disable.
		_, err := s.PatchProviderPreset(ctx, &spec.PatchProviderPresetRequest{
			ProviderName: builtinProvider,
			Body:         &spec.PatchProviderPresetRequestBody{IsEnabled: boolPtr(false)},
		})
		if err != nil {
			t.Fatalf("disable failed: %v", err)
		}
		if fetchProvider(t, s, ctx, string(builtinProvider)).IsEnabled {
			t.Fatalf("provider is still enabled after disable patch")
		}

		// Enable again.
		_, err = s.PatchProviderPreset(ctx, &spec.PatchProviderPresetRequest{
			ProviderName: builtinProvider,
			Body:         &spec.PatchProviderPresetRequestBody{IsEnabled: boolPtr(true)},
		})
		if err != nil {
			t.Fatalf("enable failed: %v", err)
		}
		if !fetchProvider(t, s, ctx, string(builtinProvider)).IsEnabled {
			t.Fatalf("provider is still disabled after enable patch")
		}
	})

	t.Run("Disable_model_preset_inside_built_in_provider", func(t *testing.T) {
		_, err := s.PatchModelPreset(ctx, &spec.PatchModelPresetRequest{
			ProviderName:  builtinProvider,
			ModelPresetID: builtinModelID,
			Body:          &spec.PatchModelPresetRequestBody{IsEnabled: false},
		})
		if err != nil {
			t.Fatalf("patch builtin model failed: %v", err)
		}

		pp := fetchProvider(t, s, ctx, string(builtinProvider))
		if mp, ok := pp.ModelPresets[builtinModelID]; !ok || mp.IsEnabled {
			t.Fatalf("model-preset not disabled inside provider map")
		}
	})
}

// providerExists is a lightweight presence check.
func providerExists(t *testing.T, s *store.ModelPresetStore, ctx context.Context,
	name inferencegoSpec.ProviderName,
) bool {
	t.Helper()
	resp, err := s.ListProviderPresets(ctx, &spec.ListProviderPresetsRequest{
		Names:           []inferencegoSpec.ProviderName{name},
		IncludeDisabled: true,
	})
	if err != nil {
		t.Fatalf("ListProviderPresets failed: %v", err)
	}
	return len(resp.Body.Providers) == 1
}

// newTestStore creates a temporary on-disk store for every test.
func newTestStore(t *testing.T) *store.ModelPresetStore {
	t.Helper()
	s, err := store.NewModelPresetStore(t.TempDir())
	if err != nil {
		t.Fatalf("store init failed: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})
	return s
}

// createProvider inserts a provider quickly or fails the test.
func createProvider(t *testing.T, s *store.ModelPresetStore, name string, enabled bool) {
	t.Helper()
	body := validProviderBody(name)
	body.IsEnabled = enabled
	_, err := s.PutProviderPreset(t.Context(), &spec.PutProviderPresetRequest{
		ProviderName: inferencegoSpec.ProviderName(name),
		Body:         body,
	})
	if err != nil {
		t.Fatalf("createProvider(%s) failed: %v", name, err)
	}
}

// createModelPreset adds a model to an existing provider.
func createModelPreset(t *testing.T, s *store.ModelPresetStore,
	prov, id string, enabled bool, slug string,
) {
	t.Helper()
	if slug == "" {
		slug = id
	}
	temp := 0.1
	_, err := s.PutModelPreset(t.Context(), &spec.PutModelPresetRequest{
		ProviderName:  inferencegoSpec.ProviderName(prov),
		ModelPresetID: spec.ModelPresetID(id),
		Body: &spec.PutModelPresetRequestBody{
			Name:        spec.ModelName(id),
			DisplayName: spec.ModelDisplayName(strings.ToUpper(id)),
			Slug:        spec.ModelSlug(slug),
			IsEnabled:   enabled,
			Temperature: &temp,
		},
	})
	if err != nil {
		t.Fatalf("createModelPreset(%s/%s) failed: %v", prov, id, err)
	}
}

// validProviderBody returns a minimal but valid provider preset body.
func validProviderBody(name string) *spec.PutProviderPresetRequestBody {
	return &spec.PutProviderPresetRequestBody{
		DisplayName:              spec.ProviderDisplayName(strings.ToUpper(name)),
		SDKType:                  inferencegoSpec.ProviderSDKTypeOpenAIChatCompletions,
		IsEnabled:                true,
		Origin:                   "https://api." + name + ".example.com",
		ChatCompletionPathPrefix: spec.DefaultOpenAIChatCompletionsPrefix,
		APIKeyHeaderKey:          spec.DefaultAuthorizationHeaderKey,
		DefaultHeaders:           spec.OpenAIChatCompletionsDefaultHeaders,
	}
}

// fetchProvider pulls a single provider by name, fails the test if missing.
func fetchProvider(t *testing.T, s *store.ModelPresetStore, ctx context.Context,
	name string,
) spec.ProviderPreset {
	t.Helper()
	resp, err := s.ListProviderPresets(ctx, &spec.ListProviderPresetsRequest{
		Names:           []inferencegoSpec.ProviderName{inferencegoSpec.ProviderName(name)},
		IncludeDisabled: true,
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(resp.Body.Providers) != 1 {
		t.Fatalf("provider %s not found", name)
	}
	return resp.Body.Providers[0]
}

// Utility helpers.
func boolPtr(b bool) *bool                  { return &b }
func ptrMPID(id string) *spec.ModelPresetID { mpid := spec.ModelPresetID(id); return &mpid }
func jsonDecodeB64(in string, out any) error {
	raw, _ := base64.StdEncoding.DecodeString(in)
	return json.Unmarshal(raw, out)
}
