package consumerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	agentDomain "github.com/flexigpt/flexigpt-app/internal/agent/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcpv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

const managedAgentPreparedTTL = 10 * time.Minute

type agentImportDestinationState struct {
	value            AgentImportDestination
	sourceGeneration string
}

type preparedAgentImport struct {
	ProfileID string    `json:"profileID"`
	ExpiresAt time.Time `json:"expiresAt"`

	SourceDigest     cryptoutil.Digest `json:"sourceDigest"`
	DefinitionDigest cryptoutil.Digest `json:"definitionDigest"`

	CanonicalDeclaration json.RawMessage `json:"canonicalDeclaration"`

	RootID   root.RootID     `json:"rootID"`
	SourceID source.SourceID `json:"sourceID"`

	Collection                 artifact.ArtifactRef `json:"collection"`
	ExpectedCollectionRevision uint64               `json:"expectedCollectionRevision"`
	ExpectedSourceGeneration   string               `json:"expectedSourceGeneration"`

	Address      source.ManagedPackageAddress `json:"address"`
	AgentLocator basespec.Locator             `json:"agentLocator"`

	Dependencies              []preparedDependencyWitness `json:"dependencies"`
	RestoredMemberships       []AgentRestoredMembership   `json:"restoredMemberships"`
	MCPSetupDescriptors       []AgentMCPSetupDescriptor   `json:"mcpSetupDescriptors"`
	RequiredConfirmationCodes []string                    `json:"requiredConfirmationCodes"`
}

type preparedDependencyWitness struct {
	Path   string          `json:"path"`
	Member json.RawMessage `json:"member"`

	Artifact         *artifact.ArtifactRef `json:"artifact,omitempty"`
	ArtifactRevision uint64                `json:"artifactRevision,omitempty"`
	DefinitionDigest cryptoutil.Digest     `json:"definitionDigest,omitempty"`

	Mapped *resolve.MappedTarget `json:"mapped,omitempty"`
}

type plannedImportIdentity struct {
	OccurrencePath string
	Type           declaration.Type
	Name           basespec.LogicalName
	LogicalVersion basespec.LogicalVersion
	Digest         cryptoutil.Digest
}

