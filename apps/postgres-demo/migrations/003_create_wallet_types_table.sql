-- ============================================================================
-- CREATE WALLET TYPES TABLE
-- Stores wallet type definitions and configurations
-- ============================================================================

-- Create wallet_types table
CREATE TABLE IF NOT EXISTS wallet_types (
    code VARCHAR(20) PRIMARY KEY,              -- Wallet type code (e.g., 'primary', 'savings')
    name VARCHAR(100) NOT NULL,                -- Display name
    description TEXT,                           -- Description of the wallet type
    icon VARCHAR(255),                          -- Icon name/URL (optional)
    is_active BOOLEAN DEFAULT true,             -- Whether this wallet type is enabled
    is_default BOOLEAN DEFAULT false,           -- Whether this is the default wallet type
    min_balance DECIMAL(15,2),                  -- Minimum balance required (optional)
    max_balance DECIMAL(15,2),                  -- Maximum balance allowed (optional)
    metadata JSONB,                             -- Additional JSON metadata (optional)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CHECK (code ~ '^[a-z0-9_]+$')             -- Only lowercase letters, numbers, underscore
    -- Note: is_default constraint enforced via trigger (see below)
);

-- Create index for active wallet types
CREATE INDEX IF NOT EXISTS idx_wallet_types_active ON wallet_types(is_active);
CREATE INDEX IF NOT EXISTS idx_wallet_types_default ON wallet_types(is_default);

-- Create function to enforce only one default wallet type
CREATE OR REPLACE FUNCTION enforce_single_default_wallet_type()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_default = true THEN
        -- If setting this to default, unset all other defaults
        UPDATE wallet_types 
        SET is_default = false 
        WHERE code != NEW.code AND is_default = true;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to enforce single default
DROP TRIGGER IF EXISTS trigger_enforce_single_default_wallet_type ON wallet_types;
CREATE TRIGGER trigger_enforce_single_default_wallet_type
    BEFORE INSERT OR UPDATE ON wallet_types
    FOR EACH ROW
    WHEN (NEW.is_default = true)
    EXECUTE FUNCTION enforce_single_default_wallet_type();

-- ============================================================================
-- INSERT DEFAULT WALLET TYPES
-- ============================================================================

-- Insert default wallet types if they don't exist
INSERT INTO wallet_types (code, name, description, icon, is_active, is_default, min_balance, max_balance, metadata) VALUES
    ('primary', 'Primary Wallet', 'Default/main wallet for daily transactions', 'wallet', true, true, 0.00, NULL, '{"color": "#4CAF50", "permissions": ["deposit", "withdraw", "transfer"]}'),
    ('savings', 'Savings Wallet', 'Savings wallet with interest or restrictions', 'savings', true, false, 0.00, NULL, '{"color": "#2196F3", "permissions": ["deposit", "withdraw"], "interest_rate": 2.5}'),
    ('investment', 'Investment Wallet', 'Wallet for investment purposes', 'chart-line', true, false, 1000.00, NULL, '{"color": "#FF9800", "permissions": ["deposit", "withdraw", "invest"], "risk_level": "medium"}'),
    ('business', 'Business Wallet', 'Wallet for business transactions', 'briefcase', true, false, 0.00, NULL, '{"color": "#9C27B0", "permissions": ["deposit", "withdraw", "transfer", "invoice"], "requires_verification": true}'),
    ('escrow', 'Escrow Wallet', 'Wallet for holding funds in escrow', 'lock', true, false, 0.00, NULL, '{"color": "#607D8B", "permissions": ["deposit"], "auto_release_days": 30}')
ON CONFLICT (code) DO NOTHING;

-- ============================================================================
-- UPDATE WALLETS TABLE CONSTRAINT
-- ============================================================================

-- Update wallets table to reference wallet_types
-- Note: This assumes wallet_types table exists first
DO $$
BEGIN
    -- Drop existing constraint if it exists
    IF EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'wallets_wallet_type_check' 
        AND conrelid = 'wallets'::regclass
    ) THEN
        ALTER TABLE wallets DROP CONSTRAINT wallets_wallet_type_check;
    END IF;

    -- Add foreign key constraint to wallet_types
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'wallets_wallet_type_fkey' 
        AND conrelid = 'wallets'::regclass
    ) THEN
        ALTER TABLE wallets 
        ADD CONSTRAINT wallets_wallet_type_fkey 
        FOREIGN KEY (wallet_type) 
        REFERENCES wallet_types(code) 
        ON DELETE RESTRICT;
    END IF;
END $$;

-- Update wallet_transactions table constraint similarly
DO $$
BEGIN
    -- Drop existing constraint if it exists
    IF EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'wallet_transactions_wallet_type_check' 
        AND conrelid = 'wallet_transactions'::regclass
    ) THEN
        ALTER TABLE wallet_transactions DROP CONSTRAINT wallet_transactions_wallet_type_check;
    END IF;

    -- Add foreign key constraint to wallet_types
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'wallet_transactions_wallet_type_fkey' 
        AND conrelid = 'wallet_transactions'::regclass
    ) THEN
        ALTER TABLE wallet_transactions 
        ADD CONSTRAINT wallet_transactions_wallet_type_fkey 
        FOREIGN KEY (wallet_type) 
        REFERENCES wallet_types(code) 
        ON DELETE RESTRICT;
    END IF;
END $$;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE wallet_types IS 'Wallet type definitions and configurations';
COMMENT ON COLUMN wallet_types.code IS 'Primary key, wallet type code (lowercase, alphanumeric + underscore)';
COMMENT ON COLUMN wallet_types.is_default IS 'Only one wallet type can be set as default';
COMMENT ON COLUMN wallet_types.metadata IS 'JSON metadata for additional wallet type properties';
