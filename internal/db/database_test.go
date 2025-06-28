package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func TestInitDB(t *testing.T) {
	// Create a temporary directory for the test DB
	tempDir, err := os.MkdirTemp("", "fman_test_db")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir) // Clean up after test

	// Override user home directory for test
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Initialize DB
	testDB := NewDatabase(nil).(*Database) // Create a new Database instance for testing

	err = testDB.InitDB()
	assert.NoError(t, err)
	assert.NotNil(t, testDB.db) // Ensure db connection is established

	// Check if the database file was created
	dbFile := filepath.Join(tempDir, ".fman", "fman.db")
	_, err = os.Stat(dbFile)
	assert.False(t, os.IsNotExist(err), "Database file should exist")

	// Check if the files table exists and has the correct schema
	rows, err := testDB.db.Query("PRAGMA table_info(files)")
	assert.NoError(t, err)
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt_value interface{}
		var pk int
		assert.NoError(t, rows.Scan(&cid, &name, &ctype, &notnull, &dflt_value, &pk))
		columns[name] = true
	}

	assert.True(t, columns["id"], "id column should exist")
	assert.True(t, columns["path"], "path column should exist")
	assert.True(t, columns["name"], "name column should exist")
	assert.True(t, columns["size"], "size column should exist")
	assert.True(t, columns["modified_at"], "modified_at column should exist")
	assert.True(t, columns["indexed_at"], "indexed_at column should exist")
	assert.True(t, columns["file_hash"], "file_hash column should exist")

	// Close the DB connection after test
	assert.NoError(t, testDB.Close())
}

func TestUpsertFile(t *testing.T) {
	// Setup in-memory DB for test
	testDB := setupTestDB(t).(*Database)
	defer teardownTestDB(t, testDB)

	file1 := &File{
		Path:       "/test/path/file1.txt",
		Name:       "file1.txt",
		Size:       100,
		ModifiedAt: time.Now().Add(-24 * time.Hour),
		FileHash:   "hash1",
	}

	// Test insert
	err := testDB.UpsertFile(file1)
	assert.NoError(t, err)

	var retrievedFile File
	err = testDB.db.Get(&retrievedFile, "SELECT * FROM files WHERE path=?", file1.Path)
	assert.NoError(t, err)
	assert.Equal(t, file1.Path, retrievedFile.Path)
	assert.Equal(t, file1.Name, retrievedFile.Name)
	assert.Equal(t, file1.Size, retrievedFile.Size)
	assert.Equal(t, file1.FileHash, retrievedFile.FileHash)

	// Test update
	file1.Size = 200
	file1.FileHash = "new_hash1"
	err = testDB.UpsertFile(file1)
	assert.NoError(t, err)

	err = testDB.db.Get(&retrievedFile, "SELECT * FROM files WHERE path=?", file1.Path)
	assert.NoError(t, err)
	assert.Equal(t, file1.Size, retrievedFile.Size)
	assert.Equal(t, file1.FileHash, retrievedFile.FileHash)
}

func TestFindFilesByName(t *testing.T) {
	// Setup in-memory DB for test
	testDB := setupTestDB(t).(*Database)
	defer teardownTestDB(t, testDB)

	filesToInsert := []*File{
		{
			Path:       "/test/path/document.pdf",
			Name:       "document.pdf",
			Size:       500,
			ModifiedAt: time.Now(),
			FileHash:   "hash_doc",
		},
		{
			Path:       "/another/path/image.jpg",
			Name:       "image.jpg",
			Size:       1024,
			ModifiedAt: time.Now(),
			FileHash:   "hash_img",
		},
		{
			Path:       "/docs/my_document.docx",
			Name:       "my_document.docx",
			Size:       300,
			ModifiedAt: time.Now(),
			FileHash:   "hash_mydoc",
		},
	}

	for _, f := range filesToInsert {
		assert.NoError(t, testDB.UpsertFile(f))
	}

	// Test exact match
	foundFiles, err := testDB.FindFilesByName("document.pdf")
	assert.NoError(t, err)
	assert.Len(t, foundFiles, 1)
	assert.Equal(t, "document.pdf", foundFiles[0].Name)

	// Test partial match
	foundFiles, err = testDB.FindFilesByName("doc")
	assert.NoError(t, err)
	assert.Len(t, foundFiles, 2)

	// Test case-insensitivity (SQLite LIKE is case-insensitive by default for ASCII)
	foundFiles, err = testDB.FindFilesByName("DOC")
	assert.NoError(t, err)
	assert.Len(t, foundFiles, 2)

	// Test no match
	foundFiles, err = testDB.FindFilesByName("nonexistent")
	assert.NoError(t, err)
	assert.Len(t, foundFiles, 0)
}

// setupTestDB initializes an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) DBInterface {
	// Use a unique in-memory database for each test function
	dbConn, err := sqlx.Connect("sqlite3", ":memory:?_foreign_keys=on")
	assert.NoError(t, err)

	testDB := NewDatabase(dbConn)

	// Create the files table
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
	_, err = testDB.(*Database).db.Exec(schema)
	assert.NoError(t, err)

	return testDB
}

// teardownTestDB closes the test database connection.
func teardownTestDB(t *testing.T, testDB DBInterface) {
	assert.NoError(t, testDB.Close())
}
