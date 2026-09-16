package collection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/collectionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

const (
	SkillManagedSourceStorageKey basespec.StorageKey = "user-skills"
	MCPManagedSourceStorageKey   basespec.StorageKey = "user-mcps"

	SkillBaselineCollectionName basespec.LogicalName = "skill-baseline"
	MCPBaselineCollectionName   basespec.LogicalName = "mcp-baseline"
)

type DomainPolicy struct {
	Name                string
	SourceStorageKey    basespec.StorageKey
	SourceDisplayName   string
	BaselineName        basespec.LogicalName
	BaselineDescription string
	AllowedMemberTypes  []declaration.Type
}

func SkillDomainPolicy() DomainPolicy {
	return DomainPolicy{
		Name:                "skill",
		SourceStorageKey:    SkillManagedSourceStorageKey,
		SourceDisplayName:   "User-managed Skills",
		BaselineName:        SkillBaselineCollectionName,
		BaselineDescription: "Application-provisioned editable Skill Collection.",
		AllowedMemberTypes: []declaration.Type{
			declaration.TypeSkill,
		},
	}
}

func MCPDomainPolicy() DomainPolicy {
	return DomainPolicy{
		Name:                "mcp",
		SourceStorageKey:    MCPManagedSourceStorageKey,
		SourceDisplayName:   "User-managed MCP artifacts",
		BaselineName:        MCPBaselineCollectionName,
		BaselineDescription: "Application-provisioned editable MCP Collection.",
		AllowedMemberTypes: []declaration.Type{
			declaration.TypeMCP,
			declaration.TypeMCPPolicy,
		},
	}
}

