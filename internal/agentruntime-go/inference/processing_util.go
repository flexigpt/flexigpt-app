package inference

import (
	"github.com/flexigpt/inference-go/debugclient"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

var defaultDebugConfig = debugclient.DebugConfig{
	Disable:                 false,
	DisableRequestBody:      false,
	DisableResponseBody:     false,
	DisableContentStripping: false,
	LogToSlog:               false,
}

func DefaultDebugConfig() debugclient.DebugConfig {
	return defaultDebugConfig
}

func disabledDebugConfig() debugclient.DebugConfig {
	config := defaultDebugConfig
	config.Disable = true
	return config
}

func makeStreamHandler(
	onText func(string) error,
	onThinking func(string) error,
) inferenceSpec.StreamHandler {
	return func(event inferenceSpec.StreamEvent) error {
		switch event.Kind {
		case inferenceSpec.StreamContentKindText:
			if onText != nil && event.Text != nil && event.Text.Text != "" {
				return onText(event.Text.Text)
			}
		case inferenceSpec.StreamContentKindThinking:
			if onThinking != nil &&
				event.Thinking != nil &&
				event.Thinking.Text != "" {
				return onThinking(event.Thinking.Text)
			}
		}
		return nil
	}
}
