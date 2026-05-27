# Model to Database Table Mapping Guide

## Current Approach

Hiện tại, project sử dụng **manual mapping** - table names và column names được hardcode trong SQL queries.

### Mapping Table

| Model Struct | Database Table | Notes |
|-------------|----------------|-------|
| `Product` | `products` | Plural, lowercase |
| `Order` | `orders` | Plural, lowercase |
| `OrderItem` | `order_items` | Snake_case |
| `Wallet` | `wallets` | Plural, lowercase |
| `WalletTransaction` | `wallet_transactions` | Snake_case |
| `WalletTypeInfo` | `wallet_types` | Wallet type definitions |
| `Account` | `accounts` | Chart of accounts |
| `Journal` | `journals` | Journal entries (bút toán) |
| `LedgerEntry` | `ledger_entries` | Double-entry bookkeeping entries |

### Field Mapping

#### Product Model → products table
```go
type Product struct {
    ID          int     // → id (SERIAL PRIMARY KEY)
    Name        string  // → name (VARCHAR(255))
    Description string  // → description (TEXT)
    Price       float64 // → price (DECIMAL(10,2))
    Stock       int     // → stock (INTEGER)
}
```

#### Order Model → orders table
```go
type Order struct {
    ID        int       // → id (SERIAL PRIMARY KEY)
    UserID    string    // → user_id (VARCHAR(255))
    Total     float64   // → total (DECIMAL(10,2))
    Status    string    // → status (VARCHAR(50))
    CreatedAt time.Time // → created_at (TIMESTAMP)
    Items     []OrderItem // → order_items table (relationship)
}
```

#### Wallet Model → wallets table
```go
type Wallet struct {
    UserID     string     // → user_id (VARCHAR(255), part of composite PK)
    WalletType WalletType // → wallet_type (VARCHAR(20), part of composite PK)
    Balance    float64    // → balance (DECIMAL(10,2))
    Frozen     float64    // → frozen (DECIMAL(10,2))
}
```

**Important:** 
- The wallets table uses a composite primary key `(user_id, wallet_type)` to support multiple wallets per user.
- **Wallet system is SEPARATED from users system**: There is NO foreign key constraint from `wallets` to a `users` table.
- The wallet system operates independently and only requires a `user_id` (string identifier).
- This allows the wallet system to work with any user identifier, regardless of whether a users table exists.
- Default wallet type is `"primary"`.

**Wallet Types:**
- `primary` - Default/main wallet for a user
- `savings` - Savings wallet (typically with interest or restrictions)
- `investment` - For investment purposes
- `business` - For business transactions
- `escrow` - For holding funds in escrow

#### WalletTransaction Model → wallet_transactions table
```go
type WalletTransaction struct {
    ID          int            // → id (SERIAL PRIMARY KEY)
    UserID      string         // → user_id (VARCHAR(255))
    WalletType  WalletType     // → wallet_type (VARCHAR(20))
    Type        string         // → type (VARCHAR(20)) - "debit" or "credit"
    Amount      float64        // → amount (DECIMAL(10,2))
    Description string         // → description (TEXT)
    OrderID     *int           // → order_id (INTEGER, nullable)
    Status      NullableStatus // → status (VARCHAR(20), nullable)
    CreatedAt   time.Time      // → created_at (TIMESTAMP)
}
```

**Note:** Transactions now include `wallet_type` to track which wallet the transaction belongs to. Foreign key `(user_id, wallet_type)` references `wallets(user_id, wallet_type)`, and `wallet_type` references `wallet_types(code)`.

#### WalletTypeInfo Model → wallet_types table
```go
type WalletTypeInfo struct {
    Code        string   // → code (VARCHAR(20) PRIMARY KEY)
    Name        string   // → name (VARCHAR(100))
    Description string   // → description (TEXT)
    Icon        *string  // → icon (VARCHAR(255), nullable)
    IsActive    bool     // → is_active (BOOLEAN)
    IsDefault   bool     // → is_default (BOOLEAN)
    MinBalance  *float64 // → min_balance (DECIMAL(15,2), nullable)
    MaxBalance  *float64 // → max_balance (DECIMAL(15,2), nullable)
    Metadata    *string  // → metadata (JSONB, nullable)
    CreatedAt   time.Time // → created_at (TIMESTAMP)
    UpdatedAt   time.Time // → updated_at (TIMESTAMP)
}
```

**Note:** The `wallet_types` table stores configuration for each wallet type. When creating a wallet, the system validates against this table to ensure the wallet type exists and is active. This allows dynamic wallet type management without code changes.

#### Account Model → accounts table
```go
type Account struct {
    ID        int         // → id (SERIAL PRIMARY KEY)
    Code      string      // → code (VARCHAR(20) UNIQUE)
    Name      string      // → name (VARCHAR(100))
    Type      AccountType // → type (VARCHAR(20))
    ParentID  *int        // → parent_id (INTEGER, nullable)
    IsSystem  bool        // → is_system (BOOLEAN)
    CreatedAt time.Time   // → created_at (TIMESTAMP)
}
```

