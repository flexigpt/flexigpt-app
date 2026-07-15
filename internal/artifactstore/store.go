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

type storeOperationContextKey struct{}

// Store owns Artifact Store business logic. It depends only on repository and
// driver facades; SQLite, MapStore, LLMTools, PostgreSQL, and other concrete
// data-access mechanisms remain outside service methods.
type Store struct {
	repository         spec.ArtifactMetadataRepository
	portableContent    spec.PortableContentRepository
	sourceMaterializer spec.SourceMaterializer

	registryMu              sync.RWMutex
	drivers                 map[spec.SourceKind]spec.SourceDriver
	definitionMaterializers map[spec.SourceKind]spec.DefinitionMaterializer
	versionMatchers         map[spec.ArtifactKind]spec.ArtifactVersionMatcher
	frontends               map[spec.FrontendID]spec.ArtifactFrontend
	frontendOrder           []spec.FrontendID
	rootHooks               map[spec.RootKind]spec.RootKindHook
	collectionHooks         map[spec.CollectionKind]spec.CollectionKindHook

	scanMu           sync.Mutex
	lifeMu           sync.Mutex
	lifeCond         *sync.Cond
	closing          bool
	closed           bool
	activeOperations int
	closeErr         error
	now              func() time.Time
}

type StoreOption func(*Store) error

func WithSourceDriver(driver spec.SourceDriver) StoreOption {
	return func(store *Store) error { return store.RegisterSourceDriver(driver) }
}

func WithSourceMaterializer(materializer spec.SourceMaterializer) StoreOption {
	return func(store *Store) error {
		if materializer == nil {
			return fmt.Errorf("%w: source materializer is nil", spec.ErrInvalidRequest)
		}
		if store.sourceMaterializer != nil {
			return fmt.Errorf("%w: source materializer is already configured", spec.ErrConflict)
		}
		store.sourceMaterializer = materializer
		return nil
	}
}

func WithDefinitionMaterializer(materializer spec.DefinitionMaterializer) StoreOption {
	return func(store *Store) error {
		return store.RegisterDefinitionMaterializer(materializer)
	}
}

func WithArtifactVersionMatcher(matcher spec.ArtifactVersionMatcher) StoreOption {
	return func(store *Store) error { return store.RegisterArtifactVersionMatcher(matcher) }
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
		if store.portableContent != nil {
			return fmt.Errorf("%w: portable content repository is already configured", spec.ErrConflict)
		}
		store.portableContent = repository
		return nil
	}
}

// NewStore composes the default SQLite app-metadata repository, MapStore
// portable-content repository, and required source drivers. Constructor
// options are applied before defaults so callers can replace any external
// adapter without changing business logic.
func NewStore(baseDir string, options ...StoreOption) (*Store, error) {
	if strings.TrimSpace(baseDir) == "" {
		return nil, fmt.Errorf("%w: base directory is empty", spec.ErrInvalidRequest)
	}
	cleanBaseDir := filepath.Clean(baseDir)

	// Open MapStore first. Besides preserving the files-only portable-content
	// boundary, this lets the approved storage adapter prepare its hierarchy
	// before SQLite opens a database beneath the same application directory.
	defaultContent, err := contentstore.NewMapStorePortableContentRepository(
		filepath.Join(cleanBaseDir, "artifact-content"),
	)
	if err != nil {
		return nil, err
	}
	metadata, err := metadatastore.OpenMetadataStore(
		context.Background(),
		filepath.Join(cleanBaseDir, "artifactstore.sqlite"),
	)
	if err != nil {
		_ = defaultContent.Close()
		return nil, err
	}
	store, err := newStore(metadata, nil)
	if err != nil {
		_ = defaultContent.Close()
		_ = metadata.Close()
		return nil, err
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(store); err != nil {
			_ = defaultContent.Close()
			_ = store.Close()
			return nil, err
		}
	}
	if store.portableContent == nil {
		store.portableContent = defaultContent
		defaultContent = nil
	}
	if defaultContent != nil {
		if err := defaultContent.Close(); err != nil {
			_ = store.Close()
			return nil, fmt.Errorf("close replaced default portable content repository: %w", err)
		}
	}
	if err := store.installRequiredSourceDrivers(); err != nil {
		_ = store.Close()
		return nil, err
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
			_ = store.Close()
			return nil, err
		}
	}
	if err := store.installRequiredSourceDrivers(); err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

