package internal

import (
	"context"
	"fmt"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func (s *Service) ListByRoot(
	ctx context.Context,
	rootID rootModel.RootID,
	options catalogModel.ListOptions,
) ([]catalogModel.Entry, error) {
	if s == nil || s.repository == nil {
		return nil, spec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Artifact catalog list context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	if err := validateCatalogListOptions(options); err != nil {
		return nil, err
	}

	values, err := s.repository.ListCatalogByRoot(ctx, rootID, options)
	if err != nil {
		return nil, err
	}
	return s.attachDocuments(ctx, values, options.IncludeDocument)
}

func (s *Service) ListBySource(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	options catalogModel.ListOptions,
) ([]catalogModel.Entry, error) {
	if s == nil || s.repository == nil {
		return nil, spec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Artifact catalog list context is nil",
			spec.ErrInvalid,
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
	if err := validateCatalogListOptions(options); err != nil {
		return nil, err
	}

	values, err := s.repository.ListCatalogBySource(
		ctx,
		rootID,
		sourceID,
		options,
	)
	if err != nil {
		return nil, err
	}
	return s.attachDocuments(ctx, values, options.IncludeDocument)
}

func (s *Service) FindByIdentity(
	ctx context.Context,
	rootID rootModel.RootID,
	kind artifactModel.ArtifactKind,
	logicalName spec.LogicalName,
	options catalogModel.ListOptions,
) ([]catalogModel.Entry, error) {
	if s == nil || s.repository == nil {
		return nil, spec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Artifact identity query context is nil",
			spec.ErrInvalid,
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
	if err := validateCatalogListOptions(options); err != nil {
		return nil, err
	}

	values, err := s.repository.FindCatalogByIdentity(
		ctx,
		rootID,
		kind,
		logicalName,
		options,
	)
	if err != nil {
		return nil, err
	}
	return s.attachDocuments(ctx, values, options.IncludeDocument)
}

func (s *Service) GetMany(
	ctx context.Context,
	refs []artifactModel.ArtifactRef,
) ([]artifactModel.Artifact, error) {
	if s == nil || s.repository == nil {
		return nil, spec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Artifact batch get context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(refs) == 0 {
		return []artifactModel.Artifact{}, nil
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
	keys []definitionModel.Key,
) ([]definitionModel.Definition, error) {
	if s == nil || s.definitions == nil {
		return nil, spec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Definition batch get context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return []definitionModel.Definition{}, nil
	}
	return s.definitions.GetDefinitions(ctx, keys)
}

func (s *Service) attachDocuments(
	ctx context.Context,
	values []catalogModel.Entry,
	includeDocument bool,
) ([]catalogModel.Entry, error) {
	output := make([]catalogModel.Entry, len(values))
	for index, value := range values {
		output[index] = value.Clone()
	}
	if !includeDocument {
		return output, nil
	}

	keys := make([]definitionModel.Key, 0, len(output))
	seen := make(map[definitionModel.Key]struct{}, len(output))
	for _, value := range output {
		if value.Definition == nil {
			continue
		}
		key := definitionModel.Key{
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
	byKey := make(map[definitionModel.Key]definitionModel.Definition, len(documents))
	for index, value := range documents {
		byKey[keys[index]] = value
	}

	for index := range output {
		if output[index].Definition == nil {
			continue
		}
		key := definitionModel.Key{
			RootID: output[index].RootID,
			Digest: output[index].Definition.Digest,
		}
		value, found := byKey[key]
		if !found {
			return nil, fmt.Errorf(
				"%w: listed Artifact Definition is unavailable",
				spec.ErrDefinitionNotFound,
			)
		}
		copyValue := value.Clone()
		output[index].Document = &copyValue
	}
	return output, nil
}

func validateCatalogListOptions(
	options catalogModel.ListOptions,
) error {
	if options.Kind != "" {
		if err := options.Kind.Validate(); err != nil {
			return err
		}
	}
	for index, name := range options.LogicalNames {
		if err := name.Validate(); err != nil {
			return fmt.Errorf(
				"catalog logicalNames[%d]: %w",
				index,
				err,
			)
		}
	}
	return nil
}
