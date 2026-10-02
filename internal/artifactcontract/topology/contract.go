// Package topology owns declarative application Artifact contract topology.
package topology

import (
	_ "embed"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

//go:embed contract_topology.yaml
var contractTopologyYAML []byte

const (
	DocumentUseCollection                           = "collection"
	DocumentUseCanonicalYAML                        = "canonicalYAML"
	DocumentUseCanonicalJSON                        = "canonicalJSON"
	DocumentUseMCPConfig                            = "mcpConfig"
	DocumentUseSkillPackage                         = "skillPackage"
	DocumentUseModelProviderPackage                 = "modelProviderPackage"
	DocumentUseModelPackage                         = "modelPackage"
	DocumentUseToolPackage                          = "toolPackage"
	DocumentUseToolCollection                       = "toolCollection"
	DocumentUseManagedCollection                    = "managedCollection"
	DocumentUseManagedModelProvider                 = "managedModelProvider"
	DocumentUseManagedModel                         = "managedModel"
	DocumentUseAgentManagedCollection               = "agentManagedCollection"
	DocumentUseManagedAgent                         = "managedAgent"
	DocumentUseManagedMCP                           = "managedMCP"
	DocumentUseManagedMCPPolicy                     = "managedMCPPolicy"
	DocumentUseAgentMarkdown                        = "agentMarkdown"
	DocumentUseWorkspaceMarkdown                    = "workspaceMarkdown"
	DocumentUseWorkspaceInstructions                = "workspaceInstructions"
	DiscoveryUseSkill                               = "skill"
	DiscoveryUseMCP                                 = "mcp"
	DiscoveryUseWorkspace                           = "workspace"
	DiscoveryUseSelector                            = "selector"
	MarkdownRuleAgent                               = "agent"
	MarkdownRuleText                                = "text"
	MarkdownRuleDefaultText                         = "defaultText"
	MarkdownRuleInstructionText                     = "instructionText"
	RepositoryRootLocator             model.Locator = "."
)

type documentFormat string

const (
	formatJSON     documentFormat = "json"
	formatYAML     documentFormat = "yaml"
	formatMarkdown documentFormat = "markdown"
)

type documentAliasWire struct {
	Locator   string          `json:"locator"`
	Format    documentFormat  `json:"format"`
	DecoderID model.DecoderID `json:"decoderID,omitempty"`
}

type documentUseWire struct {
	DocumentSets   []string        `json:"documentSets"`
	DefaultLocator string          `json:"defaultLocator,omitempty"`
	DecoderID      model.DecoderID `json:"decoderID,omitempty"`
}

type markdownRuleWire struct {
	DocumentUses        []string `json:"documentUses,omitempty"`
	ExcludeDocumentUses []string `json:"excludeDocumentUses,omitempty"`
	BasenameSuffixes    []string `json:"basenameSuffixes,omitempty"`
	Extensions          []string `json:"extensions,omitempty"`
	ExcludeRules        []string `json:"excludeRules,omitempty"`
}

type decoderHintWire struct {
	Locator    model.Locator     `json:"locator"`
	Recursive  bool              `json:"recursive"`
	DecoderIDs []model.DecoderID `json:"decoderIDs"`
}

type discoveryProfileWire struct {
	Root              model.Locator     `json:"root"`
	Recursive         bool              `json:"recursive"`
	Authoritative     bool              `json:"authoritative"`
	IncludePatterns   []string          `json:"includePatterns"`
	ExcludePatterns   []string          `json:"excludePatterns"`
	DocumentSets      []string          `json:"documentSets"`
	DecoderHints      []decoderHintWire `json:"decoderHints"`
	AllowedDecoderIDs []model.DecoderID `json:"allowedDecoderIDs"`
}

type discoveryUseWire struct {
	Profiles []string `json:"profiles"`
}

type resolverTypePolicyWire struct {
	Type                          declaration.Type `json:"type"`
	SupportsSelectors             bool             `json:"supportsSelectors,omitempty"`
	SupportsMappedFallbackTargets bool             `json:"supportsMappedFallbackTargets,omitempty"`
}

type contractTopologyWire struct {
	PackageVersions struct {
		Unversioned model.LogicalVersion `json:"unversioned"`
	} `json:"packageVersions"`

	Documents     map[string][]documentAliasWire  `json:"documents"`
	DocumentUses  map[string]documentUseWire      `json:"documentUses"`
	MarkdownRules map[string]markdownRuleWire     `json:"markdownRules"`
	Discovery     map[string]discoveryProfileWire `json:"discovery"`
	DiscoveryUses map[string]discoveryUseWire     `json:"discoveryUses"`

	Resolver struct {
		Types []resolverTypePolicyWire `json:"types"`
	} `json:"resolver"`
}

type documentAlias struct {
	locator   model.Locator
	format    documentFormat
	decoderID model.DecoderID
}

type documentUse struct {
	aliases        []documentAlias
	defaultLocator model.Locator
	decoderID      model.DecoderID
}

type markdownRule struct {
	documentUses        []string
	excludeDocumentUses []string
	basenameSuffixes    []string
	extensions          []string
	excludeRules        []string
}

// ResolverTypePolicy contains resolver behavior that is declarative rather
// than intrinsic to a concrete declaration implementation.
type ResolverTypePolicy struct {
	Type                          declaration.Type
	SupportsSelectors             bool
	SupportsMappedFallbackTargets bool
}

type contractTopology struct {
	documentSets              map[string][]documentAlias
	documentUses              map[string]documentUse
	markdownRules             map[string]markdownRule
	discoveryProfiles         map[string]source.DiscoverySpec
	discoveryUses             map[string]source.DiscoverySpec
	resolverTypePolicies      []ResolverTypePolicy
	unversionedPackageVersion model.LogicalVersion
}

var configuredContractTopology = mustLoadContractTopology(contractTopologyYAML)

func DocumentFiles(use string) ([]model.Locator, error) {
	value, found := configuredContractTopology.documentUses[use]
	if !found {
		return nil, fmt.Errorf(
			"%w: contract topology has no document use %q",
			model.ErrNotFound,
			use,
		)
	}

	output := make([]model.Locator, len(value.aliases))
	for index, alias := range value.aliases {
		output[index] = alias.locator
	}
	return output, nil
}

func MustDocumentFiles(use string) []model.Locator {
	value, err := DocumentFiles(use)
	if err != nil {
		panic(err)
	}
	return value
}

func DefaultDocumentFile(use string) (model.Locator, error) {
	value, found := configuredContractTopology.documentUses[use]
	if !found {
		return "", fmt.Errorf(
			"%w: contract topology has no document use %q",
			model.ErrNotFound,
			use,
		)
	}
	if value.defaultLocator == "" {
		return "", fmt.Errorf(
			"%w: document use %q has no default locator",
			model.ErrInvalid,
			use,
		)
	}
	return value.defaultLocator, nil
}

func MustDefaultDocumentFile(use string) model.Locator {
	value, err := DefaultDocumentFile(use)
	if err != nil {
		panic(err)
	}
	return value
}

func DefaultDocumentDecoderID(use string) (model.DecoderID, error) {
	value, found := configuredContractTopology.documentUses[use]
	if !found {
		return "", fmt.Errorf(
			"%w: contract topology has no document use %q",
			model.ErrNotFound,
			use,
		)
	}
	if value.decoderID != "" {
		return value.decoderID, nil
	}
	for _, alias := range value.aliases {
		if alias.locator == value.defaultLocator && alias.decoderID != "" {
			return alias.decoderID, nil
		}
	}
	return "", fmt.Errorf(
		"%w: document use %q has no default decoder",
		model.ErrInvalid,
		use,
	)
}

func IsDocument(locator model.Locator, use string) bool {
	return configuredContractTopology.matchesDocument(locator, use, "")
}

func IsDocumentFormat(
	locator model.Locator,
	use string,
	format documentFormat,
) bool {
	return configuredContractTopology.matchesDocument(locator, use, format)
}

func CollectionDocumentFiles() []model.Locator {
	return MustDocumentFiles(DocumentUseCollection)
}

func IsCollectionDocumentFile(value model.Locator) bool {
	return path.Base(string(value)) == string(value) &&
		IsDocument(value, DocumentUseCollection)
}

func IsMCPConfigDocument(locator model.Locator) bool {
	return IsDocumentFormat(locator, DocumentUseMCPConfig, formatJSON)
}

func IsCanonicalYAMLDocument(locator model.Locator) bool {
	return IsDocumentFormat(locator, DocumentUseCanonicalYAML, formatYAML)
}

func IsCanonicalJSONDocument(locator model.Locator) bool {
	return IsDocumentFormat(locator, DocumentUseCanonicalJSON, formatJSON)
}

func SkillPackageDocumentFiles() []model.Locator {
	return MustDocumentFiles(DocumentUseSkillPackage)
}

func DefaultSkillPackageDocumentFile() model.Locator {
	return MustDefaultDocumentFile(DocumentUseSkillPackage)
}

func IsSkillPackageDocument(locator model.Locator) bool {
	return IsDocument(locator, DocumentUseSkillPackage)
}

func ToolPackageDocumentFiles() []model.Locator {
	return MustDocumentFiles(DocumentUseToolPackage)
}

func DefaultToolPackageDocumentFile() model.Locator {
	return MustDefaultDocumentFile(DocumentUseToolPackage)
}

func IsToolPackageDocument(locator model.Locator) bool {
	return IsDocument(locator, DocumentUseToolPackage)
}

func ModelProviderDocumentFiles() []model.Locator {
	return MustDocumentFiles(DocumentUseModelProviderPackage)
}

func ModelProviderDocumentFile() model.Locator {
	return MustDefaultDocumentFile(DocumentUseModelProviderPackage)
}

func ModelDocumentFiles() []model.Locator {
	return MustDocumentFiles(DocumentUseModelPackage)
}

func ModelDocumentFile() model.Locator {
	return MustDefaultDocumentFile(DocumentUseModelPackage)
}

func AgentDeclarationDocumentFiles() []model.Locator {
	return MustDocumentFiles(DocumentUseManagedAgent)
}

func IsAgentDeclarationDocument(locator model.Locator) bool {
	return IsDocument(locator, DocumentUseManagedAgent)
}

func IsAgentMarkdownDocument(locator model.Locator) bool {
	return MatchesMarkdownRule(MarkdownRuleAgent, locator)
}

func IsTextMarkdownDocument(locator model.Locator) bool {
	return MatchesMarkdownRule(MarkdownRuleText, locator)
}

func IsWorkspaceManifestLocator(locator model.Locator) bool {
	name := strings.ToLower(path.Base(string(locator)))
	if name == "workspace.yaml" || name == "workspace.yml" || name == "workspace.json" {
		return true
	}
	for _, suffix := range []string{".workspace.yaml", ".workspace.yml", ".workspace.json"} {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

func WorkspaceManifestPatterns() []string {
	return []string{
		"**/workspace.yaml",
		"**/workspace.yml",
		"**/workspace.json",
		"**/*.workspace.yaml",
		"**/*.workspace.yml",
		"**/*.workspace.json",
	}
}

func IsDefaultTextMarkdownDocument(locator model.Locator) bool {
	return MatchesMarkdownRule(MarkdownRuleDefaultText, locator)
}

func IsInstructionMarkdownDocument(locator model.Locator) bool {
	return MatchesMarkdownRule(MarkdownRuleInstructionText, locator)
}

func MatchesMarkdownRule(rule string, locator model.Locator) bool {
	return configuredContractTopology.matchesMarkdownRule(
		rule,
		locator,
		map[string]struct{}{},
	)
}

func UnversionedPackageVersion() model.LogicalVersion {
	return configuredContractTopology.unversionedPackageVersion
}

func ResolverTypePolicies() []ResolverTypePolicy {
	return append(
		[]ResolverTypePolicy(nil),
		configuredContractTopology.resolverTypePolicies...,
	)
}

func DiscoverySpecForUse(name string) (source.DiscoverySpec, error) {
	value, found := configuredContractTopology.discoveryUses[name]
	if !found {
		return source.DiscoverySpec{}, fmt.Errorf(
			"%w: contract topology has no discovery use %q",
			model.ErrNotFound,
			name,
		)
	}
	return value.Clone(), nil
}

func DiscoverySpecAtForUse(
	name string,
	root model.Locator,
) (source.DiscoverySpec, error) {
	if err := root.Validate(true); err != nil {
		return source.DiscoverySpec{}, err
	}

	value, err := DiscoverySpecForUse(name)
	if err != nil {
		return source.DiscoverySpec{}, err
	}
	if len(value.DirectoryRoots) != 1 {
		return source.DiscoverySpec{}, fmt.Errorf(
			"%w: discovery use %q cannot be rooted dynamically",
			model.ErrInvalid,
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

func DiscoverySpecForLocatorForUse(
	name string,
	locator model.Locator,
) (source.DiscoverySpec, error) {
	if err := locator.Validate(false); err != nil {
		return source.DiscoverySpec{}, err
	}

	profile, err := DiscoverySpecForUse(name)
	if err != nil {
		return source.DiscoverySpec{}, err
	}
	if len(profile.AllowedDecoderIDs) != 1 {
		return source.DiscoverySpec{}, fmt.Errorf(
			"%w: discovery use %q must define exactly one decoder",
			model.ErrInvalid,
			name,
		)
	}

	value := source.DiscoverySpec{
		ExplicitLocators: []model.Locator{locator},
		DecoderHints: []source.DecoderHint{{
			Locator:    locator,
			Recursive:  false,
			DecoderIDs: append([]model.DecoderID(nil), profile.AllowedDecoderIDs...),
		}},
		AllowedDecoderIDs: append([]model.DecoderID(nil), profile.AllowedDecoderIDs...),
		Authoritative:     profile.Authoritative,
	}
	value = value.Normalized()
	if err := value.Validate(); err != nil {
		return source.DiscoverySpec{}, err
	}
	return value, nil
}

func DiscoveryIncludePatternsForUse(name string) ([]string, error) {
	value, err := DiscoverySpecForUse(name)
	if err != nil {
		return nil, err
	}

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
	return output, nil
}

func (t contractTopology) matchesDocument(
	locator model.Locator,
	use string,
	format documentFormat,
) bool {
	value, found := t.documentUses[use]
	if !found {
		return false
	}

	name := path.Base(string(locator))
	for _, alias := range value.aliases {
		if format != "" && alias.format != format {
			continue
		}
		if strings.EqualFold(name, string(alias.locator)) {
			return true
		}
	}
	return false
}

func (t contractTopology) matchesMarkdownRule(
	name string,
	locator model.Locator,
	active map[string]struct{},
) bool {
	rule, found := t.markdownRules[name]
	if !found {
		return false
	}
	if _, recursive := active[name]; recursive {
		return false
	}
	active[name] = struct{}{}
	defer delete(active, name)

	filename := strings.ToLower(path.Base(string(locator)))
	matched := false
	for _, use := range rule.documentUses {
		if t.matchesDocument(locator, use, "") {
			matched = true
			break
		}
	}
	if !matched {
		for _, suffix := range rule.basenameSuffixes {
			if strings.HasSuffix(filename, suffix) {
				matched = true
				break
			}
		}
	}
	if !matched {
		extension := strings.ToLower(path.Ext(filename))
		if slices.Contains(rule.extensions, extension) {
			matched = true
		}
	}
	if !matched {
		return false
	}

	for _, use := range rule.excludeDocumentUses {
		if t.matchesDocument(locator, use, "") {
			return false
		}
	}
	for _, excluded := range rule.excludeRules {
		if t.matchesMarkdownRule(excluded, locator, active) {
			return false
		}
	}
	return true
}

func mustLoadContractTopology(raw []byte) contractTopology {
	value, err := loadContractTopology(raw)
	if err != nil {
		panic(fmt.Sprintf("invalid embedded contract.yaml: %v", err))
	}
	return value
}

func loadContractTopology(raw []byte) (contractTopology, error) {
	canonical, err := yamlutil.CanonicalObjectJSON(
		raw,
		model.MaxDefinitionBytes,
	)
	if err != nil {
		return contractTopology{}, err
	}

	var wire contractTopologyWire
	if err := jsonutil.DecodeCanonicalObjectBytesInto(
		canonical,
		&wire,
		model.MaxDefinitionBytes,
	); err != nil {
		return contractTopology{}, err
	}
	if wire.PackageVersions.Unversioned == "" {
		return contractTopology{}, fmt.Errorf(
			"%w: contract topology requires an unversioned package version",
			model.ErrInvalid,
		)
	}
	if err := wire.PackageVersions.Unversioned.Validate(false); err != nil {
		return contractTopology{}, err
	}

	documentSets, err := parseDocumentSets(wire.Documents)
	if err != nil {
		return contractTopology{}, err
	}
	documentUses, err := parseDocumentUses(wire.DocumentUses, documentSets)
	if err != nil {
		return contractTopology{}, err
	}
	markdownRules, err := parseMarkdownRules(wire.MarkdownRules, documentUses)
	if err != nil {
		return contractTopology{}, err
	}
	discoveryProfiles, err := parseDiscoveryProfiles(
		wire.Discovery,
		documentSets,
	)
	if err != nil {
		return contractTopology{}, err
	}
	discoveryUses, err := parseDiscoveryUses(
		wire.DiscoveryUses,
		discoveryProfiles,
	)
	if err != nil {
		return contractTopology{}, err
	}
	resolverPolicies, err := parseResolverTypePolicies(wire.Resolver.Types)
	if err != nil {
		return contractTopology{}, err
	}

	value := contractTopology{
		documentSets:              documentSets,
		documentUses:              documentUses,
		markdownRules:             markdownRules,
		discoveryProfiles:         discoveryProfiles,
		discoveryUses:             discoveryUses,
		resolverTypePolicies:      resolverPolicies,
		unversionedPackageVersion: wire.PackageVersions.Unversioned,
	}
	if err := validateRequiredContractBindings(value); err != nil {
		return contractTopology{}, err
	}
	return value, nil
}

func parseDocumentSets(
	values map[string][]documentAliasWire,
) (map[string][]documentAlias, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf(
			"%w: contract topology has no document sets",
			model.ErrInvalid,
		)
	}

	output := make(map[string][]documentAlias, len(values))
	for name, aliases := range values {
		if err := model.ValidateIdentifier(
			"document set name",
			name,
			model.MaxKindBytes,
		); err != nil {
			return nil, err
		}
		parsed, err := parseDocumentAliases(
			"documents."+name,
			aliases,
		)
		if err != nil {
			return nil, err
		}
		output[name] = parsed
	}
	return output, nil
}

func parseDocumentAliases(
	label string,
	values []documentAliasWire,
) ([]documentAlias, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf(
			"%w: %s must contain at least one document alias",
			model.ErrInvalid,
			label,
		)
	}

	seen := make(map[string]struct{}, len(values))
	output := make([]documentAlias, 0, len(values))
	for index, value := range values {
		if err := value.Format.validate(); err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", label, index, err)
		}
		if value.DecoderID != "" {
			if err := value.DecoderID.Validate(); err != nil {
				return nil, fmt.Errorf("%s[%d] decoder: %w", label, index, err)
			}
		}

		locator := model.Locator(value.Locator)
		if err := locator.ValidatePortable(false); err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", label, index, err)
		}
		if path.Base(string(locator)) != string(locator) {
			return nil, fmt.Errorf(
				"%w: %s[%d] must be a package-root filename",
				model.ErrInvalid,
				label,
				index,
			)
		}

		key := strings.ToLower(string(locator))
		if _, duplicate := seen[key]; duplicate {
			return nil, fmt.Errorf(
				"%w: %s repeats document alias %q",
				model.ErrInvalid,
				label,
				locator,
			)
		}
		seen[key] = struct{}{}
		output = append(output, documentAlias{
			locator:   locator,
			format:    value.Format,
			decoderID: value.DecoderID,
		})
	}
	return output, nil
}

func parseDocumentUses(
	values map[string]documentUseWire,
	documentSets map[string][]documentAlias,
) (map[string]documentUse, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf(
			"%w: contract topology has no document uses",
			model.ErrInvalid,
		)
	}

	output := make(map[string]documentUse, len(values))
	for name, value := range values {
		if err := model.ValidateIdentifier(
			"document use name",
			name,
			model.MaxKindBytes,
		); err != nil {
			return nil, err
		}
		if len(value.DocumentSets) == 0 {
			return nil, fmt.Errorf(
				"%w: document use %q has no document sets",
				model.ErrInvalid,
				name,
			)
		}
		if value.DecoderID != "" {
			if err := value.DecoderID.Validate(); err != nil {
				return nil, err
			}
		}

		seenSets := make(map[string]struct{}, len(value.DocumentSets))
		seenAliases := make(map[string]documentAlias)
		aliases := make([]documentAlias, 0)
		for _, setName := range value.DocumentSets {
			if _, duplicate := seenSets[setName]; duplicate {
				return nil, fmt.Errorf(
					"%w: document use %q repeats set %q",
					model.ErrInvalid,
					name,
					setName,
				)
			}
			seenSets[setName] = struct{}{}

			set, found := documentSets[setName]
			if !found {
				return nil, fmt.Errorf(
					"%w: document use %q references unknown set %q",
					model.ErrInvalid,
					name,
					setName,
				)
			}
			for _, alias := range set {
				key := strings.ToLower(string(alias.locator))
				if previous, duplicate := seenAliases[key]; duplicate {
					if previous.format != alias.format {
						return nil, fmt.Errorf(
							"%w: document use %q has conflicting formats for %q",
							model.ErrInvalid,
							name,
							alias.locator,
						)
					}
					continue
				}
				seenAliases[key] = alias
				aliases = append(aliases, alias)
			}
		}

		var defaultLocator model.Locator
		if value.DefaultLocator != "" {
			for _, alias := range aliases {
				if strings.EqualFold(
					value.DefaultLocator,
					string(alias.locator),
				) {
					defaultLocator = alias.locator
					break
				}
			}
			if defaultLocator == "" {
				return nil, fmt.Errorf(
					"%w: document use %q default %q is not declared",
					model.ErrInvalid,
					name,
					value.DefaultLocator,
				)
			}
		}

		output[name] = documentUse{
			aliases:        aliases,
			defaultLocator: defaultLocator,
			decoderID:      value.DecoderID,
		}
	}
	return output, nil
}

func parseMarkdownRules(
	values map[string]markdownRuleWire,
	documentUses map[string]documentUse,
) (map[string]markdownRule, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf(
			"%w: contract topology has no markdown rules",
			model.ErrInvalid,
		)
	}

	output := make(map[string]markdownRule, len(values))
	for name, value := range values {
		if err := model.ValidateIdentifier(
			"markdown rule name",
			name,
			model.MaxKindBytes,
		); err != nil {
			return nil, err
		}

		validateUse := func(use string) error {
			if _, found := documentUses[use]; !found {
				return fmt.Errorf(
					"%w: markdown rule %q references document use %q",
					model.ErrInvalid,
					name,
					use,
				)
			}
			return nil
		}
		for _, use := range value.DocumentUses {
			if err := validateUse(use); err != nil {
				return nil, err
			}
		}
		for _, use := range value.ExcludeDocumentUses {
			if err := validateUse(use); err != nil {
				return nil, err
			}
		}
		if len(value.DocumentUses) == 0 &&
			len(value.BasenameSuffixes) == 0 &&
			len(value.Extensions) == 0 {
			return nil, fmt.Errorf(
				"%w: markdown rule %q has no positive matcher",
				model.ErrInvalid,
				name,
			)
		}

		suffixes, err := normalizeMarkdownMatchers(
			"markdown basename suffix",
			value.BasenameSuffixes,
		)
		if err != nil {
			return nil, err
		}
		extensions, err := normalizeMarkdownMatchers(
			"markdown extension",
			value.Extensions,
		)
		if err != nil {
			return nil, err
		}
		for _, extension := range extensions {
			if !strings.HasPrefix(extension, ".") {
				return nil, fmt.Errorf(
					"%w: markdown extension %q must begin with a dot",
					model.ErrInvalid,
					extension,
				)
			}
		}

		output[name] = markdownRule{
			documentUses:        append([]string(nil), value.DocumentUses...),
			excludeDocumentUses: append([]string(nil), value.ExcludeDocumentUses...),
			basenameSuffixes:    suffixes,
			extensions:          extensions,
			excludeRules:        append([]string(nil), value.ExcludeRules...),
		}
	}
	if err := validateMarkdownRuleGraph(output); err != nil {
		return nil, err
	}
	return output, nil
}

