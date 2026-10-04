package sqlite

import (
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
)

// forgetDefinitionsForRoot removes immutable cached payloads after a Root is
// physically purged. Root liveness checks prevent reads while absent; this
// eviction also prevents stale Definition availability if the same Root ID is
// subsequently recreated.
func (s *Store) forgetDefinitionsForRoot(rootID rootModel.RootID) {
	if s == nil {
		return
	}
	s.definitionMu.Lock()
	defer s.definitionMu.Unlock()
	for key, value := range s.definitionCache {
		if key.RootID != rootID {
			continue
		}
		delete(s.definitionCache, key)
		s.definitionBytes -= len(value.Body)
	}
	if len(s.definitionCache) == 0 || s.definitionBytes < 0 {
		s.definitionCache = make(map[definitionModel.Key]definitionModel.Definition)
		s.definitionBytes = 0
	}
}
