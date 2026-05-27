package models

// PurchaseRequest represents a purchase request
type PurchaseRequest struct {
	UserID string        `json:"user_id"`
	Items  []PurchaseItem `json:"items"`
}

// PurchaseItem represents an item in a purchase request
type PurchaseItem struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

// PurchaseResponse represents the response after a purchase
type PurchaseResponse struct {
	OrderID int     `json:"order_id"`
	Total   float64 `json:"total"`
	Status  string  `json:"status"`
	Message string  `json:"message"`
}
