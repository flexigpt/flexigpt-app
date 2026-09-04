package workspace

import (
	"context"
	"fmt"
	"maps"
	"path"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/attachmentdata"
	"github.com/flexigpt/flexigpt-app/internal/workspace/collectiondata"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

const (
	defaultAutomaticArtifactName      = "artifact"
	automaticArtifactNameSeparator    = "-"
	automaticArtifactNameDigestLength = 12
)

type workspaceCollectionBehavior struct {
	workspaceRootID basespec.RootID
	revision        string
	supports        map[basespec.ArtifactKind]spec.ArtifactSupport
	decoderIDs      []basespec.DecoderID
	profiles        spec.DiscoveryProfiles
}

var _ providerapi.DocumentPlanningBehavior = (*workspaceCollectionBehavior)(nil)

type workspacePlanningTopology struct {
	attachments     []providerapi.Attachment
	attachmentData  map[basespec.SourceID]spec.AttachmentData
	sources         map[basespec.SourceID]providerapi.Source
	primarySourceID basespec.SourceID
}

type workspaceDescriptorObservation struct {
	Preferences            spec.DiscoveryPreferences
	SourceID               basespec.SourceID
	Generation             string
	ExpectedContentDigests map[basespec.Locator]cryptoutil.Digest
}

func newWorkspaceCollectionBehavior(
	config ProviderConfig,
) (*workspaceCollectionBehavior, error) {
	normalized, err := normalizeProviderConfig(config)
	if err != nil {
		return nil, err
	}

	supports := make(
		map[basespec.ArtifactKind]spec.ArtifactSupport,
		len(normalized.supports),
	)
	for _, support := range normalized.supports {
		supports[support.Kind] = support
	}

	return &workspaceCollectionBehavior{
		workspaceRootID: normalized.workspaceRootID,
		revision:        normalized.revision,
		supports:        supports,
		decoderIDs:      append([]basespec.DecoderID(nil), normalized.decoderIDs...),
		profiles:        cloneWorkspaceDiscoveryProfiles(normalized.profiles),
	}, nil
}

func (*workspaceCollectionBehavior) CollectionKind() basespec.CollectionKind {
	return artifactbuiltin.WorkspaceCollectionV1Kind
}

func (b *workspaceCollectionBehavior) Revision() string {
	if b == nil {
		return ""
	}
	return b.revision
}

func (b *workspaceCollectionBehavior) BuildDiscoveryPlanWithDocuments(
	ctx context.Context,
	collectionValue providerapi.Collection,
	attachments []providerapi.Attachment,
	sources []providerapi.Source,
	documents providerapi.PlanningDocumentReader,
) (providerapi.Plan, error) {
	if b == nil {
		return providerapi.Plan{}, basespec.ErrClosed
	}
	if ctx == nil {
		return providerapi.Plan{}, fmt.Errorf(
			"%w: Workspace provider planning context is nil",
			spec.ErrInvalidWorkspace,
		)
	}
	if err := ctx.Err(); err != nil {
		return providerapi.Plan{}, err
	}
	if documents == nil {
		return providerapi.Plan{}, fmt.Errorf(
			"%w: Workspace provider planning document reader is required",
			spec.ErrInvalidWorkspace,
		)
	}
	if err := validateWorkspaceProviderCollection(collectionValue); err != nil {
		return providerapi.Plan{}, err
	}
	if collectionValue.RootID != b.workspaceRootID {
		return providerapi.Plan{}, fmt.Errorf(
			"%w: Workspace Collection belongs to Root %q, expected %q",
			spec.ErrInvalidWorkspace,
			collectionValue.RootID,
			b.workspaceRootID,
		)
	}
	if collectionValue.Kind != b.CollectionKind() {
		return providerapi.Plan{}, fmt.Errorf(
			"%w: Workspace provider received Collection kind %q",
			spec.ErrInvalidWorkspace,
			collectionValue.Kind,
		)
	}

	data, err := collectiondata.DecodeCollectionData(collectionValue.Data)
	if err != nil {
		return providerapi.Plan{}, err
	}
	topology, err := workspacePlanningTopologyFor(
		collectionValue,
		attachments,
		sources,
	)
	if err != nil {
		return providerapi.Plan{}, err
	}

	observation, err := readWorkspaceDescriptor(
		ctx,
		topology.primarySourceID,
		documents,
	)
	if err != nil {
		return providerapi.Plan{}, err
	}
	preferences, err := mergeWorkspaceDiscoveryPreferences(
		data.Discovery,
		observation.Preferences,
	)
	if err != nil {
		return providerapi.Plan{}, err
	}

	return b.buildDiscoveryPlan(
		topology,
		preferences,
		observation,
	)
}

func (b *workspaceCollectionBehavior) DecideAutomaticAdoption(
	ctx context.Context,
	input providerapi.AdoptionInput,
) (providerapi.AdoptionDecision, error) {
	if b == nil {
		return providerapi.AdoptionDecision{}, basespec.ErrClosed
	}
	if ctx == nil {
		return providerapi.AdoptionDecision{}, fmt.Errorf(
			"%w: Workspace automatic-adoption context is nil",
			spec.ErrInvalidWorkspace,
		)
	}
	if err := ctx.Err(); err != nil {
		return providerapi.AdoptionDecision{}, err
	}
	if input.Collection.RootID != b.workspaceRootID ||
		input.Collection.Kind != b.CollectionKind() {
		return providerapi.AdoptionDecision{}, fmt.Errorf(
			"%w: Workspace automatic adoption received another Collection",
			spec.ErrInvalidWorkspace,
		)
	}
	if input.Attachment.RootID != input.Collection.RootID ||
		input.Attachment.CollectionID != input.Collection.ID ||
		input.Attachment.SourceID != input.Occurrence.SourceID ||
		input.Occurrence.RootID != input.Collection.RootID ||
		input.Occurrence.CollectionID != input.Collection.ID {
		return providerapi.AdoptionDecision{}, fmt.Errorf(
			"%w: Workspace automatic adoption input identities do not match",
			spec.ErrInvalidWorkspace,
		)
	}

	support, supported := b.supports[input.Occurrence.Kind]
	if !supported || !input.Attachment.Enabled {
		return providerapi.AdoptionDecision{}, nil
	}
	if input.Definition.Kind != input.Occurrence.Kind {
		return workspaceAdoptionDiagnostic(
			input.Occurrence,
			spec.DiagnosticCodeArtifactKindMismatch,
			fmt.Sprintf(
				"definition kind %q does not match occurrence kind %q",
				input.Definition.Kind,
				input.Occurrence.Kind,
			),
		), nil
	}
	if input.Definition.SchemaID != support.SchemaID {
		return workspaceAdoptionDiagnostic(
			input.Occurrence,
			spec.DiagnosticCodeArtifactSchemaUnsupported,
			fmt.Sprintf(
				"definition schema %q is not supported for kind %q",
				input.Definition.SchemaID,
				input.Definition.Kind,
			),
		), nil
	}
	if err := support.Validator(input.Definition); err != nil {
		return workspaceAdoptionDiagnostic(
			input.Occurrence,
			spec.DiagnosticCodeProjectionInvalid,
			err.Error(),
		), nil
	}

	data, err := jsonutil.MarshalCanonicalObject(
		spec.ArtifactData{},
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return providerapi.AdoptionDecision{}, err
	}

	return providerapi.AdoptionDecision{
		Adopt: true,
		Name: workspaceAutomaticArtifactName(
			input.Definition.LogicalName,
			input.Occurrence,
		),
		Enabled: true,
		Data:    data,
	}, nil
}

func (b *workspaceCollectionBehavior) buildDiscoveryPlan(
	topology workspacePlanningTopology,
	preferences spec.DiscoveryPreferences,
	observation workspaceDescriptorObservation,
) (providerapi.Plan, error) {
	plans := make(
		[]providerapi.SourcePlan,
		0,
		len(topology.attachments),
	)

	for _, attachment := range topology.attachments {
		sourceValue := topology.sources[attachment.SourceID]
		if !attachment.Enabled || !sourceValue.Enabled {
			continue
		}

		operation, _ := attachmentdata.AttachmentOperationFor(
			attachment.Role,
		)
		profile := b.profiles.Attached
		if operation.IsPrimary {
			profile = b.profiles.Primary
		}

		sourcePlan := providerapi.SourcePlan{
			SourceID: attachment.SourceID,
			AllowedDecoderIDs: append(
				[]basespec.DecoderID(nil),
				b.decoderIDs...,
			),
			Authoritative: operation.DefaultAuthoritative,
			ExplicitLocators: append(
				[]basespec.Locator(nil),
				profile.ExplicitLocators...,
			),
			DirectoryRoots: workspaceProviderDirectoryRoots(
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
					Root:      root.Root,
					Recursive: root.Recursive,
					IncludePatterns: append(
						[]string(nil),
						root.IncludePatterns...,
					),
				},
			)
		}
		sourcePlan.DecoderHints = appendWorkspacePreferenceDecoderHints(
			nil,
			profilePreferences,
			b.decoderIDs,
		)

		if operation.IncludeReadmeWhenRequested &&
			preferences.IncludeReadme &&
			profile.ReadmeLocator != "" {
			sourcePlan.ExplicitLocators = appendWorkspaceUniqueLocators(
				sourcePlan.ExplicitLocators,
				profile.ReadmeLocator,
			)
		}

		if operation.AppliesWorkspaceDiscoveryPreferences {
			sourcePlan.ExplicitLocators = appendWorkspaceUniqueLocators(
				sourcePlan.ExplicitLocators,
				preferences.AdditionalLocators...,
			)
			sourcePlan.DirectoryRoots = appendWorkspaceDiscoveryRoots(
				sourcePlan.DirectoryRoots,
				preferences.AdditionalRoots,
			)
			sourcePlan.DecoderHints = appendWorkspacePreferenceDecoderHints(
				sourcePlan.DecoderHints,
				preferences,
				b.decoderIDs,
			)
			if attachment.SourceID == observation.SourceID {
				sourcePlan.ExpectedGeneration = observation.Generation
				sourcePlan.ExpectedContentDigests = maps.Clone(
					observation.ExpectedContentDigests,
				)
			}
		}

		attachmentData := topology.attachmentData[attachment.SourceID]
		if operation.AllowsAttachmentDiscoveryOverrides {
			if attachmentData.Recursive != nil {
				if len(sourcePlan.DirectoryRoots) == 0 {
					return providerapi.Plan{}, fmt.Errorf(
						"%w: Workspace attachment role %q has no directory root to override",
						spec.ErrInvalidWorkspace,
						attachment.Role,
					)
				}
				sourcePlan.DirectoryRoots[0].Recursive = *attachmentData.Recursive
			}
			if attachmentData.Authoritative != nil {
				sourcePlan.Authoritative = *attachmentData.Authoritative
			}
		}

		plans = append(plans, sourcePlan.Normalized())
	}

	sort.Slice(plans, func(left, right int) bool {
		return plans[left].SourceID < plans[right].SourceID
	})

	plan := providerapi.Plan{
		Revision: b.revision,
		Sources:  plans,
	}.Normalized()
	if err := plan.Validate(); err != nil {
		return providerapi.Plan{}, err
	}
	return plan, nil
}

