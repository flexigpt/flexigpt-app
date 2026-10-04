// Package managedfs provides the managed writable filesystem Source driver.
//
// It owns managed-root layout, staging names, physical file modes, and atomic
// complete-package replacement. Artifact-family package contents remain
// outside this provider.
package managedfs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/flexigpt/mapstore-go"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/internal/mapstoreio"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/driver"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const Kind sourceModel.SourceKind = "managed-directory"

var errPackageDifferent = errors.New("managed package content differs")

type config struct{}

// Adapter stores each managed Source beneath:
//
//	<base>/<root-storage-key>/<source-storage-key>
//
// A package below that source root is addressed generically as:
//
//	<package-kind>/<name>/<version>
//
// Artifact families own package-kind values and all package-internal file
// conventions. This adapter owns no artifact-family directory or filename.
type Adapter struct {
	base        string
	stagingBase string
	filesystem  *fsdir.Adapter
	mu          sync.Mutex
}

func New(
	base string,
	stagingBase string,
) (*Adapter, error) {
	if strings.TrimSpace(base) == "" {
		return nil, fmt.Errorf(
			"%w: managed Source base directory is empty",
			spec.ErrInvalid,
		)
	}
	absolute, err := filepath.Abs(base)
	if err != nil {
		return nil, err
	}
	stagingAbsolute, err := filepath.Abs(stagingBase)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(stagingBase) == "" {
		return nil, fmt.Errorf(
			"%w: managed Source staging directory is empty",
			spec.ErrInvalid,
		)
	}
	// Staging is outside the portable Source tree. Managed package generation
	// must therefore include every regular file below the content root.
	policy := fsdir.TraversalPolicy{
		ExcludedDirectoryNames: []string{},
		SkipGitSubmodules:      false,
	}
	filesystem, err := fsdir.NewWithTraversalPolicy(&policy)
	if err != nil {
		return nil, err
	}
	return &Adapter{
		base:        filepath.Clean(absolute),
		stagingBase: filepath.Clean(stagingAbsolute),
		filesystem:  filesystem,
	}, nil
}

func (*Adapter) Kind() sourceModel.SourceKind {
	return Kind
}

func (a *Adapter) ResolveLocalPath(
	ctx context.Context,
	value sourceModel.Source,
	locator spec.Locator,
) (string, error) {
	if err := a.validateSource(ctx, value); err != nil {
		return "", err
	}
	root, err := a.sourceRootPath(value, false)
	if err != nil {
		return "", err
	}
	filesystemValue, err := a.filesystemSource(value, root)
	if err != nil {
		return "", err
	}
	return a.filesystem.ResolveLocalPath(ctx, filesystemValue, locator)
}

func (a *Adapter) BootstrapManagedSource(
	ctx context.Context,
	value sourceModel.Source,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.validateSource(ctx, value); err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if _, err := a.sourceRootPath(value, true); err != nil {
		return err
	}
	return nil
}

