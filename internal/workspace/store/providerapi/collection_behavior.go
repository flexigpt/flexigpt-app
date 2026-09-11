package providerapi

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
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/workspacecollectionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/domain/artifactadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/domain/attachmentdata"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/domain/collectiondata"
)

const (
	defaultAutomaticArtifactName      = "artifact"
	automaticArtifactNameSeparator    = "-"
	automaticArtifactNameDigestLength = 12
)

type workspaceCollectionBehavior struct {
	workspaceRootID root.RootID
	revision        string
	supports        map[artifact.ArtifactKind]workspaceDomain.ArtifactSupport
	decoderIDs      []basespec.DecoderID
	profiles        workspaceDomain.DiscoveryProfiles
}

type workspacePlanningTopology struct {
	attachments     []providerapi.Attachment
	attachmentData  map[source.SourceID]workspaceDomain.AttachmentData
	sources         map[source.SourceID]providerapi.Source
	primarySourceID source.SourceID
}

type workspaceDescriptorObservation struct {
	Preferences            workspaceDomain.DiscoveryPreferences
	SourceID               source.SourceID
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
		map[artifact.ArtifactKind]workspaceDomain.ArtifactSupport,
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

func (*workspaceCollectionBehavior) CollectionKind() collection.CollectionKind {
	return collection.CollectionKind(
		workspacecollectionv1.WorkspaceCollectionKind,
	)
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
			workspaceDomain.ErrInvalidWorkspace,
		)
	}
	if err := ctx.Err(); err != nil {
		return providerapi.Plan{}, err
	}
	if documents == nil {
		return providerapi.Plan{}, fmt.Errorf(
			"%w: Workspace provider planning document reader is required",
			workspaceDomain.ErrInvalidWorkspace,
		)
	}
	if collectionValue.RootID != b.workspaceRootID {
		return providerapi.Plan{}, fmt.Errorf(
			"%w: Workspace Collection belongs to Root %q, expected %q",
			workspaceDomain.ErrInvalidWorkspace,
			collectionValue.RootID,
			b.workspaceRootID,
		)
	}
	if collectionValue.Kind != b.CollectionKind() {
		return providerapi.Plan{}, fmt.Errorf(
			"%w: Workspace provider received Collection kind %q",
			workspaceDomain.ErrInvalidWorkspace,
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
			workspaceDomain.ErrInvalidWorkspace,
		)
	}
	if err := ctx.Err(); err != nil {
		return providerapi.AdoptionDecision{}, err
	}
	if input.Collection.RootID != b.workspaceRootID ||
		input.Collection.Kind != b.CollectionKind() {
		return providerapi.AdoptionDecision{}, fmt.Errorf(
			"%w: Workspace automatic adoption received another Collection",
			workspaceDomain.ErrInvalidWorkspace,
		)
	}
	if input.Attachment.RootID != input.Collection.RootID ||
		input.Attachment.CollectionID != input.Collection.ID ||
		input.Attachment.SourceID != input.Occurrence.SourceID ||
		input.Occurrence.RootID != input.Collection.RootID ||
		input.Occurrence.CollectionID != input.Collection.ID {
		return providerapi.AdoptionDecision{}, fmt.Errorf(
			"%w: Workspace automatic adoption input identities do not match",
			workspaceDomain.ErrInvalidWorkspace,
		)
	}

	support, supported := b.supports[input.Occurrence.Kind]
	if !supported || !input.Attachment.Enabled {
		return providerapi.AdoptionDecision{}, nil
	}
	if input.Definition.Kind != input.Occurrence.Kind {
		return workspaceAdoptionDiagnostic(
			input.Occurrence,
			workspaceDomain.DiagnosticCodeArtifactKindMismatch,
			fmt.Sprintf(
				"definition kind %q does not match occurrence kind %q",
				input.Definition.Kind,
				input.Occurrence.Kind,
			),
		), nil
	}
	if input.Definition.SchemaID != support.SchemaID ||
		input.Definition.SchemaVersion != support.SchemaVersion {
		return workspaceAdoptionDiagnostic(
			input.Occurrence,
			workspaceDomain.DiagnosticCodeArtifactSchemaUnsupported,
			fmt.Sprintf(
				"definition schema %q/%q is not supported for kind %q",
				input.Definition.SchemaID,
				input.Definition.SchemaVersion,
				input.Definition.Kind,
			),
		), nil
	}

	data, err := artifactadapter.EncodeArtifactData(
		workspaceDomain.ArtifactData{},
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
	preferences workspaceDomain.DiscoveryPreferences,
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

		profilePreferences := workspaceDomain.DiscoveryPreferences{
			AdditionalLocators: append(
				[]basespec.Locator(nil),
				profile.ExplicitLocators...,
			),
		}
		for _, root := range profile.DirectoryRoots {
			profilePreferences.AdditionalRoots = append(
				profilePreferences.AdditionalRoots,
				workspaceDomain.DiscoveryRoot{
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
						workspaceDomain.ErrInvalidWorkspace,
						attachment.Role,
					)
				}
				sourcePlan.DirectoryRoots[0].Recursive = *attachmentData.Recursive
			}
			if attachmentData.Authoritative != nil {
				sourcePlan.Authoritative = *attachmentData.Authoritative
			}
		}

		plans = append(plans, sourcePlan)
	}

	sort.Slice(plans, func(left, right int) bool {
		return plans[left].SourceID < plans[right].SourceID
	})

	plan := providerapi.Plan{
		Revision: b.revision,
		Sources:  plans,
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
			Severity: diagnostic.SeverityError,
			Code:     code,
			Message:  diagnostic.BoundedMessage(message),
			Location: &diagnostic.Location{
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
			map[source.SourceID]workspaceDomain.AttachmentData,
			len(attachments),
		),
		sources: make(
			map[source.SourceID]providerapi.Source,
			len(sources),
		),
	}
	attachmentsBySource := make(
		map[source.SourceID]struct{},
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
				workspaceDomain.ErrInvalidWorkspace,
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
				workspaceDomain.ErrInvalidWorkspace,
				sourceValue.ID,
			)
		}
		if _, attached := attachmentsBySource[sourceValue.ID]; !attached {
			return workspacePlanningTopology{}, fmt.Errorf(
				"%w: Workspace provider received unattached Source %q",
				workspaceDomain.ErrInvalidWorkspace,
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
				workspaceDomain.ErrInvalidWorkspace,
				attachment.SourceID,
			)
		}
		if attachment.Enabled && !sourceValue.Enabled {
			return workspacePlanningTopology{}, fmt.Errorf(
				"%w: enabled Workspace attachment %q uses a disabled Source",
				workspaceDomain.ErrInvalidWorkspace,
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
				workspaceDomain.ErrInvalidWorkspace,
			)
		}
		if sourceValue.Kind != operation.RequiredSourceKind {
			return workspacePlanningTopology{}, fmt.Errorf(
				"%w: Workspace primary Source must have kind %q",
				workspaceDomain.ErrInvalidWorkspace,
				operation.RequiredSourceKind,
			)
		}
		topology.primarySourceID = sourceValue.ID
	}

	if primaryCount > 1 {
		return workspacePlanningTopology{}, fmt.Errorf(
			"%w: Workspace cannot have multiple primary attachments",
			workspaceDomain.ErrInvalidWorkspace,
		)
	}

	sort.Slice(topology.attachments, func(left, right int) bool {
		return topology.attachments[left].SourceID <
			topology.attachments[right].SourceID
	})
	return topology, nil
}

