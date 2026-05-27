#!/bin/bash

# ============================================================================
# MIGRATE WITH FRESH DATABASE SCRIPT
# ============================================================================
# This script drops and recreates the entire database, then runs all migrations
# from scratch. This provides a completely clean database that matches the
# current model definitions.
#
# Usage:
#   ./migrate_fresh_db.sh
#   DB_NAME=my_db ./migrate_fresh_db.sh
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
    print_warning "This will DROP and RECREATE the entire database: $DB_NAME"
    print_warning "ALL DATA WILL BE PERMANENTLY LOST!"
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

# drop_database drops the target database if it exists
drop_database() {
    print_info "Step 1: Dropping database if it exists..."
    
    # Connect to postgres database to drop the target database
    if PGPASSWORD="${PGPASSWORD}" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "DROP DATABASE IF EXISTS $DB_NAME;" > /dev/null 2>&1; then
        print_success "Database dropped (if it existed)"
        return 0
    else
        print_error "Failed to drop database"
        return 1
    fi
}

# create_database creates a fresh database
create_database() {
    print_info "Step 2: Creating fresh database..."
    
    if PGPASSWORD="${PGPASSWORD}" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "CREATE DATABASE $DB_NAME;" > /dev/null 2>&1; then
        print_success "Database created successfully"
        return 0
    else
        print_error "Failed to create database"
        return 1
    fi
}

# run_migrations executes all migration files in order
run_migrations() {
    print_info "Step 3: Running all migrations..."
    
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
    print_info "Database $DB_NAME has been created fresh with all migrations applied"
    
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
    print_info "Starting fresh database migration for: $DB_NAME"
    echo ""
    
    if ! drop_database; then
        exit 1
    fi
    
    if ! create_database; then
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
