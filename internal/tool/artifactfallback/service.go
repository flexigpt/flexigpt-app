package artifactfallback

import (
	"context"
	"fmt"
	"slices"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
	toolStore "github.com/flexigpt/flexigpt-app/internal/tool/store"
)

// Service resolves legacy built-in ToolStore Tools as mapped Artifact targets
// and translates mapped targets back into legacy ToolStore references.
type Service struct {
	store *toolStore.ToolStore

	// Immutable after construction. Multiple entries for one name mean that
	// Tool versions are ambiguous until an explicit binding layer is added.
	byName map[basespec.LogicalName][]TargetV1
}

type ResolveTargetRequest struct {
	Target resolve.MappedTarget `json:"target" required:"true"`
}

type ResolveTargetResponseBody struct {
	ToolRef toolSpec.ToolRef `json:"toolRef"`
}

type ResolveTargetResponse struct {
	Body *ResolveTargetResponseBody
}

func NewService(
	ctx context.Context,
	store *toolStore.ToolStore,
) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf(
			"%w: Tool fallback ToolStore is nil",
			basespec.ErrInvalid,
		)
	}
	if err := validateContext(ctx); err != nil {
		return nil, err
	}

	byName, err := buildIndex(ctx, store)
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

// ResolveTarget translates a Tool mapped target into the legacy ToolRef used
// by existing ToolStore, inference, and conversation flows.
func (s *Service) ResolveTarget(
	ctx context.Context,
	target resolve.MappedTarget,
) (toolSpec.ToolRef, error) {
	if s == nil || s.store == nil {
		return toolSpec.ToolRef{}, basespec.ErrClosed
	}
	if err := validateContext(ctx); err != nil {
		return toolSpec.ToolRef{}, err
	}

	value, err := DecodeTarget(target)
	if err != nil {
		return toolSpec.ToolRef{}, err
	}

	indexed, found, err := s.targetForName(target.Name)
	if err != nil {
		return toolSpec.ToolRef{}, err
	}
	if !found {
		return toolSpec.ToolRef{}, unavailableError(
			"Tool mapped target %q is no longer registered",
			target.Name,
		)
	}
	if indexed != value {
		return toolSpec.ToolRef{}, unavailableError(
			"Tool mapped target %q no longer matches its built-in binding",
			target.Name,
		)
	}

	return s.resolveLiveTarget(ctx, target.Name, value)
}

func buildIndex(
	ctx context.Context,
	store *toolStore.ToolStore,
) (map[basespec.LogicalName][]TargetV1, error) {
	items, err := store.ListBuiltInTools(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf(
			"%w: Tool fallback found no built-in Tools",
			basespec.ErrReferenceUnresolved,
		)
	}

	output := make(map[basespec.LogicalName][]TargetV1)
	for _, item := range items {
		if !item.IsBuiltIn || !item.ToolDefinition.IsBuiltIn {
			continue
		}

		tool := item.ToolDefinition
		if item.BundleID == "" ||
			item.ToolSlug != tool.Slug ||
			item.ToolVersion != tool.Version {
			return nil, fmt.Errorf(
				"%w: built-in Tool list item identity is inconsistent",
				basespec.ErrInvalid,
			)
		}

		target := TargetV1{
			BundleID:    item.BundleID,
			ToolID:      tool.ID,
			ToolSlug:    tool.Slug,
			ToolVersion: tool.Version,
		}
		if err := target.Validate(); err != nil {
			return nil, fmt.Errorf(
				"invalid built-in Tool %q: %w",
				tool.Slug,
				err,
			)
		}

		name := basespec.LogicalName(tool.Slug)
		if err := name.Validate(); err != nil {
			return nil, err
		}

		output[name] = appendUniqueTarget(output[name], target)
	}

	for name := range output {
		sort.Slice(output[name], func(left, right int) bool {
			a := output[name][left]
			b := output[name][right]

			if a.BundleID != b.BundleID {
				return string(a.BundleID) < string(b.BundleID)
			}
			if a.ToolSlug != b.ToolSlug {
				return string(a.ToolSlug) < string(b.ToolSlug)
			}
			if a.ToolVersion != b.ToolVersion {
				return string(a.ToolVersion) < string(b.ToolVersion)
			}
			return string(a.ToolID) < string(b.ToolID)
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
			"%w: built-in Tool name %q maps to %d Tool versions",
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
) (toolSpec.ToolRef, error) {
	if err := target.Validate(); err != nil {
		return toolSpec.ToolRef{}, err
	}

	bundle, isBuiltIn, err := s.store.GetAnyToolBundle(ctx, target.BundleID)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return toolSpec.ToolRef{}, ctxErr
		}
		return toolSpec.ToolRef{}, unavailableError(
			"built-in Tool bundle %q cannot be loaded: %v",
			target.BundleID,
			err,
		)
	}
	if !isBuiltIn ||
		!bundle.IsBuiltIn ||
		bundle.ID != target.BundleID ||
		!bundle.IsEnabled {
		return toolSpec.ToolRef{}, unavailableError(
			"built-in Tool bundle %q is unavailable",
			target.BundleID,
		)
	}

	response, err := s.store.GetTool(ctx, &toolSpec.GetToolRequest{
		BundleID: target.BundleID,
		ToolSlug: target.ToolSlug,
		Version:  target.ToolVersion,
	})
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return toolSpec.ToolRef{}, ctxErr
		}
		return toolSpec.ToolRef{}, unavailableError(
			"built-in Tool %q/%q@%q cannot be loaded: %v",
			target.BundleID,
			target.ToolSlug,
			target.ToolVersion,
			err,
		)
	}
	if response == nil || response.Body == nil {
		return toolSpec.ToolRef{}, unavailableError(
			"built-in Tool %q/%q@%q returned an empty response",
			target.BundleID,
			target.ToolSlug,
			target.ToolVersion,
		)
	}

	tool := response.Body
	if !tool.IsBuiltIn ||
		!tool.IsEnabled ||
		tool.ID != target.ToolID ||
		tool.Slug != target.ToolSlug ||
		tool.Version != target.ToolVersion ||
		string(name) != string(tool.Slug) {
		return toolSpec.ToolRef{}, unavailableError(
			"built-in Tool %q/%q@%q no longer matches its mapped target",
			target.BundleID,
			target.ToolSlug,
			target.ToolVersion,
		)
	}

	return toolSpec.ToolRef{
		BundleID:    target.BundleID,
		ToolSlug:    target.ToolSlug,
		ToolVersion: target.ToolVersion,
	}, nil
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
	if request.Type != declaration.TypeTool {
		return fmt.Errorf(
			"%w: Tool fallback received type %q",
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
			"%w: Tool fallback context is nil",
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