func validateWorkspaceProviderAttachment(
	collectionValue providerapi.Collection,
	value providerapi.Attachment,
) error {
	// Artifact Store validates generic provider input fields before this
	// behavior is invoked. This function validates Workspace relationships.
	if value.RootID != collectionValue.RootID ||
		value.CollectionID != collectionValue.ID {
		return fmt.Errorf(
			"%w: Workspace attachment belongs to another Collection",
			workspaceDomain.ErrInvalidWorkspace,
		)
	}
	if _, supported := attachmentdata.AttachmentOperationFor(value.Role); !supported {
		return fmt.Errorf(
			"%w: unsupported Workspace attachment role %q",
			workspaceDomain.ErrInvalidWorkspace,
			value.Role,
		)
	}
	if value.Revision == 0 {
		return fmt.Errorf(
			"%w: Workspace attachment revision is required",
			workspaceDomain.ErrInvalidWorkspace,
		)
	}
	return nil
}

func validateWorkspaceProviderSource(
	collectionValue providerapi.Collection,
	value providerapi.Source,
) error {
	// Artifact Store validates generic provider input fields before this
	// behavior is invoked. This function validates Workspace relationships.
	if value.RootID != collectionValue.RootID {
		return fmt.Errorf(
			"%w: Workspace Source belongs to another Root",
			workspaceDomain.ErrInvalidWorkspace,
		)
	}
	return nil
}

