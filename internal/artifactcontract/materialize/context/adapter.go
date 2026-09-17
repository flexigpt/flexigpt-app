package context

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/materialize/text"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
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
	text *text.Adapter
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
	t, err := text.New(resources)
	if err != nil {
		return nil, err
	}
	return &Adapter{text: t}, nil
}

func (a *Adapter) Resolve(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (Document, error) {
	if err := ref.Validate(); err != nil {
		return Document{}, err
	}
	value, err := a.text.Resolve(ctx, ref)
	if err != nil {
		return Document{}, err
	}
	if value.Insert != declaration.InsertUserMessage {
		return Document{}, fmt.Errorf(
			"%w: Text Artifact %q is not user-message Text",
			basespec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}
	return Document{
		Artifact:         value.Artifact,
		ArtifactRevision: value.ArtifactRevision,
		DefinitionDigest: value.DefinitionDigest,
		Name:             value.Name,
		MediaType:        value.MediaType,
		Content:          value.Content,
		Locator:          value.Locator,
	}, nil
}
