package models

// Product represents a product in the store
// Table: products
// Columns: id (SERIAL PRIMARY KEY), name (VARCHAR(255)), description (TEXT), price (DECIMAL(10,2)), stock (INTEGER)
type Product struct {
	ID          int     `db:"id" json:"id"`                       // Primary key, auto-increment
	Name        string  `db:"name" json:"name"`                   // Product name
	Description string  `db:"description" json:"description"`     // Product description
	Price       float64 `db:"price" json:"price"`                 // Price in USD
	Stock       int     `db:"stock" json:"stock"`                 // Available stock quantity
}
