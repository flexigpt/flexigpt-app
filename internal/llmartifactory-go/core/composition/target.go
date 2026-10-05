package composition

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

type TargetForm string

const (
	TargetFormArtifact TargetForm = "artifact"
	TargetFormDirect   TargetForm = "direct"
)

type TargetProvenance string

const (
	TargetProvenanceCurrentRoot       TargetProvenance = "currentRoot"
	TargetProvenanceBuiltinScope      TargetProvenance = "builtinScope"
	TargetProvenanceCompositionSource TargetProvenance = "compositionSource"
	TargetProvenanceDirectCapability  TargetProvenance = "directCapability"
)

// CapabilityTarget is a runtime-neutral resolved target.
//
// Artifact targets preserve the actual Artifact identity selected by normal
// catalog lookup. Direct targets are application-supplied capabilities and
// deliberately contain no fabricated Artifact, Source, Definition, local
// state, or lifecycle fields.
type CapabilityTarget struct {
	Form       TargetForm                 `json:"form"`
	Type       declaration.Type           `json:"type"`
	Name       spec.LogicalName           `json:"name"`
	Provenance TargetProvenance           `json:"provenance"`
	Artifact   *artifactModel.ArtifactRef `json:"artifact,omitempty"`

	ProviderIdentity string            `json:"providerIdentity,omitempty"`
	ProviderLocalID  string            `json:"providerLocalID,omitempty"`
	Evidence         cryptoutil.Digest `json:"evidence,omitempty"`
}

func (t CapabilityTarget) Validate() error {
	if err := t.Type.Validate(); err != nil {
		return err
	}
	if err := t.Name.Validate(); err != nil {
		return err
	}

	switch t.Form {
	case TargetFormArtifact:
		if t.Artifact == nil {
			return fmt.Errorf(
				"%w: Artifact capability target requires an Artifact reference",
				spec.ErrInvalid,
			)
		}
		if err := t.Artifact.Validate(); err != nil {
			return err
		}
		if t.ProviderIdentity != "" ||
			t.ProviderLocalID != "" ||
			t.Evidence != "" {
			return fmt.Errorf(
				"%w: Artifact capability target cannot contain direct-capability evidence",
				spec.ErrInvalid,
			)
		}
		switch t.Provenance {
		case TargetProvenanceCurrentRoot,
			TargetProvenanceBuiltinScope,
			TargetProvenanceCompositionSource:
		default:
			return fmt.Errorf(
				"%w: Artifact capability target has invalid provenance %q",
				spec.ErrInvalid,
				t.Provenance,
			)
		}

	case TargetFormDirect:
		if t.Artifact != nil {
			return fmt.Errorf(
				"%w: direct capability target cannot contain an Artifact reference",
				spec.ErrInvalid,
			)
		}
		if t.Provenance != TargetProvenanceDirectCapability {
			return fmt.Errorf(
				"%w: direct capability target has invalid provenance %q",
				spec.ErrInvalid,
				t.Provenance,
			)
		}
		if err := spec.ValidateIdentifier(
			"direct capability provider identity",
			t.ProviderIdentity,
			spec.MaxKindBytes,
		); err != nil {
			return err
		}
		if err := spec.ValidateRequiredText(
			"direct capability provider-local identifier",
			t.ProviderLocalID,
			spec.MaxURIBytes,
		); err != nil {
			return err
		}
		if err := cryptoutil.ValidateDigest(t.Evidence); err != nil {
			return err
		}

	default:
		return fmt.Errorf(
			"%w: unsupported capability target form %q",
			spec.ErrInvalid,
			t.Form,
		)
	}
	return nil
}

func (t CapabilityTarget) Clone() CapabilityTarget {
	output := t
	if t.Artifact != nil {
		value := *t.Artifact
		output.Artifact = &value
	}
	return output
}

type DirectCapabilityRequest struct {
	RootID rootModel.RootID

	Type  declaration.Type
	Name  spec.LogicalName
	Scope declaration.LookupScope
}

// DirectCapabilityProvider supplies a runtime-neutral target only after normal
// Artifact lookup did not resolve a source-backed target.
type DirectCapabilityProvider interface {
	ProviderIdentity() string

	ResolveDirectCapability(
		ctx context.Context,
		request DirectCapabilityRequest,
	) (CapabilityTarget, bool, error)
}

type ArtifactCapabilityRequest struct {
	Artifact artifactModel.Artifact
	Type     declaration.Type
}

// ArtifactCapabilityProjector can preserve an Artifact target or translate it
// into an explicitly direct target with provider-owned identity and evidence.
// A handled Artifact target must retain the original ArtifactRef.
type ArtifactCapabilityProjector interface {
	ProjectArtifactCapability(
		ctx context.Context,
		request ArtifactCapabilityRequest,
	) (CapabilityTarget, bool, error)
}

func pointerTarget(value CapabilityTarget) *CapabilityTarget {
	copyValue := value.Clone()
	return &copyValue
}
