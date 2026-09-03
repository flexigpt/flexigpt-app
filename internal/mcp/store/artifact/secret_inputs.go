package artifact

import (
	"fmt"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
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

// SecretInputTargets validates and returns the allowed target for every
// secret installation input used by a canonical ServerDocument.
//
// Secret values may only materialize into stdio environment values or HTTP
// header values. They are prohibited in commands, args, URLs, OAuth metadata
// URLs, and all other scalar connection fields.
func SecretInputTargets(
	document ServerDocument,
) (map[string]SecretInputTarget, error) {
	if err := ValidateServer(document); err != nil {
		return nil, err
	}
	return secretInputTargets(document.MCPServer, document.Extension)
}

func secretInputTargets(
	core CoreServer,
	extension ServerExtension,
) (map[string]SecretInputTarget, error) {
	targets := make(map[string]SecretInputTarget)
	targetOwners := make(map[string]string)
	inputs := extension.Install.Inputs

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
		extension.Auth.ClientIDMetadataDocumentURL,
		"extension.auth.clientIDMetadataDocumentURL",
	); err != nil {
		return nil, err
	}
	if err := recordEnvironment(core.Env, "mcpServer.env"); err != nil {
		return nil, err
	}
	if err := recordHeaders(core.Headers, "mcpServer.headers"); err != nil {
		return nil, err
	}

	profileNames := sortedStringKeys(extension.ConnectionProfiles)
	for _, profileName := range profileNames {
		profile := extension.ConnectionProfiles[profileName]
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
