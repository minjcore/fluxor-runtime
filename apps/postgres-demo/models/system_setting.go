package models

import (
	"time"
)

// ============================================================================
// SETTING DATA TYPES
// ============================================================================

// SettingDataType represents the data type of a setting value
type SettingDataType string

const (
	DataTypeString  SettingDataType = "string"
	DataTypeInteger SettingDataType = "integer"
	DataTypeFloat   SettingDataType = "float"
	DataTypeBoolean SettingDataType = "boolean"
	DataTypeJSON    SettingDataType = "json"
	DataTypeArray   SettingDataType = "array"
	DataTypeObject  SettingDataType = "object"
)

// String returns the string representation
func (dt SettingDataType) String() string {
	return string(dt)
}

// IsValid checks if the data type is valid
func (dt SettingDataType) IsValid() bool {
	switch dt {
	case DataTypeString, DataTypeInteger, DataTypeFloat, DataTypeBoolean, DataTypeJSON, DataTypeArray, DataTypeObject:
		return true
	}
	return false
}

// ============================================================================
// SYSTEM SETTING MODEL
// ============================================================================

// SystemSetting represents a system configuration setting
// Table: system_settings
type SystemSetting struct {
	ID            int            `db:"id" json:"id"`
	SettingKey    string         `db:"setting_key" json:"setting_key"`
	SettingValue  string         `db:"setting_value" json:"setting_value"`
	DataType      SettingDataType `db:"data_type" json:"data_type"`
	Category      *string        `db:"category" json:"category,omitempty"`
	Description   string         `db:"description" json:"description"`
	IsEncrypted   bool           `db:"is_encrypted" json:"is_encrypted"`
	IsReadonly    bool           `db:"is_readonly" json:"is_readonly"`
	ValidationRule *string       `db:"validation_rule" json:"validation_rule,omitempty"` // JSON string
	DefaultValue  *string        `db:"default_value" json:"default_value,omitempty"`
	Metadata      *string        `db:"metadata" json:"metadata,omitempty"` // JSON string
	CreatedAt     time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at" json:"updated_at"`
	UpdatedBy     *string        `db:"updated_by" json:"updated_by,omitempty"`
}

// ============================================================================
// SETTING REQUEST/RESPONSE MODELS
// ============================================================================

// CreateSettingRequest represents a request to create a new setting
type CreateSettingRequest struct {
	SettingKey    string  `json:"setting_key" binding:"required"`
	SettingValue  string  `json:"setting_value"`
	DataType      string  `json:"data_type"`
	Category      *string `json:"category,omitempty"`
	Description   string  `json:"description"`
	IsEncrypted   bool    `json:"is_encrypted"`
	IsReadonly    bool    `json:"is_readonly"`
	ValidationRule *string `json:"validation_rule,omitempty"`
	DefaultValue  *string `json:"default_value,omitempty"`
	Metadata      *string `json:"metadata,omitempty"`
}

// UpdateSettingRequest represents a request to update a setting
type UpdateSettingRequest struct {
	SettingValue  *string `json:"setting_value,omitempty"`
	Category      *string `json:"category,omitempty"`
	Description   *string `json:"description,omitempty"`
	IsEncrypted   *bool   `json:"is_encrypted,omitempty"`
	IsReadonly    *bool   `json:"is_readonly,omitempty"`
	ValidationRule *string `json:"validation_rule,omitempty"`
	DefaultValue  *string `json:"default_value,omitempty"`
	Metadata      *string `json:"metadata,omitempty"`
}

// SettingValueResponse represents a setting value response (for public API)
type SettingValueResponse struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
	Type  string      `json:"type"`
}
