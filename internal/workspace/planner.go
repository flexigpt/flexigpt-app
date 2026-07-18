package workspace

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	artifactstoreSpec "github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

type defaultDiscoveryPlanner struct{}

func (defaultDiscoveryPlanner) BuildBootstrapPlan(
	_ context.Context,
	input DiscoveryInput,
) (artifactstoreSpec.ScanPlan, error) {
	return buildWorkspacePlan(input, false)
}

func (defaultDiscoveryPlanner) BuildExpandedPlan(
	_ context.Context,
	input DiscoveryInput,
) (artifactstoreSpec.ScanPlan, error) {
	return buildWorkspacePlan(input, true)
}

func buildWorkspacePlan(input DiscoveryInput, expanded bool) (artifactstoreSpec.ScanPlan, error) {
	preferences, err := mergeDiscoveryPreferences(
		input.Workspace.Data.DiscoveryPreferences,
		input.DefinitionPreferences,
	)
	if err != nil {
		return artifactstoreSpec.ScanPlan{}, err
	}
	plans := make([]artifactstoreSpec.SourceScanPlan, 0, len(input.Workspace.Attachments))
	for _, attachment := range input.Workspace.Attachments {
		if !attachment.Enabled {
			continue
		}
		sourcePlan := artifactstoreSpec.SourceScanPlan{
			SourceID:            attachment.SourceID,
			AllowedFrontendIDs:  append([]artifactstoreSpec.FrontendID(nil), input.FrontendIDs...),
			MaxFileBytes:        artifactstoreSpec.MaxDefinitionJSONBytes,
			MaxTotalBytes:       artifactstoreSpec.MaxScanTotalBytes,
			MaxCandidates:       artifactstoreSpec.DefaultMaxScanCandidates,
			MaxTraversalEntries: artifactstoreSpec.DefaultMaxScanEntries,
			MaxTraversalDepth:   artifactstoreSpec.DefaultMaxTraversalDepth,
		}
		switch attachment.Role {
		case RolePrimary:
			sourcePlan.ExplicitLocators = bootstrapLocators(preferences.IncludeReadme)
			sourcePlan.DirectoryRoots = bootstrapRoots()
			if expanded {
				sourcePlan.ExplicitLocators = append(
					sourcePlan.ExplicitLocators,
					preferences.AdditionalLocators...,
				)
				for _, root := range preferences.AdditionalRoots {
					sourcePlan.DirectoryRoots = append(
						sourcePlan.DirectoryRoots,
						artifactstoreSpec.DirectoryScanRoot{
							Root:            root.Root,
							Recursive:       root.Recursive,
							IncludePatterns: append([]string(nil), root.IncludePatterns...),
						},
					)
				}
			}
			sourcePlan.Authoritative = true
		default:
			recursive, authoritative, err := attachmentDiscoveryBehavior(
				attachment,
				input.Workspace.Data.AttachedPackagePreferences.DiscoverRecursively,
			)
			if err != nil {
				return artifactstoreSpec.ScanPlan{}, err
			}
			sourcePlan.DirectoryRoots = []artifactstoreSpec.DirectoryScanRoot{{
				Root:            ".",
				Recursive:       recursive,
				IncludePatterns: workspaceCandidatePatterns(),
			}}
			sourcePlan.Authoritative = authoritative
		}
		normalizeSourcePlan(&sourcePlan)
		plans = append(plans, sourcePlan)
	}
	sort.Slice(plans, func(left, right int) bool {
		return plans[left].SourceID < plans[right].SourceID
	})
	return artifactstoreSpec.ScanPlan{SourcePlans: plans}, nil
}

func attachmentDiscoveryBehavior(
	attachment artifactstoreSpec.RootSourceAttachment,
	defaultRecursive bool,
) (recursive, authoritative bool, err error) {
	recursive = defaultRecursive
	switch attachment.Role {
	case RoleAttachedPackage, RoleBuiltIn, RoleAppLibrary, RoleOverlay:
		authoritative = true
	default:
		authoritative = false
	}
	if len(bytes.TrimSpace(attachment.Data)) == 0 {
		return recursive, authoritative, nil
	}
	var data AttachmentData
	if err := decodeStrictJSONObject(attachment.Data, &data, true); err != nil {
		return false, false, fmt.Errorf("decode attachment discovery data: %w", err)
	}
	if data.Recursive != nil {
		recursive = *data.Recursive
	}
	if data.Authoritative != nil {
		authoritative = *data.Authoritative
	}
	return recursive, authoritative, nil
}

func bootstrapLocators(includeReadme bool) []artifactstoreSpec.SourceLocator {
	out := []artifactstoreSpec.SourceLocator{
		workspaceDefinitionJSONLocator,
		workspaceDefinitionYAMLLocator,
		workspaceDefinitionYMLLocator,
		workspaceMCPDotJSONLocator,
		workspaceMCPDotsJSONLocator,
		workspaceMCPJSONLocator,
		workspaceMCPsJSONLocator,
		workspaceAgentsLocator,
	}
	if includeReadme {
		out = append(out, workspaceReadmeLocator)
	}
	return out
}

