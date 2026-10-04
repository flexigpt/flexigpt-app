package local

const (
	storeManifestFileName      = "store.json"
	storeMetadataFileName      = "app.sqlite"
	storeContentDirectoryName  = "content"
	storeStagingDirectoryName  = "staging"
	storeManifestTemporaryName = "store.json.tmp-"

	storeFormat        = "flexigpt-artifactstore/v1"
	storeContentLayout = "source-packages/v1"

	storeDirectoryMode = 0o750
	storeManifestMode  = 0o600
)
