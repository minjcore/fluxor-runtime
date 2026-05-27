-- ============================================================================
-- ADD WALLET TYPE MIGRATION
-- Adds wallet_type column to support multiple wallets per user
-- ============================================================================

-- Step 1: Add wallet_type column to wallets table (if not exists)
-- First, drop the old primary key constraint and any dependent foreign keys
DO $$
BEGIN
    -- Drop foreign key constraint that depends on wallets_pkey first
    IF EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'wallet_transactions_wallet_fkey' 
        AND conrelid = 'wallet_transactions'::regclass
    ) THEN
        ALTER TABLE wallet_transactions DROP CONSTRAINT wallet_transactions_wallet_fkey;
    END IF;
    
    -- Drop old primary key if it exists
    IF EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'wallets_pkey' 
        AND conrelid = 'wallets'::regclass
    ) THEN
        ALTER TABLE wallets DROP CONSTRAINT wallets_pkey;
    END IF;
END $$;

-- Add wallet_type column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'wallets' AND column_name = 'wallet_type'
    ) THEN
        ALTER TABLE wallets 
        ADD COLUMN wallet_type VARCHAR(20) NOT NULL DEFAULT 'primary';
        
        -- Add constraint for wallet_type values
        ALTER TABLE wallets 
        ADD CONSTRAINT wallets_wallet_type_check 
        CHECK (wallet_type IN ('primary', 'savings', 'investment', 'business', 'escrow'));
    END IF;
END $$;

-- Create new composite primary key
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'wallets_pkey' 
        AND conrelid = 'wallets'::regclass
    ) THEN
        ALTER TABLE wallets 
        ADD PRIMARY KEY (user_id, wallet_type);
    END IF;
END $$;

-- Step 2: Add wallet_type column to wallet_transactions table
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'wallet_transactions' AND column_name = 'wallet_type'
    ) THEN
        ALTER TABLE wallet_transactions 
        ADD COLUMN wallet_type VARCHAR(20) NOT NULL DEFAULT 'primary';
        
        -- Add constraint for wallet_type values
        ALTER TABLE wallet_transactions 
        ADD CONSTRAINT wallet_transactions_wallet_type_check 
        CHECK (wallet_type IN ('primary', 'savings', 'investment', 'business', 'escrow'));
        
        -- Add foreign key constraint to wallets
        ALTER TABLE wallet_transactions 
        ADD CONSTRAINT wallet_transactions_wallet_fkey 
        FOREIGN KEY (user_id, wallet_type) 
        REFERENCES wallets(user_id, wallet_type) 
        ON DELETE CASCADE;
        
        -- Create index for faster lookups
        CREATE INDEX IF NOT EXISTS idx_wallet_transactions_wallet_type 
        ON wallet_transactions(user_id, wallet_type);
    END IF;
END $$;

-- Step 3: Create index on wallets for wallet_type lookups
CREATE INDEX IF NOT EXISTS idx_wallets_user_type ON wallets(user_id, wallet_type);

-- ============================================================================
-- VERIFICATION QUERIES (can be run after migration)
-- ============================================================================

-- Verify wallets table structure
-- SELECT column_name, data_type, column_default, is_nullable
-- FROM information_schema.columns
-- WHERE table_name = 'wallets'
-- ORDER BY ordinal_position;

-- Verify wallet_transactions table structure
-- SELECT column_name, data_type, column_default, is_nullable
-- FROM information_schema.columns
-- WHERE table_name = 'wallet_transactions'
-- ORDER BY ordinal_position;

-- Verify constraints
-- SELECT constraint_name, constraint_type
-- FROM information_schema.table_constraints
-- WHERE table_name IN ('wallets', 'wallet_transactions');
