package goorm

import (
	"context"
	"fmt"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
	"github.com/fluxorio/fluxor/pkg/persistence"
	"gorm.io/gorm"
)

// GORMRepository provides GORM integration for the persistence package
// Wraps GORM's DB instance and implements the persistence.Repository interface
type GORMRepository struct {
	db     *gorm.DB
	config Config
}

// Config configures a GORM repository
type Config struct {
	// DB is the GORM database instance
	DB *gorm.DB

	// TableName is the database table name (optional, can be inferred from model)
	TableName string

	// Model is the GORM model type (used for type inference)
	Model interface{}

	// EnableSoftDelete enables soft delete support
	EnableSoftDelete bool
}

// Validate validates the configuration
// Fail-fast: Returns error if configuration is invalid
func (c *Config) Validate() error {
	if c.DB == nil {
		return &persistence.Error{
			Code:    persistence.ErrCodeInvalidConfig,
			Message: "DB cannot be nil",
		}
	}
	return nil
}

// NewGORMRepository creates a new GORM repository
// Fail-fast: Validates configuration before creating repository
func NewGORMRepository(config Config) (*GORMRepository, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &GORMRepository{
		db:     config.DB,
		config: config,
	}, nil
}

// FindByID retrieves an entity by its ID
func (r *GORMRepository) FindByID(ctx context.Context, id interface{}) (interface{}, error) {
	failfast.NotNil(ctx, "context")
	failfast.NotNil(id, "id")

	var result interface{}
	if r.config.Model != nil {
		// Use model type for type-safe query
		result = r.config.Model
	} else {
		// Fallback to map if no model specified
		result = make(map[string]interface{})
	}

	err := r.db.WithContext(ctx).First(result, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &persistence.Error{
				Code:    persistence.ErrCodeNotFound,
				Message: fmt.Sprintf("entity with id %v not found", id),
				Cause:   err,
			}
		}
		return nil, &persistence.Error{
			Code:    persistence.ErrCodeQuery,
			Message: "failed to query entity",
			Cause:   err,
		}
	}

	return result, nil
}

// FindAll retrieves all entities matching the query
func (r *GORMRepository) FindAll(ctx context.Context, query *persistence.Query) ([]interface{}, error) {
	failfast.NotNil(ctx, "context")
	if query == nil {
		query = persistence.NewQuery()
	}

	var results []interface{}
	db := r.db.WithContext(ctx)

	// Apply filters
	for field, value := range query.Filters {
		db = db.Where(fmt.Sprintf("%s = ?", field), value)
	}

	// Apply ordering
	for field, direction := range query.OrderBy {
		db = db.Order(fmt.Sprintf("%s %s", field, direction))
	}

	// Apply limit
	if query.Limit > 0 {
		db = db.Limit(query.Limit)
	}

	// Apply offset
	if query.Offset > 0 {
		db = db.Offset(query.Offset)
	}

	// Select specific fields if specified
	if len(query.SelectFields) > 0 {
		db = db.Select(query.SelectFields)
	}

	// Execute query
	var model interface{}
	if r.config.Model != nil {
		model = r.config.Model
	} else {
		model = make(map[string]interface{})
	}

	err := db.Find(&results, model).Error
	if err != nil {
		return nil, &persistence.Error{
			Code:    persistence.ErrCodeQuery,
			Message: "failed to query entities",
			Cause:   err,
		}
	}

	return results, nil
}

// FindOne retrieves a single entity matching the query
func (r *GORMRepository) FindOne(ctx context.Context, query *persistence.Query) (interface{}, error) {
	failfast.NotNil(ctx, "context")
	if query == nil {
		query = persistence.NewQuery()
	}

	// Limit to 1 for FindOne
	query.Limit = 1

	results, err := r.FindAll(ctx, query)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, &persistence.Error{
			Code:    persistence.ErrCodeNotFound,
			Message: "no entity found matching query",
		}
	}

	return results[0], nil
}

