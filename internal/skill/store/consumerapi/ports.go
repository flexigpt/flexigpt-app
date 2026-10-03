package consumerapi

import (
	"context"
	"fmt"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/collection"
)

type BaselineEnsurer interface {
	EnsureSkillBaselineCollection(
		ctx context.Context,
		rootID rootModel.RootID,
	) (collection.CollectionView, error)
}

type baselineEnsurer struct {
	api *API
}

func NewBaselineEnsurer(api *API) (BaselineEnsurer, error) {
	if api == nil || api.collections == nil {
		return nil, fmt.Errorf(
			"%w: Skill baseline ensurer requires collections",
			spec.ErrInvalid,
		)
	}
	return &baselineEnsurer{api: api}, nil
}

func (s *baselineEnsurer) EnsureSkillBaselineCollection(
	ctx context.Context,
	rootID rootModel.RootID,
) (collection.CollectionView, error) {
	if s == nil || s.api == nil {
		return collection.CollectionView{}, spec.ErrClosed
	}
	return s.api.ensureSkillBaselineCollection(ctx, rootID)
}
