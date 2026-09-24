package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var (
	ErrNotFound     = errors.New("store: not found")
	ErrLoginTaken   = errors.New("store: login already taken")
	ErrLoginInvalid = errors.New("store: invalid login")
)

type Store struct {
	db     *sql.DB
	idsMu  sync.Mutex
	now    func() int64
	closed bool
}

func Open(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}
	dsn := path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db, now: func() int64 { return time.Now().Unix() }}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	s.closed = true
	return s.db.Close()
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) Now() int64 {
	return s.now()
}

func (s *Store) migrate() error {
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

func (s *Store) nextID(name string) (int, error) {
	s.idsMu.Lock()
	defer s.idsMu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var value int
	err = tx.QueryRow(`SELECT value FROM counters WHERE name = ?`, name).Scan(&value)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		value = 1
	case err != nil:
		return 0, err
	default:
		value++
	}
	if _, err := tx.Exec(`INSERT INTO counters(name, value) VALUES(?, ?)
		ON CONFLICT(name) DO UPDATE SET value = excluded.value`, name, value); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return value, nil
}

func (s *Store) SetCounter(name string, value int) error {
	_, err := s.db.Exec(`INSERT INTO counters(name, value) VALUES(?, ?)
		ON CONFLICT(name) DO UPDATE SET value = excluded.value`, name, value)
	return err
}

func (s *Store) counter(name string) int {
	var value int
	if err := s.db.QueryRow(`SELECT value FROM counters WHERE name = ?`, name).Scan(&value); err != nil {
		return 0
	}
	return value
}

func (s *Store) exec(query string, args ...any) error {
	_, err := s.db.Exec(query, args...)
	return err
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func intsToAny(values []int) []any {
	out := make([]any, 0, len(values))
	for _, v := range values {
		out = append(out, v)
	}
	return out
}
