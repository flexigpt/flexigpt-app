package context

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/contextv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
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
	resources compositionapi.ResourceAPI
}

func New(
	resources compositionapi.ResourceAPI,
) (*Adapter, error) {
	if resources == nil {
		return nil, fmt.Errorf(
			"%w: Workspace Context adapter dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return &Adapter{resources: resources}, nil
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
		contextv1.ContextType,
	) {
		return Document{}, fmt.Errorf(
			"%w: Artifact %q is not Context",
			basespec.ErrReferenceUnresolved,
			resolved.Artifact.ID,
		)
	}
	if resolved.Definition.SchemaID != contextv1.ContextSchemaKey.SchemaID ||
		resolved.Definition.SchemaVersion != contextv1.ContextSchemaKey.SchemaVersion {
		return Document{}, fmt.Errorf(
			"%w: Context Artifact has unsupported schema",
			basespec.ErrReferenceUnresolved,
		)
	}

	document, err := contextv1.DecodeContextJSON(
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
	document contextv1.ContextDocument,
) (string, error) {
	if document.Content != nil {
		return *document.Content, nil
	}
	if document.Locator == nil {
		return "", fmt.Errorf(
			"%w: Context has neither inline content nor a locator",
			basespec.ErrReferenceUnresolved,
		)
	}

	base, err := declarationRelativeSourceLocator(
		*document.Locator,
		resolved.Artifact.Binding.Locator,
	)
	if err != nil {
		return "", err
	}
	entries, err := a.resources.ReadSourceTree(
		ctx,
		resolved.Artifact.RootID,
		resolved.Artifact.Binding.SourceID,
		base,
		document.Include,
		document.Exclude,
		basespec.DefaultMaxEntries,
		basespec.MaxScanBytes,
	)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "", fmt.Errorf(
			"%w: Context locator selected no source entries",
			basespec.ErrReferenceUnresolved,
		)
	}

	var output strings.Builder
	for index, entry := range entries {
		if entry.SourceGeneration !=
			resolved.RefreshState.SourceGeneration {
			return "", fmt.Errorf(
				"%w: Context Source changed during selection",
				basespec.ErrRefreshRequired,
			)
		}
		if !utf8.Valid(entry.Content) {
			return "", fmt.Errorf(
				"%w: Context source %q is not valid UTF-8",
				basespec.ErrInvalid,
				entry.Locator,
			)
		}
		if bytes.ContainsRune(entry.Content, 0) {
			return "", fmt.Errorf(
				"%w: Context source %q contains a NUL byte",
				basespec.ErrInvalid,
				entry.Locator,
			)
		}
		if index != 0 {
			output.WriteString("\n\n")
		}
		if len(entries) > 1 {
			output.WriteString("--- ")
			output.WriteString(string(entry.Locator))
			output.WriteString(" ---\n")
		}
		output.Write(entry.Content)
	}

	content := output.String()
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf(
			"%w: Context locator selected only empty content",
			basespec.ErrReferenceUnresolved,
		)
	}
	return content, nil
}

func declarationRelativeSourceLocator(
	locator declaration.Locator,
	declarationLocator basespec.Locator,
) (basespec.Locator, error) {
	return declaration.ResolveSourceRelativePathLocator(
		locator, declarationLocator,
	)
}