func workspaceAdoptionDiagnostic(
	occurrence providerapi.Occurrence,
	code string,
	message string,
) providerapi.AdoptionDecision {
	return providerapi.AdoptionDecision{
		Diagnostics: []diagnostic.Diagnostic{{
			Severity: diagnostic.DiagnosticError,
			Code:     code,
			Message:  diagnostic.BoundedDiagnosticMessage(message),
			Location: &diagnostic.DiagnosticLocation{
				Locator:            occurrence.Locator,
				SubresourceLocator: occurrence.SubresourceLocator,
			},
		}},
	}
}

func workspaceAutomaticArtifactName(
	logicalName basespec.LogicalName,
	occurrence providerapi.Occurrence,
) string {
	base := strings.TrimSpace(string(logicalName))
	if base == "" {
		base = defaultAutomaticArtifactName
	}

	digest := cryptoutil.DigestBytes([]byte(
		string(occurrence.SourceID) + "\x00" +
			string(occurrence.Locator) + "\x00" +
			string(occurrence.SubresourceLocator),
	))
	suffix := strings.TrimPrefix(
		string(digest),
		cryptoutil.DigestSHA256Prefix,
	)
	if len(suffix) > automaticArtifactNameDigestLength {
		suffix = suffix[:automaticArtifactNameDigestLength]
	}

	maximum := basespec.MaxDisplayNameBytes -
		len(automaticArtifactNameSeparator) -
		len(suffix)
	for len(base) > maximum {
		_, size := utf8.DecodeLastRuneInString(base)
		base = base[:len(base)-size]
	}
	return base + automaticArtifactNameSeparator + suffix
}

