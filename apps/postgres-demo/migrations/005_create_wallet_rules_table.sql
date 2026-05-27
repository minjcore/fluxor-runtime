-- ============================================================================
-- CREATE WALLET RULES TABLE
-- Stores business rules and constraints for wallet operations
-- ============================================================================

-- Create wallet_rules table
CREATE TABLE IF NOT EXISTS wallet_rules (
    id SERIAL PRIMARY KEY,
    rule_name VARCHAR(100) NOT NULL UNIQUE,              -- Unique rule name/identifier
    rule_type VARCHAR(50) NOT NULL,                     -- Type: 'min_balance', 'max_balance', 'daily_limit', 'transfer_limit', 'transaction_limit', etc.
    wallet_type VARCHAR(20),                             -- Apply to specific wallet type (NULL = all types)
    user_id VARCHAR(255),                                -- Apply to specific user (NULL = all users)
    operation_type VARCHAR(50),                          -- Apply to specific operation: 'transfer', 'add_balance', 'freeze', 'purchase', etc. (NULL = all operations)
    min_value DECIMAL(15,2),                             -- Minimum value (for min_balance, min_transfer, etc.)
    max_value DECIMAL(15,2),                             -- Maximum value (for max_balance, max_transfer, daily_limit, etc.)
    period_type VARCHAR(20),                             -- Period for limits: 'daily', 'weekly', 'monthly', 'per_transaction' (NULL = no period)
    is_active BOOLEAN NOT NULL DEFAULT true,              -- Whether this rule is active
    priority INTEGER DEFAULT 100,                        -- Rule priority (lower = higher priority, used when multiple rules apply)
    description TEXT,                                     -- Rule description
    error_message TEXT,                                   -- Custom error message when rule is violated
    metadata JSONB,                                       -- Additional rule configuration (JSON)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CHECK (rule_type IN ('min_balance', 'max_balance', 'daily_limit', 'transfer_limit', 'transaction_limit', 'withdrawal_limit', 'deposit_limit', 'freeze_limit', 'custom')),
    CHECK (period_type IS NULL OR period_type IN ('per_transaction', 'daily', 'weekly', 'monthly', 'yearly')),
    CHECK (operation_type IS NULL OR operation_type IN ('transfer', 'add_balance', 'freeze', 'purchase', 'withdrawal', 'deposit', 'all')),
    -- Note: wallet_type foreign key constraint enforced via foreign key (see below)
    CHECK (priority >= 0 AND priority <= 1000)
);

-- Create indexes for efficient rule lookup
CREATE INDEX IF NOT EXISTS idx_wallet_rules_type ON wallet_rules(rule_type);
CREATE INDEX IF NOT EXISTS idx_wallet_rules_wallet_type ON wallet_rules(wallet_type);
CREATE INDEX IF NOT EXISTS idx_wallet_rules_user_id ON wallet_rules(user_id);
CREATE INDEX IF NOT EXISTS idx_wallet_rules_operation ON wallet_rules(operation_type);
CREATE INDEX IF NOT EXISTS idx_wallet_rules_active ON wallet_rules(is_active);
CREATE INDEX IF NOT EXISTS idx_wallet_rules_priority ON wallet_rules(priority);

-- Add foreign key constraint to wallet_types (if wallet_type is specified)
-- Note: This will only enforce when wallet_type is NOT NULL
-- For NULL values, we use a trigger to validate
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'wallet_rules_wallet_type_fkey' 
        AND conrelid = 'wallet_rules'::regclass
    ) THEN
        -- Add foreign key constraint (only validates non-NULL values)
        ALTER TABLE wallet_rules 
        ADD CONSTRAINT wallet_rules_wallet_type_fkey 
        FOREIGN KEY (wallet_type) 
        REFERENCES wallet_types(code) 
        ON DELETE CASCADE;
    END IF;
END $$;

-- Create function to validate wallet_type when it's not NULL
CREATE OR REPLACE FUNCTION validate_wallet_rules_wallet_type()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.wallet_type IS NOT NULL THEN
        IF NOT EXISTS (SELECT 1 FROM wallet_types WHERE code = NEW.wallet_type) THEN
            RAISE EXCEPTION 'Invalid wallet_type: % does not exist in wallet_types', NEW.wallet_type;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to validate wallet_type
