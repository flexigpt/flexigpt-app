package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

type Planner struct {
	decoderIDs []artifactstore.DecoderID
	profiles   DiscoveryProfiles
}

func NewPlanner(
	profiles DiscoveryProfiles,
	decoderIDs ...artifactstore.DecoderID,
) (*Planner, error) {
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
		}
		if operation.allowsAttachmentDiscoveryOverrides {
			if attachmentData.Recursive != nil {
				sourcePlan.DirectoryRoots[0].Recursive = *attachmentData.Recursive
			}
			if attachmentData.Authoritative != nil {
				sourcePlan.Authoritative = *attachmentData.Authoritative
			}
		}
		sourcePlan.ApplyDefaults()
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

type DefinitionLoader struct {
	sources  source.Repository
	registry *source.Registry
	decoder  *DefinitionDecoder
}

func NewDefinitionLoader(
	sources source.Repository,
	registry *source.Registry,
) (*DefinitionLoader, error) {
	if sources == nil || registry == nil {
		return nil, fmt.Errorf(
			"%w: Workspace definition loader dependencies are incomplete",
			ErrInvalidWorkspace,
		)
	}
	return &DefinitionLoader{
		sources:  sources,
		registry: registry,
		decoder:  NewDefinitionDecoder(),
	}, nil
}

func (l *DefinitionLoader) Load(
	ctx context.Context,
	value Workspace,
) (DefinitionObservation, error) {
	if value.Data.PrimarySourceID == "" {
		return DefinitionObservation{}, nil
	}
	sourceValue, err := l.sources.Get(ctx, value.Data.PrimarySourceID)
	if err != nil {
		return DefinitionObservation{}, err
	}
	snapshot, err := l.registry.Open(ctx, sourceValue)
	if err != nil {
		return DefinitionObservation{}, err
	}
	defer snapshot.Close()

	observation := DefinitionObservation{
		SourceID:   sourceValue.ID,
		Generation: snapshot.Generation(),
	}
	entry, err := snapshot.Stat(ctx, DefinitionLocator)
	if errors.Is(err, artifactstore.ErrNotFound) {
		if err := snapshot.Confirm(ctx); err != nil {
			return DefinitionObservation{}, err
		}
		return observation, nil
	}
	if err != nil {
		return DefinitionObservation{}, err
	}
	if entry.SizeBytes > artifactstore.MaxDefinitionBodyBytes {
		return DefinitionObservation{}, fmt.Errorf(
			"%w: Workspace definition exceeds byte limit",
			ErrWorkspaceDefinitionInvalid,
		)
	}
	reader, err := snapshot.Open(ctx, DefinitionLocator)
	if err != nil {
		return DefinitionObservation{}, err
	}
	content, readErr := io.ReadAll(io.LimitReader(
		reader,
		artifactstore.MaxDefinitionBodyBytes+1,
	))
	closeErr := reader.Close()
	if readErr != nil {
		return DefinitionObservation{}, readErr
	}
	if closeErr != nil {
		return DefinitionObservation{}, closeErr
	}
	if len(content) > artifactstore.MaxDefinitionBodyBytes {
		return DefinitionObservation{}, ErrWorkspaceDefinitionInvalid
	}
	if err := snapshot.Confirm(ctx); err != nil {
		return DefinitionObservation{}, err
	}

	candidate := discovery.Candidate{
		Source:              sourceValue,
		Locator:             DefinitionLocator,
		SourceContentDigest: definition.DigestBytes(content),
		Content:             content,
	}
	decoded, diagnostics := l.decoder.Decode(ctx, candidate)
	if artifactstore.ContainsErrorDiagnostic(diagnostics) {
		return DefinitionObservation{}, fmt.Errorf(
			"%w: %s",
			ErrWorkspaceDefinitionInvalid,
			diagnostics[0].Message,
		)
	}
	if len(decoded) != 1 {
		return DefinitionObservation{}, ErrWorkspaceDefinitionInvalid
	}

	var document DefinitionDocument
	decoder := json.NewDecoder(bytes.NewReader(decoded[0].Definition.Body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return DefinitionObservation{}, err
	}
	observation.Preferences = document.Discovery
	return observation, nil
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