func workspacePlanningTopologyFor(
	collectionValue providerapi.Collection,
	attachments []providerapi.Attachment,
	sources []providerapi.Source,
) (workspacePlanningTopology, error) {
	topology := workspacePlanningTopology{
		attachments: make(
			[]providerapi.Attachment,
			0,
			len(attachments),
		),
		attachmentData: make(
			map[basespec.SourceID]spec.AttachmentData,
			len(attachments),
		),
		sources: make(
			map[basespec.SourceID]providerapi.Source,
			len(sources),
		),
	}
	attachmentsBySource := make(
		map[basespec.SourceID]struct{},
		len(attachments),
	)

	for index, attachment := range attachments {
		if err := validateWorkspaceProviderAttachment(
			collectionValue,
			attachment,
		); err != nil {
			return workspacePlanningTopology{}, fmt.Errorf(
				"workspace attachment %d: %w",
				index,
				err,
			)
		}
		if _, duplicate := attachmentsBySource[attachment.SourceID]; duplicate {
			return workspacePlanningTopology{}, fmt.Errorf(
				"%w: duplicate Workspace attachment Source %q",
				spec.ErrInvalidWorkspace,
				attachment.SourceID,
			)
		}

		data, err := attachmentdata.DecodeAttachmentData(
			attachment.Data,
		)
		if err != nil {
			return workspacePlanningTopology{}, err
		}
		if err := attachmentdata.ValidateAttachmentDataForRole(
			attachment.Role,
			data,
		); err != nil {
			return workspacePlanningTopology{}, err
		}

		attachmentsBySource[attachment.SourceID] = struct{}{}
		topology.attachmentData[attachment.SourceID] = data
		topology.attachments = append(
			topology.attachments,
			attachment.Clone(),
		)
	}

	for index, sourceValue := range sources {
		if err := validateWorkspaceProviderSource(
			collectionValue,
			sourceValue,
		); err != nil {
			return workspacePlanningTopology{}, fmt.Errorf(
				"workspace Source %d: %w",
				index,
				err,
			)
		}
		if _, duplicate := topology.sources[sourceValue.ID]; duplicate {
			return workspacePlanningTopology{}, fmt.Errorf(
				"%w: duplicate Workspace Source %q",
				spec.ErrInvalidWorkspace,
				sourceValue.ID,
			)
		}
		if _, attached := attachmentsBySource[sourceValue.ID]; !attached {
			return workspacePlanningTopology{}, fmt.Errorf(
				"%w: Workspace provider received unattached Source %q",
				spec.ErrInvalidWorkspace,
				sourceValue.ID,
			)
		}
		topology.sources[sourceValue.ID] = sourceValue
	}

	primaryCount := 0
	for _, attachment := range topology.attachments {
		sourceValue, found := topology.sources[attachment.SourceID]
		if !found {
			return workspacePlanningTopology{}, fmt.Errorf(
				"%w: Workspace attachment Source %q is unavailable",
				spec.ErrInvalidWorkspace,
				attachment.SourceID,
			)
		}
		if attachment.Enabled && !sourceValue.Enabled {
			return workspacePlanningTopology{}, fmt.Errorf(
				"%w: enabled Workspace attachment %q uses a disabled Source",
				spec.ErrInvalidWorkspace,
				attachment.SourceID,
			)
		}

		operation, _ := attachmentdata.AttachmentOperationFor(
			attachment.Role,
		)
		if !operation.IsPrimary {
			continue
		}

		primaryCount++
		if !attachment.Enabled || !sourceValue.Enabled {
			return workspacePlanningTopology{}, fmt.Errorf(
				"%w: Workspace primary Source and attachment must be enabled",
				spec.ErrInvalidWorkspace,
			)
		}
		if sourceValue.Kind != operation.RequiredSourceKind {
			return workspacePlanningTopology{}, fmt.Errorf(
				"%w: Workspace primary Source must have kind %q",
				spec.ErrInvalidWorkspace,
				operation.RequiredSourceKind,
			)
		}
		topology.primarySourceID = sourceValue.ID
	}

	if primaryCount > 1 {
		return workspacePlanningTopology{}, fmt.Errorf(
			"%w: Workspace cannot have multiple primary attachments",
			spec.ErrInvalidWorkspace,
		)
	}

	sort.Slice(topology.attachments, func(left, right int) bool {
		return topology.attachments[left].SourceID <
			topology.attachments[right].SourceID
	})
	return topology, nil
}

