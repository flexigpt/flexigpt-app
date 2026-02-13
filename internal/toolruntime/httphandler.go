package toolruntime

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

const (
	toolRuntimeTag        = "ToolRuntime"
	toolRuntimePathPrefix = "/tools"
)

// InitToolRuntimeHandlers registers runtime endpoints (tool invocation).
func InitToolRuntimeHandlers(api huma.API, rt *ToolRuntime) {
	huma.Register(api, huma.Operation{
		OperationID: "invoke-tool",
		Method:      http.MethodPost,
		Path:        toolRuntimePathPrefix + "/bundles/{bundleID}/tools/{toolSlug}/version/{version}",
		Summary:     "Invoke a tool version",
		Tags:        []string{toolRuntimeTag},
	}, rt.InvokeTool)
}
