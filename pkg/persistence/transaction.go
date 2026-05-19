package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// SQLTransaction is a SQL-based implementation of Transaction
type SQLTransaction struct {
	tx     *sql.Tx
	config Config
}

// Commit commits the transaction
func (t *SQLTransaction) Commit() error {
	if err := t.tx.Commit(); err != nil {
		return NewError(ErrCodeTransaction, "failed to commit transaction", err)
	}
	return nil
}

// Rollback rolls back the transaction
func (t *SQLTransaction) Rollback() error {
	if err := t.tx.Rollback(); err != nil {
		return NewError(ErrCodeTransaction, "failed to rollback transaction", err)
	}
	return nil
}

// Repository returns a repository bound to this transaction
func (t *SQLTransaction) Repository() Repository {
	return &SQLTransactionRepository{
		tx:     t.tx,
		config: t.config,
	}
}

// SQLTransactionRepository is a repository that operates within a transaction
type SQLTransactionRepository struct {
	tx     *sql.Tx
	config Config
}

// FindByID retrieves an entity by its ID within the transaction
func (r *SQLTransactionRepository) FindByID(ctx context.Context, id interface{}) (interface{}, error) {
	failfast.NotNil(ctx, "context")
	failfast.NotNil(id, "id")

	// Validate and quote identifiers to prevent SQL injection
	quotedTable, err := ValidateAndQuoteSQLIdentifier(r.config.TableName)
	if err != nil {
		return nil, err
	}
	quotedIDField, err := ValidateAndQuoteSQLIdentifier(r.config.IDField)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1", quotedTable, quotedIDField)
	
	ctx, cancel := context.WithTimeout(ctx, r.config.QueryTimeout)
	defer cancel()

	// Query for a single row using transaction
	rows, err := r.tx.QueryContext(ctx, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewError(ErrCodeNotFound, fmt.Sprintf("entity with id %v not found", id), err)
		}
		return nil, NewError(ErrCodeQuery, "failed to query entity", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, NewError(ErrCodeNotFound, fmt.Sprintf("entity with id %v not found", id), nil)
	}

	result, err := r.scanRowToMap(rows)
	if err != nil {
		return nil, NewError(ErrCodeQuery, "failed to scan entity", err)
	}

	return result, nil
}

// FindAll retrieves all entities matching the query within the transaction
func (r *SQLTransactionRepository) FindAll(ctx context.Context, query *Query) ([]interface{}, error) {
	failfast.NotNil(ctx, "context")
	if query == nil {
		query = NewQuery()
	}

	sqlQuery, args, err := r.buildSelectQuery(query)
	if err != nil {
		return nil, err
	}
	
	ctx, cancel := context.WithTimeout(ctx, r.config.QueryTimeout)
	defer cancel()

	rows, err := r.tx.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, NewError(ErrCodeQuery, "failed to query entities", err)
	}
	defer rows.Close()

	var results []interface{}
	for rows.Next() {
		result, err := r.scanRowToMap(rows)
		if err != nil {
			return nil, NewError(ErrCodeQuery, "failed to scan entity", err)
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, NewError(ErrCodeQuery, "error iterating rows", err)
	}

	return results, nil
}

// FindOne retrieves a single entity matching the query within the transaction
func (r *SQLTransactionRepository) FindOne(ctx context.Context, query *Query) (interface{}, error) {
	failfast.NotNil(ctx, "context")
	if query == nil {
		query = NewQuery()
	}

	// Limit to 1 for FindOne
	query.Limit = 1

	results, err := r.FindAll(ctx, query)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, NewError(ErrCodeNotFound, "no entity found matching query", nil)
	}

	return results[0], nil
}

