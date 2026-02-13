package store

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

const (
	toolTag        = "ToolStore"
	toolPathPrefix = "/tools"
)

// InitToolStoreHandlers registers all endpoints for tool bundles and tools.
func InitToolStoreHandlers(api huma.API, store *ToolStore) {
	huma.Register(api, huma.Operation{
		OperationID: "put-tool-bundle",
		Method:      http.MethodPut,
		Path:        toolPathPrefix + "/bundles/{bundleID}",
		Summary:     "Create or replace a tool bundle",
		Tags:        []string{toolTag},
	}, store.PutToolBundle)

	huma.Register(api, huma.Operation{
		OperationID: "patch-tool-bundle",
		Method:      http.MethodPatch,
		Path:        toolPathPrefix + "/bundles/{bundleID}",
		Summary:     "Enable or disable a tool bundle",
		Tags:        []string{toolTag},
	}, store.PatchToolBundle)

	huma.Register(api, huma.Operation{
		OperationID: "delete-tool-bundle",
		Method:      http.MethodDelete,
		Path:        toolPathPrefix + "/bundles/{bundleID}",
		Summary:     "Soft-delete a tool bundle (if empty)",
		Tags:        []string{toolTag},
	}, store.DeleteToolBundle)

	huma.Register(api, huma.Operation{
		OperationID: "list-tool-bundles",
		Method:      http.MethodGet,
		Path:        toolPathPrefix + "/bundles",
		Summary:     "List tool bundles",
		Tags:        []string{toolTag},
	}, store.ListToolBundles)

	huma.Register(api, huma.Operation{
		OperationID: "put-tool",
		Method:      http.MethodPut,
		Path:        toolPathPrefix + "/bundles/{bundleID}/tools/{toolSlug}/version/{version}",
		Summary:     "Create a new tool version",
		Tags:        []string{toolTag},
	}, store.PutTool)

	huma.Register(api, huma.Operation{
		OperationID: "patch-tool",
		Method:      http.MethodPatch,
		Path:        toolPathPrefix + "/bundles/{bundleID}/tools/{toolSlug}/version/{version}",
		Summary:     "Enable or disable a tool version",
		Tags:        []string{toolTag},
	}, store.PatchTool)

	huma.Register(api, huma.Operation{
		OperationID: "delete-tool",
		Method:      http.MethodDelete,
		Path:        toolPathPrefix + "/bundles/{bundleID}/tools/{toolSlug}/version/{version}",
		Summary:     "Hard-delete a tool version",
		Tags:        []string{toolTag},
	}, store.DeleteTool)

	huma.Register(api, huma.Operation{
		OperationID: "get-tool",
		Method:      http.MethodGet,
		Path:        toolPathPrefix + "/bundles/{bundleID}/tools/{toolSlug}/version/{version}",
		Summary:     "Get a tool version",
		Tags:        []string{toolTag},
	}, store.GetTool)

	huma.Register(api, huma.Operation{
		OperationID: "list-tools",
		Method:      http.MethodGet,
		Path:        toolPathPrefix + "/tools",
		Summary:     "List all tools (global, with filters)",
		Tags:        []string{toolTag},
	}, store.ListTools)
}
