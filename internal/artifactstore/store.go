package artifactstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/contentstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/metadatastore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/sourcedriver"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/google/uuid"
)

func isNotFound(err error) bool {
	return errors.Is(err, spec.ErrNotFound)
}

func isConflict(err error) bool {
	return errors.Is(err, spec.ErrConflict)
}

// Store owns Artifact Store business logic. It depends only on repository and
// driver facades; SQLite, MapStore, LLMTools, PostgreSQL, and other concrete
// data-access mechanisms remain outside service methods.
type Store struct {
	baseDir         string
	repository      spec.ArtifactMetadataRepository
	portableContent spec.PortableContentRepository

	registryMu      sync.RWMutex
	drivers         map[spec.SourceKind]spec.SourceDriver
	frontends       map[spec.FrontendID]spec.ArtifactFrontend
	frontendOrder   []spec.FrontendID
	rootHooks       map[spec.RootKind]spec.RootKindHook
	collectionHooks map[spec.CollectionKind]spec.CollectionKindHook

	scanMu sync.Mutex
	lifeMu sync.RWMutex
	closed bool
	now    func() time.Time
}

type StoreOption func(*Store) error

func WithSourceDriver(driver spec.SourceDriver) StoreOption {
	return func(store *Store) error { return store.RegisterSourceDriver(driver) }
}

// WithEmbeddedFSProvider registers an application-owned read-only fs.FS under
// the provider key referenced by embedded-fs-directory source configuration.
func WithEmbeddedFSProvider(providerKey string, provider fs.FS) StoreOption {
	return func(store *Store) error {
		driver, ok := store.driverFor(spec.SourceKindEmbeddedFSDirectory)
		if !ok {
			embedded := sourcedriver.NewEmbeddedFSDirectoryDriver()
			if err := store.RegisterSourceDriver(embedded); err != nil {
				return err
			}
			driver = embedded
		}
		registrar, ok := driver.(sourcedriver.EmbeddedFSProviderRegistrar)
		if !ok {
			return fmt.Errorf(
				"%w: embedded source driver does not support provider registration",
				spec.ErrUnsupported,
			)
		}
		return registrar.RegisterProvider(providerKey, provider)
	}
}

func WithArtifactFrontend(frontend spec.ArtifactFrontend) StoreOption {
	return func(store *Store) error { return store.RegisterArtifactFrontend(frontend) }
}

func WithRootKindHook(hook spec.RootKindHook) StoreOption {
	return func(store *Store) error { return store.RegisterRootKindHook(hook) }
}

func WithCollectionKindHook(hook spec.CollectionKindHook) StoreOption {
	return func(store *Store) error { return store.RegisterCollectionKindHook(hook) }
}

func WithPortableContentRepository(repository spec.PortableContentRepository) StoreOption {
	return func(store *Store) error {
		if repository == nil {
			return fmt.Errorf("%w: portable content repository is nil", spec.ErrInvalidRequest)
		}
		store.portableContent = repository
		return nil
	}
}

// NewStore composes the default SQLite app-metadata repository, MapStore
// portable-content repository, and LLMTools fs-directory driver. No business
// method itself accesses the filesystem directly.
func NewStore(baseDir string, options ...StoreOption) (*Store, error) {
	if strings.TrimSpace(baseDir) == "" {
		return nil, fmt.Errorf("%w: base directory is empty", spec.ErrInvalidRequest)
	}
	cleanBaseDir := filepath.Clean(baseDir)
	metadata, err := metadatastore.OpenMetadataStore(
		context.Background(),
		filepath.Join(cleanBaseDir, "artifactstore.sqlite"),
	)
	if err != nil {
		return nil, err
	}
	content, err := contentstore.NewMapStorePortableContentRepository(filepath.Join(cleanBaseDir, "artifact-content"))
	if err != nil {
		_ = metadata.Close()
		return nil, err
	}
	store, err := newStore(metadata, content)
	if err != nil {
		_ = content.Close()
		_ = metadata.Close()
		return nil, err
	}
	store.baseDir = cleanBaseDir
	if err := store.RegisterSourceDriver(sourcedriver.NewLLMToolsFSDirectoryDriver()); err != nil {
		_ = store.Close()
		return nil, err
	}
	if err := store.RegisterSourceDriver(sourcedriver.NewEmbeddedFSDirectoryDriver()); err != nil {
		_ = store.Close()
		return nil, err
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(store); err != nil {
			_ = store.Close()
			return nil, err
		}
	}
	return store, nil
}

// NewStoreWithMetadataRepository constructs the business layer with injected
// facades. A PostgreSQL metadata repository and a custom driver registry can
// use this constructor without changing service logic.
func NewStoreWithMetadataRepository(
	repository spec.ArtifactMetadataRepository,
	options ...StoreOption,
) (*Store, error) {
	store, err := newStore(repository, nil)
	if err != nil {
		return nil, err
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(store); err != nil {
			return nil, err
		}
	}
	return store, nil
}

func newStore(repository spec.ArtifactMetadataRepository, content spec.PortableContentRepository) (*Store, error) {
	if repository == nil {
		return nil, fmt.Errorf("%w: metadata repository is nil", spec.ErrInvalidRequest)
	}
	return &Store{
		repository:      repository,
		portableContent: content,
		drivers:         make(map[spec.SourceKind]spec.SourceDriver),
		frontends:       make(map[spec.FrontendID]spec.ArtifactFrontend),
		rootHooks:       make(map[spec.RootKind]spec.RootKindHook),
		collectionHooks: make(map[spec.CollectionKind]spec.CollectionKindHook),
		now: func() time.Time {
			return time.Now().UTC()
		},
	}, nil
}

