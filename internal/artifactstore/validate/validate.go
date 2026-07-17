package validate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

var (
	uuidV7RE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	kindRE   = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*$`)
	digestRE = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

// ValidateArtifactRoot validates app-local root metadata.
func ValidateArtifactRoot(v spec.ArtifactRoot) error {
	if err := validateID("rootID", string(v.RootID)); err != nil {
		return err
	}
	if err := validateKind("root.kind", string(v.Kind)); err != nil {
		return err
	}
	if err := validateRequiredText("root.displayName", v.DisplayName, spec.MaxDisplayNameBytes); err != nil {
		return err
	}
	if err := validateDescription("root.description", v.Description); err != nil {
		return err
	}
	if v.MountRevision == 0 || v.MountRevision > spec.MaxObservationRevision {
		return invalidf(
			"root.mountRevision must be between 1 and %d",
			spec.MaxObservationRevision,
		)
	}
	if err := validateSchemaBoundJSONObject(
		"root.data",
		v.Data,
		v.DataSchemaID,
		spec.MaxLocalDataJSONBytes,
	); err != nil {
		return err
	}
	if err := validateCreatedModified("root", v.CreatedAt, v.ModifiedAt); err != nil {
		return err
	}
	if err := validateSoftDeleted("root", v.CreatedAt, v.SoftDeletedAt, v.Enabled); err != nil {
		return err
	}
	return nil
}

// ValidateArtifactSource validates app-local source registration metadata.
func ValidateArtifactSource(v spec.ArtifactSource) error {
	if err := validateID("sourceID", string(v.SourceID)); err != nil {
		return err
	}
	if err := validateKind("source.kind", string(v.Kind)); err != nil {
		return err
	}
	if err := validateRequiredText("source.displayName", v.DisplayName, spec.MaxDisplayNameBytes); err != nil {
		return err
	}
	if err := validateSchemaID("source.configSchemaID", v.ConfigSchemaID); err != nil {
		return err
	}
	if err := validateJSONObject("source.config", v.Config, spec.MaxConfigJSONBytes); err != nil {
		return err
	}
	if err := validateKnownSourceConfig(v); err != nil {
		return err
	}
	if v.LastObservedGeneration != nil {
		if err := validateSourceGeneration("source.lastObservedGeneration", *v.LastObservedGeneration); err != nil {
			return err
		}
	}
	if v.LastScannedAt != nil {
		if err := validateOptionalTimeAfter("source.lastScannedAt", v.LastScannedAt, v.CreatedAt); err != nil {
			return err
		}
	}
	if err := ValidateDiagnostics(v.Diagnostics); err != nil {
		return err
	}
	if v.ObservationRevision > spec.MaxObservationRevision {
		return invalidf(
			"source.observationRevision exceeds %d",
			spec.MaxObservationRevision,
		)
	}
	return validateCreatedModified("source", v.CreatedAt, v.ModifiedAt)
}

// ValidateRootSourceAttachment validates an app-local root/source attachment.
func ValidateRootSourceAttachment(v spec.RootSourceAttachment) error {
	if err := validateID("attachment.rootID", string(v.RootID)); err != nil {
		return err
	}
	if err := validateID("attachment.sourceID", string(v.SourceID)); err != nil {
		return err
	}
	if err := validateKind("attachment.role", string(v.Role)); err != nil {
		return err
	}
	if v.Priority < -spec.MaxAttachmentPriority || v.Priority > spec.MaxAttachmentPriority {
		return invalidf(
			"attachment.priority must be between %d and %d",
			-spec.MaxAttachmentPriority,
			spec.MaxAttachmentPriority,
		)
	}
	if err := validateSchemaBoundJSONObject(
		"attachment.data",
		v.Data,
		v.DataSchemaID,
		spec.MaxLocalDataJSONBytes,
	); err != nil {
		return err
	}
	return validateCreatedModified("attachment", v.CreatedAt, v.ModifiedAt)
}

// ValidateArtifactPackage validates app-local package-occurrence metadata.
func ValidateArtifactPackage(v spec.ArtifactPackage) error {
	if err := validateID("package.sourceID", string(v.SourceID)); err != nil {
		return err
	}
	if err := validateSourceLocator("package.manifestLocator", v.ManifestLocator, false); err != nil {
		return err
	}
	if err := validateCatalogState("package.state", v.State); err != nil {
		return err
	}
	if v.Name != "" {
		if err := validateRequiredText("package.name", string(v.Name), spec.MaxLogicalNameBytes); err != nil {
			return err
		}
	}
	if v.Version != "" {
		if err := validateVersion("package.version", string(v.Version), true); err != nil {
			return err
		}
	}
	if err := validateOptionalText("package.displayName", v.DisplayName, spec.MaxDisplayNameBytes); err != nil {
		return err
	}
	if err := validateDescription("package.description", v.Description); err != nil {
		return err
	}
	if err := validateOptionalDigest("package.currentManifestDigest", v.CurrentManifestDigest); err != nil {
		return err
	}
	if v.State == spec.CatalogStateValid {
		if err := validateRequiredText("package.name", string(v.Name), spec.MaxLogicalNameBytes); err != nil {
			return err
		}
		if err := validateVersion("package.version", string(v.Version), false); err != nil {
			return err
		}
		if v.CurrentManifestDigest == nil {
			return invalidf(
				"package.currentManifestDigest is required when package.state is %q",
				spec.CatalogStateValid,
			)
		}
	}
	if err := validateFirstLast("package", v.FirstSeenAt, v.LastSeenAt); err != nil {
		return err
	}
	return ValidateDiagnostics(v.Diagnostics)
}

// ValidateCatalogResource validates app-local catalog metadata.
func ValidateCatalogResource(v spec.CatalogResource) error {
	if err := ValidateCatalogResourceKey(spec.CatalogResourceKey{
		SourceID:           v.SourceID,
		Locator:            v.Locator,
		SubresourceLocator: v.SubresourceLocator,
	}); err != nil {
		return err
	}
	if v.PackageManifestLocator != "" {
		if err := validateSourceLocator(
			"catalog resource.packageManifestLocator",
			v.PackageManifestLocator,
			false,
		); err != nil {
			return err
		}
	}
	if v.Kind != "" {
		if err := validateKind("catalog resource.kind", string(v.Kind)); err != nil {
			return err
		}
	}
	if v.LogicalName != "" {
		if err := validateRequiredText(
			"catalog resource.logicalName",
			string(v.LogicalName),
			spec.MaxLogicalNameBytes,
		); err != nil {
			return err
		}
	}
	if v.LogicalVersion != "" {
		if err := validateVersion("catalog resource.logicalVersion", string(v.LogicalVersion), true); err != nil {
			return err
		}
	}
	if err := validateOptionalDigest(
		"catalog resource.currentDefinitionDigest",
		v.CurrentDefinitionDigest,
	); err != nil {
		return err
	}
	if err := validateOptionalDigest("catalog resource.sourceContentDigest", v.SourceContentDigest); err != nil {
		return err
	}
	if v.FrontendID != "" {
		if err := validateKind("catalog resource.frontendID", string(v.FrontendID)); err != nil {
			return err
		}
	}
	if err := validateCatalogState("catalog resource.state", v.State); err != nil {
		return err
	}
	if v.State == spec.CatalogStateValid {
		if err := validateKind("catalog resource.kind", string(v.Kind)); err != nil {
			return err
		}
		if err := validateRequiredText(
			"catalog resource.logicalName",
			string(v.LogicalName),
			spec.MaxLogicalNameBytes,
		); err != nil {
			return err
		}
		if v.CurrentDefinitionDigest == nil || v.SourceContentDigest == nil {
			return invalidf(
				"catalog resource digests are required when catalog resource.state is %q",
				spec.CatalogStateValid,
			)
		}
		if err := validateKind("catalog resource.frontendID", string(v.FrontendID)); err != nil {
			return err
		}
	}
	if err := validateFirstLast("catalog resource", v.FirstSeenAt, v.LastSeenAt); err != nil {
		return err
	}
	return ValidateDiagnostics(v.Diagnostics)
}

// ValidateCatalogResourceRevision validates durable resource history metadata.
func ValidateCatalogResourceRevision(v spec.CatalogResourceRevision) error {
	if err := ValidateCatalogResourceKey(spec.CatalogResourceKey{
		SourceID:           v.SourceID,
		Locator:            v.Locator,
		SubresourceLocator: v.SubresourceLocator,
	}); err != nil {
		return err
	}
	if err := validateDigest("catalog resource revision.definitionDigest", v.DefinitionDigest); err != nil {
		return err
	}
	if err := validateDigest("catalog resource revision.sourceContentDigest", v.SourceContentDigest); err != nil {
		return err
	}
	if err := validateKind("catalog resource revision.kind", string(v.Kind)); err != nil {
		return err
	}
	if err := validateKind("catalog resource revision.frontendID", string(v.FrontendID)); err != nil {
		return err
	}
	return validateFirstLast("catalog resource revision", v.FirstSeenAt, v.LastSeenAt)
}

// ValidateArtifactDefinitionFile validates the portable JSON definition file
// envelope used by generic transfer operations.
func ValidateArtifactDefinitionFile(v spec.ArtifactDefinitionFile) error {
	if v.Format != spec.ArtifactDefinitionFileFormatV1 {
		return invalidf("definition file.format %q is not supported", v.Format)
	}
	return ValidateCanonicalDefinition(v.Definition)
}

// ValidateCanonicalDefinition validates portable definition structure. Digest
// recomputation is intentionally performed later by the canonical codec.
func ValidateCanonicalDefinition(v spec.CanonicalDefinition) error {
	if err := validateDigest("definition.digest", v.Digest); err != nil {
		return err
	}
	if err := validateKind("definition.kind", string(v.Kind)); err != nil {
		return err
	}
	if err := validateSchemaID("definition.schemaID", v.SchemaID); err != nil {
		return err
	}
	if err := validateVersion("definition.schemaVersion", v.SchemaVersion, false); err != nil {
		return err
	}
	if err := validateRequiredText(
		"definition.logicalName",
		string(v.LogicalName),
		spec.MaxLogicalNameBytes,
	); err != nil {
		return err
	}
	if err := validateVersion("definition.logicalVersion", string(v.LogicalVersion), true); err != nil {
		return err
	}
	if err := validateOptionalText("definition.displayName", v.DisplayName, spec.MaxDisplayNameBytes); err != nil {
		return err
	}
	if err := validateDescription("definition.description", v.Description); err != nil {
		return err
	}
	if err := validateLabels("definition.labels", v.Labels); err != nil {
		return err
	}
	if err := validateJSONObject("definition.extensions", v.Extensions, spec.MaxExtensionsJSONBytes); err != nil {
		return err
	}
	if err := validateJSONObject(
		"definition.definitionJSON",
		v.DefinitionJSON,
		spec.MaxDefinitionJSONBytes,
	); err != nil {
		return err
	}
	if len(v.DependencySelectors) > spec.MaxSelectorsPerDefinition {
		return invalidf("definition.dependencySelectors exceeds %d entries", spec.MaxSelectorsPerDefinition)
	}
	for i, selector := range v.DependencySelectors {
		if err := ValidateArtifactSelector(selector); err != nil {
			return fmt.Errorf("definition.dependencySelectors[%d]: %w", i, err)
		}
	}
	if len(v.AssetManifest) > spec.MaxAssetsPerDefinition {
		return invalidf("definition.assetManifest exceeds %d entries", spec.MaxAssetsPerDefinition)
	}
	seenPaths := make(map[spec.PortablePath]struct{}, len(v.AssetManifest))
	for i, asset := range v.AssetManifest {
		if err := ValidateAssetManifestEntry(asset); err != nil {
			return fmt.Errorf("definition.assetManifest[%d]: %w", i, err)
		}
		if _, exists := seenPaths[asset.Path]; exists {
			return invalidf("definition.assetManifest contains duplicate path %q", asset.Path)
		}
		seenPaths[asset.Path] = struct{}{}
	}
	return nil
}

// ValidatePortablePackageManifest validates the portable generic package
// manifest. It does not impose this format on frontend-native source files.
func ValidatePortablePackageManifest(v spec.PortablePackageManifest) error {
	if v.Format != spec.PortablePackageManifestFormatV1 {
		return invalidf("portable package.format %q is not supported", v.Format)
	}
	if err := validateRequiredText("portable package.name", string(v.Name), spec.MaxLogicalNameBytes); err != nil {
		return err
	}
	if err := validateVersion("portable package.version", string(v.Version), false); err != nil {
		return err
	}
	if err := validateOptionalText(
		"portable package.displayName",
		v.DisplayName,
		spec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := validateDescription("portable package.description", v.Description); err != nil {
		return err
	}
	if err := validateJSONObject("portable package.extensions", v.Extensions, spec.MaxExtensionsJSONBytes); err != nil {
		return err
	}
	if len(v.Definitions) > spec.MaxPortablePackageDefinitions {
		return invalidf("portable package.definitions exceeds %d entries", spec.MaxPortablePackageDefinitions)
	}
	seenDigests := make(map[spec.Digest]struct{}, len(v.Definitions))
	seenFiles := make(map[spec.PortablePath]struct{}, len(v.Definitions))
	for i, ref := range v.Definitions {
		if err := validateDigest("portable package definition.digest", ref.Digest); err != nil {
			return fmt.Errorf("portable package.definitions[%d]: %w", i, err)
		}
		if err := validatePortablePath("portable package definition.file", ref.File, false); err != nil {
			return fmt.Errorf("portable package.definitions[%d]: %w", i, err)
		}
		if _, exists := seenDigests[ref.Digest]; exists {
			return invalidf("portable package.definitions contains duplicate digest %q", ref.Digest)
		}
		if _, exists := seenFiles[ref.File]; exists {
			return invalidf("portable package.definitions contains duplicate file %q", ref.File)
		}
		seenDigests[ref.Digest] = struct{}{}
		seenFiles[ref.File] = struct{}{}
	}
	if len(v.Assets) > spec.MaxAssetsPerDefinition {
		return invalidf("portable package.assets exceeds %d entries", spec.MaxAssetsPerDefinition)
	}
	seenAssetPaths := make(map[spec.PortablePath]struct{}, len(v.Assets))
	for i, asset := range v.Assets {
		if err := ValidateAssetManifestEntry(asset); err != nil {
			return fmt.Errorf("portable package.assets[%d]: %w", i, err)
		}
		if _, exists := seenAssetPaths[asset.Path]; exists {
			return invalidf("portable package.assets contains duplicate path %q", asset.Path)
		}
		seenAssetPaths[asset.Path] = struct{}{}
	}
	return nil
}

// ValidateAssetManifestEntry validates one portable asset reference.
func ValidateAssetManifestEntry(v spec.AssetManifestEntry) error {
	if err := validatePortablePath("asset.path", v.Path, false); err != nil {
		return err
	}
	if err := validateDigest("asset.digest", v.Digest); err != nil {
		return err
	}
	if err := validateOptionalText("asset.mediaType", v.MediaType, spec.MaxKindBytes); err != nil {
		return err
	}
	if v.SizeBytes < 0 {
		return invalidf("asset.sizeBytes must not be negative")
	}
	return nil
}

// ValidateExportClosure validates the generic portable closure returned by an
// artifact frontend. The root definition and all of its declared assets must
// remain present in the closure.
func ValidateExportClosure(
	root spec.CanonicalDefinition,
	closure spec.ExportClosure,
) error {
	if len(closure.DefinitionDigests) == 0 ||
		len(closure.DefinitionDigests) > spec.MaxPortablePackageDefinitions {
		return invalidf(
			"export closure must contain between 1 and %d definitions",
			spec.MaxPortablePackageDefinitions,
		)
	}
	seenDigests := make(map[spec.Digest]struct{}, len(closure.DefinitionDigests))
	rootSeen := false
	for index, digest := range closure.DefinitionDigests {
		if err := validateDigest("export closure.definitionDigest", digest); err != nil {
			return fmt.Errorf("export closure.definitionDigests[%d]: %w", index, err)
		}
		if _, exists := seenDigests[digest]; exists {
			return invalidf("export closure contains duplicate definition digest %q", digest)
		}
		seenDigests[digest] = struct{}{}
		rootSeen = rootSeen || digest == root.Digest
	}
	if !rootSeen {
		return invalidf("export closure does not contain root definition %q", root.Digest)
	}
	if len(closure.Assets) > spec.MaxAssetsPerDefinition {
		return invalidf("export closure.assets exceeds %d entries", spec.MaxAssetsPerDefinition)
	}
	assetsByPath := make(map[spec.PortablePath]spec.AssetManifestEntry, len(closure.Assets))
	for index, asset := range closure.Assets {
		if err := ValidateAssetManifestEntry(asset); err != nil {
			return fmt.Errorf("export closure.assets[%d]: %w", index, err)
		}
		if _, exists := assetsByPath[asset.Path]; exists {
			return invalidf("export closure contains duplicate asset path %q", asset.Path)
		}
		assetsByPath[asset.Path] = asset
	}
	for _, required := range root.AssetManifest {
		available, ok := assetsByPath[required.Path]
		if !ok || available != required {
			return invalidf(
				"export closure does not contain root asset %q with its declared metadata",
				required.Path,
			)
		}
	}
	return nil
}

// ValidateArtifactRecord validates app-local generic item metadata.
func ValidateArtifactRecord(v spec.ArtifactRecord) error {
	if err := validateID("record.recordID", string(v.RecordID)); err != nil {
		return err
	}
	if err := validateID("record.rootID", string(v.RootID)); err != nil {
		return err
	}
	if v.CollectionID != nil {
		if err := validateID("record.collectionID", string(*v.CollectionID)); err != nil {
			return err
		}
	}
	if err := validateKind("record.kind", string(v.Kind)); err != nil {
		return err
	}
	if err := validateSlug("record.name", string(v.Name)); err != nil {
		return err
	}
	if err := validateVersion("record.version", string(v.Version), true); err != nil {
		return err
	}
	if err := ValidateCatalogResourceKey(spec.CatalogResourceKey{
		SourceID:           v.SourceID,
		Locator:            v.Locator,
		SubresourceLocator: v.SubresourceLocator,
	}); err != nil {
		return err
	}
	if err := validateRecordMode("record.recordMode", v.RecordMode); err != nil {
		return err
	}
	if err := validateTrackingMode("record.trackingMode", v.TrackingMode); err != nil {
		return err
	}
	if err := validateOptionalDigest("record.pinnedDefinitionDigest", v.PinnedDefinitionDigest); err != nil {
		return err
	}
	if err := validateOptionalDigest(
		"record.lastResolvedDefinitionDigest",
		v.LastResolvedDefinitionDigest,
	); err != nil {
		return err
	}
	switch v.TrackingMode {
	case spec.TrackingModePinDigest:
		if v.PinnedDefinitionDigest == nil {
			return invalidf(
				"record.pinnedDefinitionDigest is required when record.trackingMode is %q",
				spec.TrackingModePinDigest,
			)
		}
	case spec.TrackingModeFollowSource, spec.TrackingModeManualRefresh:
		if v.PinnedDefinitionDigest != nil {
			return invalidf(
				"record.pinnedDefinitionDigest is only valid when record.trackingMode is %q",
				spec.TrackingModePinDigest,
			)
		}
	}
	if err := validateRecordState("record.state", v.State); err != nil {
		return err
	}
	switch v.State {
	case spec.RecordStateAvailable, spec.RecordStateStale, spec.RecordStateIncompatible:
		if v.LastResolvedDefinitionDigest == nil {
			return invalidf("record.lastResolvedDefinitionDigest is required when record.state is %q", v.State)
		}
	default:
	}
	if err := validateSchemaBoundJSONObject(
		"record.data",
		v.Data,
		v.DataSchemaID,
		spec.MaxLocalDataJSONBytes,
	); err != nil {
		return err
	}
	if err := ValidateDiagnostics(v.Diagnostics); err != nil {
		return err
	}
	return validateCreatedModified("record", v.CreatedAt, v.ModifiedAt)
}

// ValidateArtifactCollection validates an app-local record grouping.
func ValidateArtifactCollection(v spec.ArtifactCollection) error {
	if err := validateID("collection.collectionID", string(v.CollectionID)); err != nil {
		return err
	}
	if err := validateID("collection.rootID", string(v.RootID)); err != nil {
		return err
	}
	if err := validateKind("collection.kind", string(v.Kind)); err != nil {
		return err
	}
	if err := validateSlug("collection.slug", string(v.Slug)); err != nil {
		return err
	}
	if err := validateRequiredText("collection.displayName", v.DisplayName, spec.MaxDisplayNameBytes); err != nil {
		return err
	}
	if err := validateDescription("collection.description", v.Description); err != nil {
		return err
	}
	if err := validateSchemaBoundJSONObject(
		"collection.data",
		v.Data,
		v.DataSchemaID,
		spec.MaxLocalDataJSONBytes,
	); err != nil {
		return err
	}
	if err := validateCreatedModified("collection", v.CreatedAt, v.ModifiedAt); err != nil {
		return err
	}
	return validateSoftDeleted("collection", v.CreatedAt, v.SoftDeletedAt, v.Enabled)
}

// ValidateRootCatalogGeneration validates app-local scan-publication metadata.
func ValidateRootCatalogGeneration(v spec.RootCatalogGeneration) error {
	if err := validateID("catalog generation.rootID", string(v.RootID)); err != nil {
		return err
	}
	if v.Generation == 0 {
		return invalidf("catalog generation.generation must be greater than zero")
	}
	if v.RootRevision == 0 || v.RootRevision > spec.MaxObservationRevision {
		return invalidf(
			"catalog generation.rootRevision must be between 1 and %d",
			spec.MaxObservationRevision,
		)
	}
	for sourceID, version := range v.SourceVersions {
		if err := validateID("catalog generation.sourceVersions sourceID", string(sourceID)); err != nil {
			return err
		}
		if err := validateSourceGeneration(
			"catalog generation.sourceVersions generation",
			version.Generation,
		); err != nil {
			return err
		}
		if version.ObservationRevision == 0 ||
			version.ObservationRevision > spec.MaxObservationRevision {
			return invalidf(
				"catalog generation source observation revision must be between 1 and %d",
				spec.MaxObservationRevision,
			)
		}
	}
	if err := validateDigest("catalog generation.scanPlanDigest", v.ScanPlanDigest); err != nil {
		return err
	}
	if err := validateDigest("catalog generation.catalogDigest", v.CatalogDigest); err != nil {
		return err
	}
	if err := validateRequiredTime("catalog generation.createdAt", v.CreatedAt); err != nil {
		return err
	}
	return ValidateDiagnostics(v.Diagnostics)
}

// ValidateArtifactDependencySnapshot validates one durable selector result.
func ValidateArtifactDependencySnapshot(v spec.ArtifactDependencySnapshot) error {
	if err := validateID("dependency.rootID", string(v.RootID)); err != nil {
		return err
	}
	if err := validateID("dependency.recordID", string(v.RecordID)); err != nil {
		return err
	}
	if v.CatalogGeneration == 0 {
		return invalidf("dependency.catalogGeneration must be greater than zero")
	}
	if err := validateDigest(
		"dependency.rootDefinitionDigest",
		v.RootDefinitionDigest,
	); err != nil {
		return err
	}
	if err := validateDigest("dependency.definitionDigest", v.DefinitionDigest); err != nil {
		return err
	}
	if v.SelectorIndex < 0 || v.SelectorIndex >= spec.MaxSelectorsPerDefinition {
		return invalidf(
			"dependency.selectorIndex must be between 0 and %d",
			spec.MaxSelectorsPerDefinition-1,
		)
	}
	if err := ValidateArtifactSelector(v.Selector); err != nil {
		return fmt.Errorf("dependency.selector: %w", err)
	}
	switch v.State {
	case spec.DependencyResolutionStateResolved:
		if len(v.Candidates) != 1 {
			return invalidf("resolved dependency must contain exactly one candidate")
		}
	case spec.DependencyResolutionStateMissing:
		if len(v.Candidates) != 0 {
			return invalidf("missing dependency must not contain candidates")
		}
	case spec.DependencyResolutionStateAmbiguous:
		if len(v.Candidates) < 2 {
			return invalidf("ambiguous dependency must contain at least two candidates")
		}
	default:
		return invalidf("dependency.state %q is invalid", v.State)
	}
	seen := make(map[string]struct{}, len(v.Candidates))
	for index, candidate := range v.Candidates {
		if err := ValidateCatalogResourceKey(candidate.Resource); err != nil {
			return fmt.Errorf("dependency.candidates[%d].resource: %w", index, err)
		}
		if err := validateDigest(
			"dependency candidate.definitionDigest",
			candidate.DefinitionDigest,
		); err != nil {
			return fmt.Errorf("dependency.candidates[%d]: %w", index, err)
		}
		key := string(candidate.Resource.SourceID) + "\x00" +
			string(candidate.Resource.Locator) + "\x00" +
			string(candidate.Resource.SubresourceLocator) + "\x00" +
			string(candidate.DefinitionDigest)
		if _, exists := seen[key]; exists {
			return invalidf("dependency contains duplicate candidate %q", key)
		}
		seen[key] = struct{}{}
	}
	if err := ValidateDiagnostics(v.Diagnostics); err != nil {
		return err
	}
	return validateRequiredTime("dependency.modifiedAt", v.ModifiedAt)
}

// ValidateArtifactSelector validates a portable dependency selector.
func ValidateArtifactSelector(v spec.ArtifactSelector) error {
	if err := validateKind("selector.kind", string(v.Kind)); err != nil {
		return err
	}
	if v.LogicalName != "" {
		if err := validateRequiredText(
			"selector.logicalName",
			string(v.LogicalName),
			spec.MaxLogicalNameBytes,
		); err != nil {
			return err
		}
	}
	if err := validateVersion("selector.versionConstraint", v.VersionConstraint, true); err != nil {
		return err
	}
	return validateLabels("selector.labels", v.Labels)
}

// ValidateTransferProvenance validates app-local transfer audit metadata.
func ValidateTransferProvenance(v spec.TransferProvenance) error {
	if err := validateID("provenance.provenanceID", string(v.ProvenanceID)); err != nil {
		return err
	}
	if err := validateID("provenance.targetRecordID", string(v.TargetRecordID)); err != nil {
		return err
	}
	if err := validateTransferOperation("provenance.operation", v.Operation); err != nil {
		return err
	}
	if v.OriginRecordID != nil {
		if err := validateID("provenance.originRecordID", string(*v.OriginRecordID)); err != nil {
			return err
		}
	}
	if v.OriginResource != nil {
		if err := ValidateCatalogResourceKey(*v.OriginResource); err != nil {
			return fmt.Errorf("provenance.originResource: %w", err)
		}
	}
	if err := validateDigest("provenance.originDefinitionDigest", v.OriginDefinitionDigest); err != nil {
		return err
	}
	return validateRequiredTime("provenance.createdAt", v.CreatedAt)
}

// ValidateCatalogResourceKey validates a source-local resource identity.
func ValidateCatalogResourceKey(v spec.CatalogResourceKey) error {
	if err := validateID("catalog resource.sourceID", string(v.SourceID)); err != nil {
		return err
	}
	if err := validateSourceLocator("catalog resource.locator", v.Locator, true); err != nil {
		return err
	}
	return validateSubresourceLocator("catalog resource.subresourceLocator", v.SubresourceLocator)
}

// ValidateDiagnostics validates a bounded current diagnostic collection.
func ValidateDiagnostics(v []spec.Diagnostic) error {
	if len(v) > spec.MaxDiagnosticsPerEntity {
		return invalidf("diagnostics exceeds %d entries", spec.MaxDiagnosticsPerEntity)
	}
	for i, diagnostic := range v {
		if err := ValidateDiagnostic(diagnostic); err != nil {
			return fmt.Errorf("diagnostics[%d]: %w", i, err)
		}
	}
	return nil
}

// ValidateDiagnostic validates one structured diagnostic.
func ValidateDiagnostic(v spec.Diagnostic) error {
	switch v.Severity {
	case spec.DiagnosticSeverityError, spec.DiagnosticSeverityWarning, spec.DiagnosticSeverityInfo:
	default:
		return invalidf("diagnostic.severity %q is invalid", v.Severity)
	}
	if err := validateKind("diagnostic.code", v.Code); err != nil {
		return err
	}
	if err := validateRequiredText("diagnostic.message", v.Message, spec.MaxDiagnosticMessageBytes); err != nil {
		return err
	}
	if v.Location == nil {
		return nil
	}
	if v.Location.Locator != "" {
		if err := validateSourceLocator("diagnostic.location.locator", v.Location.Locator, true); err != nil {
			return err
		}
	}
	if err := validateSubresourceLocator(
		"diagnostic.location.subresourceLocator",
		v.Location.SubresourceLocator,
	); err != nil {
		return err
	}
	if v.Location.Line < 0 || v.Location.Column < 0 {
		return invalidf("diagnostic location line and column must not be negative")
	}
	return nil
}

func validateKnownSourceConfig(v spec.ArtifactSource) error {
	switch v.Kind {
	case spec.SourceKindFSDirectory:
		if v.ConfigSchemaID != spec.FSDirectoryConfigSchemaID {
			return invalidf("fs-directory source.configSchemaID must be %q", spec.FSDirectoryConfigSchemaID)
		}
		var cfg spec.FSDirectorySourceConfig
		if err := decodeStrictJSONObject(v.Config, &cfg); err != nil {
			return fmt.Errorf("fs-directory source.config: %w", err)
		}
		return ValidateFSDirectorySourceConfig(cfg)
	case spec.SourceKindEmbeddedFSDirectory:
		if v.ConfigSchemaID != spec.EmbeddedFSDirectoryConfigSchemaID {
			return invalidf(
				"embedded-fs-directory source.configSchemaID must be %q",
				spec.EmbeddedFSDirectoryConfigSchemaID,
			)
		}
		var cfg spec.EmbeddedFSDirectorySourceConfig
		if err := decodeStrictJSONObject(v.Config, &cfg); err != nil {
			return fmt.Errorf("embedded-fs-directory source.config: %w", err)
		}
		return ValidateEmbeddedFSDirectorySourceConfig(cfg)
	case spec.SourceKindMemoryDirectory:
		if v.ConfigSchemaID != spec.MemoryDirectoryConfigSchemaID {
			return invalidf("memory-directory source.configSchemaID must be %q", spec.MemoryDirectoryConfigSchemaID)
		}
		var cfg spec.MemoryDirectorySourceConfig
		if err := decodeStrictJSONObject(v.Config, &cfg); err != nil {
			return fmt.Errorf("memory-directory source.config: %w", err)
		}
		return ValidateMemoryDirectorySourceConfig(cfg)
	default:
		// Future Artifact Store-owned source kinds are validated by their
		// registered driver after this generic structural validation.
		return nil
	}
}

// ValidateMemoryDirectorySourceConfig validates test-only memory driver config.
func ValidateMemoryDirectorySourceConfig(v spec.MemoryDirectorySourceConfig) error {
	if err := validateKind("memory-directory.providerKey", v.ProviderKey); err != nil {
		return err
	}
	return validateSourceLocator("memory-directory.rootLocator", v.RootLocator, true)
}

// ValidateEmbeddedFSDirectorySourceConfig validates embedded-fs driver config.
func ValidateEmbeddedFSDirectorySourceConfig(v spec.EmbeddedFSDirectorySourceConfig) error {
	if err := validateKind("embedded-fs-directory.providerKey", v.ProviderKey); err != nil {
		return err
	}
	return validateSourceLocator("embedded-fs-directory.rootLocator", v.RootLocator, true)
}

// ValidateFSDirectorySourceConfig validates the app-local filesystem driver
// configuration. The service normalizes a path before storing it.
func ValidateFSDirectorySourceConfig(v spec.FSDirectorySourceConfig) error {
	if err := validateRequiredText("fs-directory.rootPath", v.RootPath, spec.MaxFilesystemPathBytes); err != nil {
		return err
	}
	if strings.ContainsRune(v.RootPath, 0) {
		return invalidf("fs-directory.rootPath contains a NUL byte")
	}
	if !filepath.IsAbs(v.RootPath) {
		return invalidf("fs-directory.rootPath must be absolute")
	}
	if filepath.Clean(v.RootPath) != v.RootPath {
		return invalidf("fs-directory.rootPath must be normalized")
	}
	return nil
}

func validateID(label, value string) error {
	if !uuidV7RE.MatchString(value) {
		return invalidf("%s must be a canonical UUIDv7", label)
	}
	return nil
}

func validateOptionalDigest(label string, value *spec.Digest) error {
	if value == nil {
		return nil
	}
	return validateDigest(label, *value)
}

func validateDigest(label string, value spec.Digest) error {
	if !digestRE.MatchString(string(value)) {
		return invalidf("%s must be sha256:<64 lowercase hex characters>", label)
	}
	return nil
}

func validateCatalogState(label string, value spec.CatalogState) error {
	switch value {
	case spec.CatalogStateValid, spec.CatalogStateInvalid, spec.CatalogStateMissing:
		return nil
	default:
		return invalidf("%s %q is invalid", label, value)
	}
}

func validateRecordMode(label string, value spec.RecordMode) error {
	switch value {
	case spec.RecordModeLinked,
		spec.RecordModeCaptured,
		spec.RecordModeForked,
		spec.RecordModeAppLocal,
		spec.RecordModeEmbeddedOverlay:
		return nil
	default:
		return invalidf("%s %q is invalid", label, value)
	}
}

func validateTrackingMode(label string, value spec.TrackingMode) error {
	switch value {
	case spec.TrackingModeFollowSource, spec.TrackingModePinDigest, spec.TrackingModeManualRefresh:
		return nil
	default:
		return invalidf("%s %q is invalid", label, value)
	}
}

func validateRecordState(label string, value spec.RecordState) error {
	switch value {
	case spec.RecordStateAvailable,
		spec.RecordStateStale,
		spec.RecordStateMissing,
		spec.RecordStateInvalid,
		spec.RecordStateIncompatible:
		return nil
	default:
		return invalidf("%s %q is invalid", label, value)
	}
}

func validateTransferOperation(label string, value spec.TransferOperation) error {
	switch value {
	case spec.TransferOperationImport, spec.TransferOperationCapture, spec.TransferOperationFork:
		return nil
	default:
		return invalidf("%s %q is invalid", label, value)
	}
}

func validateSourceGeneration(label string, value spec.SourceGeneration) error {
	return validateRequiredText(label, string(value), spec.MaxSourceGenerationBytes)
}

func validateSlug(label, value string) error {
	if !utf8.ValidString(value) || strings.TrimSpace(value) != value || value == "" {
		return invalidf("%s must be non-empty, valid UTF-8, and trimmed", label)
	}
	if utf8.RuneCountInString(value) > spec.MaxSlugRunes {
		return invalidf("%s exceeds %d runes", label, spec.MaxSlugRunes)
	}
	if strings.HasPrefix(value, "-") || strings.HasSuffix(value, "-") {
		return invalidf("%s must not start or end with a hyphen", label)
	}
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			continue
		}
		return invalidf("%s contains an invalid character %q", label, r)
	}
	return nil
}

func validateVersion(label, value string, optional bool) error {
	if value == "" && optional {
		return nil
	}
	return validateRequiredText(label, value, spec.MaxVersionBytes)
}

func validateOptionalText(label, value string, maxBytes int) error {
	if value == "" {
		return nil
	}
	return validateRequiredText(label, value, maxBytes)
}

func validateDescription(label, value string) error {
	if value == "" {
		return nil
	}
	if !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
		return invalidf("%s must be valid UTF-8 and contain no NUL byte", label)
	}
	if len(value) > spec.MaxDescriptionBytes {
		return invalidf("%s exceeds %d bytes", label, spec.MaxDescriptionBytes)
	}
	return nil
}

func validateLabels(label string, values map[string]string) error {
	if len(values) > spec.MaxLabelsPerDefinition {
		return invalidf("%s exceeds %d entries", label, spec.MaxLabelsPerDefinition)
	}
	for key, value := range values {
		if err := validateKind(label+" key", key); err != nil {
			return err
		}
		if err := validateRequiredText(label+"["+key+"]", value, spec.MaxLabelValueBytes); err != nil {
			return err
		}
	}
	return nil
}

func validateRequiredText(label, value string, maxBytes int) error {
	if !utf8.ValidString(value) || value == "" || strings.TrimSpace(value) != value {
		return invalidf("%s must be non-empty, valid UTF-8, and trimmed", label)
	}
	if len(value) > maxBytes {
		return invalidf("%s exceeds %d bytes", label, maxBytes)
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return invalidf("%s contains a control character", label)
		}
	}
	return nil
}

func validateSourceLocator(label string, value spec.SourceLocator, allowRoot bool) error {
	return validatePortablePathValue(label, string(value), allowRoot)
}

func validateSubresourceLocator(label string, value spec.SubresourceLocator) error {
	if value == "" {
		return nil
	}
	return validatePortablePathValue(label, string(value), false)
}

func validatePortablePath(label string, value spec.PortablePath, allowRoot bool) error {
	return validatePortablePathValue(label, string(value), allowRoot)
}

func validatePortablePathValue(label, value string, allowRoot bool) error {
	if value == "." && allowRoot {
		return nil
	}
	if value == "" || len(value) > spec.MaxSourceLocatorBytes || !utf8.ValidString(value) {
		return invalidf("%s must be a non-empty, bounded, valid UTF-8 relative path", label)
	}
	if strings.ContainsRune(value, 0) || strings.Contains(value, "\\") || strings.Contains(value, ":") {
		return invalidf("%s contains a disallowed path character", label)
	}
	if path.IsAbs(value) || path.Clean(value) != value || value == "." || value == ".." ||
		strings.HasPrefix(value, "../") {
		return invalidf("%s must be normalized and remain relative to its root", label)
	}
	for part := range strings.SplitSeq(value, "/") {
		if part == "" || part == "." || part == ".." {
			return invalidf("%s contains an invalid path segment", label)
		}
		for _, r := range part {
			if unicode.IsControl(r) {
				return invalidf("%s contains a control character", label)
			}
		}
	}
	return nil
}

func validateSchemaBoundJSONObject(label string, raw json.RawMessage, schemaID spec.SchemaID, maxBytes int) error {
	if err := validateJSONObject(label, raw, maxBytes); err != nil {
		return err
	}
	if schemaID != "" {
		if err := validateSchemaID(label+"SchemaID", schemaID); err != nil {
			return err
		}
	}
	if !isEmptyJSONObject(raw) && schemaID == "" {
		return invalidf("%sSchemaID is required when %s is not empty", label, label)
	}
	return nil
}

func validateKind(label, value string) error {
	if len(value) > spec.MaxKindBytes || !kindRE.MatchString(value) {
		return invalidf("%s must be a lowercase dotted or hyphenated identifier", label)
	}
	return nil
}

func validateSchemaID(label string, value spec.SchemaID) error {
	if len(value) > spec.MaxSchemaIDBytes || !kindRE.MatchString(string(value)) {
		return invalidf("%s must be a lowercase dotted or hyphenated identifier", label)
	}
	return nil
}

func validateJSONObject(label string, raw json.RawMessage, maxBytes int) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || len(trimmed) > maxBytes || !utf8.Valid(trimmed) {
		return invalidf("%s must be a bounded, non-empty, valid UTF-8 JSON object", label)
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	first, err := decoder.Token()
	if err != nil {
		return invalidf("%s is not valid JSON: %v", label, err)
	}
	if first != json.Delim('{') {
		return invalidf("%s must be a JSON object", label)
	}
	if err := validateJSONValue(decoder, first); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return invalidf("%s contains trailing JSON values", label)
		}
		return invalidf("%s contains invalid trailing data: %v", label, err)
	}
	return nil
}

func validateJSONValue(decoder *json.Decoder, token json.Token) error {
	delim, isDelim := token.(json.Delim)
	if !isDelim {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return invalidf("invalid JSON object key: %v", err)
			}
			key, ok := keyToken.(string)
			if !ok {
				return invalidf("invalid JSON object key")
			}
			if _, exists := seen[key]; exists {
				return invalidf("duplicate JSON object key %q", key)
			}
			seen[key] = struct{}{}
			valueToken, err := decoder.Token()
			if err != nil {
				return invalidf("invalid JSON object value: %v", err)
			}
			if err := validateJSONValue(decoder, valueToken); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return invalidf("invalid JSON object terminator")
		}
	case '[':
		for decoder.More() {
			valueToken, err := decoder.Token()
			if err != nil {
				return invalidf("invalid JSON array value: %v", err)
			}
			if err := validateJSONValue(decoder, valueToken); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return invalidf("invalid JSON array terminator")
		}
	default:
		return invalidf("invalid JSON delimiter")
	}
	return nil
}

func decodeStrictJSONObject(raw json.RawMessage, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return invalidf("invalid JSON object: %v", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return invalidf("JSON object contains trailing values")
		}
		return invalidf("JSON object contains invalid trailing data: %v", err)
	}
	return nil
}

func isEmptyJSONObject(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("{}"))
}

func validateCreatedModified(label string, createdAt, modifiedAt time.Time) error {
	if err := validateRequiredTime(label+".createdAt", createdAt); err != nil {
		return err
	}
	if err := validateRequiredTime(label+".modifiedAt", modifiedAt); err != nil {
		return err
	}
	if modifiedAt.Before(createdAt) {
		return invalidf("%s.modifiedAt is before %s.createdAt", label, label)
	}
	return nil
}

func validateFirstLast(label string, firstSeenAt, lastSeenAt time.Time) error {
	if err := validateRequiredTime(label+".firstSeenAt", firstSeenAt); err != nil {
		return err
	}
	if err := validateRequiredTime(label+".lastSeenAt", lastSeenAt); err != nil {
		return err
	}
	if lastSeenAt.Before(firstSeenAt) {
		return invalidf("%s.lastSeenAt is before %s.firstSeenAt", label, label)
	}
	return nil
}

func validateSoftDeleted(label string, createdAt time.Time, softDeletedAt *time.Time, enabled bool) error {
	if softDeletedAt == nil {
		return nil
	}
	if err := validateRequiredTime(label+".softDeletedAt", *softDeletedAt); err != nil {
		return err
	}
	if softDeletedAt.Before(createdAt) {
		return invalidf("%s.softDeletedAt is before %s.createdAt", label, label)
	}
	if enabled {
		return invalidf("soft-deleted %s cannot be enabled", label)
	}
	return nil
}

func validateOptionalTimeAfter(label string, value *time.Time, minimum time.Time) error {
	if value == nil {
		return nil
	}
	if err := validateRequiredTime(label, *value); err != nil {
		return err
	}
	if !minimum.IsZero() && value.Before(minimum) {
		return invalidf("%s is before its entity creation time", label)
	}
	return nil
}

func validateRequiredTime(label string, value time.Time) error {
	if value.IsZero() {
		return invalidf("%s is zero", label)
	}
	return nil
}

func invalidf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", spec.ErrInvalid, fmt.Sprintf(format, args...))
}
