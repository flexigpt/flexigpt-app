package artifactstore

import (
	"context"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
)

type sourceScanReadBudget struct {
	Bytes        int64
	AssetEntries int
	AssetFiles   int
	AssetBytes   int64
}

func (s *Store) collectDecodedAssetManifest(
	ctx context.Context,
	driver spec.SourceDriver,
	source spec.ArtifactSource,
	candidateLocator spec.SourceLocator,
	roots []spec.SourceAssetRoot,
	plan spec.SourceScanPlan,
	budget *sourceScanReadBudget,
) ([]spec.AssetManifestEntry, error) {
	if len(roots) == 0 {
		return nil, nil
	}
	if len(roots) > spec.MaxAssetRootsPerDefinition {
		return nil, fmt.Errorf(
			"%w: decoded artifact declares more than %d source asset roots",
			spec.ErrInvalidRequest,
			spec.MaxAssetRootsPerDefinition,
		)
	}
	if budget == nil {
		return nil, fmt.Errorf("%w: source scan read budget is nil", spec.ErrInvalidRequest)
	}
	if s.portableContent == nil {
		return nil, fmt.Errorf(
			"%w: portable content repository is not configured",
			spec.ErrUnsupported,
		)
	}

	candidateDirectory := spec.SourceLocator(path.Dir(string(candidateLocator)))
	manifestByPath := make(map[spec.PortablePath]spec.AssetManifestEntry)
	maxEntries := plan.MaxTraversalEntries
	if maxEntries <= 0 {
		maxEntries = spec.DefaultMaxScanEntries
	}
	maxDepth := plan.MaxTraversalDepth
	if maxDepth <= 0 {
		maxDepth = spec.DefaultMaxTraversalDepth
	}
	maxTotalBytes := plan.MaxTotalBytes
	if maxTotalBytes <= 0 {
		maxTotalBytes = spec.MaxScanTotalBytes
	}

	for _, root := range roots {
		if err := validateSourceAssetRoot(candidateDirectory, root); err != nil {
			return nil, err
		}
		rootEntry, err := driver.Stat(ctx, source, root.Root)
		if err != nil {
			return nil, err
		}
		included, err := sourceEntryIncluded(ctx, driver, source, rootEntry)
		if err != nil {
			return nil, err
		}
		if !included {
			continue
		}
		if !rootEntry.IsDirectory || rootEntry.IsSymlink {
			return nil, fmt.Errorf(
				"%w: source asset root %q is not a safe directory",
				spec.ErrInvalidRequest,
				root.Root,
			)
		}

		var visit func(spec.SourceLocator, int) error
		visit = func(directory spec.SourceLocator, depth int) error {
			entries, err := driver.ReadDir(ctx, source, directory)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if err := ctx.Err(); err != nil {
					return err
				}
				budget.AssetEntries++
				if budget.AssetEntries > maxEntries {
					return fmt.Errorf(
						"%w: source asset traversal exceeds %d entries",
						spec.ErrInvalidRequest,
						maxEntries,
					)
				}
				entryDepth := depth + 1
				if entryDepth > maxDepth {
					return fmt.Errorf(
						"%w: source asset traversal exceeds depth %d at %q",
						spec.ErrInvalidRequest,
						maxDepth,
						entry.Locator,
					)
				}
				included, err := sourceEntryIncluded(ctx, driver, source, entry)
				if err != nil {
					return err
				}
				if !included || entry.IsSymlink {
					continue
				}
				if entry.IsDirectory {
					if root.Recursive {
						if err := visit(entry.Locator, entryDepth); err != nil {
							return err
						}
					}
					continue
				}
				if !entry.IsRegular || entry.Locator == candidateLocator {
					continue
				}
				if !matchesDirectoryRoot(root.Root, entry.Locator, spec.DirectoryScanRoot{
					Root:            root.Root,
					IncludePatterns: root.IncludePatterns,
					Recursive:       root.Recursive,
				}) {
					continue
				}

				portablePath, err := portableAssetPath(root, entry.Locator)
				if err != nil {
					return err
				}
				if _, duplicate := manifestByPath[portablePath]; duplicate {
					return fmt.Errorf(
						"%w: source asset roots emit duplicate portable path %q",
						spec.ErrInvalidRequest,
						portablePath,
					)
				}
				if len(manifestByPath) >= spec.MaxAssetsPerDefinition {
					return fmt.Errorf(
						"%w: decoded artifact exceeds %d assets",
						spec.ErrInvalidRequest,
						spec.MaxAssetsPerDefinition,
					)
				}
				budget.AssetFiles++
				if budget.AssetFiles > spec.DefaultMaxScanAssetFiles {
					return fmt.Errorf(
						"%w: source scan exceeds %d asset files",
						spec.ErrInvalidRequest,
						spec.DefaultMaxScanAssetFiles,
					)
				}
				remaining := maxTotalBytes - budget.Bytes
				if entry.SizeBytes < 0 || entry.SizeBytes > remaining {
					return fmt.Errorf(
						"%w: source scan exceeds the configured %d byte total",
						spec.ErrInvalidRequest,
						maxTotalBytes,
					)
				}
				content, err := readSourceAsset(ctx, driver, source, entry, remaining)
				if err != nil {
					return err
				}
				digest, size, err := s.portableContent.PutAsset(ctx, content)
				if err != nil {
					return err
				}
				if size != int64(len(content)) {
					return fmt.Errorf(
						"%w: persisted asset %q size changed",
						spec.ErrDigestMismatch,
						portablePath,
					)
				}
				budget.Bytes += size
				budget.AssetBytes += size
				if budget.AssetBytes > spec.MaxTransferPayloadBytes {
					return fmt.Errorf(
						"%w: source assets exceed %d bytes",
						spec.ErrInvalidRequest,
						spec.MaxTransferPayloadBytes,
					)
				}
				manifestByPath[portablePath] = spec.AssetManifestEntry{
					Path:      portablePath,
					Digest:    digest,
					SizeBytes: size,
				}
			}
			return nil
		}
		if err := visit(root.Root, 0); err != nil {
			return nil, err
		}
	}

	manifest := make([]spec.AssetManifestEntry, 0, len(manifestByPath))
	for _, entry := range manifestByPath {
		manifest = append(manifest, entry)
	}
	sort.Slice(manifest, func(left, right int) bool {
		return manifest[left].Path < manifest[right].Path
	})
	return manifest, nil
}

