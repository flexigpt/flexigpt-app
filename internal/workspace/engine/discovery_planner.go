package engine

import (
	"fmt"
	"maps"
	"slices"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
)

type Planner struct {
	decoderIDs              []artifactstore.DecoderID
	profiles                DiscoveryProfiles
	discoveryPolicyRevision string
}

func NewPlanner(
	profiles DiscoveryProfiles,
	discoveryPolicyRevision string,
	decoderIDs ...artifactstore.DecoderID,
) (*Planner, error) {
	if err := validateDiscoveryProfiles(profiles); err != nil {
		return nil, fmt.Errorf(
			"%w: %w",
			ErrInvalidWorkspace,
			err,
		)
	}
	if err := artifactstore.ValidateRequiredText(
		"workspace discovery policy revision",
		discoveryPolicyRevision,
		artifactstore.MaxVersionBytes,
	); err != nil {
		return nil, err
	}
	if len(profiles.Primary.ExplicitLocators) == 0 &&
		len(profiles.Primary.DirectoryRoots) == 0 {
		return nil, fmt.Errorf(
			"%w: primary discovery profile is required",
			ErrInvalidWorkspace,
		)
	}

	seen := make(map[artifactstore.DecoderID]struct{}, len(decoderIDs))
	values := make([]artifactstore.DecoderID, 0, len(decoderIDs))

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
	if len(values) == 0 {
		return nil, fmt.Errorf(
			"%w: Workspace discovery requires at least one decoder",
			ErrInvalidWorkspace,
		)
	}
	slices.Sort(values)
	return &Planner{
		decoderIDs:              values,
		profiles:                cloneDiscoveryProfiles(profiles),
		discoveryPolicyRevision: discoveryPolicyRevision,
	}, nil
}