func normalizeMarkdownMatchers(
	label string,
	values []string,
) ([]string, error) {
	seen := make(map[string]struct{}, len(values))
	output := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(value)
		if err := model.ValidateRequiredText(
			label,
			value,
			model.MaxLogicalNameBytes,
		); err != nil {
			return nil, err
		}
		if strings.Contains(value, "/") {
			return nil, fmt.Errorf(
				"%w: %s %q cannot contain a path separator",
				model.ErrInvalid,
				label,
				value,
			)
		}
		if _, duplicate := seen[value]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate %s %q",
				model.ErrInvalid,
				label,
				value,
			)
		}
		seen[value] = struct{}{}
		output = append(output, value)
	}
	return output, nil
}

func validateMarkdownRuleGraph(
	rules map[string]markdownRule,
) error {
	state := make(map[string]uint8, len(rules))
	var visit func(string) error
	visit = func(name string) error {
		switch state[name] {
		case 1:
			return fmt.Errorf(
				"%w: markdown rule exclusion cycle at %q",
				model.ErrInvalid,
				name,
			)
		case 2:
			return nil
		}

		rule, found := rules[name]
		if !found {
			return fmt.Errorf(
				"%w: markdown rule references unknown rule %q",
				model.ErrInvalid,
				name,
			)
		}
		state[name] = 1
		for _, excluded := range rule.excludeRules {
			if err := visit(excluded); err != nil {
				return err
			}
		}
		state[name] = 2
		return nil
	}

	for name := range rules {
		if err := visit(name); err != nil {
			return err
		}
	}
	return nil
}

