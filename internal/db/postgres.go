// internal/db/postgres.go
package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"net"
	"net/url"

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
	// Build connection string safely with url.URL
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(params.User, params.Password),
		Host:   fmt.Sprintf("%s:%d", params.Host, params.Port),
		Path:   "/" + params.Database,
	}
	dsn := u.String()

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

		// IMPORTANT: Override LookupFunc to do nothing. We want the SSH server
		// to resolve the hostname, not the local machine.
		connConfig.LookupFunc = func(ctx context.Context, host string) ([]string, error) {
			return []string{host}, nil
		}

		// Override DialFunc to use DialContext and hostname
		connConfig.DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
			remoteAddr := fmt.Sprintf("%s:%d", params.Host, params.Port)
			return tunnel.DialContext(ctx, network, remoteAddr)
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

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		if d.tunnel != nil {
			d.tunnel.Close()
		}
		return WrapConnectionError(err)
	}

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

// GetTables returns a list of tables in all non-system schemas
func (d *PostgresDriver) GetTables(ctx context.Context) ([]string, error) {
	query := `
		SELECT n.nspname || '.' || c.relname
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
		AND c.relkind IN ('r', 'v', 'm', 'f', 'p')
		ORDER BY 1`
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
			a.attname AS column_name,
			format_type(a.atttypid, a.atttypmod) AS data_type,
			NOT a.attnotnull AS nullable,
			COALESCE(pg_get_expr(d.adbin, d.adrelid), '') AS default_value,
			COALESCE(
				(SELECT 'PRI' FROM pg_index i WHERE i.indrelid = a.attrelid AND a.attnum = ANY(i.indkey::int2[]) AND i.indisprimary LIMIT 1),
				(SELECT 'UNI' FROM pg_index i WHERE i.indrelid = a.attrelid AND a.attnum = ANY(i.indkey::int2[]) AND i.indisunique AND NOT i.indisprimary LIMIT 1),
				(SELECT 'FK' FROM pg_constraint c WHERE c.conrelid = a.attrelid AND a.attnum = ANY(c.conkey::int2[]) AND c.contype = 'f' LIMIT 1),
				''
			) AS key_type
		FROM pg_attribute a
		LEFT JOIN pg_attrdef d ON a.attrelid = d.adrelid AND a.attnum = d.adnum
		JOIN pg_class cl ON a.attrelid = cl.oid
		JOIN pg_namespace n ON cl.relnamespace = n.oid
		WHERE n.nspname || '.' || cl.relname = $1 AND a.attnum > 0 AND NOT a.attisdropped
		ORDER BY a.attnum`

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
		WHERE n.nspname || '.' || cl.relname = $1
		ORDER BY conname`

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