func (p *Planner) Build(
	value Workspace,
	observation DescriptorObservation,
) (discovery.Plan, error) {
	preferences, err := mergeDiscoveryPreferences(
		value.Data.Discovery,
		observation.Preferences,
	)
	if err != nil {
		return discovery.Plan{}, err
	}

	plans := make([]discovery.SourcePlan, 0, len(value.Attachments))
	sourcesByID := make(
		map[artifactstore.SourceID]bool,
		len(value.Sources),
	)
	for _, sourceValue := range value.Sources {
		if _, duplicate := sourcesByID[sourceValue.ID]; duplicate {
			return discovery.Plan{}, fmt.Errorf(
				"%w: duplicate Workspace source %q",
				ErrInvalidWorkspace,
				sourceValue.ID,
			)
		}
		sourcesByID[sourceValue.ID] = sourceValue.Enabled
	}

	for _, attachment := range value.Attachments {
		if !attachment.Enabled {
			continue
		}
		sourceEnabled, exists := sourcesByID[attachment.SourceID]
		if !exists {
			return discovery.Plan{}, fmt.Errorf(
				"%w: attachment source %q is unavailable",
				ErrInvalidWorkspace,
				attachment.SourceID,
			)
		}
		if !sourceEnabled {
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
		if err := validateAttachmentDataForRole(attachment.Role, attachmentData); err != nil {
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
			if attachment.SourceID == observation.SourceID {
				sourcePlan.ExpectedContentDigests = maps.Clone(
					observation.ExpectedContentDigests,
				)
			}
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
	valuePlan := discovery.Plan{
		Revision: p.discoveryPolicyRevision,
		Sources:  plans,
	}
	if err := valuePlan.Validate(); err != nil {
		return discovery.Plan{}, err
	}
	return valuePlan, nil
}

func appendDiscoveryPreferenceDecoderHints(
	preferences DiscoveryPreferences,
	decoderIDs []artifactstore.DecoderID,
) []discovery.DecoderHint {
	type scope struct {
		locator   artifactstore.Locator
		recursive bool
	}

	output := make(
		[]discovery.DecoderHint,
		0,
		len(preferences.AdditionalLocators)+len(preferences.AdditionalRoots),
	)
	byScope := make(map[scope]int, cap(output))

	appendHint := func(locator artifactstore.Locator, recursive bool) {
		key := scope{locator: locator, recursive: recursive}
		if index, found := byScope[key]; found {
			seen := make(map[artifactstore.DecoderID]struct{}, len(output[index].DecoderIDs))
			for _, decoderID := range output[index].DecoderIDs {
				seen[decoderID] = struct{}{}
			}
			for _, decoderID := range decoderIDs {
				if _, exists := seen[decoderID]; exists {
					continue
				}
				seen[decoderID] = struct{}{}
				output[index].DecoderIDs = append(output[index].DecoderIDs, decoderID)
			}
			return
		}
		byScope[key] = len(output)
		output = append(output, discovery.DecoderHint{
			Locator:    locator,
			Recursive:  recursive,
			DecoderIDs: append([]artifactstore.DecoderID(nil), decoderIDs...),
		})
	}

	for _, locator := range preferences.AdditionalLocators {
		appendHint(locator, false)
	}
	for _, root := range preferences.AdditionalRoots {
		appendHint(root.Root, root.Recursive)
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Locator != output[right].Locator {
			return output[left].Locator < output[right].Locator
		}
		return !output[left].Recursive && output[right].Recursive
	})
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

// MergeDiscoveryProfile merges feature-contributed conventions into a profile
// without creating duplicate locators or directory roots. An empty pattern
// list means all files and therefore dominates narrower include patterns.
func MergeDiscoveryProfile(
	base DiscoveryProfile,
	additions DiscoveryProfile,
) DiscoveryProfile {
	output := DiscoveryProfile{
		ExplicitLocators: appendUniqueLocators(
			nil,
			base.ExplicitLocators...,
		),
		ReadmeLocator:  base.ReadmeLocator,
		DirectoryRoots: appendDirectoryRoots(nil, base.DirectoryRoots...),
	}
	output.ExplicitLocators = appendUniqueLocators(
		output.ExplicitLocators,
		additions.ExplicitLocators...,
	)
	output.DirectoryRoots = appendDirectoryRoots(
		output.DirectoryRoots,
		additions.DirectoryRoots...,
	)
	if additions.ReadmeLocator != "" {
		output.ReadmeLocator = additions.ReadmeLocator
	}
	return output
}

func appendUniqueLocators(
	values []artifactstore.Locator,
	additions ...artifactstore.Locator,
) []artifactstore.Locator {
	output := append([]artifactstore.Locator(nil), values...)
	seen := make(map[artifactstore.Locator]struct{}, len(output)+len(additions))
	for _, value := range output {
		seen[value] = struct{}{}
	}
	for _, value := range additions {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		output = append(output, value)
	}
	return output
}

func appendDiscoveryRoots(
	values []discovery.DirectoryRoot,
	additions []DiscoveryRoot,
) []discovery.DirectoryRoot {
	converted := make([]discovery.DirectoryRoot, 0, len(additions))
	for _, addition := range additions {
		converted = append(converted, discovery.DirectoryRoot{
			Root:      addition.Root,
			Recursive: addition.Recursive,
			IncludePatterns: append(
				[]string(nil),
				addition.IncludePatterns...,
			),
		})
	}
	return appendDirectoryRoots(values, converted...)
}

func appendDirectoryRoots(
	values []discovery.DirectoryRoot,
	additions ...discovery.DirectoryRoot,
) []discovery.DirectoryRoot {
	output := cloneDirectoryRoots(values)
	for _, addition := range additions {
		addition.IncludePatterns = append(
			[]string(nil),
			addition.IncludePatterns...,
		)
		merged := false
		for index := range output {
			if output[index].Root != addition.Root {
				continue
			}
			output[index].Recursive = output[index].Recursive || addition.Recursive
			output[index].IncludePatterns = mergePatterns(
				output[index].IncludePatterns,
				addition.IncludePatterns,
			)
			merged = true
			break
		}
		if merged {
			continue
		}
		output = append(output, addition)
	}
	return output
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
