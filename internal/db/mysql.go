// internal/db/mysql.go
package db

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"time"

	"github.com/go-sql-driver/mysql"
)

// MySQLDriver implements Driver for MySQL
type MySQLDriver struct {
	db      *sql.DB
	tunnel  *SSHTunnel
	netName string // Registered network name for SSH
}

// Connect establishes connection to MySQL
func (d *MySQLDriver) Connect(params ConnectParams) error {
	protocol := "tcp"
	address := fmt.Sprintf("%s:%d", params.Host, params.Port)

	// Setup SSH tunnel if configured
	if params.SSHConfig != nil && params.SSHConfig.Host != "" {
		tunnel, err := NewSSHTunnel(params.SSHConfig)
		if err != nil {
			return WrapConnectionError(fmt.Errorf("failed to create SSH tunnel: %w", err))
		}
		d.tunnel = tunnel

		// Register a unique network for this connection
		// Use a simple random suffix to avoid collisions
		d.netName = fmt.Sprintf("mysql+ssh+%d", time.Now().UnixNano())
		mysql.RegisterDialContext(d.netName, func(ctx context.Context, addr string) (net.Conn, error) {
			return tunnel.Dial("tcp", addr)
		})
		protocol = d.netName
	}

	// Build DSN: user:password@protocol(address)/dbname?param=value
	dsn := fmt.Sprintf("%s:%s@%s(%s)/%s",
		params.User,
		params.Password,
		protocol,
		address,
		params.Database,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		d.Close() // Cleanup tunnel if open failed
		return WrapConnectionError(err)
	}

	// Configure connection pooling
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection immediately (sql.Open is lazy)
	if err := db.Ping(); err != nil {
		db.Close()
		d.Close() // Cleanup tunnel
		return WrapConnectionError(err)
	}

	d.db = db
	return nil
}

// Close closes the database connection and SSH tunnel
func (d *MySQLDriver) Close() error {
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
	// Note: We can't unregister the dial context in the driver, but it's lightweight
	return dbErr
}

// Execute runs a query and returns results
func (d *MySQLDriver) Execute(ctx context.Context, query string) (*QueryResult, error) {
	return executeQuery(ctx, d.db, query)
}

// Ping checks if database is reachable
func (d *MySQLDriver) Ping(ctx context.Context) error {
	if d.db == nil {
		return WrapConnectionError(fmt.Errorf("not connected"))
	}
	return d.db.PingContext(ctx)
}

// Type returns the driver type
func (d *MySQLDriver) Type() DriverType {
	return MySQL
}

// GetTables returns a list of tables in the current database
func (d *MySQLDriver) GetTables(ctx context.Context) ([]string, error) {
	query := "SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE()"
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
func (d *MySQLDriver) GetColumns(ctx context.Context, tableName string) ([]Column, error) {
	query := `
		SELECT 
			COLUMN_NAME, 
			COLUMN_TYPE, 
			IS_NULLABLE = 'YES', 
			IFNULL(COLUMN_DEFAULT, ''),
			COLUMN_KEY 
		FROM INFORMATION_SCHEMA.COLUMNS 
		WHERE TABLE_NAME = ? AND TABLE_SCHEMA = DATABASE()
		ORDER BY ORDINAL_POSITION`

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
func (d *MySQLDriver) GetConstraints(ctx context.Context, tableName string) ([]Constraint, error) {
	query := `
		SELECT 
			CONSTRAINT_NAME, 
			CONSTRAINT_TYPE, 
			'' as DEFINITION
		FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS
		WHERE TABLE_NAME = ? AND TABLE_SCHEMA = DATABASE()`

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
