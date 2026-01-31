// internal/db/postgres.go
package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"net"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// PostgresDriver implements Driver for PostgreSQL
type PostgresDriver struct {
	db     *sql.DB
	tunnel *SSHTunnel
}

// Connect establishes connection to PostgreSQL
func (d *PostgresDriver) Connect(params ConnectParams) error {
	// Build connection string
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", params.User, params.Password, params.Host, params.Port, params.Database)

	// Parse config
	connConfig, err := pgx.ParseConfig(dsn)
	if err != nil {
		return WrapConnectionError(err)
	}

	// Setup SSH tunnel if configured
	if params.SSHConfig != nil && params.SSHConfig.Host != "" {
		tunnel, err := NewSSHTunnel(params.SSHConfig)
		if err != nil {
			return WrapConnectionError(fmt.Errorf("failed to create SSH tunnel: %w", err))
		}
		d.tunnel = tunnel

		// Override DialFunc
		connConfig.DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return tunnel.Dial(network, addr)
		}
	}

	// Register the driver configuration with stdlib
	dbStr := stdlib.RegisterConnConfig(connConfig)
	db, err := sql.Open("pgx", dbStr)
	if err != nil {
		if d.tunnel != nil {
			d.tunnel.Close()
		}
		return WrapConnectionError(err)
	}

	// Configure connection pooling
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	d.db = db
	return nil
}

// Close closes the database connection and SSH tunnel
func (d *PostgresDriver) Close() error {
	var dbErr error
	if d.db != nil {
		dbErr = d.db.Close()
	}

	if d.tunnel != nil {
		if err := d.tunnel.Close(); err != nil {
			if dbErr != nil {
				return fmt.Errorf("db close err: %v, tunnel close err: %w", dbErr, err)
			}
			return err
		}
	}
	return dbErr
}

// Execute runs a query and returns results
func (d *PostgresDriver) Execute(ctx context.Context, query string) (*QueryResult, error) {
	return executeQuery(ctx, d.db, query)
}

// Ping checks if database is reachable
func (d *PostgresDriver) Ping(ctx context.Context) error {
	if d.db == nil {
		return WrapConnectionError(fmt.Errorf("not connected"))
	}
	return d.db.PingContext(ctx)
}

// Type returns the driver type
func (d *PostgresDriver) Type() DriverType {
	return Postgres
}

// GetTables returns a list of tables in the public schema
func (d *PostgresDriver) GetTables(ctx context.Context) ([]string, error) {
	query := "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'"
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
func (d *PostgresDriver) GetColumns(ctx context.Context, tableName string) ([]Column, error) {
	query := `
		SELECT 
			column_name, 
			data_type, 
			is_nullable = 'YES' as nullable, 
			COALESCE(column_default, '') as default_value,
			COALESCE((
				SELECT 'PRI' 
				FROM information_schema.key_column_usage k 
				JOIN information_schema.table_constraints tc ON k.constraint_name = tc.constraint_name 
				WHERE k.table_name = c.table_name 
				AND k.column_name = c.column_name 
				AND tc.constraint_type = 'PRIMARY KEY' 
				LIMIT 1
			), '') as key_type
		FROM information_schema.columns c
		WHERE table_name = $1 AND table_schema = 'public'
		ORDER BY ordinal_position`

	rows, err := d.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, WrapQueryError(err)
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var col Column
		if err := rows.Scan(&col.Name, &col.Type, &col.Nullable, &col.Default, &col.Key); err != nil {
			return nil, WrapQueryError(err)
		}
		columns = append(columns, col)
	}
	return columns, rows.Err()
}

// GetConstraints returns detailed constraint metadata for a table
func (d *PostgresDriver) GetConstraints(ctx context.Context, tableName string) ([]Constraint, error) {
	query := `
		SELECT 
			conname as name, 
			CASE 
				WHEN contype = 'p' THEN 'PRIMARY KEY'
				WHEN contype = 'f' THEN 'FOREIGN KEY'
				WHEN contype = 'u' THEN 'UNIQUE'
				WHEN contype = 'c' THEN 'CHECK'
				ELSE contype::text
			END as type, 
			pg_get_constraintdef(c.oid) as definition
		FROM pg_constraint c
		JOIN pg_class cl ON cl.oid = c.conrelid
		JOIN pg_namespace n ON n.oid = cl.relnamespace
		WHERE cl.relname = $1 AND n.nspname = 'public'`

	rows, err := d.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, WrapQueryError(err)
	}
	defer rows.Close()

	var constraints []Constraint
	for rows.Next() {
		var cons Constraint
		if err := rows.Scan(&cons.Name, &cons.Type, &cons.Definition); err != nil {
			return nil, WrapQueryError(err)
		}
		constraints = append(constraints, cons)
	}
	return constraints, rows.Err()
}
