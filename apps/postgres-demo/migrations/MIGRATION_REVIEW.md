# Database Migration Review

## Overview
This document reviews all database migrations in the `apps/postgres-demo/migrations/` directory.

## Migration Files

1. **000_initial_setup.sql** - Initial schema setup with extensions and base tables
2. **001_create_ledger_tables.sql** - Double-entry bookkeeping system
3. **002_add_wallet_type.sql** - Adds wallet_type column support
4. **003_create_wallet_types_table.sql** - Wallet type definitions
5. **004_auto_create_wallets_trigger.sql** - Auto-creation of wallets for users
6. **005_create_wallet_rules_table.sql** - Business rules and constraints
7. **006_create_system_settings_table.sql** - System configuration settings
8. **007_create_approvals_table.sql** - Two-person approval system
9. **008_add_wallet_constraints.sql** - **NEW**: Critical wallet constraints

---

## Critical Issues Found & Fixed

### ✅ Issue #1: Missing CHECK Constraints on Wallets Table
**Status: FIXED** (migration 008_add_wallet_constraints.sql)

**Problem:**
The `wallets` table was missing critical CHECK constraints:
- No constraint to ensure `balance >= 0`
- No constraint to ensure `frozen >= 0`
- No constraint to ensure `balance >= frozen` (critical!)

**Impact:**
Without these constraints, the database could store invalid data where:
- Balance or frozen amounts could be negative
- Frozen amount could exceed balance (causing available balance to be negative)

**Fix:**
Added migration `008_add_wallet_constraints.sql` with three CHECK constraints:
```sql
CHECK (balance >= 0)
CHECK (frozen >= 0)
CHECK (balance >= frozen)  -- Critical: ensures available balance >= 0
```

**Related Code:**
This fix aligns with the wallet freeze logic in `wallet_service.go` which checks `(balance - frozen) >= amount` before freezing funds.

---

### ⚠️ Issue #2: Schema Conflicts Between Migrations

**Problem:**
There are some inconsistencies between migration files:

1. **wallet_types table created twice:**
   - Created in `000_initial_setup.sql` with `code VARCHAR(255)`, `metadata VARCHAR(255)`
   - Created again in `003_create_wallet_types_table.sql` with `code VARCHAR(20)`, `metadata JSONB`

2. **wallet_type column size inconsistency:**
   - `000_initial_setup.sql`: `wallet_type VARCHAR(50)`
   - `002_add_wallet_type.sql`: tries to add `wallet_type VARCHAR(20)`
   - `003_create_wallet_types_table.sql`: uses `VARCHAR(20)`

**Current Status:**
- The `IF NOT EXISTS` clauses prevent errors, but the first migration wins
- The actual schema may have VARCHAR(255) for code and VARCHAR(50) for wallet_type
- Migration 003 tries to update constraints but may not change column types

**Recommendation:**
- **Option A (Recommended)**: Remove wallet_types table creation from `000_initial_setup.sql` and let `003_create_wallet_types_table.sql` handle it completely
- **Option B**: Update `000_initial_setup.sql` to use the correct types from the start
- **Option C**: Add an explicit ALTER TABLE migration to fix column types if needed

---

### ⚠️ Issue #3: Missing Foreign Key Constraint Documentation

**Problem:**
The wallet_transactions table has a composite foreign key `(user_id, wallet_type)`, but this is not clearly visible in the initial setup.

**Current Status:**
- Migration `002_add_wallet_type.sql` adds the foreign key
- The constraint exists in the codebase

**Recommendation:**
- Ensure the foreign key is documented in migration comments
- Consider adding verification queries at the end of migrations

---

## Strengths

### ✅ Well-Structured Migrations
- Good use of `IF NOT EXISTS` clauses for idempotency
- Proper use of `DO $$` blocks for conditional operations
- Clear separation of concerns across migrations

### ✅ Comprehensive Schema
- Complete wallet system with types, rules, and transactions
- Double-entry bookkeeping system (ledger)
- Two-person approval system
- System settings management

### ✅ Data Integrity
- Foreign key constraints properly defined
- Indexes created for performance
- Triggers for automatic timestamps and business logic

### ✅ Documentation
- Good comments explaining the purpose of tables and columns
- Clear section headers in SQL files

---

## Recommendations

### 1. Run Migration 008 Immediately
The new migration `008_add_wallet_constraints.sql` should be run immediately to add critical data integrity constraints.

### 2. Review Schema Conflicts
Consider consolidating the wallet_types table creation to avoid confusion:
- Remove it from `000_initial_setup.sql` OR
- Update it to match the structure in `003_create_wallet_types_table.sql`

### 3. Add Migration Verification Script
Create a script to verify all migrations have been applied and constraints exist:
```sql
-- Verify constraints exist
SELECT conname, contype 
FROM pg_constraint 
WHERE conrelid = 'wallets'::regclass;
```

### 4. Consider Migration Versioning
Add a migration tracking table to record which migrations have been applied:
```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 5. Add Rollback Scripts (Optional)
For production environments, consider adding rollback migrations for major changes.

---

## Testing Checklist

Before deploying to production, verify:

- [ ] All migrations run successfully in order
- [ ] CHECK constraints are in place (run migration 008)
- [ ] Foreign keys are properly enforced
- [ ] Indexes are created for all lookup columns
- [ ] Triggers are functioning (updated_at, auto-create wallets)
- [ ] Default wallet types are inserted
- [ ] Default system settings are inserted
- [ ] Default wallet rules are inserted

---

## Related Files

- `apps/postgres-demo/services/wallet_service.go` - Wallet business logic
- `apps/postgres-demo/services/purchase_service.go` - Purchase transactions
- `apps/postgres-demo/main.go` - Application setup

---

## Migration Execution Order

Migrations should be run in this exact order:

1. `000_initial_setup.sql` - Base schema
2. `001_create_ledger_tables.sql` - Ledger system (optional, if using ledger)
3. `002_add_wallet_type.sql` - Adds wallet_type support
4. `003_create_wallet_types_table.sql` - Wallet type definitions
5. `004_auto_create_wallets_trigger.sql` - Auto-creation triggers
6. `005_create_wallet_rules_table.sql` - Business rules
7. `006_create_system_settings_table.sql` - System settings
8. `007_create_approvals_table.sql` - Approval system
9. `008_add_wallet_constraints.sql` - **NEW**: Critical constraints

---

## Summary

The migrations are well-structured overall, but the missing CHECK constraints on the wallets table were a critical issue that could have led to data integrity problems. Migration 008 fixes this issue.

The schema conflicts between migrations 000 and 003 should be addressed for clarity, but they don't cause runtime errors due to `IF NOT EXISTS` clauses.

All other aspects of the migration system (foreign keys, indexes, triggers, documentation) are well-implemented.
