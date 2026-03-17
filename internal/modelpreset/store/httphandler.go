package store

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// We keep a few constants for organization.
const (
	tag           = "ModelPresetStore"
	topPathPrefix = "/modelpresets"
	pathPrefix    = topPathPrefix + "/providers"
)

// InitModelPresetStoreHandlers registers all endpoints related to settings.
func InitModelPresetStoreHandlers(api huma.API, modelPresetStoreAPI *ModelPresetStore) {
	huma.Register(api, huma.Operation{
		OperationID: "patch-default-provider",
		Method:      http.MethodPatch,
		Path:        topPathPrefix + "/defaultprovider",
		Summary:     "Set a default provider",
		Tags:        []string{tag},
	}, modelPresetStoreAPI.PatchDefaultProvider)

	huma.Register(api, huma.Operation{
		OperationID: "get-default-provider",
		Method:      http.MethodGet,
		Path:        topPathPrefix + "/defaultprovider",
		Summary:     "Get the default provider",
		Tags:        []string{tag},
	}, modelPresetStoreAPI.GetDefaultProvider)

	huma.Register(api, huma.Operation{
		OperationID: "post-provider-preset",
		Method:      http.MethodPost,
		Path:        pathPrefix + "/{providerName}",
		Summary:     "Create a new provider preset",
		Tags:        []string{tag},
	}, modelPresetStoreAPI.PostProviderPreset)

	huma.Register(api, huma.Operation{
		OperationID: "patch-provider-preset",
		Method:      http.MethodPatch,
		Path:        pathPrefix + "/{providerName}",
		Summary:     "Patch an existing provider preset",
		Tags:        []string{tag},
	}, modelPresetStoreAPI.PatchProviderPreset)

	huma.Register(api, huma.Operation{
		OperationID: "delete-provider-preset",
		Method:      http.MethodDelete,
		Path:        pathPrefix + "/{providerName}",
		Summary:     "Delete all model presets for a provider",
		Tags:        []string{tag},
	}, modelPresetStoreAPI.DeleteProviderPreset)

	huma.Register(api, huma.Operation{
		OperationID: "post-model-preset",
		Method:      http.MethodPost,
		Path:        pathPrefix + "/{providerName}/models/{modelPresetID}",
		Summary:     "Create a new model preset for a given provider",
		Tags:        []string{tag},
	}, modelPresetStoreAPI.PostModelPreset)

	huma.Register(api, huma.Operation{
		OperationID: "patch-model-preset",
		Method:      http.MethodPatch,
		Path:        pathPrefix + "/{providerName}/models/{modelPresetID}",
		Summary:     "Patch properties of a single model preset",
		Tags:        []string{tag},
	}, modelPresetStoreAPI.PatchModelPreset)

	huma.Register(api, huma.Operation{
		OperationID: "delete-model-preset",
		Method:      http.MethodDelete,
		Path:        pathPrefix + "/{providerName}/models/{modelPresetID}",
		Summary:     "Delete a single model preset for a given provider",
		Tags:        []string{tag},
	}, modelPresetStoreAPI.DeleteModelPreset)

	huma.Register(api, huma.Operation{
		OperationID: "get-model-preset",
		Method:      http.MethodGet,
		Path:        pathPrefix + "/{providerName}/models/{modelPresetID}",
		Summary:     "Get a single model preset for a given provider",
		Tags:        []string{tag},
	}, modelPresetStoreAPI.GetModelPreset)

	huma.Register(api, huma.Operation{
		OperationID: "list-all-provider-presets",
		Method:      http.MethodGet,
		Path:        pathPrefix,
		Summary:     "List all presets",
		Description: "List the entire presets object from the store",
		Tags:        []string{tag},
	}, modelPresetStoreAPI.ListProviderPresets)
}