func parseDiscoveryProfiles(
	values map[string]discoveryProfileWire,
	documentSets map[string][]documentAlias,
) (map[string]source.DiscoverySpec, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf(
			"%w: contract topology has no discovery profiles",
			model.ErrInvalid,
		)
	}

	output := make(map[string]source.DiscoverySpec, len(values))
	for name, value := range values {
		if err := model.ValidateIdentifier(
			"discovery profile name",
			name,
			model.MaxKindBytes,
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
	if err := model.ValidatePathPatterns(
		name+" discovery exclude patterns",
		value.ExcludePatterns,
	); err != nil {
		return source.DiscoverySpec{}, err
	}

	seenSets := make(map[string]struct{}, len(value.DocumentSets))
	selectedAliases := make([]documentAlias, 0)
	for _, setName := range value.DocumentSets {
		if _, duplicate := seenSets[setName]; duplicate {
			return source.DiscoverySpec{}, fmt.Errorf(
				"%w: discovery profile %q repeats document set %q",
				model.ErrInvalid,
				name,
				setName,
			)
		}
		seenSets[setName] = struct{}{}

		aliases, found := documentSets[setName]
		if !found {
			return source.DiscoverySpec{}, fmt.Errorf(
				"%w: discovery profile %q references unknown document set %q",
				model.ErrInvalid,
				name,
				setName,
			)
		}
		selectedAliases = append(selectedAliases, aliases...)
	}

	includePatterns := discoveryPatterns(basePatterns, selectedAliases)
	if len(includePatterns) == 0 {
		return source.DiscoverySpec{}, fmt.Errorf(
			"%w: discovery profile %q has no include patterns",
			model.ErrInvalid,
			name,
		)
	}

	hints := make([]source.DecoderHint, 0, len(value.DecoderHints))
	for _, hint := range value.DecoderHints {
		hints = append(hints, source.DecoderHint{
			Locator:    hint.Locator,
			Recursive:  hint.Recursive,
			DecoderIDs: append([]model.DecoderID(nil), hint.DecoderIDs...),
		})
	}

	output := source.DiscoverySpec{
		DirectoryRoots: []source.DirectoryRoot{{
			Root:            value.Root,
			Recursive:       value.Recursive,
			IncludePatterns: includePatterns,
			ExcludePatterns: append([]string(nil), value.ExcludePatterns...),
		}},
		DecoderHints:      hints,
		AllowedDecoderIDs: append([]model.DecoderID(nil), value.AllowedDecoderIDs...),
		Authoritative:     value.Authoritative,
	}
	output = output.Normalized()
	if err := output.Validate(); err != nil {
		return source.DiscoverySpec{}, err
	}
	return output, nil
}

func parseDiscoveryPatterns(
	label string,
	values []string,
) ([]string, error) {
	if err := model.ValidatePathPatterns(label, values); err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(values))
	output := make([]string, 0, len(values))
	for _, value := range values {
		if _, duplicate := seen[value]; duplicate {
			return nil, fmt.Errorf(
				"%w: %s repeats pattern %q",
				model.ErrInvalid,
				label,
				value,
			)
		}
		seen[value] = struct{}{}
		output = append(output, value)
	}
	return output, nil
}

