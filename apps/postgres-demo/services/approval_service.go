package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fluxorio/fluxor/apps/postgres-demo/models"
	"github.com/fluxorio/fluxor/pkg/dbruntime"
)

// ApprovalService handles two-person approval workflow
type ApprovalService struct {
	db *dbruntime.DB
}

// NewApprovalService creates a new approval service
func NewApprovalService(db *dbruntime.DB) *ApprovalService {
	return &ApprovalService{
		db: db,
	}
}

// ============================================================================
// CREATE APPROVAL
// ============================================================================

// CreateApproval creates a new approval request requiring two-person approval
func (s *ApprovalService) CreateApproval(ctx context.Context, req models.CreateApprovalRequest, requestedBy string) (int, error) {
	// Convert data to JSON strings
	newDataJSON, err := json.Marshal(req.NewData)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal new_data: %w", err)
	}

	var originalDataJSON *string
	if req.OriginalData != nil {
		data, err := json.Marshal(req.OriginalData)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal original_data: %w", err)
		}
		str := string(data)
		originalDataJSON = &str
	}

	var metadataJSON *string
	if req.Metadata != nil {
		data, err := json.Marshal(req.Metadata)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		str := string(data)
		metadataJSON = &str
	}

	// Set default expiration (24 hours from now) if not provided
	expiresAt := req.ExpiresAt
	if expiresAt == nil {
		exp := time.Now().Add(24 * time.Hour)
		expiresAt = &exp
	}

	query := `INSERT INTO approvals (approval_type, entity_type, entity_id, operation, 
		original_data, new_data, status, requested_by, expires_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`
	
	var approvalID int
	err = s.db.QueryRowContext(ctx, query,
		req.ApprovalType, req.EntityType, req.EntityID, req.Operation,
		originalDataJSON, string(newDataJSON), models.ApprovalStatusPending, requestedBy,
		expiresAt, metadataJSON).Scan(&approvalID)
	if err != nil {
		return 0, fmt.Errorf("failed to create approval: %w", err)
	}

	return approvalID, nil
}

// ============================================================================
// APPROVE/REJECT
// ============================================================================

// Approve approves an approval request (first or second approval)
func (s *ApprovalService) Approve(ctx context.Context, approvalID int, approvedBy string) error {
	// Get current approval
	approval, err := s.GetApproval(ctx, approvalID)
	if err != nil {
		return err
	}

	// Check if already approved or rejected
	if approval.Status == models.ApprovalStatusApproved {
		return fmt.Errorf("approval already approved")
	}
	if approval.Status == models.ApprovalStatusRejected {
		return fmt.Errorf("approval already rejected")
	}
	if approval.Status == models.ApprovalStatusExpired {
		return fmt.Errorf("approval has expired")
	}

	// Check if requester is trying to approve their own request
	if approval.RequestedBy == approvedBy {
		return fmt.Errorf("cannot approve your own request")
	}

	// Check if already approved by this user
	if approval.ApprovedByFirst != nil && *approval.ApprovedByFirst == approvedBy {
		return fmt.Errorf("already approved by this user as first approver")
	}
	if approval.ApprovedBySecond != nil && *approval.ApprovedBySecond == approvedBy {
		return fmt.Errorf("already approved by this user as second approver")
	}

	// Use transaction for atomic update
	txOptions := &sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	}

	dbTx, err := s.db.DB.BeginTx(ctx, txOptions)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer dbTx.Rollback()

	// Lock the approval row
	var currentStatus models.ApprovalStatus
	var currentFirst, currentSecond *string
	lockQuery := `SELECT status, approved_by_first, approved_by_second 
		FROM approvals WHERE id = $1 FOR UPDATE`
	err = dbTx.QueryRowContext(ctx, lockQuery, approvalID).Scan(&currentStatus, &currentFirst, &currentSecond)
	if err != nil {
		return fmt.Errorf("failed to lock approval: %w", err)
	}

	// Check status again after lock
	if currentStatus != models.ApprovalStatusPending {
		return fmt.Errorf("approval status changed, cannot approve")
	}

	// Determine if this is first or second approval
	if currentFirst == nil {
		// First approval
		updateQuery := `UPDATE approvals 
			SET approved_by_first = $1, updated_at = CURRENT_TIMESTAMP
			WHERE id = $2`
		_, err = dbTx.ExecContext(ctx, updateQuery, approvedBy, approvalID)
		if err != nil {
			return fmt.Errorf("failed to record first approval: %w", err)
		}
	} else if currentSecond == nil {
		// Second approval - mark as approved
		now := time.Now()
		updateQuery := `UPDATE approvals 
			SET approved_by_second = $1, status = $2, approved_at = $3, updated_at = CURRENT_TIMESTAMP
			WHERE id = $4`
		_, err = dbTx.ExecContext(ctx, updateQuery, approvedBy, models.ApprovalStatusApproved, now, approvalID)
		if err != nil {
			return fmt.Errorf("failed to record second approval: %w", err)
		}
	} else {
		return fmt.Errorf("approval already has two approvers")
	}

	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("failed to commit approval: %w", err)
	}

	return nil
}

