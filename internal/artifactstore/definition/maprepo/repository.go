package maprepo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/flexigpt/mapstore-go"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/mapstoreio"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const maximumDefinitionFileBytes = int64(
	basespec.MaxDefinitionBytes + 64<<10,
)

type Repository struct {
	root  string
	files *mapstore.MapDirectoryStore

	mu     sync.RWMutex
	closed bool
}

type definitionKeyAttributes struct {
	RootID basespec.RootID
	Digest cryptoutil.Digest
}

type definitionPartitionProvider struct{}

func Open(root string) (*Repository, error) {
	privateRoot, err := mapstoreio.PreparePrivateDirectory(root)
	if err != nil {
		return nil, fmt.Errorf(
			"prepare definition repository: %w",
			err,
		)
	}
	files, err := mapstore.NewMapDirectoryStore(
		privateRoot,
		true,
		&definitionPartitionProvider{},
		mapstoreio.BoundedJSONEncoderDecoder{
			MaximumBytes: maximumDefinitionFileBytes,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"open definition MapDirectoryStore: %w",
			err,
		)
	}
	return &Repository{
		root:  privateRoot,
		files: files,
	}, nil
}

func (r *Repository) Close() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	r.closed = true
	if r.files == nil {
		return nil
	}
	return r.files.CloseAll()
}

