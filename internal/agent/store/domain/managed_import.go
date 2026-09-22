package domain

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcpv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/textv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

type ManagedImportIssueSeverity string

const (
	ManagedImportIssueError        ManagedImportIssueSeverity = "error"
	ManagedImportIssueConfirmation ManagedImportIssueSeverity = "confirmation"
	ManagedImportIssueInformation  ManagedImportIssueSeverity = "information"
)

type ManagedImportIssue struct {
	Code     string
	Severity ManagedImportIssueSeverity
	Path     string
	Message  string
}

type ManagedMCPSetupInput struct {
	Name                 string
	Kind                 string
	Label                string
	Description          string
	Required             bool
	ClientSecretRequired bool
}

type ManagedMCPSetupDescriptor struct {
	OccurrencePath string
	Name           basespec.LogicalName
	Transport      string
	Command        string
	URL            string
	AuthMode       string
	Inputs         []ManagedMCPSetupInput
}

type ManagedAgentImportAdmission struct {
	Issues              []ManagedImportIssue
	MCPSetupDescriptors []ManagedMCPSetupDescriptor
}

func (v *ManagedAgentImportAdmission) HasErrors() bool {
	for _, issue := range v.Issues {
		if issue.Severity == ManagedImportIssueError {
			return true
		}
	}
	return false
}

func (v *ManagedAgentImportAdmission) addError(
	code string,
	path string,
	message string,
) {
	v.Issues = append(v.Issues, ManagedImportIssue{
		Code:     code,
		Severity: ManagedImportIssueError,
		Path:     path,
		Message:  message,
	})
}

func (v *ManagedAgentImportAdmission) addConfirmation(
	code string,
	path string,
	message string,
) {
	v.Issues = append(v.Issues, ManagedImportIssue{
		Code:     code,
		Severity: ManagedImportIssueConfirmation,
		Path:     path,
		Message:  message,
	})
}

// ValidateManagedAgentImport validates semantic restrictions that are not
// expressible solely through the strict JSON Schema profile. Store-backed
// reference existence and ambiguity checks belong to consumerapi.
func ValidateManagedAgentImport(
	document agentv1.AgentDocument,
) (ManagedAgentImportAdmission, error) {
	if err := document.Validate(); err != nil {
		return ManagedAgentImportAdmission{}, err
	}

	output := ManagedAgentImportAdmission{}
	if document.Locator != nil {
		output.addError(
			"agent.import.root-locator",
			"",
			"managed Agent import cannot use a top-level locator",
		)
	}
	if document.Loop != nil || document.Workflow != nil {
		output.addError(
			"agent.import.program",
			"",
			"managed Agent import cannot declare loop or workflow",
		)
	}

	for _, member := range document.Members {
		if err := validateManagedAgentMember(
			&output,
			member,
		); err != nil {
			return ManagedAgentImportAdmission{}, err
		}
	}
	return output, nil
}