// RemoveManagedRoot removes all managed Source directories below one
// application-owned Root. The Install flow authorizes this operation before it
// reaches the adapter.
func (a *Adapter) RemoveManagedRoot(
	ctx context.Context,
	rootStorageKey spec.StorageKey,
) error {
	if a == nil {
		return spec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: managed root removal context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	root, err := a.managedRootPath(rootStorageKey)
	if err != nil {
		return err
	}
	stagingRoot, err := a.managedStagingRootPath(rootStorageKey)
	if err != nil {
		return err
	}
	return errors.Join(
		os.RemoveAll(root),
		os.RemoveAll(stagingRoot),
	)
}

func (a *Adapter) DiscardBootstrappedManagedSource(
	ctx context.Context,
	value sourceModel.Source,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.validateSource(ctx, value); err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	root, err := a.sourceRootPath(value, false)
	if err != nil {
		return err
	}
	info, err := os.Stat(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf(
			"%w: bootstrapped managed Source path is not a directory",
			spec.ErrInvalid,
		)
	}
	empty, err := managedDirectoryEmpty(root)
	if err != nil {
		return err
	}
	if !empty {
		return fmt.Errorf(
			"%w: refusing to discard a managed Source with published package content",
			spec.ErrConflict,
		)
	}
	if err := os.Remove(root); err != nil {
		return err
	}
	stagingRoot, err := a.sourceStagingPath(value, false)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(stagingRoot); err != nil {
		return err
	}
	parent := filepath.Dir(root)
	empty, err = managedDirectoryEmpty(parent)
	if err != nil {
		return err
	}
	if empty {
		if err := os.Remove(parent); err != nil {
			return err
		}
	}
	stagingParent := filepath.Dir(stagingRoot)
	if empty, err := managedDirectoryEmpty(stagingParent); err == nil && empty {
		_ = os.Remove(stagingParent)
	}
	return nil
}

func (a *Adapter) PublishPackage(
	ctx context.Context,
	value sourceModel.Source,
	publication managedpackageModel.ManagedPackagePublication,
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := a.validateSource(ctx, value); err != nil {
		return "", err
	}
	files := publication.Files
	directory, err := publication.Address.Directory()
	if err != nil {
		return "", err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	root, err := a.sourceRootPath(value, true)
	if err != nil {
		return "", err
	}
	target, err := managedPackagePath(
		root,
		directory,
		false,
	)
	if err != nil {
		return "", err
	}
	exists, equivalent, err := equivalentPackage(target, files)
	if err != nil {
		return "", err
	}
	if publication.ExpectedGeneration != "" {
		if err := spec.ValidateSourceGeneration(
			publication.ExpectedGeneration,
		); err != nil {
			return "", err
		}
		current, err := a.confirmedGeneration(ctx, value)
		if err != nil {
			return "", err
		}
		if current != publication.ExpectedGeneration {
			return "", fmt.Errorf(
				"%w: managed Source changed before package publication",
				spec.ErrConflict,
			)
		}
	}
	if exists && equivalent {
		return a.confirmedGeneration(ctx, value)
	}

	if exists && publication.ExpectedGeneration == "" {
		return "", fmt.Errorf(
			"%w: replacing managed package %v requires an expected generation",
			spec.ErrConflict,
			publication.Address,
		)
	}
	target, err = managedPackagePath(
		root,
		directory,
		true,
	)
	if err != nil {
		return "", err
	}

	stagingRoot, err := a.sourceStagingPath(value, true)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(stagingRoot, managedDirectoryMode); err != nil {
		return "", err
	}

	temporary, err := os.MkdirTemp(
		stagingRoot,
		managedPackageTemporaryPrefix,
	)
	if err != nil {
		return "", err
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(temporary)
		}
	}()

	if err := writeManagedPackageFiles(
		ctx,
		temporary,
		files,
	); err != nil {
		return "", err
	}

	var previousPackage string
	if exists {
		previousPackage, err = os.MkdirTemp(
			stagingRoot,
			managedPackagePreviousPrefix,
		)
		if err != nil {
			return "", err
		}
		if err := os.Remove(previousPackage); err != nil {
			return "", err
		}
		if err := os.Rename(target, previousPackage); err != nil {
			return "", fmt.Errorf(
				"stage previous managed package for replacement: %w",
				err,
			)
		}
	}

	restorePrevious := func(cause error) error {
		if previousPackage == "" {
			return cause
		}

		removeErr := os.RemoveAll(target)
		restoreErr := os.Rename(previousPackage, target)
		return errors.Join(cause, removeErr, restoreErr)
	}

	// Renaming the fully staged package is the source-side publication
	// boundary. For replacement, the previous complete package is retained in
	// staging until the new package has been installed successfully.
	if err := os.Rename(temporary, target); err != nil {
		targetExists, targetEquivalent, verifyErr := equivalentPackage(
			target,
			files,
		)
		if verifyErr == nil && targetExists && targetEquivalent {
			committed = true

			if previousPackage != "" {
				_ = os.RemoveAll(previousPackage)
			}
			return a.confirmedGeneration(ctx, value)
		}
		return "", restorePrevious(
			fmt.Errorf("publish managed package: %w", err),
		)
	}

	committed = true

	if previousPackage != "" {
		if err := os.RemoveAll(previousPackage); err != nil {
			return "", fmt.Errorf(
				"remove replaced managed package staging data: %w",
				err,
			)
		}
	}
	return a.confirmedGeneration(ctx, value)
}

func (a *Adapter) RemovePackage(
	ctx context.Context,
	value sourceModel.Source,
	address managedpackageModel.ManagedPackageAddress,
	expectedGeneration string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.validateSource(ctx, value); err != nil {
		return err
	}
	if err := address.Validate(); err != nil {
		return err
	}
	directory, err := address.Directory()
	if err != nil {
		return err
	}
	if err := spec.ValidateSourceGeneration(expectedGeneration); err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	current, err := a.confirmedGeneration(ctx, value)
	if err != nil {
		return err
	}
	if current != expectedGeneration {
		return fmt.Errorf(
			"%w: managed Source changed before package removal",
			spec.ErrConflict,
		)
	}
	root, err := a.sourceRootPath(value, false)
	if err != nil {
		return err
	}
	target, err := managedPackagePath(root, directory, false)
	if err != nil {
		return err
	}
	info, err := os.Stat(target)
	if errors.Is(err, os.ErrNotExist) {
		// Removal is idempotent after a prior successful source-side rename.
		// Components.RemoveManagedPackage still acknowledges the current
		// generation in metadata, so a retry converges after a crash between
		// package removal and Source revision publication.
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf(
			"%w: managed package is not a directory",
			spec.ErrInvalid,
		)
	}

	stagingRoot, err := a.sourceStagingPath(value, true)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(stagingRoot, managedDirectoryMode); err != nil {
		return err
	}

	tombstone, err := os.MkdirTemp(
		stagingRoot,
		managedPackageRemovalPrefix,
	)
	if err != nil {
		return err
	}
	if err := os.Remove(tombstone); err != nil {
		return err
	}
	if err := os.Rename(target, tombstone); err != nil {
		return err
	}
	if err := os.RemoveAll(tombstone); err != nil {
		return errors.Join(err, os.Rename(tombstone, target))
	}
	if err := pruneEmptyManagedParents(
		root,
		filepath.Dir(target),
	); err != nil {
		return err
	}
	return nil
}

func (*Adapter) NormalizeConfig(
	ctx context.Context,
	raw json.RawMessage,
) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		spec.MaxConfigBytes,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: managed Source config: %w",
			spec.ErrInvalid,
			err,
		)
	}
	var value config
	if err := jsonutil.DecodeCanonicalObjectBytesInto(
		canonical,
		&value,
		spec.MaxConfigBytes,
	); err != nil {
		return nil, fmt.Errorf(
			"%w: managed Source config must be an empty object: %w",
			spec.ErrInvalid,
			err,
		)
	}
	return json.RawMessage(jsonutil.EmptyObject), nil
}

