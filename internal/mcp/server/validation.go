package server

import (
	"encoding/json"
	"fmt"
	"maps"
	"net"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/builtin/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
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
	document ServerDocument,
	data ServerData,
) error {
	if err := server.Validate(); err != nil {
		return err
	}
	if err := ValidateServer(document); err != nil {
		return err
	}
	if err := ValidateServerData(data); err != nil {
		return err
	}

	secretTargets, err := SecretInputTargets(document)
	if err != nil {
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
		case InputText, InputPath:
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

		case InputSecret:
			if binding.Value != nil || strings.TrimSpace(binding.SecretRef) == "" {
				return fmt.Errorf(
					"%w: MCP secret input %q requires exactly one secret reference",
					basespec.ErrInvalid,
					name,
				)
			}
			target, found := secretTargets[name]
			if !found {
				return fmt.Errorf(
					"%w: MCP secret input %q is not used by a permitted connection target",
					basespec.ErrInvalid,
					name,
				)
			}
			if err := validateSecretBinding(
				server,
				binding.SecretRef,
				target,
			); err != nil {
				return fmt.Errorf("MCP secret input %q: %w", name, err)
			}

		case InputOAuthClientCredentials:
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

func CanonicalizeServer(
	input ServerDocument,
) (ServerDocument, json.RawMessage, error) {
	value, err := jsonutil.CloneJSON(input)
	if err != nil {
		return ServerDocument{}, nil, err
	}
	value.Labels = maps.Clone(value.Labels)
	value.MCPServer = NormalizeCoreServer(value.MCPServer)
	value.Extension = NormalizeServerExtension(
		string(value.LogicalName),
		value.Extension,
	)

	if err := ValidateServer(value); err != nil {
		return ServerDocument{}, nil, err
	}

	supplied := value.Digest
	value.Digest = ""
	calculated, err := cryptoutil.CanonicalDigest(value)
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

	raw, err := jsonutil.MarshalCanonicalObject(value, basespec.MaxDefinitionBytes)
	if err != nil {
		return ServerDocument{}, nil, err
	}
	return value, raw, nil
}

func NormalizeCoreServer(value CoreServer) CoreServer {
	value.Args = slices.Clone(value.Args)
	value.Env = maps.Clone(value.Env)
	value.Headers = maps.Clone(value.Headers)
	if value.Type == "" && strings.TrimSpace(value.Command) != "" {
		value.Type = ServerTypeStdio
	}
	return value
}

func NormalizeServerExtension(
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
		value.Mode = MCPHTTPAuthNone
	}
	return value
}

func ValidateServer(value ServerDocument) error {
	if value.Kind != schema.ServerKind ||
		value.SchemaID != schema.ServerSchemaID ||
		value.SchemaVersion != schema.MCPSchemaVersion {
		return fmt.Errorf(
			"%w: unsupported MCP Server schema",
			basespec.ErrInvalid,
		)
	}
	if err := basespec.ValidatePortableMetadata(
		value.LogicalName,
		value.LogicalVersion,
		value.DisplayName,
		value.Description,
		value.Labels,
	); err != nil {
		return err
	}
	return ValidateServerParts(
		string(value.LogicalName),
		value.MCPServer,
		value.Extension,
	)
}

func ValidateServerParts(
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
	case MCPHTTPAuthNone:
		if extension.Auth.ClientCredentialsInput != "" {
			return fmt.Errorf(
				"%w: no-auth server cannot declare OAuth credentials",
				basespec.ErrInvalid,
			)
		}

	case MCPHTTPAuthAPIKey:
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

	case MCPHTTPAuthOAuth:
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

	case MCPHTTPAuthClientCredentials:
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
		if err := basespec.ValidatePortableName(
			"MCP policy reference",
			string(extension.Policy.Ref),
		); err != nil {
			return err
		}
	}

	for profileName, profile := range extension.ConnectionProfiles {
		if err := basespec.ValidatePortableName(
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

func validateSecretBinding(
	server artifact.ArtifactRef,
	raw string,
	target SecretInputTarget,
) error {
	switch target.Kind {
	case SecretInputTargetStdioEnv:
		return secret.ValidateMCPSecretRef(
			raw,
			server,
			spec.MCPSecretKindStdioEnv,
			target.Slot,
		)
	case SecretInputTargetHTTPHeader:
		return secret.ValidateMCPSecretRef(
			raw,
			server,
			spec.MCPSecretKindHTTPHeader,
			target.Slot,
		)
	default:
		return fmt.Errorf(
			"%w: unsupported MCP secret input target %q",
			basespec.ErrInvalid,
			target.Kind,
		)
	}
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
	if err := basespec.ValidateLabels("", value.Labels); err != nil {
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
