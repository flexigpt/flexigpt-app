package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/managedfs"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	pluginv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/contract/v1"
	pluginDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/domain"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

const (
	SkillManagedPluginSourceStorageKey spec.StorageKey = "user-skills"
	MCPManagedPluginSourceStorageKey   spec.StorageKey = "user-mcps"

	SkillBaselinePluginName spec.LogicalName = "skill-baseline"
	MCPBaselinePluginName   spec.LogicalName = "mcp-baseline"
)

// Profile is supplied by the owning family. Plugin owns enforcement of the
// profile at managed Plugin boundaries but does not invent Agent, Skill, MCP,
// or Tool membership policy itself.
type Profile struct {
	Name                string
	SourceStorageKey    spec.StorageKey
	SourceDisplayName   string
	BaselineName        spec.LogicalName
	BaselineDisplayName string
	BaselineDescription string
	PackageKind         managedpackageModel.PackageKind
	DocumentUse         string
	MembershipPolicy    pluginDomain.MembershipPolicy

	// ReadOnly prohibits declaration authoring and baseline provisioning.
	// Local Artifact enablement remains supported.
	ReadOnly bool

	// ValidateDocument applies additional consumer-specific restrictions
	// after generic Plugin decoding and domain visibility checks.
	// It is internal configuration, never a transport projection.
	ValidateDocument func(pluginv1.PluginDocument) error
}

func (p Profile) Validate() error {
	if err := spec.ValidateIdentifier(
		"Plugin family profile name",
		p.Name,
		spec.MaxKindBytes,
	); err != nil {
		return err
	}
	if err := p.validateAuthoringConfiguration(); err != nil {
		return err
	}
	if err := p.managedPluginPackageKind().Validate(); err != nil {
		return err
	}
	documentFile, err := topology.DefaultDocumentFile(
		p.managedPluginDocumentUse(),
	)
	if err != nil {
		return err
	}
	if err := documentFile.ValidatePortable(false); err != nil {
		return err
	}
	if path.Base(string(documentFile)) != string(documentFile) {
		return fmt.Errorf(
			"%w: plugin domain document file must be a package-root file",
			spec.ErrInvalid,
		)
	}
	switch strings.ToLower(path.Ext(string(documentFile))) {
	case ".json", ".yaml", ".yml":
	default:
		return fmt.Errorf(
			"%w: plugin domain document file %q has an unsupported extension",
			spec.ErrInvalid,
			documentFile,
		)
	}
	if err := p.MembershipPolicy.Validate(); err != nil {
		return fmt.Errorf("plugin membership policy: %w", err)
	}
	return nil
}

func (p Profile) allowsMemberForm(
	value declaration.MemberForm,
) bool {
	if len(p.MembershipPolicy.AllowedForms) == 0 {
		return value == declaration.MemberNamed
	}
	return slices.Contains(p.MembershipPolicy.AllowedForms, value)
}

func (p Profile) allows(
	value declaration.Type,
) bool {
	if p.MembershipPolicy.Mode ==
		pluginDomain.MembershipModeLegacyUnconstrained {
		return true
	}
	return slices.Contains(
		p.MembershipPolicy.AllowedTypes,
		value,
	)
}

func (p Profile) managedPluginPackageKind() managedpackageModel.PackageKind {
	if p.PackageKind != "" {
		return p.PackageKind
	}
	return ManagedPluginPackageKind
}

func (p Profile) managedPluginDocumentUse() string {
	if p.DocumentUse != "" {
		return p.DocumentUse
	}
	return topology.DocumentUseManagedPlugin
}

func (p Profile) managedPluginBaselineDisplayName() string {
	if p.BaselineDisplayName != "" {
		return p.BaselineDisplayName
	}
	return string(p.BaselineName)
}

