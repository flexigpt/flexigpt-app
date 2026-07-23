package skillstore

import (
	"context"
	"fmt"
)

func (s *SkillStore) withUserWrite(
	ctx context.Context,
	_ string,
	fn func(sc *skillStoreSchema) error,
) error {
	if s == nil {
		return fmt.Errorf("%w: nil store", errSkillInvalidRequest)
	}
	if fn == nil {
		return fmt.Errorf("%w: nil write function", errSkillInvalidRequest)
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	s.mu.RLock()
	snapshot, err := s.readAllUser(false)
	s.mu.RUnlock()
	if err != nil {
		return err
	}
	if err := fn(&snapshot); err != nil {
		return err
	}

	s.mu.Lock()
	err = s.writeAllUser(snapshot)
	s.mu.Unlock()
	return err
}