func validateWorkspaceProviderCollection(
	value providerapi.Collection,
) error {
	if err := basespec.ValidateRootID(value.RootID); err != nil {
		return err
	}
	if err := basespec.ValidateCollectionID(value.ID); err != nil {
		return err
	}
	if err := basespec.ValidateCollectionKind(value.Kind); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"Workspace Collection display name",
		value.DisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"Workspace Collection description",
		value.Description,
		basespec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if value.Revision == 0 {
		return fmt.Errorf(
			"%w: Workspace Collection revision is required",
			spec.ErrInvalidWorkspace,
		)
	}
	return nil
}

func validateWorkspaceProviderAttachment(
	collectionValue providerapi.Collection,
	value providerapi.Attachment,
) error {
	if value.RootID != collectionValue.RootID ||
		value.CollectionID != collectionValue.ID {
		return fmt.Errorf(
			"%w: Workspace attachment belongs to another Collection",
			spec.ErrInvalidWorkspace,
		)
	}
	if err := basespec.ValidateSourceID(value.SourceID); err != nil {
		return err
	}
	if err := basespec.ValidateAttachmentRole(value.Role); err != nil {
		return err
	}
	if _, supported := attachmentdata.AttachmentOperationFor(value.Role); !supported {
		return fmt.Errorf(
			"%w: unsupported Workspace attachment role %q",
			spec.ErrInvalidWorkspace,
			value.Role,
		)
	}
	if value.Revision == 0 {
		return fmt.Errorf(
			"%w: Workspace attachment revision is required",
			spec.ErrInvalidWorkspace,
		)
	}
	return nil
}

