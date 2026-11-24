package modelruntime

import (
	"strings"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// CompletionFacts is the Model-domain interpretation of provider completion
// output. Session setup retains Tool binding association separately.
type CompletionFacts struct {
	Text            string
	Thinking        string
	ProtocolOutputs []jsonutil.CanonicalValue
	DebugDetails    jsonutil.CanonicalValue
	Usage           *inferenceSpec.Usage
}

func InterpretCompletion(
	value *inferenceSpec.FetchCompletionResponse,
) (CompletionFacts, error) {
	if value == nil {
		return CompletionFacts{}, nil
	}
	output := CompletionFacts{}
	var text strings.Builder
	var thinking strings.Builder
	for _, item := range value.Outputs {
		if item.OutputMessage != nil {
			for _, content := range item.OutputMessage.Contents {
				if content.Kind == inferenceSpec.ContentItemKindText &&
					content.TextItem != nil {
					text.WriteString(content.TextItem.Text)
				}
			}
		}
		if item.ReasoningMessage != nil {
			thinking.WriteString(strings.Join(item.ReasoningMessage.Thinking, ""))
		}
		raw, err := jsonutil.EncodeToJSONRaw(item)
		if err != nil {
			return CompletionFacts{}, err
		}
		canonical, err := jsonutil.CanonicalValueFrom(raw)
		if err != nil {
			return CompletionFacts{}, err
		}
		output.ProtocolOutputs = append(output.ProtocolOutputs, canonical)
	}
	if value.DebugDetails != nil {
		raw, err := jsonutil.EncodeToJSONRaw(value.DebugDetails)
		if err != nil {
			return CompletionFacts{}, err
		}
		debug, err := jsonutil.CanonicalValueFrom(raw)
		if err != nil {
			return CompletionFacts{}, err
		}
		output.DebugDetails = debug
	}
	if value.Usage != nil {
		usage := *value.Usage
		output.Usage = &usage
	}
	output.Text = text.String()
	output.Thinking = thinking.String()
	return output, nil
}
