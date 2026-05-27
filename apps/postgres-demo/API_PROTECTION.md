# API Protection Documentation

## Overview

All API endpoints (GET and POST) are protected with JWT authentication middleware. This document outlines the protection strategy and which endpoints are public vs protected.

## Protection Strategy

### 1. JWT Authentication Middleware
- All protected endpoints use `router.GETFastWith()` or `router.POSTFastWith()` with `jwtMiddleware`
- JWT middleware validates the `Authorization: Bearer <token>` header
- Invalid or missing tokens return `401 Unauthorized` with JSON error response

### 2. Public Endpoints (No Authentication Required)

These endpoints are intentionally public for system functionality:

- `GET /` - Login page
- `GET /login` - Login page
- `POST /api/auth/login` - User login (returns JWT token)
- `POST /api/auth/register` - User registration
- `POST /api/auth/logout` - User logout
- `GET /api/health` - Health check endpoint

### 3. Protected Endpoints (Require JWT Authentication)

All other endpoints require a valid JWT token in the `Authorization` header.

#### Dashboard Routes
- `GET /dashboard` - Main dashboard (protected)
- `GET /accountant` - Accountant dashboard (protected)

#### Database API Routes
- `GET /api/db/status` - Database status (protected)
- `GET /api/db/stats` - Database statistics (protected)
- `POST /api/db/query` - Execute database query (protected)

#### Purchase API Routes
- `GET /api/products` - Get products list (protected)
- `POST /api/purchase` - Make a purchase (protected)
- `GET /api/orders` - Get user orders (protected)

#### Wallet API Routes
- `GET /api/wallet/balance` - Get wallet balance (protected)
- `GET /api/wallet/transactions` - Get wallet transactions (protected)
- `POST /api/wallet/add` - Add balance to wallet (protected)
- `POST /api/wallet/transfer` - Transfer between wallets (protected)

#### Accountant API Routes
- `GET /api/accountant/trial-balance` - Get trial balance (protected)
- `GET /api/accountant/account-balances` - Get account balances (protected)
- `GET /api/accountant/journals` - Get journal entries (protected)
- `GET /api/accountant/journals/:id` - Get journal with entries (protected)
- `GET /api/accountant/accounts` - Get all accounts (protected)

#### Wallet Rules API Routes
- `GET /api/wallet-rules` - Get all wallet rules (protected)
- `POST /api/wallet-rules` - Create wallet rule (protected)
- `POST /api/wallet-rules/:id/update` - Update wallet rule (protected)
- `POST /api/wallet-rules/:id/delete` - Delete wallet rule (protected)

#### System Settings API Routes
- `GET /api/settings` - Get all system settings (protected)
- `GET /api/settings/:key` - Get specific setting (protected)
- `GET /api/settings/:key/value` - Get setting value (protected)
- `POST /api/settings` - Create system setting (protected)
- `POST /api/settings/:key/update` - Update system setting (protected)
- `POST /api/settings/:key/delete` - Delete system setting (protected)

## Authentication Flow

1. **Login**: User sends credentials to `POST /api/auth/login`
2. **Token**: Server returns JWT token in response
3. **Protected Requests**: Client includes token in `Authorization: Bearer <token>` header
4. **Validation**: JWT middleware validates token on each protected request
5. **Access**: If valid, request proceeds; if invalid, returns 401 error

## Error Responses

### Unauthorized (401)
```json
{
  "error": "unauthorized",
  "message": "invalid or missing token"
}
```

### Invalid Content Type (400)
For POST requests to API endpoints without proper Content-Type:
```json
{
  "error": "invalid_content_type",
  "message": "Content-Type must be application/json for API endpoints"
}
```

## Security Best Practices

1. **Always use HTTPS in production** - JWT tokens should only be transmitted over encrypted connections
2. **Store tokens securely** - Use httpOnly cookies or secure localStorage
3. **Token expiration** - Tokens have expiration time (configured in JWT config)
4. **Refresh tokens** - Consider implementing refresh token mechanism for long-lived sessions
5. **Rate limiting** - Consider adding rate limiting to prevent brute force attacks
6. **CORS** - Configure CORS properly to restrict which origins can access APIs

## Testing Protected Endpoints

### Using curl

```bash
# 1. Login to get token
TOKEN=$(curl -X POST http://localhost:8081/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | jq -r '.token')

# 2. Use token for protected endpoint
curl -X GET http://localhost:8081/api/wallet/balance \
  -H "Authorization: Bearer $TOKEN"
```

### Using JavaScript (fetch)

```javascript
// 1. Login
const loginResponse = await fetch('/api/auth/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ username: 'admin', password: 'admin123' })
});
const { token } = await loginResponse.json();

// 2. Use token for protected endpoint
const response = await fetch('/api/wallet/balance', {
  headers: { 'Authorization': `Bearer ${token}` }
});
const data = await response.json();
```

## Summary

✅ **All GET and POST API endpoints are protected** with JWT authentication middleware
✅ **Public endpoints** are limited to authentication and health check
✅ **Dashboard routes** are protected
✅ **Error handling** provides clear feedback for unauthorized access
✅ **Security headers** can be added via middleware if needed
