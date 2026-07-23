package fsdir

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type TraversalPolicy struct {
	ExcludedDirectoryNames []string
	SkipGitSubmodules      bool
}

type normalizedTraversalPolicy struct {
	excludedDirectoryNames map[string]struct{}
	skipGitSubmodules      bool
}

func DefaultTraversalPolicy() TraversalPolicy {
	return TraversalPolicy{
		ExcludedDirectoryNames: []string{
			".git",
			".hg",
			".svn",
			"node_modules",
			"vendor",
			"bower_components",
		},
		SkipGitSubmodules: true,
	}
}

func normalizeTraversalPolicy(input *TraversalPolicy) (normalizedTraversalPolicy, error) {
	value := DefaultTraversalPolicy()
	if input != nil {
		value.SkipGitSubmodules = input.SkipGitSubmodules
		if input.ExcludedDirectoryNames != nil {
			value.ExcludedDirectoryNames = append(
				[]string(nil),
				input.ExcludedDirectoryNames...,
			)
		}
	}

	output := normalizedTraversalPolicy{
		excludedDirectoryNames: make(map[string]struct{}, len(value.ExcludedDirectoryNames)),
		skipGitSubmodules:      value.SkipGitSubmodules,
	}

	for _, rawName := range value.ExcludedDirectoryNames {
		name := strings.TrimSpace(rawName)
		if name == "" ||
			name == "." ||
			name == ".." ||
			strings.ContainsAny(name, `/\`) {
			return normalizedTraversalPolicy{}, fmt.Errorf(
				"filesystem traversal exclusion %q is not a directory name",
				rawName,
			)
		}
		output.excludedDirectoryNames[strings.ToLower(name)] = struct{}{}
	}

	return output, nil
}

func (p normalizedTraversalPolicy) shouldSkipDirectory(name string) bool {
	_, found := p.excludedDirectoryNames[strings.ToLower(name)]
	return found
}

func (p normalizedTraversalPolicy) excludesLocator(locator string) bool {
	for segment := range strings.SplitSeq(locator, "/") {
		if p.shouldSkipDirectory(segment) {
			return true
		}
	}
	return false
}

func (p normalizedTraversalPolicy) isGitSubmoduleDirectory(directory string) bool {
	if !p.skipGitSubmodules {
		return false
	}

	gitFile := filepath.Join(directory, ".git")
	info, err := os.Lstat(gitFile)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}

	file, err := os.Open(gitFile)
	if err != nil {
		return false
	}
	content, readErr := io.ReadAll(io.LimitReader(file, 4097))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return false
	}

	return strings.HasPrefix(
		strings.ToLower(strings.TrimSpace(string(content))),
		"gitdir:",
	)
}
