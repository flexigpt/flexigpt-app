package store

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

const (
	tag        = "Settings"
	pathPrefix = "/settings"
)

func InitSettingStoreHandlers(api huma.API, s *SettingStore) {
	huma.Register(api, huma.Operation{
		OperationID: "settings-set-app-theme",
		Method:      http.MethodPut,
		Path:        pathPrefix + "/theme",
		Summary:     "Set application theme",
		Description: "Create or update the global application theme.",
		Tags:        []string{tag},
	}, s.SetAppTheme)

	huma.Register(api, huma.Operation{
		OperationID: "settings-set-debug-settings",
		Method:      http.MethodPut,
		Path:        pathPrefix + "/debug",
		Summary:     "Set debug settings",
		Description: "Create or update global debug settings such as LLM request/response logging and log level.",
		Tags:        []string{tag},
	}, s.SetDebugSettings)

	huma.Register(api, huma.Operation{
		OperationID: "settings-set-auth-key",
		Method:      http.MethodPut,
		Path:        pathPrefix + "/authkeys/{type}/key/{keyName}",
		Summary:     "Create or update an auth-key",
		Description: "Idempotently create or update an authentication key. The request body determines key type, name and secret.",
		Tags:        []string{tag},
	}, s.SetAuthKey)

	huma.Register(api, huma.Operation{
		OperationID: "settings-delete-auth-key",
		Method:      http.MethodDelete,
		Path:        pathPrefix + "/authkeys/{type}/key/{keyName}",
		Summary:     "Delete an auth-key",
		Description: "Remove an authentication key unless it is marked as built-in.",
		Tags:        []string{tag},
	}, s.DeleteAuthKey)

	huma.Register(api, huma.Operation{
		OperationID: "settings-get-auth-key",
		Method:      http.MethodGet,
		Path:        pathPrefix + "/authkeys/{type}/key/{keyName}",
		Summary:     "Get a single auth-key (secret included)",
		Description: "Returns the decrypted secret and SHA-256 for a stored key.",
		Tags:        []string{tag},
	}, s.GetAuthKey)

	huma.Register(api, huma.Operation{
		OperationID: "settings-get-all",
		Method:      http.MethodGet,
		Path:        pathPrefix,
		Summary:     "Get all settings (no secrets)",
		Description: "Returns the application theme, debug settings and metadata for every stored authentication key.",
		Tags:        []string{tag},
	}, s.GetSettings)
}
