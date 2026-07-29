package contextadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"

	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
)

type ContextDecoder struct{}

func NewContextDecoder() *ContextDecoder {
	return &ContextDecoder{}
}

func (*ContextDecoder) ID() basespec.DecoderID {
	return contextDecoderID
}

func (*ContextDecoder) Revision() string {
	return workspaceContextSchemaVersionV1
}

func DiscoveryProfile() engine.DiscoveryProfile {
	var profile engine.DiscoveryProfile
	for _, convention := range contextConventionRegistry {
		locator := basespec.Locator(convention.FileName)
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
		if candidate.RequestsDecoder(contextDecoderID) &&
			strings.EqualFold(path.Ext(string(candidate.Locator)), ".md") {
			return discovery.RecognitionPossible
		}
		return discovery.RecognitionNone
	}
	return discovery.RecognitionPreferred
}

func (*ContextDecoder) Decode(
	_ context.Context,
	candidate discovery.Candidate,
) ([]discovery.Decoded, []diagnostic.Diagnostic) {
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
		if !candidate.RequestsDecoder(contextDecoderID) ||
			!strings.EqualFold(path.Ext(string(candidate.Locator)), ".md") {
			return nil, nil
		}
		convention = contextFileSupport{
			Role: contextRoleProjectContext,
		}
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
	raw, err = jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return nil, engine.WorkspaceArtifactErrorDiagnostics(candidate.Locator, err)
	}

	value := definition.Definition{
		Kind:          contextKind,
		SchemaID:      contextSchemaID,
		SchemaVersion: workspaceContextSchemaVersionV1,
		LogicalName:   contextLogicalName(name),
		DisplayName:   name,
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
