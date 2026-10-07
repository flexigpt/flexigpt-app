// Package llmsupport resolves FlexiGPT application choices into the narrow
// support values consumed by LLM artifact families.
//
// It is intentionally application setup. No package below
// internal/llmartifactory-go imports this package or artifactsetup/topology.
package llmsupport

import (
	"encoding/json"
	"fmt"
	"path"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/iofs"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/managedfs"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	agentAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
	mcpAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp"
	modelAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	pluginDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/domain"
	skillAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill"
	toolAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool"
	workspaceAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace"
)

const (
	agentManagedSourceStorageKey = "user-agents"
	skillManagedSourceStorageKey = "user-skills"
	mcpManagedSourceStorageKey   = "user-mcps"
	modelManagedSourceStorageKey = "model-artifacts"

	agentBaselinePluginName = "agent-baseline"
	skillBaselinePluginName = "skill-baseline"
	mcpBaselinePluginName   = "mcp-baseline"
)

func document(
	use string,
) (support.Document, error) {
	locator, err := topology.DefaultDocumentFile(use)
	if err != nil {
		return support.Document{}, err
	}
	decoderID, err := topology.DefaultDocumentDecoderID(use)
	if err != nil {
		return support.Document{}, err
	}
	value := support.Document{
		Locator:   locator,
		DecoderID: decoderID,
	}
	if err := value.Validate(); err != nil {
		return support.Document{}, err
	}
	return value, nil
}

func documents(
	use string,
) (support.Documents, error) {
	defaultDocument, err := document(use)
	if err != nil {
		return support.Documents{}, err
	}

	files, err := topology.DocumentFiles(use)
	if err != nil {
		return support.Documents{}, err
	}
	patterns, err := topology.DocumentPatterns(use)
	if err != nil {
		return support.Documents{}, err
	}

	value := support.Documents{
		Default:  defaultDocument,
		Files:    files,
		Patterns: patterns,
	}
	if err := value.Validate(); err != nil {
		return support.Documents{}, err
	}

	return value, nil
}

func layout(
	kind managedpackageModel.PackageKind,
	use string,
	fixedVersion bool,
) (support.PackageLayout, error) {
	documentValue, err := document(use)
	if err != nil {
		return support.PackageLayout{}, err
	}
	value := support.PackageLayout{
		Kind:           kind,
		DefaultVersion: topology.UnversionedPackageVersion(),
		FixedVersion:   fixedVersion,
		Document:       documentValue,
	}
	if err := value.Validate(); err != nil {
		return support.PackageLayout{}, err
	}
	return value, nil
}

func managedSource(
	storageKey string,
	displayName string,
) support.SourceProfile {
	return support.SourceProfile{
		StorageKey:  spec.StorageKey(storageKey),
		Kind:        managedfs.Kind,
		DisplayName: displayName,
		Config:      json.RawMessage(`{}`),
	}
}

func profile(
	name string,
	source support.SourceProfile,
	packageLayout support.PackageLayout,
	baselineName string,
	baselineDisplayName string,
	baselineDescription string,
	policy pluginDomain.MembershipPolicy,
) pluginAPI.Profile {
	return pluginAPI.Profile{
		Name:                name,
		BuiltinRoot:         topology.BuiltinRootID(),
		Source:              &source,
		Package:             packageLayout,
		BaselineName:        spec.LogicalName(baselineName),
		BaselineDisplayName: baselineDisplayName,
		BaselineDescription: baselineDescription,
		MembershipPolicy:    policy,
	}
}

func Agent() (agentAPI.Support, error) {
	managedPackage, err := layout(
		managedpackageModel.PackageKind("agent"),
		topology.DocumentUseManagedAgent,
		true,
	)
	if err != nil {
		return agentAPI.Support{}, err
	}
	pluginPackage, err := layout(
		managedpackageModel.PackageKind("plugin"),
		topology.DocumentUseAgentManagedPlugin,
		true,
	)
	if err != nil {
		return agentAPI.Support{}, err
	}

	documentSet, err := documents(topology.DocumentUseManagedAgent)
	if err != nil {
		return agentAPI.Support{}, err
	}

	source := managedSource(
		agentManagedSourceStorageKey,
		"User-managed Agents",
	)
	value := agentAPI.Support{
		BuiltinRoot:    topology.BuiltinRootID(),
		ManagedPackage: managedPackage,
		Documents:      documentSet,
		PluginProfile: profile(
			"agent",
			source,
			pluginPackage,
			agentBaselinePluginName,
			"Agent Baseline",
			"Application-provisioned editable Agent Plugin.",
			pluginDomain.MembershipPolicy{
				Mode: pluginDomain.MembershipModeSingleType,
				AllowedTypes: []declaration.Type{
					declaration.TypeAgent,
				},
				AllowedForms: []declaration.MemberForm{
					declaration.MemberNamed,
				},
			},
		),
		ImportFormats: importFormats(topology.DocumentUseManagedAgent),
	}
	if err := value.Validate(); err != nil {
		return agentAPI.Support{}, err
	}
	return value, nil
}

