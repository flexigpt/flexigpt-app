package local

import (
	"context"
	"path/filepath"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/iofs"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/managedfs"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/driver"
)

func localSourceDrivers(ctx context.Context, config Config, base string) ([]driver.Driver, error) {
	filesystem, err := fsdir.NewWithTraversalPolicy(config.FilesystemTraversalPolicy)
	if err != nil {
		return nil, err
	}
	managed, err := managedfs.New(
		filepath.Join(base, storeContentDirectoryName),
		filepath.Join(base, storeStagingDirectoryName),
	)
	if err != nil {
		return nil, err
	}
	embedded, err := iofs.NewWithRegistrations(
		ctx,
		config.EmbeddedProviders,
	)
	if err != nil {
		return nil, err
	}

	output := make([]driver.Driver, 0, 3+len(config.AdditionalSourceDrivers))
	output = append(output, filesystem, embedded, managed)
	output = append(output, config.AdditionalSourceDrivers...)
	return output, nil
}
