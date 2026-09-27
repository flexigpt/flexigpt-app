package artifactimpl

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

func (s *Service) ListByRoot(
	ctx context.Context,
	rootID root.RootID,
	options catalog.ListOptions,
) ([]catalog.Entry, error) {
	if s == nil || s.repository == nil {
		return nil, basespec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Artifact catalog list context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := rootID.Validate(); err != nil {
		return nil, err
	}

	values, err := s.repository.ListCatalogByRoot(ctx, rootID)
	if err != nil {
		return nil, err
	}
	return s.attachDocuments(ctx, values, options)
}

func (s *Service) ListBySource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	options catalog.ListOptions,
) ([]catalog.Entry, error) {
	if s == nil || s.repository == nil {
		return nil, basespec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Artifact catalog list context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	if err := sourceID.Validate(); err != nil {
		return nil, err
	}

	values, err := s.repository.ListCatalogBySource(
		ctx,
		rootID,
		sourceID,
	)
	if err != nil {
		return nil, err
	}
	return s.attachDocuments(ctx, values, options)
}

func (s *Service) FindByIdentity(
	ctx context.Context,
	rootID root.RootID,
	kind artifact.ArtifactKind,
	logicalName basespec.LogicalName,
	options catalog.ListOptions,
) ([]catalog.Entry, error) {
	if s == nil || s.repository == nil {
		return nil, basespec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Artifact identity query context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	if err := kind.Validate(); err != nil {
		return nil, err
	}
	if err := logicalName.Validate(); err != nil {
		return nil, err
	}

	values, err := s.repository.FindCatalogByIdentity(
		ctx,
		rootID,
		kind,
		logicalName,
	)
	if err != nil {
		return nil, err
	}
	return s.attachDocuments(ctx, values, options)
}

func (s *Service) GetMany(
	ctx context.Context,
	refs []artifact.ArtifactRef,
) ([]artifact.Artifact, error) {
	if s == nil || s.repository == nil {
		return nil, basespec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Artifact batch get context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(refs) == 0 {
		return []artifact.Artifact{}, nil
	}

	for _, ref := range refs {
		if err := ref.Validate(); err != nil {
			return nil, err
		}
	}
	return s.repository.GetMany(ctx, refs)
}

func (s *Service) GetDefinitions(
	ctx context.Context,
	keys []definition.Key,
) ([]definition.Definition, error) {
	if s == nil || s.definitions == nil {
		return nil, basespec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Definition batch get context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return []definition.Definition{}, nil
	}
	return s.definitions.GetDefinitions(ctx, keys)
}

func (s *Service) attachDocuments(
	ctx context.Context,
	values []catalog.Entry,
	options catalog.ListOptions,
) ([]catalog.Entry, error) {
	output := make([]catalog.Entry, len(values))
	for index, value := range values {
		output[index] = value.Clone()
	}
	if !options.IncludeDocument {
		return output, nil
	}

	keys := make([]definition.Key, 0, len(output))
	seen := make(map[definition.Key]struct{}, len(output))
	for _, value := range output {
		if value.Definition == nil {
			continue
		}
		key := definition.Key{
			RootID: value.RootID,
			Digest: value.Definition.Digest,
		}
		if _, found := seen[key]; found {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return output, nil
	}

	documents, err := s.definitions.GetDefinitions(ctx, keys)
	if err != nil {
		return nil, err
	}
	byKey := make(map[definition.Key]definition.Definition, len(documents))
	for index, value := range documents {
		byKey[keys[index]] = value
	}

	for index := range output {
		if output[index].Definition == nil {
			continue
		}
		key := definition.Key{
			RootID: output[index].RootID,
			Digest: output[index].Definition.Digest,
		}
		value, found := byKey[key]
		if !found {
			return nil, fmt.Errorf(
				"%w: listed Artifact Definition is unavailable",
				basespec.ErrDefinitionNotFound,
			)
		}
		copyValue := value.Clone()
		output[index].Document = &copyValue
	}
	return output, nil
}
