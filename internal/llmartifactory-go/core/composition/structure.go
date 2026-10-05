package composition

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

func (r *Resolver) resolveStructure(
	ctx context.Context,
	state *resolutionState,
	node *ResolvedEntry,
	entry declaration.Entry,
	depth int,
) error {
	if node == nil || node.Artifact == nil {
		return fmt.Errorf(
			"%w: source-backed declaration has no Artifact origin",
			spec.ErrInvalid,
		)
	}
	rootID, found := node.RootID()
	if !found {
		return fmt.Errorf("%w: declaration has no Root scope", spec.ErrInvalid)
	}
	facts, err := r.interpretations.Relationships(entry)
	if err != nil {
		return err
	}
	node.Relationships = make([]ResolvedRelationship, 0, len(facts))
	for _, fact := range facts {
		relationship, err := r.resolveRelationship(
			ctx,
			state,
			rootID,
			fact,
			node.Artifact,
			depth+1,
		)
		if err != nil {
			return err
		}
		node.Relationships = append(node.Relationships, relationship)
	}
	return nil
}
