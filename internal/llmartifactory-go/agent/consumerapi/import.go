package consumerapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/managedfs"
	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	managepackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	agentv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/contract/v1"
	agentDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/domain"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	mcpv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/contract/v1"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

const managedAgentPreparedTTL = 10 * time.Minute

type agentImportDestinationState struct {
	value            AgentImportDestination
	plugin           plugin.PluginView
	sourceGeneration string
}

type preparedAgentImport struct {
	ProfileID string    `json:"profileID"`
	ExpiresAt time.Time `json:"expiresAt"`

	SourceDigest     cryptoutil.Digest `json:"sourceDigest"`
	DefinitionDigest cryptoutil.Digest `json:"definitionDigest"`

	CanonicalDeclaration json.RawMessage `json:"canonicalDeclaration"`

	RootID   rootModel.RootID     `json:"rootID"`
	SourceID sourceModel.SourceID `json:"sourceID"`

	Plugin                     artifactModel.ArtifactRef `json:"plugin"`
	ExpectedCollectionRevision uint64                    `json:"expectedCollectionRevision"`
	ExpectedSourceGeneration   string                    `json:"expectedSourceGeneration"`

	Address      managedpackageModel.ManagedPackageAddress `json:"address"`
	AgentLocator spec.Locator                              `json:"agentLocator"`

	RestoredMemberships       []AgentRestoredMembership `json:"restoredMemberships"`
	MCPSetupDescriptors       []AgentMCPSetupDescriptor `json:"mcpSetupDescriptors"`
	RequiredConfirmationCodes []string                  `json:"requiredConfirmationCodes"`
}

type plannedImportIdentity struct {
	OccurrencePath string
	Type           declaration.Type
	Name           spec.LogicalName
	LogicalVersion spec.LogicalVersion
	Digest         cryptoutil.Digest
}

func (a *API) ListAgentImportDestinations(
	ctx context.Context,
	rootID rootModel.RootID,
) ([]AgentImportDestination, error) {
	if a == nil || a.plugins == nil {
		return nil, spec.ErrClosed
	}
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	if a.protection.IsProtectedRoot(rootID) {
		return nil, nil
	}

	values, err := a.ListAgentCollections(ctx, rootID)
	if err != nil {
		return nil, err
	}

	output := make([]AgentImportDestination, 0, len(values))
	for _, value := range values {
		if !value.Editable || value.BuiltIn || !value.Enabled {
			continue
		}
		output = append(output, AgentImportDestination{
			RootID:                rootID,
			SourceID:              value.SourceID,
			Plugin:                value.Ref,
			CollectionRevision:    value.Revision,
			CollectionName:        value.Name,
			CollectionDisplayName: value.DisplayName,
			Baseline:              value.Baseline,
			Enabled:               value.Enabled,
		})
	}

	sort.Slice(output, func(left, right int) bool {
		return output[left].CollectionName < output[right].CollectionName
	})
	return output, nil
}

