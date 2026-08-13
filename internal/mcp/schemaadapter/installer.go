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

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/topology"
	"github.com/flexigpt/flexigpt-app/internal/builtin/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/mcp/bundle"
	"github.com/flexigpt/flexigpt-app/internal/mcp/overlay"
)

// LoadEmbeddedRegistry loads only the source-controlled converted MCP
// registry. It intentionally does not inspect or convert legacy runtime data,
// overlays, secrets, or user state.
func LoadEmbeddedRegistry() (Registry, fs.FS, error) {
	packages, err := schema.EmbeddedMCPArtifactPackages()
	if err != nil {
		return Registry{}, nil, err
	}
	raw, err := schema.ReadEmbeddedMCPRegistry()
	if err != nil {
		return Registry{}, nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()

	var registry Registry
	if err := decoder.Decode(&registry); err != nil {
		return Registry{}, nil, fmt.Errorf(
			"decode converted built-in MCP registry: %w",
			err,
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("converted built-in MCP registry has trailing JSON")
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
	builtInTopology := schema.BuiltinTopologyDeclaration()
	if err := builtInTopology.Validate(); err != nil {
		return nil, err
	}

	return &Installer{
		bundles:         dependencies.Bundles,
		registry:        dependencies.Registry,
		builtInTopology: builtInTopology,
		packages:        dependencies.Packages,
		overlays:        dependencies.Overlays,
		documents:       dependencies.ShareableDocuments,
	}, nil
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
		if err := i.ensureCurrentBundles(ctx); err == nil {
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
	return nil
}

func (*Installer) BuiltInName() string {
	return "mcp.bundle"
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

	scopes := make([]basespec.Locator, 0, len(i.registry.Bundles))
	for _, registered := range i.registry.OrderedBundles() {
		scopes = append(scopes, registered.PackageDirectory)
	}
	slices.Sort(scopes)
	return scopes
}

// Ensure satisfies the generic installer contract. Hydration-aware bootstrap
// calls EnsureHydration instead and supplies the current marker state.
func (i *Installer) Ensure(ctx context.Context) error {
	return i.EnsureHydration(ctx, false)
}

func (i *Installer) ensureCurrentBundles(ctx context.Context) error {
	for _, registered := range i.registry.OrderedBundles() {
		if err := i.bundles.EnsureBuiltInCurrent(
			ctx,
			collection.CollectionRef{
				RootID:       i.builtInTopology.Root.ID,
				CollectionID: registered.CollectionID,
			},
		); err != nil {
			return err
		}
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
			registered.PackageDirectory,
		)
		if err != nil {
			return nil, err
		}

		documentFile := basespec.Locator(
			path.Base(string(registered.DocumentLocator)),
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
				registered.PackageDirectory,
				registered.DocumentLocator,
			)
		}

		parsed, err := i.documents.CanonicalizeExpected(
			ctx,
			bundle.BundleCodec{}.Key(),
			raw,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"canonicalize built-in MCP document %q through the Artifact Store schema registry: %w",
				registered.DocumentLocator,
				err,
			)
		}

		document, err := bundle.BundleFromParsedDocument(parsed)
		if err != nil {
			return nil, fmt.Errorf(
				"project canonical built-in MCP document %q: %w",
				registered.DocumentLocator,
				err,
			)
		}
		if document.Digest != parsed.Digest {
			return nil, fmt.Errorf(
				"%w: built-in MCP document %q digest differs from schema registry output",
				basespec.ErrDigestMismatch,
				registered.DocumentLocator,
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
				registered.DocumentLocator,
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
		path.Base(string(registered.DocumentLocator)),
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
			registered.PackageDirectory,
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
			DocumentLocator: value.registration.DocumentLocator,
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
		SchemaVersion: schema.HydrationFingerprintSchemaVersion,
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
		output[basespec.SubresourceLocator(
			"mcpServers/"+name,
		)] = schema.ServerKind
	}
	for name := range document.BundleExtension.Policies {
		output[basespec.SubresourceLocator(
			"policies/"+name,
		)] = schema.PolicyKind
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
