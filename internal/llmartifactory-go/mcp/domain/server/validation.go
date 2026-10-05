package server

import (
	"fmt"
	"maps"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/contract/v1"
)

var installationInputNamePattern = regexp.MustCompile(
	`^[A-Za-z_][A-Za-z0-9_]*$`,
)

func (value MaterializedServer) Validate() error {
	core := value.Core
	auth := value.Auth
	if len(placeholdersInServer(core, auth, nil)) != 0 {
		return fmt.Errorf(
			"%w: materialized MCP server still contains placeholders",
			spec.ErrReferenceUnresolved,
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

func CanonicalizeServer(input ServerDocument) (ServerDocument, error) {
	value, err := jsonutil.CloneJSON(input)
	if err != nil {
		return ServerDocument{}, err
	}
	value.Labels = maps.Clone(value.Labels)
	value.MCPServer = NormalizeCoreServer(value.MCPServer)
	value.Configuration = NormalizeServerConfiguration(value.Configuration)
	if value.DisplayName == "" {
		value.DisplayName = string(value.LogicalName)
	}
	if err := value.Validate(); err != nil {
		return ServerDocument{}, err
	}
	return value, nil
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

func NormalizeServerConfiguration(
	value ServerConfiguration,
) ServerConfiguration {
	value = cloneConfiguration(value)
	value.Auth = normalizeAuthentication(value.Auth)
	return value
}

// withImplicitEnvironmentInputs implements the portable `${NAME}` behavior
// for declarations that do not provide the consumer-specific runtime
// extension. Explicitly declared text, path, and secret inputs retain their
// declared behavior.
func withImplicitEnvironmentInputs(
	core CoreServer,
	value ServerConfiguration,
) ServerConfiguration {
	output := value
	output.Install.AllowEnvironment = slices.Clone(
		value.Install.AllowEnvironment,
	)
	seen := make(map[string]struct{}, len(output.Install.AllowEnvironment))
	for _, name := range output.Install.AllowEnvironment {
		seen[name] = struct{}{}
	}
	for name := range placeholdersInServer(
		core,
		output.Auth,
		output.ConnectionProfiles,
	) {
		if _, declared := output.Install.Inputs[name]; declared {
			continue
		}
		if _, found := seen[name]; found {
			continue
		}
		seen[name] = struct{}{}
		output.Install.AllowEnvironment = append(
			output.Install.AllowEnvironment,
			name,
		)
	}
	slices.Sort(output.Install.AllowEnvironment)
	return output
}

func normalizeAuthentication(
	value AuthenticationDeclaration,
) AuthenticationDeclaration {
	if value.Mode == "" {
		value.Mode = mcpv1.HTTPAuthModeNone
	}
	return value
}

func validateParts(
	name string,
	core CoreServer,
	extension ServerConfiguration,
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
				spec.ErrInvalid,
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
				spec.ErrInvalid,
				declaration.Kind,
			)
		}
		if declaration.Kind == InputSecret ||
			declaration.Kind == InputOAuthClientCredentials {
			if declaration.Default != nil {
				return fmt.Errorf(
					"%w: secret installation input %q cannot declare a default",
					spec.ErrInvalid,
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
				spec.ErrInvalid,
				inputName,
			)
		}
		if _, duplicate := seenAllowedEnvironment[inputName]; duplicate {
			return fmt.Errorf(
				"%w: duplicate allowed environment input %q",
				spec.ErrInvalid,
				inputName,
			)
		}
		seenAllowedEnvironment[inputName] = struct{}{}
		if declaration, declared := declared[inputName]; declared &&
			(declaration.Kind == InputSecret ||
				declaration.Kind == InputOAuthClientCredentials) {
			return fmt.Errorf(
				"%w: secret input %q cannot resolve from process environment",
				spec.ErrInvalid,
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
				spec.ErrInvalid,
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
			spec.ErrReferenceUnresolved,
			inputName,
		)
	}
	if err := validateWholeURLInputReference(
		"mcpServer.url",
		core.URL,
		declared,
	); err != nil {
		return err
	}
	for profileName, profile := range extension.ConnectionProfiles {
		if profile.HTTP == nil || profile.HTTP.URL == nil {
			continue
		}
		if err := validateWholeURLInputReference(
			"connectionProfiles."+profileName+".http.url",
			*profile.HTTP.URL,
			declared,
		); err != nil {
			return err
		}
	}
	if err := validateClientIDMetadataDocumentURLTemplate(
		extension.Auth.ClientIDMetadataDocumentURL,
	); err != nil {
		return err
	}

	switch extension.Auth.Mode {
	case mcpv1.HTTPAuthModeNone:
		if extension.Auth.ClientCredentialsInput != "" {
			return fmt.Errorf(
				"%w: no-auth server cannot declare OAuth credentials",
				spec.ErrInvalid,
			)
		}

	case mcpv1.HTTPAuthModeAPIKey:
		if core.Type != ServerTypeHTTP {
			return fmt.Errorf(
				"%w: API-key authentication requires HTTP transport",
				spec.ErrInvalid,
			)
		}
		if !coreUsesRequiredSecretInput(
			core,
			extension.ConnectionProfiles,
			declared,
		) {
			return fmt.Errorf(
				"%w: API-key authentication requires a required secret header placeholder",
				spec.ErrInvalid,
			)
		}

	case mcpv1.HTTPAuthModeOAuth:
		if core.Type != ServerTypeHTTP {
			return fmt.Errorf(
				"%w: OAuth requires HTTP transport",
				spec.ErrInvalid,
			)
		}
		if inputName := extension.Auth.ClientCredentialsInput; inputName != "" {
			declaration, found := declared[inputName]
			if !found ||
				declaration.Kind != InputOAuthClientCredentials {
				return fmt.Errorf(
					"%w: OAuth clientCredentialsInput %q is invalid",
					spec.ErrInvalid,
					inputName,
				)
			}
		}

	case mcpv1.HTTPAuthModeClientCredentials:
		if core.Type != ServerTypeHTTP {
			return fmt.Errorf(
				"%w: client credentials requires HTTP transport",
				spec.ErrInvalid,
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
				spec.ErrInvalid,
			)
		}

	default:
		return fmt.Errorf(
			"%w: invalid MCP auth mode %q",
			spec.ErrInvalid,
			extension.Auth.Mode,
		)
	}

	if extension.Policy != nil {
		if err := spec.ValidatePortableName(
			"MCP policy reference",
			string(extension.Policy.Name),
		); err != nil {
			return err
		}
	}

	for profileName, profile := range extension.ConnectionProfiles {
		if err := spec.ValidatePortableName(
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
					spec.ErrInvalid,
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
	if err := spec.ValidateOptionalText(
		"MCP installation input label",
		value.Label,
		spec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := spec.ValidateOptionalText(
		"MCP installation input description",
		value.Description,
		spec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if err := spec.ValidateOptionalText(
		"MCP installation input note",
		value.Note,
		spec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if err := spec.ValidateOptionalText(
		"MCP installation input placeholder",
		value.Placeholder,
		spec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if value.Default != nil {
		if err := spec.ValidateOptionalText(
			"MCP installation input default",
			*value.Default,
			spec.MaxDescriptionBytes,
		); err != nil {
			return err
		}
	}
	if value.ClientSecretRequired &&
		value.Kind != InputOAuthClientCredentials {
		return fmt.Errorf(
			"%w: only oauthClientCredentials input %q may require clientSecret",
			spec.ErrInvalid,
			name,
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
			spec.ErrInvalid,
			name,
		)
	}
	if profile.Stdio != nil && profile.HTTP != nil {
		return fmt.Errorf(
			"%w: connection profile %q has two transport overlays",
			spec.ErrInvalid,
			name,
		)
	}

	seenPlatforms := make(map[string]struct{}, len(profile.Platforms))
	for _, platform := range profile.Platforms {
		if _, duplicate := seenPlatforms[platform]; duplicate {
			return fmt.Errorf(
				"%w: connection profile %q repeats platform %q",
				spec.ErrInvalid,
				name,
				platform,
			)
		}
		seenPlatforms[platform] = struct{}{}
	}

	if profile.Stdio != nil {
		if profile.Stdio.Command != nil {
			if err := spec.ValidateRequiredText(
				"MCP profile stdio command",
				*profile.Stdio.Command,
				spec.MaxLocatorBytes,
			); err != nil {
				return err
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
					spec.ErrInvalid,
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
		if err := spec.ValidateRequiredText(
			"MCP stdio command",
			value.Command,
			spec.MaxLocatorBytes,
		); err != nil {
			return err
		}
		if value.URL != "" || len(value.Headers) != 0 {
			return fmt.Errorf(
				"%w: stdio MCP server cannot contain HTTP fields",
				spec.ErrInvalid,
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
				spec.ErrInvalid,
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
					spec.ErrInvalid,
					name,
				)
			}
		}

	default:
		return fmt.Errorf(
			"%w: unsupported MCP server type %q",
			spec.ErrInvalid,
			value.Type,
		)
	}
	return nil
}

func validateInclude(value *Include) error {
	if value == nil {
		return nil
	}
	if err := validateIncludeValues(
		"MCP include tools",
		value.Tools,
	); err != nil {
		return err
	}
	if err := validateIncludeValues(
		"MCP include resources",
		value.Resources,
	); err != nil {
		return err
	}
	return validateIncludeValues("MCP include prompts", value.Prompts)
}

func validateIncludeValues(
	label string,
	values []string,
) error {
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		if err := spec.ValidateRequiredText(
			label,
			value,
			spec.MaxURIBytes,
		); err != nil {
			return fmt.Errorf("%s[%d]: %w", label, index, err)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf(
				"%w: %s repeats %q",
				spec.ErrInvalid,
				label,
				value,
			)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validateConnectionTimeoutMS(value int) error {
	if value < 0 || value > MaxConnectionTimeoutMS {
		return fmt.Errorf(
			"%w: MCP connection timeout must be between zero and %d milliseconds",
			spec.ErrInvalid,
			MaxConnectionTimeoutMS,
		)
	}
	return nil
}

func validateExtension(
	_ string,
	value ServerConfiguration,
) error {
	if err := validateConnectionTimeoutMS(value.TimeoutMS); err != nil {
		return err
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
		len(raw) > spec.MaxURIBytes {
		return fmt.Errorf(
			"%w: MCP HTTP URL is invalid",
			spec.ErrInvalid,
		)
	}
	if _, whole := wholeURLPlaceholderName(raw); whole {
		return nil
	}
	probe := placeholderPattern.ReplaceAllString(raw, "example")
	value, err := url.Parse(probe)
	if err != nil {
		return fmt.Errorf("%w: invalid MCP HTTP URL: %w", spec.ErrInvalid, err)
	}
	if value.User != nil || value.Fragment != "" || value.Host == "" {
		return fmt.Errorf(
			"%w: MCP HTTP URL has disallowed components",
			spec.ErrInvalid,
		)
	}
	switch strings.ToLower(value.Scheme) {
	case "http", "https":
		return nil

	default:
		return fmt.Errorf(
			"%w: MCP HTTP URL must use HTTP or HTTPS",
			spec.ErrInvalid,
		)
	}
}

func validateWholeURLInputReference(
	field string,
	raw string,
	inputs map[string]InputDeclaration,
) error {
	inputName, whole := wholeURLPlaceholderName(raw)
	if !whole {
		return nil
	}
	input, found := inputs[inputName]
	if !found || input.Kind != InputText {
		return fmt.Errorf(
			"%w: whole MCP URL placeholder in %s requires a text installation input",
			spec.ErrInvalid,
			field,
		)
	}
	return nil
}

func wholeURLPlaceholderName(value string) (string, bool) {
	matches := placeholderPattern.FindStringSubmatch(value)
	if len(matches) != 2 || matches[0] != value {
		return "", false
	}
	return matches[1], true
}

func validateClientIDMetadataDocumentURLTemplate(
	raw string,
) error {
	if raw == "" {
		return nil
	}
	if strings.TrimSpace(raw) != raw ||
		len(raw) > spec.MaxURIBytes {
		return fmt.Errorf(
			"%w: invalid OAuth client metadata URL template",
			spec.ErrInvalid,
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
			spec.ErrInvalid,
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
			spec.ErrInvalid,
		)
	}
	return nil
}

func validateHeaderName(name string) error {
	if name == "" || strings.TrimSpace(name) != name {
		return fmt.Errorf(
			"%w: invalid MCP HTTP header name",
			spec.ErrInvalid,
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
			spec.ErrInvalid,
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
			spec.ErrInvalid,
		)
	}
	for _, character := range name {
		if character < 0x20 || character == 0x7f {
			return fmt.Errorf(
				"%w: invalid MCP environment name",
				spec.ErrInvalid,
			)
		}
	}
	return nil
}

func placeholderInputNameValid(value string) bool {
	return installationInputNamePattern.MatchString(value)
}
