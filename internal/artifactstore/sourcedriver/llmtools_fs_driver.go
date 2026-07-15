package sourcedriver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
	"github.com/flexigpt/llmtools-go/fstool"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"
)

// llmToolsFSDirectoryDriver delegates every local filesystem interaction to
// LLMTools FSTool. It contains no direct os, io/fs, or filepath filesystem I/O.
type llmToolsFSDirectoryDriver struct{}

func NewLLMToolsFSDirectoryDriver() spec.SourceDriver { return &llmToolsFSDirectoryDriver{} }

func (*llmToolsFSDirectoryDriver) Kind() spec.SourceKind { return spec.SourceKindFSDirectory }

func (*llmToolsFSDirectoryDriver) ValidateConfig(_ context.Context, raw json.RawMessage) []spec.Diagnostic {
	_, err := decodeLLMToolsFSConfig(raw)
	if err == nil {
		return nil
	}
	return []spec.Diagnostic{{
		Severity: spec.DiagnosticSeverityError,
		Code:     "artifactstore.source.config.invalid",
		Message:  err.Error(),
	}}
}

func (d *llmToolsFSDirectoryDriver) Snapshot(
	ctx context.Context,
	source spec.ArtifactSource,
) (spec.SourceGeneration, error) {
	entries := make([]spec.SourceEntry, 0)
	if err := d.Walk(ctx, source, ".", func(ctx context.Context, entry spec.SourceEntry) error {
		entries = append(entries, entry)
		return nil
	}); err != nil {
		return "", err
	}
	if len(entries) > spec.DefaultMaxScanEntries {
		return "", fmt.Errorf(
			"%w: source snapshot exceeds %d entries",
			spec.ErrInvalidRequest,
			spec.DefaultMaxScanEntries,
		)
	}
	sort.Slice(entries, func(left, right int) bool { return entries[left].Locator < entries[right].Locator })
	hash := sha256.New()
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		depth := 1 + strings.Count(string(entry.Locator), "/")
		if depth > spec.DefaultMaxTraversalDepth {
			return "", fmt.Errorf(
				"%w: source snapshot exceeds depth %d at %q",
				spec.ErrInvalidRequest,
				spec.DefaultMaxTraversalDepth,
				entry.Locator,
			)
		}
		entryType := "o"
		switch {
		case entry.IsDirectory:
			entryType = "d"
		case entry.IsRegular:
			entryType = "f"
		}
		_, _ = fmt.Fprintf(
			hash,
			"%s\x00%s\x00%d\x00%s\x00",
			entryType,
			entry.Locator,
			entry.SizeBytes,
			entry.ModifiedAt.UTC().Format(time.RFC3339Nano),
		)
	}
	return spec.SourceGeneration("sha256:" + hex.EncodeToString(hash.Sum(nil))), nil
}

func (d *llmToolsFSDirectoryDriver) Open(
	ctx context.Context,
	source spec.ArtifactSource,
	locator spec.SourceLocator,
) (io.ReadCloser, error) {
	tool, config, err := d.toolFor(source)
	if err != nil {
		return nil, err
	}
	path, err := sourcePath(config.RootPath, locator)
	if err != nil {
		return nil, err
	}
	content, err := readLLMToolsFile(ctx, tool, path)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(content)), nil
}

