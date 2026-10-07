package llmartifactory

// Close intentionally does not close the borrowed generic Artifact Store or
// provider resources. LLM Artifactory owns no independently closable resource;
// deployment shutdown owns the generic Store after all consumers stop.
func (a *Artifactory) Close() error {
	return nil
}