// Create persists a new entity within the transaction
func (r *SQLTransactionRepository) Create(ctx context.Context, entity interface{}) error {
	failfast.NotNil(ctx, "context")
	failfast.NotNil(entity, "entity")

	entityMap, ok := entity.(map[string]interface{})
	if !ok {
		return NewError(ErrCodeInvalidInput, "entity must be a map[string]interface{}", nil)
	}

	// Validate and quote table name
	quotedTable, err := ValidateAndQuoteSQLIdentifier(r.config.TableName)
	if err != nil {
		return err
	}

	var fields []string
	var values []interface{}
	var placeholders []string

	idx := 1
	for field, value := range entityMap {
		if field == r.config.IDField {
			continue // Skip ID field for insert
		}
		// Validate and quote field name to prevent SQL injection
		quotedField, err := QuoteFieldName(field)
		if err != nil {
			return err
		}
		fields = append(fields, quotedField)
		values = append(values, value)
		placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
		idx++
	}

	if len(fields) == 0 {
		return NewError(ErrCodeInvalidInput, "no fields to insert", nil)
	}

	sqlQuery := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		quotedTable,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
	)

	ctx, cancel := context.WithTimeout(ctx, r.config.QueryTimeout)
	defer cancel()

	_, err = r.tx.ExecContext(ctx, sqlQuery, values...)
	if err != nil {
		return NewError(ErrCodeQuery, "failed to create entity", err)
	}

	return nil
}