func Skill() (skillAPI.Support, error) {
	packageLayout, err := layout(
		managedpackageModel.PackageKind("skill"),
		topology.DocumentUseSkillPackage,
		false,
	)
	if err != nil {
		return skillAPI.Support{}, err
	}
	pluginPackage, err := layout(
		managedpackageModel.PackageKind("plugin"),
		topology.DocumentUseManagedPlugin,
		true,
	)
	if err != nil {
		return skillAPI.Support{}, err
	}
	discovery, err := topology.DiscoverySpecForUse(
		topology.DiscoveryUseSkill,
	)
	if err != nil {
		return skillAPI.Support{}, err
	}
	documentSet, err := documents(topology.DocumentUseSkillPackage)
	if err != nil {
		return skillAPI.Support{}, err
	}

	source := managedSource(
		skillManagedSourceStorageKey,
		"User-managed Skills",
	)
	value := skillAPI.Support{
		BuiltinRoot: topology.BuiltinRootID(),
		Documents:   documentSet,
		Package:     packageLayout,
		Discovery:   discovery,
		PluginProfile: profile(
			"skill",
			source,
			pluginPackage,
			skillBaselinePluginName,
			"Skill Baseline",
			"Application-provisioned editable Skill Plugin.",
			pluginDomain.MembershipPolicy{
				Mode: pluginDomain.MembershipModeSingleType,
				AllowedTypes: []declaration.Type{
					declaration.TypeSkill,
				},
				AllowedForms: []declaration.MemberForm{
					declaration.MemberNamed,
				},
			},
		),
	}
	if err := value.Validate(); err != nil {
		return skillAPI.Support{}, err
	}
	return value, nil
}

func MCP() (mcpAPI.Support, error) {
	serverPackage, err := layout(
		managedpackageModel.PackageKind("mcp"),
		topology.DocumentUseManagedMCP,
		false,
	)
	if err != nil {
		return mcpAPI.Support{}, err
	}
	policyPackage, err := layout(
		managedpackageModel.PackageKind("mcp-policy"),
		topology.DocumentUseManagedMCPPolicy,
		true,
	)
	if err != nil {
		return mcpAPI.Support{}, err
	}
	pluginPackage, err := layout(
		managedpackageModel.PackageKind("plugin"),
		topology.DocumentUseManagedPlugin,
		true,
	)
	if err != nil {
		return mcpAPI.Support{}, err
	}
	builtinPluginDocument, err := document(topology.DocumentUsePlugin)
	if err != nil {
		return mcpAPI.Support{}, err
	}
	builtinSource, err := topology.BuiltinSource(
		topology.BuiltinSourceRolePackages,
	)
	if err != nil {
		return mcpAPI.Support{}, err
	}

	source := managedSource(
		mcpManagedSourceStorageKey,
		"User-managed MCP artifacts",
	)
	value := mcpAPI.Support{
		BuiltinRoot:           topology.BuiltinRootID(),
		BuiltinPackageSource:  builtinSource.ID,
		BuiltinPluginDocument: builtinPluginDocument,
		ServerPackage:         serverPackage,
		PolicyPackage:         policyPackage,
		PluginProfile: profile(
			"mcp",
			source,
			pluginPackage,
			mcpBaselinePluginName,
			"MCP Baseline",
			"Application-provisioned editable MCP Plugin.",
			pluginDomain.MembershipPolicy{
				Mode: pluginDomain.MembershipModeMixedType,
				AllowedTypes: []declaration.Type{
					declaration.TypeMCP,
					declaration.TypeMCPPolicy,
				},
				AllowedForms: []declaration.MemberForm{
					declaration.MemberNamed,
				},
			},
		),
	}
	if err := value.Validate(); err != nil {
		return mcpAPI.Support{}, err
	}
	return value, nil
}

func Model() (modelAPI.Support, error) {
	providerPackage, err := layout(
		managedpackageModel.PackageKind("model-provider"),
		topology.DocumentUseManagedModelProvider,
		true,
	)
	if err != nil {
		return modelAPI.Support{}, err
	}
	modelPackage, err := layout(
		managedpackageModel.PackageKind("model"),
		topology.DocumentUseManagedModel,
		true,
	)
	if err != nil {
		return modelAPI.Support{}, err
	}

	builtinSource, err := topology.BuiltinSource(
		topology.BuiltinSourceRolePackages,
	)
	if err != nil {
		return modelAPI.Support{}, err
	}

	value := modelAPI.Support{
		BuiltinRoot:          topology.BuiltinRootID(),
		BuiltinPackageSource: builtinSource.ID,
		ManagedSource: managedSource(
			modelManagedSourceStorageKey,
			"Managed Model Artifacts",
		),
		ProviderPackage: providerPackage,
		ModelPackage:    modelPackage,
	}
	if err := value.Validate(); err != nil {
		return modelAPI.Support{}, err
	}
	return value, nil
}

