package artifact

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type State string

const (
	StateAvailable    State = "available"
	StateMissing      State = "missing"
	StateInvalid      State = "invalid"
	StateIncompatible State = "incompatible"
)

func (s State) Validate(
	resolvedDefinition *cryptoutil.Digest,
	sourceContentDigest *cryptoutil.Digest,
) error {
	if resolvedDefinition != nil {
		if err := cryptoutil.ValidateDigest(*resolvedDefinition); err != nil {
			return err
		}
	}
	if sourceContentDigest != nil {
		if err := cryptoutil.ValidateDigest(*sourceContentDigest); err != nil {
			return err
		}
	}

	switch s {
	case StateAvailable:
		if resolvedDefinition == nil ||
			sourceContentDigest == nil {
			return fmt.Errorf(
				"%w: available Artifact requires Definition and source content digests",
				model.ErrInvalid,
			)
		}

	case StateIncompatible:
		if resolvedDefinition == nil ||
			sourceContentDigest == nil {
			return fmt.Errorf(
				"%w: incompatible Artifact requires observed Definition and source content digests",
				model.ErrInvalid,
			)
		}

	case StateMissing:
		if resolvedDefinition != nil ||
			sourceContentDigest != nil {
			return fmt.Errorf(
				"%w: missing Artifact cannot retain current source state",
				model.ErrInvalid,
			)
		}

	case StateInvalid:
		if resolvedDefinition != nil {
			return fmt.Errorf(
				"%w: invalid Artifact cannot retain a resolved Definition",
				model.ErrInvalid,
			)
		}

	default:
		return fmt.Errorf(
			"%w: invalid Artifact state %q",
			model.ErrInvalid,
			s,
		)
	}
	return nil
}
