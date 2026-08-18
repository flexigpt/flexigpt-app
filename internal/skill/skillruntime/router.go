package skillruntime

import (
	"context"
	"fmt"
	"sync"

	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
)

type ResolvedArtifactSkill struct {
	Artifact   artifact.ArtifactRef
	Collection collection.CollectionRef
	Definition agentskillsSpec.SkillDef
	Version    string
}

// ArtifactSkillLoader is implemented by feature adapters. It does not decide
// ownership from a durable reference shape. ArtifactRouter resolves ownership
// from the Artifact Record and its current Collection membership first.
type ArtifactSkillLoader interface {
	ResolveArtifactSkill(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (ResolvedArtifactSkill, error)

	ListCollectionSkills(
		ctx context.Context,
		ref collection.CollectionRef,
	) ([]ResolvedArtifactSkill, error)
}

type ArtifactRouter struct {
	artifacts   artifact.Reader
	collections collection.Reader

	mu      sync.RWMutex
	loaders map[basespec.CollectionKind]ArtifactSkillLoader
}

func NewArtifactRouter(
	artifacts artifact.Reader,
	collections collection.Reader,
) (*ArtifactRouter, error) {
	if artifacts == nil || collections == nil {
		return nil, fmt.Errorf(
			"%w: Artifact Skill router dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return &ArtifactRouter{
		artifacts:   artifacts,
		collections: collections,
		loaders:     map[basespec.CollectionKind]ArtifactSkillLoader{},
	}, nil
}

func (r *ArtifactRouter) Register(
	kind basespec.CollectionKind,
	loader ArtifactSkillLoader,
) error {
	if err := basespec.ValidateCollectionKind(kind); err != nil {
		return err
	}
	if loader == nil {
		return fmt.Errorf("%w: Artifact Skill loader is nil", basespec.ErrInvalid)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.loaders[kind]; exists {
		return fmt.Errorf(
			"%w: Artifact Skill loader already registered for collection kind %q",
			basespec.ErrConflict,
			kind,
		)
	}
	r.loaders[kind] = loader
	return nil
}

func (r *ArtifactRouter) ResolveArtifactSkill(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ResolvedArtifactSkill, error) {
	if err := ref.Validate(); err != nil {
		return ResolvedArtifactSkill{}, err
	}

	record, err := r.artifacts.Get(ctx, ref)
	if err != nil {
		return ResolvedArtifactSkill{}, err
	}
	collectionRef := collection.CollectionRef{
		RootID:       record.RootID,
		CollectionID: record.CollectionID,
	}
	collectionValue, err := r.collections.Get(ctx, collectionRef)
	if err != nil {
		return ResolvedArtifactSkill{}, err
	}
	loader, err := r.loader(collectionValue.Kind)
	if err != nil {
		return ResolvedArtifactSkill{}, err
	}

	value, err := loader.ResolveArtifactSkill(ctx, ref)
	if err != nil {
		return ResolvedArtifactSkill{}, err
	}
	if err := value.Validate(); err != nil {
		return ResolvedArtifactSkill{}, err
	}
	if value.Artifact != ref || value.Collection != collectionRef {
		return ResolvedArtifactSkill{}, fmt.Errorf(
			"%w: feature loader returned a runtime Skill for another Artifact or Collection",
			basespec.ErrInvalid,
		)
	}
	return value, nil
}

func (r *ArtifactRouter) ListCollectionSkills(
	ctx context.Context,
	ref collection.CollectionRef,
) ([]ResolvedArtifactSkill, error) {
	if err := ref.Validate(); err != nil {
		return nil, err
	}
	collectionValue, err := r.collections.Get(ctx, ref)
	if err != nil {
		return nil, err
	}
	loader, err := r.loader(collectionValue.Kind)
	if err != nil {
		return nil, err
	}

	values, err := loader.ListCollectionSkills(ctx, ref)
	if err != nil {
		return nil, err
	}
	seen := make(map[artifact.ArtifactRef]struct{}, len(values))
	for index, value := range values {
		if err := value.Validate(); err != nil {
			return nil, fmt.Errorf("runtime Skill %d: %w", index, err)
		}
		if value.Collection != ref {
			return nil, fmt.Errorf(
				"%w: feature loader returned a runtime Skill from another Collection",
				basespec.ErrInvalid,
			)
		}
		record, err := r.artifacts.Get(ctx, value.Artifact)
		if err != nil {
			return nil, fmt.Errorf(
				"read runtime Skill %d Artifact: %w",
				index,
				err,
			)
		}
		if record.RootID != ref.RootID || record.CollectionID != ref.CollectionID {
			return nil, fmt.Errorf(
				"%w: feature loader returned an Artifact outside the requested Collection",
				basespec.ErrInvalid,
			)
		}
		if _, duplicate := seen[value.Artifact]; duplicate {
			return nil, fmt.Errorf(
				"%w: feature loader returned duplicate runtime Artifact %q",
				basespec.ErrInvalid,
				value.Artifact.ArtifactID,
			)
		}
		seen[value.Artifact] = struct{}{}
	}
	return values, nil
}

func (s ResolvedArtifactSkill) Validate() error {
	if err := s.Artifact.Validate(); err != nil {
		return err
	}
	if err := s.Collection.Validate(); err != nil {
		return err
	}
	if s.Artifact.RootID != s.Collection.RootID {
		return fmt.Errorf(
			"%w: skill Artifact and Collection belong to different roots",
			basespec.ErrInvalid,
		)
	}
	if s.Definition.Type == "" ||
		s.Definition.Name == "" ||
		s.Definition.Location == "" {
		return fmt.Errorf(
			"%w: runtime Skill definition is incomplete",
			basespec.ErrInvalid,
		)
	}
	if s.Version == "" {
		return fmt.Errorf(
			"%w: runtime Skill version is required",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func (r *ArtifactRouter) loader(
	kind basespec.CollectionKind,
) (ArtifactSkillLoader, error) {
	r.mu.RLock()
	loader, exists := r.loaders[kind]
	r.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf(
			"%w: no Skill runtime feature adapter owns collection kind %q",
			basespec.ErrReferenceUnresolved,
			kind,
		)
	}
	return loader, nil
}
