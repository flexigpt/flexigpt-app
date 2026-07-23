package attachment

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// defaultSkippedDirectory reports directories that should not be traversed.
// Dot-prefixed directories are also skipped, including VCS metadata such as .git.
func defaultSkippedDirectory(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}

	switch strings.ToLower(name) {
	case "node_modules",
		"vendor",
		"bower_components",
		"__pycache__",
		"venv",
		"site-packages",
		"build",
		"dist",
		"out",
		"target",
		"coverage",
		"pods",
		"deriveddata":
		return true
	default:
		return false
	}
}

// WalkDirectoryWithFiles implements:
//
//   - Pure BFS over the directory tree starting at dirPath.
//   - For each directory:
//   - process all regular, non-dot files first (highest priority),
//     adding them to Files until maxFiles is reached.
//   - then enqueue its non-ignored subdirectories for BFS.
//   - As soon as Files reaches maxFiles, walking stops.
//   - Remaining directories in the BFS queue are returned as OverflowDirs,
//     with a shallow os.ReadDir() to get a one-level item count.
//   - The directory where we actually hit the limit (if any) is also listed
//     as an overflow entry with Partial = true and a count of remaining items
//     (files + subdirs) in that directory.
func WalkDirectoryWithFiles(ctx context.Context, dirPath string, maxFiles int) (*WalkDirectoryWithFilesResult, error) {
	if maxFiles <= 0 || maxFiles > maxTotalDirWalkFiles {
		maxFiles = maxTotalDirWalkFiles
	}
	if dirPath == "" {
		// Empty path. Nothing to walk.
		return &WalkDirectoryWithFilesResult{
			DirPath:      "",
			Files:        []PathInfo{},
			OverflowDirs: nil,
			MaxFiles:     maxFiles,
			TotalSize:    0,
			HasMore:      false,
		}, nil
	}

	// Root dirPath may be relative and may start with "." (e.g. ".local/share").
	// Resolve path using filepath.Abs relative to the current working directory.
	absRoot, err := filepath.Abs(dirPath)
	if err == nil {
		dirPath = absRoot
	}
	dirPath = filepath.Clean(dirPath)

	type dirNode struct {
		absPath string // absolute path to this directory
		relPath string // path relative to root dir; "" for root
	}

	files := make([]PathInfo, 0, maxFiles)
	var totalSize int64

	overflowDirs := make([]DirectoryOverflowInfo, 0)

	queue := []dirNode{
		{
			absPath: dirPath,
			relPath: "",
		},
	}

	limitReached := false
	var partialOverflow *DirectoryOverflowInfo

	for len(queue) > 0 && !limitReached {
		// BFS: pop front.
		node := queue[0]
		queue = queue[1:]

		if err := ctx.Err(); err != nil {
			// Caller canceled or deadline exceeded.
			return nil, err
		}

		entries, err := os.ReadDir(node.absPath)
		if err != nil {
			// Log and continue; if it's the root directory, propagate the error.
			slog.Debug("error while walking directory", "path", node.absPath, "error", err)
			if node.absPath == dirPath {
				return nil, err
			}
			// For non-root dirs we can't see inside; treat as an overflow dir
			// with unknown contents (FileCount = 0).
			overflowDirs = append(overflowDirs, DirectoryOverflowInfo{
				DirPath:      node.absPath,
				RelativePath: node.relPath,
				FileCount:    0,
				Partial:      false,
			})
			continue
		}

		// Split into files and dirs, ignoring hidden entries and known junk directories.
		// We attach files from this directory before recursing into its subdirectories.
		fileEntries := make([]os.DirEntry, 0, len(entries))
		dirEntries := make([]os.DirEntry, 0, len(entries))
		for _, e := range entries {
			name := e.Name()

			if e.IsDir() {
				if defaultSkippedDirectory(name) {
					continue
				}
				dirEntries = append(dirEntries, e)
				continue
			}

			// Skip dot files entirely.
			if strings.HasPrefix(name, ".") {
				continue
			}

			fileEntries = append(fileEntries, e)
		}
		// Process files first.
		filesAddedHere := 0

		for i, e := range fileEntries {
			if len(files) >= maxFiles {
				// We already hit the limit before processing this file.
				remainingFiles := len(fileEntries) - i
				remainingSubdirs := len(dirEntries)
				if remainingFiles > 0 || remainingSubdirs > 0 {
					partialOverflow = &DirectoryOverflowInfo{
						DirPath:      node.absPath,
						RelativePath: node.relPath,
						FileCount:    remainingFiles + remainingSubdirs,
						Partial:      true,
					}
				}
				limitReached = true
				break
			}

			if err := ctx.Err(); err != nil {
				return nil, err
			}

			name := e.Name()
			fullPath := filepath.Join(node.absPath, name)

			info, err := e.Info()
			if err != nil {
				slog.Debug("stat error while walking directory", "path", fullPath, "error", err)
				continue
			}
			if !info.Mode().IsRegular() {
				// Skip symlinks, sockets, devices, fifos, etc.
				continue
			}

			modTime := info.ModTime().UTC()
			pInfo := PathInfo{
				Path:    fullPath,
				Name:    info.Name(),
				Exists:  true,
				IsDir:   info.IsDir(),
				Size:    info.Size(),
				ModTime: &modTime,
			}

			files = append(files, pInfo)
			totalSize += pInfo.Size
			filesAddedHere++

			if len(files) >= maxFiles {
				// We just consumed the last allowed slot from this directory.
				remainingFiles := len(fileEntries) - (i + 1)
				remainingSubdirs := len(dirEntries)
				if remainingFiles > 0 || remainingSubdirs > 0 {
					partialOverflow = &DirectoryOverflowInfo{
						DirPath:      node.absPath,
						RelativePath: node.relPath,
						FileCount:    remainingFiles + remainingSubdirs,
						Partial:      true,
					}
				}
				limitReached = true
				break
			}
		}

		if limitReached {
			break
		}

		// Now enqueue subdirectories for BFS (next levels).
		for _, e := range dirEntries {
			name := e.Name()
			abs := filepath.Join(node.absPath, name)
			rel := name
			if node.relPath != "" {
				rel = filepath.Join(node.relPath, name)
			}
			queue = append(queue, dirNode{
				absPath: abs,
				relPath: rel,
			})
		}
	}

	// Any directories left in the BFS queue were *never* processed at all
	// because we reached maxFiles or exhausted context. We do a shallow
	// os.ReadDir on each to get a one-level "item count" for the UI.
	for _, node := range queue {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		itemCount := 0
		entries, err := os.ReadDir(node.absPath)
		if err != nil {
			slog.Debug("error while summarizing overflow directory", "path", node.absPath, "error", err)
			// Leave itemCount = 0 as "unknown".
		} else {
			for _, e := range entries {
				name := e.Name()
				if e.IsDir() && defaultSkippedDirectory(name) {
					continue
				}
				if strings.HasPrefix(name, ".") {
					// Skip dot files from the overflow count as well.
					continue
				}
				itemCount++
			}
		}

		overflowDirs = append(overflowDirs, DirectoryOverflowInfo{
			DirPath:      node.absPath,
			RelativePath: node.relPath,
			FileCount:    itemCount,
			Partial:      false,
		})
	}

	// If we had a partial directory where we stopped mid-scan, include that
	// as an overflow entry too (if it isn't already in overflowDirs).
	if partialOverflow != nil {
		overflowDirs = append(overflowDirs, *partialOverflow)
	}

	// No files at all (e.g. empty dir, only dotfiles, or everything unreadable).
	if len(files) == 0 {
		return &WalkDirectoryWithFilesResult{
			DirPath:      dirPath,
			Files:        []PathInfo{},
			OverflowDirs: overflowDirs,
			MaxFiles:     maxFiles,
			TotalSize:    0,
			HasMore:      len(overflowDirs) > 0,
		}, nil
	}

	hasMore := len(overflowDirs) > 0

	return &WalkDirectoryWithFilesResult{
		DirPath:      dirPath,
		Files:        files,
		OverflowDirs: overflowDirs,
		MaxFiles:     maxFiles,
		TotalSize:    totalSize,
		HasMore:      hasMore,
	}, nil
}
