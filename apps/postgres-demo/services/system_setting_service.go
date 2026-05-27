package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/fluxorio/fluxor/apps/postgres-demo/models"
	"github.com/fluxorio/fluxor/pkg/dbruntime"
)

// SystemSettingService handles system settings management
type SystemSettingService struct {
	db *dbruntime.DB
}

// NewSystemSettingService creates a new system setting service
func NewSystemSettingService(db *dbruntime.DB) *SystemSettingService {
	return &SystemSettingService{
		db: db,
	}
}

// ============================================================================
// GET SETTINGS
// ============================================================================

// GetSetting retrieves a setting by key
func (s *SystemSettingService) GetSetting(ctx context.Context, key string) (*models.SystemSetting, error) {
	query := `SELECT id, setting_key, setting_value, data_type, category, description, 
		is_encrypted, is_readonly, validation_rule, default_value, metadata, 
		created_at, updated_at, updated_by
		FROM system_settings WHERE setting_key = $1`
	
	var setting models.SystemSetting
	var category, validationRule, defaultValue, metadata, updatedBy sql.NullString

	err := s.db.QueryRowContext(ctx, query, key).Scan(
		&setting.ID, &setting.SettingKey, &setting.SettingValue, &setting.DataType,
		&category, &setting.Description, &setting.IsEncrypted, &setting.IsReadonly,
		&validationRule, &defaultValue, &metadata, &setting.CreatedAt, &setting.UpdatedAt, &updatedBy)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("setting '%s' not found", key)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get setting: %w", err)
	}

	if category.Valid {
		setting.Category = &category.String
	}
	if validationRule.Valid {
		setting.ValidationRule = &validationRule.String
	}
	if defaultValue.Valid {
		setting.DefaultValue = &defaultValue.String
	}
	if metadata.Valid {
		setting.Metadata = &metadata.String
	}
	if updatedBy.Valid {
		setting.UpdatedBy = &updatedBy.String
	}

	return &setting, nil
}

// GetSettingValue retrieves a setting value and parses it according to data type
func (s *SystemSettingService) GetSettingValue(ctx context.Context, key string) (interface{}, error) {
	setting, err := s.GetSetting(ctx, key)
	if err != nil {
		return nil, err
	}

	return s.parseValue(setting.SettingValue, setting.DataType)
}

// GetSettingValueWithDefault retrieves a setting value or returns default if not found
func (s *SystemSettingService) GetSettingValueWithDefault(ctx context.Context, key string, defaultValue interface{}) interface{} {
	setting, err := s.GetSetting(ctx, key)
	if err != nil {
		return defaultValue
	}

	value, err := s.parseValue(setting.SettingValue, setting.DataType)
	if err != nil {
		return defaultValue
	}

	return value
}

// GetStringSetting retrieves a string setting value
func (s *SystemSettingService) GetStringSetting(ctx context.Context, key string, defaultValue string) string {
	value := s.GetSettingValueWithDefault(ctx, key, defaultValue)
	if str, ok := value.(string); ok {
		return str
	}
	return defaultValue
}

// GetIntSetting retrieves an integer setting value
func (s *SystemSettingService) GetIntSetting(ctx context.Context, key string, defaultValue int) int {
	value := s.GetSettingValueWithDefault(ctx, key, defaultValue)
	if i, ok := value.(int); ok {
		return i
	}
	if f, ok := value.(float64); ok {
		return int(f)
	}
	return defaultValue
}

// GetFloatSetting retrieves a float setting value
func (s *SystemSettingService) GetFloatSetting(ctx context.Context, key string, defaultValue float64) float64 {
	value := s.GetSettingValueWithDefault(ctx, key, defaultValue)
	if f, ok := value.(float64); ok {
		return f
	}
	if i, ok := value.(int); ok {
		return float64(i)
	}
	return defaultValue
}

