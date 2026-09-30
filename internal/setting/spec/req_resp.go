package spec

type SetAppThemeRequestBody struct {
	Type ThemeType `json:"type" required:"true"`
	Name string    `json:"name" required:"true"`
}

type SetAppThemeRequest struct {
	Body *SetAppThemeRequestBody
}

type SetAppThemeResponse struct{}

type SetDebugSettingsRequestBody struct {
	LogLLMReqResp           bool          `json:"logLLMReqResp"           required:"true"`
	DisableContentStripping bool          `json:"disableContentStripping" required:"true"`
	LogLevel                DebugLogLevel `json:"logLevel"                required:"true"`
}

type SetDebugSettingsRequest struct {
	Body *SetDebugSettingsRequestBody
}

type SetDebugSettingsResponse struct{}

type GetSettingsRequest struct {
	ForceFetch bool `query:"forceFetch" doc:"Refresh from disk before reading." required:"false"`
}

type GetSettingsResponseBody struct {
	AppTheme AppTheme      `json:"appTheme"`
	Debug    DebugSettings `json:"debug"`
}

type GetSettingsResponse struct {
	Body *GetSettingsResponseBody
}
