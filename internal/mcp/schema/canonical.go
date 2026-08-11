package schema

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

var (
	portableNamePattern = regexp.MustCompile(
		`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`,
	)
	placeholderPattern = regexp.MustCompile(
		`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`,
	)
)

func parseBundle(
	raw []byte,
) (BundleDocument, json.RawMessage, error) {
	value, err := decodeStrict[BundleDocument](raw)
	if err != nil {
		return BundleDocument{}, nil, err
	}
	return canonicalizeBundle(value)
}

func parseServer(
	raw []byte,
) (ServerDocument, json.RawMessage, error) {
	value, err := decodeStrict[ServerDocument](raw)
	if err != nil {
		return ServerDocument{}, nil, err
	}
	return canonicalizeServer(value)
}

func parsePolicy(
	raw []byte,
) (PolicyDocument, json.RawMessage, error) {
	value, err := decodeStrict[PolicyDocument](raw)
	if err != nil {
		return PolicyDocument{}, nil, err
	}
	return canonicalizePolicy(value)
}

func canonicalizeBundle(
	input BundleDocument,
) (BundleDocument, json.RawMessage, error) {
	value, err := cloneJSON(input)
	if err != nil {
		return BundleDocument{}, nil, err
	}
	value.MCPServers = maps.Clone(value.MCPServers)
	value.BundleExtension.Servers = maps.Clone(value.BundleExtension.Servers)
	value.BundleExtension.Policies = maps.Clone(value.BundleExtension.Policies)
	value.Labels = maps.Clone(value.Labels)

	if value.MCPServers == nil {
		value.MCPServers = map[string]CoreServer{}
	}
	if value.BundleExtension.Servers == nil {
		value.BundleExtension.Servers = map[string]ServerExtension{}
	}
	if value.BundleExtension.Policies == nil {
		value.BundleExtension.Policies = map[string]PolicyDocument{}
	}

	for name, core := range value.MCPServers {
		core = normalizeCoreServer(core)
		value.MCPServers[name] = core

		extension := normalizeServerExtension(
			name,
			value.BundleExtension.Servers[name],
		)
		value.BundleExtension.Servers[name] = extension
	}

	for name, policyValue := range value.BundleExtension.Policies {
		canonical, _, err := canonicalizePolicy(policyValue)
		if err != nil {
			return BundleDocument{}, nil, fmt.Errorf(
				"policy %q: %w",
				name,
				err,
			)
		}
		value.BundleExtension.Policies[name] = canonical
	}

	if err := ValidateBundle(value); err != nil {
		return BundleDocument{}, nil, err
	}

	supplied := value.Digest
	value.Digest = ""
	calculated, err := canonicalDigest(value)
	if err != nil {
		return BundleDocument{}, nil, err
	}
	if supplied != "" && supplied != calculated {
		return BundleDocument{}, nil, fmt.Errorf(
			"%w: supplied MCP Bundle digest %q, calculated %q",
			basespec.ErrDigestMismatch,
			supplied,
			calculated,
		)
	}
	value.Digest = calculated

	raw, err := canonicalJSON(value)
	if err != nil {
		return BundleDocument{}, nil, err
	}
	return value, raw, nil
}

func canonicalizePolicy(
	input PolicyDocument,
) (PolicyDocument, json.RawMessage, error) {
	value, err := cloneJSON(input)
	if err != nil {
		return PolicyDocument{}, nil, err
	}
	value.Labels = maps.Clone(value.Labels)
	value.Body = NormalizePolicyBody(value.Body)

	if err := ValidatePolicy(value); err != nil {
		return PolicyDocument{}, nil, err
	}

	supplied := value.Digest
	value.Digest = ""
	calculated, err := canonicalDigest(value)
	if err != nil {
		return PolicyDocument{}, nil, err
	}
	if supplied != "" && supplied != calculated {
		return PolicyDocument{}, nil, fmt.Errorf(
			"%w: supplied MCP Policy digest %q, calculated %q",
			basespec.ErrDigestMismatch,
			supplied,
			calculated,
		)
	}
	value.Digest = calculated

	raw, err := canonicalJSON(value)
	if err != nil {
		return PolicyDocument{}, nil, err
	}
	return value, raw, nil
}

