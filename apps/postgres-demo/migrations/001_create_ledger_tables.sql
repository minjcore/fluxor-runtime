-- ============================================================================
-- LEDGER SYSTEM MIGRATION
-- Creates tables for double-entry bookkeeping system
-- ============================================================================

-- Table: accounts
-- Represents chart of accounts (Tài khoản kế toán)
CREATE TABLE IF NOT EXISTS accounts (
    id SERIAL PRIMARY KEY,
    code VARCHAR(20) UNIQUE NOT NULL,          -- Account code (e.g., '1100', '2100')
    name VARCHAR(100) NOT NULL,                -- Account name
    type VARCHAR(20) NOT NULL CHECK (type IN ('asset', 'liability', 'equity', 'revenue', 'expense')),
    parent_id INTEGER REFERENCES accounts(id), -- Parent account for hierarchy
    is_system BOOLEAN DEFAULT false,           -- System accounts cannot be deleted
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for faster lookups
CREATE INDEX IF NOT EXISTS idx_accounts_code ON accounts(code);
CREATE INDEX IF NOT EXISTS idx_accounts_type ON accounts(type);
CREATE INDEX IF NOT EXISTS idx_accounts_parent ON accounts(parent_id);

-- Table: journals
-- Represents journal entries (Bút toán)
-- A journal groups multiple ledger entries that must balance (debit = credit)
CREATE TABLE IF NOT EXISTS journals (
    id SERIAL PRIMARY KEY,
    reference_type VARCHAR(50) NOT NULL,       -- Type of transaction (wallet_topup, purchase, etc.)
    reference_id INTEGER,                       -- ID of related entity (order_id, transaction_id, etc.)
    description TEXT,                           -- Description of the journal entry
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'posted', 'reversed')),
    posted_at TIMESTAMP,                        -- When the journal was posted
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255)                    -- User who created this journal
);

-- Indexes for journals
CREATE INDEX IF NOT EXISTS idx_journals_reference ON journals(reference_type, reference_id);
CREATE INDEX IF NOT EXISTS idx_journals_status ON journals(status);
CREATE INDEX IF NOT EXISTS idx_journals_created_at ON journals(created_at);

-- Table: ledger_entries
-- Represents individual debit/credit entries in the ledger
-- Each entry belongs to a journal and an account
CREATE TABLE IF NOT EXISTS ledger_entries (
    id SERIAL PRIMARY KEY,
    journal_id INTEGER NOT NULL REFERENCES journals(id) ON DELETE CASCADE,
    account_id INTEGER NOT NULL REFERENCES accounts(id),
    debit DECIMAL(15,2) DEFAULT 0 CHECK (debit >= 0),
    credit DECIMAL(15,2) DEFAULT 0 CHECK (credit >= 0),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure only one of debit or credit is non-zero
    CONSTRAINT check_debit_or_credit CHECK (
        (debit > 0 AND credit = 0) OR (debit = 0 AND credit > 0)
    )
);

-- Indexes for ledger_entries
CREATE INDEX IF NOT EXISTS idx_ledger_entries_journal ON ledger_entries(journal_id);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_account ON ledger_entries(account_id);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_created_at ON ledger_entries(created_at);

-- ============================================================================
-- SYSTEM ACCOUNTS
-- These accounts are automatically created by InitializeSystemAccounts()
-- ============================================================================

-- Asset accounts (1xxx)
-- 1100: User Wallets - Total amount in all user wallets
-- 1200: Company Cash - Company's cash/bank account

-- Liability accounts (2xxx)
-- 2100: Frozen Funds - Amounts frozen for pending transactions
-- 2200: User Payables - Amounts owed to users

-- Revenue accounts (3xxx)
-- 3100: Sales Revenue - Revenue from product sales
-- 3200: Transaction Fees - Fees from transactions

-- Expense accounts (4xxx)
-- 4100: Refunds - Refund expenses
-- 4200: Operational Expenses - Other operational expenses

-- ============================================================================
-- USEFUL VIEWS FOR REPORTING
-- ============================================================================

-- View: account_balances
-- Shows current balance for each account
CREATE OR REPLACE VIEW account_balances AS
SELECT 
    a.id as account_id,
    a.code as account_code,
    a.name as account_name,
    a.type as account_type,
    COALESCE(SUM(CASE WHEN j.status = 'posted' THEN le.debit ELSE 0 END), 0) as total_debit,
    COALESCE(SUM(CASE WHEN j.status = 'posted' THEN le.credit ELSE 0 END), 0) as total_credit,
    CASE 
        WHEN a.type IN ('asset', 'expense') THEN 
            COALESCE(SUM(CASE WHEN j.status = 'posted' THEN le.debit ELSE 0 END), 0) - 
            COALESCE(SUM(CASE WHEN j.status = 'posted' THEN le.credit ELSE 0 END), 0)
        ELSE 
            COALESCE(SUM(CASE WHEN j.status = 'posted' THEN le.credit ELSE 0 END), 0) - 
            COALESCE(SUM(CASE WHEN j.status = 'posted' THEN le.debit ELSE 0 END), 0)
    END as balance
FROM accounts a
LEFT JOIN ledger_entries le ON a.id = le.account_id
LEFT JOIN journals j ON le.journal_id = j.id
GROUP BY a.id, a.code, a.name, a.type;

-- View: trial_balance
-- Shows trial balance (all accounts with balances)
CREATE OR REPLACE VIEW trial_balance AS
SELECT 
    account_id,
    account_code,
    account_name,
    account_type,
    total_debit,
    total_credit,
    balance
FROM account_balances
WHERE total_debit > 0 OR total_credit > 0
ORDER BY account_code;

-- ============================================================================
-- COMMENTS FOR DOCUMENTATION
-- ============================================================================

COMMENT ON TABLE accounts IS 'Chart of accounts for double-entry bookkeeping';
COMMENT ON TABLE journals IS 'Journal entries that group balanced ledger entries';
COMMENT ON TABLE ledger_entries IS 'Individual debit/credit entries in the ledger';
COMMENT ON VIEW account_balances IS 'Current balance for each account based on posted journals';
COMMENT ON VIEW trial_balance IS 'Trial balance report showing all accounts with activity';