func (p DomainPolicy) Validate() error {
	if err := basespec.ValidateIdentifier(
		"Collection domain name",
		p.Name,
		basespec.MaxKindBytes,
	); err != nil {
		return err
	}
	if err := p.SourceStorageKey.Validate(); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"Collection domain Source display name",
		p.SourceDisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := p.BaselineName.Validate(); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"Collection baseline description",
		p.BaselineDescription,
		basespec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if len(p.AllowedMemberTypes) == 0 {
		return fmt.Errorf(
			"%w: Collection domain %q has no allowed member types",
			basespec.ErrInvalid,
			p.Name,
		)
	}
	seen := make(map[declaration.Type]struct{}, len(p.AllowedMemberTypes))
	for _, value := range p.AllowedMemberTypes {
		if err := value.Validate(); err != nil {
			return err
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf(
				"%w: Collection domain %q repeats member type %q",
				basespec.ErrInvalid,
				p.Name,
				value,
			)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func (p DomainPolicy) allows(value declaration.Type) bool {
	return slices.Contains(p.AllowedMemberTypes, value)
}

func (a *API) EnsureBaseline(
	ctx context.Context,
	rootID root.RootID,
) (CollectionView, error) {
	if a == nil || a.domain == nil {
		return CollectionView{}, fmt.Errorf(
			"%w: Collection baseline is not configured",
			basespec.ErrUnsupported,
		)
	}
	if err := rootID.Validate(); err != nil {
		return CollectionView{}, err
	}

	sourceValue, err := a.managedSource(ctx, rootID, "")
	if err != nil {
		return CollectionView{}, err
	}
	address, err := managedCollectionAddress(a.domain.BaselineName)
	if err != nil {
		return CollectionView{}, err
	}
	locator, err := address.FileLocator(ManagedCollectionDocumentFile)
	if err != nil {
		return CollectionView{}, err
	}
	if _, err := a.EnsureManagedDeclarationDiscovery(
		ctx,
		rootID,
		sourceValue.ID,
		locator,
		decoder.JSONDecoderID,
	); err != nil {
		return CollectionView{}, err
	}
	if err := compositionapi.EnsureSourceCurrent(
		ctx,
		a.discovery,
		rootID,
		sourceValue.ID,
	); err != nil {
		return CollectionView{}, err
	}

	existing, err := a.artifacts.FindByOrigin(
		ctx,
		rootID,
		artifact.SourceBinding{
			SourceID: sourceValue.ID,
			Locator:  locator,
		},
		artifact.ArtifactKind(collectionv1.CollectionType),
	)
	switch {
	case err == nil:
		if existing.State == artifact.StateAvailable {
			if !existing.Enabled {
				existing, err = a.artifacts.SetEnabled(
					ctx,
					existing.Ref(),
					existing.Revision,
					true,
				)
				if err != nil {
					return CollectionView{}, err
				}
			}
			return a.Read(ctx, existing.Ref())
		}

		document := collectionv1.CollectionDocument{
			APIVersion:  collectionv1.CollectionSchemaVersion,
			Type:        collectionv1.CollectionType,
			Name:        string(a.domain.BaselineName),
			Description: a.domain.BaselineDescription,
		}
		record, err := a.publishDocument(
			ctx,
			rootID,
			sourceValue.ID,
			address,
			document,
			"",
			true,
		)
		if err != nil {
			return CollectionView{}, err
		}
		return a.Read(ctx, record.Ref())

	case errors.Is(err, basespec.ErrArtifactNotFound),
		errors.Is(err, basespec.ErrNotFound):
	default:
		return CollectionView{}, err
	}

	return a.create(
		ctx,
		CreateRequest{
			RootID:      rootID,
			SourceID:    sourceValue.ID,
			Name:        a.domain.BaselineName,
			Description: a.domain.BaselineDescription,
		},
		true,
	)
}

func (a *API) domainManagedSource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) (source.Summary, error) {
	if a == nil || a.domain == nil {
		return source.Summary{}, basespec.ErrClosed
	}
	if sourceID != "" {
		value, err := a.sources.Get(ctx, rootID, sourceID)
		if err != nil {
			return source.Summary{}, err
		}
		if value.Kind != source.SourceKindManagedDirectory ||
			value.StorageKey != a.domain.SourceStorageKey {
			return source.Summary{}, fmt.Errorf(
				"%w: Collection belongs to another managed domain Source",
				basespec.ErrUnsupported,
			)
		}
		if !value.Enabled {
			return source.Summary{}, fmt.Errorf(
				"%w: Collection domain Source is disabled",
				basespec.ErrConflict,
			)
		}
		return value, nil
	}

	value, _, err := a.sources.Ensure(
		ctx,
		rootID,
		source.Draft{
			ID:          source.SourceID(uuidutil.NewUUIDv7()),
			StorageKey:  a.domain.SourceStorageKey,
			Kind:        source.SourceKindManagedDirectory,
			DisplayName: a.domain.SourceDisplayName,
			Enabled:     true,
			Config:      json.RawMessage(`{}`),
			Discovery:   source.DiscoverySpec{},
		},
	)
	if err != nil {
		return source.Summary{}, err
	}
	if value.Enabled && value.DisplayName == a.domain.SourceDisplayName {
		return value, nil
	}
	return a.sources.Update(
		ctx,
		rootID,
		value.ID,
		source.Update{
			ExpectedRevision: value.Revision,
			DisplayName:      a.domain.SourceDisplayName,
			Enabled:          true,
		},
	)
}

func (a *API) validateDomainMember(
	member MemberReference,
) error {
	if a == nil || a.domain == nil {
		return nil
	}
	if !a.domain.allows(member.Type) {
		return fmt.Errorf(
			"%w: %s Collection cannot contain %q members",
			basespec.ErrUnsupported,
			a.domain.Name,
			member.Type,
		)
	}
	return nil
}

func (a *API) validateEditableDomainDocument(
	document collectionv1.CollectionDocument,
) error {
	for index, member := range document.Members {
		form, err := member.CompositionForm()
		if err != nil {
			return fmt.Errorf("collection members[%d]: %w", index, err)
		}
		if form != declaration.CompositionEntryReference {
			return fmt.Errorf(
				"%w: editable collection cannot contain declarations",
				basespec.ErrUnsupported,
			)
		}
		if a.domain != nil &&
			!a.domain.allows(member.Header().Type) {
			return fmt.Errorf(
				"%w: %s collection member %q has incompatible type %q",
				basespec.ErrUnsupported,
				a.domain.Name,
				member.Header().Name,
				member.Header().Type,
			)
		}
	}
	return nil
}

func IsBaselineCollectionArtifact(
	value artifact.Artifact,
) bool {
	if value.Kind != artifact.ArtifactKind(collectionv1.CollectionType) {
		return false
	}
	address, err := managedCollectionAddressFromLocator(
		value.Binding.Locator,
	)
	if err != nil {
		return false
	}
	return (value.LogicalName == SkillBaselineCollectionName &&
		address.Name == SkillBaselineCollectionName) ||
		(value.LogicalName == MCPBaselineCollectionName &&
			address.Name == MCPBaselineCollectionName)
}

func IsBaselineCollectionArtifactForSource(
	value artifact.Artifact,
	sourceValue source.Summary,
) bool {
	if value.Kind != artifact.ArtifactKind(collectionv1.CollectionType) ||
		value.RootID != sourceValue.RootID ||
		value.Binding.SourceID != sourceValue.ID ||
		sourceValue.Kind != source.SourceKindManagedDirectory {
		return false
	}
	address, err := managedCollectionAddressFromLocator(
		value.Binding.Locator,
	)
	if err != nil {
		return false
	}
	return (value.LogicalName == SkillBaselineCollectionName &&
		address.Name == SkillBaselineCollectionName &&
		sourceValue.StorageKey == SkillManagedSourceStorageKey) ||
		(value.LogicalName == MCPBaselineCollectionName &&
			address.Name == MCPBaselineCollectionName &&
			sourceValue.StorageKey == MCPManagedSourceStorageKey)
}

func (a *API) readCollectionDocument(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (artifact.Artifact, collectionv1.CollectionDocument, error) {
	record, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return artifact.Artifact{}, collectionv1.CollectionDocument{}, err
	}
	if record.Kind != artifact.ArtifactKind(collectionv1.CollectionType) ||
		record.State != artifact.StateAvailable {
		return artifact.Artifact{}, collectionv1.CollectionDocument{}, fmt.Errorf(
			"%w: Artifact %q is not an available Collection",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	definitionValue, err := a.artifacts.GetDefinition(ctx, ref)
	if err != nil {
		return artifact.Artifact{}, collectionv1.CollectionDocument{}, err
	}
	document, err := collectionv1.DecodeCollectionJSON(definitionValue.Body)
	if err != nil {
		return artifact.Artifact{}, collectionv1.CollectionDocument{}, err
	}
	if document.Name != string(record.LogicalName) {
		return artifact.Artifact{}, collectionv1.CollectionDocument{}, fmt.Errorf(
			"%w: Collection declaration name differs from Artifact identity",
			basespec.ErrInvalid,
		)
	}
	return record, document, nil
}

func (a *API) domainCollectionVisible(
	ctx context.Context,
	record artifact.Artifact,
	document collectionv1.CollectionDocument,
) (bool, error) {
	if a.domain == nil {
		return true, nil
	}

	sourceValue, err := a.sources.Get(
		ctx,
		record.RootID,
		record.Binding.SourceID,
	)
	if err != nil {
		return false, err
	}
	if sourceValue.StorageKey == a.domain.SourceStorageKey {
		return true, a.validateEditableDomainDocument(document)
	}
	if len(document.Members) == 0 {
		return false, nil
	}
	for _, member := range document.Members {
		if !a.domain.allows(member.Header().Type) {
			return false, nil
		}
	}
	return true, nil
}

func (a *API) Read(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (CollectionView, error) {
	if a == nil {
		return CollectionView{}, basespec.ErrClosed
	}
	ref, err := a.resolveCollectionRef(ctx, ref)
	if err != nil {
		return CollectionView{}, err
	}
	record, document, err := a.readCollectionDocument(ctx, ref)
	if err != nil {
		return CollectionView{}, err
	}
	visible, err := a.domainCollectionVisible(ctx, record, document)
	if err != nil {
		return CollectionView{}, err
	}
	if !visible {
		return CollectionView{}, fmt.Errorf(
			"%w: Collection is not visible in this domain",
			basespec.ErrUnsupported,
		)
	}

	editable := false
	sourceValue, err := a.sources.Get(
		ctx,
		record.RootID,
		record.Binding.SourceID,
	)
	if err != nil {
		return CollectionView{}, err
	}
	if record.Binding.SubresourceLocator == "" &&
		sourceValue.Kind == source.SourceKindManagedDirectory &&
		(a.domain == nil ||
			sourceValue.StorageKey == a.domain.SourceStorageKey) {
		if _, err := managedCollectionAddressFromLocator(
			record.Binding.Locator,
		); err == nil &&
			a.validateEditableDomainDocument(document) == nil {
			editable = true
		}
	}

	baseline := IsBaselineCollectionArtifactForSource(record, sourceValue)
	return readCollectionViewOf(
		record,
		document,
		editable,
		editable && !baseline,
		baseline,
	)
}

func (a *API) ListDomain(
	ctx context.Context,
	rootID root.RootID,
) ([]CollectionView, error) {
	if a == nil {
		return nil, basespec.ErrClosed
	}
	if err := rootID.Validate(); err != nil {
		return nil, err
	}

	records, err := a.artifacts.ListByRoot(ctx, rootID)
	if err != nil {
		return nil, err
	}
	output := make([]CollectionView, 0)
	seen := make(map[artifact.ArtifactRef]struct{})
	for _, record := range records {
		if record.Kind != artifact.ArtifactKind(collectionv1.CollectionType) ||
			record.State != artifact.StateAvailable {
			continue
		}
		view, err := a.Read(ctx, record.Ref())
		if errors.Is(err, basespec.ErrUnsupported) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if _, duplicate := seen[view.Artifact.Ref()]; duplicate {
			continue
		}
		seen[view.Artifact.Ref()] = struct{}{}
		output = append(output, view)
	}
	sort.Slice(output, func(left, right int) bool {
		if output[left].Name != output[right].Name {
			return output[left].Name < output[right].Name
		}
		return output[left].Artifact.ID < output[right].Artifact.ID
	})
	return output, nil
}

func readCollectionViewOf(
	record artifact.Artifact,
	document collectionv1.CollectionDocument,
	editable bool,
	deletable bool,
	baseline bool,
) (CollectionView, error) {
	ordered, err := declaration.SortedCompositionEntries(
		"Collection members",
		document.Members,
	)
	if err != nil {
		return CollectionView{}, err
	}

	view := CollectionView{
		Artifact:    record.Clone(),
		Name:        basespec.LogicalName(document.Name),
		Description: document.Description,
		Version:     basespec.LogicalVersion(document.Version),
		Editable:    editable,
		Deletable:   deletable,
		Baseline:    baseline,
		Members:     make([]MemberReference, 0, len(ordered)),
		Entries:     make([]CollectionMemberView, 0, len(ordered)),
	}
	for index, entry := range ordered {
		form, err := entry.CompositionForm()
		if err != nil {
			return CollectionView{}, fmt.Errorf(
				"collection members[%d]: %w",
				index,
				err,
			)
		}
		header := entry.Header()
		member := CollectionMemberView{
			Type:        header.Type,
			Name:        basespec.LogicalName(header.Name),
			Description: header.Description,
			Metadata:    declaration.CloneRawMessageMap(header.Metadata),
			Contained:   form == declaration.CompositionEntryContained,
		}
		if header.Locator != nil {
			locator := header.Locator.Clone()
			member.Locator = &locator
		}
		if form == declaration.CompositionEntryReference {
			reference, err := memberReferenceFromEntry(entry)
			if err != nil {
				return CollectionView{}, err
			}
			member.Server = reference.Server
			view.Members = append(view.Members, reference)
		}
		view.Entries = append(view.Entries, member)
	}
	return view, nil
}
