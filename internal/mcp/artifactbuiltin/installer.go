package artifactbuiltin

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/mcp/bundle"
	"github.com/flexigpt/flexigpt-app/internal/mcp/installation"
	"github.com/flexigpt/flexigpt-app/internal/mcp/schema"
)

const hydrationFingerprintSchemaVersion = "mcp.builtin-hydration/v1"

type protectedTopologyEnsurer interface {
	EnsureProtectedTopology(
		ctx context.Context,
		declaration topology.Declaration,
	) (topology.Installed, error)
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
	Topology protectedTopologyEnsurer
	Bundles  builtInBundleEnsurer
	Registry Registry
	Packages fs.FS
	Overlays installation.OverlayRepository
}

type Installer struct {
	topology protectedTopologyEnsurer
	bundles  builtInBundleEnsurer
	registry Registry
	packages fs.FS
	overlays installation.OverlayRepository
}

type preparedBundle struct {
	registration BundleRegistration
	document     schema.BundleDocument
}

func NewInstaller(
	dependencies InstallerDependencies,
) (*Installer, error) {
	if dependencies.Topology == nil ||
		dependencies.Bundles == nil ||
		dependencies.Packages == nil {
		return nil, fmt.Errorf(
			"%w: MCP built-in installer dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	if err := dependencies.Registry.Validate(); err != nil {
		return nil, err
	}

	return &Installer{
		topology: dependencies.Topology,
		bundles:  dependencies.Bundles,
		registry: dependencies.Registry,
		packages: dependencies.Packages,
		overlays: dependencies.Overlays,
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
		RootID:        i.registry.Topology.Root.ID,
		SourceID:      i.registry.Topology.Sources[0].ID,
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
		for _, registered := range i.registry.OrderedBundles() {
			if err := i.bundles.EnsureBuiltInCurrent(
				ctx,
				collection.CollectionRef{
					RootID:       i.registry.Topology.Root.ID,
					CollectionID: registered.CollectionID,
				},
			); err != nil {
				return err
			}
		}
		return nil
	}

	if purger, supported := i.overlays.(rootOverlayPurger); supported {
		if err := purger.PurgeRoot(
			ctx,
			i.registry.Topology.Root.ID,
		); err != nil {
			return fmt.Errorf(
				"purge stale MCP protected installation overlays: %w",
				err,
			)
		}
	}

	prepared, err := i.prepareBundles(ctx)
	if err != nil {
		return err
	}

	installed, err := i.topology.EnsureProtectedTopology(
		ctx,
		i.registry.Topology,
	)
	if err != nil {
		return err
	}
	if !containsInstalledSource(
		installed,
		i.registry.Topology.Sources[0].ID,
	) {
		return fmt.Errorf(
			"%w: protected MCP topology did not install its managed Source",
			basespec.ErrInvalid,
		)
	}

	for _, value := range prepared {
		if _, err := i.bundles.EnsureBuiltIn(
			ctx,
			bundle.EnsureBuiltInRequest{
				RootID:          i.registry.Topology.Root.ID,
				CollectionID:    value.registration.CollectionID,
				SourceID:        i.registry.Topology.Sources[0].ID,
				DocumentLocator: value.registration.DocumentLocator(),
				Document:        value.document,
				Registrations:   value.registration.ToBundleRegistrations(),
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

func (i *Installer) prepareBundles(
	ctx context.Context,
) ([]preparedBundle, error) {
	if err := i.registry.Validate(); err != nil {
		return nil, err
	}

	output := make([]preparedBundle, 0, len(i.registry.Bundles))
	for _, registered := range i.registry.OrderedBundles() {
		raw, err := fs.ReadFile(
			i.packages,
			string(registered.DocumentLocator()),
		)
		if err != nil {
			return nil, fmt.Errorf(
				"read built-in MCP document %q: %w",
				registered.DocumentLocator(),
				err,
			)
		}
		document, _, err := schema.ParseBundle(raw)
		if err != nil {
			return nil, fmt.Errorf(
				"canonicalize built-in MCP document %q: %w",
				registered.DocumentLocator(),
				err,
			)
		}

		definitions, err := bundleDefinitions(document)
		if err != nil {
			return nil, err
		}
		if len(definitions) != len(registered.Artifacts) {
			return nil, fmt.Errorf(
				"%w: static MCP registration does not cover document %q",
				basespec.ErrInvalid,
				registered.DocumentLocator(),
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
			registration: registered,
			document:     document,
		})
	}
	return output, nil
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
		CollectionID     basespec.CollectionID `json:"collectionID"`
		PackageDirectory basespec.Locator      `json:"packageDirectory"`
		DocumentDigest   cryptoutil.Digest     `json:"documentDigest"`
		Artifacts        []artifactFingerprint `json:"artifacts"`
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
			CollectionID:     value.registration.CollectionID,
			PackageDirectory: value.registration.PackageDirectory,
			DocumentDigest:   value.document.Digest,
			Artifacts:        artifacts,
		})
	}

	raw, err := json.Marshal(struct {
		SchemaVersion string               `json:"schemaVersion"`
		Topology      topology.Declaration `json:"topology"`
		Bundles       []bundleFingerprint  `json:"bundles"`
	}{
		SchemaVersion: hydrationFingerprintSchemaVersion,
		Topology:      i.registry.Topology,
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

func containsInstalledSource(
	installed topology.Installed,
	sourceID basespec.SourceID,
) bool {
	for _, value := range installed.Sources {
		if value.ID == sourceID {
			return true
		}
	}
	return false
}

func bundleDefinitions(
	document schema.BundleDocument,
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