// Reject rejects an approval request
func (s *ApprovalService) Reject(ctx context.Context, approvalID int, rejectedBy string, reason string) error {
	// Get current approval
	approval, err := s.GetApproval(ctx, approvalID)
	if err != nil {
		return err
	}

	// Check if already processed
	if approval.Status == models.ApprovalStatusApproved {
		return fmt.Errorf("approval already approved, cannot reject")
	}
	if approval.Status == models.ApprovalStatusRejected {
		return fmt.Errorf("approval already rejected")
	}
	if approval.Status == models.ApprovalStatusExpired {
		return fmt.Errorf("approval has expired")
	}

	// Check if requester is trying to reject their own request
	if approval.RequestedBy == rejectedBy {
		return fmt.Errorf("cannot reject your own request (use cancel instead)")
	}

	query := `UPDATE approvals 
		SET status = $1, rejected_by = $2, rejection_reason = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4 AND status = $5`
	
	result, err := s.db.ExecContext(ctx, query, models.ApprovalStatusRejected, rejectedBy, reason, approvalID, models.ApprovalStatusPending)
	if err != nil {
		return fmt.Errorf("failed to reject approval: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("approval not found or already processed")
	}

	return nil
}

// ============================================================================
// GET APPROVALS
// ============================================================================

// GetApproval retrieves an approval by ID
func (s *ApprovalService) GetApproval(ctx context.Context, approvalID int) (*models.Approval, error) {
	query := `SELECT id, approval_type, entity_type, entity_id, operation, 
		original_data, new_data, status, requested_by, approved_by_first, approved_by_second,
		rejected_by, rejection_reason, expires_at, metadata, created_at, updated_at, approved_at
		FROM approvals WHERE id = $1`
	
	var approval models.Approval
	var entityID, originalData, newData, approvedByFirst, approvedBySecond, rejectedBy, rejectionReason, metadata sql.NullString
	var expiresAt, approvedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, approvalID).Scan(
		&approval.ID, &approval.ApprovalType, &approval.EntityType, &entityID, &approval.Operation,
		&originalData, &newData, &approval.Status, &approval.RequestedBy,
		&approvedByFirst, &approvedBySecond, &rejectedBy, &rejectionReason,
		&expiresAt, &metadata, &approval.CreatedAt, &approval.UpdatedAt, &approvedAt)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("approval not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get approval: %w", err)
	}

	if entityID.Valid {
		approval.EntityID = &entityID.String
	}
	if originalData.Valid {
		approval.OriginalData = &originalData.String
	}
	approval.NewData = newData.String
	if approvedByFirst.Valid {
		approval.ApprovedByFirst = &approvedByFirst.String
	}
	if approvedBySecond.Valid {
		approval.ApprovedBySecond = &approvedBySecond.String
	}
	if rejectedBy.Valid {
		approval.RejectedBy = &rejectedBy.String
	}
	if rejectionReason.Valid {
		approval.RejectionReason = &rejectionReason.String
	}
	if expiresAt.Valid {
		approval.ExpiresAt = &expiresAt.Time
	}
	if metadata.Valid {
		approval.Metadata = &metadata.String
	}
	if approvedAt.Valid {
		approval.ApprovedAt = &approvedAt.Time
	}

	return &approval, nil
}

// GetPendingApprovals retrieves all pending approvals
func (s *ApprovalService) GetPendingApprovals(ctx context.Context, approvalType *models.ApprovalType) ([]models.ApprovalSummary, error) {
	var query string
	var args []interface{}

	if approvalType != nil {
		query = `SELECT id, approval_type, entity_type, entity_id, operation, status, 
			requested_by, approved_by_first, approved_by_second, created_at, approved_at, expires_at
			FROM approvals 
			WHERE status = 'pending' AND approval_type = $1 AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
			ORDER BY created_at DESC`
		args = []interface{}{*approvalType}
	} else {
		query = `SELECT id, approval_type, entity_type, entity_id, operation, status, 
			requested_by, approved_by_first, approved_by_second, created_at, approved_at, expires_at
			FROM approvals 
			WHERE status = 'pending' AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
			ORDER BY created_at DESC`
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query approvals: %w", err)
	}
	defer rows.Close()

	var approvals []models.ApprovalSummary
	for rows.Next() {
		var approval models.ApprovalSummary
		var entityID, approvedByFirst, approvedBySecond sql.NullString
		var expiresAt, approvedAt sql.NullTime

		err := rows.Scan(
			&approval.ID, &approval.ApprovalType, &approval.EntityType, &entityID, &approval.Operation,
			&approval.Status, &approval.RequestedBy, &approvedByFirst, &approvedBySecond,
			&approval.CreatedAt, &approvedAt, &expiresAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan approval: %w", err)
		}

		if entityID.Valid {
			approval.EntityID = &entityID.String
		}
		if approvedByFirst.Valid {
			approval.ApprovedByFirst = &approvedByFirst.String
		}
		if approvedBySecond.Valid {
			approval.ApprovedBySecond = &approvedBySecond.String
		}
		if expiresAt.Valid {
			approval.ExpiresAt = &expiresAt.Time
		}
		if approvedAt.Valid {
			approval.ApprovedAt = &approvedAt.Time
		}

		approvals = append(approvals, approval)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating approvals: %w", err)
	}

	return approvals, nil
}

// ============================================================================
// EXPIRATION HANDLING
// ============================================================================

// ExpirePendingApprovals marks expired pending approvals as expired
func (s *ApprovalService) ExpirePendingApprovals(ctx context.Context) (int, error) {
	query := `UPDATE approvals 
		SET status = 'expired', updated_at = CURRENT_TIMESTAMP
		WHERE status = 'pending' AND expires_at IS NOT NULL AND expires_at <= CURRENT_TIMESTAMP
		RETURNING id`
	
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to expire approvals: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
	}

	return count, nil
}