// ServerFromCanonicalBundle projects one server from a Bundle that has
// already been accepted and canonicalized by the Artifact Store shareable
// schema registry.
//
// It deliberately does not call CanonicalizeServer. MCP lifecycle code must
// not create a second portable-document validation path after registry
// canonicalization.
func ServerFromCanonicalBundle(
	bundle BundleDocument,
	name string,
) (ServerDocument, error) {
	core, found := bundle.MCPServers[name]
	if !found {
		return ServerDocument{}, fmt.Errorf(
			"%w: MCP server %q is not in the Bundle document",
			basespec.ErrNotFound,
			name,
		)
	}
	extension, found := bundle.BundleExtension.Servers[name]
	if !found {
		return ServerDocument{}, fmt.Errorf(
			"%w: canonical MCP Bundle has no extension for server %q",
			basespec.ErrInvalid,
			name,
		)
	}
	return cloneJSON(ServerDocument{
		Kind:           ServerKind,
		SchemaID:       ServerSchemaID,
		SchemaVersion:  SchemaVersion,
		LogicalName:    basespec.LogicalName(name),
		LogicalVersion: extension.LogicalVersion,
		DisplayName:    extension.DisplayName,
		Description:    extension.Description,
		Labels:         maps.Clone(extension.Labels),
		MCPServer:      core,
		Extension:      extension,
	})
}

func canonicalizeServer(
	input ServerDocument,
) (ServerDocument, json.RawMessage, error) {
	value, err := cloneJSON(input)
	if err != nil {
		return ServerDocument{}, nil, err
	}
	value.Labels = maps.Clone(value.Labels)
	value.MCPServer = normalizeCoreServer(value.MCPServer)
	value.Extension = normalizeServerExtension(
		string(value.LogicalName),
		value.Extension,
	)

	if err := ValidateServer(value); err != nil {
		return ServerDocument{}, nil, err
	}

	supplied := value.Digest
	value.Digest = ""
	calculated, err := canonicalDigest(value)
	if err != nil {
		return ServerDocument{}, nil, err
	}
	if supplied != "" && supplied != calculated {
		return ServerDocument{}, nil, fmt.Errorf(
			"%w: supplied MCP Server digest %q, calculated %q",
			basespec.ErrDigestMismatch,
			supplied,
			calculated,
		)
	}
	value.Digest = calculated

	raw, err := canonicalJSON(value)
	if err != nil {
		return ServerDocument{}, nil, err
	}
	return value, raw, nil
}

func ValidateBundle(value BundleDocument) error {
	if value.Kind != BundleKind ||
		value.SchemaID != BundleSchemaID ||
		value.SchemaVersion != SchemaVersion {
		return fmt.Errorf(
			"%w: unsupported MCP Bundle schema",
			basespec.ErrInvalid,
		)
	}
	if err := validatePortableMetadata(
		value.LogicalName,
		value.LogicalVersion,
		value.DisplayName,
		value.Description,
		value.Labels,
	); err != nil {
		return err
	}
	if len(value.MCPServers) > basespec.MaxDiscoveryCandidates {
		return fmt.Errorf(
			"%w: MCP Bundle server count exceeds limit",
			basespec.ErrInvalid,
		)
	}
	if len(value.BundleExtension.Policies) > basespec.MaxDiscoveryCandidates {
		return fmt.Errorf(
			"%w: MCP Bundle policy count exceeds limit",
			basespec.ErrInvalid,
		)
	}

	for name := range value.BundleExtension.Servers {
		if _, found := value.MCPServers[name]; !found {
			return fmt.Errorf(
				"%w: bundleExtension.servers[%q] has no mcpServers entry",
				basespec.ErrInvalid,
				name,
			)
		}
	}

	for name, core := range value.MCPServers {
		if err := validatePortableName("MCP server name", name); err != nil {
			return err
		}
		extension := value.BundleExtension.Servers[name]
		if err := validateServerParts(name, core, extension); err != nil {
			return fmt.Errorf("MCP server %q: %w", name, err)
		}
	}

	for name, policyValue := range value.BundleExtension.Policies {
		if err := validatePortableName("MCP policy name", name); err != nil {
			return err
		}
		if string(policyValue.LogicalName) != name {
			return fmt.Errorf(
				"%w: policy map key %q does not match logicalName %q",
				basespec.ErrInvalid,
				name,
				policyValue.LogicalName,
			)
		}
		if err := ValidatePolicy(policyValue); err != nil {
			return fmt.Errorf("MCP policy %q: %w", name, err)
		}
	}
	if err := validateRequiredBundlePolicyReferences(value); err != nil {
		return err
	}

	return nil
}

