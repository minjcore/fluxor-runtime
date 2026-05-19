package csql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
	"github.com/fluxorio/fluxor/pkg/persistence"
)

// CSQLRepository provides advanced SQL features and custom query building
// Extends the base persistence.Repository with additional SQL capabilities
type CSQLRepository struct {
	*persistence.SQLRepository
	db *sql.DB
}

// NewCSQLRepository creates a new CSQL repository with advanced SQL features
// Fail-fast: Validates configuration before creating repository
func NewCSQLRepository(config persistence.Config) (*CSQLRepository, error) {
	baseRepo, err := persistence.NewSQLRepository(config)
	if err != nil {
		return nil, err
	}

	return &CSQLRepository{
		SQLRepository: baseRepo,
		db:            config.ConnectionPool,
	}, nil
}

// QueryBuilder provides a fluent interface for building complex SQL queries
type QueryBuilder struct {
	table      string
	selects    []string
	joins      []string
	wheres     []string
	whereArgs  []interface{}
	groupBys   []string
	havings    []string
	havingArgs []interface{}
	orderBys   []string
	limitVal   *int
	offsetVal  *int
}

// NewQueryBuilder creates a new query builder
// Fail-fast: Validates table name to prevent SQL injection
func NewQueryBuilder(table string) *QueryBuilder {
	// Validate table name to prevent SQL injection (fail-fast)
	if err := persistence.ValidateTableName(table); err != nil {
		panic(fmt.Sprintf("fail-fast: invalid table name: %v", err))
	}
	return &QueryBuilder{
		table:      table,
		selects:    []string{"*"},
		joins:      []string{},
		wheres:     []string{},
		whereArgs:  []interface{}{},
		groupBys:   []string{},
		havings:    []string{},
		havingArgs: []interface{}{},
		orderBys:   []string{},
	}
}

// Select specifies columns to select
// Fail-fast: Validates column names to prevent SQL injection
func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder {
	if len(columns) > 0 {
		// Validate all column names (fail-fast)
		if err := persistence.ValidateFieldNames(columns); err != nil {
			panic(fmt.Sprintf("fail-fast: invalid column name: %v", err))
		}
		qb.selects = columns
	}
	return qb
}

// Join adds an INNER JOIN clause
// Fail-fast: Validates table name to prevent SQL injection
// Note: ON clause should use parameterized queries for values, but table/field names are validated
func (qb *QueryBuilder) Join(table, on string) *QueryBuilder {
	// Validate table name (fail-fast)
	if err := persistence.ValidateTableName(table); err != nil {
		panic(fmt.Sprintf("fail-fast: invalid table name: %v", err))
	}
	// Note: ON clause validation is limited - should use parameterized queries for values
	// For now, we validate that it doesn't contain obvious SQL injection patterns
	if strings.Contains(on, ";") || strings.Contains(on, "--") || strings.Contains(on, "/*") {
		panic("fail-fast: ON clause contains invalid characters")
	}
	quotedTable, _ := persistence.ValidateAndQuoteSQLIdentifier(table)
	qb.joins = append(qb.joins, fmt.Sprintf("INNER JOIN %s ON %s", quotedTable, on))
	return qb
}

// LeftJoin adds a LEFT JOIN clause
// Fail-fast: Validates table name to prevent SQL injection
func (qb *QueryBuilder) LeftJoin(table, on string) *QueryBuilder {
	// Validate table name (fail-fast)
	if err := persistence.ValidateTableName(table); err != nil {
		panic(fmt.Sprintf("fail-fast: invalid table name: %v", err))
	}
	// Validate ON clause
	if strings.Contains(on, ";") || strings.Contains(on, "--") || strings.Contains(on, "/*") {
		panic("fail-fast: ON clause contains invalid characters")
	}
	quotedTable, _ := persistence.ValidateAndQuoteSQLIdentifier(table)
	qb.joins = append(qb.joins, fmt.Sprintf("LEFT JOIN %s ON %s", quotedTable, on))
	return qb
}

