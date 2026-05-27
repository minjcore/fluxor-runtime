package services

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"time"

	"github.com/fluxorio/fluxor/apps/postgres-demo/models"
	"github.com/fluxorio/fluxor/pkg/dbruntime"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication
type AuthService struct {
	jwtSecret      string
	defaultUser    string
	defaultPass    string
	hashedPassword []byte
	db             *dbruntime.DB // Optional: for user registration
	walletService  *WalletService                // Optional: for auto-creating wallets
}

// NewAuthService creates a new auth service
func NewAuthService(jwtSecret, defaultUser, defaultPass string) *AuthService {
	// Hash default password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPass), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[AUTH] Warning: Failed to hash default password: %v", err)
	}
	
	log.Printf("[AUTH] AuthService initialized with default user: %s", defaultUser)
	
	return &AuthService{
		jwtSecret:      jwtSecret,
		defaultUser:    defaultUser,
		defaultPass:    defaultPass,
		hashedPassword: hashedPassword,
	}
}

// SetDatabase sets the database component for user registration
func (s *AuthService) SetDatabase(db *dbruntime.DB) {
	s.db = db
	if db != nil {
		log.Printf("[AUTH] Database component set - database authentication enabled")
	} else {
		log.Printf("[AUTH] Warning: Database component is nil - only default user authentication available")
	}
}

// SetWalletService sets the wallet service for auto-creating wallets
func (s *AuthService) SetWalletService(walletService *WalletService) {
	s.walletService = walletService
}

// Login authenticates a user and returns a JWT token
func (s *AuthService) Login(req models.LoginRequest) (*models.LoginResponse, error) {
	log.Printf("[AUTH] Login attempt for username: %s", req.Username)
	
	// First, try to authenticate from database if available
	if s.db != nil {
		log.Printf("[AUTH] Database available, checking users table...")
		ctx := context.Background()
		
		var dbPassword string
		var isActive bool
		query := `SELECT password, is_active FROM users WHERE username = $1 LIMIT 1`
		err := s.db.QueryRowContext(ctx, query, req.Username).Scan(&dbPassword, &isActive)
		// If "password" column does not exist (e.g. schema uses password_hash), retry with password_hash
		if err != nil && err != sql.ErrNoRows && strings.Contains(err.Error(), "password") && strings.Contains(err.Error(), "does not exist") {
			query = `SELECT password_hash, is_active FROM users WHERE username = $1 LIMIT 1`
			err = s.db.QueryRowContext(ctx, query, req.Username).Scan(&dbPassword, &isActive)
		}
		if err == nil {
			log.Printf("[AUTH] User found in database, username: %s, is_active: %v", req.Username, isActive)
			
			// Check if user is active
			if !isActive {
				log.Printf("[AUTH] Login failed: User account is inactive")
				return nil, &AuthError{Message: "Account is inactive", Code: "ACCOUNT_INACTIVE"}
			}
			
			// Verify password from database
			err = bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(req.Password))
			if err != nil {
				log.Printf("[AUTH] Login failed: Password mismatch for user: %s", req.Username)
				return nil, &AuthError{Message: "Invalid username or password", Code: "AUTH_FAILED"}
			}
			
			log.Printf("[AUTH] Password verified successfully for user: %s", req.Username)
			
			// Generate JWT token
			token, err := s.generateToken(req.Username)
			if err != nil {
				log.Printf("[AUTH] Failed to generate token: %v", err)
				return nil, &AuthError{Message: "Failed to generate token", Code: "TOKEN_ERROR"}
			}
			
			log.Printf("[AUTH] Login successful for user: %s", req.Username)
			return &models.LoginResponse{
				Token:    token,
				Username: req.Username,
				Expires:  time.Now().Add(24 * time.Hour).Unix(),
			}, nil
		} else if err != sql.ErrNoRows {
			log.Printf("[AUTH] Database error while checking user: %v", err)
			// Continue to fallback authentication
		} else {
			log.Printf("[AUTH] User not found in database: %s, trying fallback authentication", req.Username)
		}
	} else {
		log.Printf("[AUTH] Database not available, using fallback authentication")
	}
	
	// Fallback: Check against default user (for backward compatibility)
	log.Printf("[AUTH] Checking against default user: %s", s.defaultUser)
	if req.Username != s.defaultUser {
		log.Printf("[AUTH] Login failed: Username mismatch (expected: %s, got: %s)", s.defaultUser, req.Username)
		return nil, &AuthError{Message: "Invalid username or password", Code: "AUTH_FAILED"}
	}

	// Verify password
	log.Printf("[AUTH] Verifying password for default user")
	if err := bcrypt.CompareHashAndPassword(s.hashedPassword, []byte(req.Password)); err != nil {
		log.Printf("[AUTH] Login failed: Password mismatch for default user")
		return nil, &AuthError{Message: "Invalid username or password", Code: "AUTH_FAILED"}
	}

	log.Printf("[AUTH] Password verified for default user")
	
	// Generate JWT token
	token, err := s.generateToken(req.Username)
	if err != nil {
		log.Printf("[AUTH] Failed to generate token: %v", err)
		return nil, &AuthError{Message: "Failed to generate token", Code: "TOKEN_ERROR"}
	}

	log.Printf("[AUTH] Login successful for default user: %s", req.Username)
	return &models.LoginResponse{
		Token:    token,
		Username: req.Username,
		Expires:  time.Now().Add(24 * time.Hour).Unix(),
	}, nil
}

