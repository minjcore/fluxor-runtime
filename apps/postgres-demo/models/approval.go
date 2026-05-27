package models

import (
	"time"
)

// ============================================================================
// APPROVAL TYPES
// ============================================================================

// ApprovalType represents the type of approval
type ApprovalType string

const (
	ApprovalTypeSystemSetting ApprovalType = "system_setting"
	ApprovalTypeWalletRule    ApprovalType = "wallet_rule"
	ApprovalTypeTransaction   ApprovalType = "transaction"
	ApprovalTypeWalletTransfer ApprovalType = "wallet_transfer"
	ApprovalTypeLargeTransaction ApprovalType = "large_transaction"
	ApprovalTypeSystemConfig  ApprovalType = "system_config"
	ApprovalTypeCustom        ApprovalType = "custom"
)

// String returns the string representation
func (at ApprovalType) String() string {
	return string(at)
}

// ============================================================================
// APPROVAL OPERATIONS
// ============================================================================

// ApprovalOperation represents the operation type
type ApprovalOperation string

const (
	OperationCreate  ApprovalOperation = "create"
	OperationUpdate  ApprovalOperation = "update"
	OperationDelete  ApprovalOperation = "delete"
	OperationExecute ApprovalOperation = "execute"
	OperationTransfer ApprovalOperation = "transfer"
)

// String returns the string representation
func (ao ApprovalOperation) String() string {
	return string(ao)
}

// ============================================================================
// APPROVAL STATUS
// ============================================================================

// ApprovalStatus represents the status of an approval
type ApprovalStatus string

const (
	ApprovalStatusPending   ApprovalStatus = "pending"
	ApprovalStatusApproved  ApprovalStatus = "approved"
	ApprovalStatusRejected  ApprovalStatus = "rejected"
	ApprovalStatusExpired   ApprovalStatus = "expired"
	ApprovalStatusCancelled ApprovalStatus = "cancelled"
)

// String returns the string representation
func (as ApprovalStatus) String() string {
	return string(as)
}

// ============================================================================
// APPROVAL MODEL
// ============================================================================

// Approval represents a two-person approval request
// Table: approvals
type Approval struct {
	ID              int             `db:"id" json:"id"`
	ApprovalType    ApprovalType    `db:"approval_type" json:"approval_type"`
	EntityType      string          `db:"entity_type" json:"entity_type"`
	EntityID        *string         `db:"entity_id" json:"entity_id,omitempty"`
	Operation       ApprovalOperation `db:"operation" json:"operation"`
	OriginalData    *string         `db:"original_data" json:"original_data,omitempty"` // JSON string
	NewData         string          `db:"new_data" json:"new_data"` // JSON string
	Status          ApprovalStatus  `db:"status" json:"status"`
	RequestedBy     string          `db:"requested_by" json:"requested_by"`
	ApprovedByFirst *string         `db:"approved_by_first" json:"approved_by_first,omitempty"`
	ApprovedBySecond *string        `db:"approved_by_second" json:"approved_by_second,omitempty"`
	RejectedBy      *string         `db:"rejected_by" json:"rejected_by,omitempty"`
	RejectionReason *string         `db:"rejection_reason" json:"rejection_reason,omitempty"`
	ExpiresAt       *time.Time      `db:"expires_at" json:"expires_at,omitempty"`
	Metadata        *string         `db:"metadata" json:"metadata,omitempty"` // JSON string
	CreatedAt       time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at" json:"updated_at"`
	ApprovedAt      *time.Time      `db:"approved_at" json:"approved_at,omitempty"`
}

// ============================================================================
// APPROVAL REQUEST/RESPONSE MODELS
// ============================================================================

// CreateApprovalRequest represents a request to create an approval
type CreateApprovalRequest struct {
	ApprovalType ApprovalType      `json:"approval_type" binding:"required"`
	EntityType   string            `json:"entity_type" binding:"required"`
	EntityID     *string           `json:"entity_id,omitempty"`
	Operation    ApprovalOperation `json:"operation" binding:"required"`
	OriginalData interface{}       `json:"original_data,omitempty"` // Will be converted to JSON
	NewData      interface{}       `json:"new_data" binding:"required"` // Will be converted to JSON
	ExpiresAt    *time.Time        `json:"expires_at,omitempty"` // Optional expiration
	Metadata     interface{}       `json:"metadata,omitempty"` // Will be converted to JSON
}

// ApproveRequest represents a request to approve/reject
type ApproveRequest struct {
	Action         string  `json:"action" binding:"required"` // "approve" or "reject"
	RejectionReason *string `json:"rejection_reason,omitempty"` // Required if action is "reject"
}

// ApprovalSummary represents a summary of approval status
type ApprovalSummary struct {
	ID              int            `json:"id"`
	ApprovalType    string         `json:"approval_type"`
	EntityType      string         `json:"entity_type"`
	EntityID        *string        `json:"entity_id,omitempty"`
	Operation       string         `json:"operation"`
	Status          string         `json:"status"`
	RequestedBy     string         `json:"requested_by"`
	ApprovedByFirst *string        `json:"approved_by_first,omitempty"`
	ApprovedBySecond *string       `json:"approved_by_second,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	ApprovedAt      *time.Time     `json:"approved_at,omitempty"`
	ExpiresAt       *time.Time     `json:"expires_at,omitempty"`
}