func parseDiscoveryUses(
	values map[string]discoveryUseWire,
	profiles map[string]source.DiscoverySpec,
) (map[string]source.DiscoverySpec, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf(
			"%w: contract topology has no discovery uses",
			model.ErrInvalid,
		)
	}

	output := make(map[string]source.DiscoverySpec, len(values))
	for name, value := range values {
		if err := model.ValidateIdentifier(
			"discovery use name",
			name,
			model.MaxKindBytes,
		); err != nil {
			return nil, err
		}
		if len(value.Profiles) == 0 {
			return nil, fmt.Errorf(
				"%w: discovery use %q has no profiles",
				model.ErrInvalid,
				name,
			)
		}

		seen := make(map[string]struct{}, len(value.Profiles))
		selected := make([]source.DiscoverySpec, 0, len(value.Profiles))
		for _, profileName := range value.Profiles {
			if _, duplicate := seen[profileName]; duplicate {
				return nil, fmt.Errorf(
					"%w: discovery use %q repeats profile %q",
					model.ErrInvalid,
					name,
					profileName,
				)
			}
			seen[profileName] = struct{}{}
			profile, found := profiles[profileName]
			if !found {
				return nil, fmt.Errorf(
					"%w: discovery use %q references unknown profile %q",
					model.ErrInvalid,
					name,
					profileName,
				)
			}
			selected = append(selected, profile)
		}

		merged, err := mergeDiscoveryProfiles(name, selected)
		if err != nil {
			return nil, err
		}
		output[name] = merged
	}
	return output, nil
}