func (a *API) ListAgentImportDestinations(
	ctx context.Context,
	rootID root.RootID,
) ([]AgentImportDestination, error) {
	if a == nil || a.collections == nil {
		return nil, basespec.ErrClosed
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
		if !a.IsManagedAgentCollection(value) {
			continue
		}

		sourceValue, err := a.sources.Get(
			ctx,
			rootID,
			value.Artifact.Binding.SourceID,
		)
		if err != nil {
			return nil, err
		}
		if sourceValue.Kind != source.SourceKindManagedDirectory ||
			sourceValue.StorageKey != agentDomain.AgentManagedSourceStorageKey ||
			!sourceValue.Enabled {
			continue
		}

		output = append(output, AgentImportDestination{
			RootID:                rootID,
			SourceID:              value.Artifact.Binding.SourceID,
			Collection:            value,
			CollectionRevision:    value.Artifact.Revision,
			CollectionName:        value.Artifact.LogicalName,
			CollectionDisplayName: value.Artifact.DisplayName,
			Baseline:              value.Baseline,
			Enabled:               value.Artifact.Enabled,
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
		return AgentImportPreview{}, basespec.ErrClosed
	}
	if ctx == nil {
		return AgentImportPreview{}, fmt.Errorf(
			"%w: Agent import preview context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return AgentImportPreview{}, err
	}
	if request.ExpectedCollectionRevision == 0 {
		return AgentImportPreview{}, fmt.Errorf(
			"%w: expected Agent Collection revision is required",
			basespec.ErrInvalid,
		)
	}
	if !isYAMLImportPath(request.Path) {
		return AgentImportPreview{}, fmt.Errorf(
			"%w: managed Agent import accepts YAML files only",
			basespec.ErrInvalid,
		)
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
		request.Collection,
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
		basespec.MaxDefinitionBytes,
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

	canonical, err := yamlutil.CanonicalObjectJSON(
		sourceBytes,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		preview.Issues = append(preview.Issues, importIssue(
			"agent.import.yaml-invalid",
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

	normalizedYAML, err := yamlutil.CanonicalObjectYAML(
		canonical,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return preview, err
	}
	preview.NormalizedYAML = string(normalizedYAML)

	entry, err := declaration.DecodeCanonicalEntryJSON(canonical)
	if err != nil {
		preview.Issues = append(preview.Issues, importIssue(
			"agent.import.declaration-invalid",
			"",
			err,
		))
		return preview, nil
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
		return preview, err
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
		return preview, err
	}
	preview.DefinitionDigest = rootDefinition.Digest

	address, err := agentDomain.ManagedPackageAddressForAgent(
		rootDefinition.LogicalName,
	)
	if err != nil {
		return preview, err
	}
	agentLocator, err := agentDomain.ManagedPackageLocatorForAgent(address)
	if err != nil {
		return preview, err
	}

	identities, artifacts, err := plannedImportArtifacts(entry)
	if err != nil {
		return preview, err
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

	restored, membershipConflicts, err := a.analyzeAgentImportMembership(
		ctx,
		destination.value.Collection,
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

	relationships, witnesses, dependencySetups, dependencyIssues, err := a.preflightManagedAgentDependencies(
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
		Collection:                 request.Collection,
		ExpectedCollectionRevision: request.ExpectedCollectionRevision,
		ExpectedSourceGeneration:   destination.sourceGeneration,
		Address:                    address,
		AgentLocator:               agentLocator,
		Dependencies:               witnesses,
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
		return AgentImportCommitResult{}, basespec.ErrClosed
	}
	if ctx == nil {
		return AgentImportCommitResult{}, fmt.Errorf(
			"%w: Agent import commit context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return AgentImportCommitResult{}, err
	}
	if request.Prepared == "" {
		return AgentImportCommitResult{}, fmt.Errorf(
			"%w: prepared Agent import is required",
			basespec.ErrInvalid,
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
			basespec.ErrConflict,
		)
	}
	if plan.ProfileID != agentv1.ManagedAgentImportProfileID {
		return AgentImportCommitResult{}, fmt.Errorf(
			"%w: prepared Agent import profile is unsupported",
			basespec.ErrInvalid,
		)
	}
	if !sameCodes(
		plan.RequiredConfirmationCodes,
		request.AcceptedConfirmationCodes,
	) {
		return AgentImportCommitResult{}, fmt.Errorf(
			"%w: accepted confirmation codes differ from preview",
			basespec.ErrConflict,
		)
	}

	destination, err := a.agentImportDestination(
		ctx,
		plan.Collection,
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
			basespec.ErrConflict,
		)
	}

	canonical, err := a.managedAgentProfile.Validate(
		plan.CanonicalDeclaration,
	)
	if err != nil {
		return AgentImportCommitResult{}, err
	}
	entry, err := declaration.DecodeCanonicalEntryJSON(canonical)
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
			basespec.ErrInvalid,
		)
	}

	rawAgent, rootDefinition, err := agentDomain.ManagedAgentEntryPayload(entry)
	if err != nil {
		return AgentImportCommitResult{}, err
	}
	if rootDefinition.Digest != plan.DefinitionDigest {
		return AgentImportCommitResult{}, fmt.Errorf(
			"%w: prepared Agent Definition digest changed",
			basespec.ErrConflict,
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
			basespec.ErrConflict,
		)
	}
	agentLocator, err := agentDomain.ManagedPackageLocatorForAgent(address)
	if err != nil {
		return AgentImportCommitResult{}, err
	}
	if agentLocator != plan.AgentLocator {
		return AgentImportCommitResult{}, fmt.Errorf(
			"%w: prepared Agent package locator changed",
			basespec.ErrConflict,
		)
	}

	identities, _, err := plannedImportArtifacts(entry)
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
			basespec.ErrConflict,
		)
	}
	if err := a.verifyDependencyWitnesses(
		ctx,
		plan.RootID,
		plan.Dependencies,
	); err != nil {
		return AgentImportCommitResult{}, err
	}

	membership, err := a.collections.EnsureMemberForCollectionSource(
		ctx,
		collection.EnsureMemberForCollectionSourceRequest{
			Collection:       plan.Collection,
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
			"publish Agent after Collection membership publication: %w",
			err,
		)
	}

	agent, err := a.GetAgentView(ctx, record.Ref())
	if err != nil {
		return AgentImportCommitResult{}, err
	}
	collectionView, err := a.GetAgentCollection(
		ctx,
		membership.Collection.Artifact.Ref(),
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
		Collection:          collectionView,
		RestoredMemberships: plan.RestoredMemberships,
		MCPSetupDescriptors: setup,
		PreparedFingerprint: fingerprint,
	}, nil
}

func (a *API) agentImportDestination(
	ctx context.Context,
	collectionRef artifact.ArtifactRef,
	expectedRevision uint64,
) (agentImportDestinationState, error) {
	if a == nil || a.collections == nil {
		return agentImportDestinationState{}, basespec.ErrClosed
	}
	if err := collectionRef.Validate(); err != nil {
		return agentImportDestinationState{}, err
	}

	value, err := a.collections.Get(ctx, collectionRef)
	if err != nil {
		return agentImportDestinationState{}, err
	}
	if expectedRevision != 0 &&
		value.Artifact.Revision != expectedRevision {
		return agentImportDestinationState{}, basespec.ErrConflict
	}
	if !a.IsManagedAgentCollection(value) {
		return agentImportDestinationState{}, fmt.Errorf(
			"%w: selected Collection is not an editable managed Agent Collection",
			basespec.ErrUnsupported,
		)
	}
	if a.protection.IsProtectedRoot(value.Artifact.RootID) {
		return agentImportDestinationState{}, fmt.Errorf(
			"%w: protected Collections cannot receive managed Agent imports",
			basespec.ErrProtected,
		)
	}
	if err := a.requireMutable(ctx, value.Artifact.RootID, false); err != nil {
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
	if sourceValue.Kind != source.SourceKindManagedDirectory ||
		sourceValue.StorageKey != agentDomain.AgentManagedSourceStorageKey ||
		!sourceValue.Enabled {
		return agentImportDestinationState{}, fmt.Errorf(
			"%w: selected Collection does not use the enabled managed Agent Source",
			basespec.ErrInvalid,
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
			basespec.ErrRefreshRequired,
		)
	}

	return agentImportDestinationState{
		value: AgentImportDestination{
			RootID:                value.Artifact.RootID,
			SourceID:              value.Artifact.Binding.SourceID,
			Collection:            value,
			CollectionRevision:    value.Artifact.Revision,
			CollectionName:        value.Artifact.LogicalName,
			CollectionDisplayName: value.Artifact.DisplayName,
			Baseline:              value.Baseline,
			Enabled:               value.Artifact.Enabled,
		},
		sourceGeneration: inspection.State.SourceGeneration,
	}, nil
}

func plannedImportArtifacts(
	entry declaration.Entry,
) ([]plannedImportIdentity, []AgentImportArtifactPreview, error) {
	named, err := declaration.WalkNamedEntries(entry)
	if err != nil {
		return nil, nil, err
	}

	identities := make([]plannedImportIdentity, 0, len(named))
	preview := make([]AgentImportArtifactPreview, 0, len(named))
	for _, namedEntry := range named {
		value, err := decoder.DefinitionForNamedEntry(namedEntry)
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
	rootID root.RootID,
	identities []plannedImportIdentity,
) ([]AgentImportConflict, error) {
	output := make([]AgentImportConflict, 0)
	for _, identity := range identities {
		records, err := a.artifacts.FindByIdentity(
			ctx,
			rootID,
			artifact.ArtifactKind(identity.Type),
			identity.Name,
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

		builtinRecords, err := a.artifacts.FindByIdentity(
			ctx,
			agentBuiltinRootID(),
			artifact.ArtifactKind(identity.Type),
			identity.Name,
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

func (a *API) analyzeAgentImportMembership(
	ctx context.Context,
	selected collection.CollectionView,
	name basespec.LogicalName,
	agentLocator basespec.Locator,
) (
	[]AgentRestoredMembership,
	[]AgentImportConflict,
	error,
) {
	if _, err := a.collections.MemberForCollectionSource(
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
	for _, entry := range selected.Entries {
		if entry.Type != declaration.TypeAgent {
			continue
		}
		if entry.Contained || entry.Selector {
			conflicts = append(conflicts, AgentImportConflict{
				Code:    "agent.import.collection-nonexternal-member",
				Path:    "members",
				Message: "selected Collection has a contained or selector Agent member",
			})
		}
	}
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
			Code:    "agent.import.collection-member-conflict",
			Path:    "members/agent/" + string(name),
			Message: "selected Collection already contains the imported Agent name at another relationship",
		})
	}

	restored := make([]AgentRestoredMembership, 0)
	if exact {
		restored = append(restored, AgentRestoredMembership{
			Collection: selected.Artifact.Ref(),
			Path:       "members/agent/" + string(name),
			Message:    "selected Collection already declares the exact Agent relationship",
		})
	}

	collections, err := a.ListAgentCollections(
		ctx,
		selected.Artifact.RootID,
	)
	if err != nil {
		return nil, nil, err
	}
	for _, value := range collections {
		if value.Artifact.Ref() == selected.Artifact.Ref() {
			continue
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
				Collection: value.Artifact.Ref(),
				Path:       "members/agent/" + string(name),
				Message:    "existing exact relationship will become available after import",
			})
		}
	}

	return restored, conflicts, nil
}

func memberTargetsAgentLocator(
	member collection.MemberReference,
	parentLocator basespec.Locator,
	targetLocator basespec.Locator,
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
	rootID root.RootID,
	document agentv1.AgentDocument,
) (
	[]AgentImportRelationship,
	[]preparedDependencyWitness,
	[]AgentMCPSetupDescriptor,
	[]AgentImportIssue,
	error,
) {
	relationships := make([]AgentImportRelationship, 0)
	witnesses := make([]preparedDependencyWitness, 0)
	setup := make([]AgentMCPSetupDescriptor, 0)
	issues := make([]AgentImportIssue, 0)

	for _, member := range document.Members {
		form, err := member.MemberForm()
		if err != nil {
			return nil, nil, nil, nil, err
		}
		header := member.Header()
		memberPath := "members/" + string(header.Type) + "/" + header.Name

		if form == declaration.MemberNamed {
			relationship, witness, mcpSetup, issue, err := a.preflightNamedManagedDependency(
				ctx,
				rootID,
				member,
				memberPath,
			)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			if issue != nil {
				issues = append(issues, *issue)
				continue
			}
			relationships = append(relationships, relationship)
			witnesses = append(witnesses, witness)
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
			return nil, nil, nil, nil, err
		}
		mcp, err := mcpv1.DecodeMCPEntry(target)
		if err != nil {
			return nil, nil, nil, nil, err
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
			return nil, nil, nil, nil, err
		}
		relationship, witness, _, issue, err := a.preflightNamedManagedDependency(
			ctx,
			rootID,
			policyMember,
			memberPath+"/policy",
		)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if issue != nil {
			issues = append(issues, *issue)
			continue
		}
		relationships = append(relationships, relationship)
		witnesses = append(witnesses, witness)
	}
	return relationships, witnesses, setup, issues, nil
}

func (a *API) preflightNamedManagedDependency(
	ctx context.Context,
	rootID root.RootID,
	member declaration.Entry,
	memberPath string,
) (
	AgentImportRelationship,
	preparedDependencyWitness,
	*AgentMCPSetupDescriptor,
	*AgentImportIssue,
	error,
) {
	if a.declarationResolver == nil {
		return AgentImportRelationship{},
			preparedDependencyWitness{},
			nil,
			nil,
			basespec.ErrClosed
	}

	header := member.Header()
	relationshipFields, err := member.Relationship()
	if err != nil {
		return AgentImportRelationship{},
			preparedDependencyWitness{},
			nil,
			nil,
			err
	}

	target, err := a.declarationResolver.ResolveNamedRelationship(
		ctx,
		resolve.NamedRelationshipRequest{
			RootID: rootID,
			Member: member,
		},
	)
	if err != nil {
		issue := importIssue(
			"agent.import.dependency-unresolved",
			memberPath,
			err,
		)
		return AgentImportRelationship{},
			preparedDependencyWitness{},
			nil,
			&issue,
			nil
	}

	if issue := validateManagedDependencyTarget(
		header.Type,
		relationshipFields.Scope,
		target,
		memberPath,
	); issue != nil {
		return AgentImportRelationship{},
			preparedDependencyWitness{},
			nil,
			issue,
			nil
	}

	raw, err := member.CanonicalJSON()
	if err != nil {
		return AgentImportRelationship{},
			preparedDependencyWitness{},
			nil,
			nil,
			err
	}

	output := AgentImportRelationship{
		Path:   memberPath,
		Type:   header.Type,
		Name:   basespec.LogicalName(header.Name),
		Scope:  relationshipFields.Scope,
		Status: "available",
	}
	witness := preparedDependencyWitness{
		Path:   memberPath,
		Member: raw,
	}

	var setup *AgentMCPSetupDescriptor
	if target.Artifact != nil {
		ref := *target.Artifact
		record, err := a.artifacts.Get(ctx, ref)
		if err != nil {
			return AgentImportRelationship{},
				preparedDependencyWitness{},
				nil,
				nil,
				err
		}
		definitionValue, err := a.artifacts.GetDefinition(ctx, ref)
		if err != nil {
			return AgentImportRelationship{},
				preparedDependencyWitness{},
				nil,
				nil,
				err
		}

		output.Artifact = &ref
		witness.Artifact = &ref
		witness.ArtifactRevision = record.Revision
		witness.DefinitionDigest = definitionValue.Digest

		if header.Type == declaration.TypeMCP {
			document, err := mcpv1.DecodeMCPJSON(definitionValue.Body)
			if err != nil {
				return AgentImportRelationship{},
					preparedDependencyWitness{},
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
	if target.Mapped != nil {
		value := *target.Mapped
		output.Mapped = &value
		witness.Mapped = &value
	}
	return output, witness, setup, nil, nil
}

func validateManagedDependencyTarget(
	declarationType declaration.Type,
	scope declaration.LookupScope,
	target resolve.NamedRelationshipTarget,
	memberPath string,
) *AgentImportIssue {
	switch declarationType {
	case declaration.TypeModel, declaration.TypeTool:
		if scope == "" &&
			target.Mapped != nil &&
			target.Mapped.Builtin {
			return nil
		}
		if scope == declaration.LookupScopeBuiltin {
			if target.Artifact != nil &&
				target.Artifact.RootID == agentBuiltinRootID() {
				return nil
			}
			if target.Mapped != nil && target.Mapped.Builtin {
				return nil
			}
		}

	case declaration.TypeSkill,
		declaration.TypeMCP,
		declaration.TypeMCPPolicy:
		if scope == declaration.LookupScopeBuiltin &&
			target.Artifact != nil &&
			target.Artifact.RootID == agentBuiltinRootID() {
			return nil
		}
	default:
	}

	return &AgentImportIssue{
		Code:     "agent.import.reference-provenance",
		Severity: AgentImportIssueError,
		Path:     memberPath,
		Message:  "managed Agent dependency does not resolve to the required protected built-in or mapped target",
	}
}

func (a *API) verifyDependencyWitnesses(
	ctx context.Context,
	rootID root.RootID,
	witnesses []preparedDependencyWitness,
) error {
	for _, expected := range witnesses {
		member, err := declaration.DecodeCanonicalEntryJSON(
			expected.Member,
		)
		if err != nil {
			return err
		}
		target, err := a.declarationResolver.ResolveNamedRelationship(
			ctx,
			resolve.NamedRelationshipRequest{
				RootID: rootID,
				Member: member,
			},
		)
		if err != nil {
			return fmt.Errorf(
				"%w: dependency %q changed: %w",
				basespec.ErrConflict,
				expected.Path,
				err,
			)
		}

		if expected.Artifact != nil {
			if target.Artifact == nil ||
				*target.Artifact != *expected.Artifact {
				return fmt.Errorf(
					"%w: dependency %q Artifact target changed",
					basespec.ErrConflict,
					expected.Path,
				)
			}

			record, err := a.artifacts.Get(ctx, *expected.Artifact)
			if err != nil {
				return err
			}
			definitionValue, err := a.artifacts.GetDefinition(
				ctx,
				*expected.Artifact,
			)
			if err != nil {
				return err
			}
			if record.Revision != expected.ArtifactRevision ||
				definitionValue.Digest != expected.DefinitionDigest {
				return fmt.Errorf(
					"%w: dependency %q witness changed",
					basespec.ErrConflict,
					expected.Path,
				)
			}
		}

		if expected.Mapped != nil {
			if target.Mapped == nil ||
				*target.Mapped != *expected.Mapped {
				return fmt.Errorf(
					"%w: dependency %q mapped target changed",
					basespec.ErrConflict,
					expected.Path,
				)
			}
		}
	}
	return nil
}

func (a *API) publishPreparedManagedAgent(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	address source.ManagedPackageAddress,
	raw []byte,
	expectedDefinition cryptoutil.Digest,
) (artifact.Artifact, error) {
	if err := a.requireMutable(ctx, rootID, false); err != nil {
		return artifact.Artifact{}, err
	}
	if err := agentDomain.ValidateManagedAgentPackageAddress(address); err != nil {
		return artifact.Artifact{}, err
	}

	locator, err := agentDomain.ManagedPackageLocatorForAgent(address)
	if err != nil {
		return artifact.Artifact{}, err
	}
	decoderID, err := documentTopology.DefaultDocumentDecoderID(
		documentTopology.DocumentUseManagedAgent,
	)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if _, err := a.collections.EnsureManagedDeclarationDiscovery(
		ctx,
		rootID,
		sourceID,
		locator,
		decoderID,
	); err != nil {
		return artifact.Artifact{}, err
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		artifact.PublishArtifactRequest{
			RootID: rootID,
			Binding: artifact.SourceBinding{
				SourceID: sourceID,
				Locator:  locator,
			},
			ExpectedKind:        agentDomain.AgentArtifactKind,
			ExpectedLogicalName: address.Name,
			ExpectedDefinition:  expectedDefinition,
			Package: source.ManagedPackagePublication{
				Address: address,
				Files: []source.ManagedPackageFile{{
					Locator: agentDomain.ManagedAgentDocumentFile(),
					Content: append([]byte(nil), raw...),
				}},
			},
			AllowPackageReplacement: false,
		},
	)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if published.Artifact.ResolvedDefinition == nil ||
		*published.Artifact.ResolvedDefinition != expectedDefinition {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: published Agent does not match prepared Definition",
			basespec.ErrDigestMismatch,
		)
	}
	return published.Artifact.Clone(), nil
}

func (a *API) bindMCPSetupArtifacts(
	ctx context.Context,
	agentRef artifact.ArtifactRef,
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
				occurrence.Artifact == nil {
				continue
			}
			ref := *occurrence.Artifact
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
	artifactRef *artifact.ArtifactRef,
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
		Message:  err.Error(),
	}
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

func isYAMLImportPath(
	value string,
) bool {
	switch strings.ToLower(path.Ext(value)) {
	case ".yaml", ".yml":
		return true
	default:
		return false
	}
}
