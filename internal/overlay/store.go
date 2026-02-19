package overlay

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

const (
	sqlCreateGroupsTable = `
CREATE TABLE IF NOT EXISTS groups (
    group_id TEXT PRIMARY KEY
);`

	sqlCreateFlagsTable = `
CREATE TABLE IF NOT EXISTS flags (
    group_id    TEXT    NOT NULL REFERENCES groups(group_id) ON DELETE CASCADE,
    key_id      TEXT    NOT NULL,
    value       BLOB    NOT NULL,
    created_at  TIMESTAMP NOT NULL,
    modified_at TIMESTAMP NOT NULL,
    PRIMARY KEY (group_id, key_id)
);`

	sqlInsertGroup = `INSERT OR IGNORE INTO groups (group_id) VALUES (?);`

	sqlSelectFlag = `
SELECT value, created_at, modified_at
  FROM flags
 WHERE group_id = ?
   AND key_id   = ?;`

	sqlInsertFlag = `
INSERT INTO flags (group_id, key_id, value, created_at, modified_at)
VALUES (?, ?, ?, ?, ?);`

	sqlUpdateFlag = `
UPDATE flags
   SET value       = ?,
       modified_at = ?
 WHERE group_id    = ?
   AND key_id      = ?;`

	sqlDeleteFlag = `
DELETE FROM flags
 WHERE group_id = ?
   AND key_id   = ?;`
)

type Store struct {
	mu  sync.RWMutex
	db  *sql.DB
	reg map[GroupID]struct{}

	closeOnce sync.Once
	closeErr  error
}

type Option func(*Store) error

// WithKeyType registers the group of the compile-time key type K.
func WithKeyType[K Key]() Option {
	var zero K
	groupID := zero.Group()

	return func(s *Store) error {
		if _, dup := s.reg[groupID]; dup {
			return fmt.Errorf("overlay: group %q already registered", groupID)
		}
		s.reg[groupID] = struct{}{}

		_, err := s.db.ExecContext(context.Background(), sqlInsertGroup, string(groupID))
		return err
	}
}

// NewOverlayStore opens (or creates) the SQLite database at the given path.
// Parent directories are created if missing.
func NewOverlayStore(ctx context.Context, path string, opts ...Option) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("overlay: mkdir %s: %w", filepath.Dir(path), err)
	}

	db, err := sql.Open(
		"sqlite",
		path+"?busy_timeout=5000&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)",
	)
	if err != nil {
		return nil, fmt.Errorf("overlay: open sqlite: %w", err)
	}
	db.SetMaxOpenConns(2)
	if _, err := db.ExecContext(ctx, sqlCreateGroupsTable); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.ExecContext(ctx, sqlCreateFlagsTable); err != nil {
		_ = db.Close()
		return nil, err
	}

	st := &Store{
		db:  db,
		reg: make(map[GroupID]struct{}),
	}
	for _, opt := range opts {
		if err := opt(st); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	if len(st.reg) == 0 {
		_ = db.Close()
		return nil, errors.New("overlay: at least one group must be registered")
	}
	return st, nil
}

func (s *Store) GetFlag(ctx context.Context, k Key) (Flag, bool, error) {
	var zero Flag
	if err := s.ensureRegistered(k.Group()); err != nil {
		return zero, false, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.getFlagSQL(ctx, k.Group(), k.ID())
}

func (s *Store) SetFlag(ctx context.Context, k Key, val json.RawMessage) (Flag, error) {
	if err := s.ensureRegistered(k.Group()); err != nil {
		return Flag{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.setFlagSQL(ctx, k.Group(), k.ID(), val, time.Now().UTC())
}

func (s *Store) DeleteKey(ctx context.Context, k Key) error {
	if err := s.ensureRegistered(k.Group()); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(
		ctx,
		sqlDeleteFlag,
		string(k.Group()),
		string(k.ID()),
	)
	return err
}

// Close blocks until any in-flight Get/Set/Delete finishes (they hold s.mu),
// then closes the underlying *sql.DB exactly once.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closeOnce.Do(func() {
		if s.db != nil {
			s.closeErr = s.db.Close()
			s.db = nil
		}
	})

	return s.closeErr
}

func (s *Store) ensureRegistered(group GroupID) error {
	if _, ok := s.reg[group]; !ok {
		return fmt.Errorf("overlay: group %q is not registered", group)
	}
	return nil
}

func (s *Store) setFlagSQL(
	ctx context.Context,
	group GroupID,
	key KeyID,
	val json.RawMessage,
	now time.Time,
) (Flag, error) {
	old, exists, err := s.getFlagSQL(ctx, group, key)
	if err != nil {
		return Flag{}, err
	}

	if !exists {
		_, err = s.db.ExecContext(ctx, sqlInsertFlag,
			string(group), string(key), []byte(val), now, now)
		if err != nil {
			return Flag{}, err
		}
		return Flag{Value: val, CreatedAt: now, ModifiedAt: now}, nil
	}

	_, err = s.db.ExecContext(ctx, sqlUpdateFlag,
		[]byte(val), now, string(group), string(key))
	if err != nil {
		return Flag{}, err
	}
	return Flag{Value: val, CreatedAt: old.CreatedAt, ModifiedAt: now}, nil
}

func (s *Store) getFlagSQL(ctx context.Context, group GroupID, key KeyID) (Flag, bool, error) {
	var (
		raw               []byte
		created, modified time.Time
	)
	err := s.db.QueryRowContext(ctx, sqlSelectFlag, string(group), string(key)).
		Scan(&raw, &created, &modified)

	switch err {
	case nil:
		return Flag{
			Value:      json.RawMessage(raw),
			CreatedAt:  created,
			ModifiedAt: modified,
		}, true, nil
	case sql.ErrNoRows:
		return Flag{}, false, nil
	default:
		return Flag{}, false, err
	}
}
