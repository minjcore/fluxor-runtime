-- ============================================================================
-- INITIAL DATABASE SETUP
-- This migration runs before all other migrations
-- Sets up extensions, configurations, and initial database structure
-- ============================================================================

-- Enable required PostgreSQL extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- DATABASE CONFIGURATION
-- ============================================================================

-- Set timezone to UTC for consistency
SET timezone = 'UTC';

-- ============================================================================
-- INITIAL SCHEMA SETUP (if using schemas)
-- ============================================================================

-- Create public schema if it doesn't exist (usually exists by default)
-- CREATE SCHEMA IF NOT EXISTS public;

-- ============================================================================
-- HELPER FUNCTIONS (if needed)
-- ============================================================================

-- Function to update updated_at timestamp automatically
-- This will be used by tables that have updated_at columns
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON FUNCTION update_updated_at_column() IS 'Automatically updates updated_at timestamp when a row is modified';

-- ============================================================================
-- TABLE CREATIONS (from generated migrations)
-- Tables are created in dependency order
-- ============================================================================

-- ============================================================================
-- CREATE USERS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255),
    password VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);
CREATE INDEX IF NOT EXISTS idx_users_updated_at ON users(updated_at);

-- ============================================================================
-- CREATE WALLET_TYPES TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS wallet_types (
    code VARCHAR(255) NOT NULL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    icon VARCHAR(255),
    is_active BOOLEAN NOT NULL,
    is_default BOOLEAN NOT NULL,
    min_balance DECIMAL(10,2),
    max_balance DECIMAL(10,2),
    metadata VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_wallet_types_created_at ON wallet_types(created_at);
CREATE INDEX IF NOT EXISTS idx_wallet_types_updated_at ON wallet_types(updated_at);

-- ============================================================================
-- CREATE PRODUCTS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    stock INTEGER NOT NULL
);

-- ============================================================================
-- CREATE ACCOUNTS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS accounts (
    id SERIAL PRIMARY KEY,
    code VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    parent_id INTEGER,
    is_system BOOLEAN NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_accounts_parent_id ON accounts(parent_id);
CREATE INDEX IF NOT EXISTS idx_accounts_created_at ON accounts(created_at);

-- Add foreign key constraint if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'accounts_parent_id_fkey' 
        AND conrelid = 'accounts'::regclass
    ) THEN
        ALTER TABLE accounts ADD CONSTRAINT accounts_parent_id_fkey 
        FOREIGN KEY (parent_id) REFERENCES accounts(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
    END IF;
END $$;

-- ============================================================================
-- CREATE ORDERS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    total DECIMAL(10,2) NOT NULL,
    status VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);

-- ============================================================================
-- CREATE ORDER_ITEMS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    subtotal DECIMAL(10,2) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_product_id ON order_items(product_id);

-- Add foreign key constraints if they don't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'order_items_order_id_fkey' 
        AND conrelid = 'order_items'::regclass
    ) THEN
        ALTER TABLE order_items ADD CONSTRAINT order_items_order_id_fkey 
        FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
    END IF;
    
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'order_items_product_id_fkey' 
        AND conrelid = 'order_items'::regclass
    ) THEN
        ALTER TABLE order_items ADD CONSTRAINT order_items_product_id_fkey 
        FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
    END IF;
END $$;

-- ============================================================================
-- CREATE JOURNALS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS journals (
    id SERIAL PRIMARY KEY,
    reference_type VARCHAR(50) NOT NULL,
    reference_id INTEGER,
    description TEXT NOT NULL,
    status VARCHAR(50) NOT NULL,
    posted_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_journals_reference_id ON journals(reference_id);
CREATE INDEX IF NOT EXISTS idx_journals_created_at ON journals(created_at);

-- ============================================================================
-- CREATE LEDGER_ENTRIES TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS ledger_entries (
    id SERIAL PRIMARY KEY,
    journal_id INTEGER NOT NULL,
    account_id INTEGER NOT NULL,
    debit DECIMAL(10,2) NOT NULL,
    credit DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ledger_entries_journal_id ON ledger_entries(journal_id);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_account_id ON ledger_entries(account_id);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_created_at ON ledger_entries(created_at);

-- Add foreign key constraints if they don't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'ledger_entries_journal_id_fkey' 
        AND conrelid = 'ledger_entries'::regclass
    ) THEN
        ALTER TABLE ledger_entries ADD CONSTRAINT ledger_entries_journal_id_fkey 
        FOREIGN KEY (journal_id) REFERENCES journals(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
    END IF;
    
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'ledger_entries_account_id_fkey' 
        AND conrelid = 'ledger_entries'::regclass
    ) THEN
        ALTER TABLE ledger_entries ADD CONSTRAINT ledger_entries_account_id_fkey 
        FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
    END IF;
END $$;

-- ============================================================================
-- CREATE WALLETS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS wallets (
    user_id VARCHAR(255) NOT NULL,
    wallet_type VARCHAR(50) NOT NULL,
    balance DECIMAL(10,2) NOT NULL,
    frozen DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, wallet_type)
);