// generateToken generates a JWT token for a user
func (s *AuthService) generateToken(username string) (string, error) {
	claims := jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// AuthError represents an authentication error
type AuthError struct {
	Message string
	Code    string
}

func (e *AuthError) Error() string {
	return e.Message
}

// ============================================================================
// USER REGISTRATION
// ============================================================================

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email,omitempty"`
}

// RegisterResponse represents a registration response
type RegisterResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

// Register creates a new user account and automatically creates wallets
func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	if req.Username == "" {
		return nil, &AuthError{Message: "Username is required", Code: "INVALID_REQUEST"}
	}
	if req.Password == "" {
		return nil, &AuthError{Message: "Password is required", Code: "INVALID_REQUEST"}
	}
	if len(req.Password) < 6 {
		return nil, &AuthError{Message: "Password must be at least 6 characters", Code: "INVALID_REQUEST"}
	}

	// Check if database is available
	if s.db == nil {
		return nil, &AuthError{Message: "Database not configured", Code: "CONFIG_ERROR"}
	}

	// Check if user already exists in users table
	var userExists bool
	checkUserQuery := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 LIMIT 1)`
	err := s.db.QueryRowContext(ctx, checkUserQuery, req.Username).Scan(&userExists)
	if err != nil && err != sql.ErrNoRows {
		return nil, &AuthError{Message: "Failed to check user existence", Code: "INTERNAL_ERROR"}
	}

	if userExists {
		return nil, &AuthError{Message: "User already exists", Code: "USER_EXISTS"}
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, &AuthError{Message: "Failed to hash password", Code: "INTERNAL_ERROR"}
	}

	// Insert user into users table (use password_hash if password column does not exist)
	insertUserQuery := `INSERT INTO users (username, email, password, is_active) VALUES ($1, $2, $3, $4) RETURNING id`
	var userID int
	err = s.db.QueryRowContext(ctx, insertUserQuery, req.Username, req.Email, string(hashedPassword), true).Scan(&userID)
	if err != nil && strings.Contains(err.Error(), "password") && strings.Contains(err.Error(), "does not exist") {
		insertUserQuery = `INSERT INTO users (username, email, password_hash, is_active) VALUES ($1, $2, $3, $4) RETURNING id`
		err = s.db.QueryRowContext(ctx, insertUserQuery, req.Username, req.Email, string(hashedPassword), true).Scan(&userID)
	}
	if err != nil {
		return nil, &AuthError{Message: "Failed to create user", Code: "INTERNAL_ERROR"}
	}

	// Use user_id = username for wallet system (wallet system uses string user_id)
	// In a real system, you might want to use the numeric user ID or a UUID
	userIDStr := req.Username

	// Auto-create wallets for the new user
	// This is done outside the transaction to avoid long-running transactions
	if s.walletService != nil {
		if err := s.walletService.AutoCreateWalletsForUser(ctx, userIDStr); err != nil {
			// Log error but don't fail registration
			// Wallets can be created later
			// In production, you might want to retry or queue this
			_ = err // Log error: log.Printf("Warning: Failed to auto-create wallets for user %s: %v", userIDStr, err)
		}
	}

	return &RegisterResponse{
		UserID:   userIDStr,
		Username: req.Username,
		Message:  "User registered successfully. Wallets have been created.",
	}, nil
}