func (a *API) PreviewAgentImport(
	ctx context.Context,
	request AgentImportPreviewRequest,
) (AgentImportPreview, error) {
	if a == nil || a.managedAgentProfile == nil ||
		a.importSigner == nil {
		return AgentImportPreview{}, spec.ErrClosed
	}
	if ctx == nil {
		return AgentImportPreview{}, fmt.Errorf(
			"%w: Agent import preview context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return AgentImportPreview{}, err
	}
	if request.ExpectedCollectionRevision == 0 {
		return AgentImportPreview{}, fmt.Errorf(
			"%w: expected Agent Plugin revision is required",
			spec.ErrInvalid,
		)
	}
	inputFormat, err := managedAgentImportFormatForPath(request.Path)
	if err != nil {
		return AgentImportPreview{}, err
	}
	if request.ExpectedSourceDigest != "" {
		if err := cryptoutil.ValidateDigest(
			request.ExpectedSourceDigest,
		); err != nil {
			return AgentImportPreview{}, err
		}
	}

	destination, err := a.agentImportDestination(
		ctx,
		request.Plugin,
		request.ExpectedCollectionRevision,
	)
	if err != nil {
		return AgentImportPreview{}, err
	}

	preview := AgentImportPreview{
		Destination: destination.value,
	}

	sourceBytes, err := llmtoolsutil.ReadPortableTextFile(
		ctx,
		request.Path,
		spec.MaxDefinitionBytes,
	)
	if err != nil {
		return preview, err
	}
	preview.SourceDigest = cryptoutil.DigestBytes(sourceBytes)
	if request.ExpectedSourceDigest != "" &&
		request.ExpectedSourceDigest != preview.SourceDigest {
		preview.Issues = append(preview.Issues, AgentImportIssue{
			Code:     "agent.import.source-digest-mismatch",
			Severity: AgentImportIssueError,
			Message:  "selected file changed after its expected source digest was calculated",
		})
		return preview, nil
	}

	canonical, err := canonicalManagedAgentImportDocument(
		inputFormat,
		sourceBytes,
	)
	if err != nil {
		preview.Issues = append(preview.Issues, importIssue(
			inputFormat.invalidDocumentIssueCode(),
			"",
			err,
		))
		return preview, nil
	}

	canonical, err = a.managedAgentProfile.Validate(canonical)
	if err != nil {
		preview.Issues = append(preview.Issues, importIssue(
			"agent.import.strict-profile-invalid",
			"",
			err,
		))
		return preview, nil
	}

	entry, err := declaration.DecodeCanonicalEntryJSON(canonical)
	if err != nil {
		preview.Issues = append(preview.Issues, importIssue(
			"agent.import.declaration-invalid",
			"",
			err,
		))
		return preview, nil
	}

	entry, err = agentDomain.NormalizeManagedAgentImport(entry)
	if err != nil {
		return previewValidationError(
			preview,
			"agent.import.managed-normalization-invalid",
			"",
			err,
		)
	}
	canonical, err = entry.CanonicalJSON()
	if err != nil {
		return previewValidationError(
			preview,
			"agent.import.managed-normalization-invalid",
			"",
			err,
		)
	}
	canonical, err = a.managedAgentProfile.Validate(canonical)
	if err != nil {
		preview.Issues = append(preview.Issues, importIssue(
			"agent.import.strict-profile-invalid",
			"",
			err,
		))
		return preview, nil
	}

	normalizedYAML, err := yamlutil.CanonicalObjectYAML(
		canonical,
		spec.MaxDefinitionBytes,
	)
	if err != nil {
		return previewValidationError(
			preview,
			"agent.import.normalized-yaml-invalid",
			"",
			err,
		)
	}
	preview.NormalizedYAML = string(normalizedYAML)

	entry, err = declaration.DecodeCanonicalEntryJSON(canonical)
	if err != nil {
		return previewValidationError(
			preview,
			"agent.import.declaration-invalid",
			"",
			err,
		)
	}
	document, err := agentv1.DecodeAgentEntry(entry)
	if err != nil {
		preview.Issues = append(preview.Issues, importIssue(
			"agent.import.agent-invalid",
			"",
			err,
		))
		return preview, nil
	}

	admission, err := agentDomain.ValidateManagedAgentImport(document)
	if err != nil {
		return previewValidationError(
			preview,
			"agent.import.managed-content-invalid",
			"",
			err,
		)
	}
	preview.Issues = append(
		preview.Issues,
		importAdmissionIssues(admission.Issues)...,
	)
	preview.MCPSetupDescriptors = append(
		preview.MCPSetupDescriptors,
		importMCPSetupDescriptors(admission.MCPSetupDescriptors)...,
	)

	rawAgent, rootDefinition, err := agentDomain.ManagedAgentEntryPayload(entry)
	if err != nil {
		return previewValidationError(
			preview,
			"agent.import.managed-content-invalid",
			"",
			err,
		)
	}
	preview.DefinitionDigest = rootDefinition.Digest

	address, err := agentDomain.ManagedPackageAddressForAgent(
		rootDefinition.LogicalName,
	)
	if err != nil {
		return previewValidationError(
			preview,
			"agent.import.package-invalid",
			"",
			err,
		)
	}
	agentLocator, err := agentDomain.ManagedPackageLocatorForAgent(address)
	if err != nil {
		return previewValidationError(
			preview,
			"agent.import.package-invalid",
			"",
			err,
		)
	}

	identities, artifacts, err := a.plannedImportArtifacts(entry)
	if err != nil {
		return previewValidationError(
			preview,
			"agent.import.managed-content-invalid",
			"",
			err,
		)
	}
	preview.ProjectedArtifacts = artifacts
	for _, value := range artifacts {
		if value.OccurrencePath != "" {
			continue
		}
		copyValue := value
		preview.Agent = &copyValue
		break
	}

	conflicts, err := a.agentImportIdentityConflicts(
		ctx,
		destination.value.RootID,
		identities,
	)
	if err != nil {
		return preview, err
	}
	preview.Conflicts = append(preview.Conflicts, conflicts...)
	for _, conflict := range conflicts {
		preview.Issues = append(preview.Issues, AgentImportIssue{
			Code:     conflict.Code,
			Severity: AgentImportIssueError,
			Path:     conflict.Path,
			Message:  conflict.Message,
		})
	}

	packageConflict, err := a.managedAgentPackageConflict(
		ctx,
		destination.value.RootID,
		destination.value.SourceID,
		address,
	)
	if err != nil {
		return preview, err
	}
	if packageConflict != nil {
		preview.Conflicts = append(preview.Conflicts, *packageConflict)
		preview.Issues = append(preview.Issues, AgentImportIssue{
			Code:     packageConflict.Code,
			Severity: AgentImportIssueError,
			Path:     packageConflict.Path,
			Message:  packageConflict.Message,
		})
	}

	restored, membershipConflicts, err := a.analyzeAgentImportMembership(
		ctx,
		destination.plugin,
		rootDefinition.LogicalName,
		agentLocator,
	)
	if err != nil {
		return preview, err
	}
	preview.RestoredMemberships = append(
		preview.RestoredMemberships,
		restored...,
	)
	preview.Conflicts = append(
		preview.Conflicts,
		membershipConflicts...,
	)
	for _, conflict := range membershipConflicts {
		preview.Issues = append(preview.Issues, AgentImportIssue{
			Code:     conflict.Code,
			Severity: AgentImportIssueError,
			Path:     conflict.Path,
			Message:  conflict.Message,
		})
	}

	relationships, dependencySetups, dependencyIssues, err := a.preflightManagedAgentDependencies(
		ctx,
		destination.value.RootID,
		document,
	)
	if err != nil {
		return preview, err
	}
	preview.Relationships = relationships
	preview.Issues = append(preview.Issues, dependencyIssues...)
	preview.MCPSetupDescriptors = mergeMCPSetupDescriptors(
		preview.MCPSetupDescriptors,
		dependencySetups,
	)

	requiredCodes := confirmationCodes(preview.Issues)
	preview.RequiredConfirmationCodes = requiredCodes
	preview.RequiresConfirmation = len(requiredCodes) != 0
	if hasImportErrors(preview.Issues) {
		return preview, nil
	}

	plan := preparedAgentImport{
		ProfileID:                  agentv1.ManagedAgentImportProfileID,
		ExpiresAt:                  time.Now().UTC().Add(managedAgentPreparedTTL),
		SourceDigest:               preview.SourceDigest,
		DefinitionDigest:           rootDefinition.Digest,
		CanonicalDeclaration:       append(json.RawMessage(nil), rawAgent...),
		RootID:                     destination.value.RootID,
		SourceID:                   destination.value.SourceID,
		Plugin:                     request.Plugin,
		ExpectedCollectionRevision: request.ExpectedCollectionRevision,
		ExpectedSourceGeneration:   destination.sourceGeneration,
		Address:                    address,
		AgentLocator:               agentLocator,
		RestoredMemberships:        preview.RestoredMemberships,
		MCPSetupDescriptors:        preview.MCPSetupDescriptors,
		RequiredConfirmationCodes:  requiredCodes,
	}

	sealed, err := a.importSigner.Seal(plan)
	if err != nil {
		return preview, err
	}
	preview.Prepared = sealed.Envelope
	preview.PreparedFingerprint = sealed.Fingerprint
	preview.ExpiresAt = plan.ExpiresAt
	preview.CanImport = true
	return preview, nil
}

func (a *API) CommitAgentImport(
	ctx context.Context,
	request AgentImportCommitRequest,
) (AgentImportCommitResult, error) {
	if a == nil || a.managedAgentProfile == nil ||
		a.importSigner == nil {
		return AgentImportCommitResult{}, spec.ErrClosed
	}
	if ctx == nil {
		return AgentImportCommitResult{}, fmt.Errorf(
			"%w: Agent import commit context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return AgentImportCommitResult{}, err
	}
	if request.Prepared == "" {
		return AgentImportCommitResult{}, fmt.Errorf(
			"%w: prepared Agent import is required",
			spec.ErrInvalid,
		)
	}

	var plan preparedAgentImport
	fingerprint, err := a.importSigner.Open(
		request.Prepared,
		request.PreparedFingerprint,
		&plan,
	)
	if err != nil {
		return AgentImportCommitResult{}, err
	}
	if !time.Now().UTC().Before(plan.ExpiresAt) {
		return AgentImportCommitResult{}, fmt.Errorf(
			"%w: prepared Agent import has expired",
			spec.ErrConflict,
		)
	}
	if plan.ProfileID != agentv1.ManagedAgentImportProfileID {
		return AgentImportCommitResult{}, fmt.Errorf(
			"%w: prepared Agent import profile is unsupported",
			spec.ErrInvalid,
		)
	}
	if !sameCodes(
		plan.RequiredConfirmationCodes,
		request.AcceptedConfirmationCodes,
	) {
		return AgentImportCommitResult{}, fmt.Errorf(
			"%w: accepted confirmation codes differ from preview",
			spec.ErrConflict,
		)
	}

	destination, err := a.agentImportDestination(
		ctx,
		plan.Plugin,
		plan.ExpectedCollectionRevision,
	)
	if err != nil {
		return AgentImportCommitResult{}, err
	}
	if destination.value.RootID != plan.RootID ||
		destination.value.SourceID != plan.SourceID ||
		destination.sourceGeneration != plan.ExpectedSourceGeneration {
		return AgentImportCommitResult{}, fmt.Errorf(
			"%w: prepared Agent import Source witness is stale",
			spec.ErrConflict,
		)
	}

	canonical, err := a.managedAgentProfile.Validate(plan.CanonicalDeclaration)
	if err != nil {
		return AgentImportCommitResult{}, err
	}
	entry, err := declaration.DecodeCanonicalEntryJSON(canonical)
	if err != nil {
		return AgentImportCommitResult{}, err
	}
	normalizedEntry, err := agentDomain.NormalizeManagedAgentImport(entry)
	if err != nil {
		return AgentImportCommitResult{}, err
	}
	normalizedCanonical, err := normalizedEntry.CanonicalJSON()
	if err != nil {
		return AgentImportCommitResult{}, err
	}
	if !bytes.Equal(canonical, normalizedCanonical) {
		return AgentImportCommitResult{}, fmt.Errorf(
			"%w: prepared Agent import is not normalized",
			spec.ErrConflict,
		)
	}
	canonical, err = a.managedAgentProfile.Validate(normalizedCanonical)
	if err != nil {
		return AgentImportCommitResult{}, err
	}
	entry, err = declaration.DecodeCanonicalEntryJSON(canonical)
	if err != nil {
		return AgentImportCommitResult{}, err
	}
	document, err := agentv1.DecodeAgentEntry(entry)
	if err != nil {
		return AgentImportCommitResult{}, err
	}
	admission, err := agentDomain.ValidateManagedAgentImport(document)
	if err != nil {
		return AgentImportCommitResult{}, err
	}
	if admission.HasErrors() {
		return AgentImportCommitResult{}, fmt.Errorf(
			"%w: prepared Agent import no longer satisfies managed admission",
			spec.ErrInvalid,
		)
	}

	rawAgent, rootDefinition, err := agentDomain.ManagedAgentEntryPayload(entry)
	if err != nil {
		return AgentImportCommitResult{}, err
	}
	if rootDefinition.Digest != plan.DefinitionDigest {
		return AgentImportCommitResult{}, fmt.Errorf(
			"%w: prepared Agent Definition digest changed",
			spec.ErrConflict,
		)
	}
	address, err := agentDomain.ManagedPackageAddressForAgent(
		rootDefinition.LogicalName,
	)
	if err != nil {
		return AgentImportCommitResult{}, err
	}
	if address != plan.Address {
		return AgentImportCommitResult{}, fmt.Errorf(
			"%w: prepared Agent package address changed",
			spec.ErrConflict,
		)
	}
	agentLocator, err := agentDomain.ManagedPackageLocatorForAgent(address)
	if err != nil {
		return AgentImportCommitResult{}, err
	}
	if agentLocator != plan.AgentLocator {
		return AgentImportCommitResult{}, fmt.Errorf(
			"%w: prepared Agent package locator changed",
			spec.ErrConflict,
		)
	}

	identities, _, err := a.plannedImportArtifacts(entry)
	if err != nil {
		return AgentImportCommitResult{}, err
	}
	conflicts, err := a.agentImportIdentityConflicts(
		ctx,
		plan.RootID,
		identities,
	)
	if err != nil {
		return AgentImportCommitResult{}, err
	}
	if len(conflicts) != 0 {
		return AgentImportCommitResult{}, fmt.Errorf(
			"%w: prepared Agent import now conflicts with existing Artifacts",
			spec.ErrConflict,
		)
	}

	packageConflict, err := a.managedAgentPackageConflict(
		ctx,
		plan.RootID,
		plan.SourceID,
		plan.Address,
	)
	if err != nil {
		return AgentImportCommitResult{}, err
	}
	if packageConflict != nil {
		return AgentImportCommitResult{}, fmt.Errorf(
			"%w: %s",
			spec.ErrConflict,
			packageConflict.Message,
		)
	}

	membership, err := a.plugins.EnsureMemberForCollectionSource(
		ctx,
		plugin.EnsureMemberForCollectionSourceRequest{
			Plugin:           plan.Plugin,
			ExpectedRevision: plan.ExpectedCollectionRevision,
			Type:             declaration.TypeAgent,
			Name:             rootDefinition.LogicalName,
			Locator:          agentLocator,
		},
	)
	if err != nil {
		return AgentImportCommitResult{}, err
	}

	record, err := a.publishPreparedManagedAgent(
		ctx,
		plan.RootID,
		plan.SourceID,
		address,
		rawAgent,
		rootDefinition.Digest,
	)
	if err != nil {
		return AgentImportCommitResult{}, fmt.Errorf(
			"publish Agent after Plugin membership publication: %w",
			err,
		)
	}

	agent, err := a.GetAgent(ctx, record.Ref())
	if err != nil {
		return AgentImportCommitResult{}, err
	}
	collectionView, err := a.GetAgentCollection(
		ctx,
		membership.Plugin.Artifact.Ref(),
	)
	if err != nil {
		return AgentImportCommitResult{}, err
	}

	setup := a.bindMCPSetupArtifacts(
		ctx,
		record.Ref(),
		plan.MCPSetupDescriptors,
	)
	return AgentImportCommitResult{
		Agent:               agent,
		Plugin:              collectionView,
		RestoredMemberships: plan.RestoredMemberships,
		MCPSetupDescriptors: setup,
		PreparedFingerprint: fingerprint,
	}, nil
}

func (a *API) agentImportDestination(
	ctx context.Context,
	collectionRef artifactModel.ArtifactRef,
	expectedRevision uint64,
) (agentImportDestinationState, error) {
	if a == nil || a.plugins == nil {
		return agentImportDestinationState{}, spec.ErrClosed
	}
	if err := collectionRef.Validate(); err != nil {
		return agentImportDestinationState{}, err
	}

	value, err := a.plugins.Get(ctx, collectionRef)
	if err != nil {
		return agentImportDestinationState{}, err
	}
	if expectedRevision != 0 &&
		value.Artifact.Revision != expectedRevision {
		return agentImportDestinationState{}, spec.ErrConflict
	}
	if !a.IsManagedAgentCollection(value) {
		return agentImportDestinationState{}, fmt.Errorf(
			"%w: selected Plugin is not an editable managed Agent Plugin",
			spec.ErrUnsupported,
		)
	}
	if a.protection.IsProtectedRoot(value.Artifact.RootID) {
		return agentImportDestinationState{}, fmt.Errorf(
			"%w: protected Collections cannot receive managed Agent imports",
			spec.ErrProtected,
		)
	}
	if err := a.requireMutable(ctx, value.Artifact.RootID); err != nil {
		return agentImportDestinationState{}, err
	}

	sourceValue, err := a.sources.Get(
		ctx,
		value.Artifact.RootID,
		value.Artifact.Binding.SourceID,
	)
	if err != nil {
		return agentImportDestinationState{}, err
	}
	if sourceValue.Kind != managedfs.Kind ||
		sourceValue.StorageKey != agentDomain.AgentManagedSourceStorageKey ||
		!sourceValue.Enabled {
		return agentImportDestinationState{}, fmt.Errorf(
			"%w: selected Plugin does not use the enabled managed Agent Source",
			spec.ErrInvalid,
		)
	}

	inspection, err := a.discovery.InspectSource(
		ctx,
		value.Artifact.RootID,
		value.Artifact.Binding.SourceID,
	)
	if err != nil {
		return agentImportDestinationState{}, err
	}
	if !inspection.IsCurrent() {
		return agentImportDestinationState{}, fmt.Errorf(
			"%w: selected managed Agent Source requires refresh",
			spec.ErrRefreshRequired,
		)
	}

	return agentImportDestinationState{
		value: AgentImportDestination{
			RootID:                value.Artifact.RootID,
			SourceID:              value.Artifact.Binding.SourceID,
			Plugin:                value.Artifact.Ref(),
			CollectionRevision:    value.Artifact.Revision,
			CollectionName:        value.Artifact.LogicalName,
			CollectionDisplayName: value.DisplayName,
			Baseline:              value.Baseline,
			Enabled:               value.Artifact.Enabled,
		},
		plugin:           value,
		sourceGeneration: inspection.State.SourceGeneration,
	}, nil
}

func (a *API) plannedImportArtifacts(
	entry declaration.Entry,
) ([]plannedImportIdentity, []AgentImportArtifactPreview, error) {
	if a == nil || a.interpretations == nil {
		return nil, nil, spec.ErrClosed
	}

	named, err := a.interpretations.WalkNamedEntries(entry)
	if err != nil {
		return nil, nil, err
	}

	identities := make([]plannedImportIdentity, 0, len(named))
	preview := make([]AgentImportArtifactPreview, 0, len(named))
	for _, namedEntry := range named {
		value, err := a.interpretations.DefinitionForEntry(
			namedEntry.Entry,
		)
		if err != nil {
			return nil, nil, err
		}
		declarationType := declaration.Type(value.Kind)
		identities = append(identities, plannedImportIdentity{
			OccurrencePath: string(namedEntry.SubresourceLocator),
			Type:           declarationType,
			Name:           value.LogicalName,
			LogicalVersion: value.LogicalVersion,
			Digest:         value.Digest,
		})
		preview = append(preview, AgentImportArtifactPreview{
			OccurrencePath:   string(namedEntry.SubresourceLocator),
			Type:             declarationType,
			Name:             value.LogicalName,
			LogicalVersion:   value.LogicalVersion,
			DefinitionDigest: value.Digest,
		})
	}
	return identities, preview, nil
}

func (a *API) agentImportIdentityConflicts(
	ctx context.Context,
	rootID rootModel.RootID,
	identities []plannedImportIdentity,
) ([]AgentImportConflict, error) {
	output := make([]AgentImportConflict, 0)
	for _, identity := range identities {
		records, err := a.cat.FindByIdentity(
			ctx,
			rootID,
			artifactModel.ArtifactKind(identity.Type),
			identity.Name,
			catalogModel.ListOptions{},
		)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			if record.LogicalVersion != identity.LogicalVersion {
				continue
			}
			output = append(output, AgentImportConflict{
				Code: "agent.import.identity-conflict",
				Path: identity.OccurrencePath,
				Message: fmt.Sprintf(
					"target Root already contains %s/%s at Artifact %q",
					identity.Type,
					identity.Name,
					record.ID,
				),
			})
		}

		if identity.OccurrencePath != "" ||
			identity.Type != declaration.TypeAgent ||
			rootID == agentBuiltinRootID() {
			continue
		}

		builtinRecords, err := a.cat.FindByIdentity(
			ctx,
			agentBuiltinRootID(),
			artifactModel.ArtifactKind(identity.Type),
			identity.Name,
			catalogModel.ListOptions{},
		)
		if err != nil {
			return nil, err
		}
		if len(builtinRecords) != 0 {
			output = append(output, AgentImportConflict{
				Code: "agent.import.builtin-name-conflict",
				Path: identity.OccurrencePath,
				Message: fmt.Sprintf(
					"protected built-in Agent name %q is reserved",
					identity.Name,
				),
			})
		}
	}
	return output, nil
}

func (a *API) managedAgentPackageConflict(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	address managedpackageModel.ManagedPackageAddress,
) (*AgentImportConflict, error) {
	if a == nil || a.resources == nil {
		return nil, spec.ErrClosed
	}

	directory, err := address.Directory()
	if err != nil {
		return nil, err
	}
	entry, err := a.resources.StatSourceEntry(
		ctx,
		rootID,
		sourceID,
		directory,
	)
	if err != nil {
		if errors.Is(err, spec.ErrNotFound) {
			//nolint:nilnil // Explicit.
			return nil, nil
		}
		return nil, err
	}
	if err := entry.Validate(); err != nil {
		return nil, err
	}
	if entry.Locator != directory {
		return nil, fmt.Errorf(
			"%w: managed package inspection returned %q for %q",
			spec.ErrInvalid,
			entry.Locator,
			directory,
		)
	}

	return &AgentImportConflict{
		Code:    "agent.import.package-conflict",
		Path:    "package",
		Message: fmt.Sprintf("managed Agent package %q is already occupied", directory),
	}, nil
}

func (a *API) analyzeAgentImportMembership(
	ctx context.Context,
	selected plugin.PluginView,
	name spec.LogicalName,
	agentLocator spec.Locator,
) (
	[]AgentRestoredMembership,
	[]AgentImportConflict,
	error,
) {
	if _, err := a.plugins.MemberForCollectionSource(
		ctx,
		selected.Artifact.Ref(),
		declaration.TypeAgent,
		name,
		agentLocator,
	); err != nil {
		return nil, nil, err
	}

	conflicts := make([]AgentImportConflict, 0)
	exact := false
	for _, member := range selected.Members {
		if member.Type != declaration.TypeAgent ||
			member.Name != name {
			continue
		}
		if memberTargetsAgentLocator(
			member,
			selected.Artifact.Binding.Locator,
			agentLocator,
		) {
			exact = true
			continue
		}
		conflicts = append(conflicts, AgentImportConflict{
			Code:    "agent.import.plugin-member-conflict",
			Path:    "members/agent/" + string(name),
			Message: "selected Plugin already contains the imported Agent name at another relationship",
		})
	}

	restored := make([]AgentRestoredMembership, 0)
	if exact {
		restored = append(restored, AgentRestoredMembership{
			Plugin:  selected.Artifact.Ref(),
			Path:    "members/agent/" + string(name),
			Message: "selected Plugin already declares the exact Agent relationship",
		})
	}

	plugins, err := a.ListAgentCollections(
		ctx,
		selected.Artifact.RootID,
	)
	if err != nil {
		return nil, nil, err
	}
	for _, item := range plugins {
		if item.Ref == selected.Artifact.Ref() {
			continue
		}

		value, err := a.GetAgentCollection(ctx, item.Ref)
		if err != nil {
			return nil, nil, err
		}

		for _, member := range value.Members {
			if member.Type != declaration.TypeAgent ||
				member.Name != name ||
				!memberTargetsAgentLocator(
					member,
					value.Artifact.Binding.Locator,
					agentLocator,
				) {
				continue
			}
			restored = append(restored, AgentRestoredMembership{
				Plugin:  value.Artifact.Ref(),
				Path:    "members/agent/" + string(name),
				Message: "existing exact relationship will become available after import",
			})
		}
	}

	return restored, conflicts, nil
}

func memberTargetsAgentLocator(
	member plugin.MemberReference,
	parentLocator spec.Locator,
	targetLocator spec.Locator,
) bool {
	if member.Locator == nil ||
		member.Scope != "" ||
		member.Insert != "" ||
		member.Server != "" {
		return false
	}
	resolved, err := declaration.ResolveSourceRelativePathLocator(
		*member.Locator,
		parentLocator,
	)
	return err == nil && resolved == targetLocator
}

func (a *API) preflightManagedAgentDependencies(
	ctx context.Context,
	rootID rootModel.RootID,
	document agentv1.AgentDocument,
) (
	[]AgentImportRelationship,
	[]AgentMCPSetupDescriptor,
	[]AgentImportIssue,
	error,
) {
	relationships := make([]AgentImportRelationship, 0)
	setup := make([]AgentMCPSetupDescriptor, 0)
	issues := make([]AgentImportIssue, 0)

	for _, member := range document.Members {
		form, err := member.MemberForm()
		if err != nil {
			return nil, nil, nil, err
		}
		header := member.Header()
		memberPath, err := agentDomain.ManagedAgentMemberPath(member)
		if err != nil {
			return nil, nil, nil, err
		}

		if form == declaration.MemberNamed {
			relationship, mcpSetup, issue, err := a.preflightNamedManagedDependency(
				ctx,
				rootID,
				member,
				memberPath,
			)
			if err != nil {
				return nil, nil, nil, err
			}
			relationships = append(relationships, relationship)
			if issue != nil {
				issues = append(issues, *issue)
				continue
			}
			if mcpSetup != nil {
				setup = append(setup, *mcpSetup)
			}
			continue
		}

		if form != declaration.MemberContained ||
			header.Type != declaration.TypeMCP {
			continue
		}

		target, err := member.ContainedDeclaration()
		if err != nil {
			return nil, nil, nil, err
		}
		mcp, err := mcpv1.DecodeMCPEntry(target)
		if err != nil {
			return nil, nil, nil, err
		}
		if mcp.Policy == nil {
			continue
		}

		policyMember, err := declaration.NewEntry(map[string]any{
			"type":  string(declaration.TypeMCPPolicy),
			"name":  string(mcp.Policy.Name),
			"scope": string(declaration.LookupScopeBuiltin),
		})
		if err != nil {
			return nil, nil, nil, err
		}
		relationship, _, issue, err := a.preflightNamedManagedDependency(
			ctx,
			rootID,
			policyMember,
			memberPath+"/policy",
		)
		if err != nil {
			return nil, nil, nil, err
		}
		relationships = append(relationships, relationship)
		if issue != nil {
			issues = append(issues, *issue)
			continue
		}
	}
	return relationships, setup, issues, nil
}

func (a *API) preflightNamedManagedDependency(
	ctx context.Context,
	rootID rootModel.RootID,
	member declaration.Entry,
	memberPath string,
) (
	AgentImportRelationship,
	*AgentMCPSetupDescriptor,
	*AgentImportIssue,
	error,
) {
	if a.declarationResolver == nil {
		return AgentImportRelationship{},
			nil,
			nil,
			spec.ErrClosed
	}

	header := member.Header()
	relationshipFields, err := member.Relationship()
	if err != nil {
		return AgentImportRelationship{},
			nil,
			nil,
			err
	}

	target, err := a.declarationResolver.InspectNamedRelationship(
		ctx,
		composition.NamedRelationshipRequest{
			RootID: rootID,
			Member: member,
		},
	)
	if err != nil {
		return AgentImportRelationship{},
			nil,
			nil,
			err
	}

	output := AgentImportRelationship{
		Path:   memberPath,
		Type:   header.Type,
		Name:   spec.LogicalName(header.Name),
		Scope:  relationshipFields.Scope,
		Status: target.Status,
	}
	if target.Issue != nil {
		output.Code = target.Issue.Code
		output.Message = target.Issue.Message
	}
	if target.Target != nil {
		value := target.Target.Clone()
		output.Target = &value
	}

	if target.Status != composition.ResolutionAvailable {
		return output, nil, managedDependencyIssue(output), nil
	}
	if err := validateManagedDependencyTarget(
		header.Type,
		relationshipFields.Scope,
		target,
		memberPath,
	); err != nil {
		return AgentImportRelationship{}, nil, nil, err
	}

	var setup *AgentMCPSetupDescriptor
	if output.Target != nil &&
		output.Target.Form == composition.TargetFormArtifact &&
		output.Target.Artifact != nil {
		ref := *output.Target.Artifact
		_, err := a.artifacts.Get(ctx, ref)
		if err != nil {
			return AgentImportRelationship{},
				nil,
				nil,
				err
		}
		definitionValue, err := a.artifacts.GetDefinition(ctx, ref)
		if err != nil {
			return AgentImportRelationship{},
				nil,
				nil,
				err
		}

		if header.Type == declaration.TypeMCP {
			document, err := mcpv1.DecodeMCPJSON(definitionValue.Body)
			if err != nil {
				return AgentImportRelationship{},
					nil,
					nil,
					err
			}
			value := importMCPSetupDescriptor(
				&ref,
				agentDomain.MCPSetupDescriptorForDocument(
					memberPath,
					document,
				),
			)
			setup = &value
		}
	}
	return output, setup, nil, nil
}

func managedDependencyIssue(
	value AgentImportRelationship,
) *AgentImportIssue {
	code := "agent.import.dependency-unavailable"
	if value.Status == composition.ResolutionAmbiguous {
		code = "agent.import.dependency-ambiguous"
	}

	message := value.Message
	if message == "" {
		message = fmt.Sprintf(
			"managed dependency %s/%s is %s",
			value.Type,
			value.Name,
			value.Status,
		)
	}
	return &AgentImportIssue{
		Code:     code,
		Severity: AgentImportIssueWarning,
		Path:     value.Path,
		Message:  message,
	}
}

func validateManagedDependencyTarget(
	declarationType declaration.Type,
	scope declaration.LookupScope,
	target composition.NamedRelationshipInspection,
	memberPath string,
) error {
	if target.Target == nil {
		return fmt.Errorf(
			"%w: managed Agent dependency %q resolved without a capability target",
			spec.ErrReferenceUnresolved,
			memberPath,
		)
	}

	if scope != declaration.LookupScopeBuiltin {
		return fmt.Errorf(
			"%w: managed Agent dependency %q is not normalized to built-in scope",
			spec.ErrInvalid,
			memberPath,
		)
	}

	isBuiltinArtifact := func() bool {
		return target.Target.Form == composition.TargetFormArtifact &&
			target.Target.Artifact != nil &&
			target.Target.Artifact.RootID == agentBuiltinRootID()
	}
	isDirectCapability := func() bool {
		return target.Target.Form == composition.TargetFormDirect &&
			target.Target.Provenance ==
				composition.TargetProvenanceDirectCapability
	}

	switch declarationType {
	case declaration.TypeModel, declaration.TypeTool:
		if isBuiltinArtifact() || isDirectCapability() {
			return nil
		}

	case declaration.TypeSkill,
		declaration.TypeMCP,
		declaration.TypeMCPPolicy:
		if isBuiltinArtifact() {
			return nil
		}
	default:
	}

	return fmt.Errorf(
		"%w: managed Agent dependency %q does not resolve to the required protected built-in Artifact or direct capability",
		spec.ErrInvalid,
		memberPath,
	)
}

func (a *API) publishPreparedManagedAgent(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	address managedpackageModel.ManagedPackageAddress,
	raw []byte,
	expectedDefinition cryptoutil.Digest,
) (artifactModel.Artifact, error) {
	if err := a.requireMutable(ctx, rootID); err != nil {
		return artifactModel.Artifact{}, err
	}
	if err := agentDomain.ValidateManagedAgentPackageAddress(address); err != nil {
		return artifactModel.Artifact{}, err
	}

	locator, err := agentDomain.ManagedPackageLocatorForAgent(address)
	if err != nil {
		return artifactModel.Artifact{}, err
	}
	decoderID, err := topology.DefaultDocumentDecoderID(
		topology.DocumentUseManagedAgent,
	)
	if err != nil {
		return artifactModel.Artifact{}, err
	}
	if _, err := a.plugins.EnsureManagedDeclarationDiscovery(
		ctx,
		rootID,
		sourceID,
		locator,
		decoderID,
	); err != nil {
		return artifactModel.Artifact{}, err
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		managepackageModel.PublishRequest{
			RootID: rootID,
			Binding: artifactModel.SourceBinding{
				SourceID: sourceID,
				Locator:  locator,
			},
			ExpectedKind:        agentDomain.AgentArtifactKind,
			ExpectedLogicalName: address.Name,
			ExpectedDefinition:  expectedDefinition,
			Package: managedpackageModel.ManagedPackagePublication{
				Address: address,
				Files: []managedpackageModel.ManagedPackageFile{{
					Locator: agentDomain.ManagedAgentDocumentFile(),
					Content: append([]byte(nil), raw...),
				}},
			},
			AllowPackageReplacement: false,
		},
	)
	if err != nil {
		return artifactModel.Artifact{}, err
	}
	if published.Artifact.ResolvedDefinition == nil ||
		*published.Artifact.ResolvedDefinition != expectedDefinition {
		return artifactModel.Artifact{}, fmt.Errorf(
			"%w: published Agent does not match prepared Definition",
			spec.ErrDigestMismatch,
		)
	}
	return published.Artifact.Clone(), nil
}

func (a *API) bindMCPSetupArtifacts(
	ctx context.Context,
	agentRef artifactModel.ArtifactRef,
	values []AgentMCPSetupDescriptor,
) []AgentMCPSetupDescriptor {
	output := append([]AgentMCPSetupDescriptor(nil), values...)
	plan, err := a.ResolveAgentCapabilities(ctx, agentRef)
	if err != nil {
		return output
	}

	for index := range output {
		if output[index].Artifact != nil {
			continue
		}
		for _, occurrence := range plan.Occurrences {
			if occurrence.Type != declaration.TypeMCP ||
				occurrence.Name != output[index].Name ||
				occurrence.Target == nil ||
				occurrence.Target.Form != composition.TargetFormArtifact ||
				occurrence.Target.Artifact == nil {
				continue
			}
			ref := *occurrence.Target.Artifact
			output[index].Artifact = &ref
			break
		}
	}
	return output
}

func importAdmissionIssues(
	values []agentDomain.ManagedImportIssue,
) []AgentImportIssue {
	output := make([]AgentImportIssue, 0, len(values))
	for _, value := range values {
		severity := AgentImportIssueError
		switch value.Severity {
		case agentDomain.ManagedImportIssueConfirmation:
			severity = AgentImportIssueConfirmation
		case agentDomain.ManagedImportIssueInformation:
			severity = AgentImportIssueInformation
		default:
		}
		output = append(output, AgentImportIssue{
			Code:     value.Code,
			Severity: severity,
			Path:     value.Path,
			Message:  value.Message,
		})
	}
	return output
}

func importMCPSetupDescriptors(
	values []agentDomain.ManagedMCPSetupDescriptor,
) []AgentMCPSetupDescriptor {
	output := make([]AgentMCPSetupDescriptor, 0, len(values))
	for _, value := range values {
		output = append(output, importMCPSetupDescriptor(
			nil,
			value,
		))
	}
	return output
}

func importMCPSetupDescriptor(
	artifactRef *artifactModel.ArtifactRef,
	value agentDomain.ManagedMCPSetupDescriptor,
) AgentMCPSetupDescriptor {
	output := AgentMCPSetupDescriptor{
		OccurrencePath: value.OccurrencePath,
		Name:           value.Name,
		Transport:      value.Transport,
		Command:        value.Command,
		URL:            value.URL,
		AuthMode:       value.AuthMode,
	}
	if artifactRef != nil {
		copyRef := *artifactRef
		output.Artifact = &copyRef
	}
	for _, input := range value.Inputs {
		output.Inputs = append(output.Inputs, AgentMCPSetupInput{
			Name:                 input.Name,
			Kind:                 input.Kind,
			Label:                input.Label,
			Description:          input.Description,
			Required:             input.Required,
			ClientSecretRequired: input.ClientSecretRequired,
		})
	}
	return output
}

func mergeMCPSetupDescriptors(
	left []AgentMCPSetupDescriptor,
	right []AgentMCPSetupDescriptor,
) []AgentMCPSetupDescriptor {
	output := append([]AgentMCPSetupDescriptor(nil), left...)
	seen := make(map[string]int, len(output))
	for index, value := range output {
		seen[value.OccurrencePath] = index
	}
	for _, value := range right {
		if index, found := seen[value.OccurrencePath]; found {
			output[index] = value
			continue
		}
		seen[value.OccurrencePath] = len(output)
		output = append(output, value)
	}
	return output
}

func importIssue(
	code string,
	pathValue string,
	err error,
) AgentImportIssue {
	return AgentImportIssue{
		Code:     code,
		Severity: AgentImportIssueError,
		Path:     pathValue,
		Message:  diagnostic.BoundedMessage(err.Error()),
	}
}

func previewValidationError(
	preview AgentImportPreview,
	code string,
	pathValue string,
	err error,
) (AgentImportPreview, error) {
	preview.Issues = append(preview.Issues, importIssue(
		code,
		pathValue,
		err,
	))
	return preview, nil
}

func hasImportErrors(
	values []AgentImportIssue,
) bool {
	for _, value := range values {
		if value.Severity == AgentImportIssueError {
			return true
		}
	}
	return false
}

func confirmationCodes(
	values []AgentImportIssue,
) []string {
	seen := make(map[string]struct{})
	output := make([]string, 0)
	for _, value := range values {
		if value.Severity != AgentImportIssueConfirmation {
			continue
		}
		if _, duplicate := seen[value.Code]; duplicate {
			continue
		}
		seen[value.Code] = struct{}{}
		output = append(output, value.Code)
	}
	sort.Strings(output)
	return output
}

func sameCodes(
	expected []string,
	actual []string,
) bool {
	left := append([]string(nil), expected...)
	right := append([]string(nil), actual...)
	sort.Strings(left)
	sort.Strings(right)
	left = compactStrings(left)
	right = compactStrings(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func compactStrings(
	values []string,
) []string {
	if len(values) == 0 {
		return nil
	}
	output := values[:1]
	for _, value := range values[1:] {
		if output[len(output)-1] == value {
			continue
		}
		output = append(output, value)
	}
	return output
}