func mergeDiscoveryProfiles(
	name string,
	profiles []source.DiscoverySpec,
) (source.DiscoverySpec, error) {
	var (
		output        source.DiscoverySpec
		authoritative *bool
	)
	for _, profile := range profiles {
		value := profile.Clone()
		if authoritative == nil {
			next := value.Authoritative
			authoritative = &next
		} else if *authoritative != value.Authoritative {
			return source.DiscoverySpec{}, fmt.Errorf(
				"%w: discovery use %q mixes authoritative and non-authoritative profiles",
				model.ErrInvalid,
				name,
			)
		}
		output.DirectoryRoots = append(output.DirectoryRoots, value.DirectoryRoots...)
		output.ExplicitLocators = append(
			output.ExplicitLocators,
			value.ExplicitLocators...,
		)
		output.DecoderHints = append(output.DecoderHints, value.DecoderHints...)
		output.AllowedDecoderIDs = append(
			output.AllowedDecoderIDs,
			value.AllowedDecoderIDs...,
		)
	}
	if authoritative != nil {
		output.Authoritative = *authoritative
	}
	output = output.Normalized()
	if err := output.Validate(); err != nil {
		return source.DiscoverySpec{}, err
	}
	return output, nil
}

func parseResolverTypePolicies(
	values []resolverTypePolicyWire,
) ([]ResolverTypePolicy, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf(
			"%w: contract topology has no resolver type policies",
			model.ErrInvalid,
		)
	}

	seen := make(map[declaration.Type]struct{}, len(values))
	output := make([]ResolverTypePolicy, 0, len(values))
	for index, value := range values {
		if err := value.Type.Validate(); err != nil {
			return nil, fmt.Errorf("resolver types[%d]: %w", index, err)
		}
		if _, duplicate := seen[value.Type]; duplicate {
			return nil, fmt.Errorf(
				"%w: resolver type policy repeats %q",
				model.ErrInvalid,
				value.Type,
			)
		}
		seen[value.Type] = struct{}{}
		output = append(output, ResolverTypePolicy(value))
	}
	for _, declarationType := range declaration.Types() {
		if _, found := seen[declarationType]; !found {
			return nil, fmt.Errorf(
				"%w: contract topology has no resolver policy for %q",
				model.ErrInvalid,
				declarationType,
			)
		}
	}
	return output, nil
}

