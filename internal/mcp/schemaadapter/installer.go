package schemaadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/mcp/bundle"
	"github.com/flexigpt/flexigpt-app/internal/mcp/overlay"
	"github.com/flexigpt/flexigpt-app/internal/mcp/policy"
	"github.com/flexigpt/flexigpt-app/internal/mcp/server"
)

// LoadEmbeddedRegistry loads only the source-controlled converted MCP
// registry. It intentionally does not inspect or convert legacy runtime data,
// overlays, secrets, or user state.
func LoadEmbeddedRegistry() (Registry, fs.FS, error) {
	packages, err := artifactbuiltin.EmbeddedMCPPackages()
	if err != nil {
		return Registry{}, nil, err
	}
	raw, err := artifactbuiltin.ReadEmbeddedMCPRegistry()
	if err != nil {
		return Registry{}, nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()

	var registry Registry
	if err := decoder.Decode(&registry); err != nil {
		return Registry{}, nil, fmt.Errorf(
			"decode embedded built-in MCP registry: %w",
			err,
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("embedded built-in MCP registry has trailing JSON")
		}
		return Registry{}, nil, err
	}
	if err := registry.Validate(); err != nil {
		return Registry{}, nil, err
	}

	return registry, packages, nil
}

type builtInBundleEnsurer interface {
	EnsureBuiltIn(
		ctx context.Context,
		request bundle.EnsureBuiltInRequest,
	) (bundle.Bundle, error)

	EnsureBuiltInCurrent(
		ctx context.Context,
		ref collection.CollectionRef,
	) error

	Get(
		ctx context.Context,
		ref collection.CollectionRef,
	) (bundle.Bundle, error)

	ListServers(
		ctx context.Context,
		ref collection.CollectionRef,
	) ([]artifact.Artifact, error)

	ListPolicies(
		ctx context.Context,
		ref collection.CollectionRef,
	) ([]artifact.Artifact, error)
}

type rootOverlayPurger interface {
	PurgeRoot(
		ctx context.Context,
		rootID basespec.RootID,
	) error
}

type InstallerDependencies struct {
	Bundles            builtInBundleEnsurer
	Registry           Registry
	Packages           fs.FS
	Overlays           overlay.OverlayRepository
	ShareableDocuments shareable.ExpectedCanonicalizer
}

type Installer struct {
	bundles         builtInBundleEnsurer
	registry        Registry
	builtInTopology topology.Declaration
	packages        fs.FS
	overlays        overlay.OverlayRepository
	documents       shareable.ExpectedCanonicalizer
	packageScopes   []basespec.Locator
}

type preparedBundle struct {
	registration   BundleRegistration
	document       bundle.BundleDocument
	parsed         shareable.ParsedDocument
	packageAddress source.ManagedPackageAddress
	packageFiles   []source.ManagedPackageFile
	packageDigest  cryptoutil.Digest
}

func NewInstaller(
	dependencies InstallerDependencies,
) (*Installer, error) {
	if dependencies.Bundles == nil ||
		dependencies.Packages == nil {
		return nil, fmt.Errorf(
			"%w: MCP built-in installer dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	if dependencies.ShareableDocuments == nil {
		return nil, fmt.Errorf(
			"%w: MCP built-in Artifact Store document canonicalizer is required",
			basespec.ErrInvalid,
		)
	}
	if err := dependencies.Registry.Validate(); err != nil {
		return nil, err
	}
	builtInTopology := artifactbuiltin.BuiltinTopologyDeclaration()
	if err := builtInTopology.Validate(); err != nil {
		return nil, err
	}
	installer := &Installer{
		bundles:         dependencies.Bundles,
		registry:        dependencies.Registry,
		builtInTopology: builtInTopology,
		packages:        dependencies.Packages,
		overlays:        dependencies.Overlays,
		documents:       dependencies.ShareableDocuments,
	}

	// Validate every embedded document and package path before exposing the
	// installer to bootstrap. This catches stale registry paths and filenames
	// before protected topology mutation starts.
	prepared, err := installer.prepareBundles(context.Background())
	if err != nil {
		return nil, err
	}
	scopes, err := builtInPackageScopes(prepared)
	if err != nil {
		return nil, err
	}
	installer.packageScopes = scopes
	return installer, nil
}

func (i *Installer) DesiredHydration(
	ctx context.Context,
) (topology.Hydration, error) {
	if i == nil {
		return topology.Hydration{}, basespec.ErrClosed
	}
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return topology.Hydration{}, err
	}

	prepared, err := i.prepareBundles(ctx)
	if err != nil {
		return topology.Hydration{}, err
	}
	fingerprint, err := i.hydrationFingerprint(prepared)
	if err != nil {
		return topology.Hydration{}, err
	}

	return topology.Hydration{
		InstallerName: i.BuiltInName(),
		RootID:        i.builtInTopology.Root.ID,
		SourceID:      i.builtInTopology.Sources[0].ID,
		Fingerprint:   fingerprint,
	}, nil
}

// EnsureHydration is called after generic topology hydration preparation.
// A stale fingerprint causes generic topology reset before this method is
// entered. MCP then removes local protected overlays for the reset Root and
// installs only source-controlled static registrations.
func (i *Installer) EnsureHydration(
	ctx context.Context,
	current bool,
) error {
	if i == nil {
		return basespec.ErrClosed
	}
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return err
	}
	if current {
		if err := i.FinalizeHydration(ctx); err == nil {
			return nil
		} else if ctx.Err() != nil {
			return err
		}
	}

	if !current {
		if purger, supported := i.overlays.(rootOverlayPurger); supported {
			if err := purger.PurgeRoot(
				ctx,
				i.builtInTopology.Root.ID,
			); err != nil {
				return fmt.Errorf(
					"purge stale MCP protected installation overlays: %w",
					err,
				)
			}
		}
	}

	prepared, err := i.prepareBundles(ctx)
	if err != nil {
		return err
	}
	for _, value := range prepared {
		if _, err := i.bundles.EnsureBuiltIn(
			ctx,
			bundle.EnsureBuiltInRequest{
				RootID:         i.builtInTopology.Root.ID,
				CollectionID:   value.registration.CollectionID,
				SourceID:       i.builtInTopology.Sources[0].ID,
				PackageAddress: value.packageAddress,
				Document:       value.parsed,
				Registrations:  value.registration.ToBundleRegistrations(),
				PackageFiles:   value.packageFiles,
			},
		); err != nil {
			return fmt.Errorf(
				"install protected MCP Bundle %q: %w",
				value.registration.CollectionID,
				err,
			)
		}
	}

	return i.ensureCurrentBundles(ctx)
}

