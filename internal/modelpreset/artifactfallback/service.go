package artifactfallback

import (
	"context"
	"fmt"
	"slices"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	modelpresetSpec "github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	modelpresetStore "github.com/flexigpt/flexigpt-app/internal/modelpreset/store"
	"github.com/flexigpt/inference-go/modelpreset"
	"github.com/flexigpt/inference-go/spec"
)

// Service resolves legacy built-in ModelPresetStore entries as mapped Artifact
// targets and translates mapped targets back into legacy ModelPresetRefs.
type Service struct {
	store *modelpresetStore.ModelPresetStore

	// Immutable after construction. Multiple values for one name are
	// deliberately retained so resolution can report ambiguity.
	byName map[basespec.LogicalName][]TargetV1
}

type ResolveTargetRequest struct {
	Target resolve.MappedTarget `json:"target" required:"true"`
}

type ResolveTargetResponseBody struct {
	ModelPresetRef modelpresetSpec.ModelPresetRef `json:"modelPresetRef"`
}

type ResolveTargetResponse struct {
	Body *ResolveTargetResponseBody
}

func NewService(
	ctx context.Context,
	store *modelpresetStore.ModelPresetStore,
	bindings ...Binding,
) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf(
			"%w: Model fallback ModelPresetStore is nil",
			basespec.ErrInvalid,
		)
	}
	if err := validateContext(ctx); err != nil {
		return nil, err
	}

	byName, err := buildIndex(ctx, store, bindings)
	if err != nil {
		return nil, err
	}

	return &Service{
		store:  store,
		byName: byName,
	}, nil
}

// ResolveFallback implements resolve.FallbackProvider.
func (s *Service) ResolveFallback(
	ctx context.Context,
	request resolve.FallbackRequest,
) (resolve.FallbackTarget, bool, error) {
	if s == nil || s.store == nil {
		return resolve.FallbackTarget{}, false, basespec.ErrClosed
	}
	if err := validateFallbackRequest(request); err != nil {
		return resolve.FallbackTarget{}, false, err
	}
	if err := validateContext(ctx); err != nil {
		return resolve.FallbackTarget{}, false, err
	}

	value, found, err := s.targetForName(request.Name)
	if err != nil {
		return resolve.FallbackTarget{}, false, err
	}
	if !found {
		return resolve.FallbackTarget{}, false, nil
	}

	if _, err := s.resolveLiveTarget(ctx, request.Name, value); err != nil {
		return resolve.FallbackTarget{}, false, err
	}

	mapped, err := NewMappedTarget(request.Name, value)
	if err != nil {
		return resolve.FallbackTarget{}, false, err
	}

	return resolve.FallbackTarget{
		Mapped: &mapped,
	}, true, nil
}

// ResolveTarget translates a Model mapped target into the legacy
// ModelPresetRef used by existing inference and conversation flows.
func (s *Service) ResolveTarget(
	ctx context.Context,
	target resolve.MappedTarget,
) (modelpresetSpec.ModelPresetRef, error) {
	if s == nil || s.store == nil {
		return modelpresetSpec.ModelPresetRef{}, basespec.ErrClosed
	}
	if err := validateContext(ctx); err != nil {
		return modelpresetSpec.ModelPresetRef{}, err
	}

	value, err := DecodeTarget(target)
	if err != nil {
		return modelpresetSpec.ModelPresetRef{}, err
	}

	indexed, found, err := s.targetForName(target.Name)
	if err != nil {
		return modelpresetSpec.ModelPresetRef{}, err
	}
	if !found {
		return modelpresetSpec.ModelPresetRef{}, unavailableError(
			"Model mapped target %q is no longer registered",
			target.Name,
		)
	}
	if indexed != value {
		return modelpresetSpec.ModelPresetRef{}, unavailableError(
			"Model mapped target %q no longer matches its built-in binding",
			target.Name,
		)
	}

	return s.resolveLiveTarget(ctx, target.Name, value)
}

