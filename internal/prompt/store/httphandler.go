package store

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

const (
	tag        = "PromptTemplateStore"
	pathPrefix = "/prompts"
)

// InitPromptTemplateStoreHandlers registers all endpoints for prompt bundles and templates.
func InitPromptTemplateStoreHandlers(api huma.API, store *PromptTemplateStore) {
	huma.Register(api, huma.Operation{
		OperationID: "put-prompt-bundle",
		Method:      http.MethodPut,
		Path:        pathPrefix + "/bundles/{bundleID}",
		Summary:     "Create or replace a prompt bundle",
		Tags:        []string{tag},
	}, store.PutPromptBundle)

	huma.Register(api, huma.Operation{
		OperationID: "patch-prompt-bundle",
		Method:      http.MethodPatch,
		Path:        pathPrefix + "/bundles/{bundleID}",
		Summary:     "Enable or disable a prompt bundle",
		Tags:        []string{tag},
	}, store.PatchPromptBundle)

	huma.Register(api, huma.Operation{
		OperationID: "delete-prompt-bundle",
		Method:      http.MethodDelete,
		Path:        pathPrefix + "/bundles/{bundleID}",
		Summary:     "Soft-delete a prompt bundle (if empty)",
		Tags:        []string{tag},
	}, store.DeletePromptBundle)

	huma.Register(api, huma.Operation{
		OperationID: "list-prompt-bundles",
		Method:      http.MethodGet,
		Path:        pathPrefix + "/bundles",
		Summary:     "List prompt bundles",
		Tags:        []string{tag},
	}, store.ListPromptBundles)

	huma.Register(api, huma.Operation{
		OperationID: "put-prompt-template",
		Method:      http.MethodPut,
		Path:        pathPrefix + "/bundles/{bundleID}/templates/{templateSlug}/version/{version}",
		Summary:     "Create a new prompt template version",
		Tags:        []string{tag},
	}, store.PutPromptTemplate)

	huma.Register(api, huma.Operation{
		OperationID: "patch-prompt-template",
		Method:      http.MethodPatch,
		Path:        pathPrefix + "/bundles/{bundleID}/templates/{templateSlug}/version/{version}",
		Summary:     "Enable or disable a prompt template version",
		Tags:        []string{tag},
	}, store.PatchPromptTemplate)

	huma.Register(api, huma.Operation{
		OperationID: "delete-prompt-template",
		Method:      http.MethodDelete,
		Path:        pathPrefix + "/bundles/{bundleID}/templates/{templateSlug}/version/{version}",
		Summary:     "Hard-delete a prompt template version",
		Tags:        []string{tag},
	}, store.DeletePromptTemplate)

	huma.Register(api, huma.Operation{
		OperationID: "get-prompt-template",
		Method:      http.MethodGet,
		Path:        pathPrefix + "/bundles/{bundleID}/templates/{templateSlug}/version/{version}",
		Summary:     "Get a prompt template version",
		Tags:        []string{tag},
	}, store.GetPromptTemplate)

	huma.Register(api, huma.Operation{
		OperationID: "list-prompt-templates",
		Method:      http.MethodGet,
		Path:        pathPrefix + "/templates",
		Summary:     "List all prompt templates (global, with filters)",
		Tags:        []string{tag},
	}, store.ListPromptTemplates)
}