CREATE INDEX IF NOT EXISTS idx_wallets_user_id ON wallets(user_id);
CREATE INDEX IF NOT EXISTS idx_wallets_created_at ON wallets(created_at);
CREATE INDEX IF NOT EXISTS idx_wallets_updated_at ON wallets(updated_at);

-- Add foreign key constraint if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'wallets_wallet_type_fkey' 
        AND conrelid = 'wallets'::regclass
    ) THEN
        ALTER TABLE wallets ADD CONSTRAINT wallets_wallet_type_fkey 
        FOREIGN KEY (wallet_type) REFERENCES wallet_types(code) ON DELETE RESTRICT ON UPDATE RESTRICT;
    END IF;
END $$;

-- Create trigger for automatic updated_at timestamp on wallets table
DROP TRIGGER IF EXISTS trigger_wallets_updated_at ON wallets;
CREATE TRIGGER trigger_wallets_updated_at
    BEFORE UPDATE ON wallets
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- CREATE WALLET_TRANSACTIONS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS wallet_transactions (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    wallet_type VARCHAR(50) NOT NULL,
    type VARCHAR(255) NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    description TEXT NOT NULL,
    order_id INTEGER,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_wallet_transactions_user_id ON wallet_transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_order_id ON wallet_transactions(order_id);
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_created_at ON wallet_transactions(created_at);

-- Add foreign key constraints if they don't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'wallet_transactions_order_id_fkey' 
        AND conrelid = 'wallet_transactions'::regclass
    ) THEN
        ALTER TABLE wallet_transactions ADD CONSTRAINT wallet_transactions_order_id_fkey 
        FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE RESTRICT ON UPDATE RESTRICT;
    END IF;
    
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'wallet_transactions_wallet_type_fkey' 
        AND conrelid = 'wallet_transactions'::regclass
    ) THEN
        ALTER TABLE wallet_transactions ADD CONSTRAINT wallet_transactions_wallet_type_fkey 
        FOREIGN KEY (wallet_type) REFERENCES wallet_types(code) ON DELETE RESTRICT ON UPDATE RESTRICT;
    END IF;
END $$;

-- ============================================================================
-- CREATE WALLET_RULES TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS wallet_rules (
    id SERIAL PRIMARY KEY,
    rule_name VARCHAR(255) NOT NULL UNIQUE,
    rule_type VARCHAR(50) NOT NULL,
    wallet_type VARCHAR(255),
    user_id VARCHAR(255),
    operation_type VARCHAR(255),
    min_value DECIMAL(10,2),
    max_value DECIMAL(10,2),
    period_type VARCHAR(255),
    is_active BOOLEAN NOT NULL DEFAULT true,
    priority INTEGER NOT NULL,
    description TEXT NOT NULL,
    error_message VARCHAR(255),
    metadata VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_wallet_rules_user_id ON wallet_rules(user_id);
CREATE INDEX IF NOT EXISTS idx_wallet_rules_created_at ON wallet_rules(created_at);
CREATE INDEX IF NOT EXISTS idx_wallet_rules_updated_at ON wallet_rules(updated_at);

-- Add foreign key constraint if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'wallet_rules_wallet_type_fkey' 
        AND conrelid = 'wallet_rules'::regclass
    ) THEN
        ALTER TABLE wallet_rules ADD CONSTRAINT wallet_rules_wallet_type_fkey 
        FOREIGN KEY (wallet_type) REFERENCES wallet_types(code) ON DELETE CASCADE ON UPDATE RESTRICT;
    END IF;
END $$;

-- ============================================================================
-- CREATE SYSTEM_SETTINGS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS system_settings (
    id SERIAL PRIMARY KEY,
    setting_key VARCHAR(255) NOT NULL UNIQUE,
    setting_value TEXT,
    data_type VARCHAR(50) NOT NULL DEFAULT 'string',
    category VARCHAR(255),
    description TEXT NOT NULL,
    is_encrypted BOOLEAN NOT NULL DEFAULT false,
    is_readonly BOOLEAN NOT NULL DEFAULT false,
    validation_rule TEXT,
    default_value TEXT,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(255)
);

CREATE INDEX IF NOT EXISTS idx_system_settings_created_at ON system_settings(created_at);
CREATE INDEX IF NOT EXISTS idx_system_settings_updated_at ON system_settings(updated_at);

-- ============================================================================
-- CREATE APPROVALS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS approvals (
    id SERIAL PRIMARY KEY,
    approval_type VARCHAR(50) NOT NULL,
    entity_type VARCHAR(255) NOT NULL,
    entity_id VARCHAR(255),
    operation VARCHAR(50) NOT NULL,
    original_data JSONB,
    new_data JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    requested_by VARCHAR(255) NOT NULL,
    approved_by_first VARCHAR(255),
    approved_by_second VARCHAR(255),
    rejected_by VARCHAR(255),
    rejection_reason TEXT,
    expires_at TIMESTAMP,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    approved_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_approvals_entity_id ON approvals(entity_id);
CREATE INDEX IF NOT EXISTS idx_approvals_created_at ON approvals(created_at);
CREATE INDEX IF NOT EXISTS idx_approvals_updated_at ON approvals(updated_at);
