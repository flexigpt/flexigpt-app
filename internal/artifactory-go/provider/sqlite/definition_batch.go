package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const (
	definitionGetBatchSize  = 256
	definitionCacheMaxBytes = 64 << 20
)

const getDefinitionsByRootSQL = `
	SELECT ` + definitionColumns + `
	FROM artifact_definitions
	WHERE root_id = ?
	  AND digest IN (SELECT value FROM json_each(?))`

func (s *Store) getDefinitions(
	ctx context.Context,
	keys []definitionModel.Key,
) ([]definitionModel.Definition, error) {
	if len(keys) == 0 {
		return []definitionModel.Definition{}, nil
	}

	roots := make(map[rootModel.RootID]struct{})
	unique := make(map[definitionModel.Key]struct{}, len(keys))
	for _, key := range keys {
		if err := key.Validate(); err != nil {
			return nil, err
		}
		roots[key.RootID] = struct{}{}
		unique[key] = struct{}{}
	}

	rootIDs := make([]rootModel.RootID, 0, len(roots))
	for rootID := range roots {
		rootIDs = append(rootIDs, rootID)
	}
	slices.Sort(rootIDs)
	for _, rootID := range rootIDs {
		if err := s.requireActiveRoot(ctx, rootID); err != nil {
			return nil, err
		}
	}

	found := make(map[definitionModel.Key]definitionModel.Definition, len(unique))
	missingByRoot := make(map[rootModel.RootID][]cryptoutil.Digest)
	for key := range unique {
		if value, foundInCache := s.cachedDefinition(key); foundInCache {
			found[key] = value
			continue
		}
		missingByRoot[key.RootID] = append(
			missingByRoot[key.RootID],
			key.Digest,
		)
	}

	for _, rootID := range rootIDs {
		digests := missingByRoot[rootID]
		for start := 0; start < len(digests); start += definitionGetBatchSize {
			end := min(start+definitionGetBatchSize, len(digests))

			raw, err := json.Marshal(digests[start:end])
			if err != nil {
				return nil, err
			}
			rows, err := s.db.QueryContext(
				ctx,
				getDefinitionsByRootSQL,
				string(rootID),
				string(raw),
			)
			if err != nil {
				return nil, err
			}

			for rows.Next() {
				value, err := scanDefinition(rows)
				if err != nil {
					//nolint:sqlclosecheck // Ok.
					rows.Close()
					return nil, err
				}
				key := definitionModel.Key{
					RootID: rootID,
					Digest: value.Digest,
				}
				s.rememberDefinition(key, value)
				found[key] = value
			}
			rowsErr := rows.Err()
			closeErr := rows.Close()
			if rowsErr != nil {
				return nil, rowsErr
			}
			if closeErr != nil {
				return nil, closeErr
			}
		}
	}

	output := make([]definitionModel.Definition, len(keys))
	for index, key := range keys {
		value, exists := found[key]
		if !exists {
			return nil, fmt.Errorf(
				"%w: Definition %q in Root %q",
				spec.ErrDefinitionNotFound,
				key.Digest,
				key.RootID,
			)
		}
		output[index] = value.Clone()
	}
	return output, nil
}

func (s *Store) cachedDefinition(
	key definitionModel.Key,
) (definitionModel.Definition, bool) {
	s.definitionMu.RLock()
	defer s.definitionMu.RUnlock()

	value, found := s.definitionCache[key]
	if !found {
		return definitionModel.Definition{}, false
	}
	// Private immutable view. GetDefinitions clones each outgoing result.
	// No caller of this helper may mutate the borrowed value.
	return value, true
}

func (s *Store) rememberDefinition(
	key definitionModel.Key,
	value definitionModel.Definition,
) {
	size := len(value.Body)
	if size > definitionCacheMaxBytes {
		return
	}

	s.definitionMu.Lock()
	defer s.definitionMu.Unlock()

	if _, found := s.definitionCache[key]; found {
		return
	}
	if s.definitionCache == nil ||
		s.definitionBytes+size > definitionCacheMaxBytes {
		s.definitionCache = make(
			map[definitionModel.Key]definitionModel.Definition,
		)
		s.definitionBytes = 0
	}

	s.definitionCache[key] = value.Clone()
	s.definitionBytes += size
}