// RightJoin adds a RIGHT JOIN clause
// Fail-fast: Validates table name to prevent SQL injection
func (qb *QueryBuilder) RightJoin(table, on string) *QueryBuilder {
	// Validate table name (fail-fast)
	if err := persistence.ValidateTableName(table); err != nil {
		panic(fmt.Sprintf("fail-fast: invalid table name: %v", err))
	}
	// Validate ON clause
	if strings.Contains(on, ";") || strings.Contains(on, "--") || strings.Contains(on, "/*") {
		panic("fail-fast: ON clause contains invalid characters")
	}
	quotedTable, _ := persistence.ValidateAndQuoteSQLIdentifier(table)
	qb.joins = append(qb.joins, fmt.Sprintf("RIGHT JOIN %s ON %s", quotedTable, on))
	return qb
}

// Where adds a WHERE condition
func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
	qb.wheres = append(qb.wheres, condition)
	qb.whereArgs = append(qb.whereArgs, args...)
	return qb
}

// WhereEq adds an equality WHERE condition
// Fail-fast: Validates column name to prevent SQL injection
func (qb *QueryBuilder) WhereEq(column string, value interface{}) *QueryBuilder {
	// Validate and quote column name (fail-fast)
	quotedColumn, err := persistence.QuoteFieldName(column)
	if err != nil {
		panic(fmt.Sprintf("fail-fast: invalid column name: %v", err))
	}
	return qb.Where(fmt.Sprintf("%s = ?", quotedColumn), value)
}

// WhereIn adds a WHERE IN condition
// Fail-fast: Validates column name to prevent SQL injection
func (qb *QueryBuilder) WhereIn(column string, values []interface{}) *QueryBuilder {
	// Validate and quote column name (fail-fast)
	quotedColumn, err := persistence.QuoteFieldName(column)
	if err != nil {
		panic(fmt.Sprintf("fail-fast: invalid column name: %v", err))
	}
	placeholders := make([]string, len(values))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	return qb.Where(fmt.Sprintf("%s IN (%s)", quotedColumn, strings.Join(placeholders, ",")), values...)
}

// GroupBy adds a GROUP BY clause
// Fail-fast: Validates column names to prevent SQL injection
func (qb *QueryBuilder) GroupBy(columns ...string) *QueryBuilder {
	// Validate all column names (fail-fast)
	if err := persistence.ValidateFieldNames(columns); err != nil {
		panic(fmt.Sprintf("fail-fast: invalid column name: %v", err))
	}
	qb.groupBys = append(qb.groupBys, columns...)
	return qb
}

// Having adds a HAVING condition
func (qb *QueryBuilder) Having(condition string, args ...interface{}) *QueryBuilder {
	qb.havings = append(qb.havings, condition)
	qb.havingArgs = append(qb.havingArgs, args...)
	return qb
}

// OrderBy adds an ORDER BY clause
// Fail-fast: Validates column name to prevent SQL injection
func (qb *QueryBuilder) OrderBy(column string, direction string) *QueryBuilder {
	// Validate and quote column name (fail-fast)
	quotedColumn, err := persistence.QuoteFieldName(column)
	if err != nil {
		panic(fmt.Sprintf("fail-fast: invalid column name: %v", err))
	}
	if direction != "ASC" && direction != "DESC" {
		direction = "ASC"
	}
	qb.orderBys = append(qb.orderBys, fmt.Sprintf("%s %s", quotedColumn, direction))
	return qb
}

// Limit sets the LIMIT clause
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limitVal = &limit
	return qb
}

// Offset sets the OFFSET clause
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offsetVal = &offset
	return qb
}

