package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// SQLRepository is a SQL-based implementation of Repository
type SQLRepository struct {
	config Config
}

// NewSQLRepository creates a new SQL repository
// Fail-fast: Validates configuration before creating repository
func NewSQLRepository(config Config) (*SQLRepository, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &SQLRepository{
		config: config,
	}, nil
}

// FindByID retrieves an entity by its ID
func (r *SQLRepository) FindByID(ctx context.Context, id interface{}) (interface{}, error) {
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

	// Query for a single row with retry logic
	var rows *sql.Rows
	err = r.retryableOperation(ctx, func(ctx context.Context) error {
		var queryErr error
		rows, queryErr = r.config.ConnectionPool.QueryContext(ctx, query, id)
		return queryErr
	})
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

// FindAll retrieves all entities matching the query
func (r *SQLRepository) FindAll(ctx context.Context, query *Query) ([]interface{}, error) {
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

	// Query with retry logic
	var rows *sql.Rows
	err = r.retryableOperation(ctx, func(ctx context.Context) error {
		var queryErr error
		rows, queryErr = r.config.ConnectionPool.QueryContext(ctx, sqlQuery, args...)
		return queryErr
	})
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

// FindOne retrieves a single entity matching the query
func (r *SQLRepository) FindOne(ctx context.Context, query *Query) (interface{}, error) {
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

// Create persists a new entity
func (r *SQLRepository) Create(ctx context.Context, entity interface{}) error {
	failfast.NotNil(ctx, "context")
	failfast.NotNil(entity, "entity")

	// Note: In a real implementation, you would extract fields from the entity
	// This is a simplified version that assumes entity is a map[string]interface{}
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

	// Execute with retry logic
	err = r.retryableOperation(ctx, func(ctx context.Context) error {
		_, execErr := r.config.ConnectionPool.ExecContext(ctx, sqlQuery, values...)
		return execErr
	})
	if err != nil {
		return NewError(ErrCodeQuery, "failed to create entity", err)
	}

	return nil
}

// Update updates an existing entity
func (r *SQLRepository) Update(ctx context.Context, id interface{}, entity interface{}) error {
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

	// Execute with retry logic
	var result sql.Result
	err = r.retryableOperation(ctx, func(ctx context.Context) error {
		var execErr error
		result, execErr = r.config.ConnectionPool.ExecContext(ctx, sqlQuery, values...)
		return execErr
	})
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

// Delete removes an entity by ID
func (r *SQLRepository) Delete(ctx context.Context, id interface{}) error {
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

	// Execute with retry logic
	var result sql.Result
	err = r.retryableOperation(ctx, func(ctx context.Context) error {
		var execErr error
		result, execErr = r.config.ConnectionPool.ExecContext(ctx, sqlQuery, id)
		return execErr
	})
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

// Count returns the number of entities matching the query
func (r *SQLRepository) Count(ctx context.Context, query *Query) (int64, error) {
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

	// Query with retry logic
	var count int64
	err = r.retryableOperation(ctx, func(ctx context.Context) error {
		scanErr := r.config.ConnectionPool.QueryRowContext(ctx, sqlQuery, args...).Scan(&count)
		return scanErr
	})
	if err != nil {
		return 0, NewError(ErrCodeQuery, "failed to count entities", err)
	}

	return count, nil
}

// Exists checks if an entity with the given ID exists
func (r *SQLRepository) Exists(ctx context.Context, id interface{}) (bool, error) {
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

	// Query with retry logic
	var exists bool
	err = r.retryableOperation(ctx, func(ctx context.Context) error {
		scanErr := r.config.ConnectionPool.QueryRowContext(ctx, sqlQuery, id).Scan(&exists)
		return scanErr
	})
	if err != nil {
		return false, NewError(ErrCodeQuery, "failed to check existence", err)
	}

	return exists, nil
}

// BatchCreate persists multiple entities in a single operation
func (r *SQLRepository) BatchCreate(ctx context.Context, entities []interface{}) error {
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
		// Validate and quote field name
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
			// Get original field name (unquoted)
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

	// Execute with retry logic
	err = r.retryableOperation(ctx, func(ctx context.Context) error {
		_, execErr := r.config.ConnectionPool.ExecContext(ctx, sqlQuery, values...)
		return execErr
	})
	if err != nil {
		return NewError(ErrCodeQuery, "failed to batch create entities", err)
	}

	return nil
}

// BatchUpdate updates multiple entities in a single operation
func (r *SQLRepository) BatchUpdate(ctx context.Context, entities []interface{}) error {
	failfast.NotNil(ctx, "context")
	if entities == nil {
		return NewError(ErrCodeInvalidInput, "entities cannot be nil", nil)
	}
	if len(entities) == 0 {
		return NewError(ErrCodeInvalidInput, "entities cannot be empty", nil)
	}

	// Validate and quote identifiers
	quotedTable, err := ValidateAndQuoteSQLIdentifier(r.config.TableName)
	if err != nil {
		return err
	}
	quotedIDField, err := ValidateAndQuoteSQLIdentifier(r.config.IDField)
	if err != nil {
		return err
	}

	// Use a transaction for batch update to ensure atomicity
	tx, err := r.config.ConnectionPool.BeginTx(ctx, nil)
	if err != nil {
		return NewError(ErrCodeTransaction, "failed to begin transaction for batch update", err)
	}
	defer tx.Rollback()

	// Update each entity individually within transaction
	for _, entity := range entities {
		entityMap, ok := entity.(map[string]interface{})
		if !ok {
			tx.Rollback()
			return NewError(ErrCodeInvalidInput, "all entities must be map[string]interface{}", nil)
		}

		id, exists := entityMap[r.config.IDField]
		if !exists {
			tx.Rollback()
			return NewError(ErrCodeInvalidInput, "entity must have ID field", nil)
		}

		// Build UPDATE query for this entity
		var fields []string
		var values []interface{}
		idx := 1

		for field, value := range entityMap {
			if field == r.config.IDField {
				continue // Skip ID field for update
			}
			quotedField, err := QuoteFieldName(field)
			if err != nil {
				tx.Rollback()
				return err
			}
			fields = append(fields, fmt.Sprintf("%s = $%d", quotedField, idx))
			values = append(values, value)
			idx++
		}

		if len(fields) == 0 {
			continue // Skip entities with no fields to update
		}

		values = append(values, id)
		sqlQuery := fmt.Sprintf(
			"UPDATE %s SET %s WHERE %s = $%d",
			quotedTable,
			strings.Join(fields, ", "),
			quotedIDField,
			idx,
		)

		_, err = tx.ExecContext(ctx, sqlQuery, values...)
		if err != nil {
			tx.Rollback()
			return NewError(ErrCodeQuery, "failed to batch update entities", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return NewError(ErrCodeTransaction, "failed to commit batch update transaction", err)
	}

	return nil
}

// BatchDelete removes multiple entities by IDs in a single operation
func (r *SQLRepository) BatchDelete(ctx context.Context, ids []interface{}) error {
	failfast.NotNil(ctx, "context")
	if ids == nil {
		return NewError(ErrCodeInvalidInput, "ids cannot be nil", nil)
	}
	if len(ids) == 0 {
		return NewError(ErrCodeInvalidInput, "ids cannot be empty", nil)
	}

	// Validate and quote identifiers
	quotedTable, err := ValidateAndQuoteSQLIdentifier(r.config.TableName)
	if err != nil {
		return err
	}
	quotedIDField, err := ValidateAndQuoteSQLIdentifier(r.config.IDField)
	if err != nil {
		return err
	}

	// Build WHERE IN clause
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

	// Execute with retry logic
	err = r.retryableOperation(ctx, func(ctx context.Context) error {
		_, execErr := r.config.ConnectionPool.ExecContext(ctx, sqlQuery, args...)
		return execErr
	})
	if err != nil {
		return NewError(ErrCodeQuery, "failed to batch delete entities", err)
	}

	return nil
}

// BeginTransaction starts a new transaction with default isolation level
func (r *SQLRepository) BeginTransaction(ctx context.Context) (Transaction, error) {
	return r.BeginTransactionWithOptions(ctx, nil)
}

// BeginTransactionWithOptions starts a new transaction with specified options
// This allows configuring isolation level and read-only mode
// Example:
//   opts := &sql.TxOptions{
//       Isolation: sql.LevelSerializable,
//       ReadOnly:  false,
//   }
//   tx, err := repo.BeginTransactionWithOptions(ctx, opts)
func (r *SQLRepository) BeginTransactionWithOptions(ctx context.Context, opts *sql.TxOptions) (Transaction, error) {
	failfast.NotNil(ctx, "context")

	ctx, cancel := context.WithTimeout(ctx, r.config.QueryTimeout)
	defer cancel()

	tx, err := r.config.ConnectionPool.BeginTx(ctx, opts)
	if err != nil {
		return nil, NewError(ErrCodeTransaction, "failed to begin transaction", err)
	}

	return &SQLTransaction{
		tx:     tx,
		config: r.config,
	}, nil
}

// Close closes the repository
func (r *SQLRepository) Close() error {
	// Connection pool is managed externally, so we don't close it here
	return nil
}

// Helper methods

func (r *SQLRepository) buildSelectQuery(query *Query) (string, []interface{}, error) {
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

func (r *SQLRepository) buildCountQuery(query *Query) (string, []interface{}, error) {
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

// joinPlaceholders is a helper to join placeholder strings (e.g., "$1", "$2", "$3")
func (r *SQLRepository) joinPlaceholders(placeholders []string) string {
	return strings.Join(placeholders, ", ")
}

// scanRowToMap scans a row into a map[string]interface{}
func (r *SQLRepository) scanRowToMap(rows *sql.Rows) (map[string]interface{}, error) {
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
