package repository_test

import (
	"path/filepath"
	"testing"

	"github.com/Lin-Jiong-HDU/go-project-template/repository"
)

func TestInitDB_CreatesSchema(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := repository.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer func() { _ = db.Close() }()

	var count int
	err = db.QueryRow("SELECT count(*) FROM tasks").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query tasks table: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 rows, got %d", count)
	}
}

func TestInitDB_Idempotent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db1, err := repository.InitDB(dbPath)
	if err != nil {
		t.Fatalf("first InitDB failed: %v", err)
	}
	if err := db1.Close(); err != nil {
		t.Fatalf("close db1: %v", err)
	}

	db2, err := repository.InitDB(dbPath)
	if err != nil {
		t.Fatalf("second InitDB failed: %v", err)
	}
	defer func() { _ = db2.Close() }()

	var count int
	err = db2.QueryRow("SELECT count(*) FROM tasks").Scan(&count)
	if err != nil {
		t.Fatalf("failed after idempotent migration: %v", err)
	}
}