// FinalizeHydration reconciles MCP catalogs after all installers sharing the
// protected managed Source have completed package publication.
//
// It intentionally does not write MCP packages. The generic bootstrap calls
// this after the primary install pass, which lets MCP and Skill catalogs
// converge on the same final Source revision and generation.
func (i *Installer) FinalizeHydration(
	ctx context.Context,
) error {
	if i == nil {
		return basespec.ErrClosed
	}
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return err
	}
	return i.ensureCurrentBundles(ctx)
}

func (*Installer) BuiltInName() string {
	return artifactbuiltin.MCPBuiltInInstallerName
}

// BuiltInIDs exposes only Collection and Artifact IDs. The generic bootstrap
// registry reserves the protected Root and shared Source IDs itself.
func (i *Installer) BuiltInIDs() []string {
	if i == nil {
		return nil
	}

	ids := make([]string, 0)
	for _, registered := range i.registry.OrderedBundles() {
		ids = append(ids, string(registered.CollectionID))
		for _, value := range registered.Artifacts {
			ids = append(ids, string(value.ID))
		}
	}
	sort.Strings(ids)
	return ids
}

func (i *Installer) BuiltInPackageScopes() []basespec.Locator {
	if i == nil {
		return nil
	}
	return append([]basespec.Locator(nil), i.packageScopes...)
}

// builtInPackageScopes returns destination managed-package roots, not
// embedded source-tree roots. Bootstrap scope checks must protect the shared
// managed Source namespace used after hydration.
func builtInPackageScopes(
	prepared []preparedBundle,
) ([]basespec.Locator, error) {
	scopes := make([]basespec.Locator, 0, len(prepared))
	for _, value := range prepared {
		scope, err := value.packageAddress.Directory()
		if err != nil {
			return nil, err
		}
		scopes = append(scopes, scope)
	}
	slices.Sort(scopes)
	return scopes, nil
}

