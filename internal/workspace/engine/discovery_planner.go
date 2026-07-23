package engine

import (
	"fmt"
	"slices"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
)

type Planner struct {
	decoderIDs []artifactstore.DecoderID
	profiles   DiscoveryProfiles
}

func NewPlanner(
	profiles DiscoveryProfiles,
	decoderIDs ...artifactstore.DecoderID,
) (*Planner, error) {
	if err := validateDiscoveryProfiles(profiles); err != nil {
		return nil, fmt.Errorf(
			"%w: %w",
			ErrInvalidWorkspace,
			err,
		)
	}
	if len(profiles.Primary.ExplicitLocators) == 0 &&
		len(profiles.Primary.DirectoryRoots) == 0 {
		return nil, fmt.Errorf(
			"%w: primary discovery profile is required",
			ErrInvalidWorkspace,
		)
	}

	seen := make(map[artifactstore.DecoderID]struct{}, len(decoderIDs))
	values := make([]artifactstore.DecoderID, 0, len(decoderIDs)+1)

	values = append(values, DefinitionDecoderID)
	seen[DefinitionDecoderID] = struct{}{}

	for _, decoderID := range decoderIDs {
		if err := artifactstore.ValidateDecoderID(decoderID); err != nil {
			return nil, err
		}
		if _, duplicate := seen[decoderID]; duplicate {
			continue
		}
		seen[decoderID] = struct{}{}
		values = append(values, decoderID)
	}
	slices.Sort(values)
	return &Planner{
		decoderIDs: values,
		profiles:   cloneDiscoveryProfiles(profiles),
	}, nil
}

func (p *Planner) Build(
	value Workspace,
	definitionPreferences DiscoveryPreferences,
) (discovery.Plan, error) {
	preferences, err := mergeDiscoveryPreferences(
		value.Data.Discovery,
		definitionPreferences,
	)
	if err != nil {
		return discovery.Plan{}, err
	}

	plans := make([]discovery.SourcePlan, 0, len(value.Attachments))
	for _, attachment := range value.Attachments {
		if !attachment.Enabled {
			continue
		}
		operation, supported := attachmentOperationFor(attachment.Role)
		if !supported {
			return discovery.Plan{}, fmt.Errorf(
				"%w: unsupported attachment role %q",
				ErrInvalidWorkspace,
				attachment.Role,
			)
		}
		profile := p.profiles.Attached
		if operation.isPrimary {
			profile = p.profiles.Primary
		}
		attachmentData, err := decodeAttachmentData(attachment.Data)
		if err != nil {
			return discovery.Plan{}, err
		}
		sourcePlan := discovery.SourcePlan{
			SourceID: attachment.SourceID,
			AllowedDecoderIDs: append(
				[]artifactstore.DecoderID(nil),
				p.decoderIDs...,
			),
			Authoritative:     operation.defaultAuthoritative,
			MaxCandidateBytes: artifactstore.MaxCandidateBytes,
			MaxTotalBytes:     artifactstore.MaxScanBytes,
			MaxCandidates:     artifactstore.DefaultMaxCandidates,
			MaxEntries:        artifactstore.DefaultMaxEntries,
			MaxDepth:          artifactstore.DefaultMaxDepth,
			ExplicitLocators: append(
				[]artifactstore.Locator(nil),
				profile.ExplicitLocators...,
			),
			DirectoryRoots: cloneDirectoryRoots(
				profile.DirectoryRoots,
			),
		}

		if operation.includeReadmeWhenRequested &&
			preferences.IncludeReadme && profile.ReadmeLocator != "" {
			sourcePlan.ExplicitLocators = appendUniqueLocators(
				sourcePlan.ExplicitLocators,
				profile.ReadmeLocator,
			)
		}
		if operation.appliesWorkspaceDiscoveryPreferences {
			sourcePlan.ExplicitLocators = appendUniqueLocators(
				sourcePlan.ExplicitLocators,
				preferences.AdditionalLocators...,
			)
			sourcePlan.DirectoryRoots = appendDiscoveryRoots(
				sourcePlan.DirectoryRoots,
				preferences.AdditionalRoots,
			)
			sourcePlan.DecoderHints = appendDiscoveryPreferenceDecoderHints(
				preferences,
				p.decoderIDs,
			)
		}
		if operation.allowsAttachmentDiscoveryOverrides {
			if attachmentData.Recursive != nil {
				if len(sourcePlan.DirectoryRoots) == 0 {
					return discovery.Plan{}, fmt.Errorf(
						"%w: attachment role %q has no directory root to override",
						ErrInvalidWorkspace,
						attachment.Role,
					)
				}
				sourcePlan.DirectoryRoots[0].Recursive = *attachmentData.Recursive
			}
			if attachmentData.Authoritative != nil {
				sourcePlan.Authoritative = *attachmentData.Authoritative
			}
		}
		sourcePlan = sourcePlan.Normalized()
		plans = append(plans, sourcePlan)
	}
	sort.Slice(plans, func(left, right int) bool {
		return plans[left].SourceID < plans[right].SourceID
	})
	valuePlan := discovery.Plan{Sources: plans}
	if err := valuePlan.Validate(); err != nil {
		return discovery.Plan{}, err
	}
	return valuePlan, nil
}