func (r *Repository) Put(
	ctx context.Context,
	rootID basespec.RootID,
	value definition.Definition,
) (definition.Definition, error) {
	if ctx == nil {
		return definition.Definition{}, fmt.Errorf(
			"%w: definition write context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return definition.Definition{}, err
	}
	if err := basespec.ValidateRootID(rootID); err != nil {
		return definition.Definition{}, err
	}
	canonical, err := definition.Canonicalize(value)
	if err != nil {
		return definition.Definition{}, err
	}
	data, err := encodeFile(canonical)
	if err != nil {
		return definition.Definition{}, err
	}
	key, partition, path, err := r.location(rootID, canonical.Digest, true)
	if err != nil {
		return definition.Definition{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed || r.files == nil {
		return definition.Definition{}, basespec.ErrClosed
	}

	_, statErr := os.Stat(path)
	existed := statErr == nil
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return definition.Definition{}, statErr
	}
	if existed {
		if err := mapstoreio.SecureRegularFile(path); err != nil {
			return definition.Definition{}, err
		}
	}

	fileStore, err := r.files.OpenFile(key, true, data)
	if err != nil {
		return definition.Definition{}, fmt.Errorf(
			"open definition MapStore file: %w",
			err,
		)
	}
	if err := mapstoreio.SecureRegularFile(path); err != nil {
		return definition.Definition{}, err
	}
	storedMap, err := fileStore.GetAll(true)
	if err != nil {
		return definition.Definition{}, err
	}
	stored, err := decodeFile(storedMap)
	if err != nil {
		return definition.Definition{}, fmt.Errorf(
			"decode stored definition %q: %w",
			canonical.Digest,
			err,
		)
	}
	if stored.Digest != canonical.Digest {
		return definition.Definition{}, fmt.Errorf(
			"%w: requested definition %q, stored %q",
			basespec.ErrDigestMismatch,
			canonical.Digest,
			stored.Digest,
		)
	}
	if !existed {
		if err := mapstoreio.SyncRegularFile(path); err != nil {
			return definition.Definition{}, err
		}
		if err := mapstoreio.SyncDirectory(
			filepath.Join(r.root, partition),
		); err != nil {
			return definition.Definition{}, err
		}
	}
	return stored, nil
}

func (r *Repository) Get(
	ctx context.Context,
	rootID basespec.RootID,
	digest cryptoutil.Digest,
) (definition.Definition, error) {
	if ctx == nil {
		return definition.Definition{}, fmt.Errorf(
			"%w: definition read context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return definition.Definition{}, err
	}
	key, _, path, err := r.location(rootID, digest, false)
	if err != nil {
		return definition.Definition{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed || r.files == nil {
		return definition.Definition{}, basespec.ErrClosed
	}

	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return definition.Definition{}, fmt.Errorf(
			"%w: definition %q",
			basespec.ErrDefinitionNotFound,
			digest,
		)
	}
	if err != nil {
		return definition.Definition{}, err
	}
	if !info.Mode().IsRegular() {
		return definition.Definition{}, fmt.Errorf(
			"%w: definition path is not a regular file",
			basespec.ErrInvalid,
		)
	}
	if err := mapstoreio.SecureRegularFile(path); err != nil {
		return definition.Definition{}, err
	}

	fileStore, err := r.files.OpenFile(key, false, map[string]any{})
	if err != nil {
		if _, statErr := os.Lstat(path); errors.Is(statErr, os.ErrNotExist) {
			return definition.Definition{}, fmt.Errorf(
				"%w: definition %q",
				basespec.ErrDefinitionNotFound,
				digest,
			)
		}
		return definition.Definition{}, err
	}
	raw, err := fileStore.GetAll(true)
	if err != nil {
		return definition.Definition{}, err
	}
	value, err := decodeFile(raw)
	if err != nil {
		return definition.Definition{}, err
	}
	if value.Digest != digest {
		return definition.Definition{}, fmt.Errorf(
			"%w: requested %q, read %q",
			basespec.ErrDigestMismatch,
			digest,
			value.Digest,
		)
	}
	return value, nil
}

func (r *Repository) location(
	rootID basespec.RootID,
	digest cryptoutil.Digest,
	createParent bool,
) (key mapstore.FileKey, partition, path string, err error) {
	key, err = definitionFileKey(rootID, digest)
	if err != nil {
		return mapstore.FileKey{}, "", "", err
	}
	provider := &definitionPartitionProvider{}
	partition, err = provider.GetPartitionDir(key)
	if err != nil {
		return mapstore.FileKey{}, "", "", err
	}
	path, err = mapstoreio.PrivateFilePath(
		r.root,
		partition,
		key.FileName,
		createParent,
	)
	if err != nil {
		return mapstore.FileKey{}, "", "", err
	}
	return key, partition, path, nil
}

func definitionFileKey(
	rootID basespec.RootID,
	digest cryptoutil.Digest,
) (mapstore.FileKey, error) {
	if err := basespec.ValidateRootID(rootID); err != nil {
		return mapstore.FileKey{}, err
	}
	if err := cryptoutil.ValidateDigest(digest); err != nil {
		return mapstore.FileKey{}, err
	}
	hexDigest := strings.TrimPrefix(
		string(digest),
		cryptoutil.DigestSHA256Prefix,
	)
	return mapstore.FileKey{
		FileName: hexDigest + ".json",
		XAttr: definitionKeyAttributes{
			RootID: rootID,
			Digest: digest,
		},
	}, nil
}

func (*definitionPartitionProvider) GetPartitionDir(
	key mapstore.FileKey,
) (string, error) {
	attributes, ok := key.XAttr.(definitionKeyAttributes)
	if !ok {
		return "", fmt.Errorf(
			"%w: invalid definition MapStore key attributes",
			basespec.ErrInvalid,
		)
	}
	if err := basespec.ValidateRootID(attributes.RootID); err != nil {
		return "", err
	}
	if err := cryptoutil.ValidateDigest(attributes.Digest); err != nil {
		return "", err
	}
	hexDigest := strings.TrimPrefix(
		string(attributes.Digest),
		cryptoutil.DigestSHA256Prefix,
	)
	if key.FileName != hexDigest+".json" {
		return "", fmt.Errorf(
			"%w: definition MapStore filename does not match its digest",
			basespec.ErrInvalid,
		)
	}
	return filepath.Join(
		string(attributes.RootID),
		"sha256",
		hexDigest[:2],
	), nil
}

func (*definitionPartitionProvider) ListPartitions(
	_ string,
	_ string,
	_ string,
	_ int,
) (dirs []string, nextPageToken string, err error) {
	return nil, "", basespec.ErrUnsupported
}