// GetBoolSetting retrieves a boolean setting value
func (s *SystemSettingService) GetBoolSetting(ctx context.Context, key string, defaultValue bool) bool {
	value := s.GetSettingValueWithDefault(ctx, key, defaultValue)
	if b, ok := value.(bool); ok {
		return b
	}
	if str, ok := value.(string); ok {
		b, _ := strconv.ParseBool(str)
		return b
	}
	return defaultValue
}

// GetAllSettings retrieves all settings, optionally filtered by category
func (s *SystemSettingService) GetAllSettings(ctx context.Context, category *string) ([]models.SystemSetting, error) {
	var query string
	var args []interface{}

	if category != nil {
		query = `SELECT id, setting_key, setting_value, data_type, category, description, 
			is_encrypted, is_readonly, validation_rule, default_value, metadata, 
			created_at, updated_at, updated_by
			FROM system_settings WHERE category = $1 ORDER BY category, setting_key`
		args = []interface{}{*category}
	} else {
		query = `SELECT id, setting_key, setting_value, data_type, category, description, 
			is_encrypted, is_readonly, validation_rule, default_value, metadata, 
			created_at, updated_at, updated_by
			FROM system_settings ORDER BY category, setting_key`
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query settings: %w", err)
	}
	defer rows.Close()

	var settings []models.SystemSetting
	for rows.Next() {
		var setting models.SystemSetting
		var category, validationRule, defaultValue, metadata, updatedBy sql.NullString

		err := rows.Scan(
			&setting.ID, &setting.SettingKey, &setting.SettingValue, &setting.DataType,
			&category, &setting.Description, &setting.IsEncrypted, &setting.IsReadonly,
			&validationRule, &defaultValue, &metadata, &setting.CreatedAt, &setting.UpdatedAt, &updatedBy)
		if err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}

		if category.Valid {
			setting.Category = &category.String
		}
		if validationRule.Valid {
			setting.ValidationRule = &validationRule.String
		}
		if defaultValue.Valid {
			setting.DefaultValue = &defaultValue.String
		}
		if metadata.Valid {
			setting.Metadata = &metadata.String
		}
		if updatedBy.Valid {
			setting.UpdatedBy = &updatedBy.String
		}

		settings = append(settings, setting)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating settings: %w", err)
	}

	return settings, nil
}

// GetSettingsByCategory retrieves all settings in a specific category
func (s *SystemSettingService) GetSettingsByCategory(ctx context.Context, category string) ([]models.SystemSetting, error) {
	return s.GetAllSettings(ctx, &category)
}

// ============================================================================
// CREATE/UPDATE/DELETE SETTINGS
// ============================================================================

// CreateSetting creates a new system setting
func (s *SystemSettingService) CreateSetting(ctx context.Context, req models.CreateSettingRequest, updatedBy *string) (int, error) {
	// Validate data type
	dataType := models.SettingDataType(req.DataType)
	if !dataType.IsValid() {
		return 0, fmt.Errorf("invalid data type: %s", req.DataType)
	}

	// Validate value format
	if err := s.validateValue(req.SettingValue, dataType); err != nil {
		return 0, fmt.Errorf("invalid value format: %w", err)
	}

	query := `INSERT INTO system_settings (setting_key, setting_value, data_type, category, 
		description, is_encrypted, is_readonly, validation_rule, default_value, metadata, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`
	
	var settingID int
	err := s.db.QueryRowContext(ctx, query,
		req.SettingKey, req.SettingValue, req.DataType,
		req.Category, req.Description, req.IsEncrypted, req.IsReadonly,
		req.ValidationRule, req.DefaultValue, req.Metadata, updatedBy).Scan(&settingID)
	if err != nil {
		return 0, fmt.Errorf("failed to create setting: %w", err)
	}

	return settingID, nil
}