func newStore(repository spec.ArtifactMetadataRepository, content spec.PortableContentRepository) (*Store, error) {
	if repository == nil {
		return nil, fmt.Errorf("%w: metadata repository is nil", spec.ErrInvalidRequest)
	}
	store := &Store{
		repository:              repository,
		portableContent:         content,
		drivers:                 make(map[spec.SourceKind]spec.SourceDriver),
		definitionMaterializers: make(map[spec.SourceKind]spec.DefinitionMaterializer),
		versionMatchers:         make(map[spec.ArtifactKind]spec.ArtifactVersionMatcher),
		frontends:               make(map[spec.FrontendID]spec.ArtifactFrontend),
		rootHooks:               make(map[spec.RootKind]spec.RootKindHook),
		collectionHooks:         make(map[spec.CollectionKind]spec.CollectionKindHook),
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
	store.lifeCond = sync.NewCond(&store.lifeMu)
	if err := store.RegisterArtifactFrontend(portableDefinitionFrontend{}); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	s.lifeMu.Lock()
	for s.closing && !s.closed {
		s.lifeCond.Wait()
	}
	if s.closed {
		err := s.closeErr
		s.lifeMu.Unlock()
		return err
	}
	s.closing = true
	for s.activeOperations != 0 {
		s.lifeCond.Wait()
	}
	repository := s.repository
	content := s.portableContent
	s.lifeMu.Unlock()

	var closeErrors []error
	if content != nil {
		if err := content.Close(); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}
	if repository != nil {
		if err := repository.Close(); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}
	err := errors.Join(closeErrors...)

	s.lifeMu.Lock()
	s.closeErr = err
	s.closed = true
	s.closing = false
	s.lifeCond.Broadcast()
	s.lifeMu.Unlock()
	return err
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

func (s *Store) RegisterDefinitionMaterializer(materializer spec.DefinitionMaterializer) error {
	if s == nil || materializer == nil {
		return fmt.Errorf("%w: definition materializer is nil", spec.ErrInvalidRequest)
	}
	kind := materializer.Kind()
	if strings.TrimSpace(string(kind)) == "" {
		return fmt.Errorf("%w: definition materializer kind is empty", spec.ErrInvalidRequest)
	}
	s.registryMu.Lock()
	defer s.registryMu.Unlock()
	if _, exists := s.definitionMaterializers[kind]; exists {
		return fmt.Errorf("%w: definition materializer %q", spec.ErrConflict, kind)
	}
	s.definitionMaterializers[kind] = materializer
	return nil
}

func (s *Store) RegisterArtifactVersionMatcher(matcher spec.ArtifactVersionMatcher) error {
	if s == nil || matcher == nil {
		return fmt.Errorf("%w: artifact version matcher is nil", spec.ErrInvalidRequest)
	}
	kind := matcher.Kind()
	if strings.TrimSpace(string(kind)) == "" {
		return fmt.Errorf("%w: artifact version matcher kind is empty", spec.ErrInvalidRequest)
	}
	s.registryMu.Lock()
	defer s.registryMu.Unlock()
	if _, exists := s.versionMatchers[kind]; exists {
		return fmt.Errorf("%w: artifact version matcher %q", spec.ErrConflict, kind)
	}
	s.versionMatchers[kind] = matcher
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

func (s *Store) installRequiredSourceDrivers() error {
	if _, ok := s.driverFor(spec.SourceKindFSDirectory); !ok {
		if err := s.RegisterSourceDriver(sourcedriver.NewLLMToolsFSDirectoryDriver()); err != nil {
			return err
		}
	}
	if _, ok := s.driverFor(spec.SourceKindEmbeddedFSDirectory); !ok {
		if err := s.RegisterSourceDriver(sourcedriver.NewEmbeddedFSDirectoryDriver()); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) beginOperation(ctx context.Context) (context.Context, func(), error) {
	if s == nil || ctx == nil {
		return ctx, nil, spec.ErrClosed
	}
	if err := ctx.Err(); err != nil {
		return ctx, nil, err
	}
	if existing, ok := ctx.Value(storeOperationContextKey{}).(*storeOperationLease); ok &&
		existing.store == s && existing.retain() {
		var once sync.Once
		return ctx, func() {
			once.Do(func() { s.releaseOperation(existing) })
		}, nil
	}

	s.lifeMu.Lock()
	if s.closed || s.closing || s.repository == nil {
		s.lifeMu.Unlock()
		return ctx, nil, spec.ErrClosed
	}
	s.activeOperations++
	lease := &storeOperationLease{store: s, refs: 1, active: true}
	s.lifeMu.Unlock()

	var once sync.Once
	ctx = context.WithValue(ctx, storeOperationContextKey{}, lease)
	return ctx, func() {
		once.Do(func() { s.releaseOperation(lease) })
	}, nil
}

func (s *Store) releaseOperation(lease *storeOperationLease) {
	if lease == nil || !lease.release() {
		return
	}
	s.lifeMu.Lock()
	s.activeOperations--
	if s.activeOperations == 0 {
		s.lifeCond.Broadcast()
	}
	s.lifeMu.Unlock()
}

func (s *Store) newID() (string, error) {
	value, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generate UUIDv7: %w", err)
	}
	return value.String(), nil
}

func (s *Store) nextModifiedAt(previous time.Time) time.Time {
	now := s.nowUTC()
	if !now.After(previous) {
		return previous.Add(time.Nanosecond)
	}
	return now
}

func (s *Store) nowUTC() time.Time { return s.now().UTC() }

func requireExpectedModifiedAt(label string, current, expected time.Time) error {
	if expected.IsZero() {
		return fmt.Errorf("%w: %s expected modifiedAt is required", spec.ErrInvalidRequest, label)
	}
	if !current.Equal(expected) {
		return fmt.Errorf("%w: %s changed since it was read", spec.ErrConflict, label)
	}
	return nil
}

func (s *Store) definitionMaterializerFor(
	kind spec.SourceKind,
) (spec.DefinitionMaterializer, bool) {
	s.registryMu.RLock()
	defer s.registryMu.RUnlock()
	value, ok := s.definitionMaterializers[kind]
	return value, ok
}

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

func (s *Store) versionMatcherFor(kind spec.ArtifactKind) (spec.ArtifactVersionMatcher, bool) {
	s.registryMu.RLock()
	defer s.registryMu.RUnlock()
	matcher, ok := s.versionMatchers[kind]
	return matcher, ok
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
