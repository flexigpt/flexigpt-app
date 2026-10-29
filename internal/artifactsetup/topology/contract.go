// Package topology owns declarative application Artifact contract topology.
package topology

import (
	_ "embed"
	"fmt"
	"path"
	"slices"
	"strings"

	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

//go:embed contract_topology.yaml
var contractTopologyYAML []byte

const (
	DocumentUsePlugin                                  = "plugin"
	DocumentUseCanonicalYAML                           = "canonicalYAML"
	DocumentUseCanonicalJSON                           = "canonicalJSON"
	DocumentUseMCPConfig                               = "mcpConfig"
	DocumentUseSkillPackage                            = "skillPackage"
	DocumentUseModelProviderPackage                    = "modelProviderPackage"
	DocumentUseModelPackage                            = "modelPackage"
	DocumentUseToolPackage                             = "toolPackage"
	DocumentUseToolPlugin                              = "toolPlugin"
	DocumentUseManagedPlugin                           = "managedPlugin"
	DocumentUseManagedModelProvider                    = "managedModelProvider"
	DocumentUseManagedModel                            = "managedModel"
	DocumentUseAgentManagedPlugin                      = "agentManagedPlugin"
	DocumentUseManagedAgent                            = "managedAgent"
	DocumentUseManagedMCP                              = "managedMCP"
	DocumentUseManagedMCPPolicy                        = "managedMCPPolicy"
	DocumentUseAgentMarkdown                           = "agentMarkdown"
	DocumentUseWorkspaceMarkdown                       = "workspaceMarkdown"
	DocumentUseWorkspaceInstructions                   = "workspaceInstructions"
	DocumentUseWorkspaceManifest                       = "workspaceManifest"
	DiscoveryUseSkill                                  = "skill"
	DiscoveryUseMCP                                    = "mcp"
	DiscoveryUseWorkspace                              = "workspace"
	MarkdownRuleAgent                                  = "agent"
	MarkdownRuleText                                   = "text"
	MarkdownRuleDefaultText                            = "defaultText"
	MarkdownRuleInstructionText                        = "instructionText"
	RepositoryRootLocator            storeSpec.Locator = "."
)

type documentFormat string

const (
	formatJSON     documentFormat = "json"
	formatYAML     documentFormat = "yaml"
	formatMarkdown documentFormat = "markdown"
)

type documentAliasWire struct {
	Locator   string              `json:"locator"`
	Pattern   string              `json:"pattern,omitempty"`
	Format    documentFormat      `json:"format"`
	DecoderID storeSpec.DecoderID `json:"decoderID,omitempty"`
}

type documentUseWire struct {
	DocumentSets   []string            `json:"documentSets"`
	DefaultLocator string              `json:"defaultLocator,omitempty"`
	DecoderID      storeSpec.DecoderID `json:"decoderID,omitempty"`
}

type markdownRuleWire struct {
	DocumentUses        []string `json:"documentUses,omitempty"`
	ExcludeDocumentUses []string `json:"excludeDocumentUses,omitempty"`
	BasenameSuffixes    []string `json:"basenameSuffixes,omitempty"`
	Extensions          []string `json:"extensions,omitempty"`
	ExcludeRules        []string `json:"excludeRules,omitempty"`
}

type decoderHintWire struct {
	Locator    storeSpec.Locator     `json:"locator"`
	Recursive  bool                  `json:"recursive"`
	DecoderIDs []storeSpec.DecoderID `json:"decoderIDs"`
}

type discoveryProfileWire struct {
	Root              storeSpec.Locator     `json:"root"`
	Recursive         bool                  `json:"recursive"`
	Authoritative     bool                  `json:"authoritative"`
	IncludePatterns   []string              `json:"includePatterns"`
	ExcludePatterns   []string              `json:"excludePatterns"`
	DocumentSets      []string              `json:"documentSets"`
	DecoderHints      []decoderHintWire     `json:"decoderHints"`
	AllowedDecoderIDs []storeSpec.DecoderID `json:"allowedDecoderIDs"`
}

type discoveryUseWire struct {
	Profiles []string `json:"profiles"`
}

type workspaceSelectorDiscoveryWire struct {
	DocumentUses []string `json:"documentUses"`
}

type workspaceDiscoveryWire struct {
	BootstrapDiscoveryUse string                                    `json:"bootstrapDiscoveryUse"`
	SelectorTypes         map[string]workspaceSelectorDiscoveryWire `json:"selectorTypes"`
}

type contractTopologyWire struct {
	PackageVersions struct {
		Unversioned storeSpec.LogicalVersion `json:"unversioned"`
	} `json:"packageVersions"`

	Documents          map[string][]documentAliasWire  `json:"documents"`
	DocumentUses       map[string]documentUseWire      `json:"documentUses"`
	MarkdownRules      map[string]markdownRuleWire     `json:"markdownRules"`
	Discovery          map[string]discoveryProfileWire `json:"discovery"`
	DiscoveryUses      map[string]discoveryUseWire     `json:"discoveryUses"`
	WorkspaceDiscovery workspaceDiscoveryWire          `json:"workspaceDiscovery"`
}

type documentAlias struct {
	locator   storeSpec.Locator
	pattern   string
	format    documentFormat
	decoderID storeSpec.DecoderID
}

type documentUse struct {
	aliases        []documentAlias
	defaultLocator storeSpec.Locator
	decoderID      storeSpec.DecoderID
}

type markdownRule struct {
	documentUses        []string
	excludeDocumentUses []string
	basenameSuffixes    []string
	extensions          []string
	excludeRules        []string
}

type contractTopology struct {
	documentSets              map[string][]documentAlias
	documentUses              map[string]documentUse
	markdownRules             map[string]markdownRule
	discoveryProfiles         map[string]sourceModel.DiscoverySpec
	discoveryUses             map[string]sourceModel.DiscoverySpec
	workspaceBootstrap        sourceModel.DiscoverySpec
	workspaceSelectorPatterns map[string][]string
	unversionedPackageVersion storeSpec.LogicalVersion
}

var configuredContractTopology = mustLoadContractTopology(contractTopologyYAML)

func WorkspaceBootstrapDiscoverySpec() (sourceModel.DiscoverySpec, error) {
	value := configuredContractTopology.workspaceBootstrap.Clone()
	if err := value.Validate(); err != nil {
		return sourceModel.DiscoverySpec{}, err
	}
	return value, nil
}

func WorkspaceSelectorPatterns(
	declarationType string,
) ([]string, error) {
	if err := storeSpec.ValidateRequiredText(
		"Workspace selector declaration type",
		declarationType,
		storeSpec.MaxKindBytes,
	); err != nil {
		return nil, err
	}

	patterns, found := configuredContractTopology.
		workspaceSelectorPatterns[declarationType]
	if !found {
		return nil, fmt.Errorf(
			"%w: Workspace discovery has no selector patterns for type %q",
			storeSpec.ErrNotFound,
			declarationType,
		)
	}
	return append([]string(nil), patterns...), nil
}

func DocumentFiles(use string) ([]storeSpec.Locator, error) {
	value, found := configuredContractTopology.documentUses[use]
	if !found {
		return nil, fmt.Errorf(
			"%w: contract topology has no document use %q",
			storeSpec.ErrNotFound,
			use,
		)
	}

	output := make([]storeSpec.Locator, len(value.aliases))
	for index, alias := range value.aliases {
		if alias.locator == "" {
			continue
		}
		output[index] = alias.locator
	}
	return slices.DeleteFunc(output, func(value storeSpec.Locator) bool {
		return value == ""
	}), nil
}

// DocumentPatterns returns every Source-discovery pattern accepted for one
// declarative document use. Exact aliases are returned for both a Source root
// and descendants; wildcard aliases remain application-selected YAML data.
func DocumentPatterns(use string) ([]string, error) {
	value, found := configuredContractTopology.documentUses[use]
	if !found {
		return nil, fmt.Errorf("%w: contract topology has no document use %q", storeSpec.ErrNotFound, use)
	}
	return documentPatterns(value.aliases), nil
}

// DocumentPatternsForDecoder returns every application-declared source
// filename or pattern for one decoder. LLM decoder packages consume this
// support declaration rather than inferring support from filename extensions.
func DocumentPatternsForDecoder(
	decoderID storeSpec.DecoderID,
) ([]string, error) {
	if err := decoderID.Validate(); err != nil {
		return nil, err
	}

	uses := make([]string, 0, len(configuredContractTopology.documentUses))
	for use := range configuredContractTopology.documentUses {
		uses = append(uses, use)
	}
	slices.Sort(uses)

	seen := make(map[string]struct{})
	output := make([]string, 0)
	for _, useName := range uses {
		use := configuredContractTopology.documentUses[useName]
		for _, alias := range use.aliases {
			effectiveDecoder := alias.decoderID
			if effectiveDecoder == "" {
				effectiveDecoder = use.decoderID
			}

			if effectiveDecoder != decoderID {
				continue
			}
			for _, pattern := range documentPatterns(
				[]documentAlias{alias},
			) {
				if _, duplicate := seen[pattern]; duplicate {
					continue
				}
				seen[pattern] = struct{}{}
				output = append(output, pattern)
			}
		}
	}
	return output, nil
}

func MustDocumentFiles(use string) []storeSpec.Locator {
	value, err := DocumentFiles(use)
	if err != nil {
		panic(err)
	}
	return value
}

func DefaultDocumentFile(use string) (storeSpec.Locator, error) {
	value, found := configuredContractTopology.documentUses[use]
	if !found {
		return "", fmt.Errorf(
			"%w: contract topology has no document use %q",
			storeSpec.ErrNotFound,
			use,
		)
	}
	if value.defaultLocator == "" {
		return "", fmt.Errorf(
			"%w: document use %q has no default locator",
			storeSpec.ErrInvalid,
			use,
		)
	}
	return value.defaultLocator, nil
}

func MustDefaultDocumentFile(use string) storeSpec.Locator {
	value, err := DefaultDocumentFile(use)
	if err != nil {
		panic(err)
	}
	return value
}

func DefaultDocumentDecoderID(use string) (storeSpec.DecoderID, error) {
	value, found := configuredContractTopology.documentUses[use]
	if !found {
		return "", fmt.Errorf(
			"%w: contract topology has no document use %q",
			storeSpec.ErrNotFound,
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
		storeSpec.ErrInvalid,
		use,
	)
}

func IsDocument(locator storeSpec.Locator, use string) bool {
	return configuredContractTopology.matchesDocument(locator, use, "")
}

func IsDocumentFormat(
	locator storeSpec.Locator,
	use string,
	format documentFormat,
) bool {
	return configuredContractTopology.matchesDocument(locator, use, format)
}

func PluginDocumentFiles() []storeSpec.Locator {
	return MustDocumentFiles(DocumentUsePlugin)
}

func IsPluginDocumentFile(value storeSpec.Locator) bool {
	return path.Base(string(value)) == string(value) &&
		IsDocument(value, DocumentUsePlugin)
}

func IsMCPConfigDocument(locator storeSpec.Locator) bool {
	return IsDocumentFormat(locator, DocumentUseMCPConfig, formatJSON)
}

func IsCanonicalYAMLDocument(locator storeSpec.Locator) bool {
	return IsDocumentFormat(locator, DocumentUseCanonicalYAML, formatYAML)
}

func IsCanonicalJSONDocument(locator storeSpec.Locator) bool {
	return IsDocumentFormat(locator, DocumentUseCanonicalJSON, formatJSON)
}

func SkillPackageDocumentFiles() []storeSpec.Locator {
	return MustDocumentFiles(DocumentUseSkillPackage)
}

func DefaultSkillPackageDocumentFile() storeSpec.Locator {
	return MustDefaultDocumentFile(DocumentUseSkillPackage)
}

func IsSkillPackageDocument(locator storeSpec.Locator) bool {
	return IsDocument(locator, DocumentUseSkillPackage)
}

func ToolPackageDocumentFiles() []storeSpec.Locator {
	return MustDocumentFiles(DocumentUseToolPackage)
}

func DefaultToolPackageDocumentFile() storeSpec.Locator {
	return MustDefaultDocumentFile(DocumentUseToolPackage)
}

func IsToolPackageDocument(locator storeSpec.Locator) bool {
	return IsDocument(locator, DocumentUseToolPackage)
}

func ModelProviderDocumentFiles() []storeSpec.Locator {
	return MustDocumentFiles(DocumentUseModelProviderPackage)
}

func ModelProviderDocumentFile() storeSpec.Locator {
	return MustDefaultDocumentFile(DocumentUseModelProviderPackage)
}

func ModelDocumentFiles() []storeSpec.Locator {
	return MustDocumentFiles(DocumentUseModelPackage)
}

func ModelDocumentFile() storeSpec.Locator {
	return MustDefaultDocumentFile(DocumentUseModelPackage)
}

func AgentDeclarationDocumentFiles() []storeSpec.Locator {
	return MustDocumentFiles(DocumentUseManagedAgent)
}

func IsAgentDeclarationDocument(locator storeSpec.Locator) bool {
	return IsDocument(locator, DocumentUseManagedAgent)
}

func IsAgentMarkdownDocument(locator storeSpec.Locator) bool {
	return MatchesMarkdownRule(MarkdownRuleAgent, locator)
}

func IsTextMarkdownDocument(locator storeSpec.Locator) bool {
	return MatchesMarkdownRule(MarkdownRuleText, locator)
}

func IsWorkspaceManifestLocator(locator storeSpec.Locator) bool {
	return IsDocument(locator, DocumentUseWorkspaceManifest)
}

func WorkspaceManifestPatterns() []string {
	values, err := DocumentPatterns(DocumentUseWorkspaceManifest)
	if err != nil {
		panic(err)
	}
	return values
}

func IsDefaultTextMarkdownDocument(locator storeSpec.Locator) bool {
	return MatchesMarkdownRule(MarkdownRuleDefaultText, locator)
}

func IsInstructionMarkdownDocument(locator storeSpec.Locator) bool {
	return MatchesMarkdownRule(MarkdownRuleInstructionText, locator)
}

func MatchesMarkdownRule(rule string, locator storeSpec.Locator) bool {
	return configuredContractTopology.matchesMarkdownRule(
		rule,
		locator,
		map[string]struct{}{},
	)
}

func UnversionedPackageVersion() storeSpec.LogicalVersion {
	return configuredContractTopology.unversionedPackageVersion
}

func DiscoverySpecForUse(name string) (sourceModel.DiscoverySpec, error) {
	value, found := configuredContractTopology.discoveryUses[name]
	if !found {
		return sourceModel.DiscoverySpec{}, fmt.Errorf(
			"%w: contract topology has no discovery use %q",
			storeSpec.ErrNotFound,
			name,
		)
	}
	return value.Clone(), nil
}

func DiscoverySpecAtForUse(
	name string,
	root storeSpec.Locator,
) (sourceModel.DiscoverySpec, error) {
	if err := root.Validate(true); err != nil {
		return sourceModel.DiscoverySpec{}, err
	}

	value, err := DiscoverySpecForUse(name)
	if err != nil {
		return sourceModel.DiscoverySpec{}, err
	}
	if len(value.DirectoryRoots) != 1 {
		return sourceModel.DiscoverySpec{}, fmt.Errorf(
			"%w: discovery use %q cannot be rooted dynamically",
			storeSpec.ErrInvalid,
			name,
		)
	}

	value.DirectoryRoots[0].Root = root
	value = value.Normalized()
	if err := value.Validate(); err != nil {
		return sourceModel.DiscoverySpec{}, err
	}
	return value, nil
}

func DiscoverySpecForLocatorForUse(
	name string,
	locator storeSpec.Locator,
) (sourceModel.DiscoverySpec, error) {
	if err := locator.Validate(false); err != nil {
		return sourceModel.DiscoverySpec{}, err
	}

	profile, err := DiscoverySpecForUse(name)
	if err != nil {
		return sourceModel.DiscoverySpec{}, err
	}
	if len(profile.AllowedDecoderIDs) != 1 {
		return sourceModel.DiscoverySpec{}, fmt.Errorf(
			"%w: discovery use %q must define exactly one decoder",
			storeSpec.ErrInvalid,
			name,
		)
	}

	value := sourceModel.DiscoverySpec{
		ExplicitLocators: []storeSpec.Locator{locator},
		DecoderHints: []sourceModel.DecoderHint{{
			Locator:    locator,
			Recursive:  false,
			DecoderIDs: append([]storeSpec.DecoderID(nil), profile.AllowedDecoderIDs...),
		}},
		AllowedDecoderIDs: append([]storeSpec.DecoderID(nil), profile.AllowedDecoderIDs...),
		Authoritative:     profile.Authoritative,
	}
	value = value.Normalized()
	if err := value.Validate(); err != nil {
		return sourceModel.DiscoverySpec{}, err
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
	locator storeSpec.Locator,
	use string,
	format documentFormat,
) bool {
	value, found := t.documentUses[use]
	if !found {
		return false
	}

	name := strings.ToLower(path.Base(string(locator)))
	for _, alias := range value.aliases {
		if format != "" && alias.format != format {
			continue
		}
		if alias.matches(name) {
			return true
		}
	}
	return false
}

func (a documentAlias) matches(name string) bool {
	if a.locator != "" {
		return strings.EqualFold(name, string(a.locator))
	}
	matched, err := storeSpec.MatchPathPattern(a.pattern, name)
	return err == nil && matched
}

func (a documentAlias) identity() string {
	if a.locator != "" {
		return "locator:" + strings.ToLower(string(a.locator))
	}
	return "pattern:" + strings.ToLower(a.pattern)
}

func documentPatterns(aliases []documentAlias) []string {
	seen := make(map[string]struct{})
	output := make([]string, 0, len(aliases)*2)
	appendPattern := func(value string) {
		if _, duplicate := seen[value]; duplicate {
			return
		}
		seen[value] = struct{}{}
		output = append(output, value)
	}
	for _, alias := range aliases {
		if alias.locator != "" {
			appendPattern(string(alias.locator))
			appendPattern("**/" + string(alias.locator))
			continue
		}
		appendPattern(alias.pattern)
		appendPattern("**/" + alias.pattern)
	}
	return output
}

func (t contractTopology) matchesMarkdownRule(
	name string,
	locator storeSpec.Locator,
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
		storeSpec.MaxDefinitionBytes,
	)
	if err != nil {
		return contractTopology{}, err
	}

	var wire contractTopologyWire
	if err := jsonutil.DecodeCanonicalObjectBytesInto(
		canonical,
		&wire,
		storeSpec.MaxDefinitionBytes,
	); err != nil {
		return contractTopology{}, err
	}
	if wire.PackageVersions.Unversioned == "" {
		return contractTopology{}, fmt.Errorf(
			"%w: contract topology requires an unversioned package version",
			storeSpec.ErrInvalid,
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

	workspaceBootstrap, workspaceSelectorPatterns, err := parseWorkspaceDiscovery(
		wire.WorkspaceDiscovery,
		discoveryUses,
		documentUses,
	)
	if err != nil {
		return contractTopology{}, err
	}

	value := contractTopology{
		documentSets:              documentSets,
		documentUses:              documentUses,
		markdownRules:             markdownRules,
		discoveryProfiles:         discoveryProfiles,
		discoveryUses:             discoveryUses,
		workspaceBootstrap:        workspaceBootstrap,
		workspaceSelectorPatterns: workspaceSelectorPatterns,
		unversionedPackageVersion: wire.PackageVersions.Unversioned,
	}
	if err := validateRequiredContractBindings(value); err != nil {
		return contractTopology{}, err
	}
	return value, nil
}

func parseWorkspaceDiscovery(
	value workspaceDiscoveryWire,
	discoveryUses map[string]sourceModel.DiscoverySpec,
	documentUses map[string]documentUse,
) (sourceModel.DiscoverySpec, map[string][]string, error) {
	if err := storeSpec.ValidateIdentifier(
		"Workspace bootstrap discovery use",
		value.BootstrapDiscoveryUse,
		storeSpec.MaxKindBytes,
	); err != nil {
		return sourceModel.DiscoverySpec{}, nil, err
	}

	bootstrap, found := discoveryUses[value.BootstrapDiscoveryUse]
	if !found {
		return sourceModel.DiscoverySpec{}, nil, fmt.Errorf(
			"%w: Workspace bootstrap discovery use %q is unknown",
			storeSpec.ErrInvalid,
			value.BootstrapDiscoveryUse,
		)
	}
	if len(value.SelectorTypes) == 0 {
		return sourceModel.DiscoverySpec{}, nil, fmt.Errorf(
			"%w: Workspace selector discovery declarations are required",
			storeSpec.ErrInvalid,
		)
	}

	output := make(map[string][]string, len(value.SelectorTypes))
	for declarationType, selector := range value.SelectorTypes {
		if err := storeSpec.ValidateRequiredText(
			"Workspace selector declaration type",
			declarationType,
			storeSpec.MaxKindBytes,
		); err != nil {
			return sourceModel.DiscoverySpec{}, nil, err
		}

		patterns, err := patternsForWorkspaceDocumentUses(
			declarationType,
			selector.DocumentUses,
			documentUses,
		)
		if err != nil {
			return sourceModel.DiscoverySpec{}, nil, err
		}
		output[declarationType] = patterns
	}
	return bootstrap.Clone(), output, nil
}

func patternsForWorkspaceDocumentUses(
	declarationType string,
	uses []string,
	documentUses map[string]documentUse,
) ([]string, error) {
	if len(uses) == 0 {
		return nil, fmt.Errorf(
			"%w: Workspace selector type %q has no document uses",
			storeSpec.ErrInvalid,
			declarationType,
		)
	}

	seenUses := make(map[string]struct{}, len(uses))
	seenPatterns := make(map[string]struct{})
	output := make([]string, 0)
	for _, useName := range uses {
		if _, duplicate := seenUses[useName]; duplicate {
			return nil, fmt.Errorf(
				"%w: Workspace selector type %q repeats document use %q",
				storeSpec.ErrInvalid,
				declarationType,
				useName,
			)
		}
		seenUses[useName] = struct{}{}

		use, found := documentUses[useName]
		if !found {
			return nil, fmt.Errorf(
				"%w: Workspace selector type %q references unknown document use %q",
				storeSpec.ErrInvalid,
				declarationType,
				useName,
			)
		}
		for _, pattern := range documentPatterns(use.aliases) {
			if _, duplicate := seenPatterns[pattern]; duplicate {
				continue
			}
			seenPatterns[pattern] = struct{}{}
			output = append(output, pattern)
		}
	}
	return output, nil
}

func parseDocumentSets(
	values map[string][]documentAliasWire,
) (map[string][]documentAlias, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf(
			"%w: contract topology has no document sets",
			storeSpec.ErrInvalid,
		)
	}

	output := make(map[string][]documentAlias, len(values))
	for name, aliases := range values {
		if err := storeSpec.ValidateIdentifier(
			"document set name",
			name,
			storeSpec.MaxKindBytes,
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
			storeSpec.ErrInvalid,
			label,
		)
	}

	seen := make(map[string]struct{}, len(values))
	output := make([]documentAlias, 0, len(values))
	for index, value := range values {
		if (value.Locator == "") == (value.Pattern == "") {
			return nil, fmt.Errorf(
				"%w: %s[%d] must declare exactly one of locator or pattern",
				storeSpec.ErrInvalid,
				label,
				index,
			)
		}
		if err := value.Format.validate(); err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", label, index, err)
		}
		if value.DecoderID != "" {
			if err := value.DecoderID.Validate(); err != nil {
				return nil, fmt.Errorf("%s[%d] decoder: %w", label, index, err)
			}
		}

		alias := documentAlias{
			format:    value.Format,
			decoderID: value.DecoderID,
		}
		if value.Locator != "" {
			alias.locator = storeSpec.Locator(value.Locator)
			if err := alias.locator.ValidatePortable(false); err != nil {
				return nil, fmt.Errorf("%s[%d]: %w", label, index, err)
			}
			if path.Base(string(alias.locator)) != string(alias.locator) {
				return nil, fmt.Errorf(
					"%w: %s[%d] must be a package-root filename",
					storeSpec.ErrInvalid,
					label,
					index,
				)
			}
		} else {
			alias.pattern = strings.ToLower(value.Pattern)
			if strings.Contains(alias.pattern, "/") {
				return nil, fmt.Errorf("%w: %s[%d] pattern must match one filename", storeSpec.ErrInvalid, label, index)
			}
			if err := storeSpec.ValidatePathPattern(alias.pattern); err != nil {
				return nil, fmt.Errorf("%s[%d]: %w", label, index, err)
			}
		}
		key := alias.identity()
		if _, duplicate := seen[key]; duplicate {
			return nil, fmt.Errorf(
				"%w: %s repeats document alias %q",
				storeSpec.ErrInvalid,
				label,
				key,
			)
		}
		seen[key] = struct{}{}
		output = append(output, alias)
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
			storeSpec.ErrInvalid,
		)
	}

	output := make(map[string]documentUse, len(values))
	for name, value := range values {
		if err := storeSpec.ValidateIdentifier(
			"document use name",
			name,
			storeSpec.MaxKindBytes,
		); err != nil {
			return nil, err
		}
		if len(value.DocumentSets) == 0 {
			return nil, fmt.Errorf(
				"%w: document use %q has no document sets",
				storeSpec.ErrInvalid,
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
					storeSpec.ErrInvalid,
					name,
					setName,
				)
			}
			seenSets[setName] = struct{}{}

			set, found := documentSets[setName]
			if !found {
				return nil, fmt.Errorf(
					"%w: document use %q references unknown set %q",
					storeSpec.ErrInvalid,
					name,
					setName,
				)
			}
			for _, alias := range set {
				key := alias.identity()
				if previous, duplicate := seenAliases[key]; duplicate {
					if previous.format != alias.format {
						return nil, fmt.Errorf(
							"%w: document use %q has conflicting formats for %q",
							storeSpec.ErrInvalid,
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

		var defaultLocator storeSpec.Locator
		if value.DefaultLocator != "" {
			for _, alias := range aliases {
				if alias.locator == "" {
					continue
				}
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
					storeSpec.ErrInvalid,
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
			storeSpec.ErrInvalid,
		)
	}

	output := make(map[string]markdownRule, len(values))
	for name, value := range values {
		if err := storeSpec.ValidateIdentifier(
			"markdown rule name",
			name,
			storeSpec.MaxKindBytes,
		); err != nil {
			return nil, err
		}

		validateUse := func(use string) error {
			if _, found := documentUses[use]; !found {
				return fmt.Errorf(
					"%w: markdown rule %q references document use %q",
					storeSpec.ErrInvalid,
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
				storeSpec.ErrInvalid,
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
					storeSpec.ErrInvalid,
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
		if err := storeSpec.ValidateRequiredText(
			label,
			value,
			storeSpec.MaxLogicalNameBytes,
		); err != nil {
			return nil, err
		}
		if strings.Contains(value, "/") {
			return nil, fmt.Errorf(
				"%w: %s %q cannot contain a path separator",
				storeSpec.ErrInvalid,
				label,
				value,
			)
		}
		if _, duplicate := seen[value]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate %s %q",
				storeSpec.ErrInvalid,
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
				storeSpec.ErrInvalid,
				name,
			)
		case 2:
			return nil
		}

		rule, found := rules[name]
		if !found {
			return fmt.Errorf(
				"%w: markdown rule references unknown rule %q",
				storeSpec.ErrInvalid,
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
) (map[string]sourceModel.DiscoverySpec, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf(
			"%w: contract topology has no discovery profiles",
			storeSpec.ErrInvalid,
		)
	}

	output := make(map[string]sourceModel.DiscoverySpec, len(values))
	for name, value := range values {
		if err := storeSpec.ValidateIdentifier(
			"discovery profile name",
			name,
			storeSpec.MaxKindBytes,
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
) (sourceModel.DiscoverySpec, error) {
	if err := value.Root.Validate(true); err != nil {
		return sourceModel.DiscoverySpec{}, err
	}

	var basePatterns []string
	var err error
	if len(value.IncludePatterns) != 0 {
		basePatterns, err = parseDiscoveryPatterns(
			name+" discovery include patterns",
			value.IncludePatterns,
		)
		if err != nil {
			return sourceModel.DiscoverySpec{}, err
		}
	}
	if err := storeSpec.ValidatePathPatterns(
		name+" discovery exclude patterns",
		value.ExcludePatterns,
	); err != nil {
		return sourceModel.DiscoverySpec{}, err
	}

	seenSets := make(map[string]struct{}, len(value.DocumentSets))
	selectedAliases := make([]documentAlias, 0)
	for _, setName := range value.DocumentSets {
		if _, duplicate := seenSets[setName]; duplicate {
			return sourceModel.DiscoverySpec{}, fmt.Errorf(
				"%w: discovery profile %q repeats document set %q",
				storeSpec.ErrInvalid,
				name,
				setName,
			)
		}
		seenSets[setName] = struct{}{}

		aliases, found := documentSets[setName]
		if !found {
			return sourceModel.DiscoverySpec{}, fmt.Errorf(
				"%w: discovery profile %q references unknown document set %q",
				storeSpec.ErrInvalid,
				name,
				setName,
			)
		}
		selectedAliases = append(selectedAliases, aliases...)
	}

	includePatterns := discoveryPatterns(basePatterns, selectedAliases)
	if len(includePatterns) == 0 {
		return sourceModel.DiscoverySpec{}, fmt.Errorf(
			"%w: discovery profile %q has no include patterns",
			storeSpec.ErrInvalid,
			name,
		)
	}

	hints := make([]sourceModel.DecoderHint, 0, len(value.DecoderHints))
	for _, hint := range value.DecoderHints {
		hints = append(hints, sourceModel.DecoderHint{
			Locator:    hint.Locator,
			Recursive:  hint.Recursive,
			DecoderIDs: append([]storeSpec.DecoderID(nil), hint.DecoderIDs...),
		})
	}

	output := sourceModel.DiscoverySpec{
		DirectoryRoots: []sourceModel.DirectoryRoot{{
			Root:            value.Root,
			Recursive:       value.Recursive,
			IncludePatterns: includePatterns,
			ExcludePatterns: append([]string(nil), value.ExcludePatterns...),
		}},
		DecoderHints:      hints,
		AllowedDecoderIDs: append([]storeSpec.DecoderID(nil), value.AllowedDecoderIDs...),
		Authoritative:     value.Authoritative,
	}
	output = output.Normalized()
	if err := output.Validate(); err != nil {
		return sourceModel.DiscoverySpec{}, err
	}
	return output, nil
}

func parseDiscoveryPatterns(
	label string,
	values []string,
) ([]string, error) {
	if err := storeSpec.ValidatePathPatterns(label, values); err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(values))
	output := make([]string, 0, len(values))
	for _, value := range values {
		if _, duplicate := seen[value]; duplicate {
			return nil, fmt.Errorf(
				"%w: %s repeats pattern %q",
				storeSpec.ErrInvalid,
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
	profiles map[string]sourceModel.DiscoverySpec,
) (map[string]sourceModel.DiscoverySpec, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf(
			"%w: contract topology has no discovery uses",
			storeSpec.ErrInvalid,
		)
	}

	output := make(map[string]sourceModel.DiscoverySpec, len(values))
	for name, value := range values {
		if err := storeSpec.ValidateIdentifier(
			"discovery use name",
			name,
			storeSpec.MaxKindBytes,
		); err != nil {
			return nil, err
		}
		if len(value.Profiles) == 0 {
			return nil, fmt.Errorf(
				"%w: discovery use %q has no profiles",
				storeSpec.ErrInvalid,
				name,
			)
		}

		seen := make(map[string]struct{}, len(value.Profiles))
		selected := make([]sourceModel.DiscoverySpec, 0, len(value.Profiles))
		for _, profileName := range value.Profiles {
			if _, duplicate := seen[profileName]; duplicate {
				return nil, fmt.Errorf(
					"%w: discovery use %q repeats profile %q",
					storeSpec.ErrInvalid,
					name,
					profileName,
				)
			}
			seen[profileName] = struct{}{}
			profile, found := profiles[profileName]
			if !found {
				return nil, fmt.Errorf(
					"%w: discovery use %q references unknown profile %q",
					storeSpec.ErrInvalid,
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
	profiles []sourceModel.DiscoverySpec,
) (sourceModel.DiscoverySpec, error) {
	var (
		output        sourceModel.DiscoverySpec
		authoritative *bool
	)
	for _, profile := range profiles {
		value := profile.Clone()
		if authoritative == nil {
			next := value.Authoritative
			authoritative = &next
		} else if *authoritative != value.Authoritative {
			return sourceModel.DiscoverySpec{}, fmt.Errorf(
				"%w: discovery use %q mixes authoritative and non-authoritative profiles",
				storeSpec.ErrInvalid,
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
		return sourceModel.DiscoverySpec{}, err
	}
	return output, nil
}

func validateRequiredContractBindings(value contractTopology) error {
	for _, use := range []string{
		DocumentUsePlugin,
		DocumentUseCanonicalYAML,
		DocumentUseCanonicalJSON,
		DocumentUseMCPConfig,
		DocumentUseSkillPackage,
		DocumentUseModelProviderPackage,
		DocumentUseModelPackage,
		DocumentUseToolPackage,
		DocumentUseToolPlugin,
		DocumentUseManagedPlugin,
		DocumentUseManagedModelProvider,
		DocumentUseManagedModel,
		DocumentUseAgentManagedPlugin,
		DocumentUseManagedAgent,
		DocumentUseManagedMCP,
		DocumentUseManagedMCPPolicy,
		DocumentUseAgentMarkdown,
		DocumentUseWorkspaceMarkdown,
		DocumentUseWorkspaceInstructions,
		DocumentUseWorkspaceManifest,
	} {
		if _, found := value.documentUses[use]; !found {
			return fmt.Errorf(
				"%w: required document use %q is missing",
				storeSpec.ErrInvalid,
				use,
			)
		}
	}
	for _, use := range []string{
		DocumentUseSkillPackage,
		DocumentUseModelProviderPackage,
		DocumentUseModelPackage,
		DocumentUseToolPackage,
		DocumentUseToolPlugin,
		DocumentUseManagedPlugin,
		DocumentUseManagedModelProvider,
		DocumentUseManagedModel,
		DocumentUseAgentManagedPlugin,
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

	uses := make([]string, 0, len(value.documentUses))
	for use := range value.documentUses {
		uses = append(uses, use)
	}
	slices.Sort(uses)

	for _, use := range uses {
		documentUse := value.documentUses[use]
		if documentUse.defaultLocator == "" {
			continue
		}
		if _, err := value.defaultDocumentDecoderID(use); err != nil {
			return fmt.Errorf(
				"document use %q default decoder: %w",
				use,
				err,
			)
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
				storeSpec.ErrInvalid,
				name,
			)
		}
	}
	for _, name := range []string{
		DiscoveryUseSkill,
		DiscoveryUseMCP,
		DiscoveryUseWorkspace,
	} {
		if _, found := value.discoveryUses[name]; !found {
			return fmt.Errorf(
				"%w: required discovery use %q is missing",
				storeSpec.ErrInvalid,
				name,
			)
		}
	}

	for _, declarationType := range []string{
		"skill",
		"agent",
		"plugin",
		"mcp",
		"mcp.policy",
		"model",
		"workflow",
	} {
		if _, found := value.workspaceSelectorPatterns[declarationType]; !found {
			return fmt.Errorf(
				"%w: workspace selector discovery for type %q is required",
				storeSpec.ErrInvalid,
				declarationType,
			)
		}
	}

	if err := value.workspaceBootstrap.Validate(); err != nil {
		return fmt.Errorf(
			"workspace bootstrap discovery: %w",
			err,
		)
	}
	return nil
}

func (t contractTopology) defaultDocumentFile(
	use string,
) (storeSpec.Locator, error) {
	value, found := t.documentUses[use]
	if !found || value.defaultLocator == "" {
		return "", fmt.Errorf(
			"%w: document use %q has no default locator",
			storeSpec.ErrInvalid,
			use,
		)
	}
	return value.defaultLocator, nil
}

func (t contractTopology) defaultDocumentDecoderID(
	use string,
) (storeSpec.DecoderID, error) {
	value, found := t.documentUses[use]
	if !found {
		return "", fmt.Errorf(
			"%w: document use %q is unknown",
			storeSpec.ErrInvalid,
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
		storeSpec.ErrInvalid,
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
			storeSpec.ErrInvalid,
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
	for _, aliases := range groups {
		for _, pattern := range documentPatterns(aliases) {
			appendPattern(pattern)
		}
	}
	return output
}
