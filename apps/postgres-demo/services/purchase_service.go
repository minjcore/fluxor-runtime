package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/fluxorio/fluxor/apps/postgres-demo/models"
	"github.com/fluxorio/fluxor/pkg/dbruntime"
	"github.com/fluxorio/fluxor/pkg/persistence"
)

// PurchaseService handles purchase transactions
type PurchaseService struct {
	db           *dbruntime.DB
	repo         *persistence.SQLRepository
	walletService *WalletService
}

// NewPurchaseService creates a new purchase service
func NewPurchaseService(db *dbruntime.DB, walletService *WalletService) (*PurchaseService, error) {
	// Create products repository
	productsRepo, err := persistence.NewSQLRepository(persistence.DefaultConfig("products", db.DB))
	if err != nil {
		return nil, fmt.Errorf("failed to create products repository: %w", err)
	}

	return &PurchaseService{
		db:            db,
		repo:          productsRepo,
		walletService: walletService,
	}, nil
}

// SetupPurchaseTables creates the necessary database tables
func (s *PurchaseService) SetupPurchaseTables(ctx context.Context) error {
	queries := []string{
		// Products table
		`CREATE TABLE IF NOT EXISTS products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			price DECIMAL(10, 2) NOT NULL,
			stock INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		// Orders table
		`CREATE TABLE IF NOT EXISTS orders (
			id SERIAL PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			total DECIMAL(10, 2) NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		// Order items table
		`CREATE TABLE IF NOT EXISTS order_items (
			id SERIAL PRIMARY KEY,
			order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
			product_id INTEGER NOT NULL REFERENCES products(id),
			quantity INTEGER NOT NULL,
			price DECIMAL(10, 2) NOT NULL,
			subtotal DECIMAL(10, 2) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		// Wallets table - supports multiple wallets per user (composite primary key)
		// Note: wallet_type foreign key will be added by migration 003_create_wallet_types_table.sql
		`CREATE TABLE IF NOT EXISTS wallets (
			user_id VARCHAR(255) NOT NULL,
			wallet_type VARCHAR(20) NOT NULL DEFAULT 'primary',
			balance DECIMAL(10, 2) NOT NULL DEFAULT 0.00 CHECK (balance >= 0),
			frozen DECIMAL(10, 2) NOT NULL DEFAULT 0.00 CHECK (frozen >= 0),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, wallet_type),
			CHECK (balance >= frozen)
		)`,
		
		// Wallet transactions table
		// Note: wallet_type foreign key will be added by migration 003_create_wallet_types_table.sql
		`CREATE TABLE IF NOT EXISTS wallet_transactions (
			id SERIAL PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			wallet_type VARCHAR(20) NOT NULL DEFAULT 'primary',
			type VARCHAR(20) NOT NULL,
			amount DECIMAL(10, 2) NOT NULL,
			description TEXT,
			order_id INTEGER REFERENCES orders(id) ON DELETE SET NULL,
			status VARCHAR(20) DEFAULT 'completed' CHECK (status IN ('pending', 'completed', 'cancelled')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id, wallet_type) REFERENCES wallets(user_id, wallet_type) ON DELETE CASCADE
		)`,
	}

	for _, query := range queries {
		if _, err := s.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

// SeedProducts adds sample products to the database
func (s *PurchaseService) SeedProducts(ctx context.Context) error {
	products := []map[string]interface{}{
		{
			"name":        "Laptop",
			"description": "High-performance laptop",
			"price":       999.99,
			"stock":       10,
		},
		{
			"name":        "Mouse",
			"description": "Wireless mouse",
			"price":       29.99,
			"stock":       50,
		},
		{
			"name":        "Keyboard",
			"description": "Mechanical keyboard",
			"price":       79.99,
			"stock":       30,
		},
		{
			"name":        "Monitor",
			"description": "27-inch 4K monitor",
			"price":       399.99,
			"stock":       15,
		},
		{
			"name":        "Headphones",
			"description": "Noise-cancelling headphones",
			"price":       199.99,
			"stock":       25,
		},
	}

	// Check if products already exist
	var count int64
	productsRepo, _ := persistence.NewSQLRepository(persistence.DefaultConfig("products", s.db.DB))
	count, _ = productsRepo.Count(ctx, persistence.NewQuery())
	if count > 0 {
		// Products already seeded
		return nil
	}

	// Insert products
	for _, product := range products {
		if err := productsRepo.Create(ctx, product); err != nil {
			return fmt.Errorf("failed to create product: %w", err)
		}
	}

	return nil
}

// MakePurchase processes a purchase transaction with ACID guarantees
// Uses transaction isolation level Serializable for maximum consistency
func (s *PurchaseService) MakePurchase(ctx context.Context, req models.PurchaseRequest) (*models.PurchaseResponse, error) {
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("purchase must contain at least one item")
	}

	// Begin transaction directly using database connection with Serializable isolation
	txOptions := &sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	}

	dbTx, err := s.db.DB.BeginTx(ctx, txOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer dbTx.Rollback()

	// Step 1: Validate products and calculate total using raw SQL within transaction
	var total float64
	var orderItems []struct {
		ProductID int
		Quantity  int
		Price     float64
		Subtotal  float64
		Stock     int
	}

	for _, item := range req.Items {
		// Get product using raw SQL within transaction
		productQuery := `SELECT id, name, price, stock FROM products WHERE id = $1 FOR UPDATE`
		var productID int
		var name string
		var price float64
		var stock int

		err := dbTx.QueryRowContext(ctx, productQuery, item.ProductID).Scan(&productID, &name, &price, &stock)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("product %d not found", item.ProductID)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query product %d: %w", item.ProductID, err)
		}

		// Validate stock
		if stock < item.Quantity {
			return nil, fmt.Errorf("insufficient stock for product %d (%s): requested %d, available %d",
				item.ProductID, name, item.Quantity, stock)
		}

		subtotal := price * float64(item.Quantity)
		total += subtotal

		orderItems = append(orderItems, struct {
			ProductID int
			Quantity  int
			Price     float64
			Subtotal  float64
			Stock     int
		}{
			ProductID: productID,
			Quantity:  item.Quantity,
			Price:     price,
			Subtotal:  subtotal,
			Stock:     stock,
		})
	}

	// Step 2: Create order using raw SQL to get the ID
	orderQuery := `INSERT INTO orders (user_id, total, status) VALUES ($1, $2, $3) RETURNING id`
	var orderID int

	err = dbTx.QueryRowContext(ctx, orderQuery, req.UserID, total, "completed").Scan(&orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

		// Step 3: Freeze wallet amount and create pending transaction
		log.Printf("[Purchase] UserID: %s, Total: $%.2f", req.UserID, total)

		// Validate wallet type before freezing (use primary wallet for purchases)
		// Note: In the future, you might allow users to choose which wallet to use
		walletType := models.WalletTypePrimary
		if err := s.walletService.ValidateWalletType(ctx, walletType); err != nil {
			return nil, fmt.Errorf("invalid wallet type for purchase: %w", err)
		}

		// Freeze the amount and create pending transaction
		transactionID, err := s.walletService.FreezeAmountByType(ctx, dbTx, req.UserID, walletType, total, fmt.Sprintf("Purchase order #%d", orderID), &orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to freeze wallet amount: %w", err)
	}

	// Step 4: Validate goods and check stock (already done in Step 1, but this is where we ensure everything is ready)
	// At this point, products are validated and stock is sufficient

	// Step 5: Create order items and update product stock
	for _, item := range orderItems {
		// Create order item
		orderItemQuery := `INSERT INTO order_items (order_id, product_id, quantity, price, subtotal) VALUES ($1, $2, $3, $4, $5)`
		_, err = dbTx.ExecContext(ctx, orderItemQuery, orderID, item.ProductID, item.Quantity, item.Price, item.Subtotal)
		if err != nil {
			// Rollback transaction on error
			if rollbackErr := s.walletService.RollbackTransaction(ctx, dbTx, req.UserID, total, transactionID); rollbackErr != nil {
				log.Printf("Warning: Failed to rollback transaction: %v", rollbackErr)
			}
			return nil, fmt.Errorf("failed to create order item: %w", err)
		}

		// Update product stock (decrease by quantity)
		newStock := item.Stock - item.Quantity
		updateStockQuery := `UPDATE products SET stock = $1 WHERE id = $2`
		_, err = dbTx.ExecContext(ctx, updateStockQuery, newStock, item.ProductID)
		if err != nil {
			// Rollback transaction on error
			if rollbackErr := s.walletService.RollbackTransaction(ctx, dbTx, req.UserID, total, transactionID); rollbackErr != nil {
				log.Printf("Warning: Failed to rollback transaction: %v", rollbackErr)
			}
			return nil, fmt.Errorf("failed to update product stock: %w", err)
		}
	}

	// Step 6: Process 3rd party services (payment gateway, shipping, etc.)
	// This is where you would call external APIs
	// Example: paymentGateway.ProcessPayment(), shippingService.CreateShipment(), etc.
	// For now, we assume all 3rd party calls succeed
	// If 3rd party services fail, you should rollback:
	// if err := s.walletService.RollbackTransaction(ctx, dbTx, req.UserID, total, transactionID); err != nil {
	//     return nil, fmt.Errorf("3rd party service failed and rollback error: %w", err)
	// }

	// Step 7: Commit wallet transaction - deduct from frozen and mark as completed
	// Only called after all validations (goods check, 3rd party) are done
	err = s.walletService.CommitTransaction(ctx, dbTx, req.UserID, total, transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to commit wallet transaction: %w", err)
	}

	// Step 8: Commit database transaction
	if err := dbTx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit database transaction: %w", err)
	}

	return &models.PurchaseResponse{
		OrderID: orderID,
		Total:   total,
		Status:  "completed",
		Message: "Purchase completed successfully",
	}, nil
}

// GetProducts retrieves all available products
func (s *PurchaseService) GetProducts(ctx context.Context) ([]models.Product, error) {
	// Use raw SQL query for simplicity and reliability
	query := `SELECT id, name, description, price, stock FROM products ORDER BY id ASC`
	
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		var description sql.NullString
		
		err := rows.Scan(&p.ID, &p.Name, &description, &p.Price, &p.Stock)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		
		if description.Valid {
			p.Description = description.String
		}
		
		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating products: %w", err)
	}

	return products, nil
}

// GetOrders retrieves orders for a user
func (s *PurchaseService) GetOrders(ctx context.Context, userID string) ([]models.Order, error) {
	// Use raw SQL query for orders
	ordersQuery := `SELECT id, user_id, total, status, created_at FROM orders WHERE user_id = $1 ORDER BY created_at DESC`
	
	rows, err := s.db.QueryContext(ctx, ordersQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		var createdAtStr string
		
		err := rows.Scan(&o.ID, &o.UserID, &o.Total, &o.Status, &createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		
		// Parse created_at timestamp
		if createdAt, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", createdAtStr); err == nil {
			o.CreatedAt = createdAt
		} else if createdAt, err := time.Parse("2006-01-02T15:04:05Z07:00", createdAtStr); err == nil {
			o.CreatedAt = createdAt
		} else if createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr); err == nil {
			o.CreatedAt = createdAt
		}
		
		// Get order items for this order
		itemsQuery := `SELECT id, order_id, product_id, quantity, price, subtotal FROM order_items WHERE order_id = $1`
		itemRows, err := s.db.QueryContext(ctx, itemsQuery, o.ID)
		if err != nil {
			// Log error but continue - order without items is still valid
			o.Items = []models.OrderItem{}
		} else {
			var items []models.OrderItem
			for itemRows.Next() {
				var item models.OrderItem
				err := itemRows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.Price, &item.Subtotal)
				if err != nil {
					itemRows.Close()
					return nil, fmt.Errorf("failed to scan order item: %w", err)
				}
				items = append(items, item)
			}
			itemRows.Close()
			o.Items = items
		}
		
		orders = append(orders, o)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating orders: %w", err)
	}

	return orders, nil
}

// Helper function to safely get string value
func getStringValue(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
