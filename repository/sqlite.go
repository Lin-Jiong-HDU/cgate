package repository

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := migrate(db); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("migrate: %w (close: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		issue_number INTEGER NOT NULL,
		title TEXT NOT NULL,
		body TEXT DEFAULT '',
		author TEXT DEFAULT '',
		repository TEXT NOT NULL,
		html_url TEXT DEFAULT '',
		status TEXT NOT NULL DEFAULT 'pending',
		container_id TEXT DEFAULT '',
		log TEXT DEFAULT '',
		created_at DATETIME NOT NULL,
		started_at DATETIME,
		finished_at DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_tasks_repo_issue ON tasks(repository, issue_number);
	`
	_, err := db.Exec(schema)
	return err
}
