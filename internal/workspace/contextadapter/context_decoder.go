package contextadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"

	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
)

type ContextDecoder struct{}

func NewContextDecoder() *ContextDecoder {
	return &ContextDecoder{}
}

func (*ContextDecoder) ID() artifactstore.DecoderID {
	return contextDecoderID
}

func DiscoveryProfile() engine.DiscoveryProfile {
	var profile engine.DiscoveryProfile
	for _, convention := range contextConventionRegistry {
		locator := artifactstore.Locator(convention.FileName)
		switch {
		case convention.DefaultDiscovery:
			profile.ExplicitLocators = append(
				profile.ExplicitLocators,
				locator,
			)
		case convention.Preference == contextPreferenceIncludeReadme:
			profile.ReadmeLocator = locator
		}
	}
	return profile
}

func ArtifactSupport() engine.ArtifactSupport {
	return artifactSupport
}

func (*ContextDecoder) Recognize(
	_ context.Context,
	candidate discovery.Candidate,
) discovery.Recognition {
	if _, supported := contextConventionFor(candidate.Locator); !supported {
		return discovery.RecognitionNone
	}
	return discovery.RecognitionPreferred
}

func (*ContextDecoder) Decode(
	_ context.Context,
	candidate discovery.Candidate,
) ([]discovery.Decoded, []artifactstore.Diagnostic) {
	if !utf8.Valid(candidate.Content) {
		return nil, engine.WorkspaceArtifactDiagnostics(
			candidate.Locator,
			engine.DiagnosticCodeContextInvalidUTF8,
			"context file must contain valid UTF-8",
		)
	}
	if bytes.ContainsRune(candidate.Content, 0) {
		return nil, engine.WorkspaceArtifactDiagnostics(
			candidate.Locator,
			engine.DiagnosticCodeContextInvalidContent,
			"context file contains a NUL byte",
		)
	}

	name := path.Base(string(candidate.Locator))
	convention, supported := contextConventionFor(candidate.Locator)
	if !supported {
		return nil, nil
	}
	document := contextDefinition{
		Name:      name,
		Role:      convention.Role,
		MediaType: contextMarkdownMediaType,
		Content: strings.ReplaceAll(
			strings.ReplaceAll(string(candidate.Content), "\r\n", "\n"),
			"\r",
			"\n",
		),
	}
	raw, err := json.Marshal(document)
	if err != nil {
		return nil, engine.WorkspaceArtifactErrorDiagnostics(candidate.Locator, err)
	}
	raw, err = jsoncanon.CanonicalizeObject(
		raw,
		artifactstore.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return nil, engine.WorkspaceArtifactErrorDiagnostics(candidate.Locator, err)
	}

	value := definition.Definition{
		Kind:          contextKind,
		SchemaID:      contextSchemaID,
		SchemaVersion: workspaceContextSchemaVersionV1,
		LogicalName: artifactstore.LogicalName(
			strings.ToLower(strings.TrimSuffix(name, path.Ext(name))),
		),
		DisplayName: name,
		Labels: map[string]string{
			contextRoleLabelKey: convention.Role,
		},
		Body: raw,
	}
	if err := ValidateContextDefinition(value); err != nil {
		return nil, engine.WorkspaceArtifactDiagnostics(
			candidate.Locator,
			engine.DiagnosticCodeContextInvalidContent,
			err.Error(),
		)
	}
	return []discovery.Decoded{{Definition: value}}, nil
}
