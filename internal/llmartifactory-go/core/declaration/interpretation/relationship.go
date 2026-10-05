package interpretation

import (
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

type Relationship struct {
	Path []string

	Declared declaration.Entry
	Form     declaration.MemberForm
	Required bool

	ContainedPath []string
}

type MemberOptions struct {
	AllowSelector bool
	Optional      bool

	// DirectPosition omits the target type from contained relationship paths.
	// Loop/workflow slots use this because their position already determines
	// the required family type.
	DirectPosition bool
}

func MemberRelationships(
	path []string,
	values []declaration.Entry,
	options MemberOptions,
) ([]Relationship, error) {
	output := make([]Relationship, 0, len(values))
	for index, value := range values {
		relationship, err := NewMemberRelationship(
			path,
			value,
			options,
		)
		if err != nil {
			return nil, fmt.Errorf("member %d: %w", index, err)
		}
		output = append(output, relationship)
	}
	return output, nil
}

func NewMemberRelationship(
	path []string,
	member declaration.Entry,
	options MemberOptions,
) (Relationship, error) {
	if err := declarationPath(path); err != nil {
		return Relationship{}, err
	}
	form, err := member.MemberForm()
	if err != nil {
		return Relationship{}, err
	}
	if form == declaration.MemberSelector && !options.AllowSelector {
		return Relationship{}, fmt.Errorf(
			"%w: this declaration relationship does not permit selectors",
			spec.ErrInvalid,
		)
	}

	header := member.Header()
	relationshipPath := append([]string(nil), path...)
	containedPath := append([]string(nil), path...)

	switch form {
	case declaration.MemberSelector:
		identity, err := shortIdentity(member)
		if err != nil {
			return Relationship{}, err
		}
		relationshipPath = append(
			relationshipPath,
			"selector-"+identity,
		)

	case declaration.MemberNamed, declaration.MemberContained:
		if options.DirectPosition {
			relationshipPath = append(relationshipPath, header.Name)
			containedPath = append(containedPath, header.Name)
		} else {
			relationshipPath = append(
				relationshipPath,
				string(header.Type),
			)
			containedPath = append(
				containedPath,
				string(header.Type),
			)

			if insert, err := member.TextInsert(); err == nil {
				relationshipPath = append(
					relationshipPath,
					string(insert),
				)
				containedPath = append(
					containedPath,
					string(insert),
				)
			}

			relationshipPath = append(relationshipPath, header.Name)
			containedPath = append(containedPath, header.Name)
		}

		if form == declaration.MemberNamed {
			identity, err := shortIdentity(member)
			if err != nil {
				return Relationship{}, err
			}
			relationshipPath = append(relationshipPath, identity)
			containedPath = nil
		}

	default:
		return Relationship{}, fmt.Errorf(
			"%w: unsupported declaration member form %q",
			spec.ErrInvalid,
			form,
		)
	}

	output := Relationship{
		Path:     relationshipPath,
		Declared: member.Clone(),
		Form:     form,
		Required: !options.Optional,
	}
	if form == declaration.MemberContained {
		output.ContainedPath = containedPath
	}
	if err := output.Validate(); err != nil {
		return Relationship{}, err
	}
	return output, nil
}

func (r Relationship) Clone() Relationship {
	output := r
	output.Path = append([]string(nil), r.Path...)
	output.Declared = r.Declared.Clone()
	output.ContainedPath = append([]string(nil), r.ContainedPath...)
	return output
}

func (r Relationship) Validate() error {
	if len(r.Path) == 0 {
		return fmt.Errorf(
			"%w: declaration relationship path is empty",
			spec.ErrInvalid,
		)
	}
	if err := declarationPath(r.Path); err != nil {
		return err
	}
	if err := r.Declared.Validate(); err != nil {
		return err
	}
	form, err := r.Declared.MemberForm()
	if err != nil {
		return err
	}
	if form != r.Form {
		return fmt.Errorf(
			"%w: declaration relationship form differs from declaration member",
			spec.ErrInvalid,
		)
	}
	if r.Form == declaration.MemberContained {
		if len(r.ContainedPath) == 0 {
			return fmt.Errorf(
				"%w: contained declaration relationship has no subresource path",
				spec.ErrInvalid,
			)
		}
		return declarationPath(r.ContainedPath)
	}
	if len(r.ContainedPath) != 0 {
		return fmt.Errorf(
			"%w: non-contained relationship has a subresource path",
			spec.ErrInvalid,
		)
	}
	return nil
}

func declarationPath(
	values []string,
) error {
	if len(values) == 0 {
		return fmt.Errorf(
			"%w: declaration relationship path is empty",
			spec.ErrInvalid,
		)
	}
	value := spec.SubresourceLocator(strings.Join(values, "/"))
	return value.Validate()
}
