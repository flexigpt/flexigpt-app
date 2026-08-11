package bundle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/managedartifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/managed"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpArtifact "github.com/flexigpt/flexigpt-app/internal/mcp/artifact"
	"github.com/flexigpt/flexigpt-app/internal/mcp/installation"
	"github.com/flexigpt/flexigpt-app/internal/mcp/schema"
)

type noAutomaticAdoption struct{}

func (noAutomaticAdoption) Derive(
	_ context.Context,
	_ collection.Collection,
	_ catalog.Occurrence,
	_ definition.Definition,
) (
	artifact.Draft,
	bool,
	[]diagnostic.Diagnostic,
	error,
) {
	return artifact.Draft{}, false, nil, nil
}

type Dependencies struct {
	Roots            *root.Service
	Sources          *source.Service
	Collections      *collection.Service
	Artifacts        *artifact.Service
	ManagedArtifacts *managedartifact.Service
	Refresh          refresh.Runner

	Catalogs      catalog.Reader
	Definitions   definition.Reader
	SourceRuntime source.Runtime

	HasDecoder         func(basespec.DecoderID) bool
	DecoderFingerprint func() (cryptoutil.Digest, error)

	RootPolicy    protection.RootPolicy
	UserRootID    basespec.RootID
	Runtime       RuntimeInvalidator
	Overlays      installation.OverlayRepository
	SecretCleaner installation.SecretCleaner

	BaselinePolicy schema.PolicyBody
}

type API struct {
	dependencies Dependencies
}

