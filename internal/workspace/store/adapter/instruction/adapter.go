package instruction

import (
	"bytes"
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/instructionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
)

type Document struct {
	Artifact         artifact.ArtifactRef
	ArtifactRevision uint64
	DefinitionDigest string
	Name             string
	MediaType        string
	Content          string
	Locator          basespec.Locator
}

type Adapter struct {
	artifacts compositionapi.ArtifactAPI
	resources compositionapi.ResourceAPI
}

func New(
	artifacts compositionapi.ArtifactAPI,
	resources compositionapi.ResourceAPI,
) (*Adapter, error) {
	if artifacts == nil || resources == nil {
		return nil, fmt.Errorf(
			"%w: Workspace Instruction adapter dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return &Adapter{
		artifacts: artifacts,
		resources: resources,
	}, nil
}

func (a *Adapter) Resolve(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (Document, error) {
	if err := ref.Validate(); err != nil {
		return Document{}, err
	}
	resolved, err := a.resources.ResolveArtifact(
		ctx,
		ref,
		resource.ResolveOptions{},
	)
	if err != nil {
		return Document{}, err
	}
	return a.documentFromResolved(ctx, resolved)
}

func (a *Adapter) documentFromResolved(
	ctx context.Context,
	resolved resource.ResolvedArtifact,
) (Document, error) {
	if err := resolved.Validate(); err != nil {
		return Document{}, err
	}
	if resolved.Artifact.Kind != artifact.ArtifactKind(
		instructionv1.InstructionType,
	) {
		return Document{}, fmt.Errorf(
			"%w: Artifact %q is not an Instruction",
			basespec.ErrReferenceUnresolved,
			resolved.Artifact.ID,
		)
	}
	if resolved.Definition.SchemaID != instructionv1.InstructionSchemaKey.SchemaID ||
		resolved.Definition.SchemaVersion != instructionv1.InstructionSchemaKey.SchemaVersion {
		return Document{}, fmt.Errorf(
			"%w: Instruction Artifact has unsupported schema",
			basespec.ErrReferenceUnresolved,
		)
	}

	document, err := instructionv1.DecodeInstructionJSON(
		resolved.Definition.Body,
	)
	if err != nil {
		return Document{}, err
	}
	content, err := a.contentForDocument(ctx, resolved, document)
	if err != nil {
		return Document{}, err
	}

	return Document{
		Artifact:         resolved.Artifact.Ref(),
		ArtifactRevision: resolved.Artifact.Revision,
		DefinitionDigest: string(resolved.Definition.Digest),
		Name:             document.Name,
		MediaType:        document.MediaType,
		Content:          content,
		Locator:          resolved.Artifact.Binding.Locator,
	}, nil
}

func (a *Adapter) contentForDocument(
	ctx context.Context,
	resolved resource.ResolvedArtifact,
	document instructionv1.InstructionDocument,
) (string, error) {
	if document.Content != nil {
		return *document.Content, nil
	}
	if document.Locator == nil {
		return "", fmt.Errorf(
			"%w: Instruction has neither inline content nor a locator",
			basespec.ErrReferenceUnresolved,
		)
	}

	locator, err := artifactcontract.ResolveSourceRelativePathLocator(
		*document.Locator,
		resolved.Artifact.Binding.Locator,
	)
	if err != nil {
		return "", err
	}
	entry, err := a.resources.ReadSourceEntry(
		ctx,
		resolved.Artifact.RootID,
		resolved.Artifact.Binding.SourceID,
		locator,
		basespec.MaxCandidateBytes,
	)
	if err != nil {
		return "", err
	}
	if entry.SourceGeneration != resolved.RefreshState.SourceGeneration {
		return "", fmt.Errorf(
			"%w: Instruction Source changed during resolution",
			basespec.ErrRefreshRequired,
		)
	}
	if !utf8.Valid(entry.Content) || bytes.ContainsRune(entry.Content, 0) {
		return "", fmt.Errorf(
			"%w: Instruction source %q is not valid text",
			basespec.ErrInvalid,
			entry.Locator,
		)
	}
	return string(entry.Content), nil
}

func Definition(
	value resource.ResolvedArtifact,
) (definition.Definition, error) {
	if err := value.Validate(); err != nil {
		return definition.Definition{}, err
	}
	if value.Definition.Kind != artifact.ArtifactKind(
		instructionv1.InstructionType,
	) {
		return definition.Definition{}, fmt.Errorf(
			"%w: Artifact is not an Instruction",
			basespec.ErrReferenceUnresolved,
		)
	}
	return value.Definition.Clone(), nil
}
