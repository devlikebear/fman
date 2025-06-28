package db

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// File represents a file in the database.
type File struct {
	ID         int64     `db:"id"`
	Path       string    `db:"path"`
	Name       string    `db:"name"`
	Size       int64     `db:"size"`
	ModifiedAt time.Time `db:"modified_at"`
	IndexedAt  time.Time `db:"indexed_at"`
	FileHash   string    `db:"file_hash"`
}

// DBInterface defines the interface for database operations.
type DBInterface interface {
	InitDB() error
	UpsertFile(file *File) error
	FindFilesByName(namePattern string) ([]File, error)
	Close() error
}

// Database implements DBInterface using sqlx.
type Database struct {
	db *sqlx.DB
	dbPath string // Path to the database file
}

// NewDatabase creates a new Database instance.
// If dbConn is nil, it will attempt to connect to the default fman.db.
func NewDatabase(dbConn *sqlx.DB) DBInterface {
	return &Database{db: dbConn}
}

// InitDB initializes the database connection and creates the necessary tables.
func (d *Database) InitDB() error {
	if d.db != nil { // If already initialized or mocked
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	dbDir := filepath.Join(home, ".fman")
	if err := os.MkdirAll(dbDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create db directory: %w", err)
	}

	dbFile := filepath.Join(dbDir, "fman.db")

	dbConn, err := sqlx.Connect("sqlite3", dbFile)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	d.db = dbConn

	// Create the files table if it doesn't exist.
	schema := `
	CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		size INTEGER NOT NULL,
		modified_at TIMESTAMP NOT NULL,
		indexed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		file_hash TEXT
	);
	`
	_, err = d.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

// UpsertFile inserts a new file record or updates an existing one based on the path.
func (d *Database) UpsertFile(file *File) error {
	query := `
	INSERT INTO files (path, name, size, modified_at, file_hash, indexed_at)
	VALUES (?, ?, ?, ?, ?, ?)
	ON CONFLICT(path) DO UPDATE SET
		name = excluded.name,
		size = excluded.size,
		modified_at = excluded.modified_at,
		file_hash = excluded.file_hash,
		indexed_at = excluded.indexed_at;
	`
	_, err := d.db.Exec(query, file.Path, file.Name, file.Size, file.ModifiedAt, file.FileHash, time.Now())
	return err
}

// FindFilesByName searches for files by name using a LIKE query.
func (d *Database) FindFilesByName(namePattern string) ([]File, error) {
	var files []File
	query := "SELECT * FROM files WHERE name LIKE ?"
	err := d.db.Select(&files, query, "%"+namePattern+"%")
	if err != nil {
		return nil, err
	}
	return files, nil
}

// Close closes the database connection.
func (d *Database) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}
