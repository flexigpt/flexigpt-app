package store

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

const (
	skillTag        = "SkillStore"
	skillPathPrefix = "/skills"
)

// InitSkillStoreHandlers registers all endpoints for skill bundles and skills.
func InitSkillStoreHandlers(api huma.API, store *SkillStore) {
	// Bundles.
	huma.Register(api, huma.Operation{
		OperationID: "put-skill-bundle",
		Method:      http.MethodPut,
		Path:        skillPathPrefix + "/bundles/{bundleID}",
		Summary:     "Create or replace a skill bundle",
		Tags:        []string{skillTag},
	}, store.PutSkillBundle)

	huma.Register(api, huma.Operation{
		OperationID: "patch-skill-bundle",
		Method:      http.MethodPatch,
		Path:        skillPathPrefix + "/bundles/{bundleID}",
		Summary:     "Enable or disable a skill bundle",
		Tags:        []string{skillTag},
	}, store.PatchSkillBundle)

	huma.Register(api, huma.Operation{
		OperationID: "delete-skill-bundle",
		Method:      http.MethodDelete,
		Path:        skillPathPrefix + "/bundles/{bundleID}",
		Summary:     "Soft-delete a skill bundle (if empty)",
		Tags:        []string{skillTag},
	}, store.DeleteSkillBundle)

	huma.Register(api, huma.Operation{
		OperationID: "list-skill-bundles",
		Method:      http.MethodGet,
		Path:        skillPathPrefix + "/bundles",
		Summary:     "List skill bundles",
		Tags:        []string{skillTag},
	}, store.ListSkillBundles)

	// Skills within a bundle.
	huma.Register(api, huma.Operation{
		OperationID: "put-skill",
		Method:      http.MethodPut,
		Path:        skillPathPrefix + "/bundles/{bundleID}/skills/{skillSlug}",
		Summary:     "Create a skill (unique slug within bundle)",
		Tags:        []string{skillTag},
	}, store.PutSkill)

	huma.Register(api, huma.Operation{
		OperationID: "patch-skill",
		Method:      http.MethodPatch,
		Path:        skillPathPrefix + "/bundles/{bundleID}/skills/{skillSlug}",
		Summary:     "Patch a skill (enable/disable, location)",
		Tags:        []string{skillTag},
	}, store.PatchSkill)

	huma.Register(api, huma.Operation{
		OperationID: "delete-skill",
		Method:      http.MethodDelete,
		Path:        skillPathPrefix + "/bundles/{bundleID}/skills/{skillSlug}",
		Summary:     "Delete a skill (user skills only)",
		Tags:        []string{skillTag},
	}, store.DeleteSkill)

	huma.Register(api, huma.Operation{
		OperationID: "get-skill",
		Method:      http.MethodGet,
		Path:        skillPathPrefix + "/bundles/{bundleID}/skills/{skillSlug}",
		Summary:     "Get a skill",
		Tags:        []string{skillTag},
	}, store.GetSkill)

	// List skills across bundles.
	huma.Register(api, huma.Operation{
		OperationID: "list-skills",
		Method:      http.MethodGet,
		Path:        skillPathPrefix + "/skills",
		Summary:     "List skills (global, with filters)",
		Tags:        []string{skillTag},
	}, store.ListSkills)
}