func validateWorkspaceProviderSource(
	collectionValue providerapi.Collection,
	value providerapi.Source,
) error {
	if value.RootID != collectionValue.RootID {
		return fmt.Errorf(
			"%w: Workspace Source belongs to another Root",
			spec.ErrInvalidWorkspace,
		)
	}
	if err := basespec.ValidateSourceID(value.ID); err != nil {
		return err
	}
	if err := basespec.ValidateStorageKey(value.StorageKey); err != nil {
		return err
	}
	if err := basespec.ValidateSourceKind(value.Kind); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"Workspace Source display name",
		value.DisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if value.Revision == 0 {
		return fmt.Errorf(
			"%w: Workspace Source revision is required",
			spec.ErrInvalidWorkspace,
		)
	}
	return nil
}

func readWorkspaceDescriptor(
	ctx context.Context,
	primarySourceID basespec.SourceID,
	reader providerapi.PlanningDocumentReader,
) (workspaceDescriptorObservation, error) {
	if primarySourceID == "" {
		return workspaceDescriptorObservation{}, nil
	}

	request := providerapi.PlanningDocumentRequest{
		SourceID:       primarySourceID,
		Locator:        artifactbuiltin.WorkspaceDescriptorFileName,
		ExpectedSchema: artifactbuiltin.WorkspaceCollectionV1SchemaKey,
	}
	result, err := reader.ReadCanonicalDocument(ctx, request)
	if err != nil {
		return workspaceDescriptorObservation{}, fmt.Errorf(
			"%w: read Workspace descriptor: %w",
			spec.ErrWorkspaceDefinitionInvalid,
			err,
		)
	}
	if result.SourceID != primarySourceID {
		return workspaceDescriptorObservation{}, fmt.Errorf(
			"%w: Workspace descriptor reader returned another Source",
			spec.ErrWorkspaceDefinitionInvalid,
		)
	}
	if err := result.Validate(); err != nil {
		return workspaceDescriptorObservation{}, fmt.Errorf(
			"%w: Workspace descriptor reader returned invalid state: %w",
			spec.ErrWorkspaceDefinitionInvalid,
			err,
		)
	}

	observation := workspaceDescriptorObservation{
		SourceID:   primarySourceID,
		Generation: result.Generation,
	}
	if !result.Found {
		return observation, nil
	}

	parsed := result.Document.Clone()
	if parsed.Key != artifactbuiltin.WorkspaceCollectionV1SchemaKey {
		return workspaceDescriptorObservation{}, fmt.Errorf(
			"%w: Workspace descriptor schema does not identify %q",
			spec.ErrWorkspaceDefinitionInvalid,
			artifactbuiltin.WorkspaceCollectionV1Kind,
		)
	}

	descriptor, err := artifactbuiltin.ParseWorkspaceCollectionV1(
		parsed.Raw,
	)
	if err != nil {
		return workspaceDescriptorObservation{}, fmt.Errorf(
			"%w: decode canonical Workspace descriptor: %w",
			spec.ErrWorkspaceDefinitionInvalid,
			err,
		)
	}
	if descriptor.Digest == nil ||
		cryptoutil.Digest(*descriptor.Digest) != parsed.Digest {
		return workspaceDescriptorObservation{}, fmt.Errorf(
			"%w: canonical Workspace descriptor digest differs from schema registry output",
			basespec.ErrDigestMismatch,
		)
	}

	body, err := artifactbuiltin.DecodeWorkspaceCollectionV1Body(
		descriptor.Body,
	)
	if err != nil {
		return workspaceDescriptorObservation{}, fmt.Errorf(
			"%w: %w",
			spec.ErrWorkspaceDefinitionInvalid,
			err,
		)
	}

	base, err := workspaceDescriptorBaseLocator(
		artifactbuiltin.WorkspaceDescriptorFileName,
	)
	if err != nil {
		return workspaceDescriptorObservation{}, fmt.Errorf(
			"%w: resolve Workspace descriptor base: %w",
			spec.ErrWorkspaceDefinitionInvalid,
			err,
		)
	}

	preferences, err := resolveWorkspaceDescriptorPreferences(
		body.Discovery,
		base,
	)
	if err != nil {
		return workspaceDescriptorObservation{}, fmt.Errorf(
			"%w: resolve Workspace descriptor discovery preferences: %w",
			spec.ErrWorkspaceDefinitionInvalid,
			err,
		)
	}

	expectedContentDigests := make(
		map[basespec.Locator]cryptoutil.Digest,
		len(descriptor.Members),
	)
	for index, member := range descriptor.Members {
		switch {
		case member.Locator != "":
			memberLocator := basespec.Locator(member.Locator)
			resolvedLocator, err := resolveWorkspaceRelativeLocator(
				base,
				memberLocator,
				false,
			)
			if err != nil {
				return workspaceDescriptorObservation{}, fmt.Errorf(
					"%w: Workspace descriptor member %d locator: %w",
					spec.ErrWorkspaceDefinitionInvalid,
					index,
					err,
				)
			}
			if member.SubresourceLocator != "" {
				return workspaceDescriptorObservation{}, fmt.Errorf(
					"%w: Workspace descriptor member %d subresources are not supported by source discovery",
					spec.ErrWorkspaceDefinitionInvalid,
					index,
				)
			}

			preferences.AdditionalLocators = appendWorkspaceUniqueLocators(
				preferences.AdditionalLocators,
				resolvedLocator,
			)
			if member.Digest != nil {
				digest := cryptoutil.Digest(*member.Digest)
				if err := cryptoutil.ValidateDigest(digest); err != nil {
					return workspaceDescriptorObservation{}, err
				}
				if existing, found := expectedContentDigests[resolvedLocator]; found &&
					existing != digest {
					return workspaceDescriptorObservation{}, fmt.Errorf(
						"%w: Workspace descriptor members declare conflicting digests for %q",
						spec.ErrWorkspaceDefinitionInvalid,
						resolvedLocator,
					)
				}
				expectedContentDigests[resolvedLocator] = digest
			}

		case member.URI != "":
			return workspaceDescriptorObservation{}, fmt.Errorf(
				"%w: Workspace descriptor member %d requires an external URI resolver: %w",
				spec.ErrWorkspaceDefinitionInvalid,
				index,
				basespec.ErrUnsupported,
			)

		default:
			return workspaceDescriptorObservation{}, fmt.Errorf(
				"%w: Workspace descriptor member %d requires unsupported embedded content handling: %w",
				spec.ErrWorkspaceDefinitionInvalid,
				index,
				basespec.ErrUnsupported,
			)
		}
	}

	if err := spec.ValidateDiscoveryPreferences(preferences); err != nil {
		return workspaceDescriptorObservation{}, fmt.Errorf(
			"%w: %w",
			spec.ErrWorkspaceDefinitionInvalid,
			err,
		)
	}

	observation.Preferences = preferences
	if len(expectedContentDigests) != 0 {
		observation.ExpectedContentDigests = expectedContentDigests
	}
	return observation, nil
}

