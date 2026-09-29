package domain

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
)

// ArtifactNameReferenceLookupRoots returns the only Roots allowed for a
// Model-to-Provider or Provider-to-Model logical-name reference in v1.
//
// An unscoped reference searches the current Root first and then the
// protected built-in Root. A builtin-scoped reference searches only the
// protected built-in Root. Arbitrary cross-root references are intentionally
// not supported.
func ArtifactNameReferenceLookupRoots(
	reference declaration.ArtifactNameReference,
	currentRoot root.RootID,
	builtinRoot root.RootID,
) ([]root.RootID, error) {
	if err := reference.Validate(); err != nil {
		return nil, err
	}
	if err := currentRoot.Validate(); err != nil {
		return nil, err
	}

	switch reference.Scope {
	case declaration.LookupScopeBuiltin:
		if builtinRoot == "" {
			return nil, fmt.Errorf(
				"%w: built-in lookup is unavailable",
				basespec.ErrReferenceUnresolved,
			)
		}
		if err := builtinRoot.Validate(); err != nil {
			return nil, err
		}
		return []root.RootID{builtinRoot}, nil

	case "":
		output := make([]root.RootID, 0, 2)
		output = append(output, currentRoot)

		if builtinRoot == "" || builtinRoot == currentRoot {
			return output, nil
		}
		if err := builtinRoot.Validate(); err != nil {
			return nil, err
		}
		return append(output, builtinRoot), nil

	default:
		return nil, fmt.Errorf(
			"%w: unsupported Model reference scope %q",
			basespec.ErrInvalid,
			reference.Scope,
		)
	}
}