func readWorkspaceDescriptor(
	ctx context.Context,
	primarySourceID source.SourceID,
	reader providerapi.PlanningDocumentReader,
) (workspaceDescriptorObservation, error) {
	if primarySourceID == "" {
		return workspaceDescriptorObservation{}, nil
	}

	request := providerapi.PlanningDocumentRequest{
		SourceID:       primarySourceID,
		Locator:        artifactbuiltin.WorkspaceDescriptorFileName,
		ExpectedSchema: workspacecollectionv1.WorkspaceCollectionSchemaKey,
	}
	result, err := reader.ReadCanonicalDocument(ctx, request)
	if err != nil {
		return workspaceDescriptorObservation{}, fmt.Errorf(
			"%w: read Workspace descriptor: %w",
			workspaceDomain.ErrWorkspaceDefinitionInvalid,
			err,
		)
	}
	if result.SourceID != primarySourceID {
		return workspaceDescriptorObservation{}, fmt.Errorf(
			"%w: Workspace descriptor reader returned another Source",
			workspaceDomain.ErrWorkspaceDefinitionInvalid,
		)
	}
	if err := result.Validate(); err != nil {
		return workspaceDescriptorObservation{}, fmt.Errorf(
			"%w: Workspace descriptor reader returned invalid state: %w",
			workspaceDomain.ErrWorkspaceDefinitionInvalid,
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
	if parsed.Key != workspacecollectionv1.WorkspaceCollectionSchemaKey {
		return workspaceDescriptorObservation{}, fmt.Errorf(
			"%w: Workspace descriptor schema does not identify %q",
			workspaceDomain.ErrWorkspaceDefinitionInvalid,
			workspacecollectionv1.WorkspaceCollectionKind,
		)
	}

	descriptor, err := workspacecollectionv1.DecodeWorkspaceCollectionJSON(
		parsed.Raw,
	)
	if err != nil {
		return workspaceDescriptorObservation{}, fmt.Errorf(
			"%w: decode canonical Workspace descriptor: %w",
			workspaceDomain.ErrWorkspaceDefinitionInvalid,
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

	body := descriptor.Body

	base, err := workspaceDescriptorBaseLocator(
		artifactbuiltin.WorkspaceDescriptorFileName,
	)
	if err != nil {
		return workspaceDescriptorObservation{}, fmt.Errorf(
			"%w: resolve Workspace descriptor base: %w",
			workspaceDomain.ErrWorkspaceDefinitionInvalid,
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
			workspaceDomain.ErrWorkspaceDefinitionInvalid,
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
					workspaceDomain.ErrWorkspaceDefinitionInvalid,
					index,
					err,
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
						workspaceDomain.ErrWorkspaceDefinitionInvalid,
						resolvedLocator,
					)
				}
				expectedContentDigests[resolvedLocator] = digest
			}

		case member.URI != "":
			return workspaceDescriptorObservation{}, fmt.Errorf(
				"%w: Workspace descriptor member %d requires an external URI resolver: %w",
				workspaceDomain.ErrWorkspaceDefinitionInvalid,
				index,
				basespec.ErrUnsupported,
			)

		default:
			return workspaceDescriptorObservation{}, fmt.Errorf(
				"%w: Workspace descriptor member %d requires unsupported embedded content handling: %w",
				workspaceDomain.ErrWorkspaceDefinitionInvalid,
				index,
				basespec.ErrUnsupported,
			)
		}
	}

	if err := workspaceDomain.ValidateDiscoveryPreferences(preferences); err != nil {
		return workspaceDescriptorObservation{}, fmt.Errorf(
			"%w: %w",
			workspaceDomain.ErrWorkspaceDefinitionInvalid,
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
	input workspacecollectionv1.WorkspaceCollectionDiscovery,
	base basespec.Locator,
) (workspaceDomain.DiscoveryPreferences, error) {
	output := workspaceDomain.DiscoveryPreferences{
		IncludeReadme: input.IncludeReadme,
	}

	for index, locator := range input.AdditionalLocators {
		resolved, err := resolveWorkspaceRelativeLocator(
			base,
			basespec.Locator(locator),
			false,
		)
		if err != nil {
			return workspaceDomain.DiscoveryPreferences{}, fmt.Errorf(
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
			return workspaceDomain.DiscoveryPreferences{}, fmt.Errorf(
				"additionalRoots[%d]: %w",
				index,
				err,
			)
		}
		output.AdditionalRoots = append(
			output.AdditionalRoots,
			workspaceDomain.DiscoveryRoot{
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
	if err := document.Validate(false); err != nil {
		return "", fmt.Errorf("workspace descriptor locator: %w", err)
	}

	base := basespec.Locator(path.Dir(string(document)))
	if err := base.Validate(true); err != nil {
		return "", fmt.Errorf("workspace descriptor base locator: %w", err)
	}
	return base, nil
}

func resolveWorkspaceRelativeLocator(
	base basespec.Locator,
	relative basespec.Locator,
	allowRelativeRoot bool,
) (basespec.Locator, error) {
	if err := base.ValidatePortable(true); err != nil {
		return "", fmt.Errorf("portable base locator: %w", err)
	}
	if err := relative.ValidatePortable(allowRelativeRoot); err != nil {
		return "", fmt.Errorf("portable relative locator: %w", err)
	}

	switch {
	case base == ".":
		return relative, nil
	case relative == ".":
		return base, nil
	default:
		return basespec.Locator(
			string(base) + "/" + string(relative),
		), nil
	}
}

func workspaceProviderDirectoryRoots(
	input []workspaceDomain.DirectoryRoot,
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
	preferences workspaceDomain.DiscoveryPreferences,
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
	value workspaceDomain.DiscoveryProfiles,
) workspaceDomain.DiscoveryProfiles {
	return workspaceDomain.DiscoveryProfiles{
		Primary: workspaceDomain.DiscoveryProfile{
			ExplicitLocators: append(
				[]basespec.Locator(nil),
				value.Primary.ExplicitLocators...,
			),
			ReadmeLocator: value.Primary.ReadmeLocator,
			DirectoryRoots: cloneWorkspaceDirectoryRoots(
				value.Primary.DirectoryRoots,
			),
		},
		Attached: workspaceDomain.DiscoveryProfile{
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
	base workspaceDomain.DiscoveryProfile,
	additions workspaceDomain.DiscoveryProfile,
) workspaceDomain.DiscoveryProfile {
	output := workspaceDomain.DiscoveryProfile{
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
	value workspaceDomain.DiscoveryProfiles,
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
	additions []workspaceDomain.DiscoveryRoot,
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
	values []workspaceDomain.DirectoryRoot,
) []workspaceDomain.DirectoryRoot {
	output := make([]workspaceDomain.DirectoryRoot, len(values))
	for index, value := range values {
		output[index] = workspaceDomain.DirectoryRoot{
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
	left workspaceDomain.DiscoveryPreferences,
	right workspaceDomain.DiscoveryPreferences,
) (workspaceDomain.DiscoveryPreferences, error) {
	if err := workspaceDomain.ValidateDiscoveryPreferences(left); err != nil {
		return workspaceDomain.DiscoveryPreferences{}, err
	}
	if err := workspaceDomain.ValidateDiscoveryPreferences(right); err != nil {
		return workspaceDomain.DiscoveryPreferences{}, err
	}

	output := workspaceDomain.DiscoveryPreferences{
		IncludeReadme: left.IncludeReadme || right.IncludeReadme,
	}
	output.AdditionalLocators = appendWorkspaceUniqueLocators(
		left.AdditionalLocators,
		right.AdditionalLocators...,
	)

	roots := make(map[basespec.Locator]workspaceDomain.DiscoveryRoot)
	for _, values := range [][]workspaceDomain.DiscoveryRoot{
		left.AdditionalRoots,
		right.AdditionalRoots,
	} {
		for _, root := range values {
			current, found := roots[root.Root]
			if !found {
				current = workspaceDomain.DiscoveryRoot{
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
	return output, workspaceDomain.ValidateDiscoveryPreferences(output)
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
	value workspaceDomain.DiscoveryProfiles,
) error {
	if err := validateWorkspaceDiscoveryProfile(value.Primary); err != nil {
		return err
	}
	return validateWorkspaceDiscoveryProfile(value.Attached)
}

func validateWorkspaceDiscoveryProfile(
	value workspaceDomain.DiscoveryProfile,
) error {
	roots := make(
		[]workspaceDomain.DiscoveryRoot,
		0,
		len(value.DirectoryRoots),
	)
	for _, root := range value.DirectoryRoots {
		roots = append(roots, workspaceDomain.DiscoveryRoot{
			Root:      root.Root,
			Recursive: root.Recursive,
			IncludePatterns: append(
				[]string(nil),
				root.IncludePatterns...,
			),
		})
	}
	if err := workspaceDomain.ValidateDiscoveryPreferences(
		workspaceDomain.DiscoveryPreferences{
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
	return value.ReadmeLocator.Validate(false)
}
