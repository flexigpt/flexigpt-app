package materialize

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	resourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	textv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text/contract/v1"
)

type Document struct {
	Artifact         artifactModel.ArtifactRef
	ArtifactRevision uint64
	DefinitionDigest cryptoutil.Digest
	Name             string
	Insert           declaration.InsertTarget
	MediaType        string
	Content          string
	Locator          storeSpec.Locator
	SourceLocators   []storeSpec.Locator
}

type Adapter struct {
	resources resourceFlow.API
}

func NewAdapter(resources resourceFlow.API) (*Adapter, error) {
	if resources == nil {
		return nil, fmt.Errorf(
			"%w: Text materializer ResourceAPI is nil",
			storeSpec.ErrInvalid,
		)
	}
	return &Adapter{resources: resources}, nil
}

func (a *Adapter) Resolve(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (Document, error) {
	resolved, err := a.resources.ResolveArtifact(
		ctx,
		ref,
		resourceModel.ResolveOptions{},
	)
	if err != nil {
		return Document{}, err
	}
	if resolved.Artifact.Kind != artifactModel.ArtifactKind(textv1.TextType) {
		return Document{}, fmt.Errorf(
			"%w: Artifact %q is not Text",
			storeSpec.ErrReferenceUnresolved,
			resolved.Artifact.ID,
		)
	}
	if resolved.Definition.SchemaID != textv1.TextSchemaKey.SchemaID ||
		resolved.Definition.SchemaVersion != textv1.TextSchemaKey.SchemaVersion {
		return Document{}, fmt.Errorf(
			"%w: Text Artifact has unsupported schema",
			storeSpec.ErrReferenceUnresolved,
		)
	}

	declarationValue, err := textv1.DecodeAdmittedTextJSON(resolved.Definition.Body)
	if err != nil {
		return Document{}, err
	}
	content, sourceLocators, err := a.contentForDocument(ctx, resolved, declarationValue)
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
		SourceLocators:   sourceLocators,
	}, nil
}

func (a *Adapter) ResolveWithContentSource(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	contentRootID rootModel.RootID,
	contentSourceID sourceModel.SourceID,
) (Document, error) {
	if contentRootID != ref.RootID {
		return Document{}, fmt.Errorf(
			"%w: Text content Source belongs to another Root",
			storeSpec.ErrInvalid,
		)
	}

	resolved, err := a.resources.ResolveArtifact(ctx, ref, resourceModel.ResolveOptions{})
	if err != nil {
		return Document{}, err
	}
	if resolved.Artifact.Kind != artifactModel.ArtifactKind(textv1.TextType) {
		return Document{}, fmt.Errorf(
			"%w: Artifact %q is not Text",
			storeSpec.ErrReferenceUnresolved,
			resolved.Artifact.ID,
		)
	}
	if resolved.Definition.SchemaID != textv1.TextSchemaKey.SchemaID ||
		resolved.Definition.SchemaVersion != textv1.TextSchemaKey.SchemaVersion {
		return Document{}, fmt.Errorf(
			"%w: Text Artifact has unsupported schema",
			storeSpec.ErrReferenceUnresolved,
		)
	}
	declarationValue, err := textv1.DecodeAdmittedTextJSON(resolved.Definition.Body)
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
			SourceLocators:   nil,
		}, nil
	}
	if declarationValue.Locator == nil {
		return Document{}, fmt.Errorf(
			"%w: Text has neither inline content nor locator",
			storeSpec.ErrReferenceUnresolved,
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
		storeSpec.DefaultMaxEntries,
		storeSpec.MaxScanBytes,
	)
	if err != nil {
		return Document{}, err
	}
	content, sourceLocators, err := materializeSourceEntries(entries, nil)
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
		SourceLocators:   sourceLocators,
	}, nil
}

func (a *Adapter) contentForDocument(
	ctx context.Context,
	resolved resourceModel.ResolvedArtifact,
	document textv1.TextDocument,
) (string, []storeSpec.Locator, error) {
	if document.Content != nil {
		return *document.Content, nil, nil
	}
	if document.Locator == nil {
		return "", nil, fmt.Errorf(
			"%w: Text has neither inline content nor locator",
			storeSpec.ErrReferenceUnresolved,
		)
	}

	base, err := declaration.ResolveSourceRelativePathLocator(
		*document.Locator,
		resolved.Artifact.Binding.Locator,
	)
	if err != nil {
		return "", nil, err
	}
	entries, err := a.resources.ReadSourceTree(
		ctx,
		resolved.Artifact.RootID,
		resolved.Artifact.Binding.SourceID,
		base,
		document.Include,
		document.Exclude,
		storeSpec.DefaultMaxEntries,
		storeSpec.MaxScanBytes,
	)
	if err != nil {
		return "", nil, err
	}

	return materializeSourceEntries(
		entries,
		&sourceEntriesExpectation{
			revision:   resolved.RefreshState.SourceRevision,
			generation: resolved.RefreshState.SourceGeneration,
		},
	)
}

type sourceEntriesExpectation struct {
	revision   uint64
	generation string
}

func materializeSourceEntries(
	entries []resourceModel.VerifiedEntry,
	expected *sourceEntriesExpectation,
) (string, []storeSpec.Locator, error) {
	var output strings.Builder
	locators := make([]storeSpec.Locator, 0, len(entries))
	var generation string
	var revision uint64

	for index, entry := range entries {
		switch {
		case expected != nil &&
			(entry.SourceRevision != expected.revision ||
				entry.SourceGeneration != expected.generation):
			return "", nil, fmt.Errorf(
				"%w: Text Source changed during materialization",
				storeSpec.ErrRefreshRequired,
			)

		case expected == nil && index == 0:
			revision = entry.SourceRevision
			generation = entry.SourceGeneration

		case expected == nil &&
			(entry.SourceRevision != revision ||
				entry.SourceGeneration != generation):
			return "", nil, fmt.Errorf(
				"%w: Text Source changed during materialization",
				storeSpec.ErrRefreshRequired,
			)
		}

		if !utf8.Valid(entry.Content) || bytes.ContainsRune(entry.Content, 0) {
			return "", nil, fmt.Errorf(
				"%w: Text source %q is not valid UTF-8 text",
				storeSpec.ErrInvalid,
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
		locators = append(locators, entry.Locator)
	}
	return output.String(), locators, nil
}
