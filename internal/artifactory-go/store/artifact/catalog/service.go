package catalog

import (
	"context"
	"errors"
	"fmt"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// Service owns committed metadata-only catalog reads and optional immutable
// Definition attachment. It deliberately has no Artifact mutation methods.
type Service struct {
	repository  Repository
	definitions definition.API
}

func NewService(repository Repository, definitions definition.API) (*Service, error) {
	if repository == nil || definitions == nil {
		return nil, fmt.Errorf("%w: Artifact catalog dependencies are incomplete", storeSpec.ErrInvalid)
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
	logicalName storeSpec.LogicalName,
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
		output[index].Document = nil
		output[index].DocumentIssue = nil
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

	attached := make(
		map[definitionModel.Key]definitionModel.Definition,
		len(keys),
	)
	issues := make(
		map[definitionModel.Key]catalogModel.DocumentIssue,
	)
	if err := s.attachAvailableDocuments(
		ctx,
		keys,
		attached,
		issues,
	); err != nil {
		return nil, err
	}

	for index := range output {
		if output[index].Definition == nil {
			continue
		}
		key := definitionModel.Key{
			RootID: output[index].RootID,
			Digest: output[index].Definition.Digest,
		}
		if value, found := attached[key]; found {
			copyValue := value.Clone()
			output[index].Document = &copyValue
			continue
		}
		if issue, found := issues[key]; found {
			copyIssue := issue
			output[index].DocumentIssue = &copyIssue
		}
	}
	return output, nil
}

// attachAvailableDocuments uses one bulk Definition read on the healthy path.
// If one historical or damaged Definition is unavailable, it bisects only the
// failed batch. This preserves useful list rows without weakening the strict
// direct Definition lookup contract.
func (s *Service) attachAvailableDocuments(
	ctx context.Context,
	keys []definitionModel.Key,
	attached map[definitionModel.Key]definitionModel.Definition,
	issues map[definitionModel.Key]catalogModel.DocumentIssue,
) error {
	values, err := s.definitions.GetDefinitions(ctx, keys)
	if err == nil {
		complete := len(values) == len(keys)
		if complete {
			for index, value := range values {
				if value.Digest != keys[index].Digest {
					complete = false
					break
				}
			}
		}
		if complete {
			for index, value := range values {
				attached[keys[index]] = value.Clone()
			}
			return nil
		}
		err = fmt.Errorf(
			"%w: Definition attachment did not preserve requested keys",
			storeSpec.ErrInvalid,
		)
	}

	if !isDegradableDefinitionAttachmentFailure(err) {
		return err
	}
	if len(keys) == 1 {
		issues[keys[0]] = definitionAttachmentIssue(err)
		return nil
	}

	middle := len(keys) / 2
	if err := s.attachAvailableDocuments(
		ctx,
		keys[:middle],
		attached,
		issues,
	); err != nil {
		return err
	}
	return s.attachAvailableDocuments(
		ctx,
		keys[middle:],
		attached,
		issues,
	)
}

func isDegradableDefinitionAttachmentFailure(err error) bool {
	return errors.Is(err, storeSpec.ErrDefinitionNotFound) ||
		errors.Is(err, storeSpec.ErrNotFound) ||
		errors.Is(err, storeSpec.ErrDigestMismatch) ||
		errors.Is(err, storeSpec.ErrInvalid)
}

func definitionAttachmentIssue(err error) catalogModel.DocumentIssue {
	switch {
	case errors.Is(err, storeSpec.ErrDefinitionNotFound),
		errors.Is(err, storeSpec.ErrNotFound):
		return catalogModel.DocumentIssue{
			Code:    "artifact.definition-unavailable",
			Message: "the Artifact Definition is no longer available",
		}
	case errors.Is(err, storeSpec.ErrDigestMismatch):
		return catalogModel.DocumentIssue{
			Code:    "artifact.definition-mismatch",
			Message: "the Artifact Definition does not match committed evidence",
		}
	default:
		return catalogModel.DocumentIssue{
			Code:    "artifact.definition-invalid",
			Message: "the Artifact Definition cannot be used for list enrichment",
		}
	}
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
