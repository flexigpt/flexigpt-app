package server

import (
	"fmt"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	mcpDomainSecret "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/secret"
)

type SecretInputTargetKind string

const (
	//nolint:gosec // Enum.
	SecretInputTargetStdioEnv   SecretInputTargetKind = "stdioEnv"
	SecretInputTargetHTTPHeader SecretInputTargetKind = "httpHeader"
)

// SecretInputTarget is the single runtime materialization target permitted for
// one `secret` installation input.
//
// A local secret reference has a target-specific kind and slot. One portable
// secret input cannot safely bind multiple distinct environment variables or
// HTTP headers because one opaque local reference cannot prove both targets.
type SecretInputTarget struct {
	Kind SecretInputTargetKind
	Slot string
}

func (value ServerDocument) SecretInputTargets() (
	map[string]SecretInputTarget,
	error,
) {
	if err := value.Validate(); err != nil {
		return nil, err
	}
	return secretInputTargets(value.MCPServer, value.Configuration)
}

func (value ServerDocument) AcceptsSecretTarget(
	kind mcpDomainSecret.MCPSecretKind,
	slot string,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	normalizedSlot, err := kind.NormalizeSlot(slot)
	if err != nil {
		return err
	}

	switch kind {
	case mcpDomainSecret.MCPSecretKindOAuthClientCredentials:
		input := value.Configuration.Auth.ClientCredentialsInput
		declaration, found := value.Configuration.Install.Inputs[input]
		if input == "" ||
			!found ||
			declaration.Kind != InputOAuthClientCredentials {
			return fmt.Errorf(
				"%w: OAuth client credential target is not declared",
				basespec.ErrReferenceUnresolved,
			)
		}
		return nil

	case mcpDomainSecret.MCPSecretKindStdioEnv, mcpDomainSecret.MCPSecretKindHTTPHeader:
		targets, err := value.SecretInputTargets()
		if err != nil {
			return err
		}
		for _, target := range targets {
			expectedKind := mcpDomainSecret.MCPSecretKindStdioEnv
			if target.Kind == SecretInputTargetHTTPHeader {
				expectedKind = mcpDomainSecret.MCPSecretKindHTTPHeader
			}
			if expectedKind == kind &&
				strings.EqualFold(target.Slot, normalizedSlot) {
				return nil
			}
		}
		return fmt.Errorf(
			"%w: MCP secret target is not declared by the server",
			basespec.ErrReferenceUnresolved,
		)

	default:
		return fmt.Errorf(
			"%w: unsupported MCP secret kind %q",
			basespec.ErrInvalid,
			kind,
		)
	}
}

func (target SecretInputTarget) matches(
	server artifact.ArtifactRef,
	raw string,
) error {
	ref, err := mcpDomainSecret.ParseMCPSecretRef(raw)
	if err != nil {
		return err
	}

	switch target.Kind {
	case SecretInputTargetStdioEnv:
		return ref.Matches(server, mcpDomainSecret.MCPSecretKindStdioEnv, target.Slot)
	case SecretInputTargetHTTPHeader:
		return ref.Matches(server, mcpDomainSecret.MCPSecretKindHTTPHeader, target.Slot)
	default:
		return fmt.Errorf(
			"%w: unsupported MCP secret input target %q",
			basespec.ErrInvalid,
			target.Kind,
		)
	}
}