func validateManagedAgentMember(
	output *ManagedAgentImportAdmission,
	member declaration.Entry,
) error {
	form, err := member.MemberForm()
	if err != nil {
		return err
	}

	header := member.Header()
	relationship, err := member.Relationship()
	if err != nil {
		return err
	}

	memberPath := fmt.Sprintf(
		"members/%s/%s",
		header.Type,
		header.Name,
	)

	switch header.Type {
	case declaration.TypeText:
		if form != declaration.MemberContained {
			output.addError(
				"agent.import.text-form",
				memberPath,
				"managed Agent Text must be contained inline Text",
			)
			return nil
		}

		target, err := member.ContainedDeclaration()
		if err != nil {
			return err
		}
		document, err := textv1.DecodeTextEntry(target)
		if err != nil {
			return err
		}
		memberPath = fmt.Sprintf(
			"members/text/%s/%s",
			document.Insert,
			document.Name,
		)
		if document.Content == nil ||
			document.Locator != nil ||
			len(document.Include) != 0 ||
			len(document.Exclude) != 0 {
			output.addError(
				"agent.import.text-inline",
				memberPath,
				"managed Agent Text must use inline content without source selection",
			)
		}

	case declaration.TypeModel:
		validateManagedNamedReference(
			output,
			form,
			header,
			relationship.Scope,
			memberPath,
			true,
		)

	case declaration.TypeTool:
		validateManagedNamedReference(
			output,
			form,
			header,
			relationship.Scope,
			memberPath,
			true,
		)

		raw, found := relationship.Overrides["autoExecute"]
		if !found {
			return nil
		}
		var autoExecute bool
		if err := json.Unmarshal(raw, &autoExecute); err != nil {
			return err
		}
		if autoExecute {
			output.addConfirmation(
				"agent.import.tool-auto-execute",
				memberPath,
				"Tool autoExecute requires explicit confirmation",
			)
		}

	case declaration.TypeSkill:
		validateManagedBuiltinReference(
			output,
			form,
			header,
			relationship.Scope,
			memberPath,
		)

	case declaration.TypeMCP:
		switch form {
		case declaration.MemberNamed:
			validateManagedBuiltinReference(
				output,
				form,
				header,
				relationship.Scope,
				memberPath,
			)
			output.MCPSetupDescriptors = append(
				output.MCPSetupDescriptors,
				ManagedMCPSetupDescriptor{
					OccurrencePath: memberPath,
					Name:           basespec.LogicalName(header.Name),
				},
			)

		case declaration.MemberContained:
			target, err := member.ContainedDeclaration()
			if err != nil {
				return err
			}
			document, err := mcpv1.DecodeMCPEntry(target)
			if err != nil {
				return err
			}
			if document.Locator != nil || document.Server != "" {
				output.addError(
					"agent.import.inline-mcp-locator",
					memberPath,
					"contained managed MCP cannot select another declaration",
				)
			}
			if document.Transport == mcpv1.TransportStdio {
				output.addConfirmation(
					"agent.import.mcp-stdio",
					memberPath,
					"inline stdio MCP requires explicit confirmation",
				)
			}

			output.MCPSetupDescriptors = append(
				output.MCPSetupDescriptors,
				MCPSetupDescriptorForDocument(memberPath, document),
			)
			validateInlineMCPCredentialPlacement(
				output,
				memberPath,
				document,
			)

		default:
			output.addError(
				"agent.import.mcp-form",
				memberPath,
				"managed Agent MCP must be named built-in MCP or contained inline MCP",
			)
		}

	case declaration.TypeMCPPolicy:
		validateManagedBuiltinReference(
			output,
			form,
			header,
			relationship.Scope,
			memberPath,
		)

	default:
		output.addError(
			"agent.import.member-type",
			memberPath,
			fmt.Sprintf(
				"managed Agent import does not support member type %q",
				header.Type,
			),
		)
	}
	return nil
}

func validateManagedNamedReference(
	output *ManagedAgentImportAdmission,
	form declaration.MemberForm,
	header declaration.Header,
	scope declaration.LookupScope,
	path string,
	allowUnscoped bool,
) {
	if form != declaration.MemberNamed || header.Locator != nil {
		output.addError(
			"agent.import.reference-form",
			path,
			"managed Agent reference must be a named relationship without locator",
		)
		return
	}
	if allowUnscoped && scope == "" {
		return
	}
	if scope != declaration.LookupScopeBuiltin {
		output.addError(
			"agent.import.reference-scope",
			path,
			"managed Agent reference uses an unsupported lookup scope",
		)
	}
}

func validateManagedBuiltinReference(
	output *ManagedAgentImportAdmission,
	form declaration.MemberForm,
	header declaration.Header,
	scope declaration.LookupScope,
	path string,
) {
	validateManagedNamedReference(
		output,
		form,
		header,
		scope,
		path,
		false,
	)
}

