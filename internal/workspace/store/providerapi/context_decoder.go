package providerapi

import (
	"bytes"
	"context"
	"encoding/json"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/domain/artifactadapter"
	workspaceDomainContext "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain/context"
)

type ContextDecoder struct{}

func NewContextDecoder() *ContextDecoder {
	return &ContextDecoder{}
}

func (*ContextDecoder) ID() basespec.DecoderID {
	return artifactbuiltin.WorkspaceContextDecoderID
}

func (*ContextDecoder) Revision() string {
	return artifactbuiltin.WorkspaceContextSchemaVersion
}

func (*ContextDecoder) Recognize(
	_ context.Context,
	candidate providerapi.Candidate,
) providerapi.Recognition {
	if _, supported := workspaceDomainContext.RoleFor(candidate.Locator); !supported {
		if candidate.RequestsDecoder(artifactbuiltin.WorkspaceContextDecoderID) &&
			strings.EqualFold(path.Ext(string(candidate.Locator)), ".md") {
			return providerapi.RecognitionPossible
		}
		return providerapi.RecognitionNone
	}
	return providerapi.RecognitionPreferred
}

func (*ContextDecoder) Decode(
	_ context.Context,
	candidate providerapi.Candidate,
) ([]providerapi.Decoded, []diagnostic.Diagnostic) {
	if !utf8.Valid(candidate.Content) {
		return nil, artifactadapter.WorkspaceArtifactDiagnostics(
			candidate.Locator,
			workspaceDomain.DiagnosticCodeContextInvalidUTF8,
			"context file must contain valid UTF-8",
		)
	}
	if bytes.ContainsRune(candidate.Content, 0) {
		return nil, artifactadapter.WorkspaceArtifactDiagnostics(
			candidate.Locator,
			workspaceDomain.DiagnosticCodeContextInvalidContent,
			"context file contains a NUL byte",
		)
	}

	name := path.Base(string(candidate.Locator))
	role, supported := workspaceDomainContext.RoleFor(candidate.Locator)
	if !supported {
		if !candidate.RequestsDecoder(artifactbuiltin.WorkspaceContextDecoderID) ||
			!strings.EqualFold(path.Ext(string(candidate.Locator)), ".md") {
			return nil, nil
		}
		role = artifactbuiltin.WorkspaceContextRoleProjectContext
	}

	document := workspaceDomainContext.Definition{
		Name:      name,
		Role:      role,
		MediaType: artifactbuiltin.WorkspaceContextMediaTypeMarkdown,
		Content: strings.ReplaceAll(
			strings.ReplaceAll(string(candidate.Content), "\r\n", "\n"),
			"\r",
			"\n",
		),
	}
	raw, err := json.Marshal(document)
	if err != nil {
		return nil, artifactadapter.WorkspaceArtifactErrorDiagnostics(candidate.Locator, err)
	}

	value := definition.Definition{
		Kind:          artifactbuiltin.WorkspaceContextArtifactKind,
		SchemaID:      artifactbuiltin.WorkspaceContextSchemaID,
		SchemaVersion: artifactbuiltin.WorkspaceContextSchemaVersion,
		LogicalName:   workspaceDomainContext.LogicalName(name),
		DisplayName:   name,
		Labels: map[string]string{
			artifactbuiltin.WorkspaceContextRoleLabelKey: string(role),
		},
		Body: raw,
	}
	if _, err := workspaceDomainContext.ContextFromDefinition(value); err != nil {
		return nil, artifactadapter.WorkspaceArtifactDiagnostics(
			candidate.Locator,
			workspaceDomain.DiagnosticCodeContextInvalidContent,
			err.Error(),
		)
	}
	return []providerapi.Decoded{{Definition: value}}, nil
}