func secretInputTargets(
	core CoreServer,
	configuration ServerConfiguration,
) (map[string]SecretInputTarget, error) {
	targets := make(map[string]SecretInputTarget)
	targetOwners := make(map[string]string)
	inputs := configuration.Install.Inputs

	record := func(
		inputName string,
		kind SecretInputTargetKind,
		slot string,
		field string,
	) error {
		declaration, declared := inputs[inputName]
		if !declared || declaration.Kind != InputSecret {
			return nil
		}
		if strings.TrimSpace(slot) == "" {
			return fmt.Errorf(
				"%w: secret input %q has no materialization slot in %s",
				basespec.ErrInvalid,
				inputName,
				field,
			)
		}

		next := SecretInputTarget{Kind: kind, Slot: slot}
		if current, found := targets[inputName]; found {
			if current.Kind != next.Kind ||
				!strings.EqualFold(current.Slot, next.Slot) {
				return fmt.Errorf(
					"%w: secret input %q is used by multiple targets",
					basespec.ErrInvalid,
					inputName,
				)
			}
			return nil
		}

		targetKey := string(kind) + "\x00" +
			strings.ToLower(strings.TrimSpace(slot))
		if owner, found := targetOwners[targetKey]; found &&
			owner != inputName {
			return fmt.Errorf(
				"%w: secret inputs %q and %q use the same materialization target",
				basespec.ErrInvalid,
				owner,
				inputName,
			)
		}
		targetOwners[targetKey] = inputName
		targets[inputName] = next
		return nil
	}

	rejectSecret := func(value, field string) error {
		for _, inputName := range placeholderNames(value) {
			declaration, declared := inputs[inputName]
			if declared && declaration.Kind == InputSecret {
				return fmt.Errorf(
					"%w: secret input %q cannot be substituted into %s",
					basespec.ErrInvalid,
					inputName,
					field,
				)
			}
		}
		return nil
	}

	recordEnvironment := func(
		values map[string]string,
		field string,
	) error {
		keys := sortedStringKeys(values)
		for _, name := range keys {
			for _, inputName := range placeholderNames(values[name]) {
				if err := record(
					inputName,
					SecretInputTargetStdioEnv,
					name,
					field,
				); err != nil {
					return err
				}
			}
		}
		return nil
	}

	recordHeaders := func(
		values map[string]string,
		field string,
	) error {
		keys := sortedStringKeys(values)
		for _, name := range keys {
			for _, inputName := range placeholderNames(values[name]) {
				if err := record(
					inputName,
					SecretInputTargetHTTPHeader,
					name,
					field,
				); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := rejectSecret(core.Command, "mcpServer.command"); err != nil {
		return nil, err
	}
	for _, value := range core.Args {
		if err := rejectSecret(value, "mcpServer.args"); err != nil {
			return nil, err
		}
	}
	if err := rejectSecret(core.URL, "mcpServer.url"); err != nil {
		return nil, err
	}
	if err := rejectSecret(
		configuration.Auth.ClientIDMetadataDocumentURL,
		"auth.clientIDMetadataDocumentURL",
	); err != nil {
		return nil, err
	}
	if err := recordEnvironment(core.Env, "mcpServer.env"); err != nil {
		return nil, err
	}
	if err := recordHeaders(core.Headers, "mcpServer.headers"); err != nil {
		return nil, err
	}

	profileNames := sortedStringKeys(configuration.ConnectionProfiles)
	for _, profileName := range profileNames {
		profile := configuration.ConnectionProfiles[profileName]
		if profile.Stdio != nil {
			if profile.Stdio.Command != nil {
				if err := rejectSecret(
					*profile.Stdio.Command,
					"connectionProfiles."+profileName+".stdio.command",
				); err != nil {
					return nil, err
				}
			}
			if profile.Stdio.Args != nil {
				for _, value := range *profile.Stdio.Args {
					if err := rejectSecret(
						value,
						"connectionProfiles."+profileName+".stdio.args",
					); err != nil {
						return nil, err
					}
				}
			}
			if err := recordEnvironment(
				profile.Stdio.Env,
				"connectionProfiles."+profileName+".stdio.env",
			); err != nil {
				return nil, err
			}
		}
		if profile.HTTP != nil {
			if profile.HTTP.URL != nil {
				if err := rejectSecret(
					*profile.HTTP.URL,
					"connectionProfiles."+profileName+".http.url",
				); err != nil {
					return nil, err
				}
			}
			if err := recordHeaders(
				profile.HTTP.Headers,
				"connectionProfiles."+profileName+".http.headers",
			); err != nil {
				return nil, err
			}
		}
	}

	for inputName, declaration := range inputs {
		if declaration.Kind != InputSecret {
			continue
		}
		if _, found := targets[inputName]; !found {
			return nil, fmt.Errorf(
				"%w: secret input %q has no permitted environment or HTTP header target",
				basespec.ErrInvalid,
				inputName,
			)
		}
	}
	return targets, nil
}

func placeholderNames(value string) []string {
	seen := make(map[string]struct{})
	for _, match := range placeholderPattern.FindAllStringSubmatch(value, -1) {
		if len(match) == 2 {
			seen[match[1]] = struct{}{}
		}
	}
	output := make([]string, 0, len(seen))
	for name := range seen {
		output = append(output, name)
	}
	sort.Strings(output)
	return output
}

func sortedStringKeys[T any](values map[string]T) []string {
	output := make([]string, 0, len(values))
	for value := range values {
		output = append(output, value)
	}
	sort.Strings(output)
	return output
}