func resolveWorkspaceDescriptorPreferences(
	input artifactbuiltin.WorkspaceDiscoveryV1,
	base basespec.Locator,
) (spec.DiscoveryPreferences, error) {
	output := spec.DiscoveryPreferences{
		IncludeReadme: input.IncludeReadme,
	}
	for index, locator := range input.AdditionalLocators {
		resolved, err := resolveWorkspaceRelativeLocator(
			base,
			basespec.Locator(locator),
			false,
		)
		if err != nil {
			return spec.DiscoveryPreferences{}, fmt.Errorf(
				"additionalLocators[%d]: %w",
				index,
				err,
			)
		}
		output.AdditionalLocators = append(
			output.AdditionalLocators,
			resolved,
		)
	}
	for index, root := range input.AdditionalRoots {
		resolved, err := resolveWorkspaceRelativeLocator(
			base,
			basespec.Locator(root.Root),
			true,
		)
		if err != nil {
			return spec.DiscoveryPreferences{}, fmt.Errorf(
				"additionalRoots[%d]: %w",
				index,
				err,
			)
		}
		output.AdditionalRoots = append(
			output.AdditionalRoots,
			spec.DiscoveryRoot{
				Root:            resolved,
				Recursive:       root.Recursive,
				IncludePatterns: append([]string(nil), root.IncludePatterns...),
			},
		)
	}
	return output, nil
}

func workspaceDescriptorBaseLocator(
	document basespec.Locator,
) (basespec.Locator, error) {
	if err := basespec.ValidateLocator(document, false); err != nil {
		return "", fmt.Errorf("workspace descriptor locator: %w", err)
	}

	base := basespec.Locator(path.Dir(string(document)))
	if err := basespec.ValidateLocator(base, true); err != nil {
		return "", fmt.Errorf("workspace descriptor base locator: %w", err)
	}
	return base, nil
}