func (a *API) EnsureBaseline(
	ctx context.Context,
	rootID rootModel.RootID,
) (PluginView, error) {
	if a == nil || a.domain == nil {
		return PluginView{}, fmt.Errorf(
			"%w: Plugin baseline is not configured",
			spec.ErrUnsupported,
		)
	}
	if err := a.requireDeclarationAuthoring(); err != nil {
		return PluginView{}, err
	}
	if err := rootID.Validate(); err != nil {
		return PluginView{}, err
	}

	sourceValue, err := a.managedSource(ctx, rootID, "")
	if err != nil {
		return PluginView{}, err
	}
	address, err := a.managedPluginAddress(a.domain.BaselineName)
	if err != nil {
		return PluginView{}, err
	}
	documentFile := a.managedPluginDocumentFile()
	locator, err := address.FileLocator(documentFile)
	if err != nil {
		return PluginView{}, err
	}
	decoderID, err := a.managedPluginDecoderID()
	if err != nil {
		return PluginView{}, err
	}
	if _, err := a.EnsureManagedDeclarationDiscovery(
		ctx,
		rootID,
		sourceValue.ID,
		locator,
		decoderID,
	); err != nil {
		return PluginView{}, err
	}

	// Keep the managed declaration origin configured before inspecting the
	// existing baseline. This is cheap on the steady path and repairs an
	// incomplete discovery configuration without forcing a Source scan.
	existing, err := a.artifacts.FindByOrigin(
		ctx, rootID, artifactModel.SourceBinding{SourceID: sourceValue.ID, Locator: locator},
		artifactModel.ArtifactKind(pluginv1.PluginType),
	)
	switch {
	case err == nil:
		if existing.State == artifactModel.StateAvailable {
			return a.Read(ctx, existing.Ref())
		}

		metadata, err := pluginDomain.PutMetadata(
			nil,
			a.domain.MembershipPolicy,
		)
		if err != nil {
			return PluginView{}, err
		}
		document := pluginv1.PluginDocument{
			Type:        pluginv1.PluginType,
			Name:        string(a.domain.BaselineName),
			DisplayName: a.domain.managedPluginBaselineDisplayName(),
			Description: a.domain.BaselineDescription,
			Metadata:    metadata,
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
			return PluginView{}, err
		}
		return a.Read(ctx, record.Ref())

	case errors.Is(err, spec.ErrArtifactNotFound),
		errors.Is(err, spec.ErrNotFound):
	default:
		return PluginView{}, err
	}

	return a.create(
		ctx,
		CreateRequest{
			RootID:      rootID,
			SourceID:    sourceValue.ID,
			Name:        a.domain.BaselineName,
			DisplayName: a.domain.managedPluginBaselineDisplayName(),
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
	if a.domain == nil {
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
		if value.Kind != managedfs.Kind ||
			value.StorageKey != a.domain.SourceStorageKey {
			return sourceModel.Summary{}, fmt.Errorf(
				"%w: Plugin belongs to another managed domain Source",
				spec.ErrUnsupported,
			)
		}
		if !value.Enabled {
			return sourceModel.Summary{}, fmt.Errorf(
				"%w: Plugin domain Source is disabled",
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
			Kind:        managedfs.Kind,
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
			"%w: %s Plugin cannot contain %q members",
			spec.ErrUnsupported,
			a.domain.Name,
			member.Type,
		)
	}
	if !a.domain.allowsMemberForm(declaration.MemberNamed) {
		return fmt.Errorf(
			"%w: %s Plugin does not support named external members",
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
	if err := a.domain.MembershipPolicy.Allows(member); err != nil {
		return err
	}
	if !a.domain.allows(member.Header().Type) {
		return fmt.Errorf(
			"%w: %s Plugin cannot contain %q members",
			spec.ErrUnsupported,
			a.domain.Name,
			member.Header().Type,
		)
	}
	if !a.domain.allowsMemberForm(form) {
		return fmt.Errorf(
			"%w: %s Plugin cannot contain %q members",
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
			return fmt.Errorf("plugin members[%d]: %w", index, err)
		}
		if form != declaration.MemberNamed {
			return fmt.Errorf(
				"%w: editable Plugin cannot contain contained or selector members",
				spec.ErrUnsupported,
			)
		}
		if err := a.validateDomainEntry(member); err != nil {
			return fmt.Errorf("plugin members[%d]: %w", index, err)
		}
	}
	return nil
}

func (a *API) isBaselinePlugin(
	record artifactModel.Artifact,
	sourceValue sourceModel.Summary,
) bool {
	if a == nil || a.domain == nil {
		return IsBaselinePluginArtifactForSource(
			record,
			sourceValue,
		)
	}
	if a.domain.ReadOnly ||
		record.Kind != artifactModel.ArtifactKind(pluginv1.PluginType) ||
		record.RootID != sourceValue.RootID ||
		record.Binding.SourceID != sourceValue.ID ||
		record.Binding.SubresourceLocator != "" ||
		sourceValue.Kind != managedfs.Kind ||
		sourceValue.StorageKey != a.domain.SourceStorageKey {
		return false
	}

	address, err := a.managedPluginAddressFromLocator(
		record.Binding.Locator,
	)
	if err != nil {
		return false
	}
	return record.LogicalName == a.domain.BaselineName &&
		address.Name == a.domain.BaselineName
}

func IsBaselinePluginArtifact(
	value artifactModel.Artifact,
) bool {
	if value.Kind != artifactModel.ArtifactKind(pluginv1.PluginType) {
		return false
	}
	address, err := managedPluginAddressFromLocator(
		value.Binding.Locator,
	)
	if err != nil {
		return false
	}
	return (value.LogicalName == SkillBaselinePluginName &&
		address.Name == SkillBaselinePluginName) ||
		(value.LogicalName == MCPBaselinePluginName &&
			address.Name == MCPBaselinePluginName)
}

func IsBaselinePluginArtifactForSource(
	value artifactModel.Artifact,
	sourceValue sourceModel.Summary,
) bool {
	if value.Kind != artifactModel.ArtifactKind(pluginv1.PluginType) ||
		value.RootID != sourceValue.RootID ||
		value.Binding.SourceID != sourceValue.ID ||
		sourceValue.Kind != managedfs.Kind {
		return false
	}
	address, err := managedPluginAddressFromLocator(
		value.Binding.Locator,
	)
	if err != nil {
		return false
	}
	return (value.LogicalName == SkillBaselinePluginName &&
		address.Name == SkillBaselinePluginName &&
		sourceValue.StorageKey == SkillManagedPluginSourceStorageKey) ||
		(value.LogicalName == MCPBaselinePluginName &&
			address.Name == MCPBaselinePluginName &&
			sourceValue.StorageKey == MCPManagedPluginSourceStorageKey)
}

func (a *API) readPluginDocument(
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
			"%w: Plugin does not belong to this read-only domain",
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
			"%w: Plugin definition changed during read",
			spec.ErrRefreshRequired,
		)
	}
	document, err := pluginv1.FromDefinition(definitionValue)
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

func (a *API) domainPluginVisible(
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
		if sourceValue.Kind != managedfs.Kind ||
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
) (PluginView, error) {
	ref, err := a.resolvePluginRef(ctx, ref)
	if err != nil {
		return PluginView{}, err
	}
	record, document, err := a.readPluginDocument(ctx, ref)
	if err != nil {
		return PluginView{}, err
	}
	visible, err := a.domainPluginVisible(ctx, record, document)
	if err != nil {
		return PluginView{}, err
	}
	if !visible {
		return PluginView{}, fmt.Errorf(
			"%w: Plugin is not visible in this domain",
			spec.ErrUnsupported,
		)
	}
	if a.domain != nil && a.domain.ValidateDocument != nil {
		if err := a.domain.ValidateDocument(document); err != nil {
			return PluginView{}, err
		}
	}

	editable := false
	sourceValue, err := a.sources.Get(
		ctx,
		record.RootID,
		record.Binding.SourceID,
	)
	if err != nil {
		return PluginView{}, err
	}
	if record.Binding.SubresourceLocator == "" &&
		(a.domain == nil || !a.domain.ReadOnly) &&
		sourceValue.Kind == managedfs.Kind &&
		(a.domain == nil ||
			sourceValue.StorageKey == a.domain.SourceStorageKey) {
		if _, err := a.managedPluginAddressFromLocator(
			record.Binding.Locator,
		); err == nil &&
			a.validateEditableDomainDocument(document) == nil {
			editable = true
		}
	}

	baseline := a.isBaselinePlugin(record, sourceValue)
	return readPluginViewOf(
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
	return a.listPlugins(ctx, request, true)
}

func readPluginViewOf(
	record artifactModel.Artifact,
	document pluginv1.PluginDocument,
	editable bool,
	deletable bool,
	baseline bool,
) (PluginView, error) {
	ordered, err := declaration.SortedMembers(
		"Plugin members",
		document.Members,
	)
	if err != nil {
		return PluginView{}, err
	}
	displayName := document.DisplayName
	if displayName == "" {
		displayName = record.DisplayName
	}
	membership, err := pluginDomain.FromMetadata(document.Metadata)
	if err != nil {
		return PluginView{}, err
	}
	for index, member := range ordered {
		if err := membership.Allows(member); err != nil {
			return PluginView{}, fmt.Errorf(
				"plugin members[%d]: %w", index, err,
			)
		}
	}
	view := PluginView{
		Artifact:    record.Clone(),
		Name:        spec.LogicalName(document.Name),
		DisplayName: displayName,
		Description: document.Description,
		Editable:    editable,
		Deletable:   deletable,
		Baseline:    baseline,
		Membership:  membership,
		Members:     make([]MemberReference, 0, len(ordered)),
		Entries:     make([]PluginMemberView, 0, len(ordered)),
	}
	for index, entry := range ordered {
		form, err := entry.MemberForm()
		if err != nil {
			return PluginView{}, fmt.Errorf(
				"plugin members[%d]: %w",
				index,
				err,
			)
		}
		header := entry.Header()
		member := PluginMemberView{
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
				return PluginView{}, err
			}
			member.Insert = insert
		}
		if form == declaration.MemberNamed {
			reference, err := memberReferenceFromEntry(entry)
			if err != nil {
				return PluginView{}, err
			}
			member.Server = reference.Server
			view.Members = append(view.Members, reference)
		}
		view.Entries = append(view.Entries, member)
	}
	return view, nil
}