func buildIndex(
	ctx context.Context,
	store *modelpresetStore.ModelPresetStore,
	bindings []Binding,
) (map[basespec.LogicalName][]TargetV1, error) {
	providers, err := store.ListBuiltInProviderPresets(ctx)
	if err != nil {
		return nil, err
	}
	if len(providers) == 0 {
		return nil, fmt.Errorf(
			"%w: Model fallback found no built-in providers",
			basespec.ErrReferenceUnresolved,
		)
	}

	output := make(map[basespec.LogicalName][]TargetV1)
	byIdentity := make(map[string]TargetV1)

	for _, provider := range providers {
		if !provider.IsBuiltIn {
			continue
		}

		for modelID, model := range provider.ModelPresets {
			if !model.IsBuiltIn {
				continue
			}
			if model.ID != modelID {
				return nil, fmt.Errorf(
					"%w: built-in Model Preset map key %q differs from Model ID %q",
					basespec.ErrInvalid,
					modelID,
					model.ID,
				)
			}

			target := TargetV1{
				ProviderName:  provider.Name,
				ModelPresetID: model.ID,
				ModelName:     model.Name,
			}
			if err := target.Validate(); err != nil {
				return nil, fmt.Errorf(
					"invalid built-in Model Preset %q/%q: %w",
					provider.Name,
					model.ID,
					err,
				)
			}

			key := targetBindingKey(target.ProviderName, target.ModelPresetID)
			if existing, found := byIdentity[key]; found && existing != target {
				return nil, fmt.Errorf(
					"%w: built-in Model Preset identity %q is inconsistent",
					basespec.ErrIdentityConflict,
					key,
				)
			}
			byIdentity[key] = target

			name := basespec.LogicalName(model.Name)
			if err := name.Validate(); err == nil {
				output[name] = appendUniqueTarget(output[name], target)
			}
		}
	}

	seenBindings := make(map[basespec.LogicalName]struct{}, len(bindings))
	for index, binding := range bindings {
		if err := binding.Validate(); err != nil {
			return nil, fmt.Errorf("model fallback binding %d: %w", index, err)
		}
		if _, duplicate := seenBindings[binding.Name]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate Model fallback binding %q",
				basespec.ErrIdentityConflict,
				binding.Name,
			)
		}
		seenBindings[binding.Name] = struct{}{}

		target, found := byIdentity[targetBindingKey(
			binding.ProviderName,
			binding.ModelPresetID,
		)]
		if !found {
			return nil, fmt.Errorf(
				"%w: Model fallback binding %q references unknown built-in preset %q/%q",
				basespec.ErrInvalid,
				binding.Name,
				binding.ProviderName,
				binding.ModelPresetID,
			)
		}

		// Explicit bindings intentionally replace automatic bindings for this
		// logical name, including duplicate automatic model-name candidates.
		output[binding.Name] = []TargetV1{target}
	}

	for name := range output {
		sort.Slice(output[name], func(left, right int) bool {
			a := output[name][left]
			b := output[name][right]

			if a.ProviderName != b.ProviderName {
				return string(a.ProviderName) < string(b.ProviderName)
			}
			if a.ModelPresetID != b.ModelPresetID {
				return string(a.ModelPresetID) < string(b.ModelPresetID)
			}
			return string(a.ModelName) < string(b.ModelName)
		})
	}

	return output, nil
}

func appendUniqueTarget(
	values []TargetV1,
	value TargetV1,
) []TargetV1 {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

func (s *Service) targetForName(
	name basespec.LogicalName,
) (TargetV1, bool, error) {
	values, found := s.byName[name]
	if !found || len(values) == 0 {
		return TargetV1{}, false, nil
	}
	if len(values) != 1 {
		return TargetV1{}, true, fmt.Errorf(
			"%w: built-in Model name %q maps to %d Model Presets",
			basespec.ErrIdentityConflict,
			name,
			len(values),
		)
	}
	return values[0], true, nil
}

func (s *Service) resolveLiveTarget(
	ctx context.Context,
	name basespec.LogicalName,
	target TargetV1,
) (modelpresetSpec.ModelPresetRef, error) {
	if err := target.Validate(); err != nil {
		return modelpresetSpec.ModelPresetRef{}, err
	}

	response, err := s.store.GetModelPreset(
		ctx,
		&modelpresetSpec.GetModelPresetRequest{
			ProviderName:    target.ProviderName,
			ModelPresetID:   target.ModelPresetID,
			IncludeDisabled: true,
		},
	)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return modelpresetSpec.ModelPresetRef{}, ctxErr
		}
		return modelpresetSpec.ModelPresetRef{}, unavailableError(
			"built-in Model Preset %q/%q cannot be loaded: %v",
			target.ProviderName,
			target.ModelPresetID,
			err,
		)
	}
	if response == nil || response.Body == nil {
		return modelpresetSpec.ModelPresetRef{}, unavailableError(
			"built-in Model Preset %q/%q returned an empty response",
			target.ProviderName,
			target.ModelPresetID,
		)
	}

	provider := response.Body.Provider
	model := response.Body.Model
	if !provider.IsBuiltIn ||
		!provider.IsEnabled ||
		!model.IsBuiltIn ||
		!model.IsEnabled ||
		provider.Name != target.ProviderName ||
		model.ID != target.ModelPresetID ||
		model.Name != target.ModelName {
		return modelpresetSpec.ModelPresetRef{}, unavailableError(
			"built-in Model Preset %q/%q no longer matches its mapped target",
			target.ProviderName,
			target.ModelPresetID,
		)
	}

	if values, found := s.byName[name]; !found ||
		len(values) != 1 ||
		values[0] != target {
		return modelpresetSpec.ModelPresetRef{}, unavailableError(
			"built-in Model Preset %q/%q is no longer bound to %q",
			target.ProviderName,
			target.ModelPresetID,
			name,
		)
	}

	return modelpresetSpec.ModelPresetRef{
		ProviderName:  target.ProviderName,
		ModelPresetID: target.ModelPresetID,
	}, nil
}

func targetBindingKey(
	providerName spec.ProviderName,
	modelPresetID modelpreset.ModelPresetID,
) string {
	return string(providerName) + "\x00" + string(modelPresetID)
}

func validateFallbackRequest(
	request resolve.FallbackRequest,
) error {
	if err := request.RootID.Validate(); err != nil {
		return err
	}
	if err := request.Type.Validate(); err != nil {
		return err
	}
	if request.Type != declaration.TypeModel {
		return fmt.Errorf(
			"%w: Model fallback received type %q",
			basespec.ErrInvalid,
			request.Type,
		)
	}
	if err := request.Name.Validate(); err != nil {
		return err
	}
	return request.Scope.Validate()
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: Model fallback context is nil",
			basespec.ErrInvalid,
		)
	}
	return ctx.Err()
}

func unavailableError(
	format string,
	args ...any,
) error {
	return fmt.Errorf(
		"%w: %s",
		basespec.ErrReferenceUnresolved,
		fmt.Sprintf(format, args...),
	)
}
