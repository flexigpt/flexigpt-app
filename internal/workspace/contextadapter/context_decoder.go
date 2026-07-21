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
	return engine.DiscoveryProfile{
		ExplicitLocators: []artifactstore.Locator{
			agentsLocator,
			claudeLocator,
		},
		ReadmeLocator: readmeLocator,
	}
}

func ArtifactSupport() engine.ArtifactSupport {
	return artifactSupport
}

func (*ContextDecoder) Recognize(
	_ context.Context,
	candidate discovery.Candidate,
) discovery.Recognition {
	if _, supported := contextFileSupportFor(
		path.Base(string(candidate.Locator)),
	); !supported {
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
	role := contextRole(name)
	document := contextDefinition{
		Name:      name,
		Role:      role,
		MediaType: contextMarkdownMediaType,
		Content:   strings.ReplaceAll(string(candidate.Content), "\r\n", "\n"),
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
			contextRoleLabelKey: role,
		},
		Body: raw,
	}
	return []discovery.Decoded{{Definition: value}}, nil
}

func contextRole(name string) string {
	support, found := contextFileSupportFor(name)
	if !found {
		return contextRoleProjectContext
	}
	return support.role
}

func contextFileSupportFor(name string) (contextFileSupport, bool) {
	for _, support := range contextFileSupportMatrix {
		if strings.EqualFold(name, support.fileName) {
			return support, true
		}
	}
	return contextFileSupport{}, false
}
