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

	MaxLabels                 = 64
	MaxLabelValueBytes        = 256
	MaxConfigBytes            = 1 << 20
	MaxLocalDataBytes         = 1 << 20
	MaxDefinitionBodyBytes    = 4 << 20
	MaxDefinitionBytes        = 16 << 20
	MaxDefinitionDependencies = 4096
	MaxCandidateBytes         = 4 << 20
	MaxScanBytes              = int64(512 << 20)

	DefaultMaxCandidates   = 10_000
	DefaultMaxEntries      = 100_000
	DefaultMaxDepth        = 64
	MaxDiscoveryCandidates = 100_000
	MaxDiscoveryEntries    = 1_000_000
	MaxDiscoveryDepth      = 256
)

var identifierPattern = regexp.MustCompile(
	`^[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*$`,
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

type (
	RootID             string
	SourceID           string
	CollectionID       string
	ArtifactID         string
	StorageKey         string
	PackageKind        string
	SourceKind         string
	CollectionKind     string
	ArtifactKind       string
	SchemaID           string
	AttachmentRole     string
	DecoderID          string
	Locator            string
	SubresourceLocator string

	LogicalName    string
	LogicalVersion string
)

const (
	SourceKindFilesystemDirectory SourceKind = "fs-directory"
	SourceKindEmbeddedDirectory   SourceKind = "embedded-directory"
	SourceKindManagedDirectory    SourceKind = "managed-directory"
)
