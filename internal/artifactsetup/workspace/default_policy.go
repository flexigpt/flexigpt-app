package workspace

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/iofs"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/llmsupport"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coredecoder "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/decoder"
	textv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text/contract/v1"
	workspaceAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace"
	workspacev1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/contract/v1"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/domain"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

const (
	defaultPolicyProviderKey = "workspace-default-policy"
	defaultPolicyRoot        = "."
	defaultPolicyLocator     = "workspace.yaml"
	defaultPolicyID          = "default-workspace"
	defaultPolicyVersion     = "v1"
)

//go:embed workspace.yaml
var embeddedPolicyFile embed.FS

// DefaultPolicyProviderRegistrations returns the immutable embedded provider
// registrations required by the application default Workspace policy.
//
// It is exported only because internal/artifactsetup is a different Go
// package and passes the registrations to local Store assembly.
func DefaultPolicyProviderRegistrations(
	ctx context.Context,
) (map[string]iofs.ProviderRegistration, error) {
	source, err := defaultPolicySource()
	if err != nil {
		return nil, err
	}

	contentDigest, err := iofs.ContentDigest(
		ctx,
		embeddedPolicyFile,
	)
	if err != nil {
		return nil, err
	}

	return map[string]iofs.ProviderRegistration{
		source.ProviderKey: {
			Filesystem: embeddedPolicyFile,
			Immutable: &iofs.ImmutableContentEvidence{
				Revision:      source.Policy.Version,
				ContentDigest: contentDigest,
			},
		},
	}, nil
}

// DefaultWorkspaceConfig supplies the complete application-selected Workspace
// consumer configuration. Runtime-specific composition and MCP ports remain
// unset because wrappers bind those runtime capabilities explicitly.
func DefaultWorkspaceConfig() (
	workspaceAPI.Config,
	error,
) {
	source, err := defaultPolicySource()
	if err != nil {
		return workspaceAPI.Config{}, err
	}
	support, err := llmsupport.Workspace()
	if err != nil {
		return workspaceAPI.Config{}, err
	}
	config := workspaceAPI.DefaultConfig()
	config.DefaultPolicySource = source
	config.Support = support
	return config, nil
}

// defaultPolicySource binds application-shipped Workspace policy content to
// the explicit embedded Source configuration consumed by Workspace local
// deployment behavior.
func defaultPolicySource() (
	workspaceAPI.DefaultPolicySource,
	error,
) {
	policy, err := loadDefaultPolicy()
	if err != nil {
		return workspaceAPI.DefaultPolicySource{}, err
	}

	value := workspaceAPI.DefaultPolicySource{
		ProviderKey: defaultPolicyProviderKey,
		Root:        storeSpec.Locator(defaultPolicyRoot),
		Locator:     storeSpec.Locator(defaultPolicyLocator),
		DecoderID:   coredecoder.YAMLDecoderID,
		Policy:      policy,
	}
	if err := value.Validate(); err != nil {
		return workspaceAPI.DefaultPolicySource{}, err
	}
	return value, nil
}

// loadDefaultPolicy decodes the application-shipped Workspace default policy.
func loadDefaultPolicy() (workspaceDomain.DefaultPolicy, error) {
	rawYAML, err := fs.ReadFile(embeddedPolicyFile, defaultPolicyLocator)
	if err != nil {
		return workspaceDomain.DefaultPolicy{}, err
	}

	canonical, err := yamlutil.CanonicalObjectJSON(
		rawYAML,
		storeSpec.MaxDefinitionBytes,
	)
	if err != nil {
		return workspaceDomain.DefaultPolicy{}, err
	}

	document, err := workspacev1.DecodeWorkspaceJSON(canonical)
	if err != nil {
		return workspaceDomain.DefaultPolicy{}, err
	}
	if document.Name != defaultPolicyID {
		return workspaceDomain.DefaultPolicy{}, fmt.Errorf(
			"%w: base policy name is %q",
			storeSpec.ErrInvalid,
			document.Name,
		)
	}
	if err := validatePolicyDocument(document); err != nil {
		return workspaceDomain.DefaultPolicy{}, err
	}

	output := workspaceDomain.DefaultPolicy{
		ID:            defaultPolicyID,
		Version:       defaultPolicyVersion,
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

func validatePolicyDocument(
	document workspacev1.WorkspaceDocument,
) error {
	for index, member := range document.Members {
		form, err := member.MemberForm()
		if err != nil {
			return fmt.Errorf("members[%d]: %w", index, err)
		}

		header := member.Header()
		if header.Type == declaration.TypeWorkspace {
			return fmt.Errorf(
				"%w: members[%d] cannot contain workspace",
				storeSpec.ErrInvalid,
				index,
			)
		}

		if header.Type == declaration.TypeText {
			if form != declaration.MemberContained {
				return fmt.Errorf(
					"%w: members[%d] text must be contained",
					storeSpec.ErrInvalid,
					index,
				)
			}
			if header.Locator == nil {
				return fmt.Errorf(
					"%w: members[%d] text must be source-backed",
					storeSpec.ErrInvalid,
					index,
				)
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
				return fmt.Errorf(
					"%w: members[%d] text must use locator content",
					storeSpec.ErrInvalid,
					index,
				)
			}
			if _, err := declaration.ResolveSourceRelativePathLocator(
				*text.Locator,
				storeSpec.Locator(defaultPolicyLocator),
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
				return fmt.Errorf(
					"%w: members[%d] named non-text must use scope builtin",
					storeSpec.ErrInvalid,
					index,
				)
			}

		default:
			return fmt.Errorf(
				"%w: members[%d] contained non-text is not allowed",
				storeSpec.ErrInvalid,
				index,
			)
		}
	}
	return nil
}
