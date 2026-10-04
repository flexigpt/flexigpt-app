package compose

func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	s.closeOnce.Do(func() {
		if s.shutdown != nil {
			s.closeErr = s.shutdown()
			s.shutdown = nil
		}
	})
	return s.closeErr
}