// MCPSetupDescriptorForDocument projects declaration-owned MCP setup metadata.
// It deliberately contains no installation values, secret references, OAuth
// state, selected profile, or runtime connection state.
func MCPSetupDescriptorForDocument(
	occurrencePath string,
	document mcpv1.MCPDocument,
) ManagedMCPSetupDescriptor {
	output := ManagedMCPSetupDescriptor{
		OccurrencePath: occurrencePath,
		Name:           basespec.LogicalName(document.Name),
		Transport:      string(document.Transport),
		Command:        document.Command,
		URL:            document.URL,
	}
	if document.Auth != nil {
		output.AuthMode = string(document.Auth.Mode)
	}
	if document.Install == nil {
		return output
	}

	names := make([]string, 0, len(document.Install.Inputs))
	for name := range document.Install.Inputs {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		input := document.Install.Inputs[name]
		required := input.Required != nil && *input.Required
		clientSecretRequired := input.ClientSecretRequired != nil &&
			*input.ClientSecretRequired
		output.Inputs = append(output.Inputs, ManagedMCPSetupInput{
			Name:                 name,
			Kind:                 input.Kind,
			Label:                input.Label,
			Description:          input.Description,
			Required:             required,
			ClientSecretRequired: clientSecretRequired,
		})
	}
	return output
}

func validateInlineMCPCredentialPlacement(
	output *ManagedAgentImportAdmission,
	path string,
	document mcpv1.MCPDocument,
) {
	inputs := map[string]mcpv1.InstallInput{}
	if document.Install != nil {
		inputs = document.Install.Inputs
	}

	validateSensitiveValueMap(
		output,
		path+"/env",
		document.Env,
		inputs,
	)
	validateSensitiveValueMap(
		output,
		path+"/headers",
		document.Headers,
		inputs,
	)

	for name, profile := range document.ConnectionProfiles {
		if profile.Stdio != nil {
			validateSensitiveValueMap(
				output,
				path+"/connectionProfiles/"+name+"/stdio/env",
				profile.Stdio.Env,
				inputs,
			)
		}
		if profile.HTTP != nil {
			validateSensitiveValueMap(
				output,
				path+"/connectionProfiles/"+name+"/http/headers",
				profile.HTTP.Headers,
				inputs,
			)
		}
	}

	if document.Auth == nil ||
		document.Auth.ClientCredentialsInput == "" {
		return
	}
	input, found := inputs[document.Auth.ClientCredentialsInput]
	if !found || input.Kind != "oauthClientCredentials" {
		output.addError(
			"agent.import.mcp-client-credentials-input",
			path+"/auth/clientCredentialsInput",
			"clientCredentialsInput must identify a declared oauthClientCredentials installation input",
		)
	}
}

func validateSensitiveValueMap(
	output *ManagedAgentImportAdmission,
	path string,
	values map[string]string,
	inputs map[string]mcpv1.InstallInput,
) {
	for name, value := range values {
		if !sensitiveCredentialName(name) {
			continue
		}

		references := placeholderReferences(value)
		if len(references) == 0 {
			output.addError(
				"agent.import.inline-mcp-credential",
				path+"/"+name,
				"credential-like MCP values must use declared secret installation inputs",
			)
			continue
		}

		for _, reference := range references {
			input, found := inputs[reference]
			if !found ||
				(input.Kind != "secret" &&
					input.Kind != "oauthClientCredentials") {
				output.addError(
					"agent.import.inline-mcp-credential-input",
					path+"/"+name,
					"credential placeholder must identify a declared secret installation input",
				)
			}
		}
	}
}

func sensitiveCredentialName(
	value string,
) bool {
	value = strings.ToLower(value)
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, "_", "")
	return strings.Contains(value, "authorization") ||
		strings.Contains(value, "password") ||
		strings.Contains(value, "secret") ||
		strings.Contains(value, "token") ||
		strings.Contains(value, "apikey") ||
		strings.Contains(value, "credential")
}

func placeholderReferences(
	value string,
) []string {
	seen := make(map[string]struct{})
	output := make([]string, 0)

	for remaining := value; ; {
		start := strings.Index(remaining, "${")
		if start < 0 {
			break
		}
		remaining = remaining[start+2:]
		end := strings.IndexByte(remaining, '}')
		if end < 0 {
			break
		}

		name := remaining[:end]
		remaining = remaining[end+1:]
		if basespec.LogicalName(name).Validate() != nil {
			continue
		}
		if _, duplicate := seen[name]; duplicate {
			continue
		}
		seen[name] = struct{}{}
		output = append(output, name)
	}
	sort.Strings(output)
	return output
}