func resolveWorkspaceRelativeLocator(
	base basespec.Locator,
	relative basespec.Locator,
	allowRelativeRoot bool,
) (basespec.Locator, error) {
	if err := basespec.ValidatePortableLocator(base, true); err != nil {
		return "", fmt.Errorf("portable base locator: %w", err)
	}
	if err := basespec.ValidatePortableLocator(
		relative,
		allowRelativeRoot,
	); err != nil {
		return "", fmt.Errorf("portable relative locator: %w", err)
	}

	var resolved basespec.Locator
	switch {
	case base == ".":
		resolved = relative
	case relative == ".":
		resolved = base
	default:
		resolved = basespec.Locator(
			path.Join(string(base), string(relative)),
		)
	}

	if err := basespec.ValidatePortableLocator(
		resolved,
		allowRelativeRoot,
	); err != nil {
		return "", fmt.Errorf("resolved portable locator: %w", err)
	}
	return resolved, nil
}

func workspaceProviderDirectoryRoots(
	input []spec.DirectoryRoot,
) []providerapi.DirectoryRoot {
	output := make([]providerapi.DirectoryRoot, len(input))
	for index, root := range input {
		output[index] = providerapi.DirectoryRoot{
			Root:      root.Root,
			Recursive: root.Recursive,
			IncludePatterns: append(
				[]string(nil),
				root.IncludePatterns...,
			),
		}
	}
	return output
}

