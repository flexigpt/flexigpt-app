package resource

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// ResolveOptions is reserved for future resource projection options.
// Declaration source verification is unconditional.
type ResolveOptions struct{}

// ResolvedArtifact contains the verified current Store resource chain:
//
//	Artifact -> Definition -> Source binding -> Source refresh state -> Source
//
// Typed graph expansion remains outside this generic Store value.
type ResolvedArtifact struct {
	Artifact     artifact.Artifact     `json:"artifact"`
	Definition   definition.Definition `json:"definition"`
	Source       source.Summary        `json:"source"`
	RefreshState source.RefreshState   `json:"refreshState"`
}

func (r ResolvedArtifact) Validate() error {
	if err := r.Artifact.Validate(); err != nil {
		return err
	}
	if err := r.Definition.Validate(); err != nil {
		return err
	}
	if err := r.Source.Validate(); err != nil {
		return err
	}
	if err := r.RefreshState.Validate(); err != nil {
		return err
	}
	if r.Artifact.State != artifact.StateAvailable ||
		r.Artifact.ResolvedDefinition == nil ||
		r.Artifact.SourceContentDigest == nil {
		return fmt.Errorf(
			"%w: Artifact has no current available source state",
			basespec.ErrReferenceUnresolved,
		)
	}
	if r.Artifact.RootID != r.Source.RootID ||
		r.Artifact.RootID != r.RefreshState.RootID ||
		r.Artifact.Binding.SourceID != r.Source.ID ||
		r.Artifact.Binding.SourceID != r.RefreshState.SourceID {
		return fmt.Errorf(
			"%w: resolved Artifact Source does not match Artifact binding",
			basespec.ErrInvalid,
		)
	}
	if r.Source.Revision != r.RefreshState.SourceRevision {
		return fmt.Errorf(
			"%w: resolved Source revision does not match refresh state",
			basespec.ErrRefreshRequired,
		)
	}
	if r.Definition.Kind != r.Artifact.Kind ||
		r.Definition.Digest != *r.Artifact.ResolvedDefinition ||
		r.Definition.LogicalName != r.Artifact.LogicalName ||
		r.Definition.LogicalVersion != r.Artifact.LogicalVersion {
		return fmt.Errorf(
			"%w: resolved Definition does not match Artifact state",
			basespec.ErrDigestMismatch,
		)
	}
	return nil
}

func (r ResolvedArtifact) Clone() ResolvedArtifact {
	output := r
	output.Artifact = r.Artifact.Clone()
	output.Definition = r.Definition.Clone()
	output.Source = r.Source.Clone()
	output.RefreshState = r.RefreshState.Clone()
	return output
}

// VerifiedEntry is a bounded, generation-confirmed Source read. It can be
// used by consumers that need source material without creating an Artifact.
type VerifiedEntry struct {
	RootID           root.RootID       `json:"rootID"`
	SourceID         source.SourceID   `json:"sourceID"`
	Locator          basespec.Locator  `json:"locator"`
	SourceRevision   uint64            `json:"sourceRevision"`
	SourceGeneration string            `json:"sourceGeneration"`
	Content          []byte            `json:"content"`
	Digest           cryptoutil.Digest `json:"digest"`
}

func (e VerifiedEntry) Validate() error {
	if err := e.RootID.Validate(); err != nil {
		return err
	}
	if err := e.SourceID.Validate(); err != nil {
		return err
	}
	if err := e.Locator.Validate(false); err != nil {
		return err
	}
	if e.SourceRevision == 0 {
		return fmt.Errorf(
			"%w: verified entry Source revision is required",
			basespec.ErrInvalid,
		)
	}
	if err := basespec.ValidateSourceGeneration(
		e.SourceGeneration,
	); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(e.Digest); err != nil {
		return err
	}
	if cryptoutil.DigestBytes(e.Content) != e.Digest {
		return fmt.Errorf(
			"%w: verified entry content does not match digest",
			basespec.ErrDigestMismatch,
		)
	}
	return nil
}

func (e VerifiedEntry) Clone() VerifiedEntry {
	output := e
	output.Content = append([]byte(nil), e.Content...)
	return output
}
