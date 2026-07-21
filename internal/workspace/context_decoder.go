package workspace

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
)

type ContextDecoder struct{}

func NewContextDecoder() *ContextDecoder {
	return &ContextDecoder{}
}

func (*ContextDecoder) ID() artifactstore.DecoderID {
	return ContextDecoderID
}

func (*ContextDecoder) Recognize(
	_ context.Context,
	candidate discovery.Candidate,
) discovery.Recognition {
	switch strings.ToUpper(path.Base(string(candidate.Locator))) {
	case "AGENTS.MD", "CLAUDE.MD", "README.MD":
		return discovery.RecognitionPreferred
	default:
		return discovery.RecognitionNone
	}
}

func (*ContextDecoder) Decode(
	_ context.Context,
	candidate discovery.Candidate,
) ([]discovery.Decoded, []artifactstore.Diagnostic) {
	if !utf8.Valid(candidate.Content) {
		return nil, workspaceArtifactDiagnostics(
			candidate.Locator,
			"workspace.context.invalid-utf8",
			"context file must contain valid UTF-8",
		)
	}
	if bytes.ContainsRune(candidate.Content, 0) {
		return nil, workspaceArtifactDiagnostics(
			candidate.Locator,
			"workspace.context.invalid-content",
			"context file contains a NUL byte",
		)
	}

	name := path.Base(string(candidate.Locator))
	role := contextRole(name)
	document := ContextDefinition{
		Name:      name,
		Role:      role,
		MediaType: "text/markdown",
		Content:   strings.ReplaceAll(string(candidate.Content), "\r\n", "\n"),
	}
	raw, err := json.Marshal(document)
	if err != nil {
		return nil, workspaceArtifactErrorDiagnostics(candidate.Locator, err)
	}
	raw, err = jsoncanon.CanonicalizeObject(
		raw,
		artifactstore.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return nil, workspaceArtifactErrorDiagnostics(candidate.Locator, err)
	}

	value := definition.Definition{
		Kind:          ContextKind,
		SchemaID:      ContextSchemaID,
		SchemaVersion: "1",
		LogicalName: artifactstore.LogicalName(
			strings.ToLower(strings.TrimSuffix(name, path.Ext(name))),
		),
		DisplayName: name,
		Labels: map[string]string{
			"context.role": role,
		},
		Body: raw,
	}
	return []discovery.Decoded{{Definition: value}}, nil
}

func contextRole(name string) string {
	switch strings.ToUpper(name) {
	case "AGENTS.MD":
		return "agent-instructions"
	case "CLAUDE.MD":
		return "assistant-instructions"
	case "README.MD":
		return "project-readme"
	default:
		return "project-context"
	}
}

func workspaceArtifactErrorDiagnostics(
	locator artifactstore.Locator,
	err error,
) []artifactstore.Diagnostic {
	return workspaceArtifactDiagnostics(
		locator,
		"workspace.artifact.invalid",
		err.Error(),
	)
}

func workspaceArtifactDiagnostics(
	locator artifactstore.Locator,
	code string,
	message string,
) []artifactstore.Diagnostic {
	for len(message) > artifactstore.MaxDiagnosticMessageBytes {
		_, size := utf8.DecodeLastRuneInString(message)
		message = message[:len(message)-size]
	}
	return []artifactstore.Diagnostic{{
		Severity: artifactstore.DiagnosticError,
		Code:     code,
		Message:  message,
		Location: &artifactstore.DiagnosticLocation{
			Locator: locator,
		},
	}}
}

var _ discovery.Decoder = (*ContextDecoder)(nil)
