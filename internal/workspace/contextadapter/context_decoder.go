package contextadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"

	"github.com/flexigpt/flexigpt-app/internal/workspace/artifactadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
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

func DiscoveryProfile() spec.DiscoveryProfile {
	var profile spec.DiscoveryProfile
	for _, convention := range contextConventionRegistry {
		locator := basespec.Locator(convention.FileName)
		switch {
		case convention.DefaultDiscovery:
			profile.ExplicitLocators = append(
				profile.ExplicitLocators,
				locator,
			)
		case convention.Preference == artifactbuiltin.WorkspaceContextPreferenceIncludeReadme:
			profile.ReadmeLocator = locator
		}
	}
	return profile
}

func ArtifactSupport() spec.ArtifactSupport {
	return artifactSupport
}

func (*ContextDecoder) Recognize(
	_ context.Context,
	candidate providerapi.Candidate,
) providerapi.Recognition {
	if _, supported := contextConventionFor(candidate.Locator); !supported {
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
) ([]providerapi.Decoded, []providerapi.Diagnostic) {
	if !utf8.Valid(candidate.Content) {
		return nil, artifactadapter.WorkspaceArtifactDiagnostics(
			candidate.Locator,
			artifactadapter.DiagnosticCodeContextInvalidUTF8,
			"context file must contain valid UTF-8",
		)
	}
	if bytes.ContainsRune(candidate.Content, 0) {
		return nil, artifactadapter.WorkspaceArtifactDiagnostics(
			candidate.Locator,
			artifactadapter.DiagnosticCodeContextInvalidContent,
			"context file contains a NUL byte",
		)
	}

	name := path.Base(string(candidate.Locator))
	convention, supported := contextConventionFor(candidate.Locator)
	if !supported {
		if !candidate.RequestsDecoder(artifactbuiltin.WorkspaceContextDecoderID) ||
			!strings.EqualFold(path.Ext(string(candidate.Locator)), ".md") {
			return nil, nil
		}
		convention = contextFileSupport{
			Role: artifactbuiltin.WorkspaceContextRoleProjectContext,
		}
	}

	document := contextDefinition{
		Name:      name,
		Role:      convention.Role,
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
	raw, err = jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return nil, artifactadapter.WorkspaceArtifactErrorDiagnostics(candidate.Locator, err)
	}

	value := providerapi.Definition{
		Kind:          artifactbuiltin.WorkspaceContextArtifactKind,
		SchemaID:      artifactbuiltin.WorkspaceContextSchemaID,
		SchemaVersion: artifactbuiltin.WorkspaceContextSchemaVersion,
		LogicalName:   contextLogicalName(name),
		DisplayName:   name,
		Labels: map[string]string{
			artifactbuiltin.WorkspaceContextRoleLabelKey: string(convention.Role),
		},
		Body: raw,
	}
	if err := ValidateContextDefinition(value); err != nil {
		return nil, artifactadapter.WorkspaceArtifactDiagnostics(
			candidate.Locator,
			artifactadapter.DiagnosticCodeContextInvalidContent,
			err.Error(),
		)
	}
	return []providerapi.Decoded{{Definition: value}}, nil
}