// Ensure satisfies the generic installer contract. Hydration-aware bootstrap
// calls EnsureHydration instead and supplies the current marker state.
func (i *Installer) Ensure(ctx context.Context) error {
	return i.EnsureHydration(ctx, false)
}

func (i *Installer) ensureCurrentBundles(ctx context.Context) error {
	prepared, err := i.prepareBundles(ctx)
	if err != nil {
		return err
	}
	for _, expected := range prepared {
		if err := i.bundles.EnsureBuiltInCurrent(
			ctx,
			collection.CollectionRef{
				RootID:       i.builtInTopology.Root.ID,
				CollectionID: expected.registration.CollectionID,
			},
		); err != nil {
			return err
		}
		if err := i.verifyCurrentBundle(ctx, expected); err != nil {
			return err
		}
	}
	return nil
}

// verifyCurrentBundle proves that the current catalog is backed by the static
// Artifact registrations expected by the embedded MCP registry. Catalog
// freshness alone does not prove that a previous interrupted install retained
// every pinned server and policy Artifact.
func (i *Installer) verifyCurrentBundle(
	ctx context.Context,
	expected preparedBundle,
) error {
	ref := collection.CollectionRef{
		RootID:       i.builtInTopology.Root.ID,
		CollectionID: expected.registration.CollectionID,
	}
	current, err := i.bundles.Get(ctx, ref)
	if err != nil {
		return err
	}
	documentLocator, err := bundle.DocumentLocatorForPackage(
		expected.packageAddress,
	)
	if err != nil {
		return err
	}
	if current.Collection.RootID != ref.RootID ||
		current.Collection.ID != ref.CollectionID ||
		current.Source.ID != i.builtInTopology.Sources[0].ID ||
		current.PackageAddress != expected.packageAddress ||
		current.DocumentLocator != documentLocator {
		return fmt.Errorf(
			"%w: built-in MCP Bundle %q does not match static topology",
			basespec.ErrConflict,
			expected.registration.CollectionID,
		)
	}

	servers, err := i.bundles.ListServers(ctx, ref)
	if err != nil {
		return err
	}
	policies, err := i.bundles.ListPolicies(ctx, ref)
	if err != nil {
		return err
	}
	//nolint:gocritic // Need all records.
	records := append(servers, policies...)

	expectedByID := make(
		map[basespec.ArtifactID]ArtifactRegistration,
		len(expected.registration.Artifacts),
	)
	for _, registration := range expected.registration.Artifacts {
		expectedByID[registration.ID] = registration
	}
	if len(records) != len(expectedByID) {
		return fmt.Errorf(
			"%w: built-in MCP Bundle %q has %d Artifacts, expected %d",
			basespec.ErrConflict,
			expected.registration.CollectionID,
			len(records),
			len(expectedByID),
		)
	}

	seen := make(map[basespec.ArtifactID]struct{}, len(records))
	for _, record := range records {
		registration, found := expectedByID[record.ID]
		if !found {
			return fmt.Errorf(
				"%w: built-in MCP Bundle %q contains undeclared Artifact %q",
				basespec.ErrConflict,
				expected.registration.CollectionID,
				record.ID,
			)
		}
		seen[record.ID] = struct{}{}

		if record.RootID != ref.RootID ||
			record.CollectionID != ref.CollectionID ||
			record.Kind != registration.Kind ||
			record.Adoption != artifact.AdoptionPinned ||
			record.Enabled != registration.Enabled ||
			record.Binding.SourceID != current.Source.ID ||
			record.Binding.Locator != current.DocumentLocator ||
			record.Binding.SubresourceLocator != registration.Subresource ||
			record.Binding.ExpectedKind != registration.Kind ||
			record.State != artifact.StateAvailable ||
			record.ResolvedDefinition == nil {
			return fmt.Errorf(
				"%w: built-in MCP Artifact %q does not match static registry state",
				basespec.ErrReferenceUnresolved,
				record.ID,
			)
		}
	}

	for artifactID := range expectedByID {
		if _, found := seen[artifactID]; found {
			continue
		}
		return fmt.Errorf(
			"%w: built-in MCP Bundle %q is missing Artifact %q",
			basespec.ErrReferenceUnresolved,
			expected.registration.CollectionID,
			artifactID,
		)
	}
	return nil
}

