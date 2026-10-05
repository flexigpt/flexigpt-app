package defaultpolicy

import (
	"fmt"
	"io/fs"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	textv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text/contract/v1"
	workspacev1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/contract/v1"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/domain"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

const (
	ProviderKey   = "workspace-default-policy"
	PolicyRoot    = "base"
	PolicyLocator = "workspace.yaml"
	PolicyID      = "default-workspace"
	PolicyVersion = "v1"
)

func Load() (workspaceDomain.DefaultPolicy, error) {
	packages, err := artifactbuiltin.EmbeddedWorkspacePackages()
	if err != nil {
		return workspaceDomain.DefaultPolicy{}, err
	}
	rawYAML, err := fs.ReadFile(packages, PolicyRoot+"/"+PolicyLocator)
	if err != nil {
		return workspaceDomain.DefaultPolicy{}, err
	}
	canonical, err := yamlutil.CanonicalObjectJSON(rawYAML, spec.MaxDefinitionBytes)
	if err != nil {
		return workspaceDomain.DefaultPolicy{}, err
	}
	document, err := workspacev1.DecodeWorkspaceJSON(canonical)
	if err != nil {
		return workspaceDomain.DefaultPolicy{}, err
	}
	if document.Name != PolicyID {
		return workspaceDomain.DefaultPolicy{}, fmt.Errorf(
			"%w: base policy name is %q",
			spec.ErrInvalid,
			document.Name,
		)
	}
	if err := validatePolicyDocument(document); err != nil {
		return workspaceDomain.DefaultPolicy{}, err
	}
	output := workspaceDomain.DefaultPolicy{
		ID:            PolicyID,
		Version:       PolicyVersion,
		Digest:        cryptoutil.DigestBytes(canonical),
		RawYAML:       append([]byte(nil), rawYAML...),
		CanonicalJSON: append([]byte(nil), canonical...),
		Document:      document,
	}
	if err := output.Validate(); err != nil {
		return workspaceDomain.DefaultPolicy{}, err
	}
	return output, nil
}

func validatePolicyDocument(document workspacev1.WorkspaceDocument) error {
	for index, member := range document.Members {
		form, err := member.MemberForm()
		if err != nil {
			return fmt.Errorf("members[%d]: %w", index, err)
		}
		header := member.Header()
		if header.Type == declaration.TypeWorkspace {
			return fmt.Errorf("%w: members[%d] cannot contain workspace", spec.ErrInvalid, index)
		}
		if header.Type == declaration.TypeText {
			if form != declaration.MemberContained {
				return fmt.Errorf("%w: members[%d] text must be contained", spec.ErrInvalid, index)
			}
			if header.Locator == nil {
				return fmt.Errorf("%w: members[%d] text must be source-backed", spec.ErrInvalid, index)
			}
			target, err := member.ContainedDeclaration()
			if err != nil {
				return fmt.Errorf("members[%d]: %w", index, err)
			}
			text, err := textv1.DecodeTextEntry(target)
			if err != nil {
				return fmt.Errorf("members[%d]: %w", index, err)
			}
			if text.Content != nil || text.Locator == nil {
				return fmt.Errorf("%w: members[%d] text must use locator content", spec.ErrInvalid, index)
			}
			if _, err := declaration.ResolveSourceRelativePathLocator(
				*text.Locator,
				spec.Locator(PolicyLocator),
			); err != nil {
				return fmt.Errorf(
					"members[%d] text must use a local source-relative locator: %w",
					index,
					err,
				)
			}
			continue
		}
		switch form {
		case declaration.MemberSelector:
			continue
		case declaration.MemberNamed:
			relationship, err := member.Relationship()
			if err != nil {
				return fmt.Errorf("members[%d]: %w", index, err)
			}
			if relationship.Scope != declaration.LookupScopeBuiltin {
				return fmt.Errorf("%w: members[%d] named non-text must use scope builtin", spec.ErrInvalid, index)
			}
		default:
			return fmt.Errorf("%w: members[%d] contained non-text is not allowed", spec.ErrInvalid, index)
		}
	}
	return nil
}
