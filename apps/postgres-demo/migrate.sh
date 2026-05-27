#!/bin/bash

# Database connection parameters
DB_NAME="${DB_NAME:-fluxor_db}"
DB_USER="${DB_USER:-postgres}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"

# Find psql (may not be in PATH when run from some environments)
if command -v psql >/dev/null 2>&1; then
    PSQL="psql"
elif [ -x /opt/homebrew/bin/psql ]; then
    PSQL="/opt/homebrew/bin/psql"
elif [ -x /usr/local/bin/psql ]; then
    PSQL="/usr/local/bin/psql"
else
    echo "psql not found. Install PostgreSQL client tools:"
    echo "  macOS (Homebrew): brew install libpq && brew link --force libpq"
    echo "  Or: brew install postgresql"
    exit 1
fi

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "Running migrations for database: $DB_NAME"

# Run migrations in order
MIGRATIONS=(
    "migrations/000_initial_setup.sql"
    "migrations/001_create_ledger_tables.sql"
    "migrations/002_add_wallet_type.sql"
    "migrations/003_create_wallet_types_table.sql"
    "migrations/004_auto_create_wallets_trigger.sql"
    "migrations/005_create_wallet_rules_table.sql"
    "migrations/006_create_system_settings_table.sql"
    "migrations/007_create_approvals_table.sql"
)

for migration in "${MIGRATIONS[@]}"; do
    if [ -f "$migration" ]; then
        echo -e "${YELLOW}Running migration: $migration${NC}"
        if PGPASSWORD="${PGPASSWORD}" "$PSQL" -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$migration"; then
            echo -e "${GREEN}✓ Migration $migration completed successfully${NC}"
        else
            echo -e "${RED}✗ Migration $migration failed${NC}"
            exit 1
        fi
    else
        echo -e "${RED}✗ Migration file not found: $migration${NC}"
        exit 1
    fi
done

echo -e "${GREEN}All migrations completed successfully!${NC}"