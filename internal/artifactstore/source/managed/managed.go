package managed

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
	"sort"
	"strings"
	"sync"

	"github.com/flexigpt/mapstore-go"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/mapstoreio"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/fsdir"
)

const (
	Kind artifactstore.SourceKind = "managed-directory"

	directoryMode        = 0o700
	fileMode             = 0o600
	stagingDirectoryName = ".artifactstore-staging"
)

type config struct{}

// Adapter stores each managed Source beneath:
//
//	<base>/<root-id>/<source-id>
//
// The physical path remains private Source configuration derived by the
// adapter. It is never placed in Source.Config, portable definitions, or API
// projections.
type Adapter struct {
	base       string
	filesystem *fsdir.Adapter
	mu         sync.Mutex
}

func New(base string) (*Adapter, error) {
	if strings.TrimSpace(base) == "" {
		return nil, fmt.Errorf(
			"%w: managed Source base directory is empty",
			artifactstore.ErrInvalid,
		)
	}
	absolute, err := mapstoreio.PreparePrivateDirectory(base)
	if err != nil {
		return nil, fmt.Errorf(
			"prepare managed Source base directory: %w",
			err,
		)
	}

	policy := fsdir.DefaultTraversalPolicy()
	policy.ExcludedDirectoryNames = append(
		policy.ExcludedDirectoryNames,
		stagingDirectoryName,
	)
	filesystem, err := fsdir.NewWithTraversalPolicy(&policy)
	if err != nil {
		return nil, err
	}
	return &Adapter{
		base:       absolute,
		filesystem: filesystem,
	}, nil
}

func (*Adapter) Kind() artifactstore.SourceKind {
	return Kind
}

func (*Adapter) NormalizeConfig(
	ctx context.Context,
	raw json.RawMessage,
) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	canonical, err := jsoncanon.CanonicalizeObject(
		raw,
		artifactstore.MaxConfigBytes,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: managed Source config: %w",
			artifactstore.ErrInvalid,
			err,
		)
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	var value config
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf(
			"%w: managed Source config must be an empty object: %w",
			artifactstore.ErrInvalid,
			err,
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("managed Source config has trailing JSON")
		}
		return nil, fmt.Errorf("%w: %w", artifactstore.ErrInvalid, err)
	}
	return json.RawMessage(jsoncanon.EmptyObject), nil
}

func (a *Adapter) Open(
	ctx context.Context,
	value source.Source,
) (source.Snapshot, error) {
	if err := a.validateSource(ctx, value); err != nil {
		return nil, err
	}
	root, err := a.sourceRootPath(value, true)
	if err != nil {
		return nil, err
	}
	filesystemValue, err := a.filesystemSource(value, root)
	if err != nil {
		return nil, err
	}
	return a.filesystem.Open(ctx, filesystemValue)
}