func (i *Installer) prepareBundles(
	ctx context.Context,
) ([]preparedBundle, error) {
	if err := i.registry.Validate(); err != nil {
		return nil, err
	}

	output := make([]preparedBundle, 0, len(i.registry.Bundles))
	for _, registered := range i.registry.OrderedBundles() {
		embeddedFiles, err := topology.ReadPackageFiles(
			ctx,
			i.packages,
			registered.EmbeddedPackageRoot,
		)
		if err != nil {
			return nil, err
		}

		documentFile := basespec.Locator(
			path.Base(string(registered.EmbeddedDocumentLocator)),
		)
		var (
			raw           []byte
			foundDocument bool
		)
		for _, file := range embeddedFiles {
			if file.Locator != documentFile {
				continue
			}
			raw = append([]byte(nil), file.Content...)
			foundDocument = true
			break
		}
		if !foundDocument {
			return nil, fmt.Errorf(
				"%w: built-in MCP package %q lacks document %q",
				basespec.ErrInvalid,
				registered.EmbeddedPackageRoot,
				registered.EmbeddedDocumentLocator,
			)
		}

		parsed, err := i.documents.CanonicalizeExpected(
			ctx,
			artifactbuiltin.MCPBundleSchemaKey,
			raw,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"canonicalize built-in MCP document %q through the Artifact Store schema registry: %w",
				registered.EmbeddedDocumentLocator,
				err,
			)
		}

		document, err := bundle.BundleFromParsedDocument(parsed)
		if err != nil {
			return nil, fmt.Errorf(
				"project canonical built-in MCP document %q: %w",
				registered.EmbeddedDocumentLocator,
				err,
			)
		}
		if document.Digest != parsed.Digest {
			return nil, fmt.Errorf(
				"%w: built-in MCP document %q digest differs from schema registry output",
				basespec.ErrDigestMismatch,
				registered.EmbeddedDocumentLocator,
			)
		}
		packageAddress, err := bundle.PackageAddressForBundle(
			document.LogicalName,
			document.LogicalVersion,
		)
		if err != nil {
			return nil, err
		}
		packageFiles, packageDigest, err := canonicalPackageFiles(
			registered,
			packageAddress,
			embeddedFiles,
			parsed.Raw,
		)
		if err != nil {
			return nil, err
		}

		definitions, err := bundleDefinitions(document)
		if err != nil {
			return nil, err
		}
		if len(definitions) != len(registered.Artifacts) {
			return nil, fmt.Errorf(
				"%w: static MCP registration does not cover document %q",
				basespec.ErrInvalid,
				registered.EmbeddedDocumentLocator,
			)
		}
		for subresource, kind := range definitions {
			if !registeredHasKind(
				registered,
				subresource,
				kind,
			) {
				return nil, fmt.Errorf(
					"%w: static MCP registration lacks %q",
					basespec.ErrReferenceUnresolved,
					subresource,
				)
			}
		}

		output = append(output, preparedBundle{
			registration:   registered,
			document:       document,
			parsed:         parsed.Clone(),
			packageAddress: packageAddress,
			packageFiles:   packageFiles,
			packageDigest:  packageDigest,
		})
	}
	return output, nil
}

