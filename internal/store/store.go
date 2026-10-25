// Package store gestiona la base de datos SQLite de Cinecat: apertura,
// migraciones del esquema y acceso al *sql.DB.
package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite" // driver SQLite en Go puro (BSD-3-Clause)
)

//go:embed migrations/*.sql
var migrations embed.FS

const driverName = "sqlite"

const createMigrationsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version    TEXT PRIMARY KEY,
	applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
) STRICT;`

// Store envuelve la conexión a la base de datos.
type Store struct {
	db *sql.DB
}

// Open abre (o crea) la base de datos SQLite en path. Solo acepta archivos,
// salvo ":memory:", pensado para pruebas.
func Open(ctx context.Context, path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("store: falta la ruta de la base de datos")
	}

	dsn, inMemory, err := dataSourceName(path)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("store: abrir base de datos: %w", err)
	}
	if inMemory {
		// Cada conexión a ":memory:" sería una base distinta.
		db.SetMaxOpenConns(1)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: conectar con la base de datos: %w", err)
	}

	return &Store{db: db}, nil
}

// DB devuelve la conexión subyacente para consultas.
func (s *Store) DB() *sql.DB {
	return s.db
}

// Close cierra la base de datos.
func (s *Store) Close() error {
	return s.db.Close()
}

// Migrate aplica, en orden, las migraciones embebidas que aún no estén
// registradas en schema_migrations. Cada migración se aplica en una
// transacción, así que un fallo no deja el esquema a medias.
func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, createMigrationsTable); err != nil {
		return fmt.Errorf("store: crear tabla de migraciones: %w", err)
	}

	names, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("store: listar migraciones: %w", err)
	}
	sort.Strings(names)

	for _, name := range names {
		version := strings.TrimSuffix(filepath.Base(name), ".sql")

		applied, err := s.migrationApplied(ctx, version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		script, err := migrations.ReadFile(name)
		if err != nil {
			return fmt.Errorf("store: leer migración %s: %w", version, err)
		}
		if err := s.applyMigration(ctx, version, string(script)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) migrationApplied(ctx context.Context, version string) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx,
		`SELECT 1 FROM schema_migrations WHERE version = ?`, version,
	).Scan(&exists)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("store: consultar migración %s: %w", version, err)
	default:
		return true, nil
	}
}

func (s *Store) applyMigration(ctx context.Context, version, script string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: iniciar migración %s: %w", version, err)
	}
	defer tx.Rollback() //nolint:errcheck // el Commit prevalece si se hizo

	if _, err := tx.ExecContext(ctx, script); err != nil {
		return fmt.Errorf("store: aplicar migración %s: %w", version, err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version) VALUES (?)`, version,
	); err != nil {
		return fmt.Errorf("store: registrar migración %s: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: confirmar migración %s: %w", version, err)
	}
	return nil
}

func dataSourceName(path string) (dsn string, inMemory bool, err error) {
	if path == ":memory:" {
		return "file::memory:?cache=shared&_pragma=foreign_keys(1)", true, nil
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return "", false, fmt.Errorf("store: ruta de base de datos no válida: %w", err)
	}

	// URI file:/// con las barras normalizadas, para que funcionen rutas de
	// Windows y nombres con espacios.
	slash := filepath.ToSlash(abs)
	u := &url.URL{Scheme: "file", Path: "/" + strings.TrimPrefix(slash, "/")}
	q := u.Query()
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "journal_mode(WAL)")
	u.RawQuery = q.Encode()
	return u.String(), false, nil
}