func bootstrapRoots() []artifactstoreSpec.DirectoryScanRoot {
	return []artifactstoreSpec.DirectoryScanRoot{
		{
			Root:            artifactstoreSpec.SourceLocator(strings.TrimSuffix(workspaceAgentsDirectory, "/")),
			Recursive:       true,
			IncludePatterns: structuredCandidatePatterns(),
		},
		{
			Root:            artifactstoreSpec.SourceLocator(strings.TrimSuffix(workspaceModelsDirectory, "/")),
			Recursive:       true,
			IncludePatterns: structuredCandidatePatterns(),
		},
		{
			Root:            artifactstoreSpec.SourceLocator(strings.TrimSuffix(workspaceMCPDirectory, "/")),
			Recursive:       true,
			IncludePatterns: structuredCandidatePatterns(),
		},
		{
			Root:            artifactstoreSpec.SourceLocator(strings.TrimSuffix(workspaceToolsDirectory, "/")),
			Recursive:       true,
			IncludePatterns: structuredCandidatePatterns(),
		},
		{Root: workspaceSkillsDirectory, Recursive: true, IncludePatterns: []string{"SKILL.md"}},
	}
}

func mergeDiscoveryPreferences(
	left DiscoveryPreferences,
	right DiscoveryPreferences,
) (DiscoveryPreferences, error) {
	if err := validateDiscoveryPreferences(left); err != nil {
		return DiscoveryPreferences{}, err
	}
	if err := validateDiscoveryPreferences(right); err != nil {
		return DiscoveryPreferences{}, err
	}
	out := DiscoveryPreferences{
		IncludeReadme: left.IncludeReadme || right.IncludeReadme,
	}
	locators := map[artifactstoreSpec.SourceLocator]struct{}{}
	for _, values := range [][]artifactstoreSpec.SourceLocator{
		left.AdditionalLocators,
		right.AdditionalLocators,
	} {
		for _, locator := range values {
			if _, exists := locators[locator]; exists {
				continue
			}
			locators[locator] = struct{}{}
			out.AdditionalLocators = append(out.AdditionalLocators, locator)
		}
	}
	roots := map[artifactstoreSpec.SourceLocator]DiscoveryRoot{}
	for _, values := range [][]DiscoveryRoot{left.AdditionalRoots, right.AdditionalRoots} {
		for _, root := range values {
			if current, exists := roots[root.Root]; exists {
				current.Recursive = current.Recursive || root.Recursive
				current.IncludePatterns = mergeIncludePatterns(
					current.IncludePatterns,
					root.IncludePatterns,
				)
				roots[root.Root] = current
			} else {
				root.IncludePatterns = append([]string(nil), root.IncludePatterns...)
				roots[root.Root] = root
			}
		}
	}
	for _, root := range roots {
		sort.Strings(root.IncludePatterns)
		out.AdditionalRoots = append(out.AdditionalRoots, root)
	}
	slices.Sort(out.AdditionalLocators)
	sort.Slice(out.AdditionalRoots, func(left, right int) bool {
		return out.AdditionalRoots[left].Root < out.AdditionalRoots[right].Root
	})
	if err := validateDiscoveryPreferences(out); err != nil {
		return DiscoveryPreferences{}, err
	}
	return out, nil
}

func normalizeSourcePlan(plan *artifactstoreSpec.SourceScanPlan) {
	if plan == nil {
		return
	}
	locators := map[artifactstoreSpec.SourceLocator]struct{}{}
	filteredLocators := make([]artifactstoreSpec.SourceLocator, 0, len(plan.ExplicitLocators))
	for _, locator := range plan.ExplicitLocators {
		if _, exists := locators[locator]; exists {
			continue
		}
		locators[locator] = struct{}{}
		filteredLocators = append(filteredLocators, locator)
	}
	slices.Sort(filteredLocators)
	plan.ExplicitLocators = filteredLocators

	frontends := map[artifactstoreSpec.FrontendID]struct{}{}
	filteredFrontends := make([]artifactstoreSpec.FrontendID, 0, len(plan.AllowedFrontendIDs))
	for _, frontend := range plan.AllowedFrontendIDs {
		if _, exists := frontends[frontend]; exists {
			continue
		}
		frontends[frontend] = struct{}{}
		filteredFrontends = append(filteredFrontends, frontend)
	}
	slices.Sort(filteredFrontends)
	plan.AllowedFrontendIDs = filteredFrontends

	roots := make(map[artifactstoreSpec.SourceLocator]artifactstoreSpec.DirectoryScanRoot)
	for _, root := range plan.DirectoryRoots {
		if current, exists := roots[root.Root]; exists {
			current.Recursive = current.Recursive || root.Recursive
			current.IncludePatterns = mergeIncludePatterns(
				current.IncludePatterns,
				root.IncludePatterns,
			)
			roots[root.Root] = current
		} else {
			root.IncludePatterns = append([]string(nil), root.IncludePatterns...)
			roots[root.Root] = root
		}
	}
	plan.DirectoryRoots = plan.DirectoryRoots[:0]
	for _, root := range roots {
		sort.Strings(root.IncludePatterns)
		plan.DirectoryRoots = append(plan.DirectoryRoots, root)
	}
	sort.Slice(plan.DirectoryRoots, func(left, right int) bool {
		if plan.DirectoryRoots[left].Root != plan.DirectoryRoots[right].Root {
			return plan.DirectoryRoots[left].Root < plan.DirectoryRoots[right].Root
		}
		return !plan.DirectoryRoots[left].Recursive && plan.DirectoryRoots[right].Recursive
	})
}

func structuredCandidatePatterns() []string {
	return []string{"*.json", "*.yaml", "*.yml"}
}

func workspaceCandidatePatterns() []string {
	return []string{"*.json", "*.yaml", "*.yml", "SKILL.md"}
}

func mergeIncludePatterns(left, right []string) []string {
	if len(left) == 0 || len(right) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(left)+len(right))
	out := make([]string, 0, len(left)+len(right))
	for _, values := range [][]string{left, right} {
		for _, value := range values {
			if _, exists := seen[value]; exists {
				continue
			}
			seen[value] = struct{}{}
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

var _ DiscoveryPlanner = defaultDiscoveryPlanner{}
