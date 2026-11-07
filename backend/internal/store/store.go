// Package store owns the SQLite database: connections, migrations and integrity checks.
package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/serraniel/wokko/backend/internal/store/sqlcgen"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, so builds need no C toolchain
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Store has one writer connection, so writes queue instead of failing, and a reader pool
// that keeps serving while a write is in progress (WAL mode).
type Store struct {
	path    string
	writer  *sql.DB
	reader  *sql.DB
	Queries *sqlcgen.Queries // read-only queries; writes go through Tx
}

func dsn(p string, readOnly bool) string {
	q := url.Values{}
	for _, pragma := range []string{"busy_timeout(10000)", "journal_mode(WAL)", "foreign_keys(ON)", "synchronous(FULL)"} {
		q.Add("_pragma", pragma)
	}
	if readOnly {
		q.Add("_pragma", "query_only(ON)")
	} else {
		q.Set("_txlock", "immediate")
	}
	return "file:" + p + "?" + q.Encode()
}

// Open connects to the database file, creating it if necessary.
func Open(p string) (*Store, error) {
	w, err := sql.Open("sqlite", dsn(p, false))
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	w.SetMaxOpenConns(1)
	if err := w.Ping(); err != nil {
		_ = w.Close()
		return nil, fmt.Errorf("open database %s: %w", p, err)
	}
	r, err := sql.Open("sqlite", dsn(p, true))
	if err != nil {
		_ = w.Close()
		return nil, fmt.Errorf("open database: %w", err)
	}
	r.SetMaxOpenConns(8)
	return &Store{path: p, writer: w, reader: r, Queries: sqlcgen.New(r)}, nil
}

func (s *Store) Close() error {
	return errors.Join(s.reader.Close(), s.writer.Close())
}

func (s *Store) Path() string { return s.path }

// Tx runs fn in one transaction. Everything fn does is saved together, or not at all.
func (s *Store) Tx(ctx context.Context, fn func(q *sqlcgen.Queries) error) error {
	tx, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op after a successful commit
	if err := fn(sqlcgen.New(tx)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

type migration struct {
	version int
	name    string
	sql     string
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return nil, err
	}
	var out []migration
	for _, e := range entries {
		num, _, ok := strings.Cut(e.Name(), "_")
		if !ok {
			return nil, fmt.Errorf("bad migration file name %q", e.Name())
		}
		v, err := strconv.Atoi(num)
		if err != nil {
			return nil, fmt.Errorf("bad migration file name %q: %w", e.Name(), err)
		}
		b, err := migrationFS.ReadFile(path.Join("migrations", e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, migration{v, e.Name(), string(b)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

// ErrDatabaseTooNew means the data was written by a newer Wokko and must not be touched.
var ErrDatabaseTooNew = errors.New("database is newer than this version of Wokko")

// Migrate applies pending migrations, each in its own transaction, so a failure leaves the
// database at the last good version. beforeUpgrade runs only when an existing database is
// about to change; if it fails, nothing is migrated.
func (s *Store) Migrate(ctx context.Context, beforeUpgrade func() error) error {
	if _, err := s.writer.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY, name TEXT NOT NULL,
		applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))) STRICT`); err != nil {
		return fmt.Errorf("prepare migrations: %w", err)
	}
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}
	current, err := s.SchemaVersion(ctx)
	if err != nil {
		return err
	}
	latest := 0
	if n := len(migrations); n > 0 {
		latest = migrations[n-1].version
	}
	if current > latest {
		return fmt.Errorf("%w: database is at version %d, this build knows up to %d. Run a newer Wokko or restore a backup", ErrDatabaseTooNew, current, latest)
	}
	if current == latest {
		return nil
	}
	if current > 0 && beforeUpgrade != nil {
		if err := beforeUpgrade(); err != nil {
			return fmt.Errorf("safety backup before upgrade failed, database left unchanged: %w", err)
		}
	}
	for _, m := range migrations {
		if m.version <= current {
			continue
		}
		if err := s.apply(ctx, m); err != nil {
			return fmt.Errorf("migration %s failed, database left at version %d: %w", m.name, current, err)
		}
		current = m.version
	}
	return nil
}

func (s *Store) apply(ctx context.Context, m migration) error {
	tx, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, m.sql); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, name) VALUES (?, ?)`, m.version, m.name); err != nil {
		return err
	}
	return tx.Commit()
}

// SchemaVersion returns the newest applied migration, or 0 for a new database.
func (s *Store) SchemaVersion(ctx context.Context) (int, error) {
	var v sql.NullInt64
	if err := s.writer.QueryRowContext(ctx, `SELECT max(version) FROM schema_migrations`).Scan(&v); err != nil {
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	return int(v.Int64), nil
}

// LatestSchemaVersion is the newest migration this build ships.
func LatestSchemaVersion() (int, error) {
	m, err := loadMigrations()
	if err != nil || len(m) == 0 {
		return 0, err
	}
	return m[len(m)-1].version, nil
}

// IntegrityCheck verifies the database file and its foreign keys.
func (s *Store) IntegrityCheck(ctx context.Context) error {
	return CheckDB(ctx, s.reader)
}

// CheckDB runs SQLite's integrity and foreign key checks on any connection.
func CheckDB(ctx context.Context, db *sql.DB) error {
	var res string
	if err := db.QueryRowContext(ctx, `PRAGMA integrity_check(1)`).Scan(&res); err != nil {
		return fmt.Errorf("integrity check: %w", err)
	}
	if res != "ok" {
		return fmt.Errorf("integrity check found a problem: %s", res)
	}
	rows, err := db.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return fmt.Errorf("foreign key check: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		return errors.New("foreign key check found rows pointing at missing data")
	}
	return rows.Err()
}

// ForeignKeysOn reports whether a connection enforces foreign keys. SQLite defaults to off.
func (s *Store) ForeignKeysOn(ctx context.Context) (reader, writer bool, err error) {
	var r, w int
	if err = s.reader.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&r); err != nil {
		return
	}
	if err = s.writer.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&w); err != nil {
		return
	}
	return r == 1, w == 1, nil
}

// SnapshotTo writes a consistent copy of the database to dest without blocking readers.
func (s *Store) SnapshotTo(ctx context.Context, dest string) error {
	if _, err := s.writer.ExecContext(ctx, `VACUUM INTO ?`, dest); err != nil {
		return fmt.Errorf("snapshot database: %w", err)
	}
	return nil
}

// ForceSchemaVersionForTest pretends the database is at a given version. Tests only.
func (s *Store) ForceSchemaVersionForTest(ctx context.Context, v int) error {
	tx, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM schema_migrations WHERE version > ?`, v); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO schema_migrations (version, name) VALUES (?, 'forced')`, v); err != nil {
		return err
	}
	return tx.Commit()
}