// Create persists a new entity
func (r *GORMRepository) Create(ctx context.Context, entity interface{}) error {
	failfast.NotNil(ctx, "context")
	failfast.NotNil(entity, "entity")

	err := r.db.WithContext(ctx).Create(entity).Error
	if err != nil {
		return &persistence.Error{
			Code:    persistence.ErrCodeQuery,
			Message: "failed to create entity",
			Cause:   err,
		}
	}

	return nil
}

// Update updates an existing entity
func (r *GORMRepository) Update(ctx context.Context, id interface{}, entity interface{}) error {
	failfast.NotNil(ctx, "context")
	failfast.NotNil(id, "id")
	failfast.NotNil(entity, "entity")

	result := r.db.WithContext(ctx).Model(entity).Where("id = ?", id).Updates(entity)
	if result.Error != nil {
		return &persistence.Error{
			Code:    persistence.ErrCodeQuery,
			Message: "failed to update entity",
			Cause:   result.Error,
		}
	}

	if result.RowsAffected == 0 {
		return &persistence.Error{
			Code:    persistence.ErrCodeNotFound,
			Message: fmt.Sprintf("entity with id %v not found", id),
		}
	}

	return nil
}

// Delete removes an entity by ID
func (r *GORMRepository) Delete(ctx context.Context, id interface{}) error {
	failfast.NotNil(ctx, "context")
	failfast.NotNil(id, "id")

	var model interface{}
	if r.config.Model != nil {
		model = r.config.Model
	} else {
		return &persistence.Error{
			Code:    persistence.ErrCodeInvalidInput,
			Message: "Model must be specified for Delete operation",
		}
	}

	var result *gorm.DB
	if r.config.EnableSoftDelete {
		// Soft delete
		result = r.db.WithContext(ctx).Delete(model, id)
	} else {
		// Hard delete
		result = r.db.WithContext(ctx).Unscoped().Delete(model, id)
	}

	if result.Error != nil {
		return &persistence.Error{
			Code:    persistence.ErrCodeQuery,
			Message: "failed to delete entity",
			Cause:   result.Error,
		}
	}

	if result.RowsAffected == 0 {
		return &persistence.Error{
			Code:    persistence.ErrCodeNotFound,
			Message: fmt.Sprintf("entity with id %v not found", id),
		}
	}

	return nil
}

// Count returns the number of entities matching the query
func (r *GORMRepository) Count(ctx context.Context, query *persistence.Query) (int64, error) {
	failfast.NotNil(ctx, "context")
	if query == nil {
		query = persistence.NewQuery()
	}

	var model interface{}
	if r.config.Model != nil {
		model = r.config.Model
	} else {
		model = make(map[string]interface{})
	}

	var count int64
	db := r.db.WithContext(ctx).Model(model)

	// Apply filters
	for field, value := range query.Filters {
		db = db.Where(fmt.Sprintf("%s = ?", field), value)
	}

	err := db.Count(&count).Error
	if err != nil {
		return 0, &persistence.Error{
			Code:    persistence.ErrCodeQuery,
			Message: "failed to count entities",
			Cause:   err,
		}
	}

	return count, nil
}

// Exists checks if an entity with the given ID exists
func (r *GORMRepository) Exists(ctx context.Context, id interface{}) (bool, error) {
	failfast.NotNil(ctx, "context")
	failfast.NotNil(id, "id")

	var model interface{}
	if r.config.Model != nil {
		model = r.config.Model
	} else {
		return false, &persistence.Error{
			Code:    persistence.ErrCodeInvalidInput,
			Message: "Model must be specified for Exists operation",
		}
	}

	var count int64
	err := r.db.WithContext(ctx).Model(model).Where("id = ?", id).Count(&count).Error
	if err != nil {
		return false, &persistence.Error{
			Code:    persistence.ErrCodeQuery,
			Message: "failed to check existence",
			Cause:   err,
		}
	}

	return count > 0, nil
}

