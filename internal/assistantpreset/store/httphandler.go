package store

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

const (
	tag        = "AssistantPresetStore"
	pathPrefix = "/assistant-presets"
)

func InitAssistantPresetStoreHandlers(api huma.API, store *AssistantPresetStore) {
	huma.Register(api, huma.Operation{
		OperationID: "put-assistant-preset-bundle",
		Method:      http.MethodPut,
		Path:        pathPrefix + "/bundles/{bundleID}",
		Summary:     "Create or replace an assistant preset bundle",
		Tags:        []string{tag},
	}, store.PutAssistantPresetBundle)

	huma.Register(api, huma.Operation{
		OperationID: "patch-assistant-preset-bundle",
		Method:      http.MethodPatch,
		Path:        pathPrefix + "/bundles/{bundleID}",
		Summary:     "Enable or disable an assistant preset bundle",
		Tags:        []string{tag},
	}, store.PatchAssistantPresetBundle)

	huma.Register(api, huma.Operation{
		OperationID: "delete-assistant-preset-bundle",
		Method:      http.MethodDelete,
		Path:        pathPrefix + "/bundles/{bundleID}",
		Summary:     "Soft-delete an assistant preset bundle (if empty)",
		Tags:        []string{tag},
	}, store.DeleteAssistantPresetBundle)

	huma.Register(api, huma.Operation{
		OperationID: "list-assistant-preset-bundles",
		Method:      http.MethodGet,
		Path:        pathPrefix + "/bundles",
		Summary:     "List assistant preset bundles",
		Tags:        []string{tag},
	}, store.ListAssistantPresetBundles)

	huma.Register(api, huma.Operation{
		OperationID: "put-assistant-preset",
		Method:      http.MethodPut,
		Path:        pathPrefix + "/bundles/{bundleID}/presets/{assistantPresetSlug}/version/{version}",
		Summary:     "Create a new assistant preset version",
		Tags:        []string{tag},
	}, store.PutAssistantPreset)

	huma.Register(api, huma.Operation{
		OperationID: "patch-assistant-preset",
		Method:      http.MethodPatch,
		Path:        pathPrefix + "/bundles/{bundleID}/presets/{assistantPresetSlug}/version/{version}",
		Summary:     "Enable or disable an assistant preset version",
		Tags:        []string{tag},
	}, store.PatchAssistantPreset)

	huma.Register(api, huma.Operation{
		OperationID: "delete-assistant-preset",
		Method:      http.MethodDelete,
		Path:        pathPrefix + "/bundles/{bundleID}/presets/{assistantPresetSlug}/version/{version}",
		Summary:     "Hard-delete an assistant preset version",
		Tags:        []string{tag},
	}, store.DeleteAssistantPreset)

	huma.Register(api, huma.Operation{
		OperationID: "get-assistant-preset",
		Method:      http.MethodGet,
		Path:        pathPrefix + "/bundles/{bundleID}/presets/{assistantPresetSlug}/version/{version}",
		Summary:     "Get an assistant preset version",
		Tags:        []string{tag},
	}, store.GetAssistantPreset)

	huma.Register(api, huma.Operation{
		OperationID: "list-assistant-presets",
		Method:      http.MethodGet,
		Path:        pathPrefix + "/presets",
		Summary:     "List assistant preset versions",
		Tags:        []string{tag},
	}, store.ListAssistantPresets)
}