func (d *llmToolsFSDirectoryDriver) Walk(
	ctx context.Context,
	source spec.ArtifactSource,
	root spec.SourceLocator,
	walk spec.WalkFunc,
) error {
	if walk == nil {
		return fmt.Errorf("%w: walk callback is nil", spec.ErrInvalidRequest)
	}
	rootEntry, err := d.Stat(ctx, source, root)
	if err != nil {
		return err
	}
	if !rootEntry.IsDirectory {
		return fmt.Errorf("%w: walk root %q is not a directory", spec.ErrInvalidRequest, root)
	}
	var visit func(spec.SourceLocator) error
	visit = func(directory spec.SourceLocator) error {
		entries, err := d.ReadDir(ctx, source, directory)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := walk(ctx, entry); err != nil {
				return err
			}
			if entry.IsDirectory {
				if err := visit(entry.Locator); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return visit(root)
}

func (d *llmToolsFSDirectoryDriver) ReadDir(
	ctx context.Context,
	source spec.ArtifactSource,
	locator spec.SourceLocator,
) ([]spec.SourceEntry, error) {
	tool, config, err := d.toolFor(source)
	if err != nil {
		return nil, err
	}
	path, err := sourcePath(config.RootPath, locator)
	if err != nil {
		return nil, err
	}
	out, err := tool.ListDirectory(ctx, fstool.ListDirectoryArgs{
		Path:              path,
		IncludeDotEntries: true,
		Kind:              fstool.ListDirectoryEntryKindAll,
		MaxEntries:        5000,
	})
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, fmt.Errorf("%w: source directory %q", spec.ErrNotFound, locator)
	}
	if out.ReachedMaxEntries {
		return nil, fmt.Errorf(
			"%w: directory %q exceeds the LLMTools listing limit",
			spec.ErrInvalidRequest,
			locator,
		)
	}
	entries := make([]spec.SourceEntry, 0, len(out.Items))
	for _, item := range out.Items {
		childLocator, err := joinSourceLocator(locator, item.Name)
		if err != nil {
			return nil, err
		}
		if item.Kind == fstool.ListDirectoryEntryKindOther && !config.FollowSymlinks {
			entries = append(entries, spec.SourceEntry{
				Locator: childLocator,
				Name:    item.Name,
			})
			continue
		}
		child, err := d.Stat(ctx, source, childLocator)
		if err != nil {
			return nil, err
		}
		switch item.Kind {
		case fstool.ListDirectoryEntryKindFile:
			child.IsDirectory = false
			child.IsRegular = true
		case fstool.ListDirectoryEntryKindDirectory:
			child.IsDirectory = true
			child.IsRegular = false
		case fstool.ListDirectoryEntryKindOther:
			// FollowSymlinks was explicitly enabled. StatPath describes the
			// resolved target, while traversal limits prevent unbounded cycles.
		default:
			return nil, fmt.Errorf(
				"%w: unknown LLMTools directory entry kind %q",
				spec.ErrInvalidRequest,
				item.Kind,
			)
		}
		entries = append(entries, child)
	}
	sort.Slice(entries, func(left, right int) bool {
		return entries[left].Locator < entries[right].Locator
	})
	return entries, nil
}

func (d *llmToolsFSDirectoryDriver) Stat(
	ctx context.Context,
	source spec.ArtifactSource,
	locator spec.SourceLocator,
) (spec.SourceEntry, error) {
	tool, config, err := d.toolFor(source)
	if err != nil {
		return spec.SourceEntry{}, err
	}
	path, err := sourcePath(config.RootPath, locator)
	if err != nil {
		return spec.SourceEntry{}, err
	}
	out, err := tool.StatPath(ctx, fstool.StatPathArgs{Path: path})
	if err != nil {
		return spec.SourceEntry{}, err
	}
	if out == nil || !out.Exists {
		return spec.SourceEntry{}, fmt.Errorf("%w: source locator %q", spec.ErrNotFound, locator)
	}
	modified := time.Time{}
	if out.ModTime != nil {
		modified = out.ModTime.UTC()
	}
	return spec.SourceEntry{
		Locator:     locator,
		Name:        out.Name,
		SizeBytes:   out.SizeBytes,
		ModifiedAt:  modified,
		IsDirectory: out.IsDir,
		IsRegular:   !out.IsDir,
	}, nil
}

type llmToolsFSConfig struct {
	RootPath       string
	FollowSymlinks bool
}

func (d *llmToolsFSDirectoryDriver) toolFor(source spec.ArtifactSource) (*fstool.FSTool, llmToolsFSConfig, error) {
	if source.Kind != spec.SourceKindFSDirectory {
		return nil, llmToolsFSConfig{}, fmt.Errorf(
			"%w: fs driver received source kind %q",
			spec.ErrInvalidRequest,
			source.Kind,
		)
	}
	config, err := decodeLLMToolsFSConfig(source.Config)
	if err != nil {
		return nil, llmToolsFSConfig{}, err
	}
	tool, err := fstool.NewFSTool(
		fstool.WithAllowedRoots([]string{config.RootPath}),
		fstool.WithWorkBaseDir(config.RootPath),
		fstool.WithBlockSymlinks(!config.FollowSymlinks),
	)
	if err != nil {
		return nil, llmToolsFSConfig{}, err
	}
	return tool, config, nil
}

func decodeLLMToolsFSConfig(raw json.RawMessage) (llmToolsFSConfig, error) {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	var config spec.FSDirectorySourceConfig
	if err := decoder.Decode(&config); err != nil {
		return llmToolsFSConfig{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return llmToolsFSConfig{}, errors.New(
				"filesystem source config contains trailing JSON",
			)
		}
		return llmToolsFSConfig{}, err
	}
	if err := validate.ValidateFSDirectorySourceConfig(config); err != nil {
		return llmToolsFSConfig{}, err
	}
	return llmToolsFSConfig{RootPath: config.RootPath, FollowSymlinks: config.FollowSymlinks}, nil
}

func readLLMToolsFile(ctx context.Context, tool *fstool.FSTool, path string) ([]byte, error) {
	out, err := tool.ReadFile(ctx, fstool.ReadFileArgs{Path: path, Encoding: "binary"})
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: empty LLMTools file result", spec.ErrNotFound)
	}
	for _, item := range out {
		switch item.Kind {
		case llmtoolsSpec.ToolOutputKindFile:
			if item.FileItem == nil {
				continue
			}
			return base64.StdEncoding.DecodeString(item.FileItem.FileData)
		case llmtoolsSpec.ToolOutputKindImage:
			if item.ImageItem == nil {
				continue
			}
			return base64.StdEncoding.DecodeString(item.ImageItem.ImageData)
		case llmtoolsSpec.ToolOutputKindText:
			if item.TextItem != nil {
				return []byte(item.TextItem.Text), nil
			}
		default:
		}
	}
	return nil, fmt.Errorf("%w: unreadable LLMTools file result", spec.ErrInvalidRequest)
}

func sourcePath(root string, locator spec.SourceLocator) (string, error) {
	if locator == "." {
		return root, nil
	}
	value := string(locator)
	if value == "" || strings.Contains(value, "\\") || strings.Contains(value, ":") || filepath.IsAbs(value) {
		return "", fmt.Errorf("%w: invalid source locator %q", spec.ErrInvalidRequest, locator)
	}
	candidate := filepath.Join(root, filepath.FromSlash(value))
	relative, err := filepath.Rel(root, candidate)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) ||
		filepath.IsAbs(relative) {
		return "", fmt.Errorf("%w: source locator escapes root", spec.ErrInvalidRequest)
	}
	return candidate, nil
}

func joinSourceLocator(parent spec.SourceLocator, name string) (spec.SourceLocator, error) {
	if name == "" || strings.ContainsAny(name, `/\\:`) {
		return "", fmt.Errorf("%w: invalid source entry name %q", spec.ErrInvalidRequest, name)
	}
	if parent == "." {
		return spec.SourceLocator(name), nil
	}
	return spec.SourceLocator(string(parent) + "/" + name), nil
}