func Tool() (toolAPI.Support, error) {
	toolPackage, err := layout(
		managedpackageModel.PackageKind("tool"),
		topology.DocumentUseToolPackage,
		false,
	)
	if err != nil {
		return toolAPI.Support{}, err
	}
	pluginPackage, err := layout(
		managedpackageModel.PackageKind("tool-plugin"),
		topology.DocumentUseToolPlugin,
		true,
	)
	if err != nil {
		return toolAPI.Support{}, err
	}

	toolDocuments, err := documents(topology.DocumentUseToolPackage)
	if err != nil {
		return toolAPI.Support{}, err
	}
	pluginDocuments, err := documents(topology.DocumentUseToolPlugin)
	if err != nil {
		return toolAPI.Support{}, err
	}

	builtinSource, err := topology.BuiltinSource(
		topology.BuiltinSourceRolePackages,
	)
	if err != nil {
		return toolAPI.Support{}, err
	}
	readOnlySource := support.SourceProfile{
		StorageKey:  builtinSource.StorageKey,
		Kind:        builtinSource.Kind,
		DisplayName: builtinSource.DisplayName,
		Config:      append(json.RawMessage(nil), builtinSource.Config...),
	}
	if err := readOnlySource.Validate(); err != nil {
		return toolAPI.Support{}, err
	}

	value := toolAPI.Support{
		ToolPackage:     toolPackage,
		Documents:       toolDocuments,
		PluginDocuments: pluginDocuments,
		PluginProfile: pluginAPI.Profile{
			Name:           "tool",
			BuiltinRoot:    topology.BuiltinRootID(),
			ReadOnly:       true,
			ReadOnlySource: &readOnlySource,
			Package:        pluginPackage,
			MembershipPolicy: pluginDomain.MembershipPolicy{
				Mode: pluginDomain.MembershipModeSingleType,
				AllowedTypes: []declaration.Type{
					declaration.TypeTool,
				},
				AllowedForms: []declaration.MemberForm{
					declaration.MemberNamed,
				},
			},
		},
	}
	if err := value.Validate(); err != nil {
		return toolAPI.Support{}, err
	}
	return value, nil
}

func Workspace() (workspaceAPI.Support, error) {
	discovery, err := topology.DiscoverySpecForUse(
		topology.DiscoveryUseWorkspace,
	)
	if err != nil {
		return workspaceAPI.Support{}, err
	}
	selectorPatterns, err := topology.DiscoveryIncludePatternsForUse(
		topology.DiscoveryUseSelector,
	)
	if err != nil {
		return workspaceAPI.Support{}, err
	}
	manifestPatterns := topology.WorkspaceManifestPatterns()
	skillDocuments, err := documents(topology.DocumentUseSkillPackage)
	if err != nil {
		return workspaceAPI.Support{}, err
	}

	value := workspaceAPI.Support{
		DirectorySource: support.SourceProfile{
			StorageKey:  spec.StorageKey("workspace-directory"),
			Kind:        fsdir.Kind,
			DisplayName: "Workspace directory source",
		},
		PolicySource: support.SourceProfile{
			StorageKey:  spec.StorageKey("workspace-base-policy"),
			Kind:        iofs.Kind,
			DisplayName: "Workspace base policy source",
		},
		RootStorageKeyPrefix:    spec.StorageKey("workspace-directory-root-"),
		DirectoryDiscovery:      discovery,
		ManifestPatterns:        append([]string(nil), manifestPatterns...),
		SelectorIncludePatterns: append([]string(nil), selectorPatterns...),
		SkillDocuments:          skillDocuments,
	}
	if err := value.Validate(); err != nil {
		return workspaceAPI.Support{}, err
	}
	return value, nil
}

func Candidates(
	decoderID spec.DecoderID,
) (support.Candidates, error) {
	patterns, err := topology.DocumentPatternsForDecoder(decoderID)
	if err != nil {
		return support.Candidates{}, err
	}
	value := support.Candidates{Patterns: patterns}
	if err := value.Validate(); err != nil {
		return support.Candidates{}, err
	}
	return value, nil
}

func importFormats(
	use string,
) map[string]agentAPI.ImportFormat {
	files, err := topology.DocumentFiles(use)
	if err != nil {
		panic(err)
	}

	output := make(map[string]agentAPI.ImportFormat)
	for _, file := range files {
		switch path.Ext(string(file)) {
		case ".json":
			output[".json"] = agentAPI.ImportFormatJSON
		case ".yaml", ".yml":
			output[path.Ext(string(file))] = agentAPI.ImportFormatYAML
		}
	}

	keys := make([]string, 0, len(output))
	for extension := range output {
		keys = append(keys, extension)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		panic(fmt.Sprintf("document use %q has no portable import format", use))
	}
	return output
}
