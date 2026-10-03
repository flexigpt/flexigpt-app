package collection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/pluginv1"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

const (
	SkillManagedSourceStorageKey spec.StorageKey = "user-skills"
	MCPManagedSourceStorageKey   spec.StorageKey = "user-mcps"

	SkillBaselineCollectionName spec.LogicalName = "skill-baseline"
	MCPBaselineCollectionName   spec.LogicalName = "mcp-baseline"
)

type DomainPolicy struct {
	Name                string
	SourceStorageKey    spec.StorageKey
	SourceDisplayName   string
	BaselineName        spec.LogicalName
	BaselineDisplayName string
	BaselineDescription string
	PackageKind         sourceModel.PackageKind
	DocumentUse         string
	AllowedMemberTypes  []declaration.Type
	AllowedMemberForms  []declaration.MemberForm

	// ReadOnly prohibits declaration authoring and baseline provisioning.
	// Local Artifact enablement remains supported.
	ReadOnly bool

	// ValidateDocument applies additional consumer-specific restrictions
	// after generic Collection decoding and domain visibility checks.
	// It is internal configuration, never a transport projection.
	ValidateDocument func(pluginv1.PluginDocument) error
}

func SkillDomainPolicy() DomainPolicy {
	return DomainPolicy{
		Name:                "skill",
		SourceStorageKey:    SkillManagedSourceStorageKey,
		SourceDisplayName:   "User-managed Skills",
		BaselineName:        SkillBaselineCollectionName,
		BaselineDisplayName: "Skill Baseline",
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
		BaselineDisplayName: "MCP Baseline",
		BaselineDescription: "Application-provisioned editable MCP Collection.",
		AllowedMemberTypes: []declaration.Type{
			declaration.TypeMCP,
			declaration.TypeMCPPolicy,
		},
	}
}

func (p DomainPolicy) Validate() error {
	if err := spec.ValidateIdentifier(
		"Collection domain name",
		p.Name,
		spec.MaxKindBytes,
	); err != nil {
		return err
	}
	if err := p.validateAuthoringConfiguration(); err != nil {
		return err
	}
	if err := p.managedCollectionPackageKind().Validate(); err != nil {
		return err
	}
	documentFile, err := documentTopology.DefaultDocumentFile(
		p.managedCollectionDocumentUse(),
	)
	if err != nil {
		return err
	}
	if err := documentFile.ValidatePortable(false); err != nil {
		return err
	}
	if path.Base(string(documentFile)) != string(documentFile) {
		return fmt.Errorf(
			"%w: Collection domain document file must be a package-root file",
			spec.ErrInvalid,
		)
	}
	switch strings.ToLower(path.Ext(string(documentFile))) {
	case ".json", ".yaml", ".yml":
	default:
		return fmt.Errorf(
			"%w: Collection domain document file %q has an unsupported extension",
			spec.ErrInvalid,
			documentFile,
		)
	}
	if len(p.AllowedMemberTypes) == 0 {
		return fmt.Errorf(
			"%w: Collection domain %q has no allowed member types",
			spec.ErrInvalid,
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
				spec.ErrInvalid,
				p.Name,
				value,
			)
		}
		seen[value] = struct{}{}
	}
	seenForms := make(
		map[declaration.MemberForm]struct{},
		len(p.AllowedMemberForms),
	)
	for _, form := range p.AllowedMemberForms {
		switch form {
		case declaration.MemberNamed,
			declaration.MemberContained,
			declaration.MemberSelector:
		default:
			return fmt.Errorf(
				"%w: Collection domain %q has unsupported member form %q",
				spec.ErrInvalid,
				p.Name,
				form,
			)
		}
		if _, duplicate := seenForms[form]; duplicate {
			return fmt.Errorf(
				"%w: Collection domain %q repeats member form %q",
				spec.ErrInvalid,
				p.Name,
				form,
			)
		}
		seenForms[form] = struct{}{}
	}
	return nil
}

func (p DomainPolicy) allowsMemberForm(
	value declaration.MemberForm,
) bool {
	if len(p.AllowedMemberForms) == 0 {
		return value == declaration.MemberNamed
	}
	return slices.Contains(p.AllowedMemberForms, value)
}