func (a *Adapter) Open(
	ctx context.Context,
	value sourceModel.Source,
) (driver.Snapshot, error) {
	if err := a.validateSource(ctx, value); err != nil {
		return nil, err
	}
	root, err := a.sourceRootPath(value, false)
	if err != nil {
		return nil, err
	}
	filesystemValue, err := a.filesystemSource(value, root)
	if err != nil {
		return nil, err
	}
	return a.filesystem.Open(ctx, filesystemValue)
}

func (a *Adapter) validateSource(ctx context.Context, value sourceModel.Source) error {
	if a == nil || a.filesystem == nil {
		return spec.ErrClosed
	}
	if err := value.Validate(); err != nil {
		return err
	}
	if value.Kind != Kind {
		return fmt.Errorf(
			"%w: managed adapter received source kind %q",
			spec.ErrInvalid,
			value.Kind,
		)
	}
	if _, err := jsonutil.DecodeJSONRaw[config](value.Config); err != nil {
		return fmt.Errorf(
			"%w: invalid managed Source config: %w",
			spec.ErrInvalid,
			err,
		)
	}
	return nil
}

func (a *Adapter) sourceRootPath(
	value sourceModel.Source,
	create bool,
) (string, error) {
	// Every caller has already passed value through validateSource.
	root, err := a.managedRootPath(value.RootStorageKey)
	if err != nil {
		return "", err
	}
	root = filepath.Join(root, string(value.StorageKey))
	if !create {
		return root, nil
	}
	if err := os.MkdirAll(root, managedDirectoryMode); err != nil {
		return "", err
	}
	return root, nil
}

func (a *Adapter) sourceStagingPath(
	value sourceModel.Source,
	create bool,
) (string, error) {
	// Every caller has already passed value through validateSource.
	root, err := a.managedStagingRootPath(value.RootStorageKey)
	if err != nil {
		return "", err
	}
	root = filepath.Join(root, string(value.StorageKey))
	if !create {
		return root, nil
	}
	if err := os.MkdirAll(root, managedDirectoryMode); err != nil {
		return "", err
	}
	return root, nil
}

