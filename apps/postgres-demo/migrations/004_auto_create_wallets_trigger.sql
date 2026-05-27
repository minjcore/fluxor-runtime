-- ============================================================================
-- AUTO-CREATE WALLETS TRIGGER
-- Automatically creates wallets for new users based on active wallet_types
-- ============================================================================

-- IMPORTANT: Wallet system is SEPARATED from users system.
-- The wallet system operates independently and only requires a user_id (string).
-- There is NO foreign key constraint from wallets to users table.
-- This allows the wallet system to work with any user_id identifier, regardless
-- of whether a users table exists or not.

-- Create a function that can be called when a user is created (if you have a users table)
-- OR can be called manually with a user_id string.
-- The wallet system does NOT depend on users table existence.

-- Function to auto-create wallets for a user
CREATE OR REPLACE FUNCTION auto_create_wallets_for_user()
RETURNS TRIGGER AS $$
DECLARE
    wallet_type_record RECORD;
    initial_balance DECIMAL(10,2) := 0.00;
BEGIN
    -- Loop through all active wallet types
    FOR wallet_type_record IN 
        SELECT code, min_balance 
        FROM wallet_types 
        WHERE is_active = true
    LOOP
        -- Check if wallet already exists (avoid duplicates)
        IF NOT EXISTS (
            SELECT 1 FROM wallets 
            WHERE user_id = NEW.user_id 
            AND wallet_type = wallet_type_record.code
        ) THEN
            -- Set initial balance based on min_balance if defined
            IF wallet_type_record.min_balance IS NOT NULL AND wallet_type_record.min_balance > 0 THEN
                initial_balance := wallet_type_record.min_balance;
            ELSE
                initial_balance := 0.00;
            END IF;

            -- Create wallet
            INSERT INTO wallets (user_id, wallet_type, balance, frozen)
            VALUES (NEW.user_id, wallet_type_record.code, initial_balance, 0.00)
            ON CONFLICT (user_id, wallet_type) DO NOTHING;
        END IF;
    END LOOP;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- NOTE: To use this trigger, you need to have a users table
-- Uncomment and modify the following when you have a users table:
-- ============================================================================

-- Example: Create users table if it doesn't exist
-- CREATE TABLE IF NOT EXISTS users (
--     user_id VARCHAR(255) PRIMARY KEY,
--     username VARCHAR(255) UNIQUE NOT NULL,
--     email VARCHAR(255) UNIQUE,
--     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
-- );

-- Create trigger on users table
-- DROP TRIGGER IF EXISTS trigger_auto_create_wallets ON users;
-- CREATE TRIGGER trigger_auto_create_wallets
--     AFTER INSERT ON users
--     FOR EACH ROW
--     EXECUTE FUNCTION auto_create_wallets_for_user();

-- ============================================================================
-- ALTERNATIVE: Manual function call
-- ============================================================================

-- You can also call this function manually from application code:
-- SELECT auto_create_wallets_for_user_manual('user_id_here');

-- Create a manual function that can be called with user_id
CREATE OR REPLACE FUNCTION auto_create_wallets_for_user_manual(p_user_id VARCHAR(255))
RETURNS VOID AS $$
DECLARE
    wallet_type_record RECORD;
    initial_balance DECIMAL(10,2) := 0.00;
BEGIN
    -- Loop through all active wallet types
    FOR wallet_type_record IN 
        SELECT code, min_balance 
        FROM wallet_types 
        WHERE is_active = true
    LOOP
        -- Check if wallet already exists (avoid duplicates)
        IF NOT EXISTS (
            SELECT 1 FROM wallets 
            WHERE user_id = p_user_id 
            AND wallet_type = wallet_type_record.code
        ) THEN
            -- Set initial balance based on min_balance if defined
            IF wallet_type_record.min_balance IS NOT NULL AND wallet_type_record.min_balance > 0 THEN
                initial_balance := wallet_type_record.min_balance;
            ELSE
                initial_balance := 0.00;
            END IF;

            -- Create wallet
            INSERT INTO wallets (user_id, wallet_type, balance, frozen)
            VALUES (p_user_id, wallet_type_record.code, initial_balance, 0.00)
            ON CONFLICT (user_id, wallet_type) DO NOTHING;
        END IF;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON FUNCTION auto_create_wallets_for_user() IS 'Trigger function to auto-create wallets when a user is created';
COMMENT ON FUNCTION auto_create_wallets_for_user_manual(VARCHAR) IS 'Manual function to create wallets for a user based on active wallet_types';
