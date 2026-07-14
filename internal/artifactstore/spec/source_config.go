package spec

// FSDirectorySourceConfig is app-local configuration for a filesystem source.
// RootPath is deliberately allowed only here, never in portable definitions.
type FSDirectorySourceConfig struct {
	RootPath       string `json:"rootPath"`
	FollowSymlinks bool   `json:"followSymlinks"`
	ManagedByApp   bool   `json:"managedByApp"`
}

// EmbeddedFSDirectorySourceConfig addresses an application-registered fs.FS
// provider. ProviderKey is app-local registration metadata; portable package
// contents remain ordinary files within that provider.
type EmbeddedFSDirectorySourceConfig struct {
	ProviderKey string        `json:"providerKey"`
	RootLocator SourceLocator `json:"rootLocator"`
}

// MemoryDirectorySourceConfig is test-only configuration for an
// application-registered in-memory directory provider.
type MemoryDirectorySourceConfig struct {
	ProviderKey string        `json:"providerKey"`
	RootLocator SourceLocator `json:"rootLocator"`
}
