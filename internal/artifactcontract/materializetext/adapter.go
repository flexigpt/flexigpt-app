package materializetext

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/textv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type Document struct {
	Artifact         artifact.ArtifactRef
	ArtifactRevision uint64
	DefinitionDigest cryptoutil.Digest
	Name             string
	Insert           declaration.InsertTarget
	MediaType        string
	Content          string
	Locator          basespec.Locator
}

type Adapter struct {
	resources compositionapi.ResourceAPI
}

func NewAdapter(resources compositionapi.ResourceAPI) (*Adapter, error) {
	if resources == nil {
		return nil, fmt.Errorf(
			"%w: Text materializer ResourceAPI is nil",
			basespec.ErrInvalid,
		)
	}
	return &Adapter{resources: resources}, nil
}

func (a *Adapter) Resolve(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (Document, error) {
	if a == nil || a.resources == nil {
		return Document{}, basespec.ErrClosed
	}
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
	if err := resolved.Validate(); err != nil {
		return Document{}, err
	}
	if resolved.Artifact.Kind != artifact.ArtifactKind(textv1.TextType) {
		return Document{}, fmt.Errorf(
			"%w: Artifact %q is not Text",
			basespec.ErrReferenceUnresolved,
			resolved.Artifact.ID,
		)
	}
	if resolved.Definition.SchemaID != textv1.TextSchemaKey.SchemaID ||
		resolved.Definition.SchemaVersion != textv1.TextSchemaKey.SchemaVersion {
		return Document{}, fmt.Errorf(
			"%w: Text Artifact has unsupported schema",
			basespec.ErrReferenceUnresolved,
		)
	}

	declarationValue, err := textv1.DecodeTextJSON(resolved.Definition.Body)
	if err != nil {
		return Document{}, err
	}
	content, err := a.contentForDocument(ctx, resolved, declarationValue)
	if err != nil {
		return Document{}, err
	}
	return Document{
		Artifact:         resolved.Artifact.Ref(),
		ArtifactRevision: resolved.Artifact.Revision,
		DefinitionDigest: resolved.Definition.Digest,
		Name:             declarationValue.Name,
		Insert:           declarationValue.Insert,
		MediaType:        declarationValue.MediaType,
		Content:          content,
		Locator:          resolved.Artifact.Binding.Locator,
	}, nil
}

func (a *Adapter) ResolveWithContentSource(
	ctx context.Context,
	ref artifact.ArtifactRef,
	contentRootID root.RootID,
	contentSourceID source.SourceID,
) (Document, error) {
	if a == nil || a.resources == nil {
		return Document{}, basespec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return Document{}, err
	}
	if err := contentRootID.Validate(); err != nil {
		return Document{}, err
	}
	if err := contentSourceID.Validate(); err != nil {
		return Document{}, err
	}
	if contentRootID != ref.RootID {
		return Document{}, fmt.Errorf(
			"%w: Text content Source belongs to another Root",
			basespec.ErrInvalid,
		)
	}

	resolved, err := a.resources.ResolveArtifact(ctx, ref, resource.ResolveOptions{})
	if err != nil {
		return Document{}, err
	}
	if err := resolved.Validate(); err != nil {
		return Document{}, err
	}
	if resolved.Artifact.Kind != artifact.ArtifactKind(textv1.TextType) {
		return Document{}, fmt.Errorf(
			"%w: Artifact %q is not Text",
			basespec.ErrReferenceUnresolved,
			resolved.Artifact.ID,
		)
	}
	if resolved.Definition.SchemaID != textv1.TextSchemaKey.SchemaID ||
		resolved.Definition.SchemaVersion != textv1.TextSchemaKey.SchemaVersion {
		return Document{}, fmt.Errorf(
			"%w: Text Artifact has unsupported schema",
			basespec.ErrReferenceUnresolved,
		)
	}
	declarationValue, err := textv1.DecodeTextJSON(resolved.Definition.Body)
	if err != nil {
		return Document{}, err
	}
	if declarationValue.Content != nil {
		return Document{
			Artifact:         resolved.Artifact.Ref(),
			ArtifactRevision: resolved.Artifact.Revision,
			DefinitionDigest: resolved.Definition.Digest,
			Name:             declarationValue.Name,
			Insert:           declarationValue.Insert,
			MediaType:        declarationValue.MediaType,
			Content:          *declarationValue.Content,
			Locator:          resolved.Artifact.Binding.Locator,
		}, nil
	}
	if declarationValue.Locator == nil {
		return Document{}, fmt.Errorf(
			"%w: Text has neither inline content nor locator",
			basespec.ErrReferenceUnresolved,
		)
	}
	base, err := declaration.ResolveSourceRelativePathLocator(
		*declarationValue.Locator,
		resolved.Artifact.Binding.Locator,
	)
	if err != nil {
		return Document{}, err
	}
	entries, err := a.resources.ReadSourceTree(
		ctx,
		contentRootID,
		contentSourceID,
		base,
		declarationValue.Include,
		declarationValue.Exclude,
		basespec.DefaultMaxEntries,
		basespec.MaxScanBytes,
	)
	if err != nil {
		return Document{}, err
	}
	var output strings.Builder
	var generation string
	var revision uint64
	for index, entry := range entries {
		if index == 0 {
			generation = entry.SourceGeneration
			revision = entry.SourceRevision
		}
		if entry.SourceGeneration != generation || entry.SourceRevision != revision {
			return Document{}, fmt.Errorf(
				"%w: Text Source changed during materialization",
				basespec.ErrRefreshRequired,
			)
		}
		if !utf8.Valid(entry.Content) || bytes.ContainsRune(entry.Content, 0) {
			return Document{}, fmt.Errorf(
				"%w: Text source %q is not valid UTF-8 text",
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
	return Document{
		Artifact:         resolved.Artifact.Ref(),
		ArtifactRevision: resolved.Artifact.Revision,
		DefinitionDigest: resolved.Definition.Digest,
		Name:             declarationValue.Name,
		Insert:           declarationValue.Insert,
		MediaType:        declarationValue.MediaType,
		Content:          output.String(),
		Locator:          resolved.Artifact.Binding.Locator,
	}, nil
}

func (a *Adapter) contentForDocument(
	ctx context.Context,
	resolved resource.ResolvedArtifact,
	document textv1.TextDocument,
) (string, error) {
	if document.Content != nil {
		return *document.Content, nil
	}
	if document.Locator == nil {
		return "", fmt.Errorf(
			"%w: Text has neither inline content nor locator",
			basespec.ErrReferenceUnresolved,
		)
	}

	base, err := declaration.ResolveSourceRelativePathLocator(
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

	var output strings.Builder
	for index, entry := range entries {
		if entry.SourceRevision != resolved.RefreshState.SourceRevision ||
			entry.SourceGeneration != resolved.RefreshState.SourceGeneration {
			return "", fmt.Errorf(
				"%w: Text Source changed during materialization",
				basespec.ErrRefreshRequired,
			)
		}
		if !utf8.Valid(entry.Content) || bytes.ContainsRune(entry.Content, 0) {
			return "", fmt.Errorf(
				"%w: Text source %q is not valid UTF-8 text",
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
	return output.String(), nil
}