func validateSourceAssetRoot(
	candidateDirectory spec.SourceLocator,
	root spec.SourceAssetRoot,
) error {
	if err := validate.ValidateSourceLocator(root.Root, true); err != nil {
		return fmt.Errorf("%w: source asset root: %w", spec.ErrInvalidRequest, err)
	}
	if !sourceLocatorWithin(candidateDirectory, root.Root) {
		return fmt.Errorf(
			"%w: source asset root %q is outside candidate directory %q",
			spec.ErrInvalidRequest,
			root.Root,
			candidateDirectory,
		)
	}
	if root.PortablePrefix != "" {
		if err := validate.ValidatePortablePath(root.PortablePrefix, false); err != nil {
			return fmt.Errorf("%w: source asset portable prefix: %w", spec.ErrInvalidRequest, err)
		}
	}
	if len(root.IncludePatterns) > spec.MaxIncludePatternsPerRoot {
		return fmt.Errorf("%w: source asset root has too many include patterns", spec.ErrInvalidRequest)
	}
	seen := make(map[string]struct{}, len(root.IncludePatterns))
	for _, pattern := range root.IncludePatterns {
		if strings.TrimSpace(pattern) != pattern || pattern == "" {
			return fmt.Errorf("%w: source asset include pattern is invalid", spec.ErrInvalidRequest)
		}
		if _, duplicate := seen[pattern]; duplicate {
			return fmt.Errorf("%w: duplicate source asset include pattern %q", spec.ErrInvalidRequest, pattern)
		}
		seen[pattern] = struct{}{}
		if _, err := path.Match(pattern, "asset"); err != nil {
			return fmt.Errorf("%w: invalid source asset pattern %q: %w", spec.ErrInvalidRequest, pattern, err)
		}
	}
	return nil
}

func sourceLocatorWithin(parent, child spec.SourceLocator) bool {
	if parent == "." {
		return child == "." || child != ""
	}
	return child == parent || strings.HasPrefix(string(child), string(parent)+"/")
}

func portableAssetPath(root spec.SourceAssetRoot, locator spec.SourceLocator) (spec.PortablePath, error) {
	value := string(locator)
	if root.Root != "." {
		prefix := string(root.Root) + "/"
		if !strings.HasPrefix(value, prefix) {
			return "", fmt.Errorf("%w: asset locator %q escapes root %q", spec.ErrInvalidRequest, locator, root.Root)
		}
		value = strings.TrimPrefix(value, prefix)
	}
	if root.PortablePrefix != "" {
		value = path.Join(string(root.PortablePrefix), value)
	}
	portable := spec.PortablePath(value)
	if err := validate.ValidatePortablePath(portable, false); err != nil {
		return "", err
	}
	return portable, nil
}

func readSourceAsset(
	ctx context.Context,
	driver spec.SourceDriver,
	source spec.ArtifactSource,
	entry spec.SourceEntry,
	maximum int64,
) ([]byte, error) {
	reader, err := driver.Open(ctx, source, entry.Locator)
	if err != nil {
		return nil, err
	}
	content, readErr := io.ReadAll(io.LimitReader(reader, maximum+1))
	closeErr := reader.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if int64(len(content)) > maximum {
		return nil, fmt.Errorf("%w: asset %q exceeds remaining scan budget", spec.ErrInvalidRequest, entry.Locator)
	}
	if entry.SizeBytes >= 0 && int64(len(content)) != entry.SizeBytes {
		return nil, fmt.Errorf(
			"%w: asset %q changed size from %d to %d bytes while being read",
			spec.ErrConflict,
			entry.Locator,
			entry.SizeBytes,
			len(content),
		)
	}
	return content, nil
}