func New(dependencies Dependencies) (*API, error) {
	if dependencies.Roots == nil ||
		dependencies.Sources == nil ||
		dependencies.Collections == nil ||
		dependencies.Artifacts == nil ||
		dependencies.ManagedArtifacts == nil ||
		dependencies.Refresh == nil ||
		dependencies.Catalogs == nil ||
		dependencies.Definitions == nil ||
		dependencies.SourceRuntime == nil ||
		dependencies.HasDecoder == nil ||
		dependencies.DecoderFingerprint == nil ||
		dependencies.Runtime == nil ||
		dependencies.RootPolicy == nil ||
		dependencies.SecretCleaner == nil {
		return nil, fmt.Errorf(
			"%w: MCP Bundle dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	if !dependencies.HasDecoder(mcpArtifact.DecoderID) {
		return nil, fmt.Errorf(
			"%w: MCP decoder %q is not registered",
			basespec.ErrDecoderUnavailable,
			mcpArtifact.DecoderID,
		)
	}
	if dependencies.UserRootID != "" {
		if err := basespec.ValidateRootID(dependencies.UserRootID); err != nil {
			return nil, err
		}
	}
	return &API{dependencies: dependencies}, nil
}

type Bundle struct {
	Collection collection.Collection
	Data       CollectionData
	Attachment collection.Attachment
	Source     source.Summary

	DocumentLocator  basespec.Locator
	PackageDirectory basespec.Locator
}

type Registration struct {
	ArtifactID  basespec.ArtifactID
	Subresource basespec.SubresourceLocator
	Kind        basespec.ArtifactKind
	Enabled     bool
	Data        json.RawMessage
}

type CreateRequest struct {
	RootID          basespec.RootID
	CollectionID    basespec.CollectionID
	SourceID        basespec.SourceID
	DocumentLocator basespec.Locator

	Document      schema.BundleDocument
	Registrations []Registration
}

func (a *API) Create(
	ctx context.Context,
	request CreateRequest,
) (Bundle, error) {
	if a == nil {
		return Bundle{}, basespec.ErrClosed
	}
	if err := basespec.ValidateRootID(request.RootID); err != nil {
		return Bundle{}, err
	}
	if a.dependencies.UserRootID != "" &&
		request.RootID != a.dependencies.UserRootID {
		return Bundle{}, fmt.Errorf(
			"%w: user MCP Bundles must be created in Root %q",
			basespec.ErrInvalid,
			a.dependencies.UserRootID,
		)
	}
	if err := a.requireBundleMutation(
		ctx,
		request.RootID,
		false,
	); err != nil {
		return Bundle{}, err
	}
	if err := basespec.ValidateCollectionID(request.CollectionID); err != nil {
		return Bundle{}, err
	}
	if err := basespec.ValidateSourceID(request.SourceID); err != nil {
		return Bundle{}, err
	}

	documentLocator := request.DocumentLocator
	if documentLocator == "" {
		documentLocator = DefaultDocumentLocator
	}
	if err := ValidateDocumentLocator(documentLocator); err != nil {
		return Bundle{}, err
	}

	document, _, err := schema.CanonicalizeBundle(request.Document)
	if err != nil {
		return Bundle{}, err
	}
	if err := validateCreateRegistrations(
		request.RootID,
		document,
		request.Registrations,
	); err != nil {
		return Bundle{}, err
	}

	sourceValue, createdSource, err := a.dependencies.Sources.CreateWithStatus(
		ctx,
		request.RootID,
		source.Draft{
			ID:          request.SourceID,
			Kind:        managed.Kind,
			DisplayName: displayName(document),
			Enabled:     true,
			Config:      json.RawMessage(jsonutil.EmptyObject),
		},
	)
	if err != nil {
		return Bundle{}, err
	}
	cleanupSource := func(cause error) error {
		if !createdSource {
			return cause
		}
		return errors.Join(
			cause,
			a.dependencies.Sources.Discard(
				context.WithoutCancel(ctx),
				request.RootID,
				request.SourceID,
				sourceValue.Revision,
			),
		)
	}

	collectionData, err := EncodeCollectionData(CollectionData{
		SchemaVersion:           CollectionDataSchemaVersion,
		DiscoveryPolicyRevision: DiscoveryPolicyRevision,
		LogicalName:             document.LogicalName,
		LogicalVersion:          document.LogicalVersion,
		Labels:                  maps.Clone(document.Labels),
		ManagedSourceID:         request.SourceID,
	})
	if err != nil {
		return Bundle{}, cleanupSource(err)
	}
	attachmentData, err := EncodeAttachmentData(AttachmentData{
		SchemaVersion:   AttachmentDataSchemaVersion,
		DocumentLocator: documentLocator,
	})
	if err != nil {
		return Bundle{}, cleanupSource(err)
	}

	created, _, err := a.dependencies.Collections.Create(
		ctx,
		request.RootID,
		collection.Draft{
			ID:          request.CollectionID,
			Kind:        schema.BundleKind,
			DisplayName: displayName(document),
			Description: document.Description,
			Enabled:     true,
			Data:        collectionData,
		},
		[]collection.AttachmentDraft{{
			SourceID: request.SourceID,
			Role:     RoleManaged,
			Enabled:  true,
			Data:     attachmentData,
		}},
	)
	if err != nil {
		return Bundle{}, cleanupSource(err)
	}

	bundle, err := a.Get(ctx, created.Ref())
	if err != nil {
		return Bundle{}, err
	}
	if err := validateCreateBundleIntent(
		bundle,
		request,
		document,
		documentLocator,
	); err != nil {
		return Bundle{}, cleanupSource(err)
	}
	if _, err := a.ReplaceDocument(
		ctx,
		ReplaceDocumentRequest{
			Bundle:                     bundle.Collection.Ref(),
			ExpectedCollectionRevision: bundle.Collection.Revision,
			Document:                   document,
			Registrations:              request.Registrations,
			AllowProtected:             false,
		},
	); err != nil {
		return Bundle{}, err
	}
	return a.Get(ctx, created.Ref())
}

func (a *API) List(
	ctx context.Context,
	rootID basespec.RootID,
) ([]Bundle, error) {
	values, err := a.dependencies.Collections.ListByRoot(ctx, rootID)
	if err != nil {
		return nil, err
	}
	output := make([]Bundle, 0)
	for _, value := range values {
		if value.Kind != schema.BundleKind {
			continue
		}
		bundle, err := a.Get(ctx, value.Ref())
		if err != nil {
			return nil, err
		}
		output = append(output, bundle)
	}
	sort.Slice(output, func(left, right int) bool {
		return output[left].Collection.ID <
			output[right].Collection.ID
	})
	return output, nil
}

// Refresh performs an explicit MCP configuration refresh. The runtime session
// is invalidated before publication so a live client cannot continue using
// configuration that the caller explicitly asked to re-evaluate.
func (a *API) Refresh(
	ctx context.Context,
	ref collection.CollectionRef,
	allowProtected bool,
) (Bundle, error) {
	if a == nil {
		return Bundle{}, basespec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return Bundle{}, err
	}
	if err := a.requireBundleMutation(
		ctx,
		ref.RootID,
		allowProtected,
	); err != nil {
		return Bundle{}, err
	}

	bundle, err := a.Get(ctx, ref)
	if err != nil {
		return Bundle{}, err
	}
	if !bundle.Collection.Enabled {
		return Bundle{}, fmt.Errorf(
			"%w: MCP Bundle %q is disabled",
			basespec.ErrConflict,
			ref.CollectionID,
		)
	}

	if err := a.dependencies.Runtime.InvalidateCollection(ctx, ref); err != nil {
		return Bundle{}, err
	}

	if _, err := a.dependencies.Refresh.Refresh(
		ctx,
		ref,
		a.discoveryPlan(bundle),
		noAutomaticAdoption{},
	); err != nil {
		return Bundle{}, err
	}
	return a.Get(ctx, ref)
}

// EnsureBuiltInCurrent avoids managed package republishing for a current
// protected Bundle, but repairs a missing or stale Catalog after startup,
// decoder changes, or interrupted prior work.
func (a *API) EnsureBuiltInCurrent(
	ctx context.Context,
	ref collection.CollectionRef,
) error {
	if a == nil {
		return basespec.ErrClosed
	}
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return err
	}
	if err := ref.Validate(); err != nil {
		return err
	}
	if !a.dependencies.RootPolicy.IsProtectedRoot(ref.RootID) {
		return fmt.Errorf(
			"%w: MCP Bundle %q is not protected",
			basespec.ErrProtected,
			ref.CollectionID,
		)
	}

	bundle, err := a.Get(ctx, ref)
	if err != nil {
		return err
	}
	if _, err := a.currentCatalog(ctx, bundle); err == nil {
		return nil
	} else if !errors.Is(err, basespec.ErrCatalogUnavailable) &&
		!errors.Is(err, basespec.ErrCatalogStale) {
		return err
	}

	_, err = a.Refresh(ctx, ref, true)
	return err
}