// Update updates an existing entity within the transaction
func (r *SQLTransactionRepository) Update(ctx context.Context, id interface{}, entity interface{}) error {
	failfast.NotNil(ctx, "context")
	failfast.NotNil(id, "id")
	failfast.NotNil(entity, "entity")

	entityMap, ok := entity.(map[string]interface{})
	if !ok {
		return NewError(ErrCodeInvalidInput, "entity must be a map[string]interface{}", nil)
	}

	// Validate and quote table name and ID field
	quotedTable, err := ValidateAndQuoteSQLIdentifier(r.config.TableName)
	if err != nil {
		return err
	}
	quotedIDField, err := ValidateAndQuoteSQLIdentifier(r.config.IDField)
	if err != nil {
		return err
	}

	var fields []string
	var values []interface{}

	idx := 1
	for field, value := range entityMap {
		if field == r.config.IDField {
			continue // Skip ID field for update
		}
		// Validate and quote field name to prevent SQL injection
		quotedField, err := QuoteFieldName(field)
		if err != nil {
			return err
		}
		fields = append(fields, fmt.Sprintf("%s = $%d", quotedField, idx))
		values = append(values, value)
		idx++
	}

	if len(fields) == 0 {
		return NewError(ErrCodeInvalidInput, "no fields to update", nil)
	}

	values = append(values, id) // Add ID for WHERE clause
	sqlQuery := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s = $%d",
		quotedTable,
		strings.Join(fields, ", "),
		quotedIDField,
		idx,
	)

	ctx, cancel := context.WithTimeout(ctx, r.config.QueryTimeout)
	defer cancel()

	result, err := r.tx.ExecContext(ctx, sqlQuery, values...)
	if err != nil {
		return NewError(ErrCodeQuery, "failed to update entity", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return NewError(ErrCodeQuery, "failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return NewError(ErrCodeNotFound, fmt.Sprintf("entity with id %v not found", id), nil)
	}

	return nil
}

// Delete removes an entity by ID within the transaction
func (r *SQLTransactionRepository) Delete(ctx context.Context, id interface{}) error {
	failfast.NotNil(ctx, "context")
	failfast.NotNil(id, "id")

	// Validate and quote identifiers to prevent SQL injection
	quotedTable, err := ValidateAndQuoteSQLIdentifier(r.config.TableName)
	if err != nil {
		return err
	}
	quotedIDField, err := ValidateAndQuoteSQLIdentifier(r.config.IDField)
	if err != nil {
		return err
	}

	sqlQuery := fmt.Sprintf("DELETE FROM %s WHERE %s = $1", quotedTable, quotedIDField)

	ctx, cancel := context.WithTimeout(ctx, r.config.QueryTimeout)
	defer cancel()

	result, err := r.tx.ExecContext(ctx, sqlQuery, id)
	if err != nil {
		return NewError(ErrCodeQuery, "failed to delete entity", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return NewError(ErrCodeQuery, "failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return NewError(ErrCodeNotFound, fmt.Sprintf("entity with id %v not found", id), nil)
	}

	return nil
}

// Count returns the number of entities matching the query within the transaction
func (r *SQLTransactionRepository) Count(ctx context.Context, query *Query) (int64, error) {
	failfast.NotNil(ctx, "context")
	if query == nil {
		query = NewQuery()
	}

	sqlQuery, args, err := r.buildCountQuery(query)
	if err != nil {
		return 0, err
	}

	ctx, cancel := context.WithTimeout(ctx, r.config.QueryTimeout)
	defer cancel()

	var count int64
	err = r.tx.QueryRowContext(ctx, sqlQuery, args...).Scan(&count)
	if err != nil {
		return 0, NewError(ErrCodeQuery, "failed to count entities", err)
	}

	return count, nil
}

// Exists checks if an entity with the given ID exists within the transaction
func (r *SQLTransactionRepository) Exists(ctx context.Context, id interface{}) (bool, error) {
	failfast.NotNil(ctx, "context")
	failfast.NotNil(id, "id")

	// Validate and quote identifiers to prevent SQL injection
	quotedTable, err := ValidateAndQuoteSQLIdentifier(r.config.TableName)
	if err != nil {
		return false, err
	}
	quotedIDField, err := ValidateAndQuoteSQLIdentifier(r.config.IDField)
	if err != nil {
		return false, err
	}

	sqlQuery := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE %s = $1)", quotedTable, quotedIDField)

	ctx, cancel := context.WithTimeout(ctx, r.config.QueryTimeout)
	defer cancel()

	var exists bool
	err = r.tx.QueryRowContext(ctx, sqlQuery, id).Scan(&exists)
	if err != nil {
		return false, NewError(ErrCodeQuery, "failed to check existence", err)
	}

	return exists, nil
}

// BatchCreate persists multiple entities in a single operation within the transaction
func (r *SQLTransactionRepository) BatchCreate(ctx context.Context, entities []interface{}) error {
	failfast.NotNil(ctx, "context")
	if entities == nil {
		return NewError(ErrCodeInvalidInput, "entities cannot be nil", nil)
	}
	if len(entities) == 0 {
		return NewError(ErrCodeInvalidInput, "entities cannot be empty", nil)
	}

	// Validate and quote table name
	quotedTable, err := ValidateAndQuoteSQLIdentifier(r.config.TableName)
	if err != nil {
		return err
	}

	// Get field names from first entity
	firstEntity, ok := entities[0].(map[string]interface{})
	if !ok {
		return NewError(ErrCodeInvalidInput, "entities must be []map[string]interface{}", nil)
	}

	var fields []string
	for field := range firstEntity {
		if field == r.config.IDField {
			continue // Skip ID field
		}
		quotedField, err := QuoteFieldName(field)
		if err != nil {
			return err
		}
		fields = append(fields, quotedField)
	}

	if len(fields) == 0 {
		return NewError(ErrCodeInvalidInput, "no fields to insert", nil)
	}

	// Build batch INSERT query
	var placeholders []string
	var values []interface{}
	idx := 1

	for _, entity := range entities {
		entityMap, ok := entity.(map[string]interface{})
		if !ok {
			return NewError(ErrCodeInvalidInput, "all entities must be map[string]interface{}", nil)
		}

		rowPlaceholders := make([]string, len(fields))
		for i, field := range fields {
			originalField := strings.Trim(field, `"`)
			value, exists := entityMap[originalField]
			if !exists {
				value = nil
			}
			values = append(values, value)
			rowPlaceholders[i] = fmt.Sprintf("$%d", idx)
			idx++
		}
		placeholders = append(placeholders, fmt.Sprintf("(%s)", strings.Join(rowPlaceholders, ", ")))
	}

	sqlQuery := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s",
		quotedTable,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
	)

	ctx, cancel := context.WithTimeout(ctx, r.config.QueryTimeout)
	defer cancel()

	_, err = r.tx.ExecContext(ctx, sqlQuery, values...)
	if err != nil {
		return NewError(ErrCodeQuery, "failed to batch create entities", err)
	}

	return nil
}

// BatchUpdate updates multiple entities in a single operation within the transaction
func (r *SQLTransactionRepository) BatchUpdate(ctx context.Context, entities []interface{}) error {
	failfast.NotNil(ctx, "context")
	if entities == nil {
		return NewError(ErrCodeInvalidInput, "entities cannot be nil", nil)
	}
	if len(entities) == 0 {
		return NewError(ErrCodeInvalidInput, "entities cannot be empty", nil)
	}

	quotedTable, err := ValidateAndQuoteSQLIdentifier(r.config.TableName)
	if err != nil {
		return err
	}
	quotedIDField, err := ValidateAndQuoteSQLIdentifier(r.config.IDField)
	if err != nil {
		return err
	}

	// Update each entity within transaction
	for _, entity := range entities {
		entityMap, ok := entity.(map[string]interface{})
		if !ok {
			return NewError(ErrCodeInvalidInput, "all entities must be map[string]interface{}", nil)
		}

		id, exists := entityMap[r.config.IDField]
		if !exists {
			return NewError(ErrCodeInvalidInput, "entity must have ID field", nil)
		}

		var fields []string
		var values []interface{}
		idx := 1

		for field, value := range entityMap {
			if field == r.config.IDField {
				continue
			}
			quotedField, err := QuoteFieldName(field)
			if err != nil {
				return err
			}
			fields = append(fields, fmt.Sprintf("%s = $%d", quotedField, idx))
			values = append(values, value)
			idx++
		}

		if len(fields) == 0 {
			continue
		}

		values = append(values, id)
		sqlQuery := fmt.Sprintf(
			"UPDATE %s SET %s WHERE %s = $%d",
			quotedTable,
			strings.Join(fields, ", "),
			quotedIDField,
			idx,
		)

		_, err = r.tx.ExecContext(ctx, sqlQuery, values...)
		if err != nil {
			return NewError(ErrCodeQuery, "failed to batch update entities", err)
		}
	}

	return nil
}

// BatchDelete removes multiple entities by IDs in a single operation within the transaction
func (r *SQLTransactionRepository) BatchDelete(ctx context.Context, ids []interface{}) error {
	failfast.NotNil(ctx, "context")
	if ids == nil {
		return NewError(ErrCodeInvalidInput, "ids cannot be nil", nil)
	}
	if len(ids) == 0 {
		return NewError(ErrCodeInvalidInput, "ids cannot be empty", nil)
	}

	quotedTable, err := ValidateAndQuoteSQLIdentifier(r.config.TableName)
	if err != nil {
		return err
	}
	quotedIDField, err := ValidateAndQuoteSQLIdentifier(r.config.IDField)
	if err != nil {
		return err
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	sqlQuery := fmt.Sprintf(
		"DELETE FROM %s WHERE %s IN (%s)",
		quotedTable,
		quotedIDField,
		strings.Join(placeholders, ", "),
	)

	ctx, cancel := context.WithTimeout(ctx, r.config.QueryTimeout)
	defer cancel()

	_, err = r.tx.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return NewError(ErrCodeQuery, "failed to batch delete entities", err)
	}

	return nil
}

// BeginTransaction starts a new nested transaction (not supported in standard SQL)
func (r *SQLTransactionRepository) BeginTransaction(ctx context.Context) (Transaction, error) {
	return nil, NewError(ErrCodeTransaction, "nested transactions not supported", nil)
}

// Close closes the transaction repository
func (r *SQLTransactionRepository) Close() error {
	// Transaction is managed by the parent transaction
	return nil
}

// Savepoint creates a savepoint within the transaction
// Savepoints allow partial rollback within a transaction (nested transaction-like behavior)
// Example:
//   tx.Savepoint(ctx, "sp1")
//   // ... operations ...
//   tx.RollbackToSavepoint(ctx, "sp1")  // Rollback to savepoint
//   // or
//   tx.ReleaseSavepoint(ctx, "sp1")     // Release savepoint
func (t *SQLTransaction) Savepoint(ctx context.Context, name string) error {
	if name == "" {
		return NewError(ErrCodeTransaction, "savepoint name cannot be empty", nil)
	}
	
	// Validate savepoint name to prevent SQL injection
	// Savepoint names follow identifier rules but may be more permissive
	// For safety, validate similar to SQL identifiers
	if err := ValidateSQLIdentifier(name); err != nil {
		return NewError(ErrCodeTransaction, fmt.Sprintf("invalid savepoint name: %v", err), nil)
	}
	
	query := fmt.Sprintf("SAVEPOINT %s", QuoteSQLIdentifier(name))
	
	ctx, cancel := context.WithTimeout(ctx, t.config.QueryTimeout)
	defer cancel()
	
	_, err := t.tx.ExecContext(ctx, query)
	if err != nil {
		return NewError(ErrCodeTransaction, fmt.Sprintf("failed to create savepoint %s", name), err)
	}
	
	return nil
}

// RollbackToSavepoint rolls back to a previously created savepoint
// The savepoint remains active after rollback and can be rolled back to again
func (t *SQLTransaction) RollbackToSavepoint(ctx context.Context, name string) error {
	if name == "" {
		return NewError(ErrCodeTransaction, "savepoint name cannot be empty", nil)
	}
	
	// Validate savepoint name
	if err := ValidateSQLIdentifier(name); err != nil {
		return NewError(ErrCodeTransaction, fmt.Sprintf("invalid savepoint name: %v", err), nil)
	}
	
	query := fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", QuoteSQLIdentifier(name))
	
	ctx, cancel := context.WithTimeout(ctx, t.config.QueryTimeout)
	defer cancel()
	
	_, err := t.tx.ExecContext(ctx, query)
	if err != nil {
		return NewError(ErrCodeTransaction, fmt.Sprintf("failed to rollback to savepoint %s", name), err)
	}
	
	return nil
}

// ReleaseSavepoint releases a savepoint
// After release, the savepoint cannot be rolled back to
// This is useful when you want to commit the work done after a savepoint
func (t *SQLTransaction) ReleaseSavepoint(ctx context.Context, name string) error {
	if name == "" {
		return NewError(ErrCodeTransaction, "savepoint name cannot be empty", nil)
	}
	
	// Validate savepoint name
	if err := ValidateSQLIdentifier(name); err != nil {
		return NewError(ErrCodeTransaction, fmt.Sprintf("invalid savepoint name: %v", err), nil)
	}
	
	query := fmt.Sprintf("RELEASE SAVEPOINT %s", QuoteSQLIdentifier(name))
	
	ctx, cancel := context.WithTimeout(ctx, t.config.QueryTimeout)
	defer cancel()
	
	_, err := t.tx.ExecContext(ctx, query)
	if err != nil {
		return NewError(ErrCodeTransaction, fmt.Sprintf("failed to release savepoint %s", name), err)
	}
	
	return nil
}

// Helper methods (similar to SQLRepository but using transaction)

func (r *SQLTransactionRepository) buildSelectQuery(query *Query) (string, []interface{}, error) {
	// Use strings.Builder for better performance
	var builder strings.Builder
	
	// Validate and quote table name
	quotedTable, err := ValidateAndQuoteSQLIdentifier(r.config.TableName)
	if err != nil {
		return "", nil, err
	}
	
	// Build SELECT clause
	builder.WriteString("SELECT ")
	if len(query.SelectFields) > 0 {
		// Validate all select fields
		if err := ValidateFieldNames(query.SelectFields); err != nil {
			return "", nil, err
		}
		// Quote all select fields
		quotedFields := make([]string, len(query.SelectFields))
		for i, field := range query.SelectFields {
			quoted, err := QuoteFieldName(field)
			if err != nil {
				return "", nil, err
			}
			quotedFields[i] = quoted
		}
		builder.WriteString(strings.Join(quotedFields, ", "))
	} else {
		builder.WriteString("*")
	}
	
	builder.WriteString(" FROM ")
	builder.WriteString(quotedTable)

	args := []interface{}{}
	idx := 1

	// Add WHERE clause
	if len(query.Filters) > 0 {
		builder.WriteString(" WHERE ")
		conditions := make([]string, 0, len(query.Filters))
		for field, value := range query.Filters {
			// Validate and quote field name to prevent SQL injection
			quotedField, err := QuoteFieldName(field)
			if err != nil {
				return "", nil, err
			}
			conditions = append(conditions, fmt.Sprintf("%s = $%d", quotedField, idx))
			args = append(args, value)
			idx++
		}
		builder.WriteString(strings.Join(conditions, " AND "))
	}

	// Add ORDER BY clause
	if len(query.OrderBy) > 0 {
		builder.WriteString(" ORDER BY ")
		orders := make([]string, 0, len(query.OrderBy))
		for field, direction := range query.OrderBy {
			// Validate direction
			if direction != "ASC" && direction != "DESC" {
				direction = "ASC"
			}
			// Validate and quote field name to prevent SQL injection
			quotedField, err := QuoteFieldName(field)
			if err != nil {
				return "", nil, err
			}
			orders = append(orders, fmt.Sprintf("%s %s", quotedField, direction))
		}
		builder.WriteString(strings.Join(orders, ", "))
	}

	// Add LIMIT
	if query.Limit > 0 {
		builder.WriteString(fmt.Sprintf(" LIMIT %d", query.Limit))
	}

	// Add OFFSET
	if query.Offset > 0 {
		builder.WriteString(fmt.Sprintf(" OFFSET %d", query.Offset))
	}

	return builder.String(), args, nil
}

func (r *SQLTransactionRepository) buildCountQuery(query *Query) (string, []interface{}, error) {
	// Use strings.Builder for better performance
	var builder strings.Builder
	
	// Validate and quote table name
	quotedTable, err := ValidateAndQuoteSQLIdentifier(r.config.TableName)
	if err != nil {
		return "", nil, err
	}

	builder.WriteString("SELECT COUNT(*) FROM ")
	builder.WriteString(quotedTable)

	args := []interface{}{}
	idx := 1

	// Add WHERE clause
	if len(query.Filters) > 0 {
		builder.WriteString(" WHERE ")
		conditions := make([]string, 0, len(query.Filters))
		for field, value := range query.Filters {
			// Validate and quote field name to prevent SQL injection
			quotedField, err := QuoteFieldName(field)
			if err != nil {
				return "", nil, err
			}
			conditions = append(conditions, fmt.Sprintf("%s = $%d", quotedField, idx))
			args = append(args, value)
			idx++
		}
		builder.WriteString(strings.Join(conditions, " AND "))
	}

	return builder.String(), args, nil
}

// scanRowToMap scans a row into a map[string]interface{}
func (r *SQLTransactionRepository) scanRowToMap(rows *sql.Rows) (map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := rows.Scan(valuePtrs...); err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	for i, col := range columns {
		val := values[i]
		if b, ok := val.([]byte); ok {
			result[col] = string(b)
		} else {
			result[col] = val
		}
	}

	return result, nil
}
