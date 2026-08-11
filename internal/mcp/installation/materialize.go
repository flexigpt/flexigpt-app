package installation

import (
	"context"
	"fmt"
	"maps"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/schema"
)

var placeholderPattern = regexp.MustCompile(
	`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`,
)

type SecretResolver interface {
	ResolveSecret(
		ctx context.Context,
		ref string,
	) (string, error)
}

type EnvironmentResolver interface {
	ResolveEnvironment(
		ctx context.Context,
		name string,
	) (string, bool, error)
}

type Materialized struct {
	Core schema.CoreServer
	Auth schema.AuthenticationDeclaration

	ClientCredentialRef string
	SensitiveValues     []string
}

func Materialize(
	ctx context.Context,
	server artifact.ArtifactRef,
	document schema.ServerDocument,
	data ServerData,
	secrets SecretResolver,
	environment EnvironmentResolver,
) (Materialized, error) {
	if err := validateMaterializeInput(
		ctx,
		server,
		document,
		data,
	); err != nil {
		return Materialized{}, err
	}
	return MaterializeValidated(
		ctx,
		server,
		document,
		data,
		secrets,
		environment,
	)
}

// MaterializeValidated is for the internal resolver-to-runtime path. Callers
// must have already established that server, document, and data are valid.
//
// It intentionally validates only values created by profile selection and
// substitution. Those values are not known until this function runs.
func MaterializeValidated(
	ctx context.Context,
	server artifact.ArtifactRef,
	document schema.ServerDocument,
	data ServerData,
	secrets SecretResolver,
	environment EnvironmentResolver,
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

	values := make(map[string]string)
	secretValues := make(map[string]struct{})
	clientCredentialRef := ""

	for name, declaration := range document.Extension.Install.Inputs {
		binding, bound := data.Inputs[name]

		switch declaration.Kind {
		case schema.InputSecret:
			if !bound || strings.TrimSpace(binding.SecretRef) == "" {
				if declaration.Required {
					return Materialized{}, fmt.Errorf(
						"%w: required secret input %q is not bound",
						basespec.ErrReferenceUnresolved,
						name,
					)
				}
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

		case schema.InputOAuthClientCredentials:
			if !bound || strings.TrimSpace(binding.SecretRef) == "" {
				if declaration.Required {
					return Materialized{}, fmt.Errorf(
						"%w: OAuth client input %q is not bound",
						basespec.ErrReferenceUnresolved,
						name,
					)
				}
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
			if document.Extension.Auth.ClientCredentialsInput == name {
				clientCredentialRef = binding.SecretRef
			}

		case schema.InputText, schema.InputPath:
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
		}
	}

	core, err = substituteCore(core, values)
	if err != nil {
		return Materialized{}, err
	}
	auth := document.Extension.Auth
	auth.ClientIDMetadataDocumentURL, err = substitute(
		auth.ClientIDMetadataDocumentURL,
		values,
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

	if err := schema.ValidateMaterializedServer(core, auth); err != nil {
		return Materialized{}, err
	}

	sensitive := make([]string, 0, len(secretValues))
	for value := range secretValues {
		sensitive = append(sensitive, value)
	}
	slices.Sort(sensitive)

	return Materialized{
		Core:                core,
		Auth:                auth,
		ClientCredentialRef: clientCredentialRef,
		SensitiveValues:     sensitive,
	}, nil
}

func validateMaterializeInput(
	ctx context.Context,
	server artifact.ArtifactRef,
	document schema.ServerDocument,
	data ServerData,
) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: MCP materialization context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := server.Validate(); err != nil {
		return err
	}
	if err := schema.ValidateServer(document); err != nil {
		return err
	}
	return ValidateServerDataForDocument(server, document, data)
}

func selectProfile(
	base schema.CoreServer,
	profiles map[string]schema.ConnectionProfile,
	selected string,
) (schema.CoreServer, error) {
	if selected == "" {
		matches := make([]string, 0)
		for name, profile := range profiles {
			if slices.Contains(profile.Platforms, runtime.GOOS) {
				matches = append(matches, name)
			}
		}
		if len(matches) > 1 {
			return schema.CoreServer{}, fmt.Errorf(
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
		return schema.CoreServer{}, fmt.Errorf(
			"%w: MCP connection profile %q does not exist",
			basespec.ErrReferenceUnresolved,
			selected,
		)
	}
	if len(profile.Platforms) != 0 &&
		!slices.Contains(profile.Platforms, runtime.GOOS) {
		return schema.CoreServer{}, fmt.Errorf(
			"%w: MCP connection profile %q does not support %q",
			basespec.ErrInvalid,
			selected,
			runtime.GOOS,
		)
	}

	output := cloneCore(base)
	switch {
	case profile.Stdio != nil:
		if output.Type != schema.ServerTypeStdio {
			if profile.Stdio.Command == nil {
				return schema.CoreServer{}, fmt.Errorf(
					"%w: transport-changing stdio profile requires command",
					basespec.ErrInvalid,
				)
			}
			output = schema.CoreServer{
				Type: schema.ServerTypeStdio,
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
		if output.Type != schema.ServerTypeHTTP {
			if profile.HTTP.URL == nil {
				return schema.CoreServer{}, fmt.Errorf(
					"%w: transport-changing HTTP profile requires URL",
					basespec.ErrInvalid,
				)
			}
			output = schema.CoreServer{
				Type:    schema.ServerTypeHTTP,
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
		return schema.CoreServer{}, fmt.Errorf(
			"%w: MCP profile has no connection overlay",
			basespec.ErrInvalid,
		)
	}
	return output, nil
}

func substituteCore(
	input schema.CoreServer,
	values map[string]string,
) (schema.CoreServer, error) {
	output := cloneCore(input)
	var err error

	if output.Command, err = substitute(output.Command, values); err != nil {
		return schema.CoreServer{}, err
	}
	for index := range output.Args {
		output.Args[index], err = substitute(output.Args[index], values)
		if err != nil {
			return schema.CoreServer{}, err
		}
	}
	for key, value := range output.Env {
		output.Env[key], err = substitute(value, values)
		if err != nil {
			return schema.CoreServer{}, err
		}
	}
	if output.URL, err = substitute(output.URL, values); err != nil {
		return schema.CoreServer{}, err
	}
	for key, value := range output.Headers {
		output.Headers[key], err = substitute(value, values)
		if err != nil {
			return schema.CoreServer{}, err
		}
	}
	return output, nil
}

func substitute(
	input string,
	values map[string]string,
) (string, error) {
	if input == "" {
		return "", nil
	}
	var unresolved string
	output := placeholderPattern.ReplaceAllStringFunc(
		input,
		func(match string) string {
			parts := placeholderPattern.FindStringSubmatch(match)
			if len(parts) != 2 {
				return match
			}
			value, found := values[parts[1]]
			if !found {
				unresolved = parts[1]
				return match
			}
			return value
		},
	)
	if unresolved != "" {
		return "", fmt.Errorf(
			"%w: MCP input %q is unresolved",
			basespec.ErrReferenceUnresolved,
			unresolved,
		)
	}
	return output, nil
}

func cloneCore(input schema.CoreServer) schema.CoreServer {
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