// UpdateSetting updates an existing system setting
func (s *SystemSettingService) UpdateSetting(ctx context.Context, key string, req models.UpdateSettingRequest, updatedBy *string) error {
	// Get existing setting
	existing, err := s.GetSetting(ctx, key)
	if err != nil {
		return err
	}

	// Check if readonly
	if existing.IsReadonly {
		return fmt.Errorf("setting '%s' is readonly and cannot be modified", key)
	}

	// Build update query dynamically
	var updates []string
	var args []interface{}
	argIndex := 1

	if req.SettingValue != nil {
		// Validate value format
		if err := s.validateValue(*req.SettingValue, existing.DataType); err != nil {
			return fmt.Errorf("invalid value format: %w", err)
		}
		updates = append(updates, fmt.Sprintf("setting_value = $%d", argIndex))
		args = append(args, *req.SettingValue)
		argIndex++
	}

	if req.Category != nil {
		updates = append(updates, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, *req.Category)
		argIndex++
	}

	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, *req.Description)
		argIndex++
	}

	if req.IsEncrypted != nil {
		updates = append(updates, fmt.Sprintf("is_encrypted = $%d", argIndex))
		args = append(args, *req.IsEncrypted)
		argIndex++
	}

	if req.IsReadonly != nil {
		updates = append(updates, fmt.Sprintf("is_readonly = $%d", argIndex))
		args = append(args, *req.IsReadonly)
		argIndex++
	}

	if req.ValidationRule != nil {
		updates = append(updates, fmt.Sprintf("validation_rule = $%d", argIndex))
		args = append(args, *req.ValidationRule)
		argIndex++
	}

	if req.DefaultValue != nil {
		updates = append(updates, fmt.Sprintf("default_value = $%d", argIndex))
		args = append(args, *req.DefaultValue)
		argIndex++
	}

	if req.Metadata != nil {
		updates = append(updates, fmt.Sprintf("metadata = $%d", argIndex))
		args = append(args, *req.Metadata)
		argIndex++
	}

	if updatedBy != nil {
		updates = append(updates, fmt.Sprintf("updated_by = $%d", argIndex))
		args = append(args, *updatedBy)
		argIndex++
	}

	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}

	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")

	query := fmt.Sprintf(`UPDATE system_settings SET %s WHERE setting_key = $%d`,
		strings.Join(updates, ", "), argIndex)
	args = append(args, key)

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update setting: %w", err)
	}

	return nil
}

// DeleteSetting deletes a system setting
func (s *SystemSettingService) DeleteSetting(ctx context.Context, key string) error {
	// Check if readonly
	setting, err := s.GetSetting(ctx, key)
	if err != nil {
		return err
	}

	if setting.IsReadonly {
		return fmt.Errorf("setting '%s' is readonly and cannot be deleted", key)
	}

	query := `DELETE FROM system_settings WHERE setting_key = $1`
	_, err = s.db.ExecContext(ctx, query, key)
	if err != nil {
		return fmt.Errorf("failed to delete setting: %w", err)
	}

	return nil
}

// ============================================================================
// HELPER METHODS
// ============================================================================

// parseValue parses a string value according to data type
func (s *SystemSettingService) parseValue(value string, dataType models.SettingDataType) (interface{}, error) {
	switch dataType {
	case models.DataTypeString:
		return value, nil
	case models.DataTypeInteger:
		i, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("invalid integer: %w", err)
		}
		return i, nil
	case models.DataTypeFloat:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid float: %w", err)
		}
		return f, nil
	case models.DataTypeBoolean:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("invalid boolean: %w", err)
		}
		return b, nil
	case models.DataTypeJSON, models.DataTypeArray, models.DataTypeObject:
		var result interface{}
		if err := json.Unmarshal([]byte(value), &result); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
		return result, nil
	default:
		return value, nil
	}
}

// validateValue validates a value format according to data type
func (s *SystemSettingService) validateValue(value string, dataType models.SettingDataType) error {
	switch dataType {
	case models.DataTypeInteger:
		_, err := strconv.Atoi(value)
		return err
	case models.DataTypeFloat:
		_, err := strconv.ParseFloat(value, 64)
		return err
	case models.DataTypeBoolean:
		_, err := strconv.ParseBool(value)
		return err
	case models.DataTypeJSON, models.DataTypeArray, models.DataTypeObject:
		var result interface{}
		return json.Unmarshal([]byte(value), &result)
	default:
		return nil // String and others are always valid
	}
}