func ValidateServer(value ServerDocument) error {
	if value.Kind != ServerKind ||
		value.SchemaID != ServerSchemaID ||
		value.SchemaVersion != SchemaVersion {
		return fmt.Errorf(
			"%w: unsupported MCP Server schema",
			basespec.ErrInvalid,
		)
	}
	if err := validatePortableMetadata(
		value.LogicalName,
		value.LogicalVersion,
		value.DisplayName,
		value.Description,
		value.Labels,
	); err != nil {
		return err
	}
	return validateServerParts(
		string(value.LogicalName),
		value.MCPServer,
		value.Extension,
	)
}

func ValidatePolicy(value PolicyDocument) error {
	if value.Kind != PolicyKind ||
		value.SchemaID != PolicySchemaID ||
		value.SchemaVersion != SchemaVersion {
		return fmt.Errorf(
			"%w: unsupported MCP Policy schema",
			basespec.ErrInvalid,
		)
	}
	if err := validatePortableMetadata(
		value.LogicalName,
		value.LogicalVersion,
		value.DisplayName,
		value.Description,
		value.Labels,
	); err != nil {
		return err
	}
	return ValidatePolicyBody(value.Body)
}

func ValidatePolicyBody(body PolicyBody) error {
	switch body.TrustLevel {
	case mcpSpec.MCPTrustLevelTrusted, mcpSpec.MCPTrustLevelUntrusted:
	default:
		return fmt.Errorf(
			"%w: invalid MCP trust level %q",
			basespec.ErrInvalid,
			body.TrustLevel,
		)
	}

	switch body.DefaultPolicy.DefaultApprovalRule {
	case mcpSpec.MCPApprovalRuleAllow,
		mcpSpec.MCPApprovalRuleAsk,
		mcpSpec.MCPApprovalRuleDeny:
	default:
		return fmt.Errorf(
			"%w: invalid MCP approval rule",
			basespec.ErrInvalid,
		)
	}
	switch body.DefaultPolicy.DefaultExecutionMode {
	case mcpSpec.MCPExecutionModeAuto,
		mcpSpec.MCPExecutionModeManual:
	default:
		return fmt.Errorf(
			"%w: invalid MCP execution mode",
			basespec.ErrInvalid,
		)
	}

	for name, override := range body.ToolPolicies {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf(
				"%w: empty MCP tool policy name",
				basespec.ErrInvalid,
			)
		}
		if override.ToolName != name {
			return fmt.Errorf(
				"%w: MCP tool policy key and toolName differ",
				basespec.ErrInvalid,
			)
		}
		if override.ApprovalRule != nil {
			switch *override.ApprovalRule {
			case mcpSpec.MCPApprovalRuleAllow,
				mcpSpec.MCPApprovalRuleAsk,
				mcpSpec.MCPApprovalRuleDeny:
			default:
				return fmt.Errorf(
					"%w: invalid tool approval rule",
					basespec.ErrInvalid,
				)
			}
		}
		if override.ExecutionMode != nil {
			switch *override.ExecutionMode {
			case mcpSpec.MCPExecutionModeAuto,
				mcpSpec.MCPExecutionModeManual:
			default:
				return fmt.Errorf(
					"%w: invalid tool execution mode",
					basespec.ErrInvalid,
				)
			}
		}
		if override.ExpectedDigest != "" {
			if err := cryptoutil.ValidateDigest(
				cryptoutil.Digest(override.ExpectedDigest),
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func ValidateMaterializedServer(
	core CoreServer,
	auth AuthenticationDeclaration,
) error {
	if len(placeholdersInServer(core, auth, nil)) != 0 {
		return fmt.Errorf(
			"%w: materialized MCP server still contains placeholders",
			basespec.ErrReferenceUnresolved,
		)
	}
	if err := validateCoreServer(core); err != nil {
		return err
	}
	if auth.ClientIDMetadataDocumentURL != "" {
		if err := validateClientIDMetadataDocumentURL(
			auth.ClientIDMetadataDocumentURL,
		); err != nil {
			return err
		}
	}
	return nil
}

func NormalizePolicyBody(input PolicyBody) PolicyBody {
	output := input
	output.ToolPolicies = maps.Clone(input.ToolPolicies)
	if output.TrustLevel == "" {
		output.TrustLevel = mcpSpec.MCPTrustLevelUntrusted
	}
	if output.DefaultPolicy == (mcpSpec.MCPServerPolicy{}) {
		output.DefaultPolicy = mcpSpec.DefaultMCPServerPolicy()
	}
	if output.ToolPolicies == nil {
		output.ToolPolicies = map[string]mcpSpec.MCPToolPolicyOverride{}
	}
	for name, override := range output.ToolPolicies {
		if override.ToolName == "" {
			override.ToolName = name
		}
		output.ToolPolicies[name] = override
	}
	if output.AppsPolicy == (mcpSpec.MCPAppsPolicy{}) {
		output.AppsPolicy = mcpSpec.MCPAppsPolicy{
			Enabled:                          false,
			AllowAppInitiatedToolCalls:       false,
			RequireApprovalForOpenLink:       true,
			RequireApprovalForContextUpdates: true,
		}
	}
	return output
}

func normalizeCoreServer(value CoreServer) CoreServer {
	value.Args = slices.Clone(value.Args)
	value.Env = maps.Clone(value.Env)
	value.Headers = maps.Clone(value.Headers)
	if value.Type == "" && strings.TrimSpace(value.Command) != "" {
		value.Type = ServerTypeStdio
	}
	return value
}

func normalizeServerExtension(
	name string,
	value ServerExtension,
) ServerExtension {
	value.Labels = maps.Clone(value.Labels)
	value.Install.Inputs = maps.Clone(value.Install.Inputs)
	value.Install.AllowEnvironment = slices.Clone(
		value.Install.AllowEnvironment,
	)
	value.ConnectionProfiles = maps.Clone(value.ConnectionProfiles)
	value.Auth = normalizeAuthentication(value.Auth)
	if value.DisplayName == "" {
		value.DisplayName = name
	}
	return value
}

func normalizeAuthentication(
	value AuthenticationDeclaration,
) AuthenticationDeclaration {
	if value.Mode == "" {
		value.Mode = mcpSpec.MCPHTTPAuthNone
	}
	return value
}

func validateServerParts(
	name string,
	core CoreServer,
	extension ServerExtension,
) error {
	if err := validateCoreServer(core); err != nil {
		return err
	}
	if err := validateExtension(name, extension); err != nil {
		return err
	}

	declared := make(
		map[string]InputDeclaration,
		len(extension.Install.Inputs),
	)
	for inputName, declaration := range extension.Install.Inputs {
		if !placeholderInputNameValid(inputName) {
			return fmt.Errorf(
				"%w: invalid installation input name %q",
				basespec.ErrInvalid,
				inputName,
			)
		}
		if err := validateInputDeclaration(inputName, declaration); err != nil {
			return err
		}
		switch declaration.Kind {
		case InputText, InputSecret, InputPath,
			InputOAuthClientCredentials:
		default:
			return fmt.Errorf(
				"%w: invalid installation input kind %q",
				basespec.ErrInvalid,
				declaration.Kind,
			)
		}
		if declaration.Kind == InputSecret ||
			declaration.Kind == InputOAuthClientCredentials {
			if declaration.Default != nil {
				return fmt.Errorf(
					"%w: secret installation input %q cannot declare a default",
					basespec.ErrInvalid,
					inputName,
				)
			}
		}
		declared[inputName] = declaration
	}

	allowedEnvironment := make(map[string]struct{})
	seenAllowedEnvironment := make(map[string]struct{})
	for _, inputName := range extension.Install.AllowEnvironment {
		if !placeholderInputNameValid(inputName) {
			return fmt.Errorf(
				"%w: invalid allowed environment input %q",
				basespec.ErrInvalid,
				inputName,
			)
		}
		if _, duplicate := seenAllowedEnvironment[inputName]; duplicate {
			return fmt.Errorf(
				"%w: duplicate allowed environment input %q",
				basespec.ErrInvalid,
				inputName,
			)
		}
		seenAllowedEnvironment[inputName] = struct{}{}
		if declaration, declared := declared[inputName]; declared &&
			(declaration.Kind == InputSecret ||
				declaration.Kind == InputOAuthClientCredentials) {
			return fmt.Errorf(
				"%w: secret input %q cannot resolve from process environment",
				basespec.ErrInvalid,
				inputName,
			)
		}
		allowedEnvironment[inputName] = struct{}{}
	}

	placeholders := placeholdersInServer(
		core,
		extension.Auth,
		extension.ConnectionProfiles,
	)
	for inputName := range placeholders {
		if declaration, found := declared[inputName]; found &&
			declaration.Kind == InputOAuthClientCredentials {
			return fmt.Errorf(
				"%w: OAuth client credentials input %q cannot be substituted into connection fields",
				basespec.ErrInvalid,
				inputName,
			)
		}
		if _, found := declared[inputName]; found {
			continue
		}
		if _, found := allowedEnvironment[inputName]; found {
			continue
		}
		return fmt.Errorf(
			"%w: placeholder %q has no installation input declaration",
			basespec.ErrReferenceUnresolved,
			inputName,
		)
	}
	if err := validateClientIDMetadataDocumentURLTemplate(
		extension.Auth.ClientIDMetadataDocumentURL,
	); err != nil {
		return err
	}

	switch extension.Auth.Mode {
	case mcpSpec.MCPHTTPAuthNone:
		if extension.Auth.ClientCredentialsInput != "" {
			return fmt.Errorf(
				"%w: no-auth server cannot declare OAuth credentials",
				basespec.ErrInvalid,
			)
		}

	case mcpSpec.MCPHTTPAuthAPIKey:
		if core.Type != ServerTypeHTTP {
			return fmt.Errorf(
				"%w: API-key authentication requires HTTP transport",
				basespec.ErrInvalid,
			)
		}
		if !coreUsesRequiredSecretInput(
			core,
			extension.ConnectionProfiles,
			declared,
		) {
			return fmt.Errorf(
				"%w: API-key authentication requires a required secret header placeholder",
				basespec.ErrInvalid,
			)
		}

	case mcpSpec.MCPHTTPAuthOAuth:
		if core.Type != ServerTypeHTTP {
			return fmt.Errorf(
				"%w: OAuth requires HTTP transport",
				basespec.ErrInvalid,
			)
		}
		if inputName := extension.Auth.ClientCredentialsInput; inputName != "" {
			declaration, found := declared[inputName]
			if !found ||
				declaration.Kind != InputOAuthClientCredentials {
				return fmt.Errorf(
					"%w: OAuth clientCredentialsInput %q is invalid",
					basespec.ErrInvalid,
					inputName,
				)
			}
		}

	case mcpSpec.MCPHTTPAuthClientCredentials:
		if core.Type != ServerTypeHTTP {
			return fmt.Errorf(
				"%w: client credentials requires HTTP transport",
				basespec.ErrInvalid,
			)
		}
		inputName := extension.Auth.ClientCredentialsInput
		declaration, found := declared[inputName]
		if inputName == "" ||
			!found ||
			declaration.Kind != InputOAuthClientCredentials ||
			!declaration.Required {
			return fmt.Errorf(
				"%w: client credentials requires a required oauthClientCredentials input",
				basespec.ErrInvalid,
			)
		}

	default:
		return fmt.Errorf(
			"%w: invalid MCP auth mode %q",
			basespec.ErrInvalid,
			extension.Auth.Mode,
		)
	}

	if extension.Policy != nil {
		if err := validatePortableName(
			"MCP policy reference",
			string(extension.Policy.Ref),
		); err != nil {
			return err
		}
	}

	for profileName, profile := range extension.ConnectionProfiles {
		if err := validatePortableName(
			"MCP connection profile",
			profileName,
		); err != nil {
			return err
		}
		if err := validateConnectionProfile(
			profileName,
			profile,
		); err != nil {
			return err
		}
		for _, platform := range profile.Platforms {
			switch platform {
			case "linux", "darwin", "windows":
			default:
				return fmt.Errorf(
					"%w: unsupported portable platform %q",
					basespec.ErrInvalid,
					platform,
				)
			}
		}
	}
	_, err := secretInputTargets(
		core,
		extension,
	)
	return err
}

func validateInputDeclaration(
	name string,
	value InputDeclaration,
) error {
	if err := basespec.ValidateOptionalText(
		"MCP installation input label",
		value.Label,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"MCP installation input description",
		value.Description,
		basespec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"MCP installation input note",
		value.Note,
		basespec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"MCP installation input placeholder",
		value.Placeholder,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if value.Default != nil {
		if err := basespec.ValidateOptionalText(
			"MCP installation input default",
			*value.Default,
			basespec.MaxDescriptionBytes,
		); err != nil {
			return err
		}
	}
	if value.ClientSecretRequired &&
		value.Kind != InputOAuthClientCredentials {
		return fmt.Errorf(
			"%w: only oauthClientCredentials input %q may require clientSecret",
			basespec.ErrInvalid,
			name,
		)
	}
	return nil
}

func validateRequiredBundlePolicyReferences(
	value BundleDocument,
) error {
	for name, extension := range value.BundleExtension.Servers {
		if extension.Policy == nil || !extension.Policy.Required {
			continue
		}
		if _, found := value.BundleExtension.Policies[string(extension.Policy.Ref)]; found {
			continue
		}
		return fmt.Errorf(
			"%w: MCP server %q requires missing inline policy %q",
			basespec.ErrReferenceUnresolved,
			name,
			extension.Policy.Ref,
		)
	}
	return nil
}

func validateConnectionProfile(
	name string,
	profile ConnectionProfile,
) error {
	if profile.Stdio == nil && profile.HTTP == nil {
		return fmt.Errorf(
			"%w: connection profile %q has no transport overlay",
			basespec.ErrInvalid,
			name,
		)
	}
	if profile.Stdio != nil && profile.HTTP != nil {
		return fmt.Errorf(
			"%w: connection profile %q has two transport overlays",
			basespec.ErrInvalid,
			name,
		)
	}

	seenPlatforms := make(map[string]struct{}, len(profile.Platforms))
	for _, platform := range profile.Platforms {
		if _, duplicate := seenPlatforms[platform]; duplicate {
			return fmt.Errorf(
				"%w: connection profile %q repeats platform %q",
				basespec.ErrInvalid,
				name,
				platform,
			)
		}
		seenPlatforms[platform] = struct{}{}
	}

	if profile.Stdio != nil {
		if profile.Stdio.Command != nil {
			if err := basespec.ValidateRequiredText(
				"MCP profile stdio command",
				*profile.Stdio.Command,
				basespec.MaxLocatorBytes,
			); err != nil {
				return err
			}
			if shellCommand(*profile.Stdio.Command) {
				return fmt.Errorf(
					"%w: MCP profile %q uses a shell command",
					basespec.ErrInvalid,
					name,
				)
			}
		}
		for key := range profile.Stdio.Env {
			if err := validateEnvironmentName(key); err != nil {
				return err
			}
		}
		for _, key := range profile.Stdio.RemoveEnv {
			if err := validateEnvironmentName(key); err != nil {
				return err
			}
		}
	}

	if profile.HTTP != nil {
		if profile.HTTP.URL != nil {
			if err := validateURLTemplate(*profile.HTTP.URL); err != nil {
				return err
			}
		}
		for key, value := range profile.HTTP.Headers {
			if err := validateHeaderName(key); err != nil {
				return err
			}
			if strings.ContainsAny(value, "\r\n\x00") {
				return fmt.Errorf(
					"%w: MCP profile %q header %q contains CR, LF, or NUL",
					basespec.ErrInvalid,
					name,
					key,
				)
			}
		}
		for _, key := range profile.HTTP.RemoveHeaders {
			if err := validateHeaderName(key); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateCoreServer(value CoreServer) error {
	switch value.Type {
	case ServerTypeStdio:
		if err := basespec.ValidateRequiredText(
			"MCP stdio command",
			value.Command,
			basespec.MaxLocatorBytes,
		); err != nil {
			return err
		}
		if value.URL != "" || len(value.Headers) != 0 {
			return fmt.Errorf(
				"%w: stdio MCP server cannot contain HTTP fields",
				basespec.ErrInvalid,
			)
		}
		if shellCommand(value.Command) {
			return fmt.Errorf(
				"%w: MCP stdio command must execute the server directly",
				basespec.ErrInvalid,
			)
		}
		for key := range value.Env {
			if err := validateEnvironmentName(key); err != nil {
				return err
			}
		}

	case ServerTypeHTTP:
		if value.Command != "" ||
			len(value.Args) != 0 ||
			len(value.Env) != 0 {
			return fmt.Errorf(
				"%w: HTTP MCP server cannot contain stdio fields",
				basespec.ErrInvalid,
			)
		}
		if err := validateURLTemplate(value.URL); err != nil {
			return err
		}
		for name, headerValue := range value.Headers {
			if err := validateHeaderName(name); err != nil {
				return err
			}
			if strings.ContainsAny(headerValue, "\r\n\x00") {
				return fmt.Errorf(
					"%w: MCP HTTP header %q contains CR, LF, or NUL",
					basespec.ErrInvalid,
					name,
				)
			}
		}

	default:
		return fmt.Errorf(
			"%w: unsupported MCP server type %q",
			basespec.ErrInvalid,
			value.Type,
		)
	}
	return nil
}

func validateConnectionTimeoutMS(value int) error {
	if value < 0 || value > MaxConnectionTimeoutMS {
		return fmt.Errorf(
			"%w: MCP connection timeout must be between zero and %d milliseconds",
			basespec.ErrInvalid,
			MaxConnectionTimeoutMS,
		)
	}
	return nil
}

func validateExtension(
	name string,
	value ServerExtension,
) error {
	if err := basespec.ValidateOptionalText(
		"MCP server logical version",
		string(value.LogicalVersion),
		basespec.MaxVersionBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"MCP server display name",
		value.DisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"MCP server description",
		value.Description,
		basespec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if err := validateConnectionTimeoutMS(value.TimeoutMS); err != nil {
		return err
	}
	if err := validateLabels(value.Labels); err != nil {
		return err
	}
	if value.DisplayName == "" {
		return fmt.Errorf(
			"%w: MCP server %q has no display name",
			basespec.ErrInvalid,
			name,
		)
	}
	return nil
}

func validatePortableMetadata(
	logicalName basespec.LogicalName,
	logicalVersion basespec.LogicalVersion,
	displayName string,
	description string,
	labels map[string]string,
) error {
	if err := validatePortableName(
		"logical name",
		string(logicalName),
	); err != nil {
		return err
	}
	if err := basespec.ValidateLogicalVersion(
		logicalVersion,
		true,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"display name",
		displayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"description",
		description,
		basespec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	return validateLabels(labels)
}

func validateLabels(values map[string]string) error {
	if len(values) > basespec.MaxLabels {
		return fmt.Errorf(
			"%w: labels exceed %d entries",
			basespec.ErrInvalid,
			basespec.MaxLabels,
		)
	}
	for key, value := range values {
		if err := basespec.ValidateIdentifier(
			"MCP label key",
			key,
			basespec.MaxKindBytes,
		); err != nil {
			return err
		}
		if err := basespec.ValidateRequiredText(
			"MCP label value",
			value,
			basespec.MaxLabelValueBytes,
		); err != nil {
			return err
		}
	}
	return nil
}

func validatePortableName(label, value string) error {
	if !portableNamePattern.MatchString(value) {
		return fmt.Errorf(
			"%w: %s %q is not a portable MCP name",
			basespec.ErrInvalid,
			label,
			value,
		)
	}
	return nil
}

func placeholdersInServer(
	core CoreServer,
	auth AuthenticationDeclaration,
	profiles map[string]ConnectionProfile,
) map[string]struct{} {
	output := make(map[string]struct{})
	add := func(value string) {
		for _, match := range placeholderPattern.FindAllStringSubmatch(
			value,
			-1,
		) {
			if len(match) == 2 {
				output[match[1]] = struct{}{}
			}
		}
	}

	add(core.Command)
	for _, value := range core.Args {
		add(value)
	}
	for _, value := range core.Env {
		add(value)
	}
	add(core.URL)
	for _, value := range core.Headers {
		add(value)
	}
	add(auth.ClientIDMetadataDocumentURL)

	for _, profile := range profiles {
		if profile.Stdio != nil {
			if profile.Stdio.Command != nil {
				add(*profile.Stdio.Command)
			}
			if profile.Stdio.Args != nil {
				for _, value := range *profile.Stdio.Args {
					add(value)
				}
			}
			for _, value := range profile.Stdio.Env {
				add(value)
			}
		}
		if profile.HTTP != nil {
			if profile.HTTP.URL != nil {
				add(*profile.HTTP.URL)
			}
			for _, value := range profile.HTTP.Headers {
				add(value)
			}
		}
	}
	return output
}

func coreUsesRequiredSecretInput(
	core CoreServer,
	profiles map[string]ConnectionProfile,
	inputs map[string]InputDeclaration,
) bool {
	usesRequiredSecret := func(headers map[string]string) bool {
		for _, value := range headers {
			for _, match := range placeholderPattern.FindAllStringSubmatch(
				value,
				-1,
			) {
				if len(match) != 2 {
					continue
				}
				if input, found := inputs[match[1]]; found &&
					input.Kind == InputSecret &&
					input.Required {
					return true
				}
			}
		}
		return false
	}

	if usesRequiredSecret(core.Headers) {
		return true
	}
	for _, profile := range profiles {
		if profile.HTTP != nil &&
			usesRequiredSecret(profile.HTTP.Headers) {
			return true
		}
	}
	return false
}

func validateURLTemplate(raw string) error {
	if strings.TrimSpace(raw) == "" ||
		strings.TrimSpace(raw) != raw ||
		len(raw) > basespec.MaxURIBytes {
		return fmt.Errorf(
			"%w: MCP HTTP URL is invalid",
			basespec.ErrInvalid,
		)
	}
	probe := placeholderPattern.ReplaceAllString(raw, "example")
	value, err := url.Parse(probe)
	if err != nil {
		return fmt.Errorf("%w: invalid MCP HTTP URL: %w", basespec.ErrInvalid, err)
	}
	if value.User != nil || value.Fragment != "" || value.Host == "" {
		return fmt.Errorf(
			"%w: MCP HTTP URL has disallowed components",
			basespec.ErrInvalid,
		)
	}
	switch value.Scheme {
	case "https":
		return nil
	case "http":
		if !isLoopback(value.Hostname()) {
			return fmt.Errorf(
				"%w: plain HTTP MCP URL must use a loopback host",
				basespec.ErrInvalid,
			)
		}
		return nil
	default:
		return fmt.Errorf(
			"%w: MCP HTTP URL must use HTTP or HTTPS",
			basespec.ErrInvalid,
		)
	}
}

func validateClientIDMetadataDocumentURLTemplate(
	raw string,
) error {
	if raw == "" {
		return nil
	}
	if strings.TrimSpace(raw) != raw ||
		len(raw) > basespec.MaxURIBytes {
		return fmt.Errorf(
			"%w: invalid OAuth client metadata URL template",
			basespec.ErrInvalid,
		)
	}
	probe := placeholderPattern.ReplaceAllString(
		raw,
		"example",
	)
	return validateClientIDMetadataDocumentURL(probe)
}

func validateClientIDMetadataDocumentURL(raw string) error {
	value, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf(
			"%w: invalid OAuth client metadata URL: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if value.Scheme != "https" ||
		value.Host == "" ||
		value.User != nil ||
		value.Fragment != "" ||
		value.Path == "" ||
		value.Path == "/" {
		return fmt.Errorf(
			"%w: invalid OAuth client metadata URL",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func validateHeaderName(name string) error {
	if name == "" || strings.TrimSpace(name) != name {
		return fmt.Errorf(
			"%w: invalid MCP HTTP header name",
			basespec.ErrInvalid,
		)
	}
	for _, character := range name {
		if character >= 'A' && character <= 'Z' ||
			character >= 'a' && character <= 'z' ||
			character >= '0' && character <= '9' {
			continue
		}
		if strings.ContainsRune("!#$%&'*+-.^_`|~", character) {
			continue
		}
		return fmt.Errorf(
			"%w: invalid MCP HTTP header character %q",
			basespec.ErrInvalid,
			character,
		)
	}
	return nil
}

func validateEnvironmentName(name string) error {
	if name == "" ||
		strings.TrimSpace(name) != name ||
		strings.ContainsAny(name, "=\x00") {
		return fmt.Errorf(
			"%w: invalid MCP environment name",
			basespec.ErrInvalid,
		)
	}
	for _, character := range name {
		if character < 0x20 || character == 0x7f {
			return fmt.Errorf(
				"%w: invalid MCP environment name",
				basespec.ErrInvalid,
			)
		}
	}
	return nil
}

func placeholderInputNameValid(value string) bool {
	return regexp.MustCompile(
		`^[A-Za-z_][A-Za-z0-9_]*$`,
	).MatchString(value)
}

func shellCommand(command string) bool {
	command = strings.ReplaceAll(command, "\\", "/")
	parts := strings.Split(command, "/")
	base := strings.ToLower(parts[len(parts)-1])
	switch base {
	case "bash", "sh", "zsh", "cmd", "cmd.exe",
		"powershell", "powershell.exe", "pwsh", "pwsh.exe":
		return true
	default:
		return false
	}
}

func isLoopback(host string) bool {
	if strings.EqualFold(strings.TrimSpace(host), "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func canonicalDigest(value any) (cryptoutil.Digest, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return "", err
	}
	if len(canonical) > basespec.MaxDefinitionBytes {
		return "", fmt.Errorf(
			"%w: canonical MCP document exceeds byte limit",
			basespec.ErrInvalid,
		)
	}
	return cryptoutil.DigestBytes(canonical), nil
}

func canonicalJSON(value any) (json.RawMessage, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(canonical), nil
}

func decodeStrict[T any](raw []byte) (T, error) {
	var output T
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return output, err
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		return output, fmt.Errorf(
			"%w: decode MCP document: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("MCP document has trailing JSON values")
		}
		return output, fmt.Errorf("%w: %w", basespec.ErrInvalid, err)
	}
	return output, nil
}

func cloneJSON[T any](input T) (T, error) {
	var output T
	raw, err := json.Marshal(input)
	if err != nil {
		return output, err
	}
	if err := json.Unmarshal(raw, &output); err != nil {
		return output, err
	}
	return output, nil
}
