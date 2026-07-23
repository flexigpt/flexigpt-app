package attachment

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"testing"
)

func TestWalkDirectoryWithFiles_CoreScenarios(t *testing.T) {
	type testCase struct {
		name     string
		setup    func(t *testing.T) (root string)
		maxFiles int
		// Root is whatever setup returned (may be empty for some tests).
		verify func(t *testing.T, root string, res *WalkDirectoryWithFilesResult, err error)
	}

	tests := []testCase{
		{
			name: "EmptyPath",
			setup: func(t *testing.T) string {
				t.Helper()
				return ""
			},
			maxFiles: 10,
			verify: func(t *testing.T, root string, res *WalkDirectoryWithFilesResult, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				if res == nil {
					t.Fatalf("expected non-nil result")
				}
				if res.DirPath != "" {
					t.Errorf("expected DirPath %q, got %q", "", res.DirPath)
				}
				if len(res.Files) != 0 {
					t.Errorf("expected 0 files, got %d", len(res.Files))
				}
				if len(res.OverflowDirs) != 0 {
					t.Errorf("expected 0 overflow dirs, got %d", len(res.OverflowDirs))
				}
				if res.HasMore {
					t.Errorf("expected HasMore=false, got true")
				}
				// MaxFiles is within bounds, so it should be preserved.
				if res.MaxFiles != 10 {
					t.Errorf("expected MaxFiles=10, got %d", res.MaxFiles)
				}
			},
		},
		{
			name: "EmptyDirectory_NoFiles",
			setup: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			maxFiles: 10,
			verify: func(t *testing.T, root string, res *WalkDirectoryWithFilesResult, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				if res == nil {
					t.Fatalf("expected non-nil result")
				}
				if res.DirPath == "" {
					t.Errorf("expected non-empty DirPath for actual directory")
				}
				if got := len(res.Files); got != 0 {
					t.Errorf("expected 0 files, got %d", got)
				}
				if got := len(res.OverflowDirs); got != 0 {
					t.Errorf("expected 0 overflow dirs, got %d", got)
				}
				if res.HasMore {
					t.Errorf("expected HasMore=false, got true")
				}
				if res.TotalSize != 0 {
					t.Errorf("expected TotalSize=0, got %d", res.TotalSize)
				}
			},
		},
		{
			name: "SimpleUnderLimit_AllIncluded",
			setup: func(t *testing.T) string {
				t.Helper()
				root := t.TempDir()
				mustWriteFile(t, root, "f1.txt", 1)
				mustWriteFile(t, root, "f2.txt", 2)
				mustWriteFile(t, root, "f3.txt", 3)
				return root
			},
			maxFiles: 10,
			verify: func(t *testing.T, root string, res *WalkDirectoryWithFilesResult, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				if res == nil {
					t.Fatalf("expected non-nil result")
				}
				if res.HasMore {
					t.Errorf("expected HasMore=false, got true")
				}
				names := namesFromPathInfos(res.Files)
				wantNames := []string{"f1.txt", "f2.txt", "f3.txt"}
				if len(names) != len(wantNames) {
					t.Fatalf("expected %d files, got %d (%v)", len(wantNames), len(names), names)
				}
				for i, w := range wantNames {
					if names[i] != w {
						t.Errorf("expected names[%d]=%q, got %q", i, w, names[i])
					}
				}
				if sum := sumSizes(res.Files); sum != res.TotalSize {
					t.Errorf("TotalSize mismatch: result=%d, recomputed=%d", res.TotalSize, sum)
				}
				if res.TotalSize != int64(1+2+3) {
					t.Errorf("expected TotalSize=6, got %d", res.TotalSize)
				}
				if len(res.OverflowDirs) != 0 {
					t.Errorf("expected 0 overflow dirs, got %d", len(res.OverflowDirs))
				}
			},
		},
		{
			name: "SkipDotEntries_AndKnownJunkDirs",
			setup: func(t *testing.T) string {
				t.Helper()
				root := t.TempDir()
				// Dotfile at root (should be skipped).
				mustWriteFile(t, root, ".hidden_root", 10)

				// Dot directories, including VCS metadata, are skipped entirely.
				dotGit := mustMkdir(t, root, ".git")
				mustWriteFile(t, dotGit, "ignored.txt", 20)
				mustWriteFile(t, dotGit, ".ignored_hidden", 5)

				// Dependency and generated-output directories are also skipped.
				nodeModules := mustMkdir(t, root, "node_modules")
				mustWriteFile(t, nodeModules, "dependency.js", 60)
				vendor := mustMkdir(t, root, "vendor")
				mustWriteFile(t, vendor, "dependency.go", 70)
				build := mustMkdir(t, root, "build")
				mustWriteFile(t, build, "generated.js", 80)

				// Visible root file.
				mustWriteFile(t, root, "visible1.txt", 30)

				// Subdir with both visible and dotfile.
				sub := mustMkdir(t, root, "sub")
				mustWriteFile(t, sub, ".hidden_sub", 40)  // skipped
				mustWriteFile(t, sub, "visible2.txt", 50) // included
				return root
			},
			maxFiles: 10,
			verify: func(t *testing.T, root string, res *WalkDirectoryWithFilesResult, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				if res == nil {
					t.Fatalf("expected non-nil result")
				}
				if res.HasMore {
					t.Errorf("expected HasMore=false, got true")
				}
				names := namesFromPathInfos(res.Files)
				wantNames := []string{"visible1.txt", "visible2.txt"}
				if len(names) != len(wantNames) {
					t.Fatalf("expected %d visible files, got %d (%v)", len(wantNames), len(names), names)
				}
				for i, w := range wantNames {
					if names[i] != w {
						t.Errorf("expected names[%d]=%q, got %q", i, w, names[i])
					}
				}
				// Ensure dotfiles are not present in Files.
				for _, n := range names {
					if strings.HasPrefix(n, ".") {
						t.Errorf("unexpected dotfile in results: %q", n)
					}
				}
				if len(res.OverflowDirs) != 0 {
					t.Errorf("expected 0 overflow dirs, got %d", len(res.OverflowDirs))
				}
				// TotalSize should be sum of included files only.
				if res.TotalSize != int64(30+50) {
					t.Errorf("expected TotalSize=100, got %d", res.TotalSize)
				}
			},
		},
		{
			name: "RootPartialWithSubdirs",
			setup: func(t *testing.T) string {
				t.Helper()
				root := t.TempDir()

				// Three root files.
				mustWriteFile(t, root, "root1.txt", 10)
				mustWriteFile(t, root, "root2.txt", 20)
				mustWriteFile(t, root, "root3.txt", 30)

				// Two subdirs (each with one file) to prove they exist but
				// are never walked when we hit the limit in root.
				sub1 := mustMkdir(t, root, "sub1")
				mustWriteFile(t, sub1, "s1.txt", 5)

				sub2 := mustMkdir(t, root, "sub2")
				mustWriteFile(t, sub2, "s2.txt", 5)

				return root
			},
			maxFiles: 2,
			verify: func(t *testing.T, root string, res *WalkDirectoryWithFilesResult, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				if res == nil {
					t.Fatalf("expected non-nil result")
				}
				if res.MaxFiles != 2 {
					t.Errorf("expected MaxFiles=2, got %d", res.MaxFiles)
				}
				if len(res.Files) != 2 {
					t.Fatalf("expected 2 files kept, got %d", len(res.Files))
				}
				names := namesFromPathInfos(res.Files)
				for _, n := range names {
					if !strings.HasPrefix(n, "root") {
						t.Errorf("expected only root files, got %q", n)
					}
				}

				if len(res.OverflowDirs) != 1 {
					t.Fatalf(
						"expected 1 overflow dir (root partial), got %d (%+v)",
						len(res.OverflowDirs),
						res.OverflowDirs,
					)
				}
				ov := res.OverflowDirs[0]
				if ov.DirPath != res.DirPath {
					t.Errorf("expected overflow DirPath=%q, got %q", res.DirPath, ov.DirPath)
				}
				if ov.RelativePath != "" {
					t.Errorf("expected overflow RelativePath=\"\" for root, got %q", ov.RelativePath)
				}
				if !ov.Partial {
					t.Errorf("expected Partial=true for root overflow, got false")
				}
				// Remaining entries in root: 1 remaining file + 2 subdirs = 3.
				if ov.FileCount != 3 {
					t.Errorf("expected overflow FileCount=3, got %d", ov.FileCount)
				}
				if !res.HasMore {
					t.Errorf("expected HasMore=true due to root-level truncation, got false")
				}
			},
		},
		{
			name: "PartialSubdirAndOverflowQueue",
			setup: func(t *testing.T) string {
				t.Helper()
				root := t.TempDir()

				// One root file.
				mustWriteFile(t, root, "root.txt", 1)

				// Three subdirs, each with two files.
				for _, d := range []string{"a", "b", "c"} {
					dir := mustMkdir(t, root, d)
					mustWriteFile(t, dir, d+"1.txt", 1)
					mustWriteFile(t, dir, d+"2.txt", 1)
					mustWriteFile(t, dir, d+"3.txt", 1)
				}
				return root
			},
			maxFiles: 3,
			verify: func(t *testing.T, root string, res *WalkDirectoryWithFilesResult, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				if res == nil {
					t.Fatalf("expected non-nil result")
				}
				if !res.HasMore {
					t.Errorf("expected HasMore=true due to overflow, got false")
				}
				if res.MaxFiles != 3 {
					t.Errorf("expected MaxFiles=3, got %d", res.MaxFiles)
				}
				if len(res.Files) != 3 {
					t.Fatalf("expected 3 files kept, got %d", len(res.Files))
				}

				names := namesFromPathInfos(res.Files)
				if !slices.Contains(names, "root.txt") {
					t.Fatalf("expected root.txt to be present in files, got %v", names)
				}
				var others []string
				for _, n := range names {
					if n != "root.txt" {
						others = append(others, n)
					}
				}
				if len(others) != 2 {
					t.Fatalf("expected 2 non-root files, got %d (%v)", len(others), others)
				}
				// Both non-root files should come from the same subdir (a, b, or c).
				prefix := string(others[0][0])
				if !slices.Contains([]string{"a", "b", "c"}, prefix) {
					t.Errorf("expected other files to come from a/b/c, got prefix %q in %q", prefix, others[0])
				}
				for _, n := range others[1:] {
					if !strings.HasPrefix(n, prefix) {
						t.Errorf("expected all non-root files from same dir, got %v", others)
					}
				}

				if len(res.OverflowDirs) != 3 {
					t.Fatalf(
						"expected 3 overflow dirs for a, b, c; got %d (%+v)",
						len(res.OverflowDirs),
						res.OverflowDirs,
					)
				}

				// Collect by relative path.
				ovByRel := make(map[string]DirectoryOverflowInfo)
				for _, ov := range res.OverflowDirs {
					ovByRel[ov.RelativePath] = ov
				}
				for _, d := range []string{"a", "b", "c"} {
					if _, ok := ovByRel[d]; !ok {
						t.Errorf("missing overflow entry for %q", d)
					}
				}

				partialCount := 0
				for _, ov := range res.OverflowDirs {
					if ov.Partial {
						partialCount++
						// The partially-walked dir has had 2 of its files consumed, no subdirs.
						if ov.FileCount != 1 {
							t.Errorf("expected partial overflow FileCount=0, got %d for %+v", ov.FileCount, ov)
						}
					} else if ov.FileCount != 3 {
						t.Errorf("expected full overflow FileCount=2, got %d for %+v", ov.FileCount, ov)
					}
				}

				if partialCount != 1 {
					t.Errorf("expected exactly one partial overflow dir, got %d", partialCount)
				}
			},
		},
		{
			name: "PartialSubdirWithRemainingFilesAndSubdirs",
			setup: func(t *testing.T) string {
				t.Helper()
				root := t.TempDir()

				mustWriteFile(t, root, "root.txt", 1)

				dir := mustMkdir(t, root, "dir")
				mustWriteFile(t, dir, "f1.txt", 1)
				mustWriteFile(t, dir, "f2.txt", 1)
				mustWriteFile(t, dir, "f3.txt", 1)

				// Two subdirectories inside dir.
				mustMkdir(t, dir, "sub1")
				mustMkdir(t, dir, "sub2")

				return root
			},
			maxFiles: 2,
			verify: func(t *testing.T, root string, res *WalkDirectoryWithFilesResult, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				if res == nil {
					t.Fatalf("expected non-nil result")
				}

				if len(res.Files) != 2 {
					t.Fatalf("expected 2 files kept, got %d", len(res.Files))
				}
				names := namesFromPathInfos(res.Files)
				if !slices.Contains(names, "root.txt") {
					t.Fatalf("expected root.txt to be present, got %v", names)
				}
				var others []string
				for _, n := range names {
					if n != "root.txt" {
						others = append(others, n)
					}
				}
				if len(others) != 1 {
					t.Fatalf("expected 1 non-root file, got %d (%v)", len(others), others)
				}
				if !strings.HasPrefix(others[0], "f") {
					t.Errorf("expected non-root file from dir to be f*.txt, got %q", others[0])
				}

				if len(res.OverflowDirs) != 1 {
					t.Fatalf("expected 1 overflow dir for 'dir', got %d (%+v)", len(res.OverflowDirs), res.OverflowDirs)
				}
				ov := res.OverflowDirs[0]
				if ov.RelativePath != "dir" {
					t.Errorf("expected overflow.RelativePath='dir', got %q", ov.RelativePath)
				}
				if !ov.Partial {
					t.Errorf("expected Partial=true for dir overflow, got false")
				}
				// Remaining entries inside dir: two remaining files + two subdirs = 4.
				if ov.FileCount != 4 {
					t.Errorf("expected overflow FileCount=4, got %d", ov.FileCount)
				}
				if !res.HasMore {
					t.Errorf("expected HasMore=true due to dir truncation, got false")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := ""
			if tc.setup != nil {
				root = tc.setup(t)
			}
			res, err := WalkDirectoryWithFiles(t.Context(), root, tc.maxFiles)
			tc.verify(t, root, res, err)
		})
	}
}

func TestWalkDirectoryWithFiles_NonExistentDir(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "does_not_exist")
	res, err := WalkDirectoryWithFiles(t.Context(), root, 10)
	if err == nil {
		t.Fatalf("expected non-nil error for non-existent dir, got nil")
	}
	if res != nil {
		t.Fatalf("expected nil result on error, got %+v", res)
	}
}

