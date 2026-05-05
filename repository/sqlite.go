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
		finished_at DATETIME,
		task_type TEXT NOT NULL DEFAULT 'issue',
		pr_number INTEGER DEFAULT 0,
		comment_id INTEGER DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_tasks_repo_issue ON tasks(repository, issue_number);
	CREATE INDEX IF NOT EXISTS idx_tasks_repo_pr_type ON tasks(repository, pr_number, task_type);
	`
	if _, err := db.Exec(schema); err != nil {
		return err
	}

	return migrateV2(db)
}

func migrateV2(db *sql.DB) error {
	type columnMigration struct {
		table  string
		column string
		alter  string
	}
	migrations := []columnMigration{
		{"tasks", "task_type", "ALTER TABLE tasks ADD COLUMN task_type TEXT NOT NULL DEFAULT 'issue'"},
		{"tasks", "pr_number", "ALTER TABLE tasks ADD COLUMN pr_number INTEGER DEFAULT 0"},
		{"tasks", "comment_id", "ALTER TABLE tasks ADD COLUMN comment_id INTEGER DEFAULT 0"},
	}
	for _, m := range migrations {
		exists, err := columnExists(db, m.table, m.column)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := db.Exec(m.alter); err != nil {
			return err
		}
	}
	_, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_tasks_repo_pr_type ON tasks(repository, pr_number, task_type)")
	return err
}

func columnExists(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return false, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dfltValue interface{}
		var pk interface{}
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}