// BatchCreate persists multiple entities in a single operation
func (r *GORMRepository) BatchCreate(ctx context.Context, entities []interface{}) error {
	failfast.NotNil(ctx, "context")
	if entities == nil {
		return &persistence.Error{
			Code:    persistence.ErrCodeInvalidInput,
			Message: "entities cannot be nil",
		}
	}
	if len(entities) == 0 {
		return &persistence.Error{
			Code:    persistence.ErrCodeInvalidInput,
			Message: "entities cannot be empty",
		}
	}

	err := r.db.WithContext(ctx).CreateInBatches(entities, 100).Error
	if err != nil {
		return &persistence.Error{
			Code:    persistence.ErrCodeQuery,
			Message: "failed to batch create entities",
			Cause:   err,
		}
	}

	return nil
}

// BatchUpdate updates multiple entities in a single operation
func (r *GORMRepository) BatchUpdate(ctx context.Context, entities []interface{}) error {
	failfast.NotNil(ctx, "context")
	if entities == nil {
		return &persistence.Error{
			Code:    persistence.ErrCodeInvalidInput,
			Message: "entities cannot be nil",
		}
	}
	if len(entities) == 0 {
		return &persistence.Error{
			Code:    persistence.ErrCodeInvalidInput,
			Message: "entities cannot be empty",
		}
	}

	// Use transaction for batch update
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return &persistence.Error{
			Code:    persistence.ErrCodeTransaction,
			Message: "failed to begin transaction for batch update",
			Cause:   tx.Error,
		}
	}
	defer tx.Rollback()

	// Update each entity within transaction
	for _, entity := range entities {
		result := tx.Save(entity)
		if result.Error != nil {
			return &persistence.Error{
				Code:    persistence.ErrCodeQuery,
				Message: "failed to batch update entities",
				Cause:   result.Error,
			}
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return &persistence.Error{
			Code:    persistence.ErrCodeTransaction,
			Message: "failed to commit batch update transaction",
			Cause:   err,
		}
	}

	return nil
}

// BatchDelete removes multiple entities by IDs in a single operation
func (r *GORMRepository) BatchDelete(ctx context.Context, ids []interface{}) error {
	failfast.NotNil(ctx, "context")
	if ids == nil {
		return &persistence.Error{
			Code:    persistence.ErrCodeInvalidInput,
			Message: "ids cannot be nil",
		}
	}
	if len(ids) == 0 {
		return &persistence.Error{
			Code:    persistence.ErrCodeInvalidInput,
			Message: "ids cannot be empty",
		}
	}

	var model interface{}
	if r.config.Model != nil {
		model = r.config.Model
	} else {
		return &persistence.Error{
			Code:    persistence.ErrCodeInvalidInput,
			Message: "Model must be specified for BatchDelete operation",
		}
	}

	var result *gorm.DB
	if r.config.EnableSoftDelete {
		result = r.db.WithContext(ctx).Delete(model, ids)
	} else {
		result = r.db.WithContext(ctx).Unscoped().Delete(model, ids)
	}

	if result.Error != nil {
		return &persistence.Error{
			Code:    persistence.ErrCodeQuery,
			Message: "failed to batch delete entities",
			Cause:   result.Error,
		}
	}

	return nil
}

// BeginTransaction starts a new transaction
func (r *GORMRepository) BeginTransaction(ctx context.Context) (persistence.Transaction, error) {
	failfast.NotNil(ctx, "context")

	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, &persistence.Error{
			Code:    persistence.ErrCodeTransaction,
			Message: "failed to begin transaction",
			Cause:   tx.Error,
		}
	}

	return &GORMTransaction{
		tx:     tx,
		config: r.config,
	}, nil
}

// Close closes the repository
func (r *GORMRepository) Close() error {
	// GORM DB is managed externally, so we don't close it here
	return nil
}

// DB returns the underlying GORM DB instance for advanced operations
func (r *GORMRepository) DB() *gorm.DB {
	return r.db
}

