package composition

import (
	"context"
	"fmt"

	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

func (r *Resolver) resolveStructure(
	ctx context.Context,
	state *resolutionState,
	node *ResolvedEntry,
	entry declaration.Entry,
	depth int,
) error {
	if node == nil || node.Artifact == nil || node.Definition == nil {
		return fmt.Errorf(
			"%w: source-backed declaration has no Artifact origin",
			storeSpec.ErrInvalid,
		)
	}
	rootID, found := node.RootID()
	if !found {
		return fmt.Errorf("%w: declaration has no Root scope", storeSpec.ErrInvalid)
	}
	// "resolveArtifact" obtained entry through EntryFromDefinition. Generic
	// Definition integrity and family admitted semantic validation already
	// completed. Extract relationship facts without repeating that work.
	key := schemaModel.ArtifactKey(
		node.Definition.Kind,
		node.Definition.SchemaID,
		node.Definition.SchemaVersion,
	)
	facts, err := r.interpretations.AdmittedRelationshipsForValidatedSchema(
		key,
		entry,
	)
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
