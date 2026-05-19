package hibernate

import (
	"github.com/fluxorio/fluxor/pkg/persistence"
)

// Transaction represents a Hibernate transaction (like Hibernate Transaction)
type Transaction interface {
	// Commit commits the transaction
	Commit() error

	// Rollback rolls back the transaction
	Rollback() error

	// Begin begins a nested transaction (if supported)
	Begin() error

	// IsActive checks if transaction is active
	IsActive() bool
}

// hibernateTransaction wraps persistence.Transaction
type hibernateTransaction struct {
	tx persistence.Transaction
}

func (t *hibernateTransaction) Commit() error {
	return t.tx.Commit()
}

func (t *hibernateTransaction) Rollback() error {
	return t.tx.Rollback()
}

func (t *hibernateTransaction) Begin() error {
	// Nested transactions not supported in standard SQL
	return nil
}

func (t *hibernateTransaction) IsActive() bool {
	// Simplified - would need to track state
	return true
}
