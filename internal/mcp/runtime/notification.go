package runtime

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

type ClientNotificationKind string

const (
	ClientNotificationToolListChanged     ClientNotificationKind = "toolsListChanged"
	ClientNotificationResourceListChanged ClientNotificationKind = "resourcesListChanged"
	ClientNotificationPromptListChanged   ClientNotificationKind = "promptsListChanged"
	ClientNotificationResourceUpdated     ClientNotificationKind = "resourceUpdated"
	ClientNotificationProgress            ClientNotificationKind = "progress"
)

type ClientNotification struct {
	BundleID bundleitemutils.BundleID
	ServerID spec.MCPServerID
	Kind     ClientNotificationKind

	ResourceURI string

	LoggerName   string
	LoggingLevel string
	LogData      any

	Progress float64
	Total    float64
	Message  string
}

type ClientNotificationSink interface {
	OnClientNotification(ctx context.Context, event ClientNotification)
}