func (a *Adapter) ResolveLocalPath(
	ctx context.Context,
	value source.Source,
	locator artifactstore.Locator,
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

func (a *Adapter) PublishPackage(
	ctx context.Context,
	value source.Source,
	publication source.ManagedPackagePublication,
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := a.validateSource(ctx, value); err != nil {
		return "", err
	}
	files, err := validatePublication(publication)
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
		publication.Directory,
		false,
	)
	if err != nil {
		return "", err
	}
	exists, equivalent, err := equivalentPackage(target, files)
	if err != nil {
		return "", err
	}
	if exists {
		if !equivalent {
			return "", fmt.Errorf(
				"%w: managed package %q already exists with different content",
				artifactstore.ErrConflict,
				publication.Directory,
			)
		}
		return a.confirmedGeneration(ctx, value)
	}

	if publication.ExpectedGeneration != "" {
		if err := artifactstore.ValidateSourceGeneration(
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
				artifactstore.ErrConflict,
			)
		}
	}
	target, err = managedPackagePath(
		root,
		publication.Directory,
		true,
	)
	if err != nil {
		return "", err
	}

	stagingRoot, err := mapstoreio.EnsurePrivateSubdirectory(
		root,
		stagingDirectoryName,
	)
	if err != nil {
		return "", err
	}
	temporary, err := os.MkdirTemp(stagingRoot, "package-*")
	if err != nil {
		return "", err
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(temporary)
		}
	}()

	packageRoot := filepath.Join(temporary, "content")
	if err := os.Mkdir(packageRoot, directoryMode); err != nil {
		return "", err
	}
	if err := writeManagedPackageFiles(
		ctx,
		packageRoot,
		files,
	); err != nil {
		return "", err
	}

	// The target package must not already exist. Renaming a fully staged
	// directory is the Source-side publication boundary.
	if err := os.Rename(packageRoot, target); err != nil {
		exists, equivalent, verifyErr := equivalentPackage(target, files)
		if verifyErr == nil && exists && equivalent {
			committed = true
			_ = os.RemoveAll(temporary)
			return a.confirmedGeneration(ctx, value)
		}
		return "", fmt.Errorf("publish managed package: %w", err)
	}
	committed = true
	_ = os.RemoveAll(temporary)
	if err := mapstoreio.SyncDirectory(filepath.Dir(target)); err != nil {
		return "", err
	}
	return a.confirmedGeneration(ctx, value)
}

func (a *Adapter) RemovePackage(
	ctx context.Context,
	value source.Source,
	directory artifactstore.Locator,
	expectedGeneration string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.validateSource(ctx, value); err != nil {
		return err
	}
	if err := validatePackageDirectory(directory); err != nil {
		return err
	}
	if err := artifactstore.ValidateSourceGeneration(expectedGeneration); err != nil {
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
			artifactstore.ErrConflict,
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
	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf(
			"%w: managed package %q",
			artifactstore.ErrNotFound,
			directory,
		)
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf(
			"%w: managed package is not a regular directory",
			artifactstore.ErrInvalid,
		)
	}

	stagingRoot, err := mapstoreio.EnsurePrivateSubdirectory(
		root,
		stagingDirectoryName,
	)
	if err != nil {
		return err
	}
	tombstone, err := os.MkdirTemp(stagingRoot, "remove-*")
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
		_ = os.Rename(tombstone, target)
		return err
	}
	return mapstoreio.SyncDirectory(filepath.Dir(target))
}

func (a *Adapter) validateSource(ctx context.Context, value source.Source) error {
	if a == nil || a.filesystem == nil {
		return artifactstore.ErrClosed
	}
	if err := value.Validate(); err != nil {
		return err
	}
	if value.Kind != Kind {
		return fmt.Errorf(
			"%w: managed adapter received source kind %q",
			artifactstore.ErrInvalid,
			value.Kind,
		)
	}
	normalized, err := a.NormalizeConfig(ctx, value.Config)
	if err != nil {
		return err
	}
	if !bytes.Equal(normalized, []byte(jsoncanon.EmptyObject)) {
		return fmt.Errorf(
			"%w: invalid normalized managed Source config",
			artifactstore.ErrInvalid,
		)
	}
	return nil
}

func (a *Adapter) sourceRootPath(
	value source.Source,
	create bool,
) (string, error) {
	relative := filepath.Join(
		string(value.RootID),
		string(value.ID),
	)
	if create {
		return mapstoreio.EnsurePrivateSubdirectory(a.base, relative)
	}
	return mapstoreio.PrivateSubdirectoryPath(a.base, relative)
}

func (a *Adapter) filesystemSource(
	value source.Source,
	root string,
) (source.Source, error) {
	raw, err := json.Marshal(fsdir.Config{
		RootPath: root,
	})
	if err != nil {
		return source.Source{}, err
	}
	output := value.Clone()
	output.Kind = fsdir.Kind
	output.Config = raw
	return output, nil
}

