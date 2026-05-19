package hibernate

import (
	"context"
	"fmt"
	"strings"

	"github.com/fluxorio/fluxor/pkg/persistence"
)

// Query represents an HQL query (like Hibernate Query)
type Query interface {
	// SetParameter sets a query parameter (like query.setParameter())
	SetParameter(name string, value interface{}) Query

	// SetParameterList sets a list parameter (like query.setParameterList())
	SetParameterList(name string, values []interface{}) Query

	// SetFirstResult sets the first result (like query.setFirstResult())
	SetFirstResult(first int) Query

	// SetMaxResults sets max results (like query.setMaxResults())
	SetMaxResults(max int) Query

	// List executes query and returns list (like query.list())
	List(ctx context.Context) ([]interface{}, error)

	// UniqueResult executes query and returns single result (like query.uniqueResult())
	UniqueResult(ctx context.Context) (interface{}, error)

	// ExecuteUpdate executes update/delete query (like query.executeUpdate())
	ExecuteUpdate(ctx context.Context) (int, error)
}

// hqlQuery implements Query interface
type hqlQuery struct {
	session      Session
	hql          string
	parameters   map[string]interface{}
	firstResult  int
	maxResults   int
}

func newHQLQuery(session Session, hql string) Query {
	return &hqlQuery{
		session:    session,
		hql:        hql,
		parameters: make(map[string]interface{}),
		firstResult: 0,
		maxResults: 0,
	}
}

func (q *hqlQuery) SetParameter(name string, value interface{}) Query {
	q.parameters[name] = value
	return q
}

func (q *hqlQuery) SetParameterList(name string, values []interface{}) Query {
	q.parameters[name] = values
	return q
}

func (q *hqlQuery) SetFirstResult(first int) Query {
	q.firstResult = first
	return q
}

func (q *hqlQuery) SetMaxResults(max int) Query {
	q.maxResults = max
	return q
}

func (q *hqlQuery) List(ctx context.Context) ([]interface{}, error) {
	// Get repository from session
	sess := q.session.(*hibernateSession)
	query := persistence.NewQuery().
		WithOffset(q.firstResult)
	
	if q.maxResults > 0 {
		query = query.WithLimit(q.maxResults)
	}

	// Apply filters from parameters
	for name, value := range q.parameters {
		// Simple parameter substitution (in real HQL, this would be more complex)
		if _, ok := value.([]interface{}); ok {
			// Parameter list - would need WHERE IN clause
			continue
		}
		query = query.WithFilter(name, value)
	}

	return sess.repository.FindAll(ctx, query)
}

func (q *hqlQuery) UniqueResult(ctx context.Context) (interface{}, error) {
	results, err := q.List(ctx)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no result found")
	}
	if len(results) > 1 {
		return nil, fmt.Errorf("query returned more than one result")
	}
	return results[0], nil
}

func (q *hqlQuery) ExecuteUpdate(ctx context.Context) (int, error) {
	// For UPDATE/DELETE queries
	// This is simplified - real HQL translation would be more complex
	return 0, fmt.Errorf("HQL UPDATE/DELETE not fully implemented")
}

func (q *hqlQuery) translateHQL() (string, []interface{}) {
	// Simplified HQL to SQL translation
	// Real implementation would need a proper parser
	
	hql := strings.TrimSpace(q.hql)
	hql = strings.ToUpper(hql)
	
	// Basic SELECT translation
	if strings.HasPrefix(hql, "FROM") {
		// Extract entity name
		parts := strings.Fields(hql)
		if len(parts) >= 2 {
			entityName := parts[1]
			sql := fmt.Sprintf("SELECT * FROM %s", entityName)
			return sql, nil
		}
	}
	
	// Basic WHERE clause handling would go here
	return q.hql, nil
}

// Criteria represents a Criteria API query (like Hibernate Criteria)
type Criteria interface {
	// Add adds a restriction (like criteria.add())
	Add(criterion Criterion) Criteria

	// AddOrder adds an ordering (like criteria.addOrder())
	AddOrder(order Order) Criteria

	// SetFirstResult sets first result
	SetFirstResult(first int) Criteria

	// SetMaxResults sets max results
	SetMaxResults(max int) Criteria

	// List executes and returns list
	List(ctx context.Context) ([]interface{}, error)

	// UniqueResult executes and returns single result
	UniqueResult(ctx context.Context) (interface{}, error)
}

