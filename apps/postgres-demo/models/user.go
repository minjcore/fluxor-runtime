package models

import (
	"time"
)

// ============================================================================
// USER MODEL
// ============================================================================

// User represents a user in the system
// Table: users
type User struct {
	ID        int       `db:"id" json:"id"`                 // Primary key, auto-increment
	Username  string    `db:"username" json:"username"`     // Unique username
	Email     *string   `db:"email" json:"email,omitempty"` // Email address (optional)
	Password  string    `db:"password" json:"-"`            // Hashed password (excluded from JSON)
	IsActive  bool      `db:"is_active" json:"is_active"`    // Whether user account is active
	CreatedAt time.Time `db:"created_at" json:"created_at"` // Account creation timestamp
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"` // Last update timestamp
}

// ============================================================================
// AUTHENTICATION REQUEST/RESPONSE MODELS
// ============================================================================

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	Expires  int64  `json:"expires"`
}