DROP TRIGGER IF EXISTS trigger_validate_wallet_rules_wallet_type ON wallet_rules;
CREATE TRIGGER trigger_validate_wallet_rules_wallet_type
    BEFORE INSERT OR UPDATE ON wallet_rules
    FOR EACH ROW
    WHEN (NEW.wallet_type IS NOT NULL)
    EXECUTE FUNCTION validate_wallet_rules_wallet_type();

-- ============================================================================
-- INSERT DEFAULT WALLET RULES
-- ============================================================================

-- Example rules (can be customized per wallet type or user)
INSERT INTO wallet_rules (rule_name, rule_type, wallet_type, operation_type, min_value, max_value, period_type, is_active, priority, description, error_message, metadata) VALUES
    -- Global rules (apply to all wallets)
    ('global_min_balance', 'min_balance', NULL, 'all', 0.00, NULL, NULL, true, 10, 'Minimum balance for all wallets', 'Balance cannot be negative', '{}'::jsonb),
    ('global_max_transfer', 'transfer_limit', NULL, 'transfer', NULL, 10000.00, 'per_transaction', true, 20, 'Maximum transfer amount per transaction', 'Transfer amount exceeds maximum limit of $10,000', '{}'::jsonb),
    ('global_daily_transfer', 'daily_limit', NULL, 'transfer', NULL, 50000.00, 'daily', true, 30, 'Maximum daily transfer limit', 'Daily transfer limit exceeded', '{}'::jsonb),
    
    -- Savings wallet specific rules
    ('savings_min_balance', 'min_balance', 'savings', 'all', 100.00, NULL, NULL, true, 5, 'Minimum balance for savings wallet', 'Savings wallet must maintain minimum balance of $100', '{}'::jsonb),
    ('savings_max_withdrawal', 'withdrawal_limit', 'savings', 'withdrawal', NULL, 1000.00, 'daily', true, 15, 'Daily withdrawal limit for savings', 'Daily withdrawal limit for savings wallet exceeded', '{}'::jsonb),
    
    -- Investment wallet specific rules
    ('investment_min_balance', 'min_balance', 'investment', 'all', 1000.00, NULL, NULL, true, 5, 'Minimum balance for investment wallet', 'Investment wallet requires minimum balance of $1,000', '{}'::jsonb),
    ('investment_max_transfer', 'transfer_limit', 'investment', 'transfer', NULL, 5000.00, 'per_transaction', true, 10, 'Maximum transfer from investment wallet', 'Transfer from investment wallet exceeds limit', '{}'::jsonb),
    
    -- Escrow wallet specific rules
    ('escrow_no_withdrawal', 'withdrawal_limit', 'escrow', 'withdrawal', NULL, 0.00, NULL, true, 1, 'Escrow wallet cannot withdraw', 'Escrow wallet does not allow withdrawals', '{}'::jsonb),
    ('escrow_max_balance', 'max_balance', 'escrow', 'all', NULL, 100000.00, NULL, true, 5, 'Maximum balance for escrow wallet', 'Escrow wallet balance exceeds maximum limit', '{}'::jsonb),
    
    -- Business wallet specific rules
    ('business_min_transfer', 'transfer_limit', 'business', 'transfer', 10.00, NULL, 'per_transaction', true, 10, 'Minimum transfer amount for business wallet', 'Business wallet transfer must be at least $10', '{}'::jsonb),
    ('business_daily_limit', 'daily_limit', 'business', 'all', NULL, 100000.00, 'daily', true, 20, 'Daily transaction limit for business wallet', 'Daily transaction limit for business wallet exceeded', '{}'::jsonb)
ON CONFLICT (rule_name) DO NOTHING;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE wallet_rules IS 'Business rules and constraints for wallet operations';
COMMENT ON COLUMN wallet_rules.rule_type IS 'Type of rule: min_balance, max_balance, daily_limit, transfer_limit, etc.';
COMMENT ON COLUMN wallet_rules.wallet_type IS 'Apply to specific wallet type (NULL = all types)';
COMMENT ON COLUMN wallet_rules.user_id IS 'Apply to specific user (NULL = all users)';
COMMENT ON COLUMN wallet_rules.operation_type IS 'Apply to specific operation type (NULL = all operations)';
COMMENT ON COLUMN wallet_rules.priority IS 'Rule priority: lower number = higher priority (1-1000)';
COMMENT ON COLUMN wallet_rules.period_type IS 'Period for limit rules: daily, weekly, monthly, per_transaction';
COMMENT ON COLUMN wallet_rules.metadata IS 'Additional rule configuration in JSON format';
