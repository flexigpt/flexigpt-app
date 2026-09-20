package basespec

import (
	"regexp"
)

const (
	MaxKindBytes             = 128
	MaxStorageKeyBytes       = 128
	MaxFingerprintBytes      = 128
	MaxSchemaIDBytes         = 256
	MaxDisplayNameBytes      = 256
	MaxDescriptionBytes      = 16 * 1024
	MaxLogicalNameBytes      = 256
	MaxURIBytes              = 16 * 1024
	MaxVersionBytes          = 256
	MaxSourceGenerationBytes = 1024
	MaxLocatorBytes          = 4096

	MaxLabels              = 64
	MaxLabelValueBytes     = 256
	MaxConfigBytes         = 1 << 20
	MaxLocalDataBytes      = 1 << 20
	MaxDefinitionBodyBytes = 4 << 20
	MaxDefinitionBytes     = 16 << 20

	MaxDefinitionDependencies = 4096
	MaxCandidateBytes         = 4 << 20
	MaxScanBytes              = int64(512 << 20)
	MaxPathPatterns           = 4096
	MaxDecoderHints           = 4096

	DefaultMaxCandidates = 10_000
	DefaultMaxEntries    = 100_000
	DefaultMaxDepth      = 64

	MaxDiscoveryCandidates = 100_000
	MaxDiscoveryEntries    = 1_000_000
	MaxDiscoveryDepth      = 256
)

const (
	ApplicationDirectoryMode = 0o700

	ArtifactStoreManifestFileName      = "store.json"
	ArtifactStoreMetadataFileName      = "app.sqlite"
	ArtifactStoreContentDirectoryName  = "content"
	ArtifactStoreStagingDirectoryName  = "staging"
	ArtifactStoreManifestTemporaryName = "store.json.tmp-"

	ArtifactStoreFormat        = "flexigpt-artifactstore/v3"
	ArtifactStoreContentLayout = "source-packages/v1"

	ArtifactStoreDirectoryMode = 0o750
	ArtifactStoreManifestMode  = 0o600

	ManagedPackageTemporaryPrefix = "package-"
	ManagedPackagePreviousPrefix  = "previous-package-"
	ManagedPackageRemovalPrefix   = "remove-"

	ExternalGitMetadataDirectoryName = ".git"
)

var ExternalTraversalExcludedDirectoryNames = []string{
	".git",
	".hg",
	".svn",
	"node_modules",
	"vendor",
	"bower_components",
}

// identifierPattern accepts lower-camel identifiers with optional dotted or
// hyphenated lower-camel segments. Each segment must begin with a lowercase
// ASCII letter. Uppercase ASCII letters are valid after the first character of
// a segment, which permits names such as mcpConfigs and managedMCP.
var identifierPattern = regexp.MustCompile(
	`^[a-z][A-Za-z0-9]*(?:[.-][a-z][A-Za-z0-9]*)*$`,
)

var portableReservedBaseNames = map[string]struct{}{
	"AUX":    {},
	"CON":    {},
	"CLOCK$": {},
	"COM1":   {},
	"COM2":   {},
	"COM3":   {},
	"COM4":   {},
	"COM5":   {},
	"COM6":   {},
	"COM7":   {},
	"COM8":   {},
	"COM9":   {},
	"LPT1":   {},
	"LPT2":   {},
	"LPT3":   {},
	"LPT4":   {},
	"LPT5":   {},
	"LPT6":   {},
	"LPT7":   {},
	"LPT8":   {},
	"LPT9":   {},
	"NUL":    {},
	"PRN":    {},
}