func validateRequiredContractBindings(value contractTopology) error {
	for _, use := range []string{
		DocumentUseCollection,
		DocumentUseCanonicalYAML,
		DocumentUseCanonicalJSON,
		DocumentUseMCPConfig,
		DocumentUseSkillPackage,
		DocumentUseModelProviderPackage,
		DocumentUseModelPackage,
		DocumentUseToolPackage,
		DocumentUseToolCollection,
		DocumentUseManagedCollection,
		DocumentUseManagedModelProvider,
		DocumentUseManagedModel,
		DocumentUseAgentManagedCollection,
		DocumentUseManagedAgent,
		DocumentUseManagedMCP,
		DocumentUseManagedMCPPolicy,
		DocumentUseAgentMarkdown,
		DocumentUseWorkspaceMarkdown,
		DocumentUseWorkspaceInstructions,
	} {
		if _, found := value.documentUses[use]; !found {
			return fmt.Errorf(
				"%w: required document use %q is missing",
				model.ErrInvalid,
				use,
			)
		}
	}
	for _, use := range []string{
		DocumentUseSkillPackage,
		DocumentUseModelProviderPackage,
		DocumentUseModelPackage,
		DocumentUseToolPackage,
		DocumentUseToolCollection,
		DocumentUseManagedCollection,
		DocumentUseManagedModelProvider,
		DocumentUseManagedModel,
		DocumentUseAgentManagedCollection,
		DocumentUseManagedAgent,
		DocumentUseManagedMCP,
		DocumentUseManagedMCPPolicy,
	} {
		if _, err := value.defaultDocumentFile(use); err != nil {
			return err
		}
		if _, err := value.defaultDocumentDecoderID(use); err != nil {
			return err
		}
	}
	for _, name := range []string{
		MarkdownRuleAgent,
		MarkdownRuleText,
		MarkdownRuleDefaultText,
		MarkdownRuleInstructionText,
	} {
		if _, found := value.markdownRules[name]; !found {
			return fmt.Errorf(
				"%w: required markdown rule %q is missing",
				model.ErrInvalid,
				name,
			)
		}
	}
	for _, name := range []string{
		DiscoveryUseSkill,
		DiscoveryUseMCP,
		DiscoveryUseWorkspace,
		DiscoveryUseSelector,
	} {
		if _, found := value.discoveryUses[name]; !found {
			return fmt.Errorf(
				"%w: required discovery use %q is missing",
				model.ErrInvalid,
				name,
			)
		}
	}
	return nil
}

func (t contractTopology) defaultDocumentFile(
	use string,
) (model.Locator, error) {
	value, found := t.documentUses[use]
	if !found || value.defaultLocator == "" {
		return "", fmt.Errorf(
			"%w: document use %q has no default locator",
			model.ErrInvalid,
			use,
		)
	}
	return value.defaultLocator, nil
}

func (t contractTopology) defaultDocumentDecoderID(
	use string,
) (model.DecoderID, error) {
	value, found := t.documentUses[use]
	if !found {
		return "", fmt.Errorf(
			"%w: document use %q is unknown",
			model.ErrInvalid,
			use,
		)
	}
	if value.decoderID != "" {
		return value.decoderID, nil
	}
	for _, alias := range value.aliases {
		if alias.locator == value.defaultLocator && alias.decoderID != "" {
			return alias.decoderID, nil
		}
	}
	return "", fmt.Errorf(
		"%w: document use %q has no default decoder",
		model.ErrInvalid,
		use,
	)
}

func (f documentFormat) validate() error {
	switch f {
	case formatJSON, formatYAML, formatMarkdown:
		return nil
	default:
		return fmt.Errorf(
			"%w: unsupported document format %q",
			model.ErrInvalid,
			f,
		)
	}
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