// Criterion represents a query criterion
type Criterion interface {
	// ToSQL converts criterion to SQL
	ToSQL() (string, []interface{})
}

// Order represents an ordering
type Order interface {
	// ToSQL converts order to SQL
	ToSQL() string
}

// criteria implements Criteria interface
type criteria struct {
	session     Session
	entityType  interface{}
	restrictions []Criterion
	orders      []Order
	firstResult int
	maxResults  int
}

func newCriteria(session Session, entityType interface{}) Criteria {
	return &criteria{
		session:      session,
		entityType:   entityType,
		restrictions: make([]Criterion, 0),
		orders:       make([]Order, 0),
		firstResult:  0,
		maxResults:   0,
	}
}

func (c *criteria) Add(criterion Criterion) Criteria {
	c.restrictions = append(c.restrictions, criterion)
	return c
}

func (c *criteria) AddOrder(order Order) Criteria {
	c.orders = append(c.orders, order)
	return c
}

func (c *criteria) SetFirstResult(first int) Criteria {
	c.firstResult = first
	return c
}

func (c *criteria) SetMaxResults(max int) Criteria {
	c.maxResults = max
	return c
}

func (c *criteria) List(ctx context.Context) ([]interface{}, error) {
	// Build query from criteria
	query := persistence.NewQuery().
		WithOffset(c.firstResult)
	
	if c.maxResults > 0 {
		query = query.WithLimit(c.maxResults)
	}

	// Apply restrictions
	for _, restriction := range c.restrictions {
		sql, args := restriction.ToSQL()
		if len(args) > 0 {
			// Simple implementation: use first arg as filter value
			// Real implementation would parse SQL and extract field name
			_ = sql
			if fieldName := extractFieldName(sql); fieldName != "" {
				query = query.WithFilter(fieldName, args[0])
			}
		}
	}

	// Apply orders
	for _, order := range c.orders {
		sql := order.ToSQL()
		parts := strings.Fields(sql)
		if len(parts) >= 2 {
			field := parts[0]
			direction := parts[1]
			query = query.WithOrderBy(field, direction)
		}
	}

	sess := c.session.(*hibernateSession)
	return sess.repository.FindAll(ctx, query)
}

func extractFieldName(sql string) string {
	// Simple extraction - real implementation would be more robust
	parts := strings.Fields(sql)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func (c *criteria) UniqueResult(ctx context.Context) (interface{}, error) {
	results, err := c.List(ctx)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no result found")
	}
	if len(results) > 1 {
		return nil, fmt.Errorf("query returned more than one result")
	}
	return results[0], nil
}

// Simple criterion implementations

// Eq creates an equality criterion
func Eq(property string, value interface{}) Criterion {
	return &eqCriterion{property: property, value: value}
}

type eqCriterion struct {
	property string
	value    interface{}
}

func (c *eqCriterion) ToSQL() (string, []interface{}) {
	return fmt.Sprintf("%s = ?", c.property), []interface{}{c.value}
}

// Like creates a LIKE criterion
func Like(property string, value string) Criterion {
	return &likeCriterion{property: property, value: value}
}

type likeCriterion struct {
	property string
	value    string
}

func (c *likeCriterion) ToSQL() (string, []interface{}) {
	return fmt.Sprintf("%s LIKE ?", c.property), []interface{}{c.value}
}

// Asc creates an ascending order
func Asc(property string) Order {
	return &ascOrder{property: property}
}

type ascOrder struct {
	property string
}

func (o *ascOrder) ToSQL() string {
	return fmt.Sprintf("%s ASC", o.property)
}

// Desc creates a descending order
func Desc(property string) Order {
	return &descOrder{property: property}
}

type descOrder struct {
	property string
}

func (o *descOrder) ToSQL() string {
	return fmt.Sprintf("%s DESC", o.property)
}