func (a *Adapter) confirmedGeneration(
	ctx context.Context,
	value source.Source,
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

func validatePublication(
	publication source.ManagedPackagePublication,
) ([]source.ManagedPackageFile, error) {
	normalized, err := source.NormalizeManagedPackagePublication(publication)
	if err != nil {
		return nil, err
	}

	if firstSegment(normalized.Directory) == stagingDirectoryName {
		return nil, fmt.Errorf(
			"%w: managed package uses a reserved directory",
			artifactstore.ErrInvalid,
		)
	}
	for index, file := range normalized.Files {
		if firstSegment(file.Locator) == stagingDirectoryName {
			return nil, fmt.Errorf(
				"%w: managed package files[%d] use a reserved directory",
				artifactstore.ErrInvalid,
				index,
			)
		}
	}
	return normalized.Files, nil
}

func validatePackageDirectory(directory artifactstore.Locator) error {
	if err := artifactstore.ValidatePortableLocator(
		directory,
		false,
	); err != nil {
		return err
	}
	return source.ValidateManagedPackageDirectory(directory)
}

func firstSegment(locator artifactstore.Locator) string {
	value := string(locator)
	if before, _, found := strings.Cut(value, "/"); found {
		return before
	}
	return value
}

func managedPackagePath(
	root string,
	directory artifactstore.Locator,
	createParent bool,
) (string, error) {
	if err := validatePackageDirectory(directory); err != nil {
		return "", err
	}
	parent := path.Dir(string(directory))
	parentPath := root
	var err error
	if parent != "." {
		relative := filepath.FromSlash(parent)
		if createParent {
			parentPath, err = mapstoreio.EnsurePrivateSubdirectory(
				root,
				relative,
			)
		} else {
			parentPath, err = mapstoreio.PrivateSubdirectoryPath(
				root,
				relative,
			)
		}
		if err != nil {
			return "", err
		}
	}
	target := filepath.Join(
		parentPath,
		filepath.FromSlash(path.Base(string(directory))),
	)
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	if relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) ||
		filepath.IsAbs(relative) {
		return "", fmt.Errorf("%w: managed package escapes source root", artifactstore.ErrInvalid)
	}
	return target, nil
}

