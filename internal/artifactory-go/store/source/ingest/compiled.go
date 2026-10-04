package ingest

import (
	"context"
	"fmt"
	"maps"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const compiledDecoderID spec.DecoderID = "artifact.builtin-compiled"

// CompiledArtifact is trusted already-admitted evidence for one source
// subresource. It is owned by Ingest rather than an installation package plan.
type CompiledArtifact struct {
	Subresource spec.SubresourceLocator
	Definition  definitionModel.Definition
	Diagnostics []diagnostic.Diagnostic
}

func (a CompiledArtifact) Clone() CompiledArtifact {
	output := a
	output.Definition = a.Definition.Clone()
	output.Diagnostics = diagnostic.Clone(a.Diagnostics)
	return output
}

// CompiledDocument is a source-relative witness for generated content.
// Install is responsible for translating package-relative generated values to
// this shape; Ingest owns its registration and scan-time use.
type CompiledDocument struct {
	Locator   spec.Locator
	Digest    cryptoutil.Digest
	Artifacts []CompiledArtifact
}

func (d CompiledDocument) Validate() error {
	if err := d.Locator.Validate(false); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(d.Digest); err != nil {
		return err
	}
	if len(d.Artifacts) == 0 || len(d.Artifacts) > spec.MaxDiscoveryCandidates {
		return fmt.Errorf("%w: compiled document has an invalid artifact count", spec.ErrInvalid)
	}
	type typedSubresource struct {
		subresource spec.SubresourceLocator
		kind        string
	}
	seen := make(map[typedSubresource]struct{}, len(d.Artifacts))
	for index, artifact := range d.Artifacts {
		if err := artifact.Subresource.Validate(); err != nil {
			return fmt.Errorf("compiled artifacts[%d]: %w", index, err)
		}
		key := typedSubresource{artifact.Subresource, string(artifact.Definition.Kind)}
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf(
				"%w: compiled document repeats typed subresource %q",
				spec.ErrInvalid,
				artifact.Subresource,
			)
		}
		seen[key] = struct{}{}

		// Generated catalogs are trusted admission evidence. Their JSON body
		// may have been decoded from the generated payload and therefore need
		// not retain the exact canonical byte representation used to calculate
		// Definition.Digest. Do not call Definition.Validate here: that would
		// incorrectly reject a valid generated Definition solely because Body
		// has non-canonical object whitespace or key order.
		//
		// RegisterCompiledDocuments performs Definition-owned canonical
		// admission after this trusted witness shape has been accepted.
		if err := cryptoutil.ValidateDigest(artifact.Definition.Digest); err != nil {
			return fmt.Errorf(
				"compiled artifacts[%d] Definition digest: %w",
				index,
				err,
			)
		}
		if err := artifact.Definition.Kind.Validate(); err != nil {
			return fmt.Errorf("compiled artifacts[%d] Definition kind: %w", index, err)
		}
		if err := artifact.Definition.LogicalName.Validate(); err != nil {
			return fmt.Errorf("compiled artifacts[%d] Definition logical name: %w", index, err)
		}
		if err := artifact.Definition.LogicalVersion.Validate(true); err != nil {
			return fmt.Errorf("compiled artifacts[%d] Definition logical version: %w", index, err)
		}
		if err := diagnostic.Validate(artifact.Diagnostics); err != nil {
			return fmt.Errorf("compiled artifacts[%d] diagnostics: %w", index, err)
		}
	}
	return nil
}

func (d CompiledDocument) Clone() CompiledDocument {
	output := d
	output.Artifacts = make([]CompiledArtifact, len(d.Artifacts))
	for index, artifact := range d.Artifacts {
		output.Artifacts[index] = artifact.Clone()
	}
	return output
}

type compiledDocumentKey struct {
	rootID   rootModel.RootID
	sourceID sourceModel.SourceID
	locator  spec.Locator
}

type compiledDocumentRegistry struct {
	mu     sync.RWMutex
	values map[compiledDocumentKey]CompiledDocument
}

func (e *Engine) RegisterCompiledDocuments(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	documents []CompiledDocument,
) error {
	if e == nil {
		return spec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf("%w: compiled document registration context is nil", spec.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := rootID.Validate(); err != nil {
		return err
	}
	if err := sourceID.Validate(); err != nil {
		return err
	}
	pending := make(map[compiledDocumentKey]CompiledDocument, len(documents))
	for index, document := range documents {
		if err := document.Validate(); err != nil {
			return fmt.Errorf("compiled documents[%d]: %w", index, err)
		}
		key := compiledDocumentKey{rootID: rootID, sourceID: sourceID, locator: document.Locator}
		if _, duplicate := pending[key]; duplicate {
			return fmt.Errorf(
				"%w: compiled built-in catalog repeats source locator %q",
				spec.ErrIdentityConflict,
				document.Locator,
			)
		}
		admitted := document.Clone()
		for artifactIndex := range admitted.Artifacts {
			value, err := definition.Admit(admitted.Artifacts[artifactIndex].Definition)
			if err != nil {
				return fmt.Errorf("compiled documents[%d] artifacts[%d]: %w", index, artifactIndex, err)
			}
			admitted.Artifacts[artifactIndex].Definition = value
		}
		pending[key] = admitted
	}
	e.compiled.mu.Lock()
	defer e.compiled.mu.Unlock()
	if e.compiled.values == nil {
		e.compiled.values = make(map[compiledDocumentKey]CompiledDocument)
	}
	maps.Copy(e.compiled.values, pending)
	return nil
}

func (e *Engine) compiledDocument(
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	locator spec.Locator,
) (CompiledDocument, bool) {
	e.compiled.mu.RLock()
	defer e.compiled.mu.RUnlock()
	value, found := e.compiled.values[compiledDocumentKey{rootID: rootID, sourceID: sourceID, locator: locator}]
	if !found {
		return CompiledDocument{}, false
	}
	return value.Clone(), true
}