#### Journal Model → journals table
```go
type Journal struct {
    ID            int                   // → id (SERIAL PRIMARY KEY)
    ReferenceType ReferenceType         // → reference_type (VARCHAR(50))
    ReferenceID   *int                  // → reference_id (INTEGER, nullable)
    Description   string                // → description (TEXT)
    Status        NullableJournalStatus // → status (VARCHAR(20))
    PostedAt      *time.Time            // → posted_at (TIMESTAMP, nullable)
    CreatedAt     time.Time             // → created_at (TIMESTAMP)
    CreatedBy     string                // → created_by (VARCHAR(255))
    Entries       []LedgerEntry         // → ledger_entries table (relationship)
}
```

#### LedgerEntry Model → ledger_entries table
```go
type LedgerEntry struct {
    ID        int       // → id (SERIAL PRIMARY KEY)
    JournalID int       // → journal_id (INTEGER, FK to journals)
    AccountID int       // → account_id (INTEGER, FK to accounts)
    Debit     float64   // → debit (DECIMAL(15,2))
    Credit    float64   // → credit (DECIMAL(15,2))
    CreatedAt time.Time // → created_at (TIMESTAMP)
    Account   *Account  // → accounts table (joined data)
}
```

### Account Type System

The ledger system uses a standard chart of accounts:

#### Asset Accounts (1xxx)
- **1100**: User Wallets - Total amount in all user wallets
- **1200**: Company Cash - Company's cash/bank account

#### Liability Accounts (2xxx)
- **2100**: Frozen Funds - Amounts frozen for pending transactions
- **2200**: User Payables - Amounts owed to users

#### Revenue Accounts (3xxx)
- **3100**: Sales Revenue - Revenue from product sales
- **3200**: Transaction Fees - Fees from transactions

#### Expense Accounts (4xxx)
- **4100**: Refunds - Refund expenses
- **4200**: Operational Expenses - Other operational expenses

## Naming Conventions

### Table Names
- **Rule**: Plural, lowercase, snake_case
- Examples: `products`, `orders`, `order_items`, `wallets`, `wallet_transactions`, `accounts`, `journals`, `ledger_entries`

### Column Names
- **Rule**: lowercase, snake_case
- Examples: `user_id`, `created_at`, `order_id`

### Primary Keys
- **Rule**: Always named `id` (SERIAL/INTEGER)
- Exceptions: 
  - `wallets` uses composite PRIMARY KEY `(user_id, wallet_type)` to support multiple wallets per user
  - Allows users to have different wallet types (primary, savings, investment, business, escrow)

## Current Implementation

### 1. Table Creation (Manual SQL)

Tables are created in `services/purchase_service.go`:

```go
func (s *PurchaseService) SetupPurchaseTables(ctx context.Context) error {
    queries := []string{
        `CREATE TABLE IF NOT EXISTS products (...)`,
        `CREATE TABLE IF NOT EXISTS orders (...)`,
        // ...
    }
    // Execute queries
}
```

### 2. Query Mapping (Manual)

Queries manually specify table and column names:

```go
// GetProducts example
query := `SELECT id, name, description, price, stock FROM products ORDER BY id ASC`
rows, err := s.db.Query(ctx, query, userID)
// Manual scanning
err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock)
```

## Improved Approach: Using Struct Tags

Để làm mapping tự động hơn, có thể sử dụng struct tags:

### Step 1: Add DB Tags to Models

```go
// models/product.go
type Product struct {
    ID          int     `db:"id" json:"id"`
    Name        string  `db:"name" json:"name"`
    Description string  `db:"description" json:"description"`
    Price       float64 `db:"price" json:"price"`
    Stock       int     `db:"stock" json:"stock"`
}

// models/order.go
type Order struct {
    ID        int       `db:"id" json:"id"`
    UserID    string    `db:"user_id" json:"user_id"`
    Total     float64   `db:"total" json:"total"`
    Status    string    `db:"status" json:"status"`
    CreatedAt time.Time `db:"created_at" json:"created_at"`
    Items     []OrderItem `db:"-" json:"items"` // "-" means ignore for DB
}
```

### Step 2: Create Mapping Helper

