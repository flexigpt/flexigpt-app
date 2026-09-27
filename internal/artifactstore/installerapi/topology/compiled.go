package topology

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const CompiledPackageSetFormat = "artifact-compiled-package-set/v1"

// CompiledCatalog is binary-owned generated built-in metadata.
//
// It is not user input. Full package, schema, decoder, and semantic
// validation happens during generation and is proven by the reproducibility
// test. Runtime treats values from this type as trusted program constants.
type CompiledCatalog struct {
	ValidationFingerprint cryptoutil.Digest    `json:"validationFingerprint"`
	Sets                  []CompiledPackageSet `json:"sets"`
}

func (c CompiledCatalog) Clone() CompiledCatalog {
	output := c
	output.Sets = make([]CompiledPackageSet, len(c.Sets))
	for index, value := range c.Sets {
		output.Sets[index] = value.Clone()
	}
	return output
}

type CompiledPackageSet struct {
	Format    string            `json:"format"`
	Name      string            `json:"name"`
	Hydration Hydration         `json:"hydration"`
	Packages  []CompiledPackage `json:"packages"`
}

func (s CompiledPackageSet) Clone() CompiledPackageSet {
	output := s
	output.Packages = make([]CompiledPackage, len(s.Packages))
	for index, value := range s.Packages {
		output.Packages[index] = value.Clone()
	}
	return output
}

type CompiledPackage struct {
	EmbeddedRoot basespec.Locator             `json:"embeddedRoot"`
	Address      source.ManagedPackageAddress `json:"address"`
	Fingerprint  cryptoutil.Digest            `json:"fingerprint"`

	Files     []CompiledFile     `json:"files"`
	Documents []CompiledDocument `json:"documents"`
}

func (p CompiledPackage) Clone() CompiledPackage {
	output := p
	output.Files = make([]CompiledFile, len(p.Files))
	for index, file := range p.Files {
		output.Files[index] = file
		output.Files[index].Content = append([]byte(nil), file.Content...)
	}
	output.Documents = make([]CompiledDocument, len(p.Documents))
	for index, value := range p.Documents {
		output.Documents[index] = value.Clone()
	}
	return output
}

type CompiledFile struct {
	Locator basespec.Locator  `json:"locator"`
	Size    int64             `json:"size"`
	Digest  cryptoutil.Digest `json:"digest"`
	Content []byte            `json:"content"`
}

type CompiledDocument struct {
	Locator   basespec.Locator   `json:"locator"`
	Digest    cryptoutil.Digest  `json:"digest"`
	Artifacts []CompiledArtifact `json:"artifacts"`
}

func (d CompiledDocument) Clone() CompiledDocument {
	output := d
	output.Artifacts = make([]CompiledArtifact, len(d.Artifacts))
	for index, value := range d.Artifacts {
		output.Artifacts[index] = value.Clone()
	}
	return output
}

type CompiledArtifact struct {
	Subresource basespec.SubresourceLocator `json:"subresource,omitempty"`
	Definition  definition.Definition       `json:"definition"`
	Diagnostics []diagnostic.Diagnostic     `json:"diagnostics,omitempty"`
}

func (a CompiledArtifact) Clone() CompiledArtifact {
	output := a
	output.Definition = a.Definition.Clone()
	output.Diagnostics = diagnostic.Clone(a.Diagnostics)
	return output
}

// CompiledPackageLifecycle is optional domain-owned work around one generated
// package hydration plan.
//
// Artifact Store owns package writes, Source revision changes, Source refresh,
// Artifact persistence, and generated Definition verification. Domains use
// this hook only for state outside Artifact Store, such as MCP overlays and
// secret cleanup.
//
// Prepare runs before the shared Source batch changes package bytes.
// Complete runs only after every batch, refresh, and generated Artifact
// verification succeeds.
type CompiledPackageLifecycle interface {
	PrepareCompiledHydration(
		ctx context.Context,
		plan CompiledPackagePlan,
	) (any, error)

	CompleteCompiledHydration(
		ctx context.Context,
		plan CompiledPackagePlan,
		state any,
	) error
}

// CompiledRegistration supplies a self-contained generated installation payload.
// The registration can only originate from application composition.
type CompiledRegistration struct {
	Set       CompiledPackageSet
	Lifecycle CompiledPackageLifecycle
}

// CompiledPackagePlan selects packages requiring physical reconciliation.
// Current hydration markers are interpreted by BootstrapRegistry. The
// coordinator receives only changed scopes and stale package markers.
type CompiledPackagePlan struct {
	Registration CompiledRegistration
	// TopologyCurrent is false when the installer's aggregate hydration
	// marker changed or was removed. MCP uses this to discard overlays after a
	// protected topology reset.
	TopologyCurrent bool
	Changed         []basespec.Locator
	Stale           []PackageHydration
}

// CompiledHydrationCoordinator is a privileged, process-internal startup
// boundary. It is deliberately separate from ordinary Artifact APIs.
type CompiledHydrationCoordinator interface {
	RegisterCompiledPackages(
		ctx context.Context,
		values []CompiledRegistration,
	) error

	HydrateCompiledPackages(
		ctx context.Context,
		plans []CompiledPackagePlan,
	) error
}