func (a *API) Get(
	ctx context.Context,
	ref collection.CollectionRef,
) (Bundle, error) {
	if a == nil {
		return Bundle{}, basespec.ErrClosed
	}
	value, err := a.dependencies.Collections.Get(ctx, ref)
	if err != nil {
		return Bundle{}, err
	}
	if value.Kind != schema.BundleKind {
		return Bundle{}, fmt.Errorf(
			"%w: Collection %q is not an MCP Bundle",
			basespec.ErrCollectionNotFound,
			ref.CollectionID,
		)
	}

	data, err := DecodeCollectionData(value.Data)
	if err != nil {
		return Bundle{}, err
	}
	attachments, err := a.dependencies.Collections.ListAttachments(
		ctx,
		ref,
	)
	if err != nil {
		return Bundle{}, err
	}
	if len(attachments) != 1 {
		return Bundle{}, fmt.Errorf(
			"%w: MCP Bundle must have exactly one Source Attachment",
			basespec.ErrInvalid,
		)
	}
	attachment := attachments[0]
	if attachment.Role != RoleManaged &&
		attachment.Role != RoleBuiltIn {
		return Bundle{}, fmt.Errorf(
			"%w: unsupported MCP Attachment role %q",
			basespec.ErrInvalid,
			attachment.Role,
		)
	}
	attachmentData, err := DecodeAttachmentData(attachment.Data)
	if err != nil {
		return Bundle{}, err
	}
	packageDirectory, err := PackageDirectoryForDocument(
		attachmentData.DocumentLocator,
	)
	if err != nil {
		return Bundle{}, err
	}
	sourceValue, err := a.dependencies.Sources.Get(
		ctx,
		ref.RootID,
		attachment.SourceID,
	)
	if err != nil {
		return Bundle{}, err
	}
	if sourceValue.Kind != managed.Kind {
		return Bundle{}, fmt.Errorf(
			"%w: MCP Bundle requires a managed Source",
			basespec.ErrInvalid,
		)
	}
	if data.ManagedSourceID != "" &&
		data.ManagedSourceID != sourceValue.ID {
		return Bundle{}, fmt.Errorf(
			"%w: MCP Bundle managed Source ownership mismatch",
			basespec.ErrInvalid,
		)
	}

	return Bundle{
		Collection: value,
		Data:       data,
		Attachment: attachment,
		Source:     sourceValue,

		DocumentLocator:  attachmentData.DocumentLocator,
		PackageDirectory: packageDirectory,
	}, nil
}

func (a *API) discoveryPlan(
	bundle Bundle,
) discovery.Plan {
	p := discovery.SourcePlan{
		SourceID:         bundle.Source.ID,
		ExplicitLocators: []basespec.Locator{bundle.DocumentLocator},
		DecoderHints: []discovery.DecoderHint{{
			Locator:   bundle.DocumentLocator,
			Recursive: false,
			DecoderIDs: []basespec.DecoderID{
				mcpArtifact.DecoderID,
			},
		}},
		AllowedDecoderIDs: []basespec.DecoderID{
			mcpArtifact.DecoderID,
		},
		Authoritative: true,
	}.Normalized()
	return discovery.Plan{
		Revision: DiscoveryPolicyRevision,
		Sources:  []discovery.SourcePlan{p},
	}
}

func (a *API) cleanupChangedServerInstallation(
	ctx context.Context,
	record artifact.Artifact,
	before installation.ServerData,
	after installation.ServerData,
) error {
	if record.Kind != schema.ServerKind {
		return nil
	}
	if err := installation.CleanupReplacedServerSecrets(
		ctx,
		record.Ref(),
		before,
		after,
		a.dependencies.SecretCleaner,
	); err != nil {
		return fmt.Errorf(
			"MCP server installation secret cleanup remains pending: %w",
			err,
		)
	}
	return nil
}