func equivalentPackage(
	root string,
	expected []source.ManagedPackageFile,
) (exists, equivalent bool, err error) {
	info, err := os.Lstat(root)
	if errors.Is(err, os.ErrNotExist) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
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
		entry os.DirEntry,
		walkErr error,
	) error {
		if walkErr != nil {
			return walkErr
		}
		if location == root {
			return nil
		}
		entries++
		if entries > artifactstore.MaxDiscoveryEntries {
			return fmt.Errorf(
				"%w: managed package exceeds entry limit",
				artifactstore.ErrInvalid,
			)
		}
		relative, err := filepath.Rel(root, location)
		if err != nil {
			return err
		}
		if strings.Count(filepath.ToSlash(relative), "/")+1 >
			artifactstore.MaxDiscoveryDepth {
			return fmt.Errorf(
				"%w: managed package exceeds depth limit",
				artifactstore.ErrInvalid,
			)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf(
				"%w: managed package contains a symbolic link",
				artifactstore.ErrInvalid,
			)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf(
				"%w: managed package contains a non-regular file",
				artifactstore.ErrInvalid,
			)
		}
		if info.Size() < 0 || info.Size() > artifactstore.MaxScanBytes-total {
			return fmt.Errorf(
				"%w: managed package exceeds byte limit",
				artifactstore.ErrInvalid,
			)
		}
		total += info.Size()
		relative = path.Clean(filepath.ToSlash(relative))
		if err := artifactstore.ValidatePortableLocator(
			artifactstore.Locator(relative),
			false,
		); err != nil {
			return err
		}
		expectedContent, found := remaining[relative]
		if !found || info.Size() != int64(len(expectedContent)) {
			return errPackageDifferent
		}

		//nolint:gosec // No Root race path.
		content, err := os.ReadFile(location)
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

var errPackageDifferent = errors.New("managed package content differs")

type packageFileKeyAttributes struct {
	Locator artifactstore.Locator
}

type packagePartitionProvider struct{}

func (*packagePartitionProvider) GetPartitionDir(
	key mapstore.FileKey,
) (string, error) {
	attributes, ok := key.XAttr.(packageFileKeyAttributes)
	if !ok {
		return "", fmt.Errorf(
			"%w: invalid managed package MapStore key",
			artifactstore.ErrInvalid,
		)
	}
	if err := artifactstore.ValidatePortableLocator(
		attributes.Locator,
		false,
	); err != nil {
		return "", err
	}
	if key.FileName != path.Base(string(attributes.Locator)) {
		return "", fmt.Errorf(
			"%w: managed package filename does not match its locator",
			artifactstore.ErrInvalid,
		)
	}
	parent := path.Dir(string(attributes.Locator))
	if parent == "." {
		return "", nil
	}
	return filepath.FromSlash(parent), nil
}

func (*packagePartitionProvider) ListPartitions(
	_ string,
	_ string,
	_ string,
	_ int,
) (dirs []string, nextPageToken string, err error) {
	return nil, "", artifactstore.ErrUnsupported
}

func managedPackageFileKey(
	locator artifactstore.Locator,
) (mapstore.FileKey, error) {
	if err := artifactstore.ValidatePortableLocator(locator, false); err != nil {
		return mapstore.FileKey{}, err
	}
	return mapstore.FileKey{
		FileName: path.Base(string(locator)),
		XAttr: packageFileKeyAttributes{
			Locator: locator,
		},
	}, nil
}

func writeManagedPackageFiles(
	ctx context.Context,
	root string,
	files []source.ManagedPackageFile,
) error {
	directoryStore, err := mapstore.NewMapDirectoryStore(
		root,
		true,
		&packagePartitionProvider{},
		mapstoreio.RawEncoderDecoder{
			MaximumBytes: artifactstore.MaxScanBytes,
		},
	)
	if err != nil {
		return err
	}
	defer func() { _ = directoryStore.CloseAll() }()

	for _, file := range files {
		parent := path.Dir(string(file.Locator))
		if parent == "." {
			continue
		}
		if _, err := mapstoreio.EnsurePrivateSubdirectory(
			root,
			filepath.FromSlash(parent),
		); err != nil {
			return err
		}
	}

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
			artifactstore.MaxScanBytes,
		)
		if err != nil {
			return err
		}
		if !bytes.Equal(content, file.Content) {
			return fmt.Errorf(
				"%w: MapStore changed managed package file %q",
				artifactstore.ErrDigestMismatch,
				file.Locator,
			)
		}
	}
	if err := directoryStore.CloseAll(); err != nil {
		return err
	}
	return secureAndSyncManagedPackage(root)
}

func secureAndSyncManagedPackage(root string) error {
	directories := make([]string, 0)
	err := filepath.WalkDir(root, func(
		location string,
		entry os.DirEntry,
		walkErr error,
	) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := os.Lstat(location)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf(
				"%w: managed package contains a symbolic link",
				artifactstore.ErrInvalid,
			)
		}
		if info.IsDir() {
			//nolint:gosec // No Root APIs.
			if err := os.Chmod(location, directoryMode); err != nil {
				return err
			}
			directories = append(directories, location)
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf(
				"%w: managed package contains a non-regular file",
				artifactstore.ErrInvalid,
			)
		}
		//nolint:gosec // No Root APIs.
		if err := os.Chmod(location, fileMode); err != nil {
			return err
		}
		return mapstoreio.SyncRegularFile(location)
	})
	if err != nil {
		return err
	}
	sort.Slice(directories, func(left, right int) bool {
		return len(directories[left]) > len(directories[right])
	})
	for _, directory := range directories {
		if err := mapstoreio.SyncDirectory(directory); err != nil {
			return err
		}
	}
	return nil
}

var (
	_ source.Adapter              = (*Adapter)(nil)
	_ source.LocalPathResolver    = (*Adapter)(nil)
	_ source.ManagedPackageWriter = (*Adapter)(nil)
)
