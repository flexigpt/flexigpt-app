package collection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/pluginv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
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
	BaselineDisplayName string
	BaselineDescription string
	PackageKind         source.PackageKind
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
	if err := basespec.ValidateIdentifier(
		"Collection domain name",
		p.Name,
		basespec.MaxKindBytes,
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
			basespec.ErrInvalid,
		)
	}
	switch strings.ToLower(path.Ext(string(documentFile))) {
	case ".json", ".yaml", ".yml":
	default:
		return fmt.Errorf(
			"%w: Collection domain document file %q has an unsupported extension",
			basespec.ErrInvalid,
			documentFile,
		)
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
				basespec.ErrInvalid,
				p.Name,
				form,
			)
		}
		if _, duplicate := seenForms[form]; duplicate {
			return fmt.Errorf(
				"%w: Collection domain %q repeats member form %q",
				basespec.ErrInvalid,
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

func (p DomainPolicy) managedCollectionPackageKind() source.PackageKind {
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
	rootID root.RootID,
) (CollectionView, error) {
	if a == nil || a.domain == nil {
		return CollectionView{}, fmt.Errorf(
			"%w: Collection baseline is not configured",
			basespec.ErrUnsupported,
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
		artifact.ArtifactKind(pluginv1.PluginType),
	)
	switch {
	case err == nil:
		if existing.State == artifact.StateAvailable {
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
			DisplayName: a.domain.managedCollectionBaselineDisplayName(),
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
	if err := a.requireDeclarationAuthoring(); err != nil {
		return source.Summary{}, err
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
	if !a.domain.allowsMemberForm(declaration.MemberNamed) {
		return fmt.Errorf(
			"%w: %s Collection does not support named external members",
			basespec.ErrUnsupported,
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
			basespec.ErrUnsupported,
			a.domain.Name,
			member.Header().Type,
		)
	}
	if !a.domain.allowsMemberForm(form) {
		return fmt.Errorf(
			"%w: %s Collection cannot contain %q members",
			basespec.ErrUnsupported,
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
				basespec.ErrUnsupported,
			)
		}
		if err := a.validateDomainEntry(member); err != nil {
			return fmt.Errorf("collection members[%d]: %w", index, err)
		}
	}
	return nil
}

func IsBaselineCollectionArtifact(
	value artifact.Artifact,
) bool {
	if value.Kind != artifact.ArtifactKind(pluginv1.PluginType) {
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
	if value.Kind != artifact.ArtifactKind(pluginv1.PluginType) ||
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
) (artifact.Artifact, pluginv1.PluginDocument, error) {
	record, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return artifact.Artifact{}, pluginv1.PluginDocument{}, err
	}
	if record.Kind != artifact.ArtifactKind(pluginv1.PluginType) ||
		record.State != artifact.StateAvailable {
		return artifact.Artifact{}, pluginv1.PluginDocument{}, fmt.Errorf(
			"%w: Artifact %q is not an available Plugin",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if a.domain != nil &&
		a.domain.ReadOnly &&
		!a.readOnlyDomainOrigin(record) {
		return artifact.Artifact{}, pluginv1.PluginDocument{}, fmt.Errorf(
			"%w: Collection does not belong to this read-only domain",
			basespec.ErrUnsupported,
		)
	}
	definitionValue, err := a.artifacts.GetDefinition(ctx, ref)
	if err != nil {
		return artifact.Artifact{}, pluginv1.PluginDocument{}, err
	}
	if err := definitionValue.Validate(); err != nil {
		return artifact.Artifact{}, pluginv1.PluginDocument{}, err
	}
	if record.ResolvedDefinition == nil ||
		*record.ResolvedDefinition != definitionValue.Digest {
		return artifact.Artifact{}, pluginv1.PluginDocument{}, fmt.Errorf(
			"%w: Collection definition changed during read",
			basespec.ErrRefreshRequired,
		)
	}
	document, err := pluginv1.DecodePluginJSON(definitionValue.Body)
	if err != nil {
		return artifact.Artifact{}, pluginv1.PluginDocument{}, err
	}
	entry, err := declaration.NewEntry(document)
	if err != nil {
		return artifact.Artifact{}, pluginv1.PluginDocument{}, err
	}
	expected, err := decoder.DefinitionForEntry(entry)
	if err != nil {
		return artifact.Artifact{}, pluginv1.PluginDocument{}, err
	}
	if definitionValue.Kind != expected.Kind ||
		definitionValue.SchemaID != expected.SchemaID ||
		definitionValue.SchemaVersion != expected.SchemaVersion {
		return artifact.Artifact{}, pluginv1.PluginDocument{}, fmt.Errorf(
			"%w: Collection definition has an unsupported schema",
			basespec.ErrUnsupported,
		)
	}
	if document.Name != string(record.LogicalName) {
		return artifact.Artifact{}, pluginv1.PluginDocument{}, fmt.Errorf(
			"%w: Plugin declaration name differs from Artifact identity",
			basespec.ErrInvalid,
		)
	}
	return record, document, nil
}

func (a *API) domainCollectionVisible(
	ctx context.Context,
	record artifact.Artifact,
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
		if sourceValue.Kind != source.SourceKindManagedDirectory ||
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
		sourceValue.Kind == source.SourceKindManagedDirectory &&
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
		if record.Kind != artifact.ArtifactKind(pluginv1.PluginType) ||
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
		Name:        basespec.LogicalName(document.Name),
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
			Name:      basespec.LogicalName(header.Name),
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
