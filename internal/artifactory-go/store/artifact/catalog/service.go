package catalog

import (
	"context"
	"fmt"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// Service owns committed metadata-only catalog reads and optional immutable
// Definition attachment. It deliberately has no Artifact mutation methods.
type Service struct {
	repository  Repository
	definitions definition.API
}

func NewService(repository Repository, definitions definition.API) (*Service, error) {
	if repository == nil || definitions == nil {
		return nil, fmt.Errorf("%w: Artifact catalog dependencies are incomplete", spec.ErrInvalid)
	}
	return &Service{repository: repository, definitions: definitions}, nil
}

func (s *Service) ListByRoot(
	ctx context.Context,
	rootID rootModel.RootID,
	options catalogModel.ListOptions,
) ([]catalogModel.Entry, error) {
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	if err := validateListOptions(options); err != nil {
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
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	if err := sourceID.Validate(); err != nil {
		return nil, err
	}
	if err := validateListOptions(options); err != nil {
		return nil, err
	}
	values, err := s.repository.ListCatalogBySource(ctx, rootID, sourceID, options)
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
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	if err := kind.Validate(); err != nil {
		return nil, err
	}
	if err := logicalName.Validate(); err != nil {
		return nil, err
	}
	if err := validateListOptions(options); err != nil {
		return nil, err
	}
	values, err := s.repository.FindCatalogByIdentity(ctx, rootID, kind, logicalName, options)
	if err != nil {
		return nil, err
	}
	return s.attachDocuments(ctx, values, options.IncludeDocument)
}

func (s *Service) attachDocuments(
	ctx context.Context,
	values []catalogModel.Entry,
	include bool,
) ([]catalogModel.Entry, error) {
	output := make([]catalogModel.Entry, len(values))
	for index, value := range values {
		output[index] = value.Clone()
	}
	if !include {
		return output, nil
	}
	keys := make([]definitionModel.Key, 0, len(output))
	seen := make(map[definitionModel.Key]struct{}, len(output))
	for _, value := range output {
		if value.Definition == nil {
			continue
		}
		key := definitionModel.Key{RootID: value.RootID, Digest: value.Definition.Digest}
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
	if len(documents) != len(keys) {
		return nil, fmt.Errorf("%w: Definition attachment cardinality mismatch", spec.ErrInvalid)
	}
	byKey := make(map[definitionModel.Key]definitionModel.Definition, len(documents))
	for index, value := range documents {
		byKey[keys[index]] = value
	}
	for index := range output {
		if output[index].Definition == nil {
			continue
		}
		key := definitionModel.Key{RootID: output[index].RootID, Digest: output[index].Definition.Digest}
		value, found := byKey[key]
		if !found {
			return nil, fmt.Errorf("%w: listed Artifact Definition is unavailable", spec.ErrDefinitionNotFound)
		}
		copyValue := value.Clone()
		output[index].Document = &copyValue
	}
	return output, nil
}

func validateListOptions(options catalogModel.ListOptions) error {
	if options.Kind != "" {
		if err := options.Kind.Validate(); err != nil {
			return err
		}
	}
	for index, name := range options.LogicalNames {
		if err := name.Validate(); err != nil {
			return fmt.Errorf("catalog logicalNames[%d]: %w", index, err)
		}
	}
	return nil
}
