package attachmentruntime

import (
	"context"
	"errors"

	attachmentRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/attachment"
)

// NormalizedURL is the attachment-domain projection produced by URL
// normalization. It contains presentation facts only and has no Workflow,
// Session, Wails, or Artifact dependency.
type NormalizedURL struct {
	URL            string   `json:"url"`
	Normalized     string   `json:"normalized"`
	Label          string   `json:"label,omitempty"`
	Mode           string   `json:"mode"`
	AvailableModes []string `json:"availableModes,omitempty"`
}

// NormalizeURL validates and normalizes one URL through the attachment domain.
// Workflow adapters consume this typed result but do not own URL parsing.
func NormalizeURL(
	ctx context.Context,
	value string,
) (NormalizedURL, error) {
	attachment, err := attachmentRuntime.BuildAttachmentForURLWithContext(
		ctx,
		value,
	)
	if err != nil {
		return NormalizedURL{}, err
	}
	if attachment == nil || attachment.URLRef == nil {
		return NormalizedURL{}, errors.New(
			"url normalization returned no url reference",
		)
	}

	modes := make(
		[]string,
		0,
		len(attachment.AvailableContentBlockModes),
	)
	for _, mode := range attachment.AvailableContentBlockModes {
		modes = append(modes, string(mode))
	}

	return NormalizedURL{
		URL:            attachment.URLRef.URL,
		Normalized:     attachment.URLRef.Normalized,
		Label:          attachment.Label,
		Mode:           string(attachment.Mode),
		AvailableModes: modes,
	}, nil
}
