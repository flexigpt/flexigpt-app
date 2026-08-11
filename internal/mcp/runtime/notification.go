package runtime

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
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
	Server artifact.ArtifactRef
	Kind   ClientNotificationKind

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
