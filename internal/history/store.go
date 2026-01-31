// internal/history/store.go
package history

import (
	"database/sql"

	"github.com/adrg/xdg"
	_ "github.com/mattn/go-sqlite3"
)

// Store manages query history persistence
type Store struct {
	db *sql.DB
}

// NewStore creates a new history store with SQLite backend
func NewStore() (*Store, error) {
	dbPath, err := xdg.DataFile("ezdb/history.db")
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Apply SQLite pragmas
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		return nil, err
	}

	// Create table and indexes
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			profile_name TEXT NOT NULL,
			query TEXT NOT NULL,
			executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			duration_ms INTEGER NOT NULL,
			row_count INTEGER NOT NULL,
			status TEXT NOT NULL,
			error_message TEXT,
			preview TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_history_profile ON history(profile_name);
		CREATE INDEX IF NOT EXISTS idx_history_executed_at ON history(executed_at);
	`)
	if err != nil {
		return nil, err
	}

	// Migration: Ensure preview column exists for existing databases
	// This will fail silently if the column already exists or if there's another issue,
	// which is acceptable for a simple development migration.
	_, _ = db.Exec("ALTER TABLE history ADD COLUMN preview TEXT")

	store := &Store{db: db}
	// Run cleanup on initialization
	if err := store.cleanup(); err != nil {
		// Don't fail on cleanup error, just log it
		_ = err
	}
	return store, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// Add inserts a new execution into history
func (s *Store) Add(entry *HistoryEntry) error {
	query := `
		INSERT INTO history (profile_name, query, executed_at, duration_ms, row_count, status, error_message, preview)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := s.db.Exec(query,
		entry.ProfileName,
		entry.Query,
		entry.ExecutedAt,
		entry.DurationMs,
		entry.RowCount,
		entry.Status,
		entry.ErrorMessage,
		entry.Preview,
	)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	entry.ID = id

	// Prune old entries
	go s.cleanup()

	return nil
}

// enforceLimit keeps only the most recent N entries per profile
func (s *Store) enforceLimit(profileName string, limit int) error {
	_, err := s.db.Exec(`
		DELETE FROM history
		WHERE profile_name = ?
		AND id NOT IN (
			SELECT id FROM history
			WHERE profile_name = ?
			ORDER BY executed_at DESC
			LIMIT ?
		)
	`, profileName, profileName, limit)
	return err
}

// List returns paginated history entries for a profile
func (s *Store) List(profileName string, limit, offset int) ([]HistoryEntry, error) {
	rows, err := s.db.Query(`
		SELECT id, profile_name, query, executed_at, duration_ms, row_count, status, error_message, preview
		FROM history
		WHERE profile_name = ?
		ORDER BY executed_at DESC
		LIMIT ? OFFSET ?
	`, profileName, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEntries(rows)
}

// Search finds history entries by query substring
func (s *Store) Search(profileName, querySubstr string, limit int) ([]HistoryEntry, error) {
	rows, err := s.db.Query(`
		SELECT id, profile_name, query, executed_at, duration_ms, row_count, status, error_message, preview
		FROM history
		WHERE profile_name = ? AND query LIKE ?
		ORDER BY executed_at DESC
		LIMIT ?
	`, profileName, "%"+querySubstr+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEntries(rows)
}

// scanEntries scans rows into HistoryEntry slice
func scanEntries(rows *sql.Rows) ([]HistoryEntry, error) {
	var entries []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		var preview sql.NullString
		err := rows.Scan(&e.ID, &e.ProfileName, &e.Query, &e.ExecutedAt,
			&e.DurationMs, &e.RowCount, &e.Status, &e.ErrorMessage, &preview)
		if preview.Valid {
			e.Preview = preview.String
		}
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// GetByID retrieves a single history entry by ID
func (s *Store) GetByID(id int64) (*HistoryEntry, error) {
	row := s.db.QueryRow(`
		SELECT id, profile_name, query, executed_at, duration_ms, row_count, status, error_message, preview
		FROM history WHERE id = ?
	`, id)

	var e HistoryEntry
	var preview sql.NullString
	err := row.Scan(&e.ID, &e.ProfileName, &e.Query, &e.ExecutedAt,
		&e.DurationMs, &e.RowCount, &e.Status, &e.ErrorMessage, &preview)
	if preview.Valid {
		e.Preview = preview.String
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &e, err
}

// Delete removes a history entry by ID
func (s *Store) Delete(id int64) error {
	_, err := s.db.Exec("DELETE FROM history WHERE id = ?", id)
	return err
}

// cleanup removes history entries older than 90 days
func (s *Store) cleanup() error {
	_, err := s.db.Exec(`
		DELETE FROM history
		WHERE executed_at < datetime('now', '-90 days')
	`)
	return err
}

// Count returns the total number of history entries for a profile
func (s *Store) Count(profileName string) (int, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM history WHERE profile_name = ?
	`, profileName).Scan(&count)
	return count, err
}