```go
// pkg/mapping/table.go
package mapping

import (
    "reflect"
    "strings"
)

// GetTableName converts struct name to table name
// Product -> products, OrderItem -> order_items
func GetTableName(model interface{}) string {
    t := reflect.TypeOf(model)
    if t.Kind() == reflect.Ptr {
        t = t.Elem()
    }
    
    name := t.Name()
    // Convert PascalCase to snake_case and pluralize
    return pluralize(toSnakeCase(name))
}

func toSnakeCase(s string) string {
    var result strings.Builder
    for i, r := range s {
        if i > 0 && r >= 'A' && r <= 'Z' {
            result.WriteRune('_')
        }
        result.WriteRune(r)
    }
    return strings.ToLower(result.String())
}

func pluralize(s string) string {
    if strings.HasSuffix(s, "y") {
        return s[:len(s)-1] + "ies"
    }
    if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") || 
       strings.HasSuffix(s, "z") || strings.HasSuffix(s, "ch") || 
       strings.HasSuffix(s, "sh") {
        return s + "es"
    }
    return s + "s"
}

// GetColumnName gets DB column name from struct field
func GetColumnName(field reflect.StructField) string {
    // Check for db tag
    if dbTag := field.Tag.Get("db"); dbTag != "" && dbTag != "-" {
        return dbTag
    }
    // Fallback to snake_case conversion
    return toSnakeCase(field.Name)
}
```

### Step 3: Use Reflection for Queries

```go
// Example: Build SELECT query automatically
func BuildSelectQuery(model interface{}, whereClause string) string {
    tableName := mapping.GetTableName(model)
    t := reflect.TypeOf(model)
    
    var columns []string
    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        if dbTag := field.Tag.Get("db"); dbTag != "" && dbTag != "-" {
            columns = append(columns, dbTag)
        }
    }
    
    query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(columns, ", "), tableName)
    if whereClause != "" {
        query += " WHERE " + whereClause
    }
    return query
}
```

## Alternative: Using an ORM

Nếu muốn sử dụng ORM, có thể dùng:

### Option 1: GORM
```go
import "gorm.io/gorm"

type Product struct {
    gorm.Model
    Name        string  `gorm:"column:name"`
    Description string  `gorm:"column:description"`
    Price       float64 `gorm:"column:price;type:decimal(10,2)"`
    Stock       int     `gorm:"column:stock"`
}

// Table name mapping
func (Product) TableName() string {
    return "products"
}
```

### Option 2: sqlx
```go
import "github.com/jmoiron/sqlx"

type Product struct {
    ID          int     `db:"id"`
    Name        string  `db:"name"`
    Description string  `db:"description"`
    Price       float64 `db:"price"`
    Stock       int     `db:"stock"`
}

// Automatic struct scanning
var products []Product
err := db.Select(&products, "SELECT * FROM products")
```

## Ledger System (Double-Entry Bookkeeping)

### Overview

The ledger system implements **double-entry bookkeeping** where every transaction affects at least two accounts, and total debits must equal total credits.

### How It Works

1. **Journal**: Groups related ledger entries that balance (debit = credit)
2. **Ledger Entry**: Individual debit or credit entry for a specific account
3. **Account Balance**: Calculated from all posted ledger entries

### Transaction Flow with Ledger

#### Wallet Top-up ($100)
```
Journal #1: Wallet Top-up
├─ Debit:  User Wallets (1100)  $100
└─ Credit: Company Cash (1200)  $100
```

#### Purchase Freeze ($50)
```
Journal #2: Freeze Amount (status: pending)
├─ Debit:  Frozen Funds (2100)      $50
└─ Credit: User Wallets (1100)      $50
```

#### Purchase Commit ($50)
```
Journal #3: Commit Purchase
├─ Debit:  Company Cash (1200)      $50
└─ Credit: Frozen Funds (2100)      $50
```

### Database Migration

Run the migration file to create ledger tables:
```bash
psql -d fluxor_db -f migrations/001_create_ledger_tables.sql
```

Or the tables are automatically created when the application starts (via `InitializeSystemAccounts()`).

### Views

Two views are available for reporting:

1. **account_balances**: Shows current balance for each account
2. **trial_balance**: Shows all accounts with activity (trial balance report)

### Example Queries

```sql
-- Get account balance
SELECT * FROM account_balances WHERE account_code = '1100';

-- Get trial balance (must balance: total_debit = total_credit)
SELECT * FROM trial_balance;

-- Get journal with entries
SELECT j.*, le.*, a.code, a.name
FROM journals j
JOIN ledger_entries le ON j.id = le.journal_id
JOIN accounts a ON le.account_id = a.id
WHERE j.id = 1;
```

## Best Practices

1. **Consistent Naming**: 
   - Use snake_case for DB columns
   - Use PascalCase for Go struct fields
   - Use plural lowercase for table names

2. **Documentation**: 
   - Document mapping in comments
   - Keep this mapping file updated

3. **Validation**: 
   - Validate struct tags match actual DB columns
   - Use migrations to ensure schema matches models

4. **Type Safety**: 
   - Use typed structs instead of `map[string]interface{}`
   - Validate types match DB column types

## Current Project Status

✅ **Manual mapping** - Working, explicit control
⚠️ **No struct tags** - Could be improved
⚠️ **No ORM** - Using raw SQL (good for learning, more verbose)

## Recommendations

1. **Short term**: Add `db` tags to all models for documentation
2. **Medium term**: Create helper functions for common queries
3. **Long term**: Consider lightweight ORM if project grows
