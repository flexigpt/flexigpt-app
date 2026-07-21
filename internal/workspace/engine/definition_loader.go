package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

// DefinitionLoader reads the primary workspace definition before discovery so
// its discovery preferences can be incorporated into the generated plan.
type DefinitionLoader struct {
	sources  sourceLookup
	registry *source.Registry
	decoder  *DefinitionDecoder
}

func NewDefinitionLoader(
	sources sourceLookup,
	registry *source.Registry,
) (*DefinitionLoader, error) {
	if sources == nil || registry == nil {
		return nil, fmt.Errorf(
			"%w: Workspace definition loader dependencies are incomplete",
			ErrInvalidWorkspace,
		)
	}
	return &DefinitionLoader{
		sources:  sources,
		registry: registry,
		decoder:  NewDefinitionDecoder(),
	}, nil
}

func (l *DefinitionLoader) Load(
	ctx context.Context,
	value Workspace,
) (DefinitionObservation, error) {
	if value.Data.PrimarySourceID == "" {
		return DefinitionObservation{}, nil
	}
	sourceValue, err := l.sources.Get(ctx, value.Data.PrimarySourceID)
	if err != nil {
		return DefinitionObservation{}, err
	}
	snapshot, err := l.registry.Open(ctx, sourceValue)
	if err != nil {
		return DefinitionObservation{}, err
	}
	defer snapshot.Close()

	observation := DefinitionObservation{
		SourceID:   sourceValue.ID,
		Generation: snapshot.Generation(),
	}
	entry, err := snapshot.Stat(ctx, DefinitionLocator)
	if errors.Is(err, artifactstore.ErrNotFound) {
		if err := snapshot.Confirm(ctx); err != nil {
			return DefinitionObservation{}, err
		}
		return observation, nil
	}
	if err != nil {
		return DefinitionObservation{}, err
	}
	if entry.SizeBytes > artifactstore.MaxDefinitionBodyBytes {
		return DefinitionObservation{}, fmt.Errorf(
			"%w: Workspace definition exceeds byte limit",
			ErrWorkspaceDefinitionInvalid,
		)
	}
	reader, err := snapshot.Open(ctx, DefinitionLocator)
	if err != nil {
		return DefinitionObservation{}, err
	}
	content, readErr := io.ReadAll(io.LimitReader(
		reader,
		artifactstore.MaxDefinitionBodyBytes+1,
	))
	closeErr := reader.Close()
	if readErr != nil {
		return DefinitionObservation{}, readErr
	}
	if closeErr != nil {
		return DefinitionObservation{}, closeErr
	}
	if len(content) > artifactstore.MaxDefinitionBodyBytes {
		return DefinitionObservation{}, ErrWorkspaceDefinitionInvalid
	}
	if err := snapshot.Confirm(ctx); err != nil {
		return DefinitionObservation{}, err
	}

	candidate := discovery.Candidate{
		Source:              sourceValue,
		Locator:             DefinitionLocator,
		SourceContentDigest: definition.DigestBytes(content),
		Content:             content,
	}
	decoded, diagnostics := l.decoder.Decode(ctx, candidate)
	if artifactstore.ContainsErrorDiagnostic(diagnostics) {
		return DefinitionObservation{}, fmt.Errorf(
			"%w: %s",
			ErrWorkspaceDefinitionInvalid,
			diagnostics[0].Message,
		)
	}
	if len(decoded) != 1 {
		return DefinitionObservation{}, ErrWorkspaceDefinitionInvalid
	}

	var document DefinitionDocument
	decoder := json.NewDecoder(bytes.NewReader(decoded[0].Definition.Body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return DefinitionObservation{}, err
	}
	observation.Preferences = document.Discovery
	return observation, nil
}
