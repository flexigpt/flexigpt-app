// Package topology owns application-declared physical source naming and
// discovery topology. Schema validation remains owned by declaration codecs.
package topology

import (
	_ "embed"
	"fmt"
	"path"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

const layoutVersion = "v1"

type documentFormat string

const (
	formatJSON     documentFormat = "json"
	formatYAML     documentFormat = "yaml"
	formatMarkdown documentFormat = "markdown"
)

type documentAliasWire struct {
	Locator string         `json:"locator"`
	Format  documentFormat `json:"format"`
}

type layoutWire struct {
	Version string `json:"version"`

	Documents struct {
		Collections       []documentAliasWire `json:"collections"`
		MCPConfigs        []documentAliasWire `json:"mcpConfigs"`
		Agents            []documentAliasWire `json:"agents"`
		Skills            []documentAliasWire `json:"skills"`
		WorkspaceMarkdown []documentAliasWire `json:"workspaceMarkdown"`
	} `json:"documents"`

	Discovery struct {
		Workspace []string `json:"workspace"`
		Selector  []string `json:"selector"`
	} `json:"discovery"`
}

type documentAlias struct {
	locator basespec.Locator
	format  documentFormat
}

type layout struct {
	collections       []documentAlias
	mcpConfigs        []documentAlias
	agents            []documentAlias
	skills            []documentAlias
	workspaceMarkdown []documentAlias
	workspacePatterns []string
	selectorPatterns  []string
}

type aliasGroup struct {
	name    string
	aliases []documentAlias
}

//go:embed topology.yaml
var layoutYAML []byte

var configuredLayout = mustLoadLayout(layoutYAML)

// CollectionDocumentFiles returns accepted package-root filenames for a
// canonical Plugin declaration used as a Collection.
func CollectionDocumentFiles() []basespec.Locator {
	return cloneLocators(configuredLayout.collections)
}

// MCPConfigDocumentFiles returns accepted native MCP configuration filenames.
func MCPConfigDocumentFiles() []basespec.Locator {
	return cloneLocators(configuredLayout.mcpConfigs)
}

// IsCollectionDocumentFile reports whether value is an exact package-root
// Collection declaration filename.
func IsCollectionDocumentFile(
	value basespec.Locator,
) bool {
	if path.Base(string(value)) != string(value) {
		return false
	}
	for _, candidate := range configuredLayout.collections {
		if candidate.locator == value {
			return true
		}
	}
	return false
}

// IsMCPConfigDocument reports whether the basename of locator is one of the
// configured native MCP configuration names.
func IsMCPConfigDocument(
	locator basespec.Locator,
) bool {
	return aliasesMatch(
		locator,
		formatJSON,
		configuredLayout.mcpConfigs,
	)
}

// IsCanonicalYAMLDocument reports whether an explicitly configured canonical
// declaration filename must be offered to the YAML declaration decoder even
// when it has a nonstandard extension.
func IsCanonicalYAMLDocument(
	locator basespec.Locator,
) bool {
	return aliasesMatch(
		locator,
		formatYAML,
		configuredLayout.collections,
		configuredLayout.agents,
	)
}

// IsCanonicalJSONDocument reports whether an explicitly configured canonical
// declaration filename must be offered to the JSON declaration decoder even
// when it has a nonstandard extension.
func IsCanonicalJSONDocument(
	locator basespec.Locator,
) bool {
	return aliasesMatch(
		locator,
		formatJSON,
		configuredLayout.collections,
		configuredLayout.agents,
	)
}

// WorkspaceDiscoveryIncludePatterns returns the default filesystem Workspace
// source discovery patterns.
func WorkspaceDiscoveryIncludePatterns() []string {
	return discoveryPatterns(
		configuredLayout.workspacePatterns,
		configuredLayout.collections,
		configuredLayout.mcpConfigs,
		configuredLayout.agents,
		configuredLayout.skills,
		configuredLayout.workspaceMarkdown,
	)
}

// SelectorDiscoveryIncludePatterns returns the default discovery patterns
// used while expanding a selector or local declaration-locator closure.
func SelectorDiscoveryIncludePatterns() []string {
	return discoveryPatterns(
		configuredLayout.selectorPatterns,
		configuredLayout.collections,
		configuredLayout.mcpConfigs,
		configuredLayout.agents,
		configuredLayout.skills,
	)
}

// BuiltinDiscoveryIncludePatterns returns the discovery patterns required by
// the protected built-in Source.
func BuiltinDiscoveryIncludePatterns() []string {
	return discoveryPatterns(
		nil,
		configuredLayout.collections,
		configuredLayout.mcpConfigs,
		configuredLayout.agents,
		configuredLayout.skills,
	)
}

func mustLoadLayout(
	raw []byte,
) layout {
	value, err := loadLayout(raw)
	if err != nil {
		panic(fmt.Sprintf("invalid embedded topology.yaml: %v", err))
	}
	return value
}

func loadLayout(
	raw []byte,
) (layout, error) {
	canonical, err := yamlutil.CanonicalObjectJSON(
		raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return layout{}, err
	}

	var wire layoutWire
	if err := jsonutil.DecodeCanonicalObjectBytesInto(
		canonical,
		&wire,
		basespec.MaxDefinitionBytes,
	); err != nil {
		return layout{}, err
	}
	if wire.Version != layoutVersion {
		return layout{}, fmt.Errorf(
			"%w: unsupported source naming topology version %q",
			basespec.ErrInvalid,
			wire.Version,
		)
	}

	collections, err := parseAliasGroup(
		"documents.collections",
		wire.Documents.Collections,
		formatJSON,
		formatYAML,
	)
	if err != nil {
		return layout{}, err
	}
	mcpConfigs, err := parseAliasGroup(
		"documents.mcpConfigs",
		wire.Documents.MCPConfigs,
		formatJSON,
	)
	if err != nil {
		return layout{}, err
	}
	agents, err := parseAliasGroup(
		"documents.agents",
		wire.Documents.Agents,
		formatJSON,
		formatYAML,
	)
	if err != nil {
		return layout{}, err
	}
	skills, err := parseAliasGroup(
		"documents.skills",
		wire.Documents.Skills,
		formatMarkdown,
	)
	if err != nil {
		return layout{}, err
	}
	workspaceMarkdown, err := parseAliasGroup(
		"documents.workspaceMarkdown",
		wire.Documents.WorkspaceMarkdown,
		formatMarkdown,
	)
	if err != nil {
		return layout{}, err
	}
	workspacePatterns, err := parseDiscoveryPatterns(
		"workspace discovery patterns",
		wire.Discovery.Workspace,
	)
	if err != nil {
		return layout{}, err
	}
	selectorPatterns, err := parseDiscoveryPatterns(
		"selector discovery patterns",
		wire.Discovery.Selector,
	)
	if err != nil {
		return layout{}, err
	}

	if err := validateDistinctAliases(
		aliasGroup{name: "collections", aliases: collections},
		aliasGroup{name: "mcpConfigs", aliases: mcpConfigs},
		aliasGroup{name: "agents", aliases: agents},
		aliasGroup{name: "skills", aliases: skills},
		aliasGroup{
			name:    "workspaceMarkdown",
			aliases: workspaceMarkdown,
		},
	); err != nil {
		return layout{}, err
	}

	return layout{
		collections:       collections,
		mcpConfigs:        mcpConfigs,
		agents:            agents,
		skills:            skills,
		workspaceMarkdown: workspaceMarkdown,
		workspacePatterns: workspacePatterns,
		selectorPatterns:  selectorPatterns,
	}, nil
}

func parseAliasGroup(
	label string,
	values []documentAliasWire,
	allowed ...documentFormat,
) ([]documentAlias, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf(
			"%w: %s must contain at least one document name",
			basespec.ErrInvalid,
			label,
		)
	}

	allowedFormats := make(map[documentFormat]struct{}, len(allowed))
	for _, value := range allowed {
		allowedFormats[value] = struct{}{}
	}
	seen := make(map[string]struct{}, len(values))
	output := make([]documentAlias, 0, len(values))
	for index, value := range values {
		if err := value.Format.validate(); err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", label, index, err)
		}
		if _, found := allowedFormats[value.Format]; !found {
			return nil, fmt.Errorf(
				"%w: %s[%d] has unsupported format %q",
				basespec.ErrInvalid,
				label,
				index,
				value.Format,
			)
		}

		locator := basespec.Locator(value.Locator)
		if err := locator.ValidatePortable(false); err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", label, index, err)
		}
		if path.Base(string(locator)) != string(locator) {
			return nil, fmt.Errorf(
				"%w: %s[%d] must be a package-root filename",
				basespec.ErrInvalid,
				label,
				index,
			)
		}

		key := strings.ToLower(string(locator))
		if _, duplicate := seen[key]; duplicate {
			return nil, fmt.Errorf(
				"%w: %s repeats document name %q",
				basespec.ErrInvalid,
				label,
				locator,
			)
		}
		seen[key] = struct{}{}
		output = append(output, documentAlias{
			locator: locator,
			format:  value.Format,
		})
	}
	return output, nil
}