func canonicalPackageFiles(
	registered BundleRegistration,
	address source.ManagedPackageAddress,
	embeddedFiles []topology.PackageFile,
	canonicalDocument json.RawMessage,
) ([]source.ManagedPackageFile, cryptoutil.Digest, error) {
	documentFile := basespec.Locator(
		path.Base(string(registered.EmbeddedDocumentLocator)),
	)
	files := make([]source.ManagedPackageFile, len(embeddedFiles))
	foundDocument := false
	for index, file := range embeddedFiles {
		files[index] = source.ManagedPackageFile{
			Locator: file.Locator,
			Content: append([]byte(nil), file.Content...),
		}
		if file.Locator == documentFile {
			files[index].Content = append([]byte(nil), canonicalDocument...)
			foundDocument = true
		}
	}
	if !foundDocument {
		return nil, "", fmt.Errorf(
			"%w: built-in MCP package %q lacks %q",
			basespec.ErrInvalid,
			registered.EmbeddedPackageRoot,
			documentFile,
		)
	}

	publication, err := source.NormalizeManagedPackagePublication(
		source.ManagedPackagePublication{
			Address: address,
			Files:   files,
		},
	)
	if err != nil {
		return nil, "", err
	}

	type fingerprintFile struct {
		Locator basespec.Locator  `json:"locator"`
		Digest  cryptoutil.Digest `json:"digest"`
		Size    int64             `json:"size"`
	}
	fingerprint := make(
		[]fingerprintFile,
		0,
		len(publication.Files),
	)
	for _, file := range publication.Files {
		fingerprint = append(fingerprint, fingerprintFile{
			Locator: file.Locator,
			Digest:  cryptoutil.DigestBytes(file.Content),
			Size:    int64(len(file.Content)),
		})
	}
	raw, err := json.Marshal(fingerprint)
	if err != nil {
		return nil, "", err
	}
	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return nil, "", err
	}
	return publication.Files, cryptoutil.DigestBytes(canonical), nil
}

func (i *Installer) hydrationFingerprint(
	prepared []preparedBundle,
) (cryptoutil.Digest, error) {
	type artifactFingerprint struct {
		ID          basespec.ArtifactID         `json:"id"`
		Subresource basespec.SubresourceLocator `json:"subresource"`
		Kind        basespec.ArtifactKind       `json:"kind"`
		Enabled     bool                        `json:"enabled"`
	}
	type bundleFingerprint struct {
		CollectionID    basespec.CollectionID        `json:"collectionID"`
		PackageAddress  source.ManagedPackageAddress `json:"packageAddress"`
		DocumentLocator basespec.Locator             `json:"embeddedDocumentLocator"`
		DocumentDigest  cryptoutil.Digest            `json:"documentDigest"`
		PackageDigest   cryptoutil.Digest            `json:"packageDigest"`
		Artifacts       []artifactFingerprint        `json:"artifacts"`
	}

	values := make([]bundleFingerprint, 0, len(prepared))
	for _, value := range prepared {
		artifacts := make(
			[]artifactFingerprint,
			0,
			len(value.registration.Artifacts),
		)
		for _, artifactValue := range value.registration.Artifacts {
			artifacts = append(artifacts, artifactFingerprint(artifactValue))
		}
		sort.Slice(artifacts, func(left, right int) bool {
			return artifacts[left].ID < artifacts[right].ID
		})
		values = append(values, bundleFingerprint{
			CollectionID:    value.registration.CollectionID,
			PackageAddress:  value.packageAddress,
			DocumentLocator: value.registration.EmbeddedDocumentLocator,
			DocumentDigest:  value.document.Digest,
			PackageDigest:   value.packageDigest,
			Artifacts:       artifacts,
		})
	}

	raw, err := json.Marshal(struct {
		SchemaVersion string               `json:"schemaVersion"`
		Topology      topology.Declaration `json:"topology"`
		Bundles       []bundleFingerprint  `json:"bundles"`
	}{
		SchemaVersion: artifactbuiltin.MCPBundleHydrationFingerprintSchemaVersion,
		Topology:      i.builtInTopology,
		Bundles:       values,
	})
	if err != nil {
		return "", err
	}
	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return "", err
	}
	return cryptoutil.DigestBytes(canonical), nil
}

func bundleDefinitions(
	document bundle.BundleDocument,
) (map[basespec.SubresourceLocator]basespec.ArtifactKind, error) {
	output := make(
		map[basespec.SubresourceLocator]basespec.ArtifactKind,
		len(document.MCPServers)+len(document.BundleExtension.Policies),
	)
	for name := range document.MCPServers {
		output[server.ServerSubresource(
			basespec.LogicalName(name),
		)] = artifactbuiltin.ServerKind
	}
	for name := range document.BundleExtension.Policies {
		output[policy.PolicySubresource(
			basespec.LogicalName(name),
		)] = artifactbuiltin.PolicyKind
	}
	return output, nil
}

func registeredHasKind(
	registered BundleRegistration,
	subresource basespec.SubresourceLocator,
	kind basespec.ArtifactKind,
) bool {
	for _, value := range registered.Artifacts {
		if value.Subresource == subresource && value.Kind == kind {
			return true
		}
	}
	return false
}