func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	s.lifeMu.Lock()
	if s.closed {
		s.lifeMu.Unlock()
		return nil
	}
	s.closed = true
	repository := s.repository
	content := s.portableContent
	s.lifeMu.Unlock()

	var closeErrors []error
	if content != nil {
		closeErrors = append(closeErrors, content.Close())
	}
	if repository != nil {
		closeErrors = append(closeErrors, repository.Close())
	}
	return errors.Join(closeErrors...)
}

func (s *Store) RegisterSourceDriver(driver spec.SourceDriver) error {
	if s == nil || driver == nil {
		return fmt.Errorf("%w: source driver is nil", spec.ErrInvalidRequest)
	}
	kind := driver.Kind()
	if strings.TrimSpace(string(kind)) == "" {
		return fmt.Errorf("%w: source driver kind is empty", spec.ErrInvalidRequest)
	}
	s.registryMu.Lock()
	defer s.registryMu.Unlock()
	if _, exists := s.drivers[kind]; exists {
		return fmt.Errorf("%w: source driver %q", spec.ErrConflict, kind)
	}
	s.drivers[kind] = driver
	return nil
}

func (s *Store) RegisterArtifactFrontend(frontend spec.ArtifactFrontend) error {
	if s == nil || frontend == nil {
		return fmt.Errorf("%w: artifact frontend is nil", spec.ErrInvalidRequest)
	}
	id := frontend.ID()
	if strings.TrimSpace(string(id)) == "" {
		return fmt.Errorf("%w: artifact frontend ID is empty", spec.ErrInvalidRequest)
	}
	s.registryMu.Lock()
	defer s.registryMu.Unlock()
	if _, exists := s.frontends[id]; exists {
		return fmt.Errorf("%w: artifact frontend %q", spec.ErrConflict, id)
	}
	s.frontends[id] = frontend
	s.frontendOrder = append(s.frontendOrder, id)
	return nil
}

func (s *Store) RegisterRootKindHook(hook spec.RootKindHook) error {
	if s == nil || hook == nil {
		return fmt.Errorf("%w: root kind hook is nil", spec.ErrInvalidRequest)
	}
	kind := hook.Kind()
	if strings.TrimSpace(string(kind)) == "" {
		return fmt.Errorf("%w: root kind hook kind is empty", spec.ErrInvalidRequest)
	}
	s.registryMu.Lock()
	defer s.registryMu.Unlock()
	if _, exists := s.rootHooks[kind]; exists {
		return fmt.Errorf("%w: root kind hook %q", spec.ErrConflict, kind)
	}
	s.rootHooks[kind] = hook
	return nil
}

func (s *Store) RegisterCollectionKindHook(hook spec.CollectionKindHook) error {
	if s == nil || hook == nil {
		return fmt.Errorf("%w: collection kind hook is nil", spec.ErrInvalidRequest)
	}
	kind := hook.Kind()
	if strings.TrimSpace(string(kind)) == "" {
		return fmt.Errorf("%w: collection kind hook kind is empty", spec.ErrInvalidRequest)
	}
	s.registryMu.Lock()
	defer s.registryMu.Unlock()
	if _, exists := s.collectionHooks[kind]; exists {
		return fmt.Errorf("%w: collection kind hook %q", spec.ErrConflict, kind)
	}
	s.collectionHooks[kind] = hook
	return nil
}

func (s *Store) ensureOpen() error {
	if s == nil {
		return spec.ErrClosed
	}
	s.lifeMu.RLock()
	defer s.lifeMu.RUnlock()
	if s.closed || s.repository == nil {
		return spec.ErrClosed
	}
	return nil
}

func (s *Store) newID() (string, error) {
	value, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generate UUIDv7: %w", err)
	}
	return value.String(), nil
}

func (s *Store) nowUTC() time.Time { return s.now().UTC() }

func (s *Store) driverFor(kind spec.SourceKind) (spec.SourceDriver, bool) {
	s.registryMu.RLock()
	defer s.registryMu.RUnlock()
	driver, ok := s.drivers[kind]
	return driver, ok
}

func (s *Store) frontendFor(id spec.FrontendID) (spec.ArtifactFrontend, bool) {
	s.registryMu.RLock()
	defer s.registryMu.RUnlock()
	frontend, ok := s.frontends[id]
	return frontend, ok
}

func (s *Store) frontendsSnapshot() []spec.ArtifactFrontend {
	s.registryMu.RLock()
	defer s.registryMu.RUnlock()
	out := make([]spec.ArtifactFrontend, 0, len(s.frontendOrder))
	for _, id := range s.frontendOrder {
		if frontend, ok := s.frontends[id]; ok {
			out = append(out, frontend)
		}
	}
	return out
}

func (s *Store) rootHookFor(kind spec.RootKind) (spec.RootKindHook, bool) {
	s.registryMu.RLock()
	defer s.registryMu.RUnlock()
	hook, ok := s.rootHooks[kind]
	return hook, ok
}

func (s *Store) collectionHookFor(kind spec.CollectionKind) (spec.CollectionKindHook, bool) {
	s.registryMu.RLock()
	defer s.registryMu.RUnlock()
	hook, ok := s.collectionHooks[kind]
	return hook, ok
}

func normalizedJSONObject(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage("{}")
	}
	return append(json.RawMessage(nil), raw...)
}
