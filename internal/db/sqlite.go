// internal/db/sqlite.go
package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteDriver implements Driver for SQLite
type SQLiteDriver struct {
	db *sql.DB
}

// Connect establishes connection to SQLite
func (d *SQLiteDriver) Connect(params ConnectParams) error {
	// For SQLite, the database strings is the filepath
	// Strip sqlite:// prefix if present
	dsn := params.Database
	if len(dsn) > 9 && dsn[:9] == "sqlite://" {
		dsn = dsn[9:]
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return WrapConnectionError(err)
	}

	// Apply SQLite pragmas for better performance and safety
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return WrapConnectionError(fmt.Errorf("pragma foreign_keys: %w", err))
	}
	if _, err := db.Exec("PRAGMA busy_timeout = 10000"); err != nil {
		return WrapConnectionError(fmt.Errorf("pragma busy_timeout: %w", err))
	}

	d.db = db
	return nil
}

// Close closes the database connection
func (d *SQLiteDriver) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// Execute runs a query and returns results
func (d *SQLiteDriver) Execute(ctx context.Context, query string) (*QueryResult, error) {
	return executeQuery(ctx, d.db, query)
}

// Ping checks if database is reachable
func (d *SQLiteDriver) Ping(ctx context.Context) error {
	if d.db == nil {
		return WrapConnectionError(fmt.Errorf("not connected"))
	}
	return d.db.PingContext(ctx)
}

// Type returns the driver type
func (d *SQLiteDriver) Type() DriverType {
	return SQLite
}

// GetTables returns a list of tables
func (d *SQLiteDriver) GetTables(ctx context.Context) ([]string, error) {
	query := "SELECT name FROM sqlite_master WHERE type='table'"
	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, WrapQueryError(err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, WrapQueryError(err)
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

// GetColumns returns detailed column metadata for a table
func (d *SQLiteDriver) GetColumns(ctx context.Context, tableName string) ([]Column, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, WrapQueryError(err)
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var cid int
		var name string
		var dataType string
		var notNull int
		var dfltValue interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
			return nil, WrapQueryError(err)
		}

		key := ""
		if pk > 0 {
			key = "PRI"
		}

		columns = append(columns, Column{
			Name:     name,
			Type:     dataType,
			Nullable: notNull == 0,
			Default:  fmt.Sprintf("%v", dfltValue),
			Key:      key,
		})
	}
	return columns, rows.Err()
}

// GetConstraints returns detailed constraint metadata for a table
func (d *SQLiteDriver) GetConstraints(ctx context.Context, tableName string) ([]Constraint, error) {
	var constraints []Constraint

	// Foreign keys
	rows, err := d.db.QueryContext(ctx, fmt.Sprintf("PRAGMA foreign_key_list(%s)", tableName))
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, seq int
			var table, from, to, onUpdate, onDelete, match string
			if err := rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err == nil {
				constraints = append(constraints, Constraint{
					Name:       fmt.Sprintf("fk_%s_%d", tableName, id),
					Type:       "FOREIGN KEY",
					Definition: fmt.Sprintf("REFERENCES %s(%s) ON UPDATE %s ON DELETE %s", table, to, onUpdate, onDelete),
				})
			}
		}
	}

	return constraints, nil
}
