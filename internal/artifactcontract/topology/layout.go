// Package topology owns application-declared physical source naming and
// discovery topology. Schema validation remains owned by declaration codecs.
package topology

import (
	_ "embed"
	"fmt"
	"path"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

//go:embed contract_topology.yaml
var layoutYAML []byte

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

type decoderHintWire struct {
	Locator    basespec.Locator     `json:"locator"`
	Recursive  bool                 `json:"recursive"`
	DecoderIDs []basespec.DecoderID `json:"decoderIDs"`
}

type discoveryProfileWire struct {
	Root              basespec.Locator     `json:"root"`
	Recursive         bool                 `json:"recursive"`
	Authoritative     bool                 `json:"authoritative"`
	IncludePatterns   []string             `json:"includePatterns"`
	ExcludePatterns   []string             `json:"excludePatterns"`
	DocumentSets      []string             `json:"documentSets"`
	DecoderHints      []decoderHintWire    `json:"decoderHints"`
	AllowedDecoderIDs []basespec.DecoderID `json:"allowedDecoderIDs"`
}

type layoutWire struct {
	Documents struct {
		Collections       []documentAliasWire `json:"collections"`
		MCPConfigs        []documentAliasWire `json:"mcpConfigs"`
		Agents            []documentAliasWire `json:"agents"`
		Skills            []documentAliasWire `json:"skills"`
		WorkspaceMarkdown []documentAliasWire `json:"workspaceMarkdown"`
	} `json:"documents"`

	Discovery map[string]discoveryProfileWire `json:"discovery"`
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

	profiles map[string]source.DiscoverySpec
}

type aliasGroup struct {
	name    string
	aliases []documentAlias
}

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

func DiscoverySpec(
	name string,
) (source.DiscoverySpec, error) {
	value, found := configuredLayout.profiles[name]
	if !found {
		return source.DiscoverySpec{}, fmt.Errorf(
			"%w: topology has no discovery profile %q",
			basespec.ErrNotFound,
			name,
		)
	}
	return value.Clone(), nil
}

func MustDiscoverySpec(
	name string,
) source.DiscoverySpec {
	value, err := DiscoverySpec(name)
	if err != nil {
		panic(err)
	}
	return value
}

func DiscoverySpecAt(
	name string,
	root basespec.Locator,
) (source.DiscoverySpec, error) {
	if err := root.Validate(true); err != nil {
		return source.DiscoverySpec{}, err
	}

	value, err := DiscoverySpec(name)
	if err != nil {
		return source.DiscoverySpec{}, err
	}
	if len(value.DirectoryRoots) != 1 {
		return source.DiscoverySpec{}, fmt.Errorf(
			"%w: discovery profile %q cannot be rooted dynamically",
			basespec.ErrInvalid,
			name,
		)
	}

	value.DirectoryRoots[0].Root = root
	value = value.Normalized()
	if err := value.Validate(); err != nil {
		return source.DiscoverySpec{}, err
	}
	return value, nil
}

func DiscoverySpecForLocator(
	name string,
	locator basespec.Locator,
) (source.DiscoverySpec, error) {
	if err := locator.Validate(false); err != nil {
		return source.DiscoverySpec{}, err
	}

	profile, err := DiscoverySpec(name)
	if err != nil {
		return source.DiscoverySpec{}, err
	}
	if len(profile.AllowedDecoderIDs) != 1 {
		return source.DiscoverySpec{}, fmt.Errorf(
			"%w: discovery profile %q must define exactly one decoder",
			basespec.ErrInvalid,
			name,
		)
	}

	value := source.DiscoverySpec{
		ExplicitLocators: []basespec.Locator{locator},
		DecoderHints: []source.DecoderHint{{
			Locator:   locator,
			Recursive: false,
			DecoderIDs: append(
				[]basespec.DecoderID(nil),
				profile.AllowedDecoderIDs...,
			),
		}},
		AllowedDecoderIDs: append(
			[]basespec.DecoderID(nil),
			profile.AllowedDecoderIDs...,
		),
		Authoritative: profile.Authoritative,
	}
	value = value.Normalized()
	if err := value.Validate(); err != nil {
		return source.DiscoverySpec{}, err
	}
	return value, nil
}

func DiscoveryIncludePatterns(
	name string,
) ([]string, error) {
	value, err := DiscoverySpec(name)
	if err != nil {
		return nil, err
	}
	return discoveryIncludePatterns(value), nil
}

func SkillPackageDocumentFile() (basespec.Locator, error) {
	if len(configuredLayout.skills) != 1 {
		return "", fmt.Errorf(
			"%w: topology must define exactly one Skill package document filename",
			basespec.ErrInvalid,
		)
	}
	return configuredLayout.skills[0].locator, nil
}

func MustBuiltinDiscoverySpec() source.DiscoverySpec {
	return MustDiscoverySpec(
		"builtin",
	)
}

func MustWorkspaceDiscoverySpec() source.DiscoverySpec {
	return MustDiscoverySpec(
		"workspace",
	)
}

func discoveryIncludePatterns(
	value source.DiscoverySpec,
) []string {
	seen := make(map[string]struct{})
	output := make([]string, 0)
	for _, directory := range value.DirectoryRoots {
		for _, pattern := range directory.IncludePatterns {
			if _, duplicate := seen[pattern]; duplicate {
				continue
			}
			seen[pattern] = struct{}{}
			output = append(output, pattern)
		}
	}
	return output
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

	documentSets := map[string][]documentAlias{
		"collections":       collections,
		"mcpConfigs":        mcpConfigs,
		"agents":            agents,
		"skills":            skills,
		"workspaceMarkdown": workspaceMarkdown,
	}

	profiles, err := parseDiscoveryProfiles(
		wire.Discovery,
		documentSets,
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
		profiles:          profiles,
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

func parseDiscoveryProfiles(
	values map[string]discoveryProfileWire,
	documentSets map[string][]documentAlias,
) (map[string]source.DiscoverySpec, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf(
			"%w: topology has no discovery profiles",
			basespec.ErrInvalid,
		)
	}

	output := make(
		map[string]source.DiscoverySpec,
		len(values),
	)
	for name, value := range values {
		if err := basespec.ValidateIdentifier(
			"discovery profile name",
			name,
			basespec.MaxKindBytes,
		); err != nil {
			return nil, err
		}
		profile, err := parseDiscoveryProfile(
			name,
			value,
			documentSets,
		)
		if err != nil {
			return nil, err
		}
		output[name] = profile
	}
	return output, nil
}

func parseDiscoveryProfile(
	name string,
	value discoveryProfileWire,
	documentSets map[string][]documentAlias,
) (source.DiscoverySpec, error) {
	if err := value.Root.Validate(true); err != nil {
		return source.DiscoverySpec{}, err
	}

	var basePatterns []string
	var err error
	if len(value.IncludePatterns) != 0 {
		basePatterns, err = parseDiscoveryPatterns(
			name+" discovery include patterns",
			value.IncludePatterns,
		)
		if err != nil {
			return source.DiscoverySpec{}, err
		}
	}
	if err := basespec.ValidatePathPatterns(
		name+" discovery exclude patterns",
		value.ExcludePatterns,
	); err != nil {
		return source.DiscoverySpec{}, err
	}

	seenSets := make(
		map[string]struct{},
		len(value.DocumentSets),
	)
	selectedAliases := make([]documentAlias, 0)
	for _, setName := range value.DocumentSets {
		if _, duplicate := seenSets[setName]; duplicate {
			return source.DiscoverySpec{}, fmt.Errorf(
				"%w: discovery profile %q repeats document set %q",
				basespec.ErrInvalid,
				name,
				setName,
			)
		}
		seenSets[setName] = struct{}{}

		aliases, found := documentSets[setName]
		if !found {
			return source.DiscoverySpec{}, fmt.Errorf(
				"%w: discovery profile %q references unknown document set %q",
				basespec.ErrInvalid,
				name,
				setName,
			)
		}
		selectedAliases = append(selectedAliases, aliases...)
	}

	includePatterns := discoveryPatterns(
		basePatterns,
		selectedAliases,
	)
	if len(includePatterns) == 0 {
		return source.DiscoverySpec{}, fmt.Errorf(
			"%w: discovery profile %q has no include patterns",
			basespec.ErrInvalid,
			name,
		)
	}

	hints := make(
		[]source.DecoderHint,
		0,
		len(value.DecoderHints),
	)
	for _, hint := range value.DecoderHints {
		hints = append(hints, source.DecoderHint{
			Locator:    hint.Locator,
			Recursive:  hint.Recursive,
			DecoderIDs: append([]basespec.DecoderID(nil), hint.DecoderIDs...),
		})
	}

	output := source.DiscoverySpec{
		DirectoryRoots: []source.DirectoryRoot{{
			Root:            value.Root,
			Recursive:       value.Recursive,
			IncludePatterns: includePatterns,
			ExcludePatterns: append(
				[]string(nil),
				value.ExcludePatterns...,
			),
		}},
		DecoderHints: append(
			[]source.DecoderHint(nil),
			hints...,
		),
		AllowedDecoderIDs: append(
			[]basespec.DecoderID(nil),
			value.AllowedDecoderIDs...,
		),
		Authoritative: value.Authoritative,
	}
	output = output.Normalized()
	if err := output.Validate(); err != nil {
		return source.DiscoverySpec{}, err
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
