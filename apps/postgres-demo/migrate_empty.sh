#!/bin/bash

# ============================================================================
# MIGRATE WITH EMPTY DATABASE SCRIPT
# ============================================================================
# This script drops all tables and database objects, then runs all migrations
# from scratch. This is useful for development/testing when you need a clean
# database schema that matches the current model definitions.
#
# Usage:
#   ./migrate_empty.sh
#   DB_NAME=my_db ./migrate_empty.sh
#
# Environment Variables:
#   DB_NAME     - Database name (default: fluxor_db)
#   DB_USER     - Database user (default: postgres)
#   DB_HOST     - Database host (default: localhost)
#   DB_PORT     - Database port (default: 5432)
#   PGPASSWORD  - Database password
# ============================================================================

# ============================================================================
# CONFIGURATION
# ============================================================================

# Database connection parameters (can be overridden by environment variables)
DB_NAME="${DB_NAME:-fluxor_db}"
DB_USER="${DB_USER:-postgres}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"

# ============================================================================
# CONSTANTS
# ============================================================================

# ANSI color codes for terminal output
readonly GREEN='\033[0;32m'
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Migration files in execution order (must match model dependencies)
readonly MIGRATIONS=(
    "migrations/000_initial_setup.sql"
    "migrations/001_create_ledger_tables.sql"
    "migrations/002_add_wallet_type.sql"
    "migrations/003_create_wallet_types_table.sql"
    "migrations/004_auto_create_wallets_trigger.sql"
    "migrations/005_create_wallet_rules_table.sql"
    "migrations/006_create_system_settings_table.sql"
    "migrations/007_create_approvals_table.sql"
)

# ============================================================================
# UTILITY FUNCTIONS
# ============================================================================

# print_info prints an informational message
print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

# print_success prints a success message
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# print_error prints an error message
print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# print_warning prints a warning message
print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# ============================================================================
# VALIDATION FUNCTIONS
# ============================================================================

# check_dependencies verifies that required tools are available
check_dependencies() {
    if ! command -v psql &> /dev/null; then
        print_error "psql command not found. Please install PostgreSQL client tools."
        exit 1
    fi
}

# confirm_action prompts the user for confirmation
confirm_action() {
    print_warning "This will DROP ALL TABLES and DATA in database: $DB_NAME"
    print_warning "Are you sure you want to continue? (yes/no)"
    read -r confirmation
    
    if [ "$confirmation" != "yes" ]; then
        print_info "Migration cancelled."
        exit 0
    fi
}

# ============================================================================
# DATABASE OPERATIONS
# ============================================================================

# drop_all_objects drops all tables, functions, triggers, sequences, views, and types
drop_all_objects() {
    print_info "Step 1: Dropping all existing tables and objects..."
    
    local drop_sql
    drop_sql=$(cat <<'EOF'
-- Drop all database objects in the public schema
DO $$
DECLARE
    r RECORD;
BEGIN
    -- Drop all tables (CASCADE to handle dependencies)
    FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') LOOP
        EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
    END LOOP;
    
    -- Drop all functions
    FOR r IN (
        SELECT proname, oidvectortypes(proargtypes) as argtypes 
        FROM pg_proc 
        WHERE pronamespace = (SELECT oid FROM pg_namespace WHERE nspname = 'public')
    ) LOOP
        EXECUTE 'DROP FUNCTION IF EXISTS ' || quote_ident(r.proname) || '(' || r.argtypes || ') CASCADE';
    END LOOP;
    
    -- Drop all sequences
    FOR r IN (
        SELECT sequence_name 
        FROM information_schema.sequences 
        WHERE sequence_schema = 'public'
    ) LOOP
        EXECUTE 'DROP SEQUENCE IF EXISTS ' || quote_ident(r.sequence_name) || ' CASCADE';
    END LOOP;
    
    -- Drop all views
    FOR r IN (
        SELECT table_name 
        FROM information_schema.views 
        WHERE table_schema = 'public'
    ) LOOP
        EXECUTE 'DROP VIEW IF EXISTS ' || quote_ident(r.table_name) || ' CASCADE';
    END LOOP;
    
    -- Drop all custom types
    FOR r IN (
        SELECT typname 
        FROM pg_type 
        WHERE typnamespace = (SELECT oid FROM pg_namespace WHERE nspname = 'public') 
        AND typtype = 'c'
    ) LOOP
        EXECUTE 'DROP TYPE IF EXISTS ' || quote_ident(r.typname) || ' CASCADE';
    END LOOP;
END
$$;
EOF
)
    
    if PGPASSWORD="${PGPASSWORD}" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$drop_sql" > /dev/null 2>&1; then
        print_success "All existing tables and objects dropped"
        return 0
    else
        print_error "Failed to drop existing objects"
        return 1
    fi
}

# run_migrations executes all migration files in order
run_migrations() {
    print_info "Step 2: Running all migrations from scratch..."
    
    local migration_count=0
    local failed_migrations=()
    
    for migration in "${MIGRATIONS[@]}"; do
        if [ ! -f "$migration" ]; then
            print_error "Migration file not found: $migration"
            failed_migrations+=("$migration")
            continue
        fi
        
        ((migration_count++))
        echo -e "${YELLOW}[$migration_count/${#MIGRATIONS[@]}] Running migration: $migration${NC}"
        
        if PGPASSWORD="${PGPASSWORD}" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$migration" > /dev/null 2>&1; then
            print_success "Migration $migration completed successfully"
        else
            print_error "Migration $migration failed"
            failed_migrations+=("$migration")
        fi
    done
    
    if [ ${#failed_migrations[@]} -eq 0 ]; then
        return 0
    else
        print_error "Failed migrations:"
        for failed in "${failed_migrations[@]}"; do
            print_error "  - $failed"
        done
        return 1
    fi
}

# show_summary displays migration summary and statistics
show_summary() {
    print_success "All migrations completed successfully!"
    print_info "Database $DB_NAME has been migrated with empty/fresh schema"
    
    # Get table count
    local table_count
    table_count=$(PGPASSWORD="${PGPASSWORD}" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" 2>/dev/null | xargs)
    print_info "Total tables created: $table_count"
}

# ============================================================================
# MAIN EXECUTION
# ============================================================================

main() {
    check_dependencies
    confirm_action
    
    echo ""
    print_info "Starting fresh migration for database: $DB_NAME"
    echo ""
    
    if ! drop_all_objects; then
        exit 1
    fi
    
    echo ""
    
    if ! run_migrations; then
        exit 1
    fi
    
    echo ""
    show_summary
}

# Execute main function
main
