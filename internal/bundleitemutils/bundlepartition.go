package bundleitemutils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/flexigpt/mapstore-go"
)

// bundlePartitionAttr is a type for bundle partition attributes.
type bundlePartitionAttr string

// BundlePartitionProvider implements mapstore.PartitionProvider for bundle directories.
type BundlePartitionProvider struct{}

// GetPartitionDir returns the partition directory for a given file key.
// The directory is provided via the XAttr field.
func (b *BundlePartitionProvider) GetPartitionDir(key mapstore.FileKey) (string, error) {
	if key.FileName == "" {
		return "", ErrInvalidFilename
	}
	dir, ok := key.XAttr.(bundlePartitionAttr)
	if !ok || dir == "" {
		return "", ErrBundleAttributeMissing
	}
	return string(dir), nil
}

// ListPartitions lists all bundle directories under baseDir.
func (b *BundlePartitionProvider) ListPartitions(
	baseDir, sortOrder, pageToken string,
	pageSize int,
) (dirs []string, nextPageToken string, err error) {
	return listDirs(baseDir, sortOrder, pageToken, pageSize)
}

func GetBundlePartitionFileKey(filename, dirname string) mapstore.FileKey {
	return mapstore.FileKey{FileName: filename, XAttr: bundlePartitionAttr(dirname)}
}

// listDirs returns a paginated and sorted list of directories in the base directory.
func listDirs(
	baseDir string,
	sortOrder string,
	pageToken string,
	pageSize int,
) (dirs []string, nextPageToken string, err error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read base directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}

	// Sort partitions.
	switch strings.ToLower(sortOrder) {
	case mapstore.SortOrderAscending:
		sort.Strings(dirs)
	case mapstore.SortOrderDescending:
		sort.Sort(sort.Reverse(sort.StringSlice(dirs)))
	default:
		return nil, "", fmt.Errorf("invalid sort order: %s", sortOrder)
	}

	// Decode page token.
	start := 0
	if pageToken != "" {
		tokenData, err := base64.StdEncoding.DecodeString(pageToken)
		if err != nil {
			return nil, "", fmt.Errorf("invalid page token: %w", err)
		}
		if err := json.Unmarshal(tokenData, &start); err != nil {
			return nil, "", fmt.Errorf("invalid page token: %w", err)
		}
	}

	// Apply pagination.
	end := min(start+pageSize, len(dirs))

	// Generate next page token.
	if end < len(dirs) {
		nextpageTokenData, _ := json.Marshal(end)
		nextPageToken = base64.StdEncoding.EncodeToString(nextpageTokenData)
	}

	return dirs[start:end], nextPageToken, nil
}
