package bundle

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	skillArtifact "github.com/flexigpt/flexigpt-app/internal/skill/store/artifact"
)

type skillCollectionBehavior struct{}

func NewCollectionBehavior() providerapi.CollectionBehavior {
	return skillCollectionBehavior{}
}

func (skillCollectionBehavior) CollectionKind() basespec.CollectionKind {
	return artifactbuiltin.SkillCollectionV1Kind
}

func (skillCollectionBehavior) Revision() string {
	return DiscoveryPolicyRevision
}

func (b skillCollectionBehavior) BuildDiscoveryPlan(
	ctx context.Context,
	collectionValue providerapi.Collection,
	attachments []providerapi.Attachment,
	sources []providerapi.Source,
) (providerapi.Plan, error) {
	if ctx == nil {
		return providerapi.Plan{}, fmt.Errorf(
			"%w: skill collection behavior context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return providerapi.Plan{}, err
	}
	if collectionValue.Kind != b.CollectionKind() {
		return providerapi.Plan{}, fmt.Errorf(
			"%w: skill collection behavior received collection kind %q",
			basespec.ErrInvalid,
			collectionValue.Kind,
		)
	}

	data, err := DecodeCollectionData(collectionValue.Data)
	if err != nil {
		return providerapi.Plan{}, err
	}

	sourcesByID := make(
		map[basespec.SourceID]providerapi.Source,
		len(sources),
	)
	for index, sourceValue := range sources {
		if sourceValue.RootID != collectionValue.RootID {
			return providerapi.Plan{}, fmt.Errorf(
				"%w: skill provider source %d belongs to another root",
				basespec.ErrInvalid,
				index,
			)
		}
		if _, duplicate := sourcesByID[sourceValue.ID]; duplicate {
			return providerapi.Plan{}, fmt.Errorf(
				"%w: skill provider received duplicate source %q",
				basespec.ErrInvalid,
				sourceValue.ID,
			)
		}
		sourcesByID[sourceValue.ID] = sourceValue
	}

	plans := make(
		[]providerapi.SourcePlan,
		0,
		len(attachments),
	)

	for index, attachment := range attachments {
		if attachment.RootID != collectionValue.RootID ||
			attachment.CollectionID != collectionValue.ID {
			return providerapi.Plan{}, fmt.Errorf(
				"%w: skill provider attachment %d belongs to another collection",
				basespec.ErrInvalid,
				index,
			)
		}
		if err := validateRole(attachment.Role); err != nil {
			return providerapi.Plan{}, err
		}

		sourceValue, found := sourcesByID[attachment.SourceID]
		if !found {
			return providerapi.Plan{}, fmt.Errorf(
				"%w: skill provider attachment source %q is unavailable",
				basespec.ErrAttachmentNotFound,
				attachment.SourceID,
			)
		}
		if err := validateRoleSourceKind(
			attachment.Role,
			sourceValue.Kind,
		); err != nil {
			return providerapi.Plan{}, err
		}

		attachmentData, err := DecodeAttachmentData(attachment.Data)
		if err != nil {
			return providerapi.Plan{}, err
		}
		expectedContentDigests, err := attachmentData.SourceExpectedContentDigests()
		if err != nil {
			return providerapi.Plan{}, err
		}

		if !attachment.Enabled || !sourceValue.Enabled {
			continue
		}

		plans = append(plans, providerapi.SourcePlan{
			SourceID: attachment.SourceID,
			DirectoryRoots: []providerapi.DirectoryRoot{{
				Root:      attachmentData.DiscoveryRoot,
				Recursive: true,
				IncludePatterns: []string{
					string(
						artifactbuiltin.AgentSkillDefinitionFileName,
					),
				},
			}},
			DecoderHints: []providerapi.DecoderHint{{
				Locator:   attachmentData.DiscoveryRoot,
				Recursive: true,
				DecoderIDs: []basespec.DecoderID{
					artifactbuiltin.AgentSkillDecoderID,
				},
			}},
			ExpectedContentDigests: expectedContentDigests,
			AllowedDecoderIDs: []basespec.DecoderID{
				artifactbuiltin.AgentSkillDecoderID,
			},
			Authoritative: true,
		}.Normalized())
	}

	if err := validateProviderBundleAttachmentTopology(
		data,
		attachments,
	); err != nil {
		return providerapi.Plan{}, err
	}

	plan := providerapi.Plan{
		Revision: b.Revision(),
		Sources:  plans,
	}
	if err := plan.Validate(); err != nil {
		return providerapi.Plan{}, err
	}
	return plan.Normalized(), nil
}

func (skillCollectionBehavior) DecideAutomaticAdoption(
	ctx context.Context,
	input providerapi.AdoptionInput,
) (providerapi.AdoptionDecision, error) {
	if ctx == nil {
		return providerapi.AdoptionDecision{}, fmt.Errorf(
			"%w: skill automatic adoption context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return providerapi.AdoptionDecision{}, err
	}
	if input.Collection.Kind != artifactbuiltin.SkillCollectionV1Kind {
		return providerapi.AdoptionDecision{}, fmt.Errorf(
			"%w: skill automatic adoption received collection kind %q",
			basespec.ErrInvalid,
			input.Collection.Kind,
		)
	}
	if input.Attachment.SourceID != input.Occurrence.SourceID {
		return providerapi.AdoptionDecision{}, fmt.Errorf(
			"%w: skill occurrence source does not match its attachment",
			basespec.ErrInvalid,
		)
	}

	if input.Occurrence.Kind != artifactbuiltin.AgentSkillArtifactKind {
		return providerapi.AdoptionDecision{}, nil
	}
	if !input.Attachment.Enabled {
		return providerapi.AdoptionDecision{}, nil
	}

	switch input.Attachment.Role {
	case RoleExternal, RoleLibrary:
	default:
		return providerapi.AdoptionDecision{}, nil
	}

	if err := skillArtifact.ValidateDefinition(input.Definition); err != nil {
		return providerapi.AdoptionDecision{
			Diagnostics: []providerapi.Diagnostic{{
				Severity: providerapi.DiagnosticError,
				Code:     "skill.bundle.definition-invalid",
				Message:  providerapi.BoundedDiagnosticMessage(err.Error()),
				Location: &providerapi.DiagnosticLocation{
					Locator: input.Occurrence.Locator,
					SubresourceLocator: input.Occurrence.
						SubresourceLocator,
				},
			}},
		}, nil
	}

	name := input.Definition.DisplayName
	if name == "" {
		name = string(input.Definition.LogicalName)
	}

	return providerapi.AdoptionDecision{
		Adopt:   true,
		Name:    name,
		Enabled: true,
		Data:    emptyArtifactData(),
	}, nil
}

func validateProviderBundleAttachmentTopology(
	data CollectionData,
	attachments []providerapi.Attachment,
) error {
	var (
		managedAttachmentCount int
		managedAttachmentID    basespec.SourceID
		builtInAttachmentCount int
	)

	for _, attachment := range attachments {
		switch attachment.Role {
		case artifactbuiltin.ManagedAttachmentRole:
			managedAttachmentCount++
			managedAttachmentID = attachment.SourceID

		case artifactbuiltin.BuiltInAttachmentRole:
			builtInAttachmentCount++
		}
	}

	if managedAttachmentCount > 1 {
		return fmt.Errorf(
			"%w: skill bundle has multiple managed attachments",
			basespec.ErrInvalid,
		)
	}
	if builtInAttachmentCount > 1 {
		return fmt.Errorf(
			"%w: skill bundle has multiple built-in attachments",
			basespec.ErrInvalid,
		)
	}

	if data.ManagedSourceID == "" {
		return nil
	}
	if managedAttachmentCount != 1 ||
		managedAttachmentID != data.ManagedSourceID {
		return fmt.Errorf(
			"%w: bundle-owned managed Source %q is not its sole managed attachment",
			basespec.ErrInvalid,
			data.ManagedSourceID,
		)
	}

	return nil
}