// GORMTransaction is a GORM-based implementation of Transaction
type GORMTransaction struct {
	tx     *gorm.DB
	config Config
}

// Commit commits the transaction
func (t *GORMTransaction) Commit() error {
	if err := t.tx.Commit().Error; err != nil {
		return &persistence.Error{
			Code:    persistence.ErrCodeTransaction,
			Message: "failed to commit transaction",
			Cause:   err,
		}
	}
	return nil
}

// Rollback rolls back the transaction
func (t *GORMTransaction) Rollback() error {
	if err := t.tx.Rollback().Error; err != nil {
		return &persistence.Error{
			Code:    persistence.ErrCodeTransaction,
			Message: "failed to rollback transaction",
			Cause:   err,
		}
	}
	return nil
}

// Savepoint creates a savepoint within the transaction
func (t *GORMTransaction) Savepoint(ctx context.Context, name string) error {
	if name == "" {
		return &persistence.Error{
			Code:    persistence.ErrCodeTransaction,
			Message: "savepoint name cannot be empty",
		}
	}
	
	// Validate savepoint name
	if err := persistence.ValidateSQLIdentifier(name); err != nil {
		return &persistence.Error{
			Code:    persistence.ErrCodeTransaction,
			Message: fmt.Sprintf("invalid savepoint name: %v", err),
		}
	}
	
	query := fmt.Sprintf("SAVEPOINT %s", persistence.QuoteSQLIdentifier(name))
	if err := t.tx.Exec(query).Error; err != nil {
		return &persistence.Error{
			Code:    persistence.ErrCodeTransaction,
			Message: fmt.Sprintf("failed to create savepoint %s", name),
			Cause:   err,
		}
	}
	
	return nil
}

// RollbackToSavepoint rolls back to a previously created savepoint
func (t *GORMTransaction) RollbackToSavepoint(ctx context.Context, name string) error {
	if name == "" {
		return &persistence.Error{
			Code:    persistence.ErrCodeTransaction,
			Message: "savepoint name cannot be empty",
		}
	}
	
	// Validate savepoint name
	if err := persistence.ValidateSQLIdentifier(name); err != nil {
		return &persistence.Error{
			Code:    persistence.ErrCodeTransaction,
			Message: fmt.Sprintf("invalid savepoint name: %v", err),
		}
	}
	
	query := fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", persistence.QuoteSQLIdentifier(name))
	if err := t.tx.Exec(query).Error; err != nil {
		return &persistence.Error{
			Code:    persistence.ErrCodeTransaction,
			Message: fmt.Sprintf("failed to rollback to savepoint %s", name),
			Cause:   err,
		}
	}
	
	return nil
}

// ReleaseSavepoint releases a savepoint
func (t *GORMTransaction) ReleaseSavepoint(ctx context.Context, name string) error {
	if name == "" {
		return &persistence.Error{
			Code:    persistence.ErrCodeTransaction,
			Message: "savepoint name cannot be empty",
		}
	}
	
	// Validate savepoint name
	if err := persistence.ValidateSQLIdentifier(name); err != nil {
		return &persistence.Error{
			Code:    persistence.ErrCodeTransaction,
			Message: fmt.Sprintf("invalid savepoint name: %v", err),
		}
	}
	
	query := fmt.Sprintf("RELEASE SAVEPOINT %s", persistence.QuoteSQLIdentifier(name))
	if err := t.tx.Exec(query).Error; err != nil {
		return &persistence.Error{
			Code:    persistence.ErrCodeTransaction,
			Message: fmt.Sprintf("failed to release savepoint %s", name),
			Cause:   err,
		}
	}
	
	return nil
}

// Repository returns a repository bound to this transaction
func (t *GORMTransaction) Repository() persistence.Repository {
	return &GORMRepository{
		db: t.tx,
		config: Config{
			TableName:        t.config.TableName,
			Model:            t.config.Model,
			EnableSoftDelete: t.config.EnableSoftDelete,
		},
	}
}