func (a *Adapter) managedRootPath(
	rootStorageKey spec.StorageKey,
) (string, error) {
	if err := rootStorageKey.Validate(); err != nil {
		return "", err
	}

	root := filepath.Join(a.base, string(rootStorageKey))
	relative, err := filepath.Rel(a.base, root)
	if err != nil {
		return "", err
	}
	if relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) ||
		filepath.IsAbs(relative) {
		return "", fmt.Errorf(
			"%w: managed Root path escapes managed Source base",
			spec.ErrInvalid,
		)
	}
	return root, nil
}

func (a *Adapter) managedStagingRootPath(
	rootStorageKey spec.StorageKey,
) (string, error) {
	if err := rootStorageKey.Validate(); err != nil {
		return "", err
	}

	root := filepath.Join(a.stagingBase, string(rootStorageKey))
	relative, err := filepath.Rel(a.stagingBase, root)
	if err != nil {
		return "", err
	}
	if relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) ||
		filepath.IsAbs(relative) {
		return "", fmt.Errorf(
			"%w: managed Source staging path escapes staging base",
			spec.ErrInvalid,
		)
	}
	return root, nil
}

func (a *Adapter) filesystemSource(
	value sourceModel.Source,
	root string,
) (sourceModel.Source, error) {
	raw, err := json.Marshal(fsdir.Config{
		RootPath: root,
	})
	if err != nil {
		return sourceModel.Source{}, err
	}
	output := value.Clone()
	output.Kind = fsdir.Kind
	output.Config = raw
	return output, nil
}

func (a *Adapter) confirmedGeneration(
	ctx context.Context,
	value sourceModel.Source,
) (string, error) {
	snapshot, err := a.Open(ctx, value)
	if err != nil {
		return "", err
	}
	generation := snapshot.Generation()
	confirmErr := snapshot.Confirm(ctx)
	closeErr := snapshot.Close()
	if confirmErr != nil || closeErr != nil {
		return "", errors.Join(confirmErr, closeErr)
	}
	return generation, nil
}

func managedPackagePath(
	root string,
	directory spec.Locator,
	createParent bool,
) (string, error) {
	parent := path.Dir(string(directory))
	parentPath := root
	if parent != "." {
		parentPath = filepath.Join(root, filepath.FromSlash(parent))
		if createParent {
			if err := os.MkdirAll(parentPath, managedDirectoryMode); err != nil {
				return "", err
			}
		}
	}
	return filepath.Join(
		parentPath,
		filepath.FromSlash(path.Base(string(directory))),
	), nil
}

func pruneEmptyManagedParents(root, start string) error {
	root = filepath.Clean(root)
	current := filepath.Clean(start)
	for current != root {
		entries, err := os.ReadDir(current)
		if errors.Is(err, os.ErrNotExist) {
			current = filepath.Dir(current)
			continue
		}
		if err != nil {
			return err
		}
		if len(entries) != 0 {
			return nil
		}
		parent := filepath.Dir(current)
		if err := os.Remove(current); err != nil {
			return err
		}
		current = parent
	}
	return nil
}

func equivalentPackage(
	root string,
	expected []managedpackageModel.ManagedPackageFile,
) (exists, equivalent bool, err error) {
	info, err := os.Stat(root)
	if errors.Is(err, os.ErrNotExist) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	if !info.IsDir() {
		return true, false, nil
	}

	var (
		entries int
		total   int64
	)

	remaining := make(map[string][]byte, len(expected))
	for _, file := range expected {
		remaining[string(file.Locator)] = file.Content
	}
	err = filepath.WalkDir(root, func(
		location string,
		_ os.DirEntry,
		walkErr error,
	) error {
		if walkErr != nil {
			return walkErr
		}
		if location == root {
			return nil
		}
		entries++
		if entries > spec.MaxDiscoveryEntries {
			return fmt.Errorf(
				"%w: managed package exceeds entry limit",
				spec.ErrInvalid,
			)
		}
		relative, err := filepath.Rel(root, location)
		if err != nil {
			return err
		}
		if strings.Count(filepath.ToSlash(relative), "/")+1 >
			spec.MaxDiscoveryDepth {
			return fmt.Errorf(
				"%w: managed package exceeds depth limit",
				spec.ErrInvalid,
			)
		}
		info, err := os.Stat(location)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf(
				"%w: managed package contains a non-regular file",
				spec.ErrInvalid,
			)
		}
		if info.Size() < 0 || info.Size() > spec.MaxScanBytes-total {
			return fmt.Errorf(
				"%w: managed package exceeds byte limit",
				spec.ErrInvalid,
			)
		}
		total += info.Size()
		relative = filepath.ToSlash(relative)
		if err := spec.Locator(relative).ValidatePortable(false); err != nil {
			return err
		}
		expectedContent, found := remaining[relative]
		if !found || info.Size() != int64(len(expectedContent)) {
			return errPackageDifferent
		}

		content, err := readManagedPackageFile(
			location,
			int64(len(expectedContent)),
		)
		if err != nil {
			return err
		}
		if !bytes.Equal(content, expectedContent) {
			return errPackageDifferent
		}
		delete(remaining, relative)
		return nil
	})
	if errors.Is(err, errPackageDifferent) {
		return true, false, nil
	}
	if err != nil {
		return true, false, err
	}
	return true, len(remaining) == 0, nil
}