func (p DomainPolicy) managedCollectionPackageKind() sourceModel.PackageKind {
	if p.PackageKind != "" {
		return p.PackageKind
	}
	return ManagedCollectionPackageKind
}

func (p DomainPolicy) managedCollectionDocumentUse() string {
	if p.DocumentUse != "" {
		return p.DocumentUse
	}
	return documentTopology.DocumentUseManagedCollection
}

func (p DomainPolicy) managedCollectionBaselineDisplayName() string {
	if p.BaselineDisplayName != "" {
		return p.BaselineDisplayName
	}
	return string(p.BaselineName)
}

func (p DomainPolicy) allows(value declaration.Type) bool {
	return slices.Contains(p.AllowedMemberTypes, value)
}

func (a *API) EnsureBaseline(
	ctx context.Context,
	rootID rootModel.RootID,
) (CollectionView, error) {
	if a == nil || a.domain == nil {
		return CollectionView{}, fmt.Errorf(
			"%w: Collection baseline is not configured",
			spec.ErrUnsupported,
		)
	}
	if err := a.requireDeclarationAuthoring(); err != nil {
		return CollectionView{}, err
	}
	if err := rootID.Validate(); err != nil {
		return CollectionView{}, err
	}

	sourceValue, err := a.managedSource(ctx, rootID, "")
	if err != nil {
		return CollectionView{}, err
	}
	address, err := a.managedCollectionAddress(a.domain.BaselineName)
	if err != nil {
		return CollectionView{}, err
	}
	documentFile := a.managedCollectionDocumentFile()
	locator, err := address.FileLocator(documentFile)
	if err != nil {
		return CollectionView{}, err
	}
	decoderID, err := a.managedCollectionDecoderID()
	if err != nil {
		return CollectionView{}, err
	}
	if _, err := a.EnsureManagedDeclarationDiscovery(
		ctx,
		rootID,
		sourceValue.ID,
		locator,
		decoderID,
	); err != nil {
		return CollectionView{}, err
	}

	// Keep the managed declaration origin configured before inspecting the
	// existing baseline. This is cheap on the steady path and repairs an
	// incomplete discovery configuration without forcing a Source scan.
	existing, err := a.artifacts.FindByOrigin(
		ctx,
		rootID,
		artifactModel.SourceBinding{
			SourceID: sourceValue.ID,
			Locator:  locator,
		},
		artifactModel.ArtifactKind(pluginv1.PluginType),
	)
	if err != nil && !errors.Is(err, spec.ErrArtifactNotFound) && !errors.Is(err, spec.ErrNotFound) {
		return CollectionView{}, err
	} else if existing.State == artifactModel.StateAvailable {
		return a.Read(ctx, existing.Ref())
	}

	existing, err = a.artifacts.FindByOrigin(
		ctx, rootID, artifactModel.SourceBinding{SourceID: sourceValue.ID, Locator: locator},
		artifactModel.ArtifactKind(pluginv1.PluginType),
	)
	switch {
	case err == nil:
		if existing.State == artifactModel.StateAvailable {
			return a.Read(ctx, existing.Ref())
		}
		document := pluginv1.PluginDocument{
			Type:        pluginv1.PluginType,
			Name:        string(a.domain.BaselineName),
			DisplayName: a.domain.managedCollectionBaselineDisplayName(),
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

	case errors.Is(err, spec.ErrArtifactNotFound),
		errors.Is(err, spec.ErrNotFound):
	default:
		return CollectionView{}, err
	}

	return a.create(
		ctx,
		CreateRequest{
			RootID:      rootID,
			SourceID:    sourceValue.ID,
			Name:        a.domain.BaselineName,
			DisplayName: a.domain.managedCollectionBaselineDisplayName(),
			Description: a.domain.BaselineDescription,
		},
		true,
	)
}

func (a *API) domainManagedSource(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
) (sourceModel.Summary, error) {
	if a == nil || a.domain == nil {
		return sourceModel.Summary{}, spec.ErrClosed
	}
	if err := a.requireDeclarationAuthoring(); err != nil {
		return sourceModel.Summary{}, err
	}
	if sourceID != "" {
		value, err := a.sources.Get(ctx, rootID, sourceID)
		if err != nil {
			return sourceModel.Summary{}, err
		}
		if value.Kind != sourceModel.SourceKindManagedDirectory ||
			value.StorageKey != a.domain.SourceStorageKey {
			return sourceModel.Summary{}, fmt.Errorf(
				"%w: Collection belongs to another managed domain Source",
				spec.ErrUnsupported,
			)
		}
		if !value.Enabled {
			return sourceModel.Summary{}, fmt.Errorf(
				"%w: Collection domain Source is disabled",
				spec.ErrConflict,
			)
		}
		return value, nil
	}

	value, _, err := a.sources.Ensure(
		ctx,
		rootID,
		sourceModel.Draft{
			ID:          sourceModel.SourceID(uuidutil.NewUUIDv7()),
			StorageKey:  a.domain.SourceStorageKey,
			Kind:        sourceModel.SourceKindManagedDirectory,
			DisplayName: a.domain.SourceDisplayName,
			Enabled:     true,
			Config:      json.RawMessage(`{}`),
			Discovery:   sourceModel.DiscoverySpec{},
		},
	)
	if err != nil {
		return sourceModel.Summary{}, err
	}
	if value.Enabled && value.DisplayName == a.domain.SourceDisplayName {
		return value, nil
	}
	return a.sources.Update(
		ctx,
		rootID,
		value.ID,
		sourceModel.Update{
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
			spec.ErrUnsupported,
			a.domain.Name,
			member.Type,
		)
	}
	if !a.domain.allowsMemberForm(declaration.MemberNamed) {
		return fmt.Errorf(
			"%w: %s Collection does not support named external members",
			spec.ErrUnsupported,
			a.domain.Name,
		)
	}
	return nil
}

func (a *API) validateDomainEntry(
	member declaration.Entry,
) error {
	form, err := member.MemberForm()
	if err != nil {
		return err
	}
	if a == nil || a.domain == nil {
		return nil
	}
	if !a.domain.allows(member.Header().Type) {
		return fmt.Errorf(
			"%w: %s Collection cannot contain %q members",
			spec.ErrUnsupported,
			a.domain.Name,
			member.Header().Type,
		)
	}
	if !a.domain.allowsMemberForm(form) {
		return fmt.Errorf(
			"%w: %s Collection cannot contain %q members",
			spec.ErrUnsupported,
			a.domain.Name,
			form,
		)
	}
	return nil
}

func (a *API) validateEditableDomainDocument(
	document pluginv1.PluginDocument,
) error {
	for index, member := range document.Members {
		form, err := member.MemberForm()
		if err != nil {
			return fmt.Errorf("collection members[%d]: %w", index, err)
		}
		if form != declaration.MemberNamed {
			return fmt.Errorf(
				"%w: editable Plugin cannot contain contained or selector members",
				spec.ErrUnsupported,
			)
		}
		if err := a.validateDomainEntry(member); err != nil {
			return fmt.Errorf("collection members[%d]: %w", index, err)
		}
	}
	return nil
}

func IsBaselineCollectionArtifact(
	value artifactModel.Artifact,
) bool {
	if value.Kind != artifactModel.ArtifactKind(pluginv1.PluginType) {
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
	value artifactModel.Artifact,
	sourceValue sourceModel.Summary,
) bool {
	if value.Kind != artifactModel.ArtifactKind(pluginv1.PluginType) ||
		value.RootID != sourceValue.RootID ||
		value.Binding.SourceID != sourceValue.ID ||
		sourceValue.Kind != sourceModel.SourceKindManagedDirectory {
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
	ref artifactModel.ArtifactRef,
) (artifactModel.Artifact, pluginv1.PluginDocument, error) {
	record, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return artifactModel.Artifact{}, pluginv1.PluginDocument{}, err
	}
	if record.Kind != artifactModel.ArtifactKind(pluginv1.PluginType) ||
		record.State != artifactModel.StateAvailable {
		return artifactModel.Artifact{}, pluginv1.PluginDocument{}, fmt.Errorf(
			"%w: Artifact %q is not an available Plugin",
			spec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if a.domain != nil &&
		a.domain.ReadOnly &&
		!a.readOnlyDomainOrigin(record) {
		return artifactModel.Artifact{}, pluginv1.PluginDocument{}, fmt.Errorf(
			"%w: Collection does not belong to this read-only domain",
			spec.ErrUnsupported,
		)
	}
	definitionValue, err := a.artifacts.GetDefinition(ctx, ref)
	if err != nil {
		return artifactModel.Artifact{}, pluginv1.PluginDocument{}, err
	}

	if record.ResolvedDefinition == nil ||
		*record.ResolvedDefinition != definitionValue.Digest {
		return artifactModel.Artifact{}, pluginv1.PluginDocument{}, fmt.Errorf(
			"%w: Collection definition changed during read",
			spec.ErrRefreshRequired,
		)
	}
	document, err := pluginv1.DecodePluginJSON(definitionValue.Body)
	if err != nil {
		return artifactModel.Artifact{}, pluginv1.PluginDocument{}, err
	}

	if document.Name != string(record.LogicalName) {
		return artifactModel.Artifact{}, pluginv1.PluginDocument{}, fmt.Errorf(
			"%w: Plugin declaration name differs from Artifact identity",
			spec.ErrInvalid,
		)
	}
	return record, document, nil
}

func (a *API) domainCollectionVisible(
	ctx context.Context,
	record artifactModel.Artifact,
	document pluginv1.PluginDocument,
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
	if a.domain.ReadOnly {
		if sourceValue.Kind != sourceModel.SourceKindManagedDirectory ||
			!a.readOnlyDomainOrigin(record) {
			return false, nil
		}
		return true, a.validateEditableDomainDocument(document)
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
	ref artifactModel.ArtifactRef,
) (CollectionView, error) {
	if a == nil {
		return CollectionView{}, spec.ErrClosed
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
			spec.ErrUnsupported,
		)
	}
	if a.domain != nil && a.domain.ValidateDocument != nil {
		if err := a.domain.ValidateDocument(document); err != nil {
			return CollectionView{}, err
		}
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
		(a.domain == nil || !a.domain.ReadOnly) &&
		sourceValue.Kind == sourceModel.SourceKindManagedDirectory &&
		(a.domain == nil ||
			sourceValue.StorageKey == a.domain.SourceStorageKey) {
		if _, err := a.managedCollectionAddressFromLocator(
			record.Binding.Locator,
		); err == nil &&
			a.validateEditableDomainDocument(document) == nil {
			editable = true
		}
	}

	baseline := IsBaselineCollectionArtifactForSource(record, sourceValue)
	if a.domain != nil && a.domain.ReadOnly {
		baseline = false
	}
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
	request ListRequest,
) ([]ListItem, error) {
	return a.listCollections(ctx, request, true)
}

func readCollectionViewOf(
	record artifactModel.Artifact,
	document pluginv1.PluginDocument,
	editable bool,
	deletable bool,
	baseline bool,
) (CollectionView, error) {
	ordered, err := declaration.SortedMembers(
		"Collection members",
		document.Members,
	)
	if err != nil {
		return CollectionView{}, err
	}
	displayName := document.DisplayName
	if displayName == "" {
		displayName = record.DisplayName
	}
	view := CollectionView{
		Artifact:    record.Clone(),
		Name:        spec.LogicalName(document.Name),
		DisplayName: displayName,
		Description: document.Description,
		Editable:    editable,
		Deletable:   deletable,
		Baseline:    baseline,
		Members:     make([]MemberReference, 0, len(ordered)),
		Entries:     make([]CollectionMemberView, 0, len(ordered)),
	}
	for index, entry := range ordered {
		form, err := entry.MemberForm()
		if err != nil {
			return CollectionView{}, fmt.Errorf(
				"collection members[%d]: %w",
				index,
				err,
			)
		}
		header := entry.Header()
		member := CollectionMemberView{
			Type:      header.Type,
			Name:      spec.LogicalName(header.Name),
			Contained: form == declaration.MemberContained,
			Selector:  form == declaration.MemberSelector,
		}
		if header.Locator != nil {
			locator := header.Locator.Clone()
			member.Locator = &locator
		}
		if header.Type == declaration.TypeText {
			insert, err := entry.TextInsert()
			if err != nil {
				return CollectionView{}, err
			}
			member.Insert = insert
		}
		if form == declaration.MemberNamed {
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