func TestWalkDirectoryWithFiles_ContextCanceled(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mustWriteFile(t, root, "file.txt", 10)

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // cancel before walking

	res, err := WalkDirectoryWithFiles(ctx, root, 10)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if res != nil {
		t.Fatalf("expected nil result when context is canceled, got %+v", res)
	}
}

func TestWalkDirectoryWithFiles_MaxFilesClamped(t *testing.T) {
	root := t.TempDir()

	// Create more files than the global maxTotalDirWalkFiles limit.
	totalFiles := maxTotalDirWalkFiles + 5
	for i := range totalFiles {
		name := fmt.Sprintf("f%03d.txt", i)
		mustWriteFile(t, root, name, 1)
	}

	type testCase struct {
		name     string
		maxFiles int
	}

	tests := []testCase{
		{name: "Zero", maxFiles: 0},
		{name: "Negative", maxFiles: -5},
		{name: "AboveGlobal", maxFiles: maxTotalDirWalkFiles + 100},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := WalkDirectoryWithFiles(t.Context(), root, tc.maxFiles)
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if res == nil {
				t.Fatalf("expected non-nil result")
			}
			if res.MaxFiles != maxTotalDirWalkFiles {
				t.Errorf("expected MaxFiles=%d, got %d", maxTotalDirWalkFiles, res.MaxFiles)
			}
			if len(res.Files) != maxTotalDirWalkFiles {
				t.Fatalf("expected %d files kept, got %d", maxTotalDirWalkFiles, len(res.Files))
			}
			if len(res.OverflowDirs) != 1 {
				t.Fatalf(
					"expected 1 overflow dir (root partial), got %d (%+v)",
					len(res.OverflowDirs),
					res.OverflowDirs,
				)
			}
			ov := res.OverflowDirs[0]
			if !ov.Partial {
				t.Errorf("expected Partial=true for root overflow, got false")
			}
			if ov.DirPath != res.DirPath {
				t.Errorf("expected overflow DirPath=%q, got %q", res.DirPath, ov.DirPath)
			}
			if ov.RelativePath != "" {
				t.Errorf("expected overflow RelativePath=\"\" for root, got %q", ov.RelativePath)
			}
			// Remaining files in root that were not included.
			expectedRemaining := totalFiles - maxTotalDirWalkFiles
			if ov.FileCount != expectedRemaining {
				t.Errorf("expected overflow FileCount=%d, got %d", expectedRemaining, ov.FileCount)
			}
			if !res.HasMore {
				t.Errorf("expected HasMore=true when clamping occurs, got false")
			}
		})
	}
}

