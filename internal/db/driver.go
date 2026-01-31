// internal/db/driver.go
package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// DriverType represents supported database types
type DriverType string

const (
	Postgres DriverType = "postgres"
	MySQL    DriverType = "mysql"
	SQLite   DriverType = "sqlite"
)

// Column represents table column metadata
type Column struct {
	Name     string
	Type     string
	Nullable bool
	Default  string
	Key      string // PRI, UNI, MUL
}

// Constraint represents table constraint metadata
type Constraint struct {
	Name       string
	Type       string // PRIMARY KEY, FOREIGN KEY, UNIQUE, etc.
	Definition string
}

// ConnectParams holds database connection details
type ConnectParams struct {
	Host      string
	Port      int
	User      string
	Password  string
	Database  string
	SSHConfig *SSHConfig // Optional SSH tunnel config
}

// Driver defines the interface for database operations
type Driver interface {
	Connect(params ConnectParams) error
	Close() error
	Execute(ctx context.Context, query string) (*QueryResult, error)
	Ping(ctx context.Context) error
	Type() DriverType
	GetTables(ctx context.Context) ([]string, error)
	GetColumns(ctx context.Context, tableName string) ([]Column, error)
	GetConstraints(ctx context.Context, tableName string) ([]Constraint, error)
}

// QueryResult contains query execution results
type QueryResult struct {
	Columns      []string
	Rows         [][]string
	ExecTime     time.Duration
	RowCount     int
	IsSelect     bool
	AffectedRows int64
}

// NewDriver creates a new driver instance by type
func NewDriver(driverType DriverType) (Driver, error) {
	switch driverType {
	case Postgres:
		return &PostgresDriver{}, nil
	case MySQL:
		return &MySQLDriver{}, nil
	case SQLite:
		return &SQLiteDriver{}, nil
	default:
		return nil, fmt.Errorf("unknown driver type: %s", driverType)
	}
}

// executeQuery executes a query and returns results
func executeQuery(ctx context.Context, db *sql.DB, query string) (*QueryResult, error) {
	start := time.Now()
	trimmed := strings.TrimSpace(strings.ToUpper(query))

	// Detect SELECT vs DML
	if strings.HasPrefix(trimmed, "SELECT") || strings.HasPrefix(trimmed, "WITH") ||
		strings.HasPrefix(trimmed, "EXPLAIN") || strings.HasPrefix(trimmed, "DESCRIBE") ||
		strings.HasPrefix(trimmed, "SHOW") {
		return executeSelect(ctx, db, query, start)
	}
	return executeDML(ctx, db, query, start)
}

// executeSelect executes a SELECT query
func executeSelect(ctx context.Context, db *sql.DB, query string, start time.Time) (*QueryResult, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, WrapQueryError(err)
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	var results [][]string

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, WrapQueryError(err)
		}

		row := make([]string, len(columns))
		for i, v := range values {
			row[i] = formatValue(v)
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, WrapQueryError(err)
	}

	return &QueryResult{
		Columns:  columns,
		Rows:     results,
		ExecTime: time.Since(start),
		RowCount: len(results),
		IsSelect: true,
	}, nil
}

// executeDML executes INSERT/UPDATE/DELETE queries
func executeDML(ctx context.Context, db *sql.DB, query string, start time.Time) (*QueryResult, error) {
	result, err := db.ExecContext(ctx, query)
	if err != nil {
		return nil, WrapQueryError(err)
	}
	affected, _ := result.RowsAffected()
	return &QueryResult{
		ExecTime:     time.Since(start),
		IsSelect:     false,
		AffectedRows: affected,
	}, nil
}

// formatValue converts interface{} to string for display
func formatValue(v interface{}) string {
	if v == nil {
		return "NULL"
	}

	switch val := v.(type) {
	case []byte:
		return string(val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}
