-- ============================================================================
-- Ensure users table has a password column (for auth)
-- Some setups or shared DBs may have users without this column.
-- ============================================================================

ALTER TABLE users ADD COLUMN IF NOT EXISTS password VARCHAR(255);

-- Backfill from password_hash if that column exists and password is null
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'users' AND column_name = 'password_hash'
    ) AND EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'users' AND column_name = 'password'
    ) THEN
        UPDATE users SET password = password_hash
        WHERE password IS NULL AND password_hash IS NOT NULL;
    END IF;
END $$;