func readManagedPackageFile(
	location string,
	maximum int64,
) ([]byte, error) {
	file, err := os.Open(location)
	if err != nil {
		return nil, err
	}
	content, readErr := io.ReadAll(io.LimitReader(file, maximum+1))
	closeErr := file.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if int64(len(content)) > maximum {
		return nil, errPackageDifferent
	}
	return content, nil
}

type packageFileKeyAttributes struct {
	Locator spec.Locator
}

type packagePartitionProvider struct{}

func (*packagePartitionProvider) GetPartitionDir(
	key mapstore.FileKey,
) (string, error) {
	attributes, ok := key.XAttr.(packageFileKeyAttributes)
	if !ok {
		return "", fmt.Errorf(
			"%w: invalid managed package MapStore key",
			spec.ErrInvalid,
		)
	}
	if err := attributes.Locator.ValidatePortable(false); err != nil {
		return "", err
	}
	if key.FileName != path.Base(string(attributes.Locator)) {
		return "", fmt.Errorf(
			"%w: managed package filename does not match its locator",
			spec.ErrInvalid,
		)
	}
	parent := path.Dir(string(attributes.Locator))
	if parent == "." {
		return "", nil
	}
	return parent, nil
}

func (*packagePartitionProvider) ListPartitions(
	_ string,
	_ string,
	_ string,
	_ int,
) (dirs []string, nextPageToken string, err error) {
	return nil, "", spec.ErrUnsupported
}

func writeManagedPackageFiles(
	ctx context.Context,
	root string,
	files []managedpackageModel.ManagedPackageFile,
) error {
	directoryStore, err := mapstore.NewMapDirectoryStore(
		root,
		true,
		&packagePartitionProvider{},
		mapstoreio.RawEncoderDecoder{
			MaximumBytes: spec.MaxScanBytes,
		},
	)
	if err != nil {
		return err
	}
	storesClosed := false
	defer func() {
		if !storesClosed {
			_ = directoryStore.CloseAll()
		}
	}()

	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		key, err := managedPackageFileKey(file.Locator)
		if err != nil {
			return err
		}
		fileStore, err := directoryStore.OpenFile(
			key,
			true,
			mapstoreio.RawData(file.Content),
		)
		if err != nil {
			return fmt.Errorf(
				"write managed package file %q through MapStore: %w",
				file.Locator,
				err,
			)
		}
		stored, err := fileStore.GetAll(false)
		if err != nil {
			return err
		}
		content, err := mapstoreio.RawBytes(
			stored,
			spec.MaxScanBytes,
		)
		if err != nil {
			return err
		}
		if !bytes.Equal(content, file.Content) {
			return fmt.Errorf(
				"%w: MapStore changed managed package file %q",
				spec.ErrDigestMismatch,
				file.Locator,
			)
		}
	}
	if err := directoryStore.CloseAll(); err != nil {
		return err
	}
	storesClosed = true
	return nil
}

func managedPackageFileKey(
	locator spec.Locator,
) (mapstore.FileKey, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return mapstore.FileKey{}, err
	}
	return mapstore.FileKey{
		FileName: path.Base(string(locator)),
		XAttr: packageFileKeyAttributes{
			Locator: locator,
		},
	}, nil
}

func managedDirectoryEmpty(location string) (bool, error) {
	directory, err := os.Open(location)
	if err != nil {
		return false, err
	}
	entries, readErr := directory.ReadDir(1)
	closeErr := directory.Close()
	if readErr == io.EOF {
		return true, closeErr
	}
	if readErr != nil {
		return false, errors.Join(readErr, closeErr)
	}
	if closeErr != nil {
		return false, closeErr
	}
	return len(entries) == 0, nil
}
