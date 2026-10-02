package discovery

import (
	"fmt"
	"maps"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/api/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const compiledDecoderID model.DecoderID = "artifact.builtin-compiled"

type compiledDocumentKey struct {
	rootID   root.RootID
	sourceID source.SourceID
	locator  model.Locator
}

type compiledDocumentRegistry struct {
	mu     sync.RWMutex
	values map[compiledDocumentKey]topology.CompiledDocument
}

func (e *Engine) RegisterCompiledDocuments(
	rootID root.RootID,
	sourceID source.SourceID,
	packages []topology.CompiledPackage,
) error {
	if e == nil {
		return model.ErrClosed
	}

	pending := make(
		map[compiledDocumentKey]topology.CompiledDocument,
	)
	for _, packageValue := range packages {
		for _, document := range packageValue.Documents {
			locator, err := packageValue.Address.FileLocator(
				document.Locator,
			)
			if err != nil {
				return err
			}
			if err := cryptoutil.ValidateDigest(document.Digest); err != nil {
				return err
			}

			key := compiledDocumentKey{
				rootID:   rootID,
				sourceID: sourceID,
				locator:  locator,
			}
			if _, duplicate := pending[key]; duplicate {
				return fmt.Errorf(
					"%w: compiled built-in catalog repeats source locator %q",
					model.ErrIdentityConflict,
					locator,
				)
			}
			pending[key] = document.Clone()
		}
	}

	e.compiled.mu.Lock()
	defer e.compiled.mu.Unlock()

	if e.compiled.values == nil {
		e.compiled.values = make(
			map[compiledDocumentKey]topology.CompiledDocument,
		)
	}
	maps.Copy(e.compiled.values, pending)
	return nil
}

func (e *Engine) compiledDocument(
	rootID root.RootID,
	sourceID source.SourceID,
	locator model.Locator,
) (topology.CompiledDocument, bool) {
	e.compiled.mu.RLock()
	defer e.compiled.mu.RUnlock()

	value, found := e.compiled.values[compiledDocumentKey{
		rootID:   rootID,
		sourceID: sourceID,
		locator:  locator,
	}]
	if !found {
		return topology.CompiledDocument{}, false
	}
	// Private immutable view. Discovery clones emitted Definitions before
	// handing them to the rest of the refresh pipeline.
	return value, true
}
