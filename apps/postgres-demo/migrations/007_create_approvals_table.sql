-- ============================================================================
-- CREATE APPROVALS TABLE
-- Implements two-person approval system for sensitive operations
-- ============================================================================

-- Create approvals table
CREATE TABLE IF NOT EXISTS approvals (
    id SERIAL PRIMARY KEY,
    approval_type VARCHAR(50) NOT NULL,                    -- Type: 'system_setting', 'wallet_rule', 'transaction', 'wallet_transfer', etc.
    entity_type VARCHAR(50) NOT NULL,                      -- Entity type: 'system_setting', 'wallet_rule', 'wallet_transaction', etc.
    entity_id VARCHAR(255),                                 -- ID of the entity being changed (e.g., setting key, rule ID, transaction ID)
    operation VARCHAR(20) NOT NULL,                         -- Operation: 'create', 'update', 'delete', 'execute'
    original_data JSONB,                                     -- Original data before change (for rollback)
    new_data JSONB NOT NULL,                                 -- New data/change being requested
    status VARCHAR(20) NOT NULL DEFAULT 'pending',          -- Status: 'pending', 'approved', 'rejected', 'expired'
    requested_by VARCHAR(255) NOT NULL,                     -- User who requested the change
    approved_by_first VARCHAR(255),                          -- First approver (first person to approve)
    approved_by_second VARCHAR(255),                         -- Second approver (second person to approve - required)
    rejected_by VARCHAR(255),                                 -- User who rejected (if rejected)
    rejection_reason TEXT,                                   -- Reason for rejection
    expires_at TIMESTAMP,                                     -- Expiration time for approval request
    metadata JSONB,                                          -- Additional metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    approved_at TIMESTAMP,                                   -- When both approvals were completed
    
    -- Constraints
    CHECK (approval_type IN ('system_setting', 'wallet_rule', 'transaction', 'wallet_transfer', 'large_transaction', 'system_config', 'custom')),
    CHECK (operation IN ('create', 'update', 'delete', 'execute', 'transfer')),
    CHECK (status IN ('pending', 'approved', 'rejected', 'expired', 'cancelled')),
    CHECK (approved_by_first IS NULL OR approved_by_first != requested_by),
    CHECK (approved_by_second IS NULL OR approved_by_second != requested_by),
    CHECK (approved_by_second IS NULL OR approved_by_first IS NOT NULL) -- Second approval requires first
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_approvals_type ON approvals(approval_type);
CREATE INDEX IF NOT EXISTS idx_approvals_status ON approvals(status);
CREATE INDEX IF NOT EXISTS idx_approvals_requested_by ON approvals(requested_by);
CREATE INDEX IF NOT EXISTS idx_approvals_entity ON approvals(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_approvals_expires ON approvals(expires_at) WHERE status = 'pending';

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE approvals IS 'Two-person approval system for sensitive operations';
COMMENT ON COLUMN approvals.approval_type IS 'Type of approval: system_setting, wallet_rule, transaction, etc.';
COMMENT ON COLUMN approvals.entity_type IS 'Type of entity being changed';
COMMENT ON COLUMN approvals.entity_id IS 'ID of the entity (e.g., setting key, rule ID)';
COMMENT ON COLUMN approvals.operation IS 'Operation: create, update, delete, execute';
COMMENT ON COLUMN approvals.original_data IS 'Original data before change (JSON)';
COMMENT ON COLUMN approvals.new_data IS 'New data/change being requested (JSON)';
COMMENT ON COLUMN approvals.status IS 'Status: pending (waiting for approvals), approved (both approved), rejected, expired';
COMMENT ON COLUMN approvals.requested_by IS 'User who requested the change';
COMMENT ON COLUMN approvals.approved_by_first IS 'First person to approve';
COMMENT ON COLUMN approvals.approved_by_second IS 'Second person to approve (required for completion)';
COMMENT ON COLUMN approvals.expires_at IS 'Expiration time - pending approvals expire after this time';