// Build constructs the SQL query and returns the query string and arguments
// Fail-fast: Validates all identifiers before building query
func (qb *QueryBuilder) Build() (string, []interface{}) {
	var query strings.Builder
	var args []interface{}

	// Validate table name (fail-fast)
	quotedTable, err := persistence.ValidateAndQuoteSQLIdentifier(qb.table)
	if err != nil {
		panic(fmt.Sprintf("fail-fast: invalid table name: %v", err))
	}

	// SELECT - quote all select fields
	query.WriteString("SELECT ")
	quotedSelects := make([]string, len(qb.selects))
	for i, col := range qb.selects {
		if col == "*" {
			quotedSelects[i] = "*"
		} else {
			quoted, err := persistence.QuoteFieldName(col)
			if err != nil {
				panic(fmt.Sprintf("fail-fast: invalid select field: %v", err))
			}
			quotedSelects[i] = quoted
		}
	}
	query.WriteString(strings.Join(quotedSelects, ", "))

	// FROM
	query.WriteString(" FROM ")
	query.WriteString(quotedTable)

	// JOINs
	for _, join := range qb.joins {
		query.WriteString(" ")
		query.WriteString(join)
	}

	// WHERE
	if len(qb.wheres) > 0 {
		query.WriteString(" WHERE ")
		query.WriteString(strings.Join(qb.wheres, " AND "))
		args = append(args, qb.whereArgs...)
	}

	// GROUP BY - quote all group by fields
	if len(qb.groupBys) > 0 {
		query.WriteString(" GROUP BY ")
		quotedGroupBys := make([]string, len(qb.groupBys))
		for i, col := range qb.groupBys {
			quoted, err := persistence.QuoteFieldName(col)
			if err != nil {
				panic(fmt.Sprintf("fail-fast: invalid group by field: %v", err))
			}
			quotedGroupBys[i] = quoted
		}
		query.WriteString(strings.Join(quotedGroupBys, ", "))
	}

	// HAVING
	if len(qb.havings) > 0 {
		query.WriteString(" HAVING ")
		query.WriteString(strings.Join(qb.havings, " AND "))
		args = append(args, qb.havingArgs...)
	}

	// ORDER BY
	if len(qb.orderBys) > 0 {
		query.WriteString(" ORDER BY ")
		query.WriteString(strings.Join(qb.orderBys, ", "))
	}

	// LIMIT
	if qb.limitVal != nil {
		query.WriteString(fmt.Sprintf(" LIMIT %d", *qb.limitVal))
	}

	// OFFSET
	if qb.offsetVal != nil {
		query.WriteString(fmt.Sprintf(" OFFSET %d", *qb.offsetVal))
	}

	return query.String(), args
}

// Execute executes the built query and returns rows
func (r *CSQLRepository) Execute(ctx context.Context, qb *QueryBuilder) (*sql.Rows, error) {
	failfast.NotNil(ctx, "context")
	failfast.NotNil(qb, "query builder")

	query, args := qb.Build()
	return r.db.QueryContext(ctx, query, args...)
}

// ExecuteOne executes the built query and returns a single row
func (r *CSQLRepository) ExecuteOne(ctx context.Context, qb *QueryBuilder) (*sql.Row, error) {
	failfast.NotNil(ctx, "context")
	failfast.NotNil(qb, "query builder")

	query, args := qb.Build()
	row := r.db.QueryRowContext(ctx, query, args...)
	return row, nil
}

// ExecuteRaw executes a raw SQL query
func (r *CSQLRepository) ExecuteRaw(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	failfast.NotNil(ctx, "context")
	if query == "" {
		return nil, fmt.Errorf("fail-fast: query cannot be empty")
	}

	return r.db.QueryContext(ctx, query, args...)
}

// ExecuteRawOne executes a raw SQL query and returns a single row
func (r *CSQLRepository) ExecuteRawOne(ctx context.Context, query string, args ...interface{}) *sql.Row {
	failfast.NotNil(ctx, "context")
	if query == "" {
		panic("fail-fast: query cannot be empty")
	}

	return r.db.QueryRowContext(ctx, query, args...)
}

// Exec executes a raw SQL statement (INSERT, UPDATE, DELETE)
func (r *CSQLRepository) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	failfast.NotNil(ctx, "context")
	if query == "" {
		return nil, fmt.Errorf("fail-fast: query cannot be empty")
	}

	return r.db.ExecContext(ctx, query, args...)
}
