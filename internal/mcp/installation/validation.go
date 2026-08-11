package installation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/schema"
	"github.com/flexigpt/flexigpt-app/internal/mcp/secret"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

var installationInputNamePattern = regexp.MustCompile(
	`^[A-Za-z_][A-Za-z0-9_]*$`,
)

// ValidateServerDataForDocument validates local installation data against the
// immutable canonical server semantics that own the input declarations.
func ValidateServerDataForDocument(
	server artifact.ArtifactRef,
	document schema.ServerDocument,
	data ServerData,
) error {
	if err := server.Validate(); err != nil {
		return err
	}
	if err := schema.ValidateServer(document); err != nil {
		return err
	}
	if err := ValidateServerData(data); err != nil {
		return err
	}

	if data.SelectedConnectionProfile != "" {
		if _, found := document.Extension.ConnectionProfiles[data.SelectedConnectionProfile]; !found {
			return fmt.Errorf(
				"%w: selected MCP connection profile %q does not exist",
				basespec.ErrReferenceUnresolved,
				data.SelectedConnectionProfile,
			)
		}
	}

	for name, binding := range data.Inputs {
		if !installationInputNamePattern.MatchString(name) {
			return fmt.Errorf(
				"%w: invalid MCP installation input name %q",
				basespec.ErrInvalid,
				name,
			)
		}

		declaration, declared := document.Extension.Install.Inputs[name]
		if !declared {
			return fmt.Errorf(
				"%w: MCP installation input %q is not declared by the server",
				basespec.ErrInvalid,
				name,
			)
		}

		switch declaration.Kind {
		case schema.InputText, schema.InputPath:
			if binding.SecretRef != "" {
				return fmt.Errorf(
					"%w: MCP input %q must use a local value, not a secret reference",
					basespec.ErrInvalid,
					name,
				)
			}
			if binding.Value == nil {
				return fmt.Errorf(
					"%w: MCP input %q requires a local value",
					basespec.ErrInvalid,
					name,
				)
			}

		case schema.InputSecret:
			if binding.Value != nil || strings.TrimSpace(binding.SecretRef) == "" {
				return fmt.Errorf(
					"%w: MCP secret input %q requires exactly one secret reference",
					basespec.ErrInvalid,
					name,
				)
			}
			if err := validateSecretBinding(
				server,
				binding.SecretRef,
				false,
			); err != nil {
				return fmt.Errorf("MCP secret input %q: %w", name, err)
			}

		case schema.InputOAuthClientCredentials:
			if binding.Value != nil || strings.TrimSpace(binding.SecretRef) == "" {
				return fmt.Errorf(
					"%w: MCP OAuth client input %q requires exactly one secret reference",
					basespec.ErrInvalid,
					name,
				)
			}
			if err := secret.ValidateMCPSecretRef(
				binding.SecretRef,
				server,
				spec.MCPSecretKindOAuthClientCredentials,
				"clientCredentials",
			); err != nil {
				return fmt.Errorf("MCP OAuth client input %q: %w", name, err)
			}

		default:
			return fmt.Errorf(
				"%w: MCP input %q has unsupported kind %q",
				basespec.ErrInvalid,
				name,
				declaration.Kind,
			)
		}
	}

	seen := make(map[artifact.ArtifactRef]struct{}, len(data.AdditionalPolicies))
	for _, ref := range data.AdditionalPolicies {
		if err := ref.Validate(); err != nil {
			return err
		}
		if ref.RootID != server.RootID {
			return fmt.Errorf(
				"%w: additional MCP policy belongs to another Root",
				basespec.ErrInvalid,
			)
		}
		if _, duplicate := seen[ref]; duplicate {
			return fmt.Errorf(
				"%w: duplicate additional MCP policy Artifact",
				basespec.ErrInvalid,
			)
		}
		seen[ref] = struct{}{}
	}

	return nil
}

func validateSecretBinding(
	server artifact.ArtifactRef,
	raw string,
	oauthClient bool,
) error {
	parsed, err := secret.ParseMCPSecretRef(raw)
	if err != nil {
		return err
	}
	if parsed.Server != server {
		return fmt.Errorf(
			"%w: secret reference belongs to another MCP Server Artifact",
			basespec.ErrInvalid,
		)
	}

	if oauthClient {
		if parsed.Kind != spec.MCPSecretKindOAuthClientCredentials {
			return fmt.Errorf(
				"%w: expected OAuth client credentials secret reference",
				basespec.ErrInvalid,
			)
		}
		return nil
	}

	switch parsed.Kind {
	case spec.MCPSecretKindHTTPHeader, spec.MCPSecretKindStdioEnv:
		return nil
	default:
		return fmt.Errorf(
			"%w: secret reference kind %q cannot bind a normal MCP secret input",
			basespec.ErrInvalid,
			parsed.Kind,
		)
	}
}
