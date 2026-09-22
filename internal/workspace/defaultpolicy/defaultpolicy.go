package defaultpolicy

import (
	"bytes"
	"fmt"
	"io/fs"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/textv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/workspacev1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

const (
	ProviderKey   = "workspace-default-policy"
	PolicyRoot    = "base"
	PolicyLocator = "workspace.yaml"
	PolicyID      = "default-workspace"
	PolicyVersion = "v1"
)

type Policy struct {
	ID            string                        `json:"policyID"`
	Version       string                        `json:"policyVersion"`
	Digest        cryptoutil.Digest             `json:"policyDigest"`
	RawYAML       []byte                        `json:"-"`
	CanonicalJSON []byte                        `json:"-"`
	Document      workspacev1.WorkspaceDocument `json:"-"`
}

func Load() (Policy, error) {
	packages, err := builtin.EmbeddedWorkspacePackages()
	if err != nil {
		return Policy{}, err
	}
	rawYAML, err := fs.ReadFile(packages, PolicyRoot+"/"+PolicyLocator)
	if err != nil {
		return Policy{}, err
	}
	canonical, err := yamlutil.CanonicalObjectJSON(rawYAML, basespec.MaxDefinitionBytes)
	if err != nil {
		return Policy{}, err
	}
	document, err := workspacev1.DecodeWorkspaceJSON(canonical)
	if err != nil {
		return Policy{}, err
	}
	if document.Name != PolicyID {
		return Policy{}, fmt.Errorf("%w: base policy name is %q", basespec.ErrInvalid, document.Name)
	}
	if err := validatePolicyDocument(document); err != nil {
		return Policy{}, err
	}
	output := Policy{
		ID:            PolicyID,
		Version:       PolicyVersion,
		Digest:        cryptoutil.DigestBytes(canonical),
		RawYAML:       append([]byte(nil), rawYAML...),
		CanonicalJSON: append([]byte(nil), canonical...),
		Document:      document,
	}
	return output, nil
}

func (p Policy) Validate() error {
	if p.ID != PolicyID || p.Version != PolicyVersion {
		return fmt.Errorf("%w: base policy identity is invalid", basespec.ErrInvalid)
	}
	if p.Document.Name != p.ID {
		return fmt.Errorf("%w: base policy document identity is invalid", basespec.ErrInvalid)
	}
	if err := cryptoutil.ValidateDigest(p.Digest); err != nil {
		return err
	}
	if err := validatePolicyDocument(p.Document); err != nil {
		return err
	}
	canonical, err := p.Document.CanonicalJSON()
	if err != nil {
		return err
	}
	if !bytes.Equal(canonical, p.CanonicalJSON) ||
		cryptoutil.DigestBytes(canonical) != p.Digest {
		return fmt.Errorf("%w: base policy bytes or digest do not match its document", basespec.ErrDigestMismatch)
	}
	if len(p.RawYAML) == 0 {
		return fmt.Errorf("%w: base policy YAML is empty", basespec.ErrInvalid)
	}
	return nil
}

func validatePolicyDocument(document workspacev1.WorkspaceDocument) error {
	for index, member := range document.Members {
		form, err := member.MemberForm()
		if err != nil {
			return fmt.Errorf("members[%d]: %w", index, err)
		}
		header := member.Header()
		if header.Type == declaration.TypeWorkspace {
			return fmt.Errorf("%w: members[%d] cannot contain workspace", basespec.ErrInvalid, index)
		}
		if header.Type == declaration.TypeText {
			if form != declaration.MemberContained {
				return fmt.Errorf("%w: members[%d] text must be contained", basespec.ErrInvalid, index)
			}
			if header.Locator == nil {
				return fmt.Errorf("%w: members[%d] text must be source-backed", basespec.ErrInvalid, index)
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
				return fmt.Errorf("%w: members[%d] text must use locator content", basespec.ErrInvalid, index)
			}
			if _, err := declaration.ResolveSourceRelativePathLocator(
				*text.Locator,
				basespec.Locator(PolicyLocator),
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
				return fmt.Errorf("%w: members[%d] named non-text must use scope builtin", basespec.ErrInvalid, index)
			}
		default:
			return fmt.Errorf("%w: members[%d] contained non-text is not allowed", basespec.ErrInvalid, index)
		}
	}
	return nil
}