func TestWalkDirectoryWithFiles_UnreadableSubdirOverflow_NoFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-based test skipped on Windows")
	}

	root := t.TempDir()
	sub := mustMkdir(t, root, "sub")

	// Make sub unreadable.
	if err := os.Chmod(sub, 0o000); err != nil {
		t.Skipf("chmod not supported: %v", err)
	}
	defer func() { _ = os.Chmod(sub, 0o755) }() // best-effort restore so cleanup works

	res, err := WalkDirectoryWithFiles(t.Context(), root, 10)
	if err != nil {
		t.Fatalf("expected nil error walking root, got %v", err)
	}
	if res == nil {
		t.Fatalf("expected non-nil result")
	}

	if len(res.Files) != 0 {
		t.Fatalf("expected 0 files (no readable files), got %d", len(res.Files))
	}
	if len(res.OverflowDirs) != 1 {
		t.Fatalf("expected 1 overflow dir for unreadable sub, got %d (%+v)", len(res.OverflowDirs), res.OverflowDirs)
	}
	ov := res.OverflowDirs[0]
	if ov.RelativePath != "sub" {
		t.Errorf("expected overflow.RelativePath='sub', got %q", ov.RelativePath)
	}
	if ov.Partial {
		t.Errorf("expected Partial=false for unreadable dir overflow, got true")
	}
	// FileCount is 0 for "unknown" due to read error.
	if ov.FileCount != 0 {
		t.Errorf("expected overflow FileCount=0 for unreadable dir, got %d", ov.FileCount)
	}
	if !res.HasMore {
		t.Errorf("expected HasMore=true when there is an unreadable subtree, got false")
	}
}

func mustWriteFile(t *testing.T, dir, name string, size int) string {
	t.Helper()
	full := filepath.Join(dir, name)
	data := bytes.Repeat([]byte("x"), size)
	if err := os.WriteFile(full, data, 0o600); err != nil {
		t.Fatalf("failed to write file %q: %v", full, err)
	}
	return full
}

func mustMkdir(t *testing.T, dir, name string) string {
	t.Helper()
	full := filepath.Join(dir, name)
	if err := os.Mkdir(full, 0o755); err != nil {
		t.Fatalf("failed to mkdir %q: %v", full, err)
	}
	return full
}

func namesFromPathInfos(files []PathInfo) []string {
	names := make([]string, 0, len(files))
	for _, f := range files {
		names = append(names, f.Name)
	}
	sort.Strings(names)
	return names
}

func sumSizes(files []PathInfo) int64 {
	var total int64
	for _, f := range files {
		total += f.Size
	}
	return total
}