func parseDiscoveryPatterns(
	label string,
	values []string,
) ([]string, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf(
			"%w: %s must contain at least one pattern",
			basespec.ErrInvalid,
			label,
		)
	}
	if err := basespec.ValidatePathPatterns(label, values); err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(values))
	output := make([]string, 0, len(values))
	for _, value := range values {
		if _, duplicate := seen[value]; duplicate {
			return nil, fmt.Errorf(
				"%w: %s repeats pattern %q",
				basespec.ErrInvalid,
				label,
				value,
			)
		}
		seen[value] = struct{}{}
		output = append(output, value)
	}
	return output, nil
}

func validateDistinctAliases(
	groups ...aliasGroup,
) error {
	seen := make(map[string]string)
	for _, group := range groups {
		for _, alias := range group.aliases {
			key := strings.ToLower(string(alias.locator))
			if previous, duplicate := seen[key]; duplicate {
				return fmt.Errorf(
					"%w: document name %q belongs to both %s and %s",
					basespec.ErrInvalid,
					alias.locator,
					previous,
					group.name,
				)
			}
			seen[key] = group.name
		}
	}
	return nil
}

func (f documentFormat) validate() error {
	switch f {
	case formatJSON, formatYAML, formatMarkdown:
		return nil
	default:
		return fmt.Errorf(
			"%w: unsupported document format %q",
			basespec.ErrInvalid,
			f,
		)
	}
}

func cloneLocators(
	values []documentAlias,
) []basespec.Locator {
	output := make([]basespec.Locator, len(values))
	for index, value := range values {
		output[index] = value.locator
	}
	return output
}

func aliasesMatch(
	locator basespec.Locator,
	format documentFormat,
	groups ...[]documentAlias,
) bool {
	name := path.Base(string(locator))
	for _, group := range groups {
		for _, alias := range group {
			if alias.format == format &&
				strings.EqualFold(name, string(alias.locator)) {
				return true
			}
		}
	}
	return false
}

func discoveryPatterns(
	base []string,
	groups ...[]documentAlias,
) []string {
	seen := make(map[string]struct{})
	output := make([]string, 0, len(base)+16)
	appendPattern := func(value string) {
		if _, duplicate := seen[value]; duplicate {
			return
		}
		seen[value] = struct{}{}
		output = append(output, value)
	}

	for _, value := range base {
		appendPattern(value)
	}
	for _, group := range groups {
		for _, alias := range group {
			appendPattern(string(alias.locator))
			appendPattern("**/" + string(alias.locator))
		}
	}
	return output
}
