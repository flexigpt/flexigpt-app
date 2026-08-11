package discovery

import (
	"fmt"
	"maps"
	"slices"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/workspace/attachmentdata"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

type Planner struct {
	decoderIDs              []basespec.DecoderID
	profiles                spec.DiscoveryProfiles
	discoveryPolicyRevision string
}

func NewPlanner(
	profiles spec.DiscoveryProfiles,
	discoveryPolicyRevision string,
	decoderIDs ...basespec.DecoderID,
) (*Planner, error) {
	if err := validateDiscoveryProfiles(profiles); err != nil {
		return nil, fmt.Errorf(
			"%w: %w",
			spec.ErrInvalidWorkspace,
			err,
		)
	}
	if err := basespec.ValidateRequiredText(
		"workspace discovery policy revision",
		discoveryPolicyRevision,
		basespec.MaxVersionBytes,
	); err != nil {
		return nil, err
	}
	if len(profiles.Primary.ExplicitLocators) == 0 &&
		len(profiles.Primary.DirectoryRoots) == 0 {
		return nil, fmt.Errorf(
			"%w: primary discovery profile is required",
			spec.ErrInvalidWorkspace,
		)
	}

	seen := make(map[basespec.DecoderID]struct{}, len(decoderIDs))
	values := make([]basespec.DecoderID, 0, len(decoderIDs))

	for _, decoderID := range decoderIDs {
		if err := basespec.ValidateDecoderID(decoderID); err != nil {
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
			spec.ErrInvalidWorkspace,
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
	value spec.Workspace,
	observation DescriptorObservation,
) (discovery.Plan, error) {
	preferences, err := MergeDiscoveryPreferences(
		value.Data.Discovery,
		observation.Preferences,
	)
	if err != nil {
		return discovery.Plan{}, err
	}

	plans := make([]discovery.SourcePlan, 0, len(value.Attachments))
	sourcesByID := make(
		map[basespec.SourceID]bool,
		len(value.Sources),
	)
	for _, sourceValue := range value.Sources {
		if _, duplicate := sourcesByID[sourceValue.ID]; duplicate {
			return discovery.Plan{}, fmt.Errorf(
				"%w: duplicate Workspace source %q",
				spec.ErrInvalidWorkspace,
				sourceValue.ID,
			)
		}
		sourcesByID[sourceValue.ID] = sourceValue.Enabled
	}

	for _, att := range value.Attachments {
		if !att.Enabled {
			continue
		}
		sourceEnabled, exists := sourcesByID[att.SourceID]
		if !exists {
			return discovery.Plan{}, fmt.Errorf(
				"%w: attachment source %q is unavailable",
				spec.ErrInvalidWorkspace,
				att.SourceID,
			)
		}
		if !sourceEnabled {
			continue
		}
		operation, supported := attachmentdata.AttachmentOperationFor(att.Role)
		if !supported {
			return discovery.Plan{}, fmt.Errorf(
				"%w: unsupported attachment role %q",
				spec.ErrInvalidWorkspace,
				att.Role,
			)
		}
		profile := p.profiles.Attached
		if operation.IsPrimary {
			profile = p.profiles.Primary
		}
		attachmentData, err := attachmentdata.DecodeAttachmentData(att.Data)
		if err != nil {
			return discovery.Plan{}, err
		}
		if err := attachmentdata.ValidateAttachmentDataForRole(att.Role, attachmentData); err != nil {
			return discovery.Plan{}, err
		}
		sourcePlan := discovery.SourcePlan{
			SourceID: att.SourceID,
			AllowedDecoderIDs: append(
				[]basespec.DecoderID(nil),
				p.decoderIDs...,
			),
			Authoritative:     operation.DefaultAuthoritative,
			MaxCandidateBytes: basespec.MaxCandidateBytes,
			MaxTotalBytes:     basespec.MaxScanBytes,
			MaxCandidates:     basespec.DefaultMaxCandidates,
			MaxEntries:        basespec.DefaultMaxEntries,
			MaxDepth:          basespec.DefaultMaxDepth,
			ExplicitLocators: append(
				[]basespec.Locator(nil),
				profile.ExplicitLocators...,
			),
			DirectoryRoots: cloneDirectoryRoots(
				profile.DirectoryRoots,
			),
		}
		profilePreferences := spec.DiscoveryPreferences{
			AdditionalLocators: append(
				[]basespec.Locator(nil),
				profile.ExplicitLocators...,
			),
		}
		for _, root := range profile.DirectoryRoots {
			profilePreferences.AdditionalRoots = append(
				profilePreferences.AdditionalRoots,
				spec.DiscoveryRoot{
					Root:            root.Root,
					Recursive:       root.Recursive,
					IncludePatterns: append([]string(nil), root.IncludePatterns...),
				},
			)
		}
		sourcePlan.DecoderHints = appendDiscoveryPreferenceDecoderHints(
			nil,
			profilePreferences,
			p.decoderIDs,
		)

		if operation.IncludeReadmeWhenRequested &&
			preferences.IncludeReadme && profile.ReadmeLocator != "" {
			sourcePlan.ExplicitLocators = appendUniqueLocators(
				sourcePlan.ExplicitLocators,
				profile.ReadmeLocator,
			)
		}
		if operation.AppliesWorkspaceDiscoveryPreferences {
			sourcePlan.ExplicitLocators = appendUniqueLocators(
				sourcePlan.ExplicitLocators,
				preferences.AdditionalLocators...,
			)
			sourcePlan.DirectoryRoots = appendDiscoveryRoots(
				sourcePlan.DirectoryRoots,
				preferences.AdditionalRoots,
			)
			sourcePlan.DecoderHints = appendDiscoveryPreferenceDecoderHints(
				sourcePlan.DecoderHints,
				preferences,
				p.decoderIDs,
			)
			if att.SourceID == observation.SourceID {
				sourcePlan.ExpectedContentDigests = maps.Clone(
					observation.ExpectedContentDigests,
				)
			}
		}
		if operation.AllowsAttachmentDiscoveryOverrides {
			if attachmentData.Recursive != nil {
				if len(sourcePlan.DirectoryRoots) == 0 {
					return discovery.Plan{}, fmt.Errorf(
						"%w: attachment role %q has no directory root to override",
						spec.ErrInvalidWorkspace,
						att.Role,
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
	current []discovery.DecoderHint,
	preferences spec.DiscoveryPreferences,
	decoderIDs []basespec.DecoderID,
) []discovery.DecoderHint {
	type scope struct {
		locator   basespec.Locator
		recursive bool
	}

	output := make(
		[]discovery.DecoderHint,
		len(current),
		len(current)+len(preferences.AdditionalLocators)+len(preferences.AdditionalRoots),
	)
	for index, value := range current {
		output[index] = value
		output[index].DecoderIDs = append(
			[]basespec.DecoderID(nil),
			value.DecoderIDs...,
		)
	}
	byScope := make(map[scope]int, cap(output))
	for index, hint := range output {
		byScope[scope{
			locator:   hint.Locator,
			recursive: hint.Recursive,
		}] = index
	}

	appendHint := func(locator basespec.Locator, recursive bool) {
		key := scope{locator: locator, recursive: recursive}
		if index, found := byScope[key]; found {
			seen := make(map[basespec.DecoderID]struct{}, len(output[index].DecoderIDs))
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
			DecoderIDs: append([]basespec.DecoderID(nil), decoderIDs...),
		})
	}

	for _, locator := range preferences.AdditionalLocators {
		appendHint(locator, false)
	}
	for _, root := range preferences.AdditionalRoots {
		appendHint(root.Root, root.Recursive)
	}

	for index := range output {
		slices.Sort(output[index].DecoderIDs)
	}
	sort.Slice(output, func(left, right int) bool {
		if output[left].Locator != output[right].Locator {
			return output[left].Locator < output[right].Locator
		}
		return !output[left].Recursive && output[right].Recursive
	})
	return output
}

func cloneDiscoveryProfiles(value spec.DiscoveryProfiles) spec.DiscoveryProfiles {
	return spec.DiscoveryProfiles{
		Primary: spec.DiscoveryProfile{
			ExplicitLocators: append([]basespec.Locator(nil), value.Primary.ExplicitLocators...),
			ReadmeLocator:    value.Primary.ReadmeLocator,
			DirectoryRoots:   cloneDirectoryRoots(value.Primary.DirectoryRoots),
		},
		Attached: spec.DiscoveryProfile{
			ExplicitLocators: append([]basespec.Locator(nil), value.Attached.ExplicitLocators...),
			ReadmeLocator:    value.Attached.ReadmeLocator,
			DirectoryRoots:   cloneDirectoryRoots(value.Attached.DirectoryRoots),
		},
	}
}

// MergeDiscoveryProfile merges feature-contributed conventions into a profile
// without creating duplicate locators or directory roots. An empty pattern
// list means all files and therefore dominates narrower include patterns.
func MergeDiscoveryProfile(
	base spec.DiscoveryProfile,
	additions spec.DiscoveryProfile,
) spec.DiscoveryProfile {
	output := spec.DiscoveryProfile{
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
	values []basespec.Locator,
	additions ...basespec.Locator,
) []basespec.Locator {
	output := append([]basespec.Locator(nil), values...)
	seen := make(map[basespec.Locator]struct{}, len(output)+len(additions))
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
	values []spec.DirectoryRoot,
	additions []spec.DiscoveryRoot,
) []spec.DirectoryRoot {
	converted := make([]spec.DirectoryRoot, 0, len(additions))
	for _, addition := range additions {
		converted = append(converted, spec.DirectoryRoot{
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
	values []spec.DirectoryRoot,
	additions ...spec.DirectoryRoot,
) []spec.DirectoryRoot {
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
			output[index].IncludePatterns = MergePatterns(
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

func cloneDirectoryRoots(
	values []spec.DirectoryRoot,
) []spec.DirectoryRoot {
	output := make([]spec.DirectoryRoot, len(values))
	for index, value := range values {
		output[index] = value
		output[index].IncludePatterns = append(
			[]string(nil),
			value.IncludePatterns...,
		)
	}
	return output
}

func MergeDiscoveryPreferences(
	left,
	right spec.DiscoveryPreferences,
) (spec.DiscoveryPreferences, error) {
	if err := spec.ValidateDiscoveryPreferences(left); err != nil {
		return spec.DiscoveryPreferences{}, err
	}
	if err := spec.ValidateDiscoveryPreferences(right); err != nil {
		return spec.DiscoveryPreferences{}, err
	}

	output := spec.DiscoveryPreferences{
		IncludeReadme: left.IncludeReadme || right.IncludeReadme,
	}
	locators := make(map[basespec.Locator]struct{})
	for _, values := range [][]basespec.Locator{
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

	roots := make(map[basespec.Locator]spec.DiscoveryRoot)
	for _, values := range [][]spec.DiscoveryRoot{
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
				current.IncludePatterns = MergePatterns(
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
	return output, spec.ValidateDiscoveryPreferences(output)
}

func MergePatterns(left, right []string) []string {
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

func validateDiscoveryProfiles(value spec.DiscoveryProfiles) error {
	if err := validateDiscoveryProfile(value.Primary); err != nil {
		return err
	}
	return validateDiscoveryProfile(value.Attached)
}

func validateDiscoveryProfile(value spec.DiscoveryProfile) error {
	roots := make([]spec.DiscoveryRoot, 0, len(value.DirectoryRoots))
	for _, root := range value.DirectoryRoots {
		roots = append(roots, spec.DiscoveryRoot{
			Root:            root.Root,
			Recursive:       root.Recursive,
			IncludePatterns: append([]string(nil), root.IncludePatterns...),
		})
	}
	if err := spec.ValidateDiscoveryPreferences(spec.DiscoveryPreferences{
		AdditionalLocators: append(
			[]basespec.Locator(nil),
			value.ExplicitLocators...,
		),
		AdditionalRoots: roots,
	}); err != nil {
		return err
	}
	if value.ReadmeLocator == "" {
		return nil
	}
	return basespec.ValidateLocator(value.ReadmeLocator, false)
}
