// internal/db/errors.go
package db

import "fmt"

// ConnectionError wraps database connection failures
type ConnectionError struct {
	Underlying error
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("connection failed: %v", e.Underlying)
}

// QueryError wraps query execution failures
type QueryError struct {
	Underlying error
}

func (e *QueryError) Error() string {
	return fmt.Sprintf("query failed: %v", e.Underlying)
}

// WrapConnectionError creates a ConnectionError from underlying error
func WrapConnectionError(err error) error {
	return &ConnectionError{Underlying: err}
}

// WrapQueryError creates a QueryError from underlying error
func WrapQueryError(err error) error {
	return &QueryError{Underlying: err}
}
