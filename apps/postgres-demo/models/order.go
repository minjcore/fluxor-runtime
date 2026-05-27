package models

import "time"

// Order represents a purchase order
// Table: orders
// Columns: id (SERIAL PRIMARY KEY), user_id (VARCHAR(255)), total (DECIMAL(10,2)), status (VARCHAR(50)), created_at (TIMESTAMP)
type Order struct {
	ID        int       `db:"id" json:"id"`                       // Primary key, auto-increment
	UserID    string    `db:"user_id" json:"user_id"`             // User who made the order
	Total     float64   `db:"total" json:"total"`                 // Total order amount
	Status    string    `db:"status" json:"status"`                // Order status (pending, completed, cancelled)
	CreatedAt time.Time `db:"created_at" json:"created_at"`        // Order creation timestamp
	Items     []OrderItem `db:"-" json:"items"`                   // Related order items (not stored in orders table)
}

// OrderItem represents an item in an order
// Table: order_items
// Columns: id (SERIAL PRIMARY KEY), order_id (INTEGER FK), product_id (INTEGER FK), quantity (INTEGER), price (DECIMAL(10,2)), subtotal (DECIMAL(10,2))
type OrderItem struct {
	ID        int     `db:"id" json:"id"`                         // Primary key, auto-increment
	OrderID   int     `db:"order_id" json:"order_id"`             // Foreign key to orders.id
	ProductID int     `db:"product_id" json:"product_id"`         // Foreign key to products.id
	Quantity  int     `db:"quantity" json:"quantity"`             // Quantity purchased
	Price     float64 `db:"price" json:"price"`                   // Price per unit at time of purchase
	Subtotal  float64 `db:"subtotal" json:"subtotal"`             // Total for this item (price * quantity)
}