func appendDiscoveryPreferenceDecoderHints(
	preferences DiscoveryPreferences,
	decoderIDs []artifactstore.DecoderID,
) []discovery.DecoderHint {
	output := make([]discovery.DecoderHint, 0, len(preferences.AdditionalLocators)+len(preferences.AdditionalRoots))
	for _, locator := range preferences.AdditionalLocators {
		output = append(output, discovery.DecoderHint{
			Locator:    locator,
			DecoderIDs: append([]artifactstore.DecoderID(nil), decoderIDs...),
		})
	}
	for _, root := range preferences.AdditionalRoots {
		output = append(output, discovery.DecoderHint{
			Locator:    root.Root,
			Recursive:  root.Recursive,
			DecoderIDs: append([]artifactstore.DecoderID(nil), decoderIDs...),
		})
	}
	return output
}

func cloneDiscoveryProfiles(value DiscoveryProfiles) DiscoveryProfiles {
	return DiscoveryProfiles{
		Primary: DiscoveryProfile{
			ExplicitLocators: append([]artifactstore.Locator(nil), value.Primary.ExplicitLocators...),
			ReadmeLocator:    value.Primary.ReadmeLocator,
			DirectoryRoots:   cloneDirectoryRoots(value.Primary.DirectoryRoots),
		},
		Attached: DiscoveryProfile{
			ExplicitLocators: append([]artifactstore.Locator(nil), value.Attached.ExplicitLocators...),
			ReadmeLocator:    value.Attached.ReadmeLocator,
			DirectoryRoots:   cloneDirectoryRoots(value.Attached.DirectoryRoots),
		},
	}
}

func cloneDirectoryRoots(
	values []discovery.DirectoryRoot,
) []discovery.DirectoryRoot {
	output := make([]discovery.DirectoryRoot, len(values))
	for index, value := range values {
		output[index] = value
		output[index].IncludePatterns = append(
			[]string(nil),
			value.IncludePatterns...,
		)
	}
	return output
}

func appendUniqueLocators(
	values []artifactstore.Locator,
	additions ...artifactstore.Locator,
) []artifactstore.Locator {
	seen := make(map[artifactstore.Locator]struct{}, len(values)+len(additions))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, value := range additions {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}

func appendDiscoveryRoots(
	values []discovery.DirectoryRoot,
	additions []DiscoveryRoot,
) []discovery.DirectoryRoot {
	for _, addition := range additions {
		merged := false
		for index := range values {
			if values[index].Root != addition.Root {
				continue
			}
			values[index].Recursive = values[index].Recursive || addition.Recursive
			values[index].IncludePatterns = mergePatterns(
				values[index].IncludePatterns,
				addition.IncludePatterns,
			)
			merged = true
			break
		}
		if merged {
			continue
		}
		values = append(values, discovery.DirectoryRoot{
			Root:      addition.Root,
			Recursive: addition.Recursive,
			IncludePatterns: append(
				[]string(nil),
				addition.IncludePatterns...,
			),
		})
	}
	return values
}

func mergeDiscoveryPreferences(
	left,
	right DiscoveryPreferences,
) (DiscoveryPreferences, error) {
	if err := validateDiscoveryPreferences(left); err != nil {
		return DiscoveryPreferences{}, err
	}
	if err := validateDiscoveryPreferences(right); err != nil {
		return DiscoveryPreferences{}, err
	}

	output := DiscoveryPreferences{
		IncludeReadme: left.IncludeReadme || right.IncludeReadme,
	}
	locators := make(map[artifactstore.Locator]struct{})
	for _, values := range [][]artifactstore.Locator{
		left.AdditionalLocators,
		right.AdditionalLocators,
	} {
		for _, locator := range values {
			if _, exists := locators[locator]; exists {
				continue
			}
			locators[locator] = struct{}{}
			output.AdditionalLocators = append(
				output.AdditionalLocators,
				locator,
			)
		}
	}

	roots := make(map[artifactstore.Locator]DiscoveryRoot)
	for _, values := range [][]DiscoveryRoot{
		left.AdditionalRoots,
		right.AdditionalRoots,
	} {
		for _, root := range values {
			current, exists := roots[root.Root]
			if !exists {
				current = root
				current.IncludePatterns = append(
					[]string(nil),
					root.IncludePatterns...,
				)
			} else {
				current.Recursive = current.Recursive || root.Recursive
				current.IncludePatterns = mergePatterns(
					current.IncludePatterns,
					root.IncludePatterns,
				)
			}
			roots[root.Root] = current
		}
	}
	for _, root := range roots {
		output.AdditionalRoots = append(output.AdditionalRoots, root)
	}
	slices.Sort(output.AdditionalLocators)
	sort.Slice(output.AdditionalRoots, func(left, right int) bool {
		return output.AdditionalRoots[left].Root <
			output.AdditionalRoots[right].Root
	})
	return output, validateDiscoveryPreferences(output)
}

func mergePatterns(left, right []string) []string {
	if len(left) == 0 || len(right) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	output := make([]string, 0, len(left)+len(right))
	for _, values := range [][]string{left, right} {
		for _, value := range values {
			if _, exists := seen[value]; exists {
				continue
			}
			seen[value] = struct{}{}
			output = append(output, value)
		}
	}
	sort.Strings(output)
	return output
}
