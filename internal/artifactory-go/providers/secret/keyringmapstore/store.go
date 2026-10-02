// Package keyringmapstore provides Artifact Store's default secret value
// backend through MapStore and keyringencdec.
//
// The MapStore file contains only encrypted secret values indexed by opaque
// Artifact Store secret refs. Secret metadata, SHA-256 values, Artifact refs,
// namespaces, slots, revisions, and cleanup state remain in Artifact Store
// SQLite metadata.
package keyringmapstore

import (
	"context"
	"fmt"
	"strings"

	"github.com/flexigpt/mapstore-go"
	"github.com/flexigpt/mapstore-go/jsonencdec"
	"github.com/flexigpt/mapstore-go/keyringencdec"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/api/secretapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/secret"
)

const (
	storeName = "keyring-mapstore"

	defaultKeyringService = "FlexiGPTArtifactSecretStore"
	defaultKeyringUser    = "artifactstore-v1"

	secretValuesKey = "values"
)

// Config controls the keyring encryption identity used by MapStore.
//
// The encryption key is held in the operating system keyring by
// keyringencdec. The MapStore file contains AES-GCM ciphertext only.
type Config struct {
	KeyringService string
	KeyringUser    string
}

// Store delegates file lifecycle, locking, flushing, and persistence behavior
// to MapStore. It performs no direct filesystem operations.
type Store struct {
	store      *mapstore.MapFileStore
	encEncrypt mapstore.IOEncoderDecoder
}

var _ secretapi.ValueStore = (*Store)(nil)

// New opens one MapStore secret file.
//
// The caller provides the resolved file path, just as SettingStore provides
// its settings file path to mapstore.NewMapFileStore. This package does not
// create directories, manipulate paths, or perform direct filesystem I/O.
func New(
	file string,
	config Config,
) (*Store, error) {
	if strings.TrimSpace(file) == "" {
		return nil, fmt.Errorf(
			"%w: Artifact Store secret MapStore file is required",
			basespec.ErrInvalid,
		)
	}

	if config.KeyringService == "" {
		config.KeyringService = defaultKeyringService
	}
	if config.KeyringUser == "" {
		config.KeyringUser = defaultKeyringUser
	}
	if err := basespec.ValidateRequiredText(
		"Artifact Store secret keyring service",
		config.KeyringService,
		basespec.MaxURIBytes,
	); err != nil {
		return nil, err
	}
	if err := basespec.ValidateRequiredText(
		"Artifact Store secret keyring user",
		config.KeyringUser,
		basespec.MaxURIBytes,
	); err != nil {
		return nil, err
	}

	encoder, err := keyringencdec.NewEncryptedStringValueEncoderDecoder(
		config.KeyringService,
		config.KeyringUser,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: initialize Artifact Store keyring encryption: %w",
			basespec.ErrSecretUnavailable,
			err,
		)
	}

	output := &Store{
		encEncrypt: encoder,
	}
	fileStore, err := mapstore.NewMapFileStore(
		file,
		map[string]any{
			secretValuesKey: map[string]any{},
		},
		jsonencdec.JSONEncoderDecoder{},
		mapstore.WithCreateIfNotExists(true),
		mapstore.WithFileAutoFlush(true),
		mapstore.WithValueEncDecGetter(
			output.valueEncDecGetter,
		),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: initialize Artifact Store secret MapStore: %w",
			basespec.ErrSecretUnavailable,
			err,
		)
	}
	output.store = fileStore
	return output, nil
}

func (*Store) Name() string {
	return storeName
}

// Put creates one secret value under an opaque Artifact Store ref.
//
// Artifact Store generates refs uniquely. The existing-key check is retained
// as corruption and accidental-replay protection.
func (s *Store) Put(
	ctx context.Context,
	ref secret.Ref,
	value string,
) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	if err := ref.Validate(); err != nil {
		return err
	}
	if err := secret.ValidateValue(value); err != nil {
		return err
	}

	id, err := ref.ID()
	if err != nil {
		return err
	}

	values, err := s.values(false)
	if err != nil {
		return err
	}
	if _, exists := values[id]; exists {
		return fmt.Errorf(
			"%w: Artifact Store secret ref %q already exists",
			basespec.ErrConflict,
			ref,
		)
	}

	if err := s.store.SetKey(
		[]string{secretValuesKey, id},
		value,
	); err != nil {
		return fmt.Errorf(
			"%w: persist Artifact Store secret: %w",
			basespec.ErrSecretUnavailable,
			err,
		)
	}
	return nil
}

func (s *Store) Get(
	ctx context.Context,
	ref secret.Ref,
) (string, error) {
	if err := s.ready(ctx); err != nil {
		return "", err
	}
	if err := ref.Validate(); err != nil {
		return "", err
	}

	id, err := ref.ID()
	if err != nil {
		return "", err
	}

	values, err := s.values(false)
	if err != nil {
		return "", err
	}
	raw, found := values[id]
	if !found {
		return "", fmt.Errorf(
			"%w: Artifact Store secret ref %q",
			basespec.ErrSecretNotFound,
			ref,
		)
	}

	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf(
			"%w: Artifact Store secret ref %q is not a string",
			basespec.ErrInvalid,
			ref,
		)
	}
	if err := secret.ValidateValue(value); err != nil {
		return "", err
	}
	return value, nil
}

// Delete is idempotent. It removes only the encrypted MapStore value for the
// opaque ref. All logical lifecycle bookkeeping has already happened in
// Artifact Store SQLite before this method is called.
func (s *Store) Delete(
	ctx context.Context,
	ref secret.Ref,
) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	if err := ref.Validate(); err != nil {
		return err
	}

	id, err := ref.ID()
	if err != nil {
		return err
	}

	values, err := s.values(false)
	if err != nil {
		return err
	}
	if _, found := values[id]; !found {
		return nil
	}

	if err := s.store.DeleteKey(
		[]string{secretValuesKey, id},
	); err != nil {
		return fmt.Errorf(
			"%w: delete Artifact Store secret: %w",
			basespec.ErrSecretUnavailable,
			err,
		)
	}
	return nil
}

func (s *Store) Close() error {
	if s == nil || s.store == nil {
		return nil
	}
	return s.store.Close()
}

func (s *Store) ready(
	ctx context.Context,
) error {
	if s == nil || s.store == nil || s.encEncrypt == nil {
		return basespec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: Artifact Store secret MapStore context is nil",
			basespec.ErrInvalid,
		)
	}
	return ctx.Err()
}

func (s *Store) values(
	forceFetch bool,
) (map[string]any, error) {
	raw, err := s.store.GetAll(forceFetch)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: read Artifact Store secret MapStore: %w",
			basespec.ErrSecretUnavailable,
			err,
		)
	}

	values, found := raw[secretValuesKey].(map[string]any)
	if !found {
		return nil, fmt.Errorf(
			"%w: Artifact Store secret MapStore has invalid value layout",
			basespec.ErrInvalid,
		)
	}
	return values, nil
}

// valueEncDecGetter encrypts only the secret value leaves. MapStore owns file
// synchronization, atomic persistence, and all normal map mutation behavior.
func (s *Store) valueEncDecGetter(
	path []string,
) mapstore.IOEncoderDecoder {
	if len(path) == 2 && path[0] == secretValuesKey {
		return s.encEncrypt
	}
	return nil
}