func appendWorkspacePreferenceDecoderHints(
	current []providerapi.DecoderHint,
	preferences spec.DiscoveryPreferences,
	decoderIDs []basespec.DecoderID,
) []providerapi.DecoderHint {
	type scope struct {
		locator   basespec.Locator
		recursive bool
	}

	output := make(
		[]providerapi.DecoderHint,
		len(current),
		len(current)+
			len(preferences.AdditionalLocators)+
			len(preferences.AdditionalRoots),
	)
	for index, value := range current {
		output[index] = value.Clone()
	}

	byScope := make(map[scope]int, len(output))
	for index, hint := range output {
		byScope[scope{
			locator:   hint.Locator,
			recursive: hint.Recursive,
		}] = index
	}

	appendHint := func(
		locator basespec.Locator,
		recursive bool,
	) {
		key := scope{locator: locator, recursive: recursive}
		if index, found := byScope[key]; found {
			seen := make(
				map[basespec.DecoderID]struct{},
				len(output[index].DecoderIDs),
			)
			for _, decoderID := range output[index].DecoderIDs {
				seen[decoderID] = struct{}{}
			}
			for _, decoderID := range decoderIDs {
				if _, exists := seen[decoderID]; exists {
					continue
				}
				seen[decoderID] = struct{}{}
				output[index].DecoderIDs = append(
					output[index].DecoderIDs,
					decoderID,
				)
			}
			return
		}

		byScope[key] = len(output)
		output = append(output, providerapi.DecoderHint{
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

func cloneWorkspaceDiscoveryProfiles(
	value spec.DiscoveryProfiles,
) spec.DiscoveryProfiles {
	return spec.DiscoveryProfiles{
		Primary: spec.DiscoveryProfile{
			ExplicitLocators: append(
				[]basespec.Locator(nil),
				value.Primary.ExplicitLocators...,
			),
			ReadmeLocator: value.Primary.ReadmeLocator,
			DirectoryRoots: cloneWorkspaceDirectoryRoots(
				value.Primary.DirectoryRoots,
			),
		},
		Attached: spec.DiscoveryProfile{
			ExplicitLocators: append(
				[]basespec.Locator(nil),
				value.Attached.ExplicitLocators...,
			),
			ReadmeLocator: value.Attached.ReadmeLocator,
			DirectoryRoots: cloneWorkspaceDirectoryRoots(
				value.Attached.DirectoryRoots,
			),
		},
	}
}

func mergeWorkspaceDiscoveryProfile(
	base spec.DiscoveryProfile,
	additions spec.DiscoveryProfile,
) spec.DiscoveryProfile {
	output := spec.DiscoveryProfile{
		ExplicitLocators: appendWorkspaceUniqueLocators(
			nil,
			base.ExplicitLocators...,
		),
		ReadmeLocator: base.ReadmeLocator,
		DirectoryRoots: appendWorkspaceDirectoryRoots(
			nil,
			base.DirectoryRoots...,
		),
	}
	output.ExplicitLocators = appendWorkspaceUniqueLocators(
		output.ExplicitLocators,
		additions.ExplicitLocators...,
	)
	output.DirectoryRoots = appendWorkspaceDirectoryRoots(
		output.DirectoryRoots,
		additions.DirectoryRoots...,
	)
	if additions.ReadmeLocator != "" {
		output.ReadmeLocator = additions.ReadmeLocator
	}
	return output
}

func workspaceDiscoveryProfilesEmpty(
	value spec.DiscoveryProfiles,
) bool {
	return len(value.Primary.ExplicitLocators) == 0 &&
		len(value.Primary.DirectoryRoots) == 0 &&
		len(value.Attached.ExplicitLocators) == 0 &&
		len(value.Attached.DirectoryRoots) == 0
}

func appendWorkspaceUniqueLocators(
	values []basespec.Locator,
	additions ...basespec.Locator,
) []basespec.Locator {
	output := append([]basespec.Locator(nil), values...)
	seen := make(
		map[basespec.Locator]struct{},
		len(output)+len(additions),
	)
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

func appendWorkspaceDiscoveryRoots(
	values []providerapi.DirectoryRoot,
	additions []spec.DiscoveryRoot,
) []providerapi.DirectoryRoot {
	converted := make(
		[]providerapi.DirectoryRoot,
		0,
		len(additions),
	)
	for _, addition := range additions {
		converted = append(converted, providerapi.DirectoryRoot{
			Root:      addition.Root,
			Recursive: addition.Recursive,
			IncludePatterns: append(
				[]string(nil),
				addition.IncludePatterns...,
			),
		})
	}
	return appendWorkspaceDirectoryRoots(values, converted...)
}

func appendWorkspaceDirectoryRoots(
	values []providerapi.DirectoryRoot,
	additions ...providerapi.DirectoryRoot,
) []providerapi.DirectoryRoot {
	output := make([]providerapi.DirectoryRoot, len(values))
	for index, value := range values {
		output[index] = value.Clone()
	}

	for _, addition := range additions {
		addition = addition.Clone()
		merged := false
		for index := range output {
			if output[index].Root != addition.Root {
				continue
			}
			output[index].Recursive =
				output[index].Recursive || addition.Recursive
			output[index].IncludePatterns = mergeWorkspacePatterns(
				output[index].IncludePatterns,
				addition.IncludePatterns,
			)
			merged = true
			break
		}
		if !merged {
			output = append(output, addition)
		}
	}
	return output
}

func cloneWorkspaceDirectoryRoots(
	values []spec.DirectoryRoot,
) []spec.DirectoryRoot {
	output := make([]spec.DirectoryRoot, len(values))
	for index, value := range values {
		output[index] = spec.DirectoryRoot{
			Root:      value.Root,
			Recursive: value.Recursive,
			IncludePatterns: append(
				[]string(nil),
				value.IncludePatterns...,
			),
		}
	}
	return output
}

func mergeWorkspaceDiscoveryPreferences(
	left spec.DiscoveryPreferences,
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
	output.AdditionalLocators = appendWorkspaceUniqueLocators(
		left.AdditionalLocators,
		right.AdditionalLocators...,
	)

	roots := make(map[basespec.Locator]spec.DiscoveryRoot)
	for _, values := range [][]spec.DiscoveryRoot{
		left.AdditionalRoots,
		right.AdditionalRoots,
	} {
		for _, root := range values {
			current, found := roots[root.Root]
			if !found {
				current = spec.DiscoveryRoot{
					Root:      root.Root,
					Recursive: root.Recursive,
					IncludePatterns: append(
						[]string(nil),
						root.IncludePatterns...,
					),
				}
			} else {
				current.Recursive = current.Recursive || root.Recursive
				current.IncludePatterns = mergeWorkspacePatterns(
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

func mergeWorkspacePatterns(
	left []string,
	right []string,
) []string {
	if len(left) == 0 || len(right) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(left)+len(right))
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

func validateWorkspaceDiscoveryProfiles(
	value spec.DiscoveryProfiles,
) error {
	if err := validateWorkspaceDiscoveryProfile(value.Primary); err != nil {
		return err
	}
	return validateWorkspaceDiscoveryProfile(value.Attached)
}

func validateWorkspaceDiscoveryProfile(
	value spec.DiscoveryProfile,
) error {
	roots := make(
		[]spec.DiscoveryRoot,
		0,
		len(value.DirectoryRoots),
	)
	for _, root := range value.DirectoryRoots {
		roots = append(roots, spec.DiscoveryRoot{
			Root:      root.Root,
			Recursive: root.Recursive,
			IncludePatterns: append(
				[]string(nil),
				root.IncludePatterns...,
			),
		})
	}
	if err := spec.ValidateDiscoveryPreferences(
		spec.DiscoveryPreferences{
			AdditionalLocators: append(
				[]basespec.Locator(nil),
				value.ExplicitLocators...,
			),
			AdditionalRoots: roots,
		},
	); err != nil {
		return err
	}
	if value.ReadmeLocator == "" {
		return nil
	}
	return basespec.ValidateLocator(value.ReadmeLocator, false)
}