func (a *API) cleanupRemovedServerInstallation(
	ctx context.Context,
	record artifact.Artifact,
) error {
	if record.Kind != schema.ServerKind {
		return nil
	}

	data, err := a.serverInstallationDataForCleanup(ctx, record)
	if err != nil {
		return err
	}
	if err := installation.CleanupRemovedServerSecrets(
		ctx,
		record.Ref(),
		data,
		a.dependencies.SecretCleaner,
	); err != nil {
		return fmt.Errorf(
			"MCP server removal secret cleanup remains pending: %w",
			err,
		)
	}
	return nil
}

func (a *API) serverInstallationDataForCleanup(
	ctx context.Context,
	record artifact.Artifact,
) (installation.ServerData, error) {
	if record.Kind != schema.ServerKind {
		return installation.DefaultServerData(), nil
	}

	if !a.dependencies.RootPolicy.IsProtectedRoot(record.RootID) {
		return installation.DecodeServerData(record.Data)
	}
	if a.dependencies.Overlays == nil {
		return installation.ServerData{}, fmt.Errorf(
			"%w: protected MCP overlay store is unavailable",
			basespec.ErrReferenceUnresolved,
		)
	}

	overlay, found, err := a.dependencies.Overlays.GetServerOverlay(
		ctx,
		record.Ref(),
	)
	if err != nil {
		return installation.ServerData{}, err
	}
	if !found {
		return installation.DefaultServerData(), nil
	}
	return overlay.ServerData, nil
}

// validateCreateRegistrations establishes all request-derived valid state
// before source or Collection mutation begins.
func validateCreateRegistrations(
	rootID basespec.RootID,
	document schema.BundleDocument,
	registrations []Registration,
) error {
	definitions, err := definitionsForDocument(document)
	if err != nil {
		return err
	}
	values, err := registrationMap(registrations, definitions)
	if err != nil {
		return err
	}

	subresources := make([]basespec.SubresourceLocator, 0, len(values))
	for subresource := range values {
		subresources = append(subresources, subresource)
	}
	slices.Sort(subresources)

	for _, subresource := range subresources {
		registration := values[subresource]
		if registration.Kind != schema.ServerKind {
			continue
		}

		data, err := registrationData(registration)
		if err != nil {
			return err
		}
		serverDefinition, err := serverDocumentFromDefinition(
			definitions[subresource],
		)
		if err != nil {
			return err
		}
		serverData, err := installation.DecodeServerData(data)
		if err != nil {
			return err
		}
		if err := installation.ValidateServerDataForDocument(
			artifact.ArtifactRef{
				RootID:     rootID,
				ArtifactID: registration.ArtifactID,
			},
			serverDefinition,
			serverData,
		); err != nil {
			return err
		}
	}
	return nil
}

func definitionsForDocument(
	document schema.BundleDocument,
) (
	map[basespec.SubresourceLocator]definition.Definition,
	error,
) {
	output := make(
		map[basespec.SubresourceLocator]definition.Definition,
		len(document.MCPServers)+len(document.BundleExtension.Policies),
	)
	for name := range document.MCPServers {
		serverDocument, err := schema.ServerFromBundle(document, name)
		if err != nil {
			return nil, err
		}
		value, err := mcpArtifact.DefinitionForServer(serverDocument)
		if err != nil {
			return nil, err
		}
		output[mcpArtifact.ServerSubresource(
			basespec.LogicalName(name),
		)] = value
	}
	for name, policyDocument := range document.BundleExtension.Policies {
		value, err := mcpArtifact.DefinitionForPolicy(policyDocument)
		if err != nil {
			return nil, err
		}
		output[mcpArtifact.PolicySubresource(
			basespec.LogicalName(name),
		)] = value
	}
	return output, nil
}

func validateCreateBundleIntent(
	value Bundle,
	request CreateRequest,
	document schema.BundleDocument,
	documentLocator basespec.Locator,
) error {
	if value.Collection.RootID != request.RootID ||
		value.Collection.ID != request.CollectionID ||
		value.Collection.Kind != schema.BundleKind ||
		value.Source.ID != request.SourceID ||
		value.DocumentLocator != documentLocator ||
		value.Data.ManagedSourceID != request.SourceID ||
		value.Data.LogicalName != document.LogicalName ||
		value.Data.LogicalVersion != document.LogicalVersion ||
		!maps.Equal(value.Data.Labels, document.Labels) {
		return fmt.Errorf(
			"%w: MCP Bundle creation intent differs from existing state",
			basespec.ErrConflict,
		)
	}
	return nil
}

func displayName(document schema.BundleDocument) string {
	if document.DisplayName != "" {
		return document.DisplayName
	}
	return string(document.LogicalName)
}
