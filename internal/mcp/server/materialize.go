package server

import (
	"context"
	"fmt"
	"maps"
	"runtime"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

type Materialized struct {
	Core CoreServer
	Auth AuthenticationDeclaration

	ClientCredentialRef            string
	ClientCredentialSecretRequired bool
	TimeoutMS                      int
	SensitiveValues                []string
}

// MaterializeValidated is for the internal resolver-to-runtime path. Callers
// must have already established that server, document, and data are valid.
//
// It intentionally validates only values created by profile selection and
// substitution. Those values are not known until this function runs.
func MaterializeValidated(
	ctx context.Context,
	server artifact.ArtifactRef,
	document ServerDocument,
	data ServerData,
	secrets SecretResolver,
	environment EnvironmentResolver,
) (Materialized, error) {
	return materializeValidated(
		ctx,
		server,
		document,
		data,
		secrets,
		environment,
		true,
	)
}

// MaterializeInspectionValidated creates a sanitized materialization for
// health and setup inspection. It validates installation shape and resolves
// non-secret values, but it never loads a secret value from Setting Store.
func MaterializeInspectionValidated(
	ctx context.Context,
	server artifact.ArtifactRef,
	document ServerDocument,
	data ServerData,
	environment EnvironmentResolver,
) (Materialized, error) {
	return materializeValidated(ctx, server, document, data, nil, environment, false)
}

func materializeValidated(
	ctx context.Context,
	server artifact.ArtifactRef,
	document ServerDocument,
	data ServerData,
	secrets SecretResolver,
	environment EnvironmentResolver,
	resolveSecrets bool,
) (Materialized, error) {
	if ctx == nil {
		return Materialized{}, fmt.Errorf(
			"%w: MCP materialization context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return Materialized{}, err
	}

	core, err := selectProfile(
		document.MCPServer,
		document.Extension.ConnectionProfiles,
		data.SelectedConnectionProfile,
	)
	if err != nil {
		return Materialized{}, err
	}

	timeoutMS := document.Extension.TimeoutMS
	if timeoutMS == 0 {
		timeoutMS = DefaultConnectionTimeoutMS
	}

	values := make(map[string]string)
	secretValues := make(map[string]struct{})
	unboundOptional := make(map[string]struct{})
	clientCredentialRef := ""

	for name, declaration := range document.Extension.Install.Inputs {
		binding, bound := data.Inputs[name]

		switch declaration.Kind {
		case InputSecret:
			if !bound || strings.TrimSpace(binding.SecretRef) == "" {
				if declaration.Required {
					return Materialized{}, fmt.Errorf(
						"%w: required secret input %q is not bound",
						basespec.ErrReferenceUnresolved,
						name,
					)
				}
				unboundOptional[name] = struct{}{}
				continue
			}
			if !resolveSecrets {
				values[name] = "configured"
				continue
			}
			if secrets == nil {
				return Materialized{}, fmt.Errorf(
					"%w: secret resolver is unavailable",
					basespec.ErrReferenceUnresolved,
				)
			}
			value, err := secrets.ResolveSecret(ctx, binding.SecretRef)
			if err != nil {
				return Materialized{}, err
			}
			if value == "" {
				return Materialized{}, fmt.Errorf(
					"%w: secret input %q is empty",
					basespec.ErrReferenceUnresolved,
					name,
				)
			}
			values[name] = value
			secretValues[value] = struct{}{}

		case InputOAuthClientCredentials:
			if !bound || strings.TrimSpace(binding.SecretRef) == "" {
				if declaration.Required {
					return Materialized{}, fmt.Errorf(
						"%w: OAuth client input %q is not bound",
						basespec.ErrReferenceUnresolved,
						name,
					)
				}
				unboundOptional[name] = struct{}{}
				continue
			}
			if document.Extension.Auth.ClientCredentialsInput == name {
				clientCredentialRef = binding.SecretRef
			}
			if !resolveSecrets {
				continue
			}
			if secrets == nil {
				return Materialized{}, fmt.Errorf(
					"%w: secret resolver is unavailable",
					basespec.ErrReferenceUnresolved,
				)
			}
			value, err := secrets.ResolveSecret(ctx, binding.SecretRef)
			if err != nil {
				return Materialized{}, err
			}
			if strings.TrimSpace(value) == "" {
				return Materialized{}, fmt.Errorf(
					"%w: OAuth client input %q is empty",
					basespec.ErrReferenceUnresolved,
					name,
				)
			}
			secretValues[value] = struct{}{}

		case InputText, InputPath:
			switch {
			case bound && binding.Value != nil:
				values[name] = *binding.Value
			case declaration.Default != nil:
				values[name] = *declaration.Default
			case declaration.Required:
				return Materialized{}, fmt.Errorf(
					"%w: required MCP input %q is not bound",
					basespec.ErrReferenceUnresolved,
					name,
				)
			default:
				unboundOptional[name] = struct{}{}
			}

		default:
			return Materialized{}, fmt.Errorf(
				"%w: unsupported MCP input kind %q",
				basespec.ErrInvalid,
				declaration.Kind,
			)
		}
	}

	for _, name := range document.Extension.Install.AllowEnvironment {
		if _, present := values[name]; present {
			continue
		}
		if environment == nil {
			continue
		}
		value, found, err := environment.ResolveEnvironment(ctx, name)
		if err != nil {
			return Materialized{}, err
		}
		if found {
			values[name] = value
			delete(unboundOptional, name)
		}
	}

	core, err = substituteCore(core, values, unboundOptional)
	if err != nil {
		return Materialized{}, err
	}
	auth := document.Extension.Auth
	auth.ClientIDMetadataDocumentURL, err = substituteOptionalScalar(
		auth.ClientIDMetadataDocumentURL,
		values,
		unboundOptional,
	)
	if err != nil {
		return Materialized{}, err
	}

	if auth.ClientCredentialsInput != "" &&
		clientCredentialRef == "" &&
		auth.Mode == "clientCredentials" {
		return Materialized{}, fmt.Errorf(
			"%w: required OAuth client credentials are not configured",
			basespec.ErrReferenceUnresolved,
		)
	}

	if err := ValidateMaterializedServer(core, auth); err != nil {
		return Materialized{}, err
	}

	sensitive := make([]string, 0, len(secretValues))
	for value := range secretValues {
		sensitive = append(sensitive, value)
	}
	slices.Sort(sensitive)

	return Materialized{
		Core:                           core,
		Auth:                           auth,
		ClientCredentialRef:            clientCredentialRef,
		ClientCredentialSecretRequired: document.OAuthClientSecretRequired(),
		TimeoutMS:                      timeoutMS,
		SensitiveValues:                sensitive,
	}, nil
}

func selectProfile(
	base CoreServer,
	profiles map[string]ConnectionProfile,
	selected string,
) (CoreServer, error) {
	if selected == "" {
		matches := make([]string, 0)
		for name, profile := range profiles {
			if slices.Contains(profile.Platforms, runtime.GOOS) {
				matches = append(matches, name)
			}
		}
		if len(matches) > 1 {
			return CoreServer{}, fmt.Errorf(
				"%w: multiple MCP connection profiles match platform %q",
				basespec.ErrConflict,
				runtime.GOOS,
			)
		}
		if len(matches) == 1 {
			selected = matches[0]
		}
	}
	if selected == "" {
		return cloneCore(base), nil
	}

	profile, found := profiles[selected]
	if !found {
		return CoreServer{}, fmt.Errorf(
			"%w: MCP connection profile %q does not exist",
			basespec.ErrReferenceUnresolved,
			selected,
		)
	}
	if len(profile.Platforms) != 0 &&
		!slices.Contains(profile.Platforms, runtime.GOOS) {
		return CoreServer{}, fmt.Errorf(
			"%w: MCP connection profile %q does not support %q",
			basespec.ErrInvalid,
			selected,
			runtime.GOOS,
		)
	}

	output := cloneCore(base)
	switch {
	case profile.Stdio != nil:
		if output.Type != ServerTypeStdio {
			if profile.Stdio.Command == nil {
				return CoreServer{}, fmt.Errorf(
					"%w: transport-changing stdio profile requires command",
					basespec.ErrInvalid,
				)
			}
			output = CoreServer{
				Type: ServerTypeStdio,
				Env:  map[string]string{},
			}
		}
		if profile.Stdio.Command != nil {
			output.Command = *profile.Stdio.Command
		}
		if profile.Stdio.Args != nil {
			output.Args = slices.Clone(*profile.Stdio.Args)
		}
		if output.Env == nil {
			output.Env = map[string]string{}
		}
		maps.Copy(output.Env, profile.Stdio.Env)
		for _, name := range profile.Stdio.RemoveEnv {
			delete(output.Env, name)
		}

	case profile.HTTP != nil:
		if output.Type != ServerTypeHTTP {
			if profile.HTTP.URL == nil {
				return CoreServer{}, fmt.Errorf(
					"%w: transport-changing HTTP profile requires URL",
					basespec.ErrInvalid,
				)
			}
			output = CoreServer{
				Type:    ServerTypeHTTP,
				Headers: map[string]string{},
			}
		}
		if profile.HTTP.URL != nil {
			output.URL = *profile.HTTP.URL
		}
		if output.Headers == nil {
			output.Headers = map[string]string{}
		}
		maps.Copy(output.Headers, profile.HTTP.Headers)
		for _, name := range profile.HTTP.RemoveHeaders {
			deleteFold(output.Headers, name)
		}

	default:
		return CoreServer{}, fmt.Errorf(
			"%w: MCP profile has no connection overlay",
			basespec.ErrInvalid,
		)
	}
	return output, nil
}

func substituteCore(
	input CoreServer,
	values map[string]string,
	optional map[string]struct{},
) (CoreServer, error) {
	output := cloneCore(input)
	var err error

	if output.Command, err = substituteRequired(output.Command, values); err != nil {
		return CoreServer{}, err
	}
	if output.Args, err = substituteArguments(output.Args, values, optional); err != nil {
		return CoreServer{}, err
	}
	if output.Env, err = substituteOptionalMap(output.Env, values, optional); err != nil {
		return CoreServer{}, err
	}
	if output.URL, err = substituteRequired(output.URL, values); err != nil {
		return CoreServer{}, err
	}
	if output.Headers, err = substituteOptionalMap(output.Headers, values, optional); err != nil {
		return CoreServer{}, err
	}
	return output, nil
}

func substituteRequired(
	input string,
	values map[string]string,
) (string, error) {
	output, missing := substituteTemplate(input, values)
	if len(missing) != 0 {
		return "", unresolvedInputError(missing[0])
	}
	return output, nil
}

func substituteOptionalScalar(
	input string,
	values map[string]string,
	optional map[string]struct{},
) (string, error) {
	output, missing := substituteTemplate(input, values)
	if len(missing) == 0 {
		return output, nil
	}
	if missingAreOptional(missing, optional) && placeholderOnly(input) {
		return "", nil
	}
	return "", unresolvedInputError(missing[0])
}

func substituteArguments(
	input []string,
	values map[string]string,
	optional map[string]struct{},
) ([]string, error) {
	output := make([]string, 0, len(input))
	for _, value := range input {
		resolved, missing := substituteTemplate(value, values)
		if len(missing) == 0 {
			output = append(output, resolved)
			continue
		}
		if missingAreOptional(missing, optional) && placeholderOnly(value) {
			continue
		}
		return nil, unresolvedInputError(missing[0])
	}
	return output, nil
}

func substituteOptionalMap(
	input map[string]string,
	values map[string]string,
	optional map[string]struct{},
) (map[string]string, error) {
	if len(input) == 0 {
		return maps.Clone(input), nil
	}

	output := make(map[string]string, len(input))
	for key, value := range input {
		resolved, missing := substituteTemplate(value, values)
		if len(missing) != 0 {
			if missingAreOptional(missing, optional) {
				continue
			}
			return nil, unresolvedInputError(missing[0])
		}
		output[key] = resolved
	}
	return output, nil
}

func substituteTemplate(
	input string,
	values map[string]string,
) (output string, missing []string) {
	missingSet := make(map[string]struct{})
	output = placeholderPattern.ReplaceAllStringFunc(
		input,
		func(match string) string {
			parts := placeholderPattern.FindStringSubmatch(match)
			if len(parts) != 2 {
				return match
			}
			value, found := values[parts[1]]
			if !found {
				missingSet[parts[1]] = struct{}{}
				return match
			}
			return value
		},
	)
	if len(missingSet) == 0 {
		return output, nil
	}
	missing = make([]string, 0, len(missingSet))
	for name := range missingSet {
		missing = append(missing, name)
	}
	slices.Sort(missing)
	return output, missing
}

func missingAreOptional(
	missing []string,
	optional map[string]struct{},
) bool {
	for _, name := range missing {
		if _, found := optional[name]; !found {
			return false
		}
	}
	return len(missing) != 0
}

func placeholderOnly(value string) bool {
	matches := placeholderPattern.FindAllString(value, -1)
	return len(matches) != 0 && strings.Join(matches, "") == value
}

func unresolvedInputError(name string) error {
	return fmt.Errorf(
		"%w: MCP input %q is unresolved",
		basespec.ErrReferenceUnresolved,
		name,
	)
}

func cloneCore(input CoreServer) CoreServer {
	output := input
	output.Args = slices.Clone(input.Args)
	output.Env = maps.Clone(input.Env)
	output.Headers = maps.Clone(input.Headers)
	return output
}

func deleteFold(values map[string]string, name string) {
	for key := range values {
		if strings.EqualFold(strings.TrimSpace(key), strings.TrimSpace(name)) {
			delete(values, key)
		}
	}
}
