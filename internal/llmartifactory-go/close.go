package llmartifactory

// Close releases only LLM-owned in-memory registrations. It intentionally
// does not close the borrowed generic Artifact Store or any provider resource.
func (a *Artifactory) Close() error {
	if a == nil {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.closed {
		return nil
	}
	a.closed = true
	a.schemaCodecs = nil
	a.decoders = nil
	a.locatorFactories = nil
	a.store = nil
	return nil
}
